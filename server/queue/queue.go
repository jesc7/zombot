package queue

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type Priority int

const (
	PRIORITY_NORMAL Priority = iota
	PRIORITY_HIGH
	PRIORITY_CRITICAL
)

// Queue очередь с ограничителем частоты выборки
type Queue struct {
	C     chan any
	qCrit []any
	qHigh []any
	qNorm []any
	stop  bool
	mu    sync.Mutex
	cond  *sync.Cond
	lim   *rate.Limiter
}

func NewQ(ctx context.Context, limit rate.Limit) *Queue {
	q := &Queue{
		lim: rate.NewLimiter(limit, int(limit)),
		C:   make(chan any, 1),
	}
	q.cond = sync.NewCond(&q.mu)

	go func() {
		defer func() {
			q.mu.Lock()
			q.stop = true
			close(q.C)
			q.mu.Unlock()
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			q.mu.Lock()
			for (len(q.qCrit)+len(q.qHigh)+len(q.qNorm)) == 0 && ctx.Err() == nil {
				q.cond.Wait()
			}

			if ctx.Err() != nil {
				q.mu.Unlock()
				return
			}

			var item any
			if len(q.qCrit) > 0 {
				item = q.qCrit[0]
				q.qCrit = q.qCrit[1:]
			} else if len(q.qHigh) > 0 {
				item = q.qHigh[0]
				q.qHigh = q.qHigh[1:]
			} else if len(q.qNorm) > 0 {
				item = q.qNorm[0]
				q.qNorm = q.qNorm[1:]
			}
			q.mu.Unlock()

			if err := q.lim.Wait(ctx); err != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case q.C <- item:
			}
		}
	}()

	return q
}

func (q *Queue) Add(o any, priority Priority) {
	q.mu.Lock()
	if q.stop {
		q.mu.Unlock()
		return
	}

	switch priority {
	case PRIORITY_CRITICAL:
		q.qCrit = append(q.qCrit, o)
	case PRIORITY_HIGH:
		q.qHigh = append(q.qHigh, o)
	default:
		q.qNorm = append(q.qNorm, o)
	}

	q.mu.Unlock()
	q.cond.Signal()
}

func (q *Queue) Wait(ctx context.Context, wo *WaitObj, priority Priority) error {
	wo.wg = &sync.WaitGroup{}
	wo.wg.Add(1)
	q.Add(wo, priority)

	done := make(chan struct{})
	go func() {
		wo.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type WaitObj struct {
	O    any
	OnOk func(args ...any)
	wg   *sync.WaitGroup
}

func (wo *WaitObj) Done() {
	if wo.wg != nil {
		wo.wg.Done()
	}
}
