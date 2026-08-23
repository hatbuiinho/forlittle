package ipc

import (
	"sync"

	"forlittle/windows-agent/internal/timecontrol"
)

// Hub provides the Service-to-agent contract. The Windows named-pipe adapter is
// attached later; keeping this interface small makes the policy engine testable.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan timecontrol.StateMessage]struct{}
}

func NewHub() *Hub { return &Hub{subscribers: make(map[chan timecontrol.StateMessage]struct{})} }

func (h *Hub) Subscribe() (<-chan timecontrol.StateMessage, func()) {
	channel := make(chan timecontrol.StateMessage, 4)
	h.mu.Lock()
	h.subscribers[channel] = struct{}{}
	h.mu.Unlock()
	return channel, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subscribers[channel]; ok {
			delete(h.subscribers, channel)
			close(channel)
		}
	}
}

func (h *Hub) Publish(message timecontrol.StateMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- message:
		default:
		}
	}
}
