// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	model "github.com/DataDog/agent-payload/v5/process"

	"github.com/stretchr/testify/assert"

	"os"
	"testing"
)

func TestEncodePayload(t *testing.T) {
	var mb model.MessageBody
	mb = &model.CollectorNode{
		ClusterName: "nams-cluster-name",
		ClusterId:   "nams-cluster-id",
		GroupId:     0,
		GroupSize:   0,
		Nodes: []*model.Node{
			{
				Metadata: &model.Metadata{
					Name: "node-1",
				},
			}, {
				Metadata: &model.Metadata{
					Name: "node-2",
				},
			},
			{
				Metadata: &model.Metadata{
					Name: "node-3",
				},
			},
		},
		Tags: []string{"custom:tag"},
	}
	payload, err := EncodePayload(mb)
	assert.NoError(t, err)
	file, err := os.Create("node.bin")
	defer file.Close()
	_, err = file.Write(payload)
	assert.NoError(t, err)
}
