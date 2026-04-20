package queue

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Queue очередь с ограничителем частоты выборки
type Queue struct {
	Q    chan any
	q    []any
	stop bool
	mu   sync.Mutex
	lim  *rate.Limiter
}

type Priority int

const (
	PRIORITY_NORMAL Priority = iota
	PRIORITY_HIGH
	PRIORITY_CRITICAL
)

func NewQ(ctx context.Context, limit rate.Limit) *Queue {
	q := &Queue{
		lim: rate.NewLimiter(limit, int(limit)),
		Q:   make(chan any, 1),
	}

	go func() {
		defer func() {
			if recover() == nil {
				q.stop = true
				close(q.Q)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return

			default:
			out:
				switch len(q.q) != 0 {
				case true:
					for i := 1; len(q.q) != 0; i++ {
						func() {
							q.lim.Wait(ctx)
							q.mu.Lock()
							defer q.mu.Unlock()
							q.Q <- q.q[0]
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
	}()
	return q
}

func (q *Queue) Append(obj any, priority Priority) {
	if !q.stop {
		q.mu.Lock()
		defer q.mu.Unlock()

		switch priority {
		case PRIORITY_CRITICAL:
			q.q = append([]any{obj}, q.q...)
		case PRIORITY_HIGH:
			if half := len(q.q) / 2; half == 0 {
				q.q = append(q.q, obj)
			} else {
				q.q = append(q.q[0:half], append([]any{obj}, q.q[half:]...)...)
			}
		default:
			q.q = append(q.q, obj)
		}
	}
}
