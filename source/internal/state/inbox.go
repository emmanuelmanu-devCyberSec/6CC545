// A supplied bounded inbox adapter: IPC and HTTP share the same instance.
package state

import (
	"example.edu/6cc545/cw1/internal/ipc"
	"sync"
)

type Inbox struct {
	mu       sync.Mutex
	messages []ipc.Message
}

func NewInbox() *Inbox { return &Inbox{messages: []ipc.Message{}} }
func (i *Inbox) Record(m ipc.Message) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.messages = append(i.messages, m)
	if len(i.messages) > 20 {
		i.messages = append([]ipc.Message{}, i.messages[len(i.messages)-20:]...)
	}
}
func (i *Inbox) Snapshot() []ipc.Message {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]ipc.Message{}, i.messages...)
}
