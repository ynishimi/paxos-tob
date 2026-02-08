package paxostob_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
	"github.com/ynishimi/paxos-tob/paxostob/testutil"
)

func TestConsensusSolo(t *testing.T) {
	p1 := paxostob.NewInmemTransport("peer1")

	const NumPeers = 1

	p1cons := paxostob.NewCons(p1, 1, NumPeers)

	msg := testutil.NewTestMsg(p1cons.GetAddress(), "it's p1's proposal")
	fmt.Println(msg)

	// send prepare msg
	p1cons.Propose(msg)

	select {
	case decidedMsg := <-p1cons.Decide():
		require.Equal(t, decidedMsg.Payload(), msg.Payload())

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery")
	}
}

func TestConsensusSimple(t *testing.T) {
	p1 := paxostob.NewInmemTransport("peer1")
	p2 := paxostob.NewInmemTransport("peer2")
	p1.AddPeer(p2)
	p2.AddPeer(p1)

	const NumPeers = 2

	p1cons := paxostob.NewCons(p1, 1, NumPeers)
	p2cons := paxostob.NewCons(p2, 2, NumPeers)

	msg := testutil.NewTestMsg(p1cons.GetAddress(), "it's p1's proposal")
	fmt.Println(msg)

	// send prepare msg
	p1cons.Propose(msg)

	var d1 paxostob.Message
	var d2 paxostob.Message

	select {
	case d1 = <-p1cons.Decide():
		fmt.Println(d1)

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery (d1)")
	}

	select {
	case d2 = <-p2cons.Decide():
		fmt.Println(d2)

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery (d2)")
	}

	require.Equal(t, d1, d2)
}

func TestConsensusThreeNodes(t *testing.T) {
	const NumPeers = 3

	p1 := paxostob.NewInmemTransport("peer1")
	p2 := paxostob.NewInmemTransport("peer2")
	p3 := paxostob.NewInmemTransport("peer3")

	trans := [NumPeers]*paxostob.InmemoryTransport{p1, p2, p3}
	for _, i := range trans {
		for _, j := range trans {
			i.AddPeer(j)
		}
	}

	p1cons := paxostob.NewCons(p1, 1, NumPeers)
	p2cons := paxostob.NewCons(p2, 2, NumPeers)
	p3cons := paxostob.NewCons(p3, 3, NumPeers)

	msg := testutil.NewTestMsg(p1cons.GetAddress(), "it's p1's proposal")

	// send prepare msg
	p1cons.Propose(msg)

	d1 := <-p1cons.Decide()
	fmt.Println(d1)
	d2 := <-p2cons.Decide()
	fmt.Println(d2)
	d3 := <-p3cons.Decide()
	fmt.Println(d3)

	require.Equal(t, d1, d2)
	require.Equal(t, d2, d3)
}

// todo: add a situation where some nodes crashed (using Close())
