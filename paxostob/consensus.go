package paxostob

// Consensus defines the primitives of paxos consensus.
type Consensus interface {
	Prepare(msg Message)
	Promise(msg Message)
	Propose(msg Message)
	Accept(msg Message)
}
