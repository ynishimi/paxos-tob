package paxostob

import "fmt"

// type Consensus interface {
// 	Propose(msg Message)
// 	Decide(msg Message)
// }

type Paxos interface {
	// proposer
	Prepare(msg Message)
	Propose(msg Message)

	// acceptor
	Promise(msg Message)
	Accept(msg Message)

	Close()
}

// Messaging
type paxosMsg struct {
	src     string
	payload string
}

func (m *paxosMsg) Src() string {
	return m.src
}
func (m *paxosMsg) Payload() string {
	return m.payload
}

func (m *paxosMsg) String() string {
	return fmt.Sprintf("%s: %s", m.src, m.payload)
}

type prepareMsg struct {
	paxosMsg
}
type proposeMsg struct {
	paxosMsg
}
type promiseMsg struct {
	paxosMsg
}
type pcceptMsg struct {
	paxosMsg
}

type PaxosCons struct {
	transport  Transport
	addr       string
	promisedID uint
	accepted   struct {
		ID    uint
		value string
	}
	close chan struct{}
}

func NewPaxosCons(transport Transport) *PaxosCons {
	newCons := &PaxosCons{
		transport: transport,
		addr:      transport.GetAddress(),
	}
	newCons.handleIncomingMsg()

	return newCons
}

func (cons *PaxosCons) GetAddress() string {
	return cons.addr
}

// broadcasts prepare messages and collect the promises
func (cons *PaxosCons) Prepare(msg Message) error {

	// todo: create prepare msg w/ ID etc
	prepareMsg := &prepareMsg{
		paxosMsg{
			src:     cons.addr,
			payload: msg.Payload(),
		},
	}

	err := cons.transport.Broadcast(prepareMsg)
	return err
}

func (cons *PaxosCons) Promise(msg Message) {
	// judge if msg can be promised
	// based on ID

	// return promise w/ newest ID + accepted msg

}

func (cons *PaxosCons) Propose(msg Message) {

}

func (cons *PaxosCons) Accept(msg Message) {

}

func (cons *PaxosCons) Close() {
	cons.close <- struct{}{}
}

// handle msg (perhaps using chan)
func (cons *PaxosCons) handleIncomingMsg() {
	go func() {
	recvLoop:
		for {
			select {
			case incomingMsg := <-cons.transport.Deliver():
				// todo: parse msg and call funcs (promise / accept)
				switch v := incomingMsg.(type) {
				case *prepareMsg:
					// promise
					cons.Promise(v)
				}

			case <-cons.close:
				// stop
				break recvLoop
			}
		}
	}()
}
