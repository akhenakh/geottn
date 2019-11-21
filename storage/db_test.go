package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	ts := time.Now().UTC()
	k := "MYDEVICE"

	dk := DataKey(k, ts, 48.8, 2.2)
	ndk, nts, lat, lng, err := ReadDataKey(dk)
	require.NoError(t, err)
	require.Equal(t, k, ndk)
	require.Equal(t, ts, nts)
	require.InDelta(t, 48.8, lat, 0.0001)
	require.InDelta(t, 2.2, lng, 0.0001)
	t.Log("DataKey", dk, string(dk))

	pk := PointKey(48.8, 2.2, ts, k)
	cell, nts, npk, err := ReadPointKey(pk)
	require.NoError(t, err)
	require.Equal(t, k, npk)
	require.Equal(t, ts, nts)
	require.InDelta(t, 48.8, cell.LatLng().Lat.Degrees(), 0.0001)
	require.InDelta(t, 2.2, cell.LatLng().Lng.Degrees(), 0.0001)
	t.Log("PointKey", pk, string(pk))
}
