//  Unless explicitly stated otherwise all files in this repository are licensed
//  under the Apache License Version 2.0.
//  This product includes software developed at Datadog (https://www.datadoghq.com/).
//  Copyright 2016-present Datadog, Inc.

package testutil

import (
	"math/rand"
	"time"
)

// RandomPortPair returns a pair of unique ports
func RandomPortPair() (int, int) {
	rand.Seed(time.Now().UnixNano())
	aPort := rand.Intn(32768) + 16384
	bPort := 0
	for {
		bPort = rand.Intn(32768) + 16384
		if bPort != aPort {
			break
		}
	}
	return aPort, bPort
}
