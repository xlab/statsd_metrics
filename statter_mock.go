package metrics

import (
	"time"

	log "github.com/xlab/suplog"
)

func newMockStatter(noop bool) Statter {
	return &mockStatter{
		noop: noop,
		fields: log.Fields{
			"module": "mock_statter",
		},
	}
}

type mockStatter struct {
	fields log.Fields
	noop   bool
}

func (s *mockStatter) Count(name string, value int64, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s: %v", name, value)
	return nil
}

func (s *mockStatter) Incr(name string, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s", name)
	return nil
}

func (s *mockStatter) Decr(name string, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s", name)
	return nil
}

func (s *mockStatter) Gauge(name string, value float64, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s: %v", name, value)
	return nil
}

func (s *mockStatter) Timing(name string, value time.Duration, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s: %v", name, value)
	return nil
}

func (s *mockStatter) Histogram(name string, value float64, tags []string, rate float64) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s: %v", name, value)
	return nil
}

func (s *mockStatter) Unique(bucket string, value string) error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("Bucket %s: %v", bucket, value)
	return nil
}

func (s *mockStatter) Close() error {
	if s.noop {
		return nil
	}
	log.WithFields(log.WithFn(s.fields)).Debugf("closed at %s", time.Now())
	return nil
}
