package tailers

import (
	"context"
	"errors"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// TailerID uniquely identifies a tailer.  It is used as a key in the
// auditor registry, among other locations.  It should be unique to a single
// stream of log messages.  The format is arbitrary, but should begin with the
// name of the tailer type to avoid conflicts.
type TailerID = string

// Tailer produces log message.Message instances on an output channel, feeding
// into the logs-agent pipeline.
//
// A Tailer is an actor, running in a goroutine and handling events.
type Tailer interface {
	// Identifier returns the identifier of the tailer.
	Identifier() TailerID

	// Start starts the tailer.  Startup errors are returned immediately, but
	// errors during the runtime of the tailer must be handled separately.
	Start() error

	// Stop stops the tailer's goroutine and waits for it to complete. A tailer
	// cannot be started after it has stopped.  A tailer that did not
	// successfully start will never stop.
	Stop()
}

// A RunFunc implements the "loop" of a tailer.  It should run until the given
// context is cancelled.
type RunFunc = func(context.Context)

// TailerBase implements basic tailer functionality.  It is intended to be
// embedded in the struct for each tailer implementation.
type TailerBase struct {
	// Source contains the configuration for this tailer
	Source *config.LogSource

	// OutputChan is the channel to which the tailer sends messages
	OutputChan chan *message.Message

	// identifier contains the tailer's identifier, as returned from Identifier()
	identifier TailerID

	// run implements the "loop" of the tailer.  It is called in a dedicated
	// goroutine.
	run RunFunc

	// cancel is the CancelFunc for the context passed to run
	cancel context.CancelFunc

	// done is closed when the tailer has stopped
	done chan struct{}
}

var _ Tailer = &TailerBase{}

// NewTailerBase creates a new TailerBase.  This should be used as part of each
// tailer's constructor.  The `run` parameter is run in a dedicated goroutine.
func NewTailerBase(source *config.LogSource, run RunFunc, identifier TailerID, outputChan chan *message.Message) TailerBase {
	return TailerBase{
		Source:     source,
		OutputChan: outputChan,
		identifier: identifier,
		run:        run,
		done:       make(chan struct{}, 1),
	}
}

// Identifier gets the TailerID for this Tailer.  It implements Tailer#Identifier.
func (t *TailerBase) Identifier() TailerID {
	return t.identifier
}

// Start starts the tailer.  It implements Tailer#Start.
//
// Embedders that wish to perform additional work on startup should implement
// this method and call through to the embedded implementation after that
// startup work is complete.
func (t *TailerBase) Start() error {
	if t.cancel != nil {
		return errors.New("cannot start tailer after it is stopped")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	go func() {
		defer func() {
			close(t.done)
		}()
		t.run(ctx)
	}()

	return nil
}

// Stop stops the tailer, blocking until it is finished.  It implements Tailer#Stop.
//
// A tailer cannot be started again after it has stopped.
func (t *TailerBase) Stop() {
	t.cancel()
	<-t.done
}
