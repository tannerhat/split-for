package splitter

import "github.com/sirupsen/logrus"

type SplitterOption[J any, R any] func(s *splitter[J, R])

func WithLogger[J any, R any](log logrus.FieldLogger) SplitterOption[J, R] {
	return func(s *splitter[J, R]) {
		s.log = log
	}
}

func WithFunction[J any, R any](f func(J) R, workerCount int) SplitterOption[J, R] {
	return WithErrorFunction(
		func(job J) (R, error) {
			return f(job), nil
		}, workerCount)
}

func WithErrorFunction[J any, R any](f func(J) (R, error), workerCount int) SplitterOption[J, R] {
	return func(s *splitter[J, R]) {
		for i := 0; i < workerCount; i++ {
			s.operations = append(s.operations, f)
		}
	}
}

func WithFunctions[J any, R any](funcs []func(J) R) SplitterOption[J, R] {
	return func(s *splitter[J, R]) {
		for _, f := range funcs {
			withErr := func(fn func(J) R) func(J) (R, error) {
				return func(job J) (R, error) {
					return fn(job), nil
				}
			}(f)
			s.operations = append(s.operations, withErr)
		}
	}
}

func WithErrorFunctions[J any, R any](funcs []func(J) (R, error)) SplitterOption[J, R] {
	return func(s *splitter[J, R]) {
		s.operations = funcs
	}
}

func StopOnError[J any, R any]() SplitterOption[J, R] {
	return func(s *splitter[J, R]) {
		s.stopOnError = true
	}
}
