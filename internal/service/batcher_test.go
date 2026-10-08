package service

import (
	"context"
	"errors"
	"fleettrack/internal/model"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func echoFlush(calls *atomic.Int32, sizes *[]int, mu *sync.Mutex) func(context.Context, []model.Telemetry) ([]itemResult[model.Telemetry], error) {
	return func(_ context.Context, batch []model.Telemetry) ([]itemResult[model.Telemetry], error) {
		calls.Add(1)
		mu.Lock()
		*sizes = append(*sizes, len(batch))
		mu.Unlock()
		out := make([]itemResult[model.Telemetry], len(batch))
		for i, t := range batch {
			t.TelemetryID = t.VehicleID * 10
			out[i] = itemResult[model.Telemetry]{value: t}
		}
		return out, nil
	}
}

func TestBatcherFlushesOnSize(t *testing.T) {
	var calls atomic.Int32
	var sizes []int
	var mu sync.Mutex
	b := newBatcher[model.Telemetry, model.Telemetry](10, 3, time.Hour, echoFlush(&calls, &sizes, &mu))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.Run(ctx)

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			res, err := b.Submit(ctx, model.Telemetry{VehicleID: id})
			if err != nil {
				t.Errorf("submit %d: %v", id, err)
				return
			}
			if res.TelemetryID != id*10 {
				t.Errorf("submit %d got id %d, want %d", id, res.TelemetryID, id*10)
			}
		}(i)
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("flush calls = %d, want 1", calls.Load())
	}
	if sizes[0] != 3 {
		t.Fatalf("batch size = %d, want 3", sizes[0])
	}
}

func TestBatcherFlushesOnTimer(t *testing.T) {
	var calls atomic.Int32
	var sizes []int
	var mu sync.Mutex
	b := newBatcher[model.Telemetry, model.Telemetry](10, 100, 20*time.Millisecond, echoFlush(&calls, &sizes, &mu))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.Run(ctx)

	start := time.Now()
	if _, err := b.Submit(ctx, model.Telemetry{VehicleID: 1}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 15*time.Millisecond {
		t.Fatalf("returned too early: %v", elapsed)
	}
	if sizes[0] != 1 {
		t.Fatalf("batch size = %d, want 1", sizes[0])
	}
}

func TestBatcherPropagatesFlushError(t *testing.T) {
	flushErr := errors.New("db down")
	b := newBatcher[model.Telemetry, model.Telemetry](10, 2, time.Hour, func(_ context.Context, _ []model.Telemetry) ([]itemResult[model.Telemetry], error) {
		return nil, flushErr
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.Run(ctx)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = b.Submit(ctx, model.Telemetry{VehicleID: i + 1})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if !errors.Is(err, flushErr) {
			t.Fatalf("submit %d err = %v, want %v", i, err, flushErr)
		}
	}
}

func TestBatcherDrainsBufferOnShutdown(t *testing.T) {
	var calls atomic.Int32
	var sizes []int
	var mu sync.Mutex
	b := newBatcher[model.Telemetry, model.Telemetry](10, 100, time.Hour, echoFlush(&calls, &sizes, &mu))

	ctx, cancel := context.WithCancel(context.Background())

	results := make(chan error, 1)
	go func() {
		_, err := b.Submit(context.Background(), model.Telemetry{VehicleID: 7})
		results <- err
	}()

	time.Sleep(20 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		b.Run(ctx)
		close(done)
	}()

	cancel()
	<-done

	if err := <-results; err != nil {
		t.Fatalf("pending request got err on shutdown: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("flush calls = %d, want 1", calls.Load())
	}
}
