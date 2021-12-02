// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.Datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package tagset

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCachingFactory(t *testing.T) {
	testFactory(t, func() Factory { return NewCachingFactory(10, 5) })
	testFactoryCaching(t, func() Factory { return NewCachingFactory(10, 5) })
}

func TestCachingFactory_Union_Fuzz(t *testing.T) {
	f := NewCachingFactory(100, 1)
	fuzz(func(seed int64) {
		r := rand.New(rand.NewSource(seed))

		bothBuilder := f.NewBuilder(30)

		n := r.Intn(15)
		aBuilder := f.NewBuilder(n)
		for i := 0; i < n; i++ {
			t := fmt.Sprintf("tag%d", r.Intn(30))
			aBuilder.Add(t)
			bothBuilder.Add(t)
		}
		a := aBuilder.Freeze()

		n = r.Intn(15)
		bBuilder := f.NewBuilder(n)
		for i := 0; i < n; i++ {
			t := fmt.Sprintf("tag%d", r.Intn(30))
			bBuilder.Add(t)
			bothBuilder.Add(t)
		}
		b := bBuilder.Freeze()

		union := f.Union(a, b)
		union.validate(t)

		both := bothBuilder.Freeze()
		both.validate(t)

		require.Equal(t, both.Hash(), union.Hash())
		require.Equal(t, both.Sorted(), union.Sorted())
	})
}

func TestCachingFactory_Telemetry(t *testing.T) {
	telemetryPeriod = 10 * time.Millisecond
	defer func() { telemetryPeriod = time.Second }()

	tlmChan := make(chan Telemetry)

	tc := NewCachingFactoryWithTelemetry(10, 2, "test", tlmChan)

	// use the factory in a tight loop until it spits out some
	// telemetry
	stop := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}

			tc.NewTags([]string{fmt.Sprintf("tag:%d", i%20)})
			i++
		}
	}()

	tlm := <-tlmChan
	stop <- struct{}{}

	require.Equal(t, "test", tlm.FactoryName)

	// the content of the telemetry will depend on timing, but validate
	// its general form
	require.Equal(t, numCacheIDs, len(tlm.Caches))
	require.Equal(t, 2, len(tlm.Caches["byTagsetHashCache"].Maps))
	require.NotEqual(t, 0, tlm.Caches["byTagsetHashCache"].Maps[1].Searches)
}
