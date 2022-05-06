package splitter

import "github.com/sirupsen/logrus"

type SplitterOption[J any, R any] func(s *Splitter[J, R])

func WithLogger[J any, R any](log logrus.FieldLogger) SplitterOption[J, R] {
	return func(s *Splitter[J, R]) {
		s.log = log
	}
}

// WithFunction option tells the splitter to use the provided function to process jobs.
// workerCount determines how many routines running this function will be created.
func WithFunction[J any, R any](f func(J) R, workerCount int) SplitterOption[J, R] {
	return WithErrorFunction(
		func(job J) (R, error) {
			return f(job), nil
		}, workerCount)
}

func WithErrorFunction[J any, R any](f func(J) (R, error), workerCount int) SplitterOption[J, R] {
	return func(s *Splitter[J, R]) {
		s.operations = make([]func(J) (R, error), workerCount)
		for i := 0; i < workerCount; i++ {
			s.operations[i] = f
		}
	}
}

func WithFunctions[J any, R any](funcs []func(J) R) SplitterOption[J, R] {
	return func(s *Splitter[J, R]) {
		s.operations = make([]func(J) (R, error), len(funcs))
		for i, f := range funcs {
			withErr := func(fn func(J) R) func(J) (R, error) {
				return func(job J) (R, error) {
					return fn(job), nil
				}
			}(f)
			s.operations[i] = withErr
		}
	}
}

func WithErrorFunctions[J any, R any](funcs []func(J) (R, error)) SplitterOption[J, R] {
	return func(s *Splitter[J, R]) {
		s.operations = funcs
	}
}

func StopOnError[J any, R any]() SplitterOption[J, R] {
	return func(s *Splitter[J, R]) {
		s.stopOnError = true
	}
}
