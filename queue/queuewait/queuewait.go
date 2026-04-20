package queuewait

import (
	"context"
	"sync"

	"github.com/jesc7/zombot/queue"
	"golang.org/x/time/rate"
)

type QWaitObj struct {
	o   any
	evt func(res ...any)
	wg  *sync.WaitGroup
}

type QWait struct {
	*queue.Queue
}

func NewQWait(ctx context.Context, limit rate.Limit) QWait {
	return QWait{
		Queue: queue.NewQ(ctx, limit),
	}
}

func (q QWait) Wait(o *QWaitObj, priority queue.Priority) {
	o.wg = &sync.WaitGroup{}
	o.wg.Add(1)
	q.Add(o, priority)
	o.wg.Wait()
}
