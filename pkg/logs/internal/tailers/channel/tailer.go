// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package channel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/tailers"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// serviceEnvVar is the environment variable of the service tag (this is used only for the serverless agent)
const serviceEnvVar = "DD_SERVICE"

// Tailer consumes and processes a channel of strings, and sends them to a stream of log messages.
type Tailer struct {
	tailers.TailerBase
	inputChan chan *config.ChannelMessage
}

// NewTailer returns a new Tailer
func NewTailer(source *config.LogSource, inputChan chan *config.ChannelMessage, outputChan chan *message.Message) *Tailer {
	t := &Tailer{
		inputChan: inputChan,
	}
	t.TailerBase = tailers.NewTailerBase(source, t.run, fmt.Sprintf("channel:%p", inputChan), outputChan)
	fmt.Printf("%#v\n", t)
	return t
}

func (t *Tailer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case logline, ok := <-t.inputChan:
			// Loop terminates when the channel is closed.
			if !ok {
				return
			}

			origin := message.NewOrigin(t.Source)
			tags := origin.Tags()

			origin.SetService(computeServiceName(logline.Lambda, os.Getenv(serviceEnvVar)))

			if len(t.Source.Config.Tags) > 0 {
				tags = append(tags, t.Source.Config.Tags...)
			}
			origin.SetTags(tags)
			if logline.Lambda != nil {
				t.OutputChan <- message.NewMessageFromLambda(logline.Content, origin, message.StatusInfo, logline.Timestamp, logline.Lambda.ARN, logline.Lambda.RequestID, time.Now().UnixNano())
			} else {
				t.OutputChan <- message.NewMessage(logline.Content, origin, message.StatusInfo, time.Now().UnixNano())
			}
		}
	}
}

func computeServiceName(lambdaConfig *config.Lambda, serviceName string) string {
	if lambdaConfig == nil {
		return "agent"
	}
	if len(serviceName) > 0 {
		return strings.ToLower(serviceName)
	}
	return ""
}
