package parsertesting

import "github.com/DataDog/datadog-agent/pkg/logs/internal/parsers"

// ParserTester supports testing parsers, setting up channels and
// handling input and output.
type ParserTester struct {
	parser parsers.Parser
	input  chan []byte
	output chan parsers.Message
}

// NewParserTester creates a new tester for the given parser. It
// starts the parser
func NewParserTester(parser parsers.Parser) *ParserTester {
	input := make(chan []byte, 1)
	output := make(chan parsers.Message, 1)
	pt := &ParserTester{parser, input, output}
	pt.parser.Start(input, output)
	return pt
}

// SendLine sends a line to the parser.
func (pt *ParserTester) SendLine(line []byte) {
	pt.input <- line
}

// GetMessage gets a message from the parser, blocking until it arrives.
func (pt *ParserTester) GetMessage() parsers.Message {
	return <-pt.output
}

// Stop stops the parser
func (pt *ParserTester) Stop() {
	close(pt.input)
	pt.parser.Wait()
}
