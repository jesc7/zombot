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
	Q    chan any
	q    []any
	stop bool
	mu   *sync.Mutex
	cond *sync.Cond
	lim  *rate.Limiter
}

func NewQ(ctx context.Context, limit rate.Limit) *Queue {
	q := &Queue{
		lim: rate.NewLimiter(limit, int(limit)),
		Q:   make(chan any, 1),
	}
	q.cond = sync.NewCond(q.mu)

	go func() {
		defer func() {
			if recover() == nil {
				q.mu.Lock()
				q.stop = true
				close(q.Q)
				q.mu.Unlock()
			}
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			q.mu.Lock()
			for len(q.q) == 0 && ctx.Err() == nil {
				q.cond.Wait()
			}

			if ctx.Err() != nil {
				q.mu.Unlock()
				return
			}

			item := q.q[0]
			q.q = q.q[1:]
			q.mu.Unlock()

			if err := q.lim.Wait(ctx); err != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case q.Q <- item:
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
		q.q = append([]any{o}, q.q...)
	case PRIORITY_HIGH:
		if half := len(q.q) / 2; half == 0 {
			q.q = append(q.q, o)
		} else {
			q.q = append(q.q[0:half], append([]any{o}, q.q[half:]...)...)
		}
	default:
		q.q = append(q.q, o)
	}

	q.mu.Unlock()
	q.cond.Signal()
}

type WaitObj struct {
	O    any
	OnOk func(args ...any)
	wg   *sync.WaitGroup
}

func (o *WaitObj) Done() {
	o.wg.Done()
}

func (q *Queue) Wait(wo *WaitObj, priority Priority) {
	wo.wg = &sync.WaitGroup{}
	wo.wg.Add(1)
	q.Add(wo, priority)
	wo.wg.Wait()
}
