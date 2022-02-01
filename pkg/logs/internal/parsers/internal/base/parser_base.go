package base

import (
	"github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// ParserBase is a base for a simple parser that maps one input line to one message.
//
// It is intended to be embedded in the struct for each such parser.  That
// parser should initialize it to its zero value and set the `Process` field.
type ParserBase struct {
	// input carries input to the parser
	input chan []byte

	// output carries the parser's output
	output chan parsers.Message

	// stopped will be closed whne the worker goroutine is finished
	stopped chan struct{}

	// Process performs the actual parsing.  Note that messages will be sent to
	// output even if an error is returned, in the interest of not losing data.
	Process func(input []byte) (parsers.Message, error)
}

// Implementation of Parser#Start
func (p *ParserBase) Start(input chan []byte, output chan parsers.Message) {
	p.input = input
	p.output = output
	p.stopped = make(chan struct{}, 0)

	go p.run()
}

// Implementation of Parser#Wait
func (p *ParserBase) Wait() {
	<-p.stopped
}

// run executes in a dedicated goroutine and takes messages from the input,
// processes them, and sends them to the output.
func (p *ParserBase) run() {
	defer func() {
		close(p.stopped)
	}()
	for line := range p.input {
		msg, err := p.Process(line)
		if err != nil {
			log.Debug(err)
		} else {
			p.output <- msg
		}
	}
}
