// Package leveldb implements a GraphStore using a LevelDB backend database.
package leveldb

import (
	"fmt"
	"io"
	"os"

	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/keyvalue"

	"levigo"
)

// levelDB is a wrapper around a levigo.DB that implements keyvalue.DB
type levelDB struct {
	db    *levigo.DB
	cache *levigo.Cache

	// save options to reduce number of allocations during high load
	readOpts  *levigo.ReadOptions
	scanOpts  *levigo.ReadOptions
	writeOpts *levigo.WriteOptions
}

// CacheCapacity is the caching capacity (in bytes) used for any newly created
// leveldb GraphStores.
var CacheCapacity = 4 * 1048576 // 4mb

// ValidDB determines if the given path could be a LevelDB database.
func ValidDB(path string) bool {
	stat, err := os.Stat(path)
	return os.IsNotExist(err) || (err == nil && stat.IsDir())
}

// Open returns a GraphStore backed by a LevelDB database at the given filepath.
func Open(path string) (storage.GraphStore, error) {
	options := levigo.NewOptions()
	defer options.Close()
	cache := levigo.NewLRUCache(CacheCapacity)
	options.SetCache(cache)
	options.SetCreateIfMissing(true)
	db, err := levigo.Open(path, options)
	if err != nil {
		return nil, fmt.Errorf("could not open LevelDB at %q: %v", path, err)
	}
	scanOpts := levigo.NewReadOptions()
	scanOpts.SetFillCache(false)
	return &keyvalue.Store{&levelDB{
		db:        db,
		cache:     cache,
		readOpts:  levigo.NewReadOptions(),
		scanOpts:  scanOpts,
		writeOpts: levigo.NewWriteOptions(),
	}}, nil
}

// Close will close the underlying LevelDB database.
func (s *levelDB) Close() error {
	s.db.Close()
	s.cache.Close()
	s.readOpts.Close()
	s.scanOpts.Close()
	s.writeOpts.Close()
	return nil
}

// Writer implements part of the keyvalue.DB interface.
func (s *levelDB) Writer() (keyvalue.Writer, error) {
	return &writer{s, levigo.NewWriteBatch()}, nil
}

// Reader implements part of the keyvalue.DB interface.
func (s *levelDB) Reader(key []byte) (keyvalue.Iterator, error) {
	iter := s.db.NewIterator(s.readOpts)
	iter.Seek(key)
	return &iterator{iter}, nil
}

// Scanner implements part of the keyvalue.DB interface.
func (s *levelDB) Scanner() (keyvalue.Iterator, error) {
	iter := s.db.NewIterator(s.scanOpts)
	iter.SeekToFirst()
	return &iterator{iter}, nil
}

type writer struct {
	s *levelDB
	*levigo.WriteBatch
}

// Write implements part of the keyvalue.Writer interface.
func (w *writer) Write(key, val []byte) error {
	w.Put(key, val)
	return nil
}

// Close implements part of the keyvalue.Writer interface.
func (w *writer) Close() error {
	if err := w.s.db.Write(w.s.writeOpts, w.WriteBatch); err != nil {
		return err
	}
	w.WriteBatch.Close()
	return nil
}

type iterator struct{ it *levigo.Iterator }

// Close implements part of the keyvalue.Iterator interface.
func (i iterator) Close() error {
	i.it.Close()
	return nil
}

// Next implements part of the keyvalue.Iterator interface.
func (i iterator) Next() ([]byte, []byte, error) {
	if !i.it.Valid() {
		if err := i.it.GetError(); err != nil {
			return nil, nil, err
		}
		return nil, nil, io.EOF
	}
	key, val := i.it.Key(), i.it.Value()
	i.it.Next()
	return key, val, nil
}
