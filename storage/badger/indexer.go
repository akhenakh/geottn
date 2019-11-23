package badger

import (
	"bytes"
	"errors"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/geo/s2"

	"github.com/akhenakh/geottn/storage"
)

type Indexer struct {
	*badger.DB
}

// StoreTx is storing k and v but also geoindex at lat lng for the  most recent entry
func (idx *Indexer) StoreTx(txi storage.Tx, k string, v []byte, lat, lng float64, t time.Time) error {
	tx, ok := txi.(*badger.Txn)
	if !ok {
		return errors.New("invalid tx passed")
	}

	// the geo key G
	pk := storage.PointKey(lat, lng, t, k)

	// the datakey D
	dk := storage.DataKey(k, t, lat, lng)

	// the listing key L
	kk := storage.ListKey(k)

	// Check for existing
	// get rid of the last 64bits ts and 64bits s2 cell to iterate on the prefix
	prefix := dk[:len(dk)-8-8]
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	it := tx.NewIterator(opts)
	defer it.Close()

	exist := false

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		exist = true
		item := it.Item()
		ck := item.KeyCopy(nil)

		// already existing key while iterating D ?
		// if ck is dk we dont want to erase it since it's the exact same entry
		if bytes.Equal(ck, dk) {
			return nil
		}

		// always delete associated G entry since we ever want one at a time on the geo index
		ek, et, elat, elng, err := storage.ReadDataKey(ck)
		if err != nil {
			return err
		}
		epk := storage.PointKey(elat, elng, et, ek)
		if err := tx.Delete(epk); err != nil {
			return err
		}
	}

	// storing G
	e := badger.NewEntry(pk, v)
	if err := tx.SetEntry(e); err != nil {
		return err
	}

	// storing D
	e = badger.NewEntry(dk, v)
	if err := tx.SetEntry(e); err != nil {
		return err
	}

	// storing L
	if !exist {
		e = badger.NewEntry(kk, nil)
		if err := tx.SetEntry(e); err != nil {
			return err
		}

	}
	return nil

}

// Store is storing k and v but also geoindex at lat lng
func (idx *Indexer) Store(k string, v []byte, lat, lng float64, t time.Time) error {
	txn := idx.NewTransaction(true)
	defer txn.Discard()

	if err := idx.StoreTx(txn, k, v, lat, lng, t); err != nil {
		return err
	}

	return txn.Commit()
}

// GetAll return all entries for k up to count
func (idx *Indexer) GetAll(k string, count int) ([]storage.DataPoint, error) {
	var res []storage.DataPoint
	existing := 0
	err := idx.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = count
		if opts.PrefetchSize <= 0 {
			opts.PrefetchSize = 10
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := storage.DataKey(k, storage.MaxGeoTime, 0.0, 0.0)
		// get rid of the last 64bits of ts and 64 bits of cell to iterate on the prefix
		prefix = prefix[:len(prefix)-8-8]
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if count > 0 && existing >= count {
				break
			}

			item := it.Item()
			k := item.KeyCopy(nil)
			dk, t, lat, lng, err := storage.ReadDataKey(k)
			if err != nil {
				return err
			}

			valc, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			dt := storage.DataPoint{
				Time:  t,
				Value: valc,
				Lat:   lat,
				Lng:   lng,
				Key:   dk,
			}
			res = append(res, dt)
			existing++
		}
		return nil
	})

	return res, err
}

// Get the most recent entry for k
func (idx *Indexer) Get(k string) (*storage.DataPoint, error) {
	res, err := idx.GetAll(k, 1)
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, nil
	}
	return &res[0], err
}

// Keys list all keys
func (idx *Indexer) Keys() ([]string, error) {
	var res []string
	err := idx.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := []byte(storage.Prefix + "L")

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			rk := k[len(storage.Prefix)+1:]
			res = append(res, string(rk))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RectPointSearch returns all Points contained in the rect
func (idx *Indexer) RectSearch(urlat, urlng, bllat, bllng float64) ([]storage.DataPoint, error) {
	rect := s2.RectFromLatLng(s2.LatLngFromDegrees(bllat, bllng))
	rect = rect.AddPoint(s2.LatLngFromDegrees(urlat, urlng))
	coverer := &s2.RegionCoverer{MaxCells: 8}
	cu := coverer.Covering(rect)
	var res []storage.DataPoint

	for _, c := range cu {
		// a key prefix+"G"+cellid
		start := make([]byte, len(storage.Prefix)+1+8)
		copy(start, storage.Prefix+"G")
		copy(start[len(storage.Prefix)+1:], storage.Uint64tob(uint64(c.RangeMin())))
		stop := make([]byte, len(storage.Prefix)+1+8)
		copy(stop, storage.Prefix+"G")
		copy(stop[len(storage.Prefix)+1:], storage.Uint64tob(uint64(c.RangeMax())))

		err := idx.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = true
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Seek(start); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				if bytes.Compare(k, stop) > 0 {
					break
				}
				ck := item.KeyCopy(nil)
				c, t, rk, _ := storage.ReadPointKey(ck)
				if rect.ContainsPoint(c.Point()) {
					p := storage.DataPoint{
						Lat:  c.LatLng().Lat.Degrees(),
						Lng:  c.LatLng().Lng.Degrees(),
						Time: t,
						Key:  rk,
					}

					cv, err := item.ValueCopy(nil)
					if err != nil {
						return err
					}
					p.Value = cv

					res = append(res, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// RadiusSearch returns the Points found in the index inside radius (no data)
func (idx *Indexer) RadiusSearch(lat, lng, radius float64) ([]storage.DataPoint, error) {
	center := s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng))
	acap := s2.CapFromCenterArea(center, storage.S2RadialAreaMeters(radius))
	coverer := &s2.RegionCoverer{MaxCells: 8}
	cu := coverer.Covering(acap)
	var res []storage.DataPoint

	for _, c := range cu {
		// a key prefix+"G"+cellid
		start := make([]byte, len(storage.Prefix)+1+8)
		copy(start, storage.Prefix+"G")
		copy(start[len(storage.Prefix)+1:], storage.Uint64tob(uint64(c.RangeMin())))
		stop := make([]byte, len(storage.Prefix)+1+8)
		copy(stop, storage.Prefix+"G")
		copy(stop[len(storage.Prefix)+1:], storage.Uint64tob(uint64(c.RangeMax())))

		err := idx.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = true
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Seek(start); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				if bytes.Compare(k, stop) > 0 {
					break
				}
				ck := item.KeyCopy(nil)
				c, t, rk, _ := storage.ReadPointKey(ck)

				if acap.ContainsPoint(c.Point()) {
					p := storage.DataPoint{
						Lat:  c.LatLng().Lat.Degrees(),
						Lng:  c.LatLng().Lng.Degrees(),
						Time: t,
						Key:  rk,
					}
					cv, err := item.ValueCopy(nil)
					if err != nil {
						return err
					}
					p.Value = cv

					res = append(res, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (idx *Indexer) Begin() storage.Tx {
	txn := idx.NewTransaction(true)
	return txn
}
