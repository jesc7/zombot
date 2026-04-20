package queuewait

import (
	"context"
	"sync"

	"github.com/jesc7/zombot/queuewait/queue"
	"golang.org/x/time/rate"
)

type QWaitObj struct {
	o    any
	onOk func(args ...any) (res any)
	wg   *sync.WaitGroup
}

func (o *QWaitObj) Done() {
	o.wg.Done()
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
	o.wg = &sync.WaitGroup{}
	o.wg.Add(1)
	q.Add(o, priority)
	o.wg.Wait()
}
