package queuewait

import (
	"context"
	"sync"

	"github.com/jesc7/zombot/queue"
	"golang.org/x/time/rate"
)

type Obj struct {
	obj any
	evt func(res ...any)
	wg  *sync.WaitGroup
}

type QueueWait struct {
	*queue.Queue
}

func (q QueueWait) Wait(obj Obj, priority queue.Priority) {
	obj.wg = &sync.WaitGroup{}
	obj.wg.Add(1)
	q.Append(obj, priority)
	obj.wg.Wait()
}

func NewQWait(ctx context.Context, limit rate.Limit) QueueWait {
	return QueueWait{
		Queue: queue.NewQ(ctx, limit),
	}
}
