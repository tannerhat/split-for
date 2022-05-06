package splitter

import (
	"context"
)

type job[J any] struct {
	index int
	value J
}

// func funcToSliceFunc[J any, R any](f func(J) (R, error), results []R) func(job[J]) (R, error) {
// 	return func(j job[J]) (R, error) {
// 		result, err := f(j.value)
// 		if err != nil {
// 			return result, err
// 		}
// 		results[j.index] = result
// 		return result, nil
// 	}
// }

func funcToSliceFunc[J any, R any](f func(J) R, results []R) func(job[J]) R {
	return func(j job[J]) R {
		results[j.index] = f(j.value)
		return results[j.index]
	}
}

func SplitSplice[J any, R any](ctx context.Context, jobs []J, f func(J) R, workerCount int) ([]R, error) {
	// This was code for if SplitSlice wanted to take all function options. For now it only
	// does what WithFunction doesn (single func + worker count)
	//
	// // make a dummy splitter to apply the splitter options to, the options will populate the functions
	// // slice so that we can modify it to set the right index into the results slice
	// splitter := &Splitter[J, R]{
	// 	operations: make([]func(J) (R, error), 0),
	// }
	// // apply options (the source of the operations slice)
	// for _, opt := range opts {
	// 	opt(splitter)
	// }

	// results slice to be populated
	results := make([]R, len(jobs))

	// // now convert the funcs we have into a type that takes a job[J] and returns R
	// // func(j J) R -> func(j job[J]) -> R
	// // this is done by wrapping the functions in another func that runs the original
	// // function on j.value, and puts the result in results[j.index]
	// funcs := make([]func(job[J]) (R, error), len(splitter.operations))
	// for i, f := range splitter.operations {
	// 	funcs[i] = funcToSliceFunc(f, results)
	// }

	sliceSplitter := New[job[J], R](ctx, WithFunction(funcToSliceFunc(f, results), workerCount))

	for jobIdx := 0; jobIdx < len(jobs); {
		err := sliceSplitter.Do(job[J]{index: jobIdx, value: jobs[jobIdx]})
		if err == nil {
			jobIdx++
		}
		// the only possible error is ErrJobChannelFull, in that case we just want to
		// retry the same job, so just not incrementing is enough

		// okay and check that we haven't been canceled
		select {
		case err := <-sliceSplitter.Errors():
			return results, err
		default:
			// ok everything's good still
		}
	}
	sliceSplitter.Done()

	// ok now it's just a matter of draining the results queue to make sure the slice
	// gets set

	for {
		select {
		case _, ok := <-sliceSplitter.Results():
			if !ok {
				// yay we did it, all results are in
				return results, nil
			}
		case err := <-sliceSplitter.Errors():
			return results, err
		}
	}
}
