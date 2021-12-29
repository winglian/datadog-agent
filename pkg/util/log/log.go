// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package log implements logging for the datadog agent.  It wraps seelog, and
// supports logging to multiple destinations, buffering messages logged before
// setup, and scrubbing secrets from log messages.
//
// Compatibility
//
// This module is exported and can be used outside of the datadog-agent
// repository, but is not designed as a general-purpose logging system.  Its
// API may change incompatibly.
package log

import (
	"errors"

	"github.com/cihub/seelog"
)

var (
	defaultStackDepth = 4

	// Default is the default logger. Package-level functions in this package
	// call this logger.
	Default *DDLogger

	// DefaultJmx is the default logger for JMX.  Package-level JMX functions
	// in this package call this logger.
	DefaultJmx *DDLogger
)

func init() {
	// begin the process with a deferred logger, that will forward to the real
	// thing once SetupLogging is called.  These log at the Trace level so that
	// all messages are captured.  When those messages are forwarded, they will
	// be level-filtered again.
	Default = newDDLogger(newDeferredLogger(), seelog.TraceLvl)
	DefaultJmx = newDDLogger(newDeferredLogger(), seelog.TraceLvl)
}

// SetupLogger setup agent wide logger
func SetupLogger(i seelog.LoggerInterface, level string) {
	Default = setupCommonLogger(Default, i, level)
}

// SetupJMXLogger setup JMXfetch specific logger
func SetupJMXLogger(i seelog.LoggerInterface, level string) {
	DefaultJmx = setupCommonLogger(DefaultJmx, i, level)
}

func setupCommonLogger(current *DDLogger, i seelog.LoggerInterface, level string) *DDLogger {
	sl := newSeeLogger()
	sl.registerLogger("", i)

	scrl := newScrubbingLogger(sl)

	lvl, ok := seelog.LogLevelFromString(level)
	if !ok {
		lvl = seelog.InfoLvl
	}

	rv := newDDLogger(scrl, lvl)

	// if the previous logger was a deferredLogger, forward its messages to the new one
	// TODO: refactor
	deferred, _ := current.inner.(*deferredLogger)
	if deferred != nil {
		deferred.forward(rv)
	}

	return rv
}

// Trace logs at the trace level
func Trace(v ...interface{}) {
	Default.Trace(v...)
}

// Tracef logs with format at the trace level
func Tracef(format string, params ...interface{}) {
	Default.Tracef(format, params...)
}

// TracecStackDepth logs at the trace level with context and the current stack
// depth plus the additional given one
func TracecStackDepth(message string, depth int, context ...interface{}) {
	Default.TracecStackDepth(message, depth, context...)
}

// Tracec logs at the trace level with context
func Tracec(message string, context ...interface{}) {
	Default.Tracec(message, context...)
}

// TraceStackDepth logs at the trace level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func TraceStackDepth(depth int, v ...interface{}) {
	Default.TraceStackDepth(depth, v...)
}

// Debug logs at the Debug level
func Debug(v ...interface{}) {
	Default.Debug(v...)
}

// Debugf logs with format at the Debug level
func Debugf(format string, params ...interface{}) {
	Default.Debugf(format, params...)
}

// DebugcStackDepth logs at the Debug level with context and the current stack
// depth plus the additional given one
func DebugcStackDepth(message string, depth int, context ...interface{}) {
	Default.DebugcStackDepth(message, depth, context...)
}

// Debugc logs at the Debug level with context
func Debugc(message string, context ...interface{}) {
	Default.Debugc(message, context...)
}

// DebugStackDepth logs at the Debug level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func DebugStackDepth(depth int, v ...interface{}) {
	Default.DebugStackDepth(depth, v...)
}

// Info logs at the info level
func Info(v ...interface{}) {
	Default.Info(v...)
}

// Infof logs with format at the info level
func Infof(format string, params ...interface{}) {
	Default.Infof(format, params...)
}

// InfocStackDepth logs at the info level with context and the current stack
// depth plus the additional given one
func InfocStackDepth(message string, depth int, context ...interface{}) {
	Default.InfocStackDepth(message, depth, context...)
}

// Infoc logs at the info level with context
func Infoc(message string, context ...interface{}) {
	Default.Infoc(message, context...)
}

// InfoStackDepth logs at the info level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func InfoStackDepth(depth int, v ...interface{}) {
	Default.InfoStackDepth(depth, v...)
}

// Warn logs at the warn level
func Warn(v ...interface{}) error {
	return Default.Warn(v...)
}

// Warnf logs with format at the warn level
func Warnf(format string, params ...interface{}) error {
	return Default.Warnf(format, params...)
}

