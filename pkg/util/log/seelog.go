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

func (l *seeLogger) trace(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Trace(message)
	}
}

func (l *seeLogger) debug(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Debug(message)
	}
}

func (l *seeLogger) info(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Info(message)
	}
}

func (l *seeLogger) warn(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Warn(message) //nolint:errcheck
	}
}

func (l *seeLogger) error(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Error(message) //nolint:errcheck
	}
}

func (l *seeLogger) critical(message string, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.setDepth(depth)
	for _, sl := range l.loggers {
		sl.Critical(message) //nolint:errcheck
	}
}

func (l *seeLogger) withContext(context []interface{}) ddLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	sub := newSeeLogger()
	for n, sl := range l.loggers {
		subsl, err := seelog.CloneLogger(sl)
		if err != nil {
			// for un-cloneable loggers, just use the existing logger
			subsl = sl
		} else {
			subsl.SetContext(context)
		}
		sub.loggers[n] = subsl
	}
	return sub
}

func (l *seeLogger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, sl := range l.loggers {
		sl.Flush()
	}
}

// setDepth sets the depth for all loggers.  It must be called with the
// mutex held.
func (l *seeLogger) setDepth(depth int) {
	for _, sl := range l.loggers {
		sl.SetAdditionalStackDepth(depth) //nolint:errcheck
	}
}
