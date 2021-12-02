// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package uptane

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"

	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
)

type localStore struct {
	repository string
	store      *store.Store
}

func (s *localStore) GetMeta() (map[string]json.RawMessage, error) {
	meta, err := s.store.GetMeta(s.repository)
	if err != nil {
		if !errors.Is(err, bbolt.ErrBucketNotFound) {
			return nil, err
		}
		meta = make(map[string]json.RawMessage)
	}

	if _, found := meta["root.json"]; !found {
		var rootMetadata []byte
		var err error
		switch s.repository {
		case "director":
			rootMetadata = getDirectorRoot()
		case "config":
			rootMetadata = getConfigRoot()
		default:
			return nil, fmt.Errorf("unexpected root name")
		}
		if err != nil {
			return nil, err
		}
		meta["root.json"] = rootMetadata
	}

	return meta, nil
}

func (s *localStore) SetMeta(name string, meta json.RawMessage) error {
	return s.store.SetMeta(s.repository, name, meta)
}

func (s *localStore) DeleteMeta(name string) error {
	return s.store.DeleteMeta(s.repository, name)
}

type localStoreConfig struct {
	localStore
}

func newLocalStoreConfig(store *store.Store) *localStoreConfig {
	return &localStoreConfig{
		localStore: localStore{repository: "config", store: store},
	}
}

type localStoreDirector struct {
	localStore
}

func newLocalStoreDirector(store *store.Store) *localStoreDirector {
	return &localStoreDirector{
		localStore: localStore{repository: "director", store: store},
	}
}
