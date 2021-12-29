package log

import (
	"fmt"

	"github.com/cihub/seelog"
)

// DDLogger allows sending log messages.
type DDLogger struct {
	inner ddLogger
	level seelog.LogLevel
}

func newDDLogger(inner ddLogger, level seelog.LogLevel) *DDLogger {
	return &DDLogger{inner, level}
}

// trace implements the full trace-level functionality for the public methods
func (l *DDLogger) trace(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.TraceLvl) {
		l.inner.trace(message, context, depth)
	}
}

// Trace logs at the trace level
func (l *DDLogger) Trace(v ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.trace(buildLogEntry(v...), nil, defaultStackDepth)
	}
}

// Tracef logs with format at the trace level
func (l *DDLogger) Tracef(format string, params ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.trace(fmt.Sprintf(format, params...), nil, defaultStackDepth)
	}
}

// TraceStackDepth logs at the trace level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) TracecStackDepth(message string, depth int, context ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.trace(message, context, depth)
	}
}

// Tracec logs at the trace level with context
func (l *DDLogger) Tracec(message string, context ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.trace(message, context, defaultStackDepth)
	}
}

// TracecStackDepth logs at the trace level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) TraceStackDepth(depth int, v ...interface{}) {
	if l.shouldLog(seelog.TraceLvl) {
		l.trace(buildLogEntry(v...), nil, depth)
	}
}

// debug implements the full debug-level functionality for the public methods
func (l *DDLogger) debug(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.DebugLvl) {
		l.inner.debug(message, context, depth)
	}
}

// Debug logs at the debug level
func (l *DDLogger) Debug(v ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.debug(buildLogEntry(v...), nil, defaultStackDepth)
	}
}

// Debugf logs with format at the debug level
func (l *DDLogger) Debugf(format string, params ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.debug(fmt.Sprintf(format, params...), nil, defaultStackDepth)
	}
}

// DebugStackDepth logs at the debug level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) DebugcStackDepth(message string, depth int, context ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.debug(message, context, depth)
	}
}

// Debugc logs at the debug level with context
func (l *DDLogger) Debugc(message string, context ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.debug(message, context, defaultStackDepth)
	}
}

// DebugcStackDepth logs at the debug level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) DebugStackDepth(depth int, v ...interface{}) {
	if l.shouldLog(seelog.DebugLvl) {
		l.debug(buildLogEntry(v...), nil, depth)
	}
}

// info implements the full info-level functionality for the public methods
func (l *DDLogger) info(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.InfoLvl) {
		l.inner.info(message, context, depth)
	}
}

// Info logs at the info level
func (l *DDLogger) Info(v ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.info(buildLogEntry(v...), nil, defaultStackDepth)
	}
}

// Infof logs with format at the info level
func (l *DDLogger) Infof(format string, params ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.info(fmt.Sprintf(format, params...), nil, defaultStackDepth)
	}
}

// InfoStackDepth logs at the info level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) InfocStackDepth(message string, depth int, context ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.info(message, context, depth)
	}
}

// Infoc logs at the info level with context
func (l *DDLogger) Infoc(message string, context ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.info(message, context, defaultStackDepth)
	}
}

// InfocStackDepth logs at the info level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) InfoStackDepth(depth int, v ...interface{}) {
	if l.shouldLog(seelog.InfoLvl) {
		l.info(buildLogEntry(v...), nil, depth)
	}
}

// warn implements the full warn-level functionality for the public methods
func (l *DDLogger) warn(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.InfoLvl) {
		l.inner.warn(message, context, depth)
	}
}

// Warn logs at the warn level and returns an error containing the formated log message
func (l *DDLogger) Warn(v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.WarnLvl) {
		l.inner.warn(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// Warnf logs with format at the warn level and returns an error containing the formated log message
func (l *DDLogger) Warnf(format string, params ...interface{}) error {
	message := fmt.Sprintf(format, params...)
	if l.shouldLog(seelog.WarnLvl) {
		l.inner.warn(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// WarnStackDepth logs at the warn level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) WarncStackDepth(message string, depth int, context ...interface{}) error {
	if l.shouldLog(seelog.WarnLvl) {
		l.warn(message, context, depth)
	}
	return scrubbedError(message)
}

// Warnc logs at the warn level with context
func (l *DDLogger) Warnc(message string, context ...interface{}) error {
	if l.shouldLog(seelog.WarnLvl) {
		l.warn(message, context, defaultStackDepth)
	}
	return scrubbedError(message)
}

// WarncStackDepth logs at the warn level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) WarnStackDepth(depth int, v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.WarnLvl) {
		l.warn(message, nil, depth)
	}
	return scrubbedError(message)
}

// error implements the full error-level functionality for the public methods
func (l *DDLogger) error(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.InfoLvl) {
		l.inner.error(message, context, depth)
	}
}

// Error logs at the error level and returns an error containing the formated log message
func (l *DDLogger) Error(v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.ErrorLvl) {
		l.inner.error(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// Errorf logs with format at the error level and returns an error containing the formated log message
func (l *DDLogger) Errorf(format string, params ...interface{}) error {
	message := fmt.Sprintf(format, params...)
	if l.shouldLog(seelog.ErrorLvl) {
		l.inner.error(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// ErrorStackDepth logs at the error level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) ErrorcStackDepth(message string, depth int, context ...interface{}) error {
	if l.shouldLog(seelog.ErrorLvl) {
		l.error(message, context, depth)
	}
	return scrubbedError(message)
}

// Errorc logs at the error level with context
func (l *DDLogger) Errorc(message string, context ...interface{}) error {
	if l.shouldLog(seelog.ErrorLvl) {
		l.error(message, context, defaultStackDepth)
	}
	return scrubbedError(message)
}

// ErrorcStackDepth logs at the error level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) ErrorStackDepth(depth int, v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.ErrorLvl) {
		l.error(message, nil, depth)
	}
	return scrubbedError(message)
}

// critical implements the full critical-level functionality for the public methods
func (l *DDLogger) critical(message string, context []interface{}, depth int) {
	if l.shouldLog(seelog.InfoLvl) {
		l.inner.critical(message, context, depth)
	}
}

// Critical logs at the critical level and returns an error containing the formated log message
func (l *DDLogger) Critical(v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.CriticalLvl) {
		l.inner.critical(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// Criticalf logs with format at the critical level and returns an error containing the formated log message
func (l *DDLogger) Criticalf(format string, params ...interface{}) error {
	message := fmt.Sprintf(format, params...)
	if l.shouldLog(seelog.CriticalLvl) {
		l.inner.critical(message, nil, defaultStackDepth)
	}
	return scrubbedError(message)
}

// CriticalStackDepth logs at the critical level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func (l *DDLogger) CriticalcStackDepth(message string, depth int, context ...interface{}) error {
	if l.shouldLog(seelog.CriticalLvl) {
		l.critical(message, context, depth)
	}
	return scrubbedError(message)
}

// Criticalc logs at the critical level with context
func (l *DDLogger) Criticalc(message string, context ...interface{}) error {
	if l.shouldLog(seelog.CriticalLvl) {
		l.critical(message, context, defaultStackDepth)
	}
	return scrubbedError(message)
}

// CriticalcStackDepth logs at the critical level with context and the current stack
// depth plus the additional given one
func (l *DDLogger) CriticalStackDepth(depth int, v ...interface{}) error {
	message := buildLogEntry(v...)
	if l.shouldLog(seelog.CriticalLvl) {
		l.critical(message, nil, depth)
	}
	return scrubbedError(message)
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
