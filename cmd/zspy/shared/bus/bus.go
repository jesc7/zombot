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

func (b *Bus) Register(name string) chan any {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.chans[name]; ok {
		return ch
	}
	ch := make(chan any)
	b.chans[name] = ch
	return ch
}

func (b *Bus) Unregister(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.chans[name]; ok {
		close(ch)
		delete(b.chans, name)
	}
}

func (b *Bus) Write(name string, value any) error {
	if b.closed {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.chans[name]
	if !ok {
		return errors.New("named chan not found")
	}
	go func() { ch <- value }()
	return nil
}
