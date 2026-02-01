package kvs_test

import (
	"testing"

	"github.com/ynishimi/paxos-tob/paxostob"
	"github.com/ynishimi/paxos-tob/paxostob/example/kvs"
)

func TestKvsSinglePut(t *testing.T) {
	const NumPeers = 1
	p1 := paxostob.NewInmemTransport("peer1")

	kvs := kvs.NewSimpleKvs(paxostob.NewTobBroadcaster(p1, 1, NumPeers, paxostob.NewCons(p1, 1, NumPeers)))

	kvs.Put("a")

	kvs.Get("me")
}
