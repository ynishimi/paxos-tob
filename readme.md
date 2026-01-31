# `paxos-tob`

## What's Paxos / Total Order Broadcast(TOB)?

Paxos algorithm is one of the implementations of a uniform consensus.
Paxos does not require a Best-Effort Broadcast(beb), which means that the algorithm works in the environment where the packets can drop or delayed.

While Paxos is similar to Raft, which is also used for replicating a state machine in a distributed environment, Paxos does not require a leader, making everyone in the distributed system propose its value.

The algorithm of the library is based on a paper ["Paxos Made Simple"](https://lamport.azurewebsites.net/pubs/paxos-simple.pdf) by Leslie Lamport.

## Structure

### Main components

- Total Order Broadcast
- Consensus
- Paxos

### Others

- Transport
- Storage

### Example

- Distributed KVS
