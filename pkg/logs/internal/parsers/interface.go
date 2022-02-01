// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package parsers

// Parser supports transforming raw log "lines" into messages with some associated
// envelope metadata (timestamp, severity, etc.).  This is a low-level parser,
// meant to decipher on-the-wire and on-disk formats into log messages.  It does
// not interpret what the user would consider the message itself (e.g., syslog
// priority).
//
// This parsing comes after "line parsing" (breaking input into multiple lines) and
// before further processing and aggregation of log messages.
type Parser interface {
	// Start the parser, reading lines from input and writing the results to
	// output.
	Start(input chan []byte, output chan Message)

	// Wait for the parser to stop.  The input channel must be closed first.
	// This will return when the last input item has been processed and its
	// results sent to the output channel.
	Wait()
}
