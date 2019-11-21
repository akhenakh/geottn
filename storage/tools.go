package storage

import (
	"encoding/binary"
	"math"
	"time"
)

const earthCircumferenceMeter = 40075017

var (
	// MaxGeoTime helper to query into the future
	MaxGeoTime = time.Unix(0, math.MaxInt64)

	// MinGeoTime helper to query into the past
	MinGeoTime = time.Unix(0, 0)
)

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func int64tob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func Uint64tob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func S2RadialAreaMeters(radius float64) float64 {
	r := (radius / earthCircumferenceMeter) * math.Pi * 2
	return math.Pi * r * r
}
