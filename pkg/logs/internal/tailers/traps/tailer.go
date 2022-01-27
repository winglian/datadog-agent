// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package traps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/tailers"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/DataDog/datadog-agent/pkg/snmp/traps"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Tailer consumes and processes a stream of trap packets, and sends them to a stream of log messages.
type Tailer struct {
	tailers.TailerBase
	inputChan traps.PacketsChannel
}

// NewTailer returns a new Tailer
func NewTailer(source *config.LogSource, inputChan traps.PacketsChannel, outputChan chan *message.Message) *Tailer {
	t := &Tailer{
		inputChan: inputChan,
	}
	t.TailerBase = tailers.NewTailerBase(source, t.run, fmt.Sprintf("traps:%p", inputChan), outputChan)
	return t
}

// Start starts the tailer.
func (t *Tailer) run(ctx context.Context) {
	// Loop terminates when the channel is closed.
	for {
		select {
		case <-ctx.Done():
			return
		case packet := <-t.inputChan:
			data, err := traps.FormatPacketToJSON(packet)
			if err != nil {
				log.Errorf("failed to format packet: %s", err)
				continue
			}
			t.Source.BytesRead.Add(int64(len(data)))
			content, err := json.Marshal(data)
			if err != nil {
				log.Errorf("failed to serialize packet data to JSON: %s", err)
				continue
			}
			origin := message.NewOrigin(t.Source)
			origin.SetTags(traps.GetTags(packet))
			t.OutputChan <- message.NewMessage(content, origin, message.StatusInfo, time.Now().UnixNano())
		}
	}
}
