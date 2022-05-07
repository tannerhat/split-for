package splitter

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// ErrJobChannelFull is returned on calls to Do where the internal job channel is full,
// the job is dropped and the user must try Do again if they want the given job processed.
var ErrJobChannelFull = fmt.Errorf("job channel is full, retry in a hot second")

// ErrContextCancel is returned on the error channel if the context passed to the constructor
// is Done before jobs finish processing. The splitter will stop doing work and close
// the results channel.
var ErrContextCancel = fmt.Errorf("context cancellation")

// ErrUserCancel is returned on the error channel if the user called Cancel before jobs
// finish processing. The splitter will stop doing work and close the results channel.
var ErrUserCancel = fmt.Errorf("user cancellation")
var errorCancelMsg = "cancelling due to processing error: %w"

// Splitter splits the work of the jobs in a channel among a worker
// for each function it starts with. Results are put into a channel.
// Errors can also be read from a channel
type Splitter[J any, R any] struct {
	jobs        <-chan J
	results     chan R
	errors      chan error
	operations  []func(J) (R, error)
	stopOnError bool

	workerDone sync.WaitGroup
	cancelLock sync.Mutex
	done       chan bool

	log logrus.FieldLogger
}

// NewSplitter creates a Splitter that will read from the given job chan using workers
// as defined by the SplitterOptions provided. The caller can add jobs to the channel before
// or after the splitter is created. Once the channel is closed, the splitter will exit after
// it finishes all the jobs inserted into the channel before the close.
func NewSplitter[J any, R any](ctx context.Context, jobs <-chan J, opts ...SplitterOption[J, R]) *Splitter[J, R] {
	sf := &Splitter[J, R]{
		jobs:       jobs,
		results:    make(chan R, 1000),
		errors:     make(chan error, 1000),
		operations: make([]func(J) (R, error), 0),
		workerDone: sync.WaitGroup{},
		cancelLock: sync.Mutex{},
		done:       make(chan bool),
	}

	// apply options (the source of the operations slice)
	for _, opt := range opts {
		opt(sf)
	}

	// this routine checks for context cancellation and kills the splitter if it happens
	go func() {
		select {
		case <-ctx.Done():
			// context cancelled, cancel the splitter and get outta here
			sf.cancel(ErrContextCancel)
		case <-sf.done:
			// splitter is done (either user cancel or all jobs done), get outta here
			return
		}
	}()

	for i, f := range sf.operations {
		sf.workerDone.Add(1)
		go func(id int, fn func(J) (R, error)) {
			defer sf.workerDone.Done()
			for {
				select {
				case <-sf.done:
					// it's cancelled, get outta here
					return
				case next, ok := <-sf.jobs:
					if !ok {
						// jobs channel is closed, no more jobs are coming so we can close this worker
						return
					}
					res, err := fn(next)
					// the function failed! either stop or just send through the error channel and move on
					if err != nil {
						if sf.stopOnError {
							// stopOnError means we give up after any error, cancel(err) will send err
							// to the error chan and cause all the workers to stop/
							sf.cancel(fmt.Errorf(errorCancelMsg, err))
							return
						} else {
							sf.errors <- err
						}
					} else {
						sf.results <- res
					}
				}
			}
		}(i, f)
	}

	// routine to monitor if the workers are done
	go func() {
		sf.workerDone.Wait()
		// once the workers exit, close the results and error channels
		close(sf.results)
		close(sf.errors)

		// now, if sf.done isn't closed, we close it (have to acquire lock)
		// pass nil because we don't want to send an error if we close here
		sf.cancel(nil)
	}()
	return sf
}

// Errors gives the Splitter's error channel. Any error from the workers along with context
// or user cancel will be passed to the error channel. Closed by the Splitter after workers exit.
func (sf *Splitter[J, R]) Errors() <-chan error {
	return sf.errors
}

// Results gives the Splitter's results channel. All results from the workers are sent to
// this channel. Closed by the Splitter after workers exit.
func (sf *Splitter[J, R]) Results() <-chan R {
	return sf.results
}

// Cancel forces the splitter to stop. The workers will exit (after they finish with the job they
// are currently processing).
func (sf *Splitter[J, R]) Cancel() {
	sf.cancel(ErrUserCancel)
}

// cancel handles any reason the splitter would stop before finishing the jobs. The workers
// will exit as a result.
func (sf *Splitter[J, R]) cancel(err error) {
	// lock because there are mutliple sources of Cancel (context, user, error in processing)
	// and we need to prevent multiple calls to close(sf.done)
	defer sf.cancelLock.Unlock()
	sf.cancelLock.Lock()
	// check for already closed
	select {
	case <-sf.done:
		return
	default:
	}
	if err != nil {
		sf.errors <- err
	}
	close(sf.done)
}
