package base

import (
	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// CombiningParserBase is a base for a parser which may combine multiple input
// lines into a single output message.
//
// Note that this is different from multiline logs support.  This type combines
// protocol-level messages containing fragments of log messages, such as from
// containers.
//
// It is intended to be embedded in the struct for each such parser, setting the `Process` field.
type CombiningParserBase struct {
	// input carries input to the parser
	input chan []byte

	// output carries the parser's output
	output chan parsers.Message

	// stopped will be closed whne the worker goroutine is finished
	stopped chan struct{}

	// acc is the accumulator used to accumulate partial messages
	acc parsers.Message

	// Process performs the actual parsing.  It should always return a message,
	// as that message may be flushed after a timeout.  Even in the case of an
	// error, the message will be sent to the output channel, in the interest
	// of not losing data.
	//
	// The second return value is false for complete output messages that can
	// be sent immediately, and true for partial messages.
	Process func(input []byte) (msg parsers.Message, partial bool, err error)
}

// Implementation of Parser#Start
func (p *CombiningParserBase) Start(input chan []byte, output chan parsers.Message) {
	p.input = input
	p.output = output
	p.stopped = make(chan struct{}, 0)

	go p.run()
}

// Implementation of Parser#Wait
func (p *CombiningParserBase) Wait() {
	<-p.stopped
}

// run executes in a dedicated goroutine and takes messages from the input,
// processes them, and sends them to the output.
func (p *CombiningParserBase) run() {
	defer func() {
		close(p.stopped)
	}()
	for line := range p.input {
		msg, partial, err := p.Process(line)
		if err != nil {
			log.Debug(err)
		}
		// NOTE: messages are still handled after an error, in hopes of not losing data
		p.acc.Content = append(p.acc.Content, msg.Content...)
		p.acc.Status = msg.Status
		p.acc.Timestamp = msg.Timestamp
		if partial {
			// TODO: set flush timer
		} else {
			p.flush()
		}
	}
}
func (p *CombiningParserBase) flush() {
	// copy the accumulated message to the output channel
	p.output <- p.acc
	// ..and then reset the accumulator
	p.acc.Content = nil
}
