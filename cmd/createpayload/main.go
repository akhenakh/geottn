package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/akhenakh/cayenne"
)

var (
	lat     = flag.Float64("lat", 48.8, "The Latitude")
	lng     = flag.Float64("lng", 2.2, "The Longitude")
	channel = flag.Int("channel", 1, "The channel")
)

func main() {
	flag.Parse()

	e := cayenne.NewEncoder()
	e.AddGPS(uint8(*channel), float32(*lat), float32(*lng), 0.0)

	b := e.Bytes()
	hexPayload := hex.EncodeToString(b)
	fmt.Println("Data", hexPayload)
}
