// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"sync"
)

// DynamicConfig contains configuration items which may change
// dynamically over time.
type DynamicConfig struct {
	// RateByService contains the rate for each service/env tuple,
	// used in priority sampling by client libs.
	RateByService RateByService
}

// NewDynamicConfig creates a new dynamic config object which maps service signatures
// to their corresponding sampling rates. Each service will have a default assigned
// matching the service rate of the specified env.
func NewDynamicConfig(env string) *DynamicConfig {
	return &DynamicConfig{RateByService: RateByService{defaultEnv: env}}
}

// RateByService stores the sampling rate per service. It is thread-safe, so
// one can read/write on it concurrently, using getters and setters.
type RateByService struct {
	defaultEnv string // env. to use for service defaults

	mu          sync.RWMutex // guards rates
	localRates  map[string]float64
	remoteRates map[string]float64
}

func uploadRates(defaultEnv string, new map[ServiceSignature]float64, dest map[string]float64) map[string]float64 {
	if dest == nil {
		dest = make(map[string]float64, len(new))
	}
	for k := range dest {
		delete(dest, k)
	}
	for k, v := range new {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		dest[k.String()] = v
		if k.Env == defaultEnv {
			// if this is the default env, then this is also the
			// service's default rate unbound to any env.
			dest[ServiceSignature{Name: k.Name}.String()] = v
		}
	}
	return dest
}

// SetAll the sampling rate for all services. If a service/env is not
// in the map, then the entry is removed.
func (rbs *RateByService) SetAll(localRates, remoteRates map[ServiceSignature]float64) {
	rbs.mu.Lock()
	defer rbs.mu.Unlock()

	rbs.localRates = uploadRates(rbs.defaultEnv, localRates, rbs.localRates)
	rbs.remoteRates = uploadRates(rbs.defaultEnv, remoteRates, rbs.remoteRates)
}

// GetAll returns all sampling rates for all services.
func (rbs *RateByService) GetAll() (localRates, remoteRates map[string]float64) {
	rbs.mu.RLock()
	defer rbs.mu.RUnlock()
	return copyMap(rbs.localRates), copyMap(rbs.remoteRates)
}

func copyMap(old map[string]float64) map[string]float64 {
	res := make(map[string]float64, len(old))
	for k, v := range old {
		res[k] = v
	}
	return res
}
