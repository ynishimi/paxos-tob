package paxostob_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
)

func TestPrepareSend(t *testing.T) {
	p1 := paxostob.NewTransport("peer1")
	p2 := paxostob.NewTransport("peer2")
	p1.AddPeer(p2)

	p1cons := paxostob.NewPaxosCons(p1)
	// p2cons := paxostob.NewPaxosCons(p2)

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
		require.Equal(t, msg.String(), incomingMsg.String())

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery")
	}

	// todo: should find msg at paxos layer
}
