package timecontrol

import "sync"

// CommandBroker is intentionally ephemeral. It improves command latency, while the
// persisted command table remains the recovery path after any disconnect or restart.
type CommandBroker struct {
	mu      sync.RWMutex
	clients map[string]map[chan []byte]struct{}
}

func NewCommandBroker() *CommandBroker {
	return &CommandBroker{clients: make(map[string]map[chan []byte]struct{})}
}

func (b *CommandBroker) Subscribe(machineID string) (<-chan []byte, func()) {
	channel := make(chan []byte, 8)
	b.mu.Lock()
	if b.clients[machineID] == nil {
		b.clients[machineID] = make(map[chan []byte]struct{})
	}
	b.clients[machineID][channel] = struct{}{}
	b.mu.Unlock()

	return channel, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		clients := b.clients[machineID]
		if clients == nil {
			return
		}
		if _, ok := clients[channel]; !ok {
			return
		}
		delete(clients, channel)
		close(channel)
		if len(clients) == 0 {
			delete(b.clients, machineID)
		}
	}
}

func (b *CommandBroker) Notify(machineID string, payload []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for client := range b.clients[machineID] {
		select {
		case client <- payload:
		default:
			// A slow connection does not block policy delivery. It will recover
			// command state through GET /commands after reconnecting.
		}
	}
}
