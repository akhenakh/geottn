package web

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akhenakh/cayenne"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"

	"github.com/akhenakh/geottn/storage"
)

var (
	pathTpl = []string{"index.html"}
)

type Server struct {
	appName     string
	logger      log.Logger
	geoDB       storage.Indexer
	config      Config
	FileHandler http.Handler
	Box         *packr.Box
}

type Config struct {
	// the cayenne channel used for gps messages
	Channel int

	// the Key for mapbox or self hosted
	TilesKey string

	// Set to false if using MapBox
	SelfHostedMap bool

	// the URL where to point to get tiles if not using MapBox
	TilesURL string
}

func NewServer(appName string, logger log.Logger, geoDB storage.Indexer, cfg Config) *Server {
	logger = log.With(logger, "component", "web")
	return &Server{
		appName: appName,
		logger:  logger,
		config:  cfg,
		geoDB:   geoDB,
	}
}

func (s *Server) DataQuery(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var serverSpan opentracing.Span
	operationName := "/api/data"
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		level.Debug(s.logger).Log("msg", "can't find a span", "error", err)
	}

	serverSpan = opentracing.StartSpan(
		operationName,
		ext.RPCServerOption(wireContext))

	defer serverSpan.Finish()
	ctx = opentracing.ContextWithSpan(ctx, serverSpan)

	vars := mux.Vars(r)

	w.Header().Set("Content-Type", "application/json")

	dps, err := s.geoDB.GetAll(vars["key"], 100)
	if err != nil {
		level.Error(s.logger).Log("msg", "can't query Get", "key", vars["key"], "error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	res := make([]map[string]interface{}, len(dps))
	for i, dp := range dps {
		dec := cayenne.NewDecoder(bytes.NewBuffer(dp.Value))
		msg, err := dec.DecodeUplink()
		if err != nil {
			level.Error(s.logger).Log("msg", "can't decode uplink message", "key", vars["key"], "error", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		jsresp := make(map[string]interface{})
		for k, v := range msg.Values() {
			jsresp[k] = v
		}
		jsresp["device_id"] = dp.Key
		jsresp["time"] = dp.Time

		res[i] = jsresp
	}

	b, err := json.Marshal(res)
	if err != nil {
		level.Error(s.logger).Log("msg", "can't marshal json", "key", vars["key"], "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(b)
}

func (s *Server) RectQuery(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var serverSpan opentracing.Span
	operationName := "/api/rect"
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		level.Debug(s.logger).Log("msg", "can't find a span", "error", err)
	}

	serverSpan = opentracing.StartSpan(
		operationName,
		ext.RPCServerOption(wireContext))

	defer serverSpan.Finish()
	ctx = opentracing.ContextWithSpan(ctx, serverSpan)

	vars := mux.Vars(r)
	urlat, err := strconv.ParseFloat(vars["urlat"], 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	urlng, err := strconv.ParseFloat(vars["urlng"], 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bllat, err := strconv.ParseFloat(vars["bllat"], 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bllng, err := strconv.ParseFloat(vars["bllng"], 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	dpts, err := s.geoDB.RectSearch(urlat, urlng, bllat, bllng)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	fc := geojson.FeatureCollection{}
	for _, p := range dpts {
		f := &geojson.Feature{}
		f.Properties = make(map[string]interface{})
		f.Properties["device_id"] = p.Key
		f.Properties["ts"] = p.Time

		pg := geom.NewPointFlat(geom.XY, []float64{p.Lng, p.Lat})
		f.Geometry = pg
		fc.Features = append(fc.Features, f)
	}
	b, err := fc.MarshalJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(b)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" {
		path = "index.html"
	}

	p := map[string]interface{}{
		"TilesURL":      s.config.TilesURL,
		"TilesKey":      s.config.TilesKey,
		"Lat":           48.864716,
		"Lng":           2.349014,
		"SelfHostedMap": s.config.SelfHostedMap,
	}

	// serve file normally
	if !isTpl(path) {
		s.FileHandler.ServeHTTP(w, r)
		return
	}

	tmplt := template.New(path)

	sf, err := s.Box.FindString(path)
	if err != nil {
		level.Error(s.logger).Log("msg", "can't open template", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	tmplt, err = tmplt.Parse(sf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		level.Error(s.logger).Log("msg", "can't parse template", "error", err)
		return
	}

	ctype := mime.TypeByExtension(filepath.Ext(path))
	w.Header().Set("Content-Type", ctype)

	tmplt.Execute(w, p)
}

func isTpl(path string) bool {
	for _, p := range pathTpl {
		if p == path {
			return true
		}
	}
	return false
}
