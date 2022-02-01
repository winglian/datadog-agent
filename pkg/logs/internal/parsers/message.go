package parsers

// Message represents a message parsed from a single line of log data
type Message struct {
	// Content is the message content.  If this is nil then the message
	// should be considered empty and ignored.
	Content []byte

	// Status is the status parsed from the message, if any.
	Status string

	// Timestamp is the message timestamp from the source, if any, as an
	// ISO-8601-formatted string (./pkg/logs/config.DateFormat).  Log sources
	// which do not contain a timestamp (such as files) leave this set to "".
	Timestamp string

	// IsPartial indicates that this is a partial message.  If the parser
	// supports partial lines, then this is true only for the message returned
	// from the last parsed line in a multi-line message.
	IsPartial bool
}
