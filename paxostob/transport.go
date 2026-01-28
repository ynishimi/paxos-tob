package paxostob

import (
	"fmt"
	"sync"
)

type Transport interface {
	Send(dest string, message Message) error
	Broadcast(message Message) error
	Deliver() <-chan Message
	GetAddress() string
}

type InmemoryTransport struct {
	addr            string
	incomingMsgChan chan Message
	peers           map[string]*InmemoryTransport
	mu              sync.RWMutex
}

// creates an instance of InmemoryTransport
func NewInmemTransport(addr string) *InmemoryTransport {
	return &InmemoryTransport{
		addr:            addr,
		incomingMsgChan: make(chan Message, 10),
		peers:           make(map[string]*InmemoryTransport),
	}
}

func (t *InmemoryTransport) AddPeer(newPeer *InmemoryTransport) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.peers[newPeer.addr] = newPeer
}

func (t *InmemoryTransport) Send(dest string, message Message) error {
	// send message to a dest's incomingMessage channel
	t.mu.RLock()
	destTransport, ok := t.peers[dest]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("peer not found")
	}

	destTransport.incomingMsgChan <- message
	return nil
}

func (t *InmemoryTransport) Broadcast(message Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// assumption: all nodes are in the map
	for _, destTransport := range t.peers {
		destTransport.incomingMsgChan <- message
	}

	// send to yourself
	t.incomingMsgChan <- message

	return nil
}

// returns chan to receive messages
func (t *InmemoryTransport) Deliver() <-chan Message {
	return t.incomingMsgChan
}

func (t *InmemoryTransport) GetAddress() string {
	return t.addr
}
