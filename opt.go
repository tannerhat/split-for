package splitter

import "github.com/sirupsen/logrus"

type config struct {
	stopOnError bool
	log         logrus.FieldLogger
}

func (c *config) load(opts ...SplitterOption) *config {
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type SplitterOption func(c *config)

func WithLogger(log logrus.FieldLogger) SplitterOption {
	return func(c *config) {
		c.log = log
	}
}

func StopOnError() SplitterOption {
	return func(c *config) {
		c.stopOnError = true
	}
}
