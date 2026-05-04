package bus

import (
	"errors"
	"sync"
)

type Bus struct {
	mu     sync.Mutex
	chans  map[string]chan any // shared.Envelope
	closed bool
}

func NewBus() *Bus {
	return &Bus{
		chans: make(map[string]chan any /*shared.Envelope*/),
	}
}

func (b *Bus) Close() {
	b.closed = true
	for _, v := range b.chans {
		func() {
			defer recover()
			close(v)
		}()
	}
}

func (b *Bus) Register(name string) (chan any /*shared.Envelope*/, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.chans[name]; ok {
		return nil, errors.New("name already exist")
	}
	ch := make(chan any /*shared.Envelope*/)
	b.chans[name] = ch
	return ch, nil
}

func (b *Bus) Write(name string, value any /*shared.Envelope*/) error {
	if b.closed {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.chans[name]
	if !ok {
		return errors.New("name not found")
	}
	go func() { ch <- value }()
	return nil
}
