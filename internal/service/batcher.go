package service

import (
	"context"
	"time"
)

type itemResult[Out any] struct {
	value Out
	err   error
}

type batchRequest[In, Out any] struct {
	in   In
	resp chan itemResult[Out]
}

type batcher[In, Out any] struct {
	in      chan batchRequest[In, Out]
	maxSize int
	maxWait time.Duration
	flush   func(ctx context.Context, batch []In) ([]itemResult[Out], error)
}

func newBatcher[In, Out any](
	bufferSize, maxSize int,
	maxWait time.Duration,
	flush func(ctx context.Context, batch []In) ([]itemResult[Out], error),
) *batcher[In, Out] {
	return &batcher[In, Out]{
		in:      make(chan batchRequest[In, Out], bufferSize),
		maxSize: maxSize,
		maxWait: maxWait,
		flush:   flush,
	}
}

// Submit ставит элемент в пачку и ждёт результат её записи или отмену ctx.
func (b *batcher[In, Out]) Submit(ctx context.Context, item In) (Out, error) {
	var zero Out
	req := batchRequest[In, Out]{in: item, resp: make(chan itemResult[Out], 1)}

	select {
	case b.in <- req:
	case <-ctx.Done():
		return zero, ctx.Err()
	}

	select {
	case r := <-req.resp:
		return r.value, r.err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// Run собирает элементы в пачки и отправляет их в flush по размеру или таймеру.
// По отмене ctx дописывает буфер и завершает работу.
func (b *batcher[In, Out]) Run(ctx context.Context) {
	batch := make([]batchRequest[In, Out], 0, b.maxSize)
	timer := time.NewTimer(b.maxWait)
	stopTimer(timer)
	timerActive := false

	flushBatch := func(flushCtx context.Context) {
		if len(batch) == 0 {
			return
		}
		items := make([]In, len(batch))
		for i, req := range batch {
			items[i] = req.in
		}
		results, err := b.flush(flushCtx, items)
		for i, req := range batch {
			if err != nil {
				req.resp <- itemResult[Out]{err: err}
				continue
			}
			req.resp <- results[i]
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			b.drain(&batch)
			flushBatch(context.WithoutCancel(ctx))
			return

		case req := <-b.in:
			if len(batch) == 0 {
				timer.Reset(b.maxWait)
				timerActive = true
			}
			batch = append(batch, req)
			if len(batch) >= b.maxSize {
				stopTimer(timer)
				timerActive = false
				flushBatch(ctx)
			}

		case <-timer.C:
			timerActive = false
			flushBatch(ctx)
		}

		if len(batch) == 0 && timerActive {
			stopTimer(timer)
			timerActive = false
		}
	}
}

func (b *batcher[In, Out]) drain(batch *[]batchRequest[In, Out]) {
	for {
		select {
		case req := <-b.in:
			*batch = append(*batch, req)
		default:
			return
		}
	}
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
