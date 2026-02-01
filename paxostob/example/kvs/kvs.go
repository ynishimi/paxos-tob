package kvs

import (
	"sync"

	"github.com/ynishimi/paxos-tob/paxostob"
	"github.com/ynishimi/paxos-tob/paxostob/testutil"
)

// type value interface {
// 	serialize() paxostob.Message
// }

// type kvs interface {
// 	Put(v value)
// 	Get() value
// }

type kvs interface {
	Put(v string)
	Get() string
}

type simpleKvs struct {
	tob     paxostob.TotalOrderBroadcast
	storage map[string]string
	mu      sync.RWMutex
}

func NewSimpleKvs(tob paxostob.TotalOrderBroadcast) *simpleKvs {
	kvs := &simpleKvs{
		tob:     tob,
		storage: make(map[string]string),
	}

	// recv delivered msgs

	go func() {
		for deliveredMsg := range tob.Deliver() {
			kvs.store(deliveredMsg)
		}
	}()

	return kvs

}

// func (kvs *simpleKvs) Put(v value) {
// 	kvs.tob.Broadcast(v.serialize())
// }

func (kvs *simpleKvs) Put(v string) {
	// todo: fix this
	msg := testutil.NewTestMsg("me", v)
	kvs.tob.Broadcast(msg)
}

func (kvs *simpleKvs) Get(key string) string {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	return kvs.storage[key]
}

func (kvs *simpleKvs) store(msg paxostob.DeliveredMsg) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	kvs.storage["test"] = msg.String()

}
