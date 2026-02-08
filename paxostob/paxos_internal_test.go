package paxostob

// created to find out which part of paxos is broken.
// see consensus_test.go for unit tests.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob/testutil"
)

func TestAcceptorPromiseState(t *testing.T) {
	p1 := NewInmemTransport("peer1")
	p2 := NewInmemTransport("peer2")
	p2.AddPeer(p1)

	paxos := NewPaxos(p2, 2, 2)

	// check initial state
	require.Equal(t, uint(0), paxos.promisedID)

	// send prepare with ID=5
	// incoming prepare (p1 -> p2)
	prepMsg := &prepareMsg{
		paxosMsg: paxosMsg{
			src:        "peer1",
			paxosValue: paxosValue{msgID: 5, value: testutil.NewTestMsg("peer1", "paxos value")},
		},
	}

	err := paxos.promise(prepMsg)
	require.NoError(t, err)

	// check promisedID was updated
	require.Equal(t, uint(5), paxos.promisedID)
}

// // test proposer promise counting
// func TestProposerPromiseCounter(t *testing.T) {
// 	p1 := NewInmemTransport("peer1")
// 	// create instance of paxos directly
// 	paxos := NewPaxos(p1, 1, 1)

// 	require.Equal(t, uint(0), paxos.promiseCounter)

// 	// simulate receiving promises
// 	promMsg := &promiseMsg{
// 		paxosMsg: paxosMsg{
// 			src:        "imaginaryPeer1",
// 			paxosValue: paxosValue{msgID: paxos.curID, value: testutil.NewTestMsg("peer1", "paxos value")},
// 		},
// 		acceptedValue: &acceptedValue{},
// 	}

// 	paxos.handlePromise(promMsg)
// 	require.Equal(t, uint(1), paxos.promiseCounter)
// }
