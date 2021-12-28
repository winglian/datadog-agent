package log

// ddLogger is an interface to allow sending log messages.  This is a private interface
// that has a more limited API than the DDLogger type, which wraps it.
//
// Loggers are threadsafe.
type ddLogger interface {
	// trace logs a message at the TRACE level, stripping depth stack frames
	trace(message string, depth int)

	// debug logs a message at the DEBUG level, stripping depth stack frames
	debug(message string, depth int)

	// info logs a message at the INFO level, stripping depth stack frames
	info(message string, depth int)

	// warn logs a message at the WARN level, stripping depth stack frames
	warn(message string, depth int)

	// error logs a message at the ERROR level, stripping depth stack frames
	error(message string, depth int)

	// critical logs a message at the CRITICAL level, stripping depth stack frames
	critical(message string, depth int)

	// withContext derives a new logger from this one which will include the
	// given context in all log messages.
	withContext(context []interface{}) ddLogger

	// Flush flushes any cached output and waits for it to complete.
	flush()
}
