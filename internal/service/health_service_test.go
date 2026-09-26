package service

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"testing"
	"time"
)

type fakePinger struct {
	err        error
	blockOnCtx bool
	gotCtx     context.Context
}

func (f *fakePinger) Ping(ctx context.Context) error {
	f.gotCtx = ctx
	if f.blockOnCtx {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.err
}

func TestHealthServiceCheckHealth(t *testing.T) {
	s := NewHealthService(&fakePinger{err: errors.New("db down")})

	if err := s.CheckHealth(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestHealthServiceCheckReadiness(t *testing.T) {
	errDB := errors.New("connection refused")

	tests := []struct {
		name         string
		pinger       *fakePinger
		ctxTimeout   time.Duration
		wantErr      bool
		wantIsErrors []error
	}{
		{
			name:    "database available",
			pinger:  &fakePinger{},
			wantErr: false,
		},
		{
			name:         "ping failed",
			pinger:       &fakePinger{err: errDB},
			wantErr:      true,
			wantIsErrors: []error{model.ErrServiceUnavailable, errDB},
		},
		{
			name:         "ping timed out",
			pinger:       &fakePinger{blockOnCtx: true},
			ctxTimeout:   10 * time.Millisecond,
			wantErr:      true,
			wantIsErrors: []error{model.ErrServiceUnavailable, context.DeadlineExceeded},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHealthService(tt.pinger)

			ctx := context.Background()
			if tt.ctxTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.ctxTimeout)
				defer cancel()
			}

			err := s.CheckReadiness(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr = %v, got %v", tt.wantErr, err)
			}
			for _, target := range tt.wantIsErrors {
				if !errors.Is(err, target) {
					t.Errorf("expected errors.Is(err, %v) to be true, err = %v", target, err)
				}
			}
		})
	}
}

func TestHealthServiceCheckReadinessSetsDeadline(t *testing.T) {
	pinger := &fakePinger{}
	s := NewHealthService(pinger)

	if err := s.CheckReadiness(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := pinger.gotCtx.Deadline(); !ok {
		t.Fatal("expected ping context to have a deadline")
	}
}
