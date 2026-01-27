package paxostob_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
)

func TestPreparePromiseSuccess(t *testing.T) {
	p1 := paxostob.NewInmemTransport("peer1")
	p2 := paxostob.NewInmemTransport("peer2")
	p1.AddPeer(p2)

	const NumPeers = 2
	p1cons := paxostob.NewPaxos(p1, 1, NumPeers)
	_ = paxostob.NewPaxos(p2, 2, NumPeers)

	msg := &TestMsg{
		src:     p1cons.GetAddress(),
		payload: "prepare from peer1",
	}

	// send prepare msg
	err := p1cons.Prepare(msg)
	require.NoError(t, err)

	// p2 should deliver at transport layer
	select {
	case incomingMsg := <-p2.Deliver():
		// success
		fmt.Println(incomingMsg)
		require.Equal(t, msg.Src(), incomingMsg.Src())
		// require.Equal(t, msg.String(), incomingMsg.String())

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery")
	}

	// todo: p2 should find msg at paxos layer
	// require.Equal(t, p2cons.GetPromisedID(), -1)

	// todo: propose

	// todo: promise
}
