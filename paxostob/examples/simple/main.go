package main

import (
	"fmt"

	"github.com/ynishimi/paxos-tob/paxostob"
)

func main() {
	// Create transports
	// Each peer needs its own transport.
	// Currently, In-memory transport is available. Alternatively, you can write your own transport layer that satisfies Transport interface specified in `transport.go`.
	t1 := paxostob.NewInmemTransport("peer1")
	t2 := paxostob.NewInmemTransport("peer2")

	// Connect peers so they can communicate
	t1.AddPeer(t2)
	t2.AddPeer(t1)

	// Create Total Order Broadcast instances

	const NumPeers = 2

	sender := paxostob.NewTobBroadcaster(
		t1.GetAddress(),
		func() paxostob.Consensus {
			return paxostob.NewCons(t1, 1, NumPeers)
		})

	receiver := paxostob.NewTobBroadcaster(
		t2.GetAddress(),
		func() paxostob.Consensus {
			return paxostob.NewCons(t2, 2, NumPeers)
		})

	// Broadcast a message
	msg := paxostob.NewSimpleMsg(sender.GetAddress(), "hello, paxos!")
	sender.Broadcast(msg)

	// Receive delivered message
	delivered := <-receiver.Deliver()
	fmt.Println("Delivered: ", delivered)
}
