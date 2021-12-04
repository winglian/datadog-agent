// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/remote/store"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/version"
)

const (
	minimalRefreshInterval = time.Second * 5
	defaultMaxBucketSize   = 10
	defaultURL             = ""
)

// Opts defines the remote config service options
type Opts struct {
	URL                    string
	APIKey                 string
	RemoteConfigurationKey string
	Hostname               string
	DBPath                 string
	RefreshInterval        time.Duration
	MaxBucketSize          int
	ReadOnly               bool
}

// Service defines the remote config management service responsible for fetching, storing
// and dispatching the configurations
type Service struct {
	sync.RWMutex
	ctx    context.Context
	opts   Opts
	store  *store.Store
	client Client

	subscribers           []*Subscriber
	configSnapshotVersion uint64
	configRootVersion     uint64
	directorRootVersion   uint64
	orgID                 string
}

// Refresh configurations by:
// - collecting the new subscribers or the one whose configuration has expired
// - create a query
// - send the query to the backend
//
func (s *Service) refresh() {
	log.Debug("Refreshing configurations")

	request := pbgo.ClientLatestConfigsRequest{
		AgentVersion:                 version.AgentVersion,
		Hostname:                     s.opts.Hostname,
		CurrentConfigSnapshotVersion: s.configSnapshotVersion,
		CurrentConfigRootVersion:     s.configRootVersion,
		CurrentDirectorRootVersion:   s.directorRootVersion,
	}

	// determine which configuration we need to refresh
	var refreshSubscribers = map[string][]*Subscriber{}

	s.RLock()
	defer s.RUnlock()

	now := time.Now()

	for _, subscriber := range s.subscribers {
		product := subscriber.product
		if subscriber.lastUpdate.Add(subscriber.refreshRate).Before(now) {
			log.Debugf("Add '%s' to the list of configurations to refresh", product)

			if subscriber.lastUpdate.IsZero() {
				request.NewProducts = append(request.NewProducts, product)
			} else {
				request.Products = append(request.Products, product)
			}

			refreshSubscribers[product.String()] = append(refreshSubscribers[product.String()], subscriber)
		}
	}

	if len(refreshSubscribers) == 0 {
		log.Debugf("Nothing to fetch")
		return
	}

	// Fetch the configuration from the backend
	response, err := s.client.Fetch(s.ctx, &request)
	if err != nil {
		log.Errorf("Failed to fetch remote configuration: %s", err)
		return
	}

	refreshedProducts := make(map[*pbgo.DelegatedMeta][]*pbgo.File)

TARGETFILE:
	for _, targetFile := range response.TargetFiles {
		product, err := getTargetProduct(targetFile.Path)
		if err != nil {
			log.Error(err)
			continue
		}

		for _, delegatedTarget := range response.ConfigMetas.DelegatedTargets {
			if delegatedTarget.Role == product {
				log.Debugf("Received configuration for product %s", product)
				refreshedProducts[delegatedTarget] = append(refreshedProducts[delegatedTarget], targetFile)
				continue TARGETFILE
			}
		}

		log.Errorf("Failed to find delegated target for %s", product)
		return
	}

	log.Debugf("Possibly notify subscribers")
	for delegatedTarget, targetFiles := range refreshedProducts {
		configResponse := &pbgo.ConfigResponse{
			ConfigSnapshotVersion:        response.DirectorMetas.Snapshot.Version,
			ConfigDelegatedTargetVersion: delegatedTarget.Version,
			DirectoryRoots:               response.DirectorMetas.Roots,
			DirectoryTargets:             response.DirectorMetas.Targets,
			TargetFiles:                  targetFiles,
		}

		product := delegatedTarget.GetRole()
	SUBSCRIBER:
		for _, subscriber := range refreshSubscribers[product] {
			if response.DirectorMetas.Snapshot.Version <= subscriber.lastVersion {
				log.Debugf("Nothing to do, subscriber version %d > %d", subscriber.lastVersion, response.DirectorMetas.Snapshot.Version)
				continue
			}

			if err := s.notifySubscriber(subscriber, configResponse); err != nil {
				log.Errorf("failed to notify subscriber: %s", err)
				continue SUBSCRIBER
			}
		}

		if err := s.store.StoreConfig(product, configResponse); err != nil {
			log.Errorf("failed to persistent config for product %s: %s", product, err)
		}
	}

	if response.ConfigMetas != nil {
		if rootCount := len(response.ConfigMetas.Roots); rootCount > 0 {
			s.configRootVersion = response.ConfigMetas.Roots[rootCount-1].Version
		}
		if response.ConfigMetas.Snapshot != nil {
			s.configSnapshotVersion = response.ConfigMetas.Snapshot.Version
		}
	}

	if response.DirectorMetas != nil {
		if rootCount := len(response.DirectorMetas.Roots); rootCount > 0 {
			s.directorRootVersion = response.DirectorMetas.Roots[rootCount-1].Version
		}
	}

	log.Debugf("Stored last known version to config snapshot %d, config root %d, snapshot root %d", s.configSnapshotVersion, s.configRootVersion, s.directorRootVersion)
}