// WarncStackDepth logs at the warn level with context and the current stack
// depth plus the additional given one
func WarncStackDepth(message string, depth int, context ...interface{}) error {
	return Default.WarncStackDepth(message, depth, context...)
}

// Warnc logs at the warn level with context
func Warnc(message string, context ...interface{}) error {
	return Default.Warnc(message, context...)
}

// WarnStackDepth logs at the warn level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func WarnStackDepth(depth int, v ...interface{}) error {
	return Default.WarnStackDepth(depth, v...)
}

// Error logs at the error level
func Error(v ...interface{}) error {
	return Default.Error(v...)
}

// Errorf logs with format at the error level
func Errorf(format string, params ...interface{}) error {
	return Default.Errorf(format, params...)
}

// ErrorcStackDepth logs at the error level with context and the current stack
// depth plus the additional given one
func ErrorcStackDepth(message string, depth int, context ...interface{}) error {
	return Default.ErrorcStackDepth(message, depth, context...)
}

// Errorc logs at the error level with context
func Errorc(message string, context ...interface{}) error {
	return Default.Errorc(message, context...)
}

// ErrorStackDepth logs at the error level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func ErrorStackDepth(depth int, v ...interface{}) error {
	return Default.ErrorStackDepth(depth, v...)
}

// Critical logs at the critical level
func Critical(v ...interface{}) error {
	return Default.Critical(v...)
}

// Criticalf logs with format at the critical level
func Criticalf(format string, params ...interface{}) error {
	return Default.Criticalf(format, params...)
}

// CriticalcStackDepth logs at the critical level with context and the current stack
// depth plus the additional given one
func CriticalcStackDepth(message string, depth int, context ...interface{}) error {
	return Default.CriticalcStackDepth(message, depth, context...)
}

// Criticalc logs at the critical level with context
func Criticalc(message string, context ...interface{}) error {
	return Default.Criticalc(message, context...)
}

// CriticalStackDepth logs at the critical level and the current stack depth plus the
// additional given one and returns an error containing the formated log
// message
func CriticalStackDepth(depth int, v ...interface{}) error {
	return Default.CriticalStackDepth(depth, v...)
}

// JMXError Logs for JMX check
func JMXError(v ...interface{}) error {
	return DefaultJmx.Error(v...)
}

//JMXInfo Logs
func JMXInfo(v ...interface{}) {
	DefaultJmx.Info(v...)
}

// Flush flushes all the messages in the logger.
func Flush() {
	Default.Flush()
	DefaultJmx.Flush()
}

// ReplaceLogger allows replacing the internal logger, returns old logger
func ReplaceLogger(l seelog.LoggerInterface) seelog.LoggerInterface {
	// this is equivalent to re-initializing the logger with the given
	// interface and the existing level
	old := Default
	SetupLogger(l, Default.level.String())
	// TODO: refactor
	if scrl, ok := old.inner.(*scrubbingLogger); ok {
		if sl, ok := scrl.inner.(*seeLogger); ok {
			return sl.loggers[""]
		}
	}
	return nil
}

func getCurrentSeeLogger() (*seeLogger, error) {
	// TODO: refactor
	if scrl, ok := Default.inner.(*scrubbingLogger); ok {
		if sl, ok := scrl.inner.(*seeLogger); ok {
			return sl, nil
		}
	}

	return nil, errors.New("cannot register: logger not initialized")
}

// RegisterAdditionalLogger registers an additional logger for logging.  The
// name is arbitrary.  The logger passed to SetupLogging is registered with the
// name "".
func RegisterAdditionalLogger(n string, l seelog.LoggerInterface) error {
	sl, err := getCurrentSeeLogger()
	if err != nil {
		return err
	}
	return sl.registerLogger(n, l)
}

// UnregisterAdditionalLogger unregisters additional logger with name n
func UnregisterAdditionalLogger(n string) error {
	sl, err := getCurrentSeeLogger()
	if err != nil {
		return err
	}
	return sl.unregisterLogger(n)
}

// ShouldLog returns whether a given log level should be logged by the default logger
func ShouldLog(lvl seelog.LogLevel) bool {
	return Default.shouldLog(lvl)
}

// GetLogLevel returns a seelog native representation of the current
// log level
func GetLogLevel() (seelog.LogLevel, error) {
	return Default.level, nil
}

// ChangeLogLevel changes the current log level, valide levels are trace, debug,
// info, warn, error, critical and off, it requires a new seelog logger because
// an existing one cannot be updated
func ChangeLogLevel(l seelog.LoggerInterface, level string) error {
	// this is equivalent to simply setting up the logger again
	SetupLogger(l, level)
	return nil
}
