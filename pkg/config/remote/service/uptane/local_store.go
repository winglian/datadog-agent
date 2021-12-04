// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package uptane

import (
	"encoding/json"
	fmt "fmt"
	"os"

	"github.com/DataDog/datadog-agent/pkg/config/remote/meta"
	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"go.etcd.io/bbolt"
)

var (
	metaRootKey = []byte("root.json")
)

type repoLocalStore struct {
	path        string
	metasBucket []byte
	rootsBucket []byte
	db          *bbolt.DB
}

func newLocalStore(path string, cacheKey string, initialRoots meta.EmbeddedRoots) (*repoLocalStore, error) {
	s := &repoLocalStore{
		path:        path,
		metasBucket: []byte(fmt.Sprintf("%s_metas", cacheKey)),
		rootsBucket: []byte(fmt.Sprintf("%s_roots", cacheKey)),
	}
	db, err := s.open(initialRoots)
	if err != nil {
		return nil, err
	}
	s.db = db
	return s, nil
}

func (s *repoLocalStore) open(initialRoots meta.EmbeddedRoots) (*bbolt.DB, error) {
	db, err := bbolt.Open(s.path, 0600, &bbolt.Options{})
	if err != nil {
		if err := os.Remove(s.path); err != nil {
			return nil, fmt.Errorf("failed to remove corrupted database: %w", err)
		}
		if db, err = bbolt.Open(s.path, 0600, &bbolt.Options{}); err != nil {
			return nil, err
		}
	}
	err = s.init(initialRoots)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (s *repoLocalStore) init(initialRoots meta.EmbeddedRoots) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(s.metasBucket)
		if err != nil {
			return fmt.Errorf("failed to create metas bucket: %v", err)
		}
		_, err = tx.CreateBucketIfNotExists(s.rootsBucket)
		if err != nil {
			return fmt.Errorf("failed to create roots bucket: %v", err)
		}
		rootsBucket := tx.Bucket(s.rootsBucket)
		for version, root := range initialRoots {
			rootKey := []byte(fmt.Sprintf("%d.root.json", version))
			err := rootsBucket.Put(rootKey, root)
			if err != nil {
				return fmt.Errorf("failed set embeded root in roots bucket: %v", err)
			}
		}
		metasBucket := tx.Bucket(s.rootsBucket)
		if metasBucket.Get(metaRootKey) == nil {
			err := rootsBucket.Put(metaRootKey, initialRoots.Last())
			if err != nil {
				return fmt.Errorf("failed set embeded root in roots bucket: %v", err)
			}
		}
		return nil
	})
}

func (s *repoLocalStore) close() error {
	return s.db.Close()
}

// GetMeta returns a map of all the metadata files
func (s *repoLocalStore) GetMeta() (map[string]json.RawMessage, error) {
	meta := make(map[string]json.RawMessage)
	err := s.db.View(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(s.metasBucket)
		cursor := metaBucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			tmp := make([]byte, len(v))
			copy(tmp, v)
			meta[string(k)] = json.RawMessage(tmp)
		}
		return nil
	})
	return meta, err
}

// SetMeta stores a tuf metadata file
func (s *repoLocalStore) SetMeta(name string, meta json.RawMessage) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(s.metasBucket)
		return metaBucket.Put([]byte(name), meta)
	})
}

// DeleteMeta deletes a tuf metadata file
func (s *repoLocalStore) DeleteMeta(name string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(s.metasBucket)
		return metaBucket.Delete([]byte(name))
	})
}

type localStore struct {
	configLocalStore   *repoLocalStore
	directorLocalStore *repoLocalStore
}

func newLocalStoreConfig(store *store.Store) *localStoreConfig {
	return &localStoreConfig{
		localStore: localStore{repository: "config", store: store},
	}
}
