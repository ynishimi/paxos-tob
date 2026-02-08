package paxostob

import (
	"fmt"
	"sync"
	"time"

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
	chosenValue chan Message
	mu          sync.RWMutex
}

func (cons *Cons) GetAddress() string {
	return cons.paxos.addr
}

func NewCons(transport Transport, uID uint, numPeers uint) *Cons {
	cons := &Cons{
		decided:     false,
		chosenValue: make(chan Message),
	}

	cons.paxos = NewPaxos(transport, uID, numPeers)
	// wait for choose msgs
	go func() {
		msg := <-cons.paxos.chosen
		cons.handleChoose(msg)
	}()

	return cons
}

func (cons *Cons) Propose(msg Message) {
	go func() {
		if err := cons.paxos.Prepare(msg); err != nil {
			log.Error().Err(err).Msg("failed to prepare msg in paxos")
		}
	}()
}

func (cons *Cons) handleChoose(v paxosValue) {

	cons.mu.Lock()
	if cons.decided {
		cons.mu.Unlock()
		return
	}
	cons.decided = true
	cons.mu.Unlock()

	cons.chosenValue <- v.value
}

// todo: should return a value (not only string)
func (cons *Cons) Decide() <-chan Message {
	// todo: return decided value upon sending chan (by handleChoose())
	return cons.chosenValue
}

func (cons *Cons) Close() {
	cons.paxos.Close()
}

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
	curID           uint
	proposingValue  Message
	promiseCounter  uint
	prepareTimeout  time.Duration
	prepareApproved chan struct{}

	proposeTimeout  time.Duration
	proposeApproved chan paxosValue

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
	value Message
}

type acceptedValue struct {
	acceptedID uint
	value      Message
}

// Messaging
type paxosMsg struct {
	src string
	paxosValue
}

func (m *paxosMsg) Src() string {
	return m.src
}
func (m *paxosMsg) Payload() []byte {
	return m.paxosValue.value.Payload()
}

func (m *paxosMsg) String() string {
	return fmt.Sprintf("[src:%s msgID:%v] %s", m.src, m.msgID, m.value)
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

		curID: uID,
		// proposingValue: "",
		promiseCounter: 0,

		prepareTimeout:  time.Second,
		prepareApproved: make(chan struct{}),
		proposeTimeout:  time.Second,
		proposeApproved: make(chan paxosValue, 1),

		promisedID:    0,
		acceptedValue: &acceptedValue{},

		acceptCount: make(map[paxosValue]acceptInfo),
		chosen:      make(chan paxosValue, 1),
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

	paxos.proposingValue = msg

	paxos.mu.Unlock()

	return paxos.broadcastPrepare()

}

func (paxos *Paxos) broadcastPrepare() error {

	paxos.mu.RLock()
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
	paxos.mu.RUnlock()

	log.Debug().Msgf("prepareMsg sent: %s", prepareMsg)
	err := paxos.transport.Broadcast(prepareMsg)

	if err != nil {
		return err
	}

	// set a timer; if time is over, retry w/ new id
	select {
	case <-paxos.prepareApproved:
		log.Debug().Msg("Prepare approved")
		go paxos.onPrepareApproved()

	// timeout: retry
	case <-time.After(paxos.prepareTimeout):

		paxos.mu.Lock()
		// update the ID
		paxos.curID = paxos.curID + paxos.uID
		paxos.mu.Unlock()

		// send prepare with new msg
		paxos.broadcastPrepare()
	}
	return nil
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

			// notify approval to Prepare()
			paxos.prepareApproved <- struct{}{}
		}
	} else {
		log.Info().Msgf("not curID: %v", msg.msgID)
	}
	paxos.mu.Unlock()
}
func (paxos *Paxos) onPrepareApproved() {
	if err := paxos.propose(); err != nil {
		log.Error().Err(err).Msg("propose failed")
	}
}

func (paxos *Paxos) handleAccept(msg *acceptMsg) {
	paxos.mu.Lock()
	defer paxos.mu.Unlock()

	log.Debug().Msg("handleAccept")

	// everyone counts number of acceptMsgs
	acceptInfo := paxos.acceptCount[msg.paxosValue]
	acceptInfo.count++

	if acceptInfo.count >= paxos.PaxosThreshold && !acceptInfo.chosen {
		log.Debug().Msg("threshold satisfied")
		acceptInfo.chosen = true

		// notify proposer's propose() (disable timeout for re-sending)
		// if not a proposer, goes to default
		select {
		case paxos.proposeApproved <- msg.paxosValue:
		default:
		}

		// send chosen value to consensus layer
		paxos.chosen <- msg.paxosValue
	}
	paxos.acceptCount[msg.paxosValue] = acceptInfo
}

func (paxos *Paxos) propose() error {

	paxos.mu.RLock()
	proposeMsg := &proposeMsg{
		paxosMsg: paxosMsg{
			src: paxos.addr,
			paxosValue: paxosValue{
				msgID: paxos.curID,
				value: paxos.proposingValue,
			},
		},
	}
	paxos.mu.RUnlock()

	log.Debug().Msgf("proposeMsg broadcast: %s", proposeMsg)

	err := paxos.transport.Broadcast(proposeMsg)

	if err != nil {
		return err
	}

	// wait for propose to be accepted
	select {
	case <-paxos.proposeApproved:
		log.Debug().Msg("propose approved")

	case <-time.After(paxos.proposeTimeout):
		paxos.mu.Lock()
		// update the ID
		paxos.curID = paxos.curID + paxos.uID

		paxos.mu.Unlock()

		// send prepare with new msg
		paxos.broadcastPrepare()
	}
	return nil
}

func (paxos *Paxos) accept(msg *proposeMsg) error {
	log.Debug().Msg("accept")
	paxos.mu.Lock()
	defer paxos.mu.Unlock()
	log.Debug().Msg("lock")
	if msg.msgID < paxos.promisedID {
		log.Info().Msgf("msgID smaller than promisedID: %v < %v", msg.msgID, paxos.promisedID)
		return nil
	}

	// todo: avoid injection(attempt to override a value when this instance already has a value)?

	// update acceptedID
	paxos.acceptedValue = &acceptedValue{
		acceptedID: msg.msgID,
		value:      msg.value,
	}

	// send accept
	acceptMsg := &acceptMsg{
		paxosMsg: paxosMsg{
			src: paxos.addr,
			paxosValue: paxosValue{
				msgID: msg.msgID,
				value: msg.value,
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
				switch v := incomingMsg.(type) {
				case *prepareMsg:
					// acceptor promise
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
