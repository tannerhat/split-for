package splitter

import "github.com/sirupsen/logrus"

type SplitterOption[J any, R any]  func(s *splitter[J, R] )

func WithLogger[J any, R any] (log logrus.FieldLogger) SplitterOption[J, R] {
	return func(s *splitter[J, R] ) {
		s.log = log
	}
}
