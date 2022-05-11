package splitter

import (
	"context"
)

type job[J any] struct {
	index int
	value J
}

// funcToSliceFunc takes a func(J) R and a results slice and creates a func(job job[J]) R
// that runs the original func on job.value and sets the value as results[job.idx]
func funcToSliceFunc[J any, R any](f func(J) R, results []R) func(job[J]) R {
	return func(j job[J]) R {
		results[j.index] = f(j.value)
		return results[j.index]
	}
}

// SplitSlice is a wrapper around the split function that takes a slice of jobs and returns a slice
// of results where f(job[n]) == results[n].
func SplitSlice[J any, R any](ctx context.Context, jobs []J, f func(J) R, workerCount int) ([]R, error) {
	results := make([]R, len(jobs))

	// create splitter with the jobChan and funcToSliceFunc as the func, the wrapper will set the
	// results into the results slice in the order the were inserted into the jobChan
	sliceSplitter := NewSplitter[job[J], R](ctx, WithFunction(funcToSliceFunc(f, results), workerCount), StopOnError[job[J], R]())

	// load the job channel with all the jobs
	for jobIdx := 0; jobIdx < len(jobs); jobIdx++ {
		sliceSplitter.Do(job[J]{index: jobIdx, value: jobs[jobIdx]})
	}
	// close the job channel to signal to the splitter that no more jobs are coming
	sliceSplitter.Done()

	// ok now it's just a matter of draining the results queue to make sure the slice
	// gets set for each result
	for {
		select {
		case _, ok := <-sliceSplitter.Results():
			if !ok {
				// yay we did it, all results are in
				return results, nil
			}
		case err, ok := <-sliceSplitter.Errors():
			// make sure this is an error and not channel close
			if ok {
				// something went wrong, quit
				return nil, err
			}
		}
	}
}
