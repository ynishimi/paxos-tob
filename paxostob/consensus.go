package paxostob

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type Consensus interface {
	Propose(msg Message)

	// as the process is allowed to choose the value multiple times but the value should be the same for all time, effectively the first choise should be decided. it satisfies integration property of consensus(no process decide twice).
	Decide() <-chan Message
	Close()
}

type Cons struct {
	paxos       *Paxos
	decided     bool
	chosenValue chan string
	mu          sync.Mutex
}

func (cons *Cons) GetAddress() string {
	return cons.paxos.addr
}

func NewCons(transport Transport, uID uint, numPeers uint) *Cons {
	cons := &Cons{
		decided:     false,
		chosenValue: make(chan string),
	}

	cons.paxos = NewPaxos(transport, uID, numPeers)
	// todo: wait for choose msgs?
	go func() {
		msg := <-cons.paxos.chosen
		cons.handleChoose(msg)
	}()

	return cons
}

func (cons *Cons) Propose(msg Message) {
	go cons.paxos.Prepare(msg)
}

func (cons *Cons) handleChoose(v paxosValue) {
	cons.mu.Lock()
	defer cons.mu.Unlock()

	// only decide once
	if !cons.decided {
		cons.decided = true
		cons.chosenValue <- v.value
	}
}

// todo: should return a value (not only string)
func (cons *Cons) Decide() <-chan string {
	// todo: return decided value upon sending chan (by handleChoose())
	return cons.chosenValue
}

func (cons *Cons) Close() {
	cons.paxos.Close()
}

// // Paxos consensus
// type Paxos interface {
// 	Prepare(msg Message) error
// 	promise(msg *prepareMsg) error
// 	propose(msg *promiseMsg) error
// 	accept(msg *proposeMsg) error
// }

type acceptInfo struct {
	count  uint
	chosen bool
}

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
	acceptedValue *acceptedValue

	// learner
	acceptCount map[paxosValue]acceptInfo
	chosen      chan paxosValue

	// other properties
	close chan struct{}
	mu    sync.RWMutex
}

func (paxos *Paxos) GetPromisedID() uint {
	paxos.mu.RLock()
	defer paxos.mu.RUnlock()
	return paxos.promisedID
}

type paxosValue struct {
	msgID uint
	value string
}

type acceptedValue struct {
	acceptedID uint
	value      string
}

// Messaging
type paxosMsg struct {
	src string
	paxosValue
}

func (m *paxosMsg) Src() string {
	return m.src
}
func (m *paxosMsg) Payload() string {
	return m.paxosValue.value
}

func (m *paxosMsg) String() string {
	return fmt.Sprintf("%s[msgID:%v]: %s", m.src, m.msgID, m.value)
}

type prepareMsg struct {
	paxosMsg
}
type promiseMsg struct {
	paxosMsg
	acceptedValue *acceptedValue
}

type proposeMsg struct {
	paxosMsg
}
type acceptMsg struct {
	paxosMsg
}

func NewPaxos(transport Transport, uID uint, numPeers uint) *Paxos {
	newPaxos := &Paxos{
		transport: transport,
		addr:      transport.GetAddress(),
		uID:       uID,

		numPeers:       numPeers,
		PaxosThreshold: uint(numPeers/2 + 1),

		curID:          uID,
		proposingValue: "",
		promiseCounter: 0,

		promisedID:    0,
		acceptedValue: &acceptedValue{},

		acceptCount: make(map[paxosValue]acceptInfo),
		chosen:      make(chan paxosValue),
		close:       make(chan struct{}),
	}
	newPaxos.handleIncomingMsg()

	return newPaxos
}

func (paxos *Paxos) GetAddress() string {
	return paxos.addr
}

// broadcasts prepare messages and collect the promises
func (paxos *Paxos) Prepare(msg Message) error {
	paxos.mu.Lock()
	defer paxos.mu.Unlock()

	paxos.proposingValue = msg.Payload()

	// create prepare msg w/ ID etc
	prepareMsg := &prepareMsg{
		paxosMsg: paxosMsg{
			src: paxos.addr,
			paxosValue: paxosValue{
				msgID: paxos.curID,
				value: paxos.proposingValue,
			},
		},
	}

	log.Debug().Msgf("prepareMsg sent: %s", prepareMsg)
	err := paxos.transport.Broadcast(prepareMsg)
	return err
}

