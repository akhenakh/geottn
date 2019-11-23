package storage

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/golang/geo/s2"
)

const Prefix = "TT"

type Indexer interface {
	Store(k string, v []byte, lat, lng float64, t time.Time) error
	StoreTx(tx Tx, k string, v []byte, lat, lng float64, t time.Time) error
	Get(k string) (*DataPoint, error)
	Keys() ([]string, error)
	GetAll(k string, count int) ([]DataPoint, error)
	RadiusSearch(lat, lng, radius float64) ([]DataPoint, error)
	RectSearch(urlat, urlng, bllat, bllng float64) ([]DataPoint, error)
	Begin() Tx
}

type Tx interface {
	Discard()
	Commit() error
}

type DataPoint struct {
	Lat, Lng float64
	Key      string
	Value    []byte
	Time     time.Time
}

func DataKey(k string, t time.Time, lat, lng float64) []byte {
	// the data key Prefix+"D"+k+#+time+s2
	dk := make([]byte, len(Prefix)+1+len(k)+1+8+8)
	copy(dk, Prefix+"D")
	copy(dk[len(Prefix)+1:], k)
	dk[len(Prefix)+1+len(k)] = '#'
	// using reverse timestamp
	ts := int64tob(math.MaxInt64 - t.UnixNano())
	copy(dk[len(Prefix)+1+len(k)+1:], ts)

	c := s2.CellIDFromLatLng(s2.LatLngFromDegrees(lat, lng))
	copy(dk[len(Prefix)+1+len(k)+1+8:], itob(uint64(c)))
	return dk
}

// PointKey returns the key generated for a position + id
func PointKey(lat, lng float64, t time.Time, k string) []byte {
	c := s2.CellIDFromLatLng(s2.LatLngFromDegrees(lat, lng))
	// a key Prefix+"G"+cellid+ts+k
	gk := make([]byte, len(Prefix)+1+8+8+len(k))
	copy(gk, Prefix+"G")
	copy(gk[len(Prefix)+1:], itob(uint64(c)))
	// using reverse timestamp
	ts := int64tob(math.MaxInt64 - t.UnixNano())
	copy(gk[len(Prefix)+1+8:], ts)
	copy(gk[len(Prefix)+1+8+8:], k)
	return gk
}

// ListingKey returns the key used to list all keys
func ListKey(k string) []byte {
	// a key Prefix+"L"+key
	return []byte(Prefix + "L" + k)
}

// ReadPointKey returns cell, time, key
func ReadPointKey(pk []byte) (s2.CellID, time.Time, string, error) {
	buf := bytes.NewBuffer(pk[len(Prefix)+1:])
	var c s2.CellID
	var t time.Time

	// read back cell
	err := binary.Read(buf, binary.BigEndian, &c)
	if err != nil {
		return c, t, "", err
	}

	// read back time
	var ts int64
	err = binary.Read(buf, binary.BigEndian, &ts)
	if err != nil {
		return c, t, "", err
	}
	// reverse ts back
	t = time.Unix(0, math.MaxInt64-ts).UTC()

	k := make([]byte, len(pk)-len(Prefix)-1-8-8)
	copy(k, pk[len(Prefix)+1+8+8:])

	return c, t, string(k), nil
}

// ReadDataKey returns the key, time, lat, lng
// note that lat lng will have a small delta compared to the original values
// induced by the s2 cell
func ReadDataKey(dk []byte) (string, time.Time, float64, float64, error) {
	// the data key Prefix+"D"+k+#+time+s2
	var t time.Time
	var c s2.CellID

	buf := bytes.NewBuffer(dk[len(dk)-8-8:])

	// read back time
	var ts int64
	err := binary.Read(buf, binary.BigEndian, &ts)
	if err != nil {
		return "", t, 0.0, 0.0, err
	}
	// reverse ts back
	t = time.Unix(0, math.MaxInt64-ts).UTC()

	// read back cell
	err = binary.Read(buf, binary.BigEndian, &c)
	if err != nil {
		return "", t, 0.0, 0.0, err
	}

	k := make([]byte, len(dk)-len(Prefix)-1-1-8-8)
	copy(k, dk[len(Prefix)+1:len(dk)-1-8-8])

	return string(k), t, c.LatLng().Lat.Degrees(), c.LatLng().Lng.Degrees(), nil
}
