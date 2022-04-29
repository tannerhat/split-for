package splitter

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type splitter[J any, R any] struct {
	jobs        chan J
	results     chan R
	errors      chan error
	operations  []func(J) (R, error)
	stopOnError bool

	workerDone sync.WaitGroup
	cancelLock sync.Mutex
	done       chan bool

	log logrus.FieldLogger
}

func New[J any, R any](ctx context.Context, opts ...SplitterOption[J, R]) *splitter[J, R] {
	sf := &splitter[J, R]{
		jobs:       make(chan J, 1000),
		results:    make(chan R, 1000),
		errors:     make(chan error, 1000),
		operations: make([]func(J) (R, error), 0),
		workerDone: sync.WaitGroup{},
		cancelLock: sync.Mutex{},
		done:       make(chan bool, 0),
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
			sf.Cancel()
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
					if err != nil {
						sf.errors <- err
						if sf.stopOnError {
							sf.Cancel()
							return
						}
					}
					sf.results <- res
				default:
				}
			}
		}(i, f)
	}

	// once the workers exit, close the results channel
	go func() {
		sf.workerDone.Wait()
		close(sf.results)
		close(sf.errors)
	}()
	return sf
}

//
func (sf *splitter[J, R]) Do(job J) {
	sf.jobs <- job
}

func (sf *splitter[J, R]) Errors() <-chan error {
	return sf.errors
}

func (sf *splitter[J, R]) Results() <-chan R {
	return sf.results
}

func (sf *splitter[J, R]) Cancel() {
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
	close(sf.done)
}

// Done signals to the splitter that no more jobs are coming in and allows workers
// to exit once they have completed the current jobs.
func (sf *splitter[J, R]) Done() {
	close(sf.jobs)
}
