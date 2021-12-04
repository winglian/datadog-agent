// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package service

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/client"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/uptane"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"go.etcd.io/bbolt"
)

const (
	minimalRefreshInterval = time.Second * 5
	defaultMaxBucketSize   = 10
)

// Service defines the remote config management service responsible for fetching, storing
// and dispatching the configurations
type Service struct {
	sync.RWMutex

	refreshInterval time.Duration
	remoteConfigKey remoteConfigKey

	ctx    context.Context
	db     *bbolt.DB
	client *uptane.Client
}

// Refresh configurations by:
// - collecting the new subscribers or the one whose configuration has expired
// - create a query
// - send the query to the backend
//
func (s *Service) refresh() {
	log.Debug("Refreshing configurations")

}

// Start the remote configuration management service
func (s *Service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		for {
			select {
			case <-time.After(s.refreshInterval):
				s.refresh()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// NewService instantiates a new remote configuration management service
func NewService() (*Service, error) {
	refreshInterval := config.Datadog.GetDuration("remote_configuration.refresh_interval")
	if refreshInterval < minimalRefreshInterval {
		refreshInterval = minimalRefreshInterval
	}

	rawRemoteConfigKey := config.Datadog.GetString("remote_configuration.key")
	remoteConfigKey, err := parseRemoteConfigKey(rawRemoteConfigKey)
	if err != nil {
		return nil, err
	}

	apiKey := config.Datadog.GetString("api_key")
	if config.Datadog.IsSet("remote_configuration.api_key") {
		apiKey = config.Datadog.GetString("remote_configuration.api_key")
	}
	apiKey = config.SanitizeAPIKey(apiKey)
	hostname, err := util.GetHostname(context.Background())
	if err != nil {
		return nil, err
	}
	backendURL := config.Datadog.GetString("remote_configuration.endpoint")
	client.NewHTTPClient(backendURL, apiKey, remoteConfigKey.appKey, hostname)

	dbPath := path.Join(config.Datadog.GetString("run_path"), "remote-config.db")
	db, err := openCacheDB(dbPath)
	if err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("%s/%d/", remoteConfigKey.datacenter, remoteConfigKey.orgID)
	client, err := uptane.NewClient(db, cacheKey, remoteConfigKey.orgID)
	if err != nil {
		return nil, err
	}

	return &Service{
		ctx:             context.Background(),
		refreshInterval: refreshInterval,
		remoteConfigKey: remoteConfigKey,
		db:              db,
		client:          client,
	}, nil
}
