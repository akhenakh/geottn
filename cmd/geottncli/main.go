package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/akhenakh/cayenne"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"

	"github.com/akhenakh/geottn/geottnsvc"
)

var (
	geoTTNURI = flag.String("geoTTNURI", "localhost:9200", "geoTTN grpc URI")
	lat       = flag.Float64("lat", 48.8, "Lat")
	lng       = flag.Float64("lng", 2.2, "Lng")
	radius    = flag.Float64("radius", 1000, "Radius in meters")
	key       = flag.String("key", "", "ask for a key, if empty perform radius search")
	topic     = flag.String("topic", "metar", "topic")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(*geoTTNURI,
		grpc.WithInsecure(),
		grpc.WithBalancerName(roundrobin.Name), //nolint:staticcheck
	)
	if err != nil {
		log.Fatal(err)
	}

	c := geottnsvc.NewGeoTTNClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if *key != "" {
		dp, err := c.Get(ctx, &geottnsvc.GetRequest{
			Key: *key,
		})
		if err != nil {
			log.Fatal(err)
		}

		dec := cayenne.NewDecoder(bytes.NewBuffer(dp.Payload))
		msg, err := dec.DecodeUplink()
		if err != nil {
			log.Fatal(err)
		}

		response := make(map[string]interface{})
		for k, v := range msg.Values() {
			response[k] = v
		}
		response["latitude"] = dp.Latitude
		response["longitude"] = dp.Longitude
		response["device_id"] = dp.DeviceId
		response["time"] = dp.Time

		log.Println(response)

		os.Exit(0)
	}
	rep, err := c.RadiusSearch(ctx, &geottnsvc.RadiusSearchRequest{
		Lat:    *lat,
		Lng:    *lng,
		Radius: *radius,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("query ok", len(rep.Points))

	for _, dp := range rep.Points {
		dec := cayenne.NewDecoder(bytes.NewBuffer(dp.Payload))
		msg, err := dec.DecodeUplink()
		if err != nil {
			log.Fatal(err)
		}

		response := make(map[string]interface{})
		for k, v := range msg.Values() {
			response[k] = v
		}
		response["device_id"] = dp.DeviceId
		response["time"] = dp.Time

		log.Println(response)
	}

}
