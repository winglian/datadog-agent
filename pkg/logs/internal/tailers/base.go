package tailers

import (
	"context"
	"errors"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// TODO: error handling? update source.Status?

// A RunFunc implements the "loop" of a tailer.  It should run until the given
// context is cancelled.
type RunFunc = func(context.Context)

// TailerBase implements basic tailer functionality.  It is intended to be
// embedded in the struct for each tailer implementation.
type TailerBase struct {
	source     *config.LogSource
	outputChan chan *message.Message

	// run implements the "loop" of the tailer.  It is called in a dedicated
	// goroutine.
	run RunFunc

	// cancel is the CancelFunc for the context passed to run
	cancel context.CancelFunc

	// done is closed when the tailer has stopped
	done chan struct{}
}

// NewTailerBase creates a new TailerBase.  This should be used as part of each
// tailer's constructor.
func NewTailerBase(source *config.LogSource, run RunFunc, outputChan chan *message.Message) TailerBase {
	return TailerBase{
		source:     source,
		outputChan: outputChan,
		run:        run,
		done:       make(chan struct{}, 1),
	}
}

/// Start starts the tailer.
func (t *TailerBase) Start() error {
	if t.cancel != nil {
		return errors.New("Tailer has already been started")
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

// Stop stops the tailer, blocking until it is finished.
//
// A tailer cannot be started again after it has stopped.
func (t *TailerBase) Stop() {
	t.cancel()
	<-t.done
}
