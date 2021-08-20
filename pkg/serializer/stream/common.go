package stream

// OnErrItemTooBigPolicy defines the behavior when OnErrItemTooBig occurs.
type OnErrItemTooBigPolicy int

const (
	// DropItemOnErrItemTooBig skips the error and continues when ErrItemTooBig is encountered
	DropItemOnErrItemTooBig OnErrItemTooBigPolicy = iota

	// FailOnErrItemTooBig returns the error and stop when ErrItemTooBig is encountered
	FailOnErrItemTooBig
)
