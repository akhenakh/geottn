package geottnsvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TheThingsNetwork/ttn/core/types"
	log "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/health"

	"github.com/akhenakh/geottn/storage"
)

type Server struct {
	appName string
	logger  log.Logger
	Health  *health.Server
	GeoDB   storage.Indexer
	config  Config
}

type Config struct {
	// the cayenne channel used for gps messages
	Channel int
}

func NewServer(appName string, logger log.Logger, cfg Config) *Server {
	logger = log.With(logger, "component", "server")
	return &Server{
		appName: appName,
		logger:  logger,
		config:  cfg,
	}
}

// HandleMessage handles message from TTN
func (s *Server) HandleMessage(ctx context.Context, msg *types.UplinkMessage) {
	MsgReceivedCounter.Inc()
	if msg.PayloadFields == nil {
		level.Debug(s.logger).Log("msg", "received msg with empty PayloadFields")
		return
	}

	gpsi, ok := msg.PayloadFields[fmt.Sprintf("gps_%d", s.config.Channel)]
	if !ok {
		level.Debug(s.logger).Log("msg", "received msg with no gps in PayloadFields")
		return
	}

	gps := gpsi.(map[string]interface{})

	lat := gps["latitude"].(float64)
	lng := gps["longitude"].(float64)

	level.Debug(s.logger).Log("msg", "received msg", "device_id", msg.DevID, "latitude", lat, "longitude", lng)

	err := s.GeoDB.Store(msg.DevID, msg.PayloadRaw, lat, lng, time.Now())
	if err != nil {
		level.Error(s.logger).Log("msg", "can't store datapoint", "error", err)
		return
	}
	InsertCounter.Inc()
}

func (s *Server) Store(ctx context.Context, dp *DataPoint) (*empty.Empty, error) {
	e := &empty.Empty{}
	t, err := ptypes.Timestamp(dp.Time)
	if err != nil {
		return e, err
	}
	err = s.GeoDB.Store(dp.DeviceId, dp.Payload, dp.Latitude, dp.Longitude, t)
	if err != nil {
		return e, err
	}
	InsertCounter.Inc()
	return e, nil
}

func (s *Server) RadiusSearch(ctx context.Context, req *RadiusSearchRequest) (*DataPoints, error) {
	dps, err := s.GeoDB.RadiusSearch(req.Lat, req.Lng, req.Radius)
	if err != nil {
		return nil, err
	}

	res := &DataPoints{
		Points: make([]*DataPoint, len(dps)),
	}
	for i, dp := range dps {
		res.Points[i] = StorageToDataPoint(&dp)
	}
	return res, nil
}

func (s *Server) RectSearch(ctx context.Context, req *RectSearchRequest) (*DataPoints, error) {
	dps, err := s.GeoDB.RectSearch(req.Urlat, req.Urlng, req.Bllat, req.Bllng)
	if err != nil {
		return nil, err
	}

	res := &DataPoints{
		Points: make([]*DataPoint, len(dps)),
	}
	for i, dp := range dps {
		res.Points[i] = StorageToDataPoint(&dp)
	}
	return res, nil
}

func (s *Server) Get(ctx context.Context, req *GetRequest) (*DataPoint, error) {
	dps, err := s.GeoDB.Get(req.Key)
	if err != nil {
		return nil, err
	}

	return StorageToDataPoint(dps), nil
}

func (s *Server) GetAll(ctx context.Context, in *GetRequest) (*DataPoints, error) {
	return nil, errors.New("not implemented")
}

func StorageToDataPoint(dp *storage.DataPoint) *DataPoint {
	if dp == nil {
		return nil
	}
	t, _ := ptypes.TimestampProto(dp.Time)
	return &DataPoint{
		DeviceId:  dp.Key,
		Latitude:  dp.Lat,
		Longitude: dp.Lng,
		Payload:   dp.Value,
		Time:      t,
	}
}