func getTargetProduct(path string) (string, error) {
	splits := strings.SplitN(path, "/", 3)
	if len(splits) < 3 {
		return "", fmt.Errorf("Failed to determine product for target file %s", path)
	}

	return splits[1], nil
}

func (s *Service) notifySubscriber(subscriber *Subscriber, configResponse *pbgo.ConfigResponse) error {
	log.Debugf("Notifying subscriber %s with version %d", subscriber.product, configResponse.DirectoryTargets.Version)

	if err := subscriber.callback(configResponse); err != nil {
		return err
	}

	subscriber.lastUpdate = time.Now()
	subscriber.lastVersion = configResponse.DirectoryTargets.Version

	return nil
}

// RegisterSubscriber registers a new subscriber for a product's configurations
func (s *Service) RegisterSubscriber(subscriber *Subscriber) {
	s.Lock()
	s.subscribers = append(s.subscribers, subscriber)
	s.Unlock()

	product := subscriber.product
	log.Debugf("New registered subscriber for %s", product.String())

	config, err := s.store.GetLastConfig(product.String())
	if err == nil {
		log.Debugf("Found cached configuration for product %s", product)
		if err := s.notifySubscriber(subscriber, config); err != nil {
			log.Error(err)
		}
	} else {
		log.Debugf("No stored configuration for product %s", product)
	}
}

// UnregisterSubscriber unregisters a subscriber for a product's configurations
func (s *Service) UnregisterSubscriber(unregister *Subscriber) {
	s.Lock()
	for i, subscriber := range s.subscribers {
		if subscriber == unregister {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
		}
	}
	s.Unlock()
}

// Start the remote configuration management service
func (s *Service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		for {
			select {
			case <-time.After(s.opts.RefreshInterval):
				s.refresh()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// GetConfigs returns config for the given product
func (s *Service) GetConfigs(product string) ([]*pbgo.ConfigResponse, error) {
	return s.store.GetConfigs(product)
}

// GetStore returns the configuration store
func (s *Service) GetStore() *store.Store {
	return s.store
}

// NewService instantiates a new remote configuration management service
func NewService(opts Opts) (*Service, error) {
	if opts.RefreshInterval <= 0 {
		opts.RefreshInterval = config.Datadog.GetDuration("remote_configuration.refresh_interval")
	}

	if opts.RefreshInterval < minimalRefreshInterval {
		opts.RefreshInterval = minimalRefreshInterval
	}

	if opts.DBPath == "" {
		opts.DBPath = path.Join(config.Datadog.GetString("run_path"), "remote-config.db")
	}

	if opts.APIKey == "" {
		apiKey := config.Datadog.GetString("api_key")
		if config.Datadog.IsSet("remote_configuration.api_key") {
			apiKey = config.Datadog.GetString("remote_configuration.api_key")
		}
		opts.APIKey = config.SanitizeAPIKey(apiKey)
	}

	if opts.RemoteConfigurationKey == "" {
		opts.RemoteConfigurationKey = config.Datadog.GetString("remote_configuration.key")
	}

	if opts.URL == "" {
		opts.URL = config.Datadog.GetString("remote_configuration.endpoint")
	}

	if opts.Hostname == "" {
		hostname, err := util.GetHostname(context.Background())
		if err != nil {
			return nil, err
		}
		opts.Hostname = hostname
	}

	if opts.MaxBucketSize <= 0 {
		opts.MaxBucketSize = defaultMaxBucketSize
	}

	if opts.URL == "" {
		opts.URL = defaultURL
	}

	split := strings.SplitN(opts.RemoteConfigurationKey, "/", 3)
	if len(split) < 3 {
		return nil, fmt.Errorf("invalid remote configuration key format, should be datacenter/org_id/app_key")
	}

	datacenter, org, appKey := split[0], split[1], split[2]

	store, err := store.NewStore(opts.DBPath, !opts.ReadOnly, opts.MaxBucketSize, datacenter+"/"+org)
	if err != nil {
		return nil, err
	}

	return &Service{
		ctx:    context.Background(),
		client: NewHTTPClient(opts.URL, opts.APIKey, appKey, opts.Hostname),
		store:  store,

		opts:  opts,
		orgID: org,
	}, nil
}
