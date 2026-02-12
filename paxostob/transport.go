package paxostob

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
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
	t := &InmemoryTransport{
		addr:            addr,
		incomingMsgChan: make(chan Message, 10),
		peers:           make(map[string]*InmemoryTransport),
	}
	t.peers[addr] = t
	return t
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

type LossyTransport struct {
	InmemoryTransport
	lossRate float64
	delay    time.Duration
	// reordering bool
}

func NewLossyTransport(addr string, lossRate float64, delay time.Duration) *LossyTransport {
	return &LossyTransport{
		InmemoryTransport: *NewInmemTransport(addr),
		lossRate:          lossRate,
		delay:             delay,
		// reordering:        reordering,
	}
}

func (t *LossyTransport) AddPeer(newPeer *LossyTransport) {
	t.InmemoryTransport.AddPeer(&newPeer.InmemoryTransport)
}

func (t *LossyTransport) Send(dest string, message Message) error {
	// drops based on lossRate
	if rand.Float64() < t.lossRate {
		log.Debug().Msg("dropped: rand.Float64() < t.lossRate")
		return nil
	}

	if t.delay > 0 {
		time.Sleep(t.delay)
	}

	return t.InmemoryTransport.Send(dest, message)
}

func (t *LossyTransport) Broadcast(message Message) error {
	t.InmemoryTransport.mu.RLock()
	defer t.InmemoryTransport.mu.RUnlock()

	// assumption: all nodes are in the map
sendLoop:
	for _, destTransport := range t.InmemoryTransport.peers {
		// drops based on lossRate
		if rand.Float64() < t.lossRate {
			log.Debug().Msg("dropped: rand.Float64() < t.lossRate")
			continue sendLoop
		}

		if t.delay > 0 {
			time.Sleep(t.delay)
		}

		destTransport.incomingMsgChan <- message
	}

	// send to yourself
	// loss should not be applied to itself
	t.InmemoryTransport.incomingMsgChan <- message

	return nil
}

func (t *LossyTransport) Deliver() <-chan Message {
	return t.InmemoryTransport.Deliver()
}

func (t *LossyTransport) GetAddress() string {
	return t.InmemoryTransport.GetAddress()
}
