package queue

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Priority int

const (
	PRIORITY_NORMAL Priority = iota
	PRIORITY_HIGH
	PRIORITY_CRITICAL
)

// Queue приоритетная очередь с ограничителем частоты выборки
type Queue struct {
	C    chan any
	q    [3][]any
	stop bool
	mu   sync.Mutex
	//cond *sync.Cond
	lim *rate.Limiter
}

func NewQ(ctx context.Context, limit rate.Limit) *Queue {
	q := &Queue{
		lim: rate.NewLimiter(limit, int(limit)),
		C:   make(chan any, 1),
	}

	go func() {
		defer func() {
			q.mu.Lock()
			q.stop = true
			close(q.C)
			q.mu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return

			default:
			out:
				q.mu.Lock()
				var pri Priority
				switch {
				case len(q.q[PRIORITY_CRITICAL]) > 0:
					pri = PRIORITY_CRITICAL
				case len(q.q[PRIORITY_HIGH]) > 0:
					pri = PRIORITY_HIGH
				default:
					pri = PRIORITY_NORMAL
				}
				item := q.q[pri][0]
				q.q[pri] = q.q[pri][1:]
				q.mu.Unlock()

				switch len(q.q) != 0 {
				case true:
					for i := 1; len(q.q) != 0; i++ {
						func() {
							q.lim.Wait(ctx)
							q.mu.Lock()
							defer q.mu.Unlock()

							select {
							case q.Q <- q.q[0]:
							}
							q.q = q.q[1:]
						}()
						if i%10 == 0 || ctx.Err() != nil {
							break out
						}
					}

				default:
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		for {
			if ctx.Err() != nil {
				return
			}

			q.mu.Lock()
			for (len(q.q[PRIORITY_NORMAL])+len(q.q[PRIORITY_HIGH])+len(q.q[PRIORITY_CRITICAL])) == 0 && ctx.Err() == nil {
				q.cond.Wait()
			}

			if ctx.Err() != nil {
				q.mu.Unlock()
				return
			}

			var pri Priority
			switch {
			case len(q.q[PRIORITY_CRITICAL]) > 0:
				pri = PRIORITY_CRITICAL
			case len(q.q[PRIORITY_HIGH]) > 0:
				pri = PRIORITY_HIGH
			default:
				pri = PRIORITY_NORMAL
			}
			item := q.q[pri][0]
			q.q[pri] = q.q[pri][1:]
			q.mu.Unlock()

			if err := q.lim.Wait(ctx); err != nil {
				return
			}

			q.C <- item
			/*select {
			case <-ctx.Done():
				return
			case q.C <- item:
			}*/
		}
	}()

	return q
}

func NewQ__(ctx context.Context, limit rate.Limit) *Queue {
	q := &Queue{
		lim: rate.NewLimiter(limit, int(limit)),
		C:   make(chan any, 1),
	}
	/*
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
				for (len(q.q[PRIORITY_NORMAL])+len(q.q[PRIORITY_HIGH])+len(q.q[PRIORITY_CRITICAL])) == 0 && ctx.Err() == nil {
					q.cond.Wait()
				}

				if ctx.Err() != nil {
					q.mu.Unlock()
					return
				}

				var pri Priority
				switch {
				case len(q.q[PRIORITY_CRITICAL]) > 0:
					pri = PRIORITY_CRITICAL
				case len(q.q[PRIORITY_HIGH]) > 0:
					pri = PRIORITY_HIGH
				default:
					pri = PRIORITY_NORMAL
				}
				item := q.q[pri][0]
				q.q[pri] = q.q[pri][1:]
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
	*/
	return q
}

func (q *Queue) Add(o any, priority Priority) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stop {
		return
	}
	q.q[priority] = append(q.q[priority], o)
}

func (q *Queue) Add__(o any, priority Priority) {
	/*
		q.mu.Lock()
		if q.stop {
			q.mu.Unlock()
			return
		}

		q.q[priority] = append(q.q[priority], o)
		q.mu.Unlock()
		q.cond.Signal()
	*/
}

func (q *Queue) Wait(ctx context.Context, wo *WaitObj, priority Priority) error {
	wo.wg = &sync.WaitGroup{}
	wo.wg.Add(1)
	q.Add(wo, priority)
	wo.wg.Wait()
	return nil
}

func (q *Queue) Wait__(ctx context.Context, wo *WaitObj, priority Priority) error {
	/*
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
			wo.wg.Done()
			return ctx.Err()
		}
	*/
	return nil
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
