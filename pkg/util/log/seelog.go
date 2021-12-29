package log

import (
	"errors"
	"sync"

	"github.com/cihub/seelog"
)

// seeLogger is a ddLogger implementation that calls through to
// embedded seelog instances.
type seeLogger struct {
	// mu protects all fields in this struct
	mu sync.Mutex

	// currentContext is the context the loggers are currently configured with
	currentContext []interface{}

	// currentStackDepth is the stack depth the loggers are currently configured with
	currentStackDepth int

	// loggers are the seelog loggers to which log messages will be sent.
	loggers map[string]seelog.LoggerInterface
}

var _ ddLogger = (*seeLogger)(nil)

func newSeeLogger() *seeLogger {
	return &seeLogger{
		loggers: make(map[string]seelog.LoggerInterface),
	}
}

func (l *seeLogger) registerLogger(name string, sl seelog.LoggerInterface) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.loggers[name]; ok {
		return errors.New("logger already registered with that name")
	}
	l.loggers[name] = sl
	return nil
}

func (l *seeLogger) unregisterLogger(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.loggers, name)
	return nil
}

func (l *seeLogger) trace(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Trace(message)
	}
}

func (l *seeLogger) debug(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Debug(message)
	}
}

func (l *seeLogger) info(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Info(message)
	}
}

func (l *seeLogger) warn(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Warn(message) //nolint:errcheck
	}
}

func (l *seeLogger) error(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Error(message) //nolint:errcheck
	}
}

func (l *seeLogger) critical(message string, context []interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setupLoggers(context, depth)
	for _, sl := range l.loggers {
		sl.Critical(message) //nolint:errcheck
	}
}

func (l *seeLogger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, sl := range l.loggers {
		sl.Flush()
	}
}

// setupLoggers sets the context and depth for all loggers.  It must be called
// with the mutex held.
func (l *seeLogger) setupLoggers(context []interface{}, depth int) {
	for _, sl := range l.loggers {
		sl.SetContext(context)            //nolint:errcheck
		sl.SetAdditionalStackDepth(depth) //nolint:errcheck
	}
}
