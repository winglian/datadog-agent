// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package decoder

import (
	"bytes"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// LineParser handles decoded lines, parsing them into decoder.Message's using
// an embedded parsers.Parser.
//
// Input and output channels are given to the constructor for the concrete
// type.  After Start(), the actor runs until its input channel is closed.
// After all inputs are processed, the actor closes its output channel.
type LineParser interface {
	Start()
}

// SingleLineParser makes sure that multiple lines from a same content
// are properly put together.
type SingleLineParser struct {
	parser     parsers.Parser
	inputChan  chan *DecodedInput
	outputChan chan *Message
}

// NewSingleLineParser returns a new SingleLineParser.
func NewSingleLineParser(
	inputChan chan *DecodedInput,
	outputChan chan *Message,
	parser parsers.Parser) *SingleLineParser {
	return &SingleLineParser{
		parser:     parser,
		inputChan:  inputChan,
		outputChan: outputChan,
	}
}

// Start starts the parser.
func (p *SingleLineParser) Start() {
	go p.run()
}

// run consumes new lines and processes them.
func (p *SingleLineParser) run() {
	for input := range p.inputChan {
		p.process(input)
	}
	close(p.outputChan)
}

func (p *SingleLineParser) process(input *DecodedInput) {
	// Just parse an pass to the next step
	msg, err := p.parser.Parse(input.content)
	if err != nil {
		log.Debug(err)
	}
	p.outputChan <- NewMessage(msg.Content, msg.Status, input.rawDataLen, msg.Timestamp)
}

// MultiLineParser makes sure that chunked lines are properly put together.
type MultiLineParser struct {
	buffer       *bytes.Buffer
	flushTimeout time.Duration
	inputChan    chan *DecodedInput
	outputChan   chan *Message
	parser       parsers.Parser
	rawDataLen   int
	lineLimit    int
	status       string
	timestamp    string
}

// NewMultiLineParser returns a new MultiLineParser.
func NewMultiLineParser(
	inputChan chan *DecodedInput,
	outputChan chan *Message,
	flushTimeout time.Duration,
	parser parsers.Parser,
	lineLimit int,
) *MultiLineParser {
	return &MultiLineParser{
		inputChan:    inputChan,
		outputChan:   outputChan,
		buffer:       bytes.NewBuffer(nil),
		flushTimeout: flushTimeout,
		lineLimit:    lineLimit,
		parser:       parser,
	}
}

// Start starts the handler.
func (p *MultiLineParser) Start() {
	go p.run()
}

// run processes new lines from the channel and makes sur the content is properly sent when
// it stayed for too long in the buffer.
func (p *MultiLineParser) run() {
	flushTimer := time.NewTimer(p.flushTimeout)
	defer func() {
		flushTimer.Stop()
		// make sure the content stored in the buffer gets sent,
		// this can happen when the stop is called in between two timer ticks.
		p.sendLine()
		// and close the outputChan, signalling to downstream that this component is finished
		close(p.outputChan)
	}()
	for {
		select {
		case message, isOpen := <-p.inputChan:
			if !isOpen {
				// inputChan has been closed, no more lines are expected
				return
			}
			// process the new line and restart the timeout
			if !flushTimer.Stop() {
				// flushTimer.stop() doesn't prevent the timer to tick,
				// makes sure the event is consumed to avoid sending
				// just one piece of the content.
				select {
				case <-flushTimer.C:
				default:
				}
			}
			p.process(message)
			flushTimer.Reset(p.flushTimeout)
		case <-flushTimer.C:
			// no chunk has been collected since a while,
			// the content is supposed to be complete.
			p.sendLine()
		}
	}
}

// process buffers and aggregates partial lines
func (p *MultiLineParser) process(input *DecodedInput) {
	msg, err := p.parser.Parse(input.content)
	if err != nil {
		log.Debug(err)
	}
	// track the raw data length and the timestamp so that the agent tails
	// from the right place at restart
	p.rawDataLen += input.rawDataLen
	p.timestamp = msg.Timestamp
	p.status = msg.Status
	p.buffer.Write(msg.Content)

	if !msg.IsPartial || p.buffer.Len() >= p.lineLimit {
		// the current chunk marks the end of an aggregated line
		p.sendLine()
	}
}

// sendBuffer forwards the content stored in the buffer
// to the output channel.
func (p *MultiLineParser) sendLine() {
	defer func() {
		p.buffer.Reset()
		p.rawDataLen = 0
	}()

	content := make([]byte, p.buffer.Len())
	copy(content, p.buffer.Bytes())
	if len(content) > 0 || p.rawDataLen > 0 {
		p.outputChan <- NewMessage(content, p.status, p.rawDataLen, p.timestamp)
	}
}
