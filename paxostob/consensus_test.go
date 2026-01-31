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

	msg := &TestMsg{
		src:     p1cons.GetAddress(),
		payload: "it's p1's proposal",
	}

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
