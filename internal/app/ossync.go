package app

import (
	"context"
	"errors"
	"time"

	"ctun/internal/sysnet"
)

const osSettingTimeout = 20 * time.Second

type osSync[T comparable] struct {
	desired func() T
	apply   func(context.Context, T) error
	onError func(error)

	kick chan struct{}
	stop context.CancelFunc
	done chan struct{}
}

func newOSSync[T comparable](desired func() T, apply func(context.Context, T) error, onError func(error)) *osSync[T] {
	return &osSync[T]{desired: desired, apply: apply, onError: onError, kick: make(chan struct{}, 1)}
}

func (s *osSync[T]) start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.stop, s.done = cancel, make(chan struct{})
	s.trigger()
	go s.loop(ctx)
}

func (s *osSync[T]) trigger() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *osSync[T]) loop(ctx context.Context) {
	defer close(s.done)
	var applied T
	known := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.kick:
		}
		want := s.desired()
		if known && want == applied {
			continue
		}
		if err := s.run(want); err != nil {
			if !errors.Is(err, sysnet.ErrUnsupported) {
				s.onError(err)
			}
			known = false
			continue
		}
		applied, known = want, true
	}
}

func (s *osSync[T]) run(want T) error {
	ctx, cancel := context.WithTimeout(context.Background(), osSettingTimeout)
	defer cancel()
	return s.apply(ctx, want)
}

func (s *osSync[T]) shutdown() error {
	if s.stop != nil {
		s.stop()
		<-s.done
	}
	var zero T
	return s.run(zero)
}
