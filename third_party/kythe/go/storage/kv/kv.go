// Package kv implements a GraphStore using a kv backend database.
package kv

import (
	"fmt"
	"os"

	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/keyvalue"

	kv "github.com/cznic/kv"
)

// kvStore is a wrapper around a kv.DB that implements the GraphStore interface.
type kvStore struct{ *kv.DB }

// ValidDB determines if the given path could be a kv database.
func ValidDB(path string) bool {
	stat, err := os.Stat(path)
	return os.IsNotExist(err) || (err == nil && !stat.IsDir())
}

// Open returns a GraphStore backed by a kv database at the given filepath. The
// db is created, if missing.
func Open(path string) (storage.GraphStore, error) {
	opts := &kv.Options{}
	db, err := kv.Create(path, opts)
	if os.IsExist(err) {
		db, err = kv.Open(path, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("could not open kv database at %q: %v", path, err)
	}
	return &keyvalue.Store{&kvStore{db}}, nil
}

// InMemory returns a GraphStore backed by an in-memory kv database.
func InMemory() (storage.GraphStore, error) {
	db, err := kv.CreateMem(&kv.Options{})
	if err != nil {
		return nil, err
	}
	return &keyvalue.Store{&kvStore{db}}, nil
}

// Writer implements part of the keyvalue.DB interface.
func (s *kvStore) Writer() (keyvalue.Writer, error) {
	return &writer{s}, nil
}

// Reader implements part of the keyvalue.DB interface.
func (s *kvStore) Reader(key []byte) (keyvalue.Iterator, error) {
	iter, _, err := s.DB.Seek(key)
	if err != nil {
		return nil, err
	}
	return &iterator{iter}, nil
}

// Scanner implements part of the keyvalue.DB interface.
func (s *kvStore) Scanner() (keyvalue.Iterator, error) {
	iter, err := s.DB.SeekFirst()
	if err != nil {
		return nil, err
	}
	return &iterator{iter}, nil
}

type writer struct{ *kvStore }
type iterator struct{ *kv.Enumerator }

// Write implements part of the keyvalue.Writer interface.
func (w writer) Write(key, val []byte) error { return w.Set(key, val) }

// Close implements part of the keyvalue.Writer interface.
func (writer) Close() error { return nil }

// Close implements part of the keyvalue.Iterator interface.
func (iterator) Close() error { return nil }
