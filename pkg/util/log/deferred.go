package log

import (
	"fmt"
	"os"
	"sync"

	"github.com/cihub/seelog"
)

// deferredCall describes a deferred log call that will be executed when a full
// logger is configured.
type deferredCall func(logger *DDLogger)

// deferredLogger defers logs until a full logger is available, at which time
// `forward` can repeat the calls.  deferredLogger instances do not show
// calling functions, so stack depth is ignored.
type deferredLogger struct {
	// mu covers all other fields in this struct
	mu sync.Mutex

	// context is an array of arbitrary values that provide context for
	// log messages
	context []interface{}

	// calls is the list of deferred calls that will be executed when forwarding begins
	calls []deferredCall

	// forwardTo is the logger to which all calls are forwarded.  If this is nil, then
	// calls are deferred.
	forwardTo *DDLogger
}

var _ ddLogger = (*deferredLogger)(nil)

func newDeferredLogger() *deferredLogger {
	return &deferredLogger{}
}

func (l *deferredLogger) call(call deferredCall) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.forwardTo != nil {
		call(l.forwardTo)
	} else {
		l.calls = append(l.calls, call)
	}
}

func (l *deferredLogger) trace(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.trace(message, context, depth) })
}

func (l *deferredLogger) debug(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.debug(message, context, depth) })
}

func (l *deferredLogger) info(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.info(message, context, depth) })
}

func (l *deferredLogger) warn(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.warn(message, context, depth) })
}

func (l *deferredLogger) error(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.error(message, context, depth) })
	l.fallback(seelog.ErrorLvl, message)
}

func (l *deferredLogger) critical(message string, context []interface{}, depth int) {
	l.call(func(logger *DDLogger) { logger.critical(message, context, depth) })
	l.fallback(seelog.CriticalLvl, message)
}

func (l *deferredLogger) flush() {
	// nothing to do
}

// For error and critical logs, messages are sent to stderr immediately, in
// case those errors are so severe that the logs are never forwarded.
func (l *deferredLogger) fallback(level seelog.LogLevel, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", level.String(), message)
}

// forward begins forwarding calls to the given logger, beginning with any
// deferred calls.  All child loggers are also forwarded.
func (l *deferredLogger) forward(logger *DDLogger) {
	l.mu.Lock()

	l.forwardTo = logger
	for _, call := range l.calls {
		call(logger)
	}

	l.mu.Unlock()
}