func (paxos *Paxos) promise(prepareMsg *prepareMsg) error {
	// judge if msg can be promised
	// based on ID
	paxos.mu.Lock()
	defer paxos.mu.Unlock()

	dest := prepareMsg.paxosMsg.src

	if prepareMsg.msgID > paxos.promisedID {
		// newer prepare: make a promise
		paxos.promisedID = prepareMsg.msgID

		promiseMsg := &promiseMsg{
			paxosMsg:      prepareMsg.paxosMsg,
			acceptedValue: paxos.acceptedValue,
		}
		// change src of msg
		promiseMsg.paxosMsg.src = paxos.addr

		log.Debug().Msgf("promiseMsg sent: %s -> %s", promiseMsg, dest)

		// return promise w/ newest ID + accepted msg
		return paxos.transport.Send(dest, promiseMsg)

	} else {
		// older prepare: ignore
		log.Info().Msgf("msg ignored: %s", prepareMsg.paxosMsg.String())
		return nil
	}
}

func (paxos *Paxos) handlePromise(msg *promiseMsg) {
	paxos.mu.Lock()

	log.Debug().Msgf("incoming promise: %s", msg)

	// checks if msg is about current promise
	if msg.msgID == paxos.curID {
		paxos.promiseCounter++

		// update acceptedValue
		if msg.acceptedValue != nil {
			if paxos.acceptedValue == nil || (msg.acceptedValue.acceptedID > paxos.acceptedValue.acceptedID) {
				paxos.proposingValue = msg.acceptedValue.value
			}
		}

		if paxos.promiseCounter == paxos.PaxosThreshold {
			log.Info().Msgf("prepare approved: %s", msg)
			paxos.mu.Unlock()
			go func() {
				if err := paxos.propose(); err != nil {
					log.Error().Err(err).Msg("propose failed")
				}
			}()
			return
		}
	} else {
		log.Info().Msgf("not curID: %v", msg.msgID)
	}
	paxos.mu.Unlock()
}

func (paxos *Paxos) handleAccept(msg *acceptMsg) {
	paxos.mu.Lock()
	defer paxos.mu.Unlock()

	// everyone counts number of acceptMsgs
	acceptInfo := paxos.acceptCount[msg.paxosValue]
	acceptInfo.count++

	// todo: choose the value of the ID (let's say the value is n), once [number of acceptMsgs with ID=n] >= [majority]
	if acceptInfo.count == paxos.PaxosThreshold {
		if !acceptInfo.chosen {
			acceptInfo.chosen = true
			// send a chosen value to consensus layer
			paxos.chosen <- msg.paxosValue
		}
	}
	paxos.acceptCount[msg.paxosValue] = acceptInfo
}

func (paxos *Paxos) propose() error {
	paxos.mu.RLock()
	defer paxos.mu.RUnlock()

	proposeMsg := &proposeMsg{
		paxosMsg: paxosMsg{
			src: paxos.addr,
			paxosValue: paxosValue{
				msgID: paxos.curID,
				value: paxos.proposingValue,
			},
		},
	}

	log.Debug().Msgf("proposeMsg broadcast: %s", proposeMsg)

	return paxos.transport.Broadcast(proposeMsg)
}

func (paxos *Paxos) accept(msg *proposeMsg) error {
	paxos.mu.Lock()
	defer paxos.mu.Unlock()

	if msg.msgID < paxos.promisedID {
		log.Info().Msgf("msgID smaller than promisedID: %v < %v", msg.msgID, paxos.promisedID)
		return nil
	}

	// todo: avoid injection(attempt to override a value when this instance already has a value)?

	// update acceptedID
	paxos.acceptedValue = &acceptedValue{
		acceptedID: msg.msgID,
		value:      msg.Payload(),
	}

	// send accept
	acceptMsg := &acceptMsg{
		paxosMsg: paxosMsg{
			src: paxos.addr,
			paxosValue: paxosValue{
				msgID: msg.msgID,
				value: msg.Payload(),
			},
		},
	}

	log.Debug().Msgf("acceptMsg sent: %s", acceptMsg)

	return paxos.transport.Broadcast(acceptMsg)
}

// handle msg (perhaps using chan)
func (paxos *Paxos) handleIncomingMsg() {
	go func() {
	recvLoop:
		for {
			select {
			case incomingMsg := <-paxos.transport.Deliver():
				log.Debug().Msgf("incoming msg[%T]: %s", incomingMsg, incomingMsg)
				// todo: parse msg and call funcs (promise / accept)
				switch v := incomingMsg.(type) {
				case *prepareMsg:
					// promise
					go paxos.promise(v)

				case *promiseMsg:
					// proposer counts msg
					go paxos.handlePromise(v)

				case *proposeMsg:
					// acceptor responds to proposal
					go paxos.accept(v)

				case *acceptMsg:
					// everyone counts msg
					go paxos.handleAccept(v)
				}

			case <-paxos.close:
				// stop
				break recvLoop
			}
		}
	}()
}

func (paxos *Paxos) Close() {
	paxos.close <- struct{}{}
}
