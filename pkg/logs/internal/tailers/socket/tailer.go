// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package socket

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/decoder"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers/noop"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/tailers"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// Tailer reads data from a net.Conn.  It uses a `read` callback to be generic
// over types of connections.
//
// This tailer contains three components, communicating with channels:
//  - readForever
//  - decoder
//  - message forwarder
type Tailer struct {
	tailers.TailerBase
	Conn    net.Conn
	read    func(*Tailer) ([]byte, error)
	decoder *decoder.Decoder
}

// NewTailer returns a new Tailer
func NewTailer(source *config.LogSource, conn net.Conn, outputChan chan *message.Message, read func(*Tailer) ([]byte, error)) *Tailer {
	t := &Tailer{
		Conn:    conn,
		read:    read,
		decoder: decoder.InitializeDecoder(source, noop.New()),
	}
	t.TailerBase = tailers.NewTailerBase(source, t.run, fmt.Sprintf("socket:%p", conn), outputChan)
	return t
}

// Stop implements Tailer#Stop.
func (t *Tailer) Stop() {
	t.interruptRead()
	t.TailerBase.Stop()
}

// run runs the goroutines representing this tailer
func (t *Tailer) run(ctx context.Context) {
	// Of the three tailer components, readForever and the decoder will run in their
	// own goroutines, while forwardMessages will run in this goroutine.  The Stop
	// method "poisons" the readForever component, which eventually stops, and that
	// stop percolates through the decoder to forwardMessages.
	go t.readForever(ctx)
	t.decoder.Start()

	// forwardMessages runs in this goroutine, so t.TailerBase.Stop() will not
	// return until it finishes.
	t.forwardMessages()
}

// Interrupt an ongoing read operation, causing
func (t *Tailer) interruptRead() {
	// close the Conn so that any ongoing conn.Read returns immediately
	t.Conn.Close()
}

// forwardMessages forwards messages to output channel
func (t *Tailer) forwardMessages() {
	for output := range t.decoder.OutputChan {
		if len(output.Content) > 0 {
			t.OutputChan <- message.NewMessageWithSource(output.Content, message.StatusInfo, t.Source, output.IngestionTimestamp)
		}
	}
}

// readForever reads the data from conn.
func (t *Tailer) readForever(ctx context.Context) {
	defer func() {
		// close our input
		t.Conn.Close()
		// propagate the closure downstream to the decoder
		t.decoder.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			// stop reading data from the connection
			return
		default:
			data, err := t.read(t)
			if err != nil && err == io.EOF {
				// connection has been closed client-side, stop from reading new data
				return
			}
			if err != nil {
				// an error occurred, stop from reading new data
				log.Warnf("Couldn't read message from connection: %v", err)
				return
			}
			t.Source.BytesRead.Add(int64(len(data)))
			t.decoder.InputChan <- decoder.NewInput(data)
		}
	}
}
