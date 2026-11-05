// A supplied bounded inbox adapter: IPC and HTTP share the same instance.
package state

import (
	"sync"

	"example.edu/6cc545/cw1/internal/ipc"
)

type Inbox struct {
	mu       sync.Mutex
	messages []ipc.Message
}

func NewInbox() *Inbox {
	return &Inbox{messages: []ipc.Message{}}
}

// Record preserves the callback signature used by the IPC servers.
func (i *Inbox) Record(m ipc.Message) {
	i.RecordIfNew(m)
}

// RecordIfNew prevents duplicate announcements from being recorded.
func (i *Inbox) RecordIfNew(m ipc.Message) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, existing := range i.messages {
		if existing.NodeID == m.NodeID &&
			existing.RequestID == m.RequestID {
			return false
		}
	}

	i.messages = append(i.messages, m)

	if len(i.messages) > 20 {
		i.messages = append(
			[]ipc.Message{},
			i.messages[len(i.messages)-20:]...,
		)
	}

	return true
}

func (i *Inbox) Snapshot() []ipc.Message {
	i.mu.Lock()
	defer i.mu.Unlock()

	return append([]ipc.Message{}, i.messages...)
}
