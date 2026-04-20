package queuewait

import (
	"context"
	"sync"

	"github.com/jesc7/zombot/queue"
	"golang.org/x/time/rate"
)

type QWaitObj struct {
	O    any
	OnOk func(args ...any) (res any)
	Wg   *sync.WaitGroup
}

func (o *QWaitObj) Done() {
	o.Wg.Done()
}

type QWait struct {
	*queue.Queue
}

func NewQWait(ctx context.Context, limit rate.Limit) *QWait {
	return &QWait{
		Queue: queue.NewQ(ctx, limit),
	}
}

func (q QWait) Wait(o *QWaitObj, priority queue.Priority) {
	o.Wg = &sync.WaitGroup{}
	o.Wg.Add(1)
	q.Add(o, priority)
	o.Wg.Wait()
}
