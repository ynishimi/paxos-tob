package paxostob_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
)

func TestConsensusSimple(t *testing.T) {
	p1 := paxostob.NewInmemTransport("peer1")
	p2 := paxostob.NewInmemTransport("peer2")
	p1.AddPeer(p2)
	p2.AddPeer(p1)

	const NumPeers = 2

	p1cons := paxostob.NewCons(p1, 1, NumPeers)
	p2cons := paxostob.NewCons(p2, 2, NumPeers)

	msg := &TestMsg{
		src:     p1cons.GetAddress(),
		payload: "it's p1's proposal",
	}

	fmt.Println(msg)

	// send prepare msg
	p1cons.Propose(msg)

	d1 := <-p1cons.Decide()
	fmt.Println(d1)
	d2 := <-p2cons.Decide()
	fmt.Println(d2)

	require.Equal(t, d1, d2)

	// // p2 should deliver at transport layer
	// select {
	// case incomingMsg := <-p2.Deliver():
	// 	// success
	// 	fmt.Println(incomingMsg)
	// 	require.Equal(t, msg.Src(), incomingMsg.Src())
	// 	// require.Equal(t, msg.String(), incomingMsg.String())

	// case <-time.After(time.Second):
	// 	t.Fatal("timeout waiting for message delivery")
	// }
}
