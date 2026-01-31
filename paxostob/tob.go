package paxostob

import (
	"fmt"
	"sync"
)

// total order broadcast uses multiple instances of consensus.
// consFactory defines the general function to create a network.
type consFactory func() Consensus

type TotalOrderBroadcast interface {
	// todo: error handling
	Broadcast(msg Message)
	Deliver() <-chan DeliveredMsg
}

type DeliveredMsg struct {
	src string
	msg Message
}

func (m *DeliveredMsg) Src() string {
	return m.src
}

func (m *DeliveredMsg) Payload() string {
	return m.msg.Payload()
}

func (m *DeliveredMsg) String() string {
	return fmt.Sprintf("%s: %s", m.src, m.msg.String())
}

// todo: can I have infinite number of consensus?
type consID uint
type lastDeliveredID uint

type decision struct {
	consID consID
	msg    Message
}

type TobBroadcaster struct {
	// current instance of consensus
	consFactory consFactory
	consMap     map[consID]Consensus

	// stores broadcasted msgs
	proposedMap map[consID]Message

	curConsID consID
	// for total order property, the peer should only deliver the  ID next to it
	lastDeliveredID lastDeliveredID

	// stores pending msgs to map
	decidedMap map[consID]Message
	// channel for storing to waitingDeliverMap
	decidedChan chan decision

	// channel for sending delivered msgs to upper layer
	deliveredChan chan DeliveredMsg

	mu sync.RWMutex
}

// creates new inistance of TobBroadcast
func NewTobBroadcaster(transport Transport, uID uint, numPeers uint, c Consensus) *TobBroadcaster {
	tob := &TobBroadcaster{
		consFactory: func() Consensus {
			return NewCons(transport, uID, numPeers)
		},
		consMap: make(map[consID]Consensus),

		proposedMap: make(map[consID]Message),

		// starts with 1
		curConsID:       1,
		lastDeliveredID: 0,

		decidedMap:  make(map[consID]Message),
		decidedChan: make(chan decision),

		deliveredChan: make(chan DeliveredMsg),
	}

	// recv decided msgs from consensus layer
	go func() {
		for decision := range tob.decidedChan {
			tob.handleDecide(decision)
		}
	}()

	return tob
}

func (b *TobBroadcaster) Broadcast(msg Message) {
	b.propose(msg)
}

// return Delivered messages.
func (b *TobBroadcaster) Deliver() <-chan DeliveredMsg {
	return b.deliveredChan
}

// proposes msg with the new instance of consensus
func (b *TobBroadcaster) propose(msg Message) {
	b.mu.Lock()

	// on each round of consensus, peers propose its head.
	// todo: remember the proposed values?

	cons := b.consFactory()
	curConsID := b.curConsID
	b.consMap[curConsID] = cons

	// register a proposal
	b.proposedMap[curConsID] = msg

	b.curConsID++

	b.mu.Unlock()

	go func(curConsID consID, cons Consensus) {
		cons.Propose(msg)

		// wait for the deliver on consensus layer

		b.decidedChan <- decision{
			consID: curConsID,
			msg:    <-cons.Decide(),
		}

	}(curConsID, cons)
}

// handle decided msg (reorder and deliver in order, based on consID)
func (b *TobBroadcaster) handleDecide(decision decision) {

	b.mu.Lock()

	// set the decision to decidedMap
	b.decidedMap[decision.consID] = decision.msg

	proposal := b.proposedMap[decision.consID]
	b.mu.Unlock()

	// try to deliver msgs
	b.flush()

	// if the head of the queue was found to be decided, remove the element from the queue.
	// if the decided message was proposed by myself and not nil, propose it again with newer consID.
	if (proposal != decision.msg) && (proposal != nil) {
		// my proposal was rejected; propose again as a new proposal
		b.propose(decision.msg)
	}
}

// upon broadcasting, the message is sent to a fifo queue (or sent immediately, prob using channels?).
func (b *TobBroadcaster) flush() {
	// flush all the decisions
	for b.curConsID != consID(b.lastDeliveredID) {

		// upon getting decided value, checks the consID and the deliver it if it's next to lastDeliveredID. e.g., if lastDeliveredID is 5, a decision is accepted iff decision.consID is 6.

		// todo: repeatedly access consID(b.lastDeliveredID)+1
		b.mu.RLock()
		nextDeliver := b.decidedMap[consID(b.lastDeliveredID)+1]
		b.mu.RUnlock()
		if nextDeliver == nil {
			return
		}

		// deliver msg
		// todo: should modify consensus to attach src
		b.deliveredChan <- DeliveredMsg{
			src: nextDeliver.Src(),
			msg: nextDeliver,
		}
	}
}
