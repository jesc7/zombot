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
	C chan any
	q [3][]any
	//qCrit []any
	//qHigh []any
	//qNorm []any
	stop bool
	mu   sync.Mutex
	cond *sync.Cond
	lim  *rate.Limiter
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
			//for (len(q.qCrit)+len(q.qHigh)+len(q.qNorm)) == 0 && ctx.Err() == nil {
			for (len(q.q[PRIORITY_NORMAL])+len(q.q[PRIORITY_HIGH])+len(q.q[PRIORITY_CRITICAL])) == 0 && ctx.Err() == nil {
				q.cond.Wait()
			}

			if ctx.Err() != nil {
				q.mu.Unlock()
				return
			}

			var item any
			if len(q.q[PRIORITY_CRITICAL]) > 0 {
				item = q.q[PRIORITY_CRITICAL][0]
				q.q[PRIORITY_CRITICAL] = q.q[PRIORITY_CRITICAL][1:]
			} else if len(q.q[PRIORITY_HIGH]) > 0 {
				item = q.q[PRIORITY_HIGH][0]
				q.q[PRIORITY_HIGH] = q.q[PRIORITY_HIGH][1:]
			} else if len(q.q[PRIORITY_NORMAL]) > 0 {
				item = q.q[PRIORITY_NORMAL][0]
				q.q[PRIORITY_NORMAL] = q.q[PRIORITY_NORMAL][1:]
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

	q.q[priority] = append(q.q[priority], o)
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
