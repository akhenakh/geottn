package badger

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/require"
)

func openStore(t *testing.T) (*badger.DB, func()) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)

	opt := badger.DefaultOptions(dir)

	db, err := badger.Open(opt)
	require.NoError(t, err)

	return db, func() {
		if db != nil {
			db.Close()
		}
		os.RemoveAll(dir)
	}
}

func TestStoreGeoVal(t *testing.T) {
	bdb, clean := openStore(t)
	defer clean()

	idx := &Indexer{
		DB: bdb,
	}

	ts := time.Now().UTC()
	k := "KEY"
	v := []byte("VALUE")
	err := idx.Store(k, v, 48.8, 2.2, ts)
	require.NoError(t, err)

	dps, err := idx.RadiusSearch(48.8, 2.2, 10000)
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, v, dps[0].Value)

	dps, err = idx.RadiusSearch(44.8, 2.2, 10000)
	require.NoError(t, err)
	require.Len(t, dps, 0)

	dps, err = idx.RectSearch(48.83, 2.56, 48.62, 2.13)
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, v, dps[0].Value)
}

func TestKeys(t *testing.T) {
	bdb, clean := openStore(t)
	defer clean()

	idx := &Indexer{
		DB: bdb,
	}
	ts := time.Now().UTC()
	k := "KEY"
	v := []byte("VALUE")
	err := idx.Store(k, v, 48.8, 2.2, ts)
	require.NoError(t, err)

	res, err := idx.Keys()
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, k, res[0])
}

func TestGetAll(t *testing.T) {
	bdb, clean := openStore(t)
	defer clean()

	idx := &Indexer{
		DB: bdb,
	}
	ts := time.Now().UTC()
	k := "KEY"
	v := []byte("VALUE")
	err := idx.Store(k, v, 48.8, 2.2, ts)
	require.NoError(t, err)

	// storing a second one with the same key
	ts2 := time.Now().UTC()
	v2 := []byte("VALUE2")
	err = idx.Store(k, v2, 48.802, 2.201, ts2)
	require.NoError(t, err)

	// we should find only the latest
	dps, err := idx.RadiusSearch(48.8, 2.2, 10000)
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, v2, dps[0].Value)

	// we should find the two
	res, err := idx.GetAll(k, 2)
	require.NoError(t, err)
	require.Len(t, res, 2)

	dp, err := idx.Get(k)
	require.NoError(t, err)
	require.Equal(t, v2, dp.Value)
	require.Equal(t, k, dp.Key)
	require.Equal(t, ts2, dp.Time)
	require.InDelta(t, 48.802, dp.Lat, 0.002)
	require.InDelta(t, 2.201, dp.Lng, 0.002)
}
