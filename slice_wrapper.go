package splitter

import (
	"context"
)

type job[J any] struct {
	index int
	value J
}

// workerFuncsToSliceWorkerFuncs takes WorkerFuncs and a results slice and creates a func(job job[J]) R
// that runs the original func on job.value and sets the value as results[job.idx]
func workerFuncsToSliceWorkerFuncs[J any, R any](funcs WorkerFuncs[J, R], results []R) WorkerFuncs[job[J], R] {
	ret := make(WorkerFuncs[job[J], R], len(funcs))
	for i, f := range funcs {
		ret[i] = func(j job[J]) (R, error) {
			res, err := f(j.value)
			results[j.index] = res
			return res, err
		}
	}
	return ret
}

// SplitSlice is a wrapper around the split function that takes a slice of jobs and returns a slice
// of results where f(job[n]) == results[n].
func SplitSlice[J any, R any](ctx context.Context, jobs []J, funcs WorkerFuncs[J, R]) ([]R, error) {
	results := make([]R, len(jobs))

	// create splitter with the jobChan and funcToSliceFunc as the func, the wrapper will set the
	// results into the results slice in the order the were inserted into the jobChan
	sliceSplitter := NewSplitter[job[J], R](ctx, workerFuncsToSliceWorkerFuncs(funcs, results), StopOnError())

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
