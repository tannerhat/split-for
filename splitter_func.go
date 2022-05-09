package splitter

import (
	"context"
	"fmt"
	"sync"
)

func Split[J any, R any](ctx context.Context, jobs <-chan J, opts ...SplitterOption[J, R]) (<-chan R, <-chan error, func()) {
	results := make(chan R, 1000)
	errors := make(chan error, 1000)
	operations := make([]func(J) (R, error), 0)
	workerDone := sync.WaitGroup{}
	cancelLock := sync.Mutex{}
	done := make(chan bool)
	stopOnError := true

	cancel := func(err error) {
		// lock because there are mutliple sources of Cancel (context, user, error in processing)
		// and we need to prevent multiple calls to close(sf.done)
		defer cancelLock.Unlock()
		cancelLock.Lock()
		// check for already closed
		select {
		case <-done:
			return
		default:
		}
		if err != nil {
			errors <- err
		}
		close(done)
	}

	// this routine checks for context cancellation and kills the splitter if it happens
	go func() {
		select {
		case <-ctx.Done():
			// context cancelled, cancel the splitter and get outta here
			cancel(ErrContextCancel)
		case <-done:
			// splitter is done (either user cancel or all jobs done), get outta here
			return
		}
	}()

	for i, f := range operations {
		workerDone.Add(1)
		go func(id int, fn func(J) (R, error)) {
			defer workerDone.Done()
			for {
				select {
				case <-done:
					// it's cancelled, get outta here
					return
				case next, ok := <-jobs:
					if !ok {
						// jobs channel is closed, no more jobs are coming so we can close this worker
						return
					}
					res, err := fn(next)
					// the function failed! either stop or just send through the error channel and move on
					if err != nil {
						if stopOnError {
							// stopOnError means we give up after any error, cancel(err) will send err
							// to the error chan and cause all the workers to stop/
							cancel(fmt.Errorf(errorCancelMsg, err))
							return
						} else {
							errors <- err
						}
					} else {
						results <- res
					}
				}
			}
		}(i, f)
	}

	// routine to monitor if the workers are done
	go func() {
		workerDone.Wait()
		// once the workers exit, close the results and error channels
		close(results)
		close(errors)

		// now, if sf.done isn't closed, we close it (have to acquire lock)
		// pass nil because we don't want to send an error if we close here
		cancel(nil)
	}()
	return results, errors, func() { cancel(ErrUserCancel) }
}
