package splitter

// WorkerFuncs are needed by the splitter function. Each func gets its own worker and processes
// jobs independently of the other funcs. They can take any type and return any type, but they also
// need to return an error. This is because the splitter supports forwarding errors so all functions
// need to return them. There's a few convenience functions for converting common function signatures
// into type WorkerFunc.
type WorkerFuncs[J any, R any] []func(J) (R, error)

// FromFunction takes func f and creates WorkerFuncs where there's workerCount copies of a function
// f' such that f'(J) ->  (f(J), nil.(error))
func FromFunction[J any, R any](f func(J) R, workerCount int) WorkerFuncs[J, R] {
	return FromErrorFunction[J, R](
		func(job J) (R, error) {
			return f(job), nil
		}, workerCount)
}

// FromErrorFunction takes func f and creates WorkerFuncs where there's workerCount copies of a f
func FromErrorFunction[J any, R any](f func(J) (R, error), workerCount int) WorkerFuncs[J, R] {
	operations := make([]func(J) (R, error), workerCount)
	for i := 0; i < workerCount; i++ {
		operations[i] = f
	}
	return operations
}

// FromFunctions takes a slice of functions funcs and creates WorkerFuncs where for each func in funcs, there
// is a func f' where f'(J) -> (funcs[i](J), nil.(error))
func FromFunctions[J any, R any](funcs []func(J) R) WorkerFuncs[J, R] {
	operations := make([]func(J) (R, error), len(funcs))
	for i, f := range funcs {
		withErr := func(fn func(J) R) func(J) (R, error) {
			return func(job J) (R, error) {
				return fn(job), nil
			}
		}(f)
		operations[i] = withErr
	}
	return operations
}

// FromErrorFunctions is just a cast, it doesn't do anything but it's here to complete the set. Also
// if I decide to make WorkerFuncs more than just a typedef, I'd need to add it anyway.
func FromErrorFunctions[J any, R any](funcs []func(J) (R, error)) WorkerFuncs[J, R] {
	return funcs
}
