package paxostob

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type Consensus interface {
	Propose(msg Message)
	Decide() <-chan Message
	Close()
}

// // Paxos consensus
// type Paxos interface {
// 	Prepare(msg Message) error
// 	promise(msg *prepareMsg) error
// 	propose(msg *promiseMsg) error
// 	accept(msg *proposeMsg) error
// }

type Paxos struct {
	// info
	transport Transport
	addr      string
	uID       uint

	// global info
	numPeers       uint
	PaxosThreshold uint

	// proposer
	curID uint
	// todo: accept other types
	proposingValue string
	promiseCounter uint

	// acceptor
	promisedID    uint
	acceptedValue acceptedValue

	// other properties
	close chan struct{}
	mu    sync.RWMutex
}

func (cons *Paxos) GetPromisedID() uint {
	cons.mu.RLock()
	defer cons.mu.RUnlock()
	return cons.promisedID
}

type acceptedValue struct {
	acceptedID uint
	value      string
}

// Messaging
type paxosMsg struct {
	src     string
	payload string
	msgID   uint
}

func (m *paxosMsg) Src() string {
	return m.src
}
func (m *paxosMsg) Payload() string {
	return m.payload
}

func (m *paxosMsg) String() string {
	return fmt.Sprintf("%s[%v]: %s", m.src, m.msgID, m.payload)
}

type prepareMsg struct {
	paxosMsg
}
type promiseMsg struct {
	paxosMsg
	acceptedValue acceptedValue
}

type proposeMsg struct {
	paxosMsg
}
type acceptMsg struct {
	paxosMsg
}

func NewPaxosCons(transport Transport, uID uint, numPeers uint) *Paxos {
	newCons := &Paxos{
		transport:      transport,
		addr:           transport.GetAddress(),
		uID:            uID,
		numPeers:       numPeers,
		PaxosThreshold: uint(numPeers/2 + 1),
		curID:          uID,
		promiseCounter: 0,
		promisedID:     0,
		acceptedValue: acceptedValue{
			acceptedID: 0,
		},
	}
	newCons.handleIncomingMsg()

	return newCons
}

func (cons *Paxos) GetAddress() string {
	return cons.addr
}

// broadcasts prepare messages and collect the promises
func (cons *Paxos) Prepare(msg Message) error {
	cons.mu.Lock()
	defer cons.mu.Unlock()

	cons.proposingValue = msg.Payload()

	// create prepare msg w/ ID etc
	prepareMsg := &prepareMsg{
		paxosMsg: paxosMsg{
			src:     cons.addr,
			payload: cons.proposingValue,
			msgID:   cons.curID,
		},
	}

	err := cons.transport.Broadcast(prepareMsg)
	return err
}

func (cons *Paxos) promise(prepareMsg *prepareMsg) error {
	// judge if msg can be promised
	// based on ID
	cons.mu.Lock()
	defer cons.mu.Unlock()

	src := prepareMsg.paxosMsg.src

	if prepareMsg.msgID > cons.promisedID {
		// newer prepare: make a promise
		cons.promisedID = prepareMsg.msgID

		promiseMsg := &promiseMsg{
			paxosMsg:      prepareMsg.paxosMsg,
			acceptedValue: cons.acceptedValue,
		}
		// change src of msg
		promiseMsg.paxosMsg.src = cons.addr

		// return promise w/ newest ID + accepted msg
		return cons.transport.Send(src, prepareMsg)

	} else {
		// older prepare: ignore
		log.Info().Msgf("msg ignored: %s", prepareMsg.paxosMsg.String())
		return nil
	}
}

func (cons *Paxos) handlePromise(msg *promiseMsg) {
	cons.mu.Lock()
	defer cons.mu.Unlock()

	// checks if msg is about current promise
	if msg.msgID == cons.curID {
		cons.promiseCounter++

		// update acceptedValue
		if msg.acceptedValue.acceptedID > cons.acceptedValue.acceptedID {
			// todo: maybe cons should have payload they are trying to propose
			cons.proposingValue = msg.acceptedValue.value
		}

		if cons.promiseCounter > cons.PaxosThreshold {
			cons.mu.Unlock()
			log.Info().Msgf("Approved: %s", msg.String())
			go func() {
				if err := cons.propose(); err != nil {
					log.Error().Err(err).Msg("propose failed")
				}
			}()
		}
	} else {
		log.Info().Msgf("not curID: %v", msg.msgID)
	}
}

func (cons *Paxos) handleAccept(msg *acceptMsg) {
	// todo: everyone counts number of acceptMsgs
}

func (cons *Paxos) propose() error {
	proposeMsg := &proposeMsg{
		paxosMsg: paxosMsg{
			src:     cons.addr,
			payload: cons.proposingValue,
			msgID:   cons.curID,
		},
	}

	return cons.transport.Broadcast(proposeMsg)
}

func (cons *Paxos) accept(msg *proposeMsg) error {
	cons.mu.Lock()
	defer cons.mu.Unlock()

	if msg.msgID != cons.promisedID {
		return nil
	}

	// todo: avoid injection(attempt to override a value when this instance already has a value)?

	// send accept
	acceptMsg := &acceptMsg{
		paxosMsg: paxosMsg{
			src:     cons.addr,
			payload: cons.proposingValue,
			msgID:   msg.msgID,
		},
	}
	return cons.transport.Broadcast(acceptMsg)
}

func (cons *Paxos) Close() {
	cons.close <- struct{}{}
}

// handle msg (perhaps using chan)
func (cons *Paxos) handleIncomingMsg() {
	go func() {
	recvLoop:
		for {
			select {
			case incomingMsg := <-cons.transport.Deliver():
				// todo: parse msg and call funcs (promise / accept)
				switch v := incomingMsg.(type) {
				case *prepareMsg:
					// promise
					go cons.promise(v)

				case *promiseMsg:
					// proposer counts msg
					go cons.handlePromise(v)

				case *proposeMsg:
					// acceptor responds to proposal
					go cons.accept(v)

				case *acceptMsg:
					// todo: everyone counts msg
					go cons.handleAccept(v)
				}

			case <-cons.close:
				// stop
				break recvLoop
			}
		}
	}()
}
