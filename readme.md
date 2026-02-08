# `paxos-tob`

[![Go Reference](https://pkg.go.dev/badge/github.com/ynishimi/paxos-tob.svg)](https://pkg.go.dev/github.com/ynishimi/paxos-tob)

Paxos algorithm is one of the implementations of a uniform consensus.
Paxos does not require a Best-Effort Broadcast(beb), which means that the algorithm works in the environment where the packets can drop or delayed.

While Paxos is similar to Raft, which is also used for replicating a state machine in a distributed environment, Paxos does not require a leader, making everyone in the distributed system propose its value.

The algorithm of the library is based on a paper "Paxos Made Simple"(<https://lamport.azurewebsites.net/pubs/paxos-simple.pdf>) by Leslie Lamport.

## Structure

### Main components

- Total Order Broadcast: Provides total order delivery using multiple instances of consensus.
- Consensus: Offers Propose and Decide functions using a paxos instance. This satisfies the interface and properties of uniform consensus (Validity, Integrity, Uniform agreement, Termination and Integration) specified on page 211 in the book "Introduction to Reliable and Secure Distributed Programming" and the content of the lecture in Distributed algorithms (CS-451) at EPFL.
- Paxos: Implements the foundation for (Uniform) consensus algorithm. This is known to work in the network condition where the messages can take arbitrarily long to be delivered, can be duplicated, and can be lost. The architecture of the implementation is inspired from the course materials presented by Decentralized systems engineering (CS-438) at EPFL.

### Others / Current limitations

- Transport: Currently, only in-memory implementation is available.
- No persistent storage.

## Usage

The codebase below is an excerpt from [examples/simple/main.go](paxostob/examples/simple/main.go).

```go

package main

import (
 "fmt"

 "github.com/ynishimi/paxos-tob/paxostob"
)

func main() {
 // Create transports. Each peer needs its own transport.
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


```

## Example

See [examples/kvs](paxostob/examples/kvs) for a distributed key-value store implementation.
