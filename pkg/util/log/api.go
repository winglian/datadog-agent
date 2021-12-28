package log

import (
	"fmt"

	"github.com/cihub/seelog"
)

// DDLogger allows sending log messages.
type DDLogger struct {
	inner      ddLogger
	level      seelog.LogLevel
	stackDepth int
}

func newDDLogger(inner ddLogger, level seelog.LogLevel, stackDepth int) *DDLogger {
	return &DDLogger{inner, level, stackDepth}
}

// Trace logs at the trace level
func (l *DDLogger) Trace(v ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		entry := buildLogEntry(v...)
		l.inner.trace(entry, l.stackDepth)
	}
}

// Tracef logs with format at the trace level
func (l *DDLogger) Tracef(format string, params ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.inner.trace(fmt.Sprintf(format, params...), l.stackDepth)
	}
}

// Debug logs at the debug level
func (l *DDLogger) Debug(v ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		entry := buildLogEntry(v...)
		l.inner.debug(entry, l.stackDepth)
	}
}

// Debugf logs with format at the debug level
func (l *DDLogger) Debugf(format string, params ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.inner.debug(fmt.Sprintf(format, params...), l.stackDepth)
	}
}

// Info logs at the info level
func (l *DDLogger) Info(v ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		entry := buildLogEntry(v...)
		l.inner.info(entry, l.stackDepth)
	}
}

// Infof logs with format at the info level
func (l *DDLogger) Infof(format string, params ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.inner.info(fmt.Sprintf(format, params...), l.stackDepth)
	}
}

// Warn logs at the warn level and returns an error containing the formated log message
func (l *DDLogger) Warn(v ...interface{}) error {
	if l.shouldLog(seelog.WarnLvl) {
		entry := buildLogEntry(v...)
		l.inner.warn(entry, l.stackDepth)
	}
	return formatError(v...)
}

// Warnf logs with format at the warn level and returns an error containing the formated log message
func (l *DDLogger) Warnf(format string, params ...interface{}) error {
	if l.shouldLog(seelog.WarnLvl) {
		l.inner.warn(fmt.Sprintf(format, params...), l.stackDepth)
	}
	return formatErrorf(format, params...)
}

// Error logs at the error level and returns an error containing the formated log message
func (l *DDLogger) Error(v ...interface{}) error {
	if l.shouldLog(seelog.ErrorLvl) {
		entry := buildLogEntry(v...)
		l.inner.error(entry, l.stackDepth)
	}
	return formatError(v...)
}

// Errorf logs with format at the error level and returns an error containing the formated log message
func (l *DDLogger) Errorf(format string, params ...interface{}) error {
	if l.shouldLog(seelog.ErrorLvl) {
		l.inner.error(fmt.Sprintf(format, params...), l.stackDepth)
	}
	return formatErrorf(format, params...)
}

// Critical logs at the critical level and returns an error containing the formated log message
func (l *DDLogger) Critical(v ...interface{}) error {
	if l.shouldLog(seelog.CriticalLvl) {
		entry := buildLogEntry(v...)
		l.inner.critical(entry, l.stackDepth)
	}
	return formatError(v...)
}

// Criticalf logs with format at the critical level and returns an error containing the formated log message
func (l *DDLogger) Criticalf(format string, params ...interface{}) error {
	if l.shouldLog(seelog.CriticalLvl) {
		l.inner.critical(fmt.Sprintf(format, params...), l.stackDepth)
	}
	return formatErrorf(format, params...)
}

// WithStackDepth derives a new logger from this one which will strip the given
// number of addition frames from the stack to determine the name of the
// calling function.
func (l *DDLogger) WithStackDepth(stackDepth int) *DDLogger {
	return newDDLogger(l.inner, l.level, l.stackDepth+stackDepth)
}

// WithContext derives a new logger from this one which includes the given
// context values in every log message.
func (l *DDLogger) WithContext(context ...interface{}) *DDLogger {
	inner := l.inner.withContext(context)
	return newDDLogger(inner, l.level, l.stackDepth)
}

// Flush flushes any cached output and waits for it to complete.
func (l *DDLogger) Flush() {
	l.inner.flush()
}

// shouldLog returns true if the logger is configured to log at
// the given level.
func (l *DDLogger) shouldLog(lvl seelog.LogLevel) bool {
	return lvl >= l.level
}
