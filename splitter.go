package splitter

import (
	"context"
	"fmt"

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
	results     <-chan R
	errors      <-chan error
	jobs        chan J
	operations  []func(J) (R, error)
	cancel      func()
	stopOnError bool

	log logrus.FieldLogger
}

// NewSplitter creates a Splitter that will read from the given job chan using workers
// as defined by the SplitterOptions provided. The caller can add jobs to the channel before
// or after the splitter is created. Once the channel is closed, the splitter will exit after
// it finishes all the jobs inserted into the channel before the close.
func NewSplitter[J any, R any](ctx context.Context, opts ...SplitterOption[J, R]) *Splitter[J, R] {
	sf := &Splitter[J, R]{}

	for _, opt := range opts {
		opt(sf)
	}
	sf.jobs = make(chan J, 1000)
	results, errors, cancel := Split[J, R](ctx, sf.jobs, sf.operations, sf.stopOnError)
	sf.errors = errors
	sf.results = results
	sf.cancel = cancel

	return sf
}

func (sf *Splitter[J, R]) Do(job J) error {
	select {
	case sf.jobs <- job:
		return nil
	default:
		return ErrJobChannelFull
	}
}

func (sf *Splitter[J, R]) Done() {
	close(sf.jobs)
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
	sf.cancel()
}
