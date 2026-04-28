package bus

import (
	"errors"
	"sync"

	"github.com/jesc7/zombot/cmd/zspy/shared"
)

type Bus struct {
	mu    sync.Mutex
	chans map[string]chan shared.Envelope
}

func NewBus() *Bus {
	return &Bus{
		chans: make(map[string]chan shared.Envelope),
	}
}

func (b *Bus) Register(name string, ch chan shared.Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.chans[name]; ok {
		return errors.New("name already exist")
	}
	b.chans[name] = ch
	return nil
}

func (b *Bus) Write(name string, value shared.Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.chans[name]
	if !ok {
		return errors.New("name not found")
	}
	go func() { ch <- value }()
	return nil
}
