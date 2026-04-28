package bus

import (
	"errors"
	"sync"
)

type Bus struct {
	chans map[string]chan any
	sync.Mutex
}

func NewBus() *Bus {
	return &Bus{
		chans: make(map[string]chan any),
	}
}

func (b *Bus) Register(name string, ch chan any) error {
	b.Lock()
	defer b.Unlock()

	if _, ok := b.chans[name]; ok {
		return errors.New("name already exist")
	}
	b.chans[name] = ch
	return nil
}

func (b *Bus) Write(name string, value any) error {
	b.Lock()
	defer b.Unlock()

	ch, ok := b.chans[name]
	if !ok {
		return errors.New("name not found")
	}
	ch <- value
	return nil
}
