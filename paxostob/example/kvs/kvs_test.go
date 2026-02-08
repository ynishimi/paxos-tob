package kvs_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
	"github.com/ynishimi/paxos-tob/paxostob/example/kvs"
)

// mockTob
type mockTob struct {
	addr      string
	deliverCh chan paxostob.DeliveredMsg
}

func newMockTob(addr string) *mockTob {
	return &mockTob{
		addr:      addr,
		deliverCh: make(chan paxostob.DeliveredMsg, 100),
	}
}

func (m *mockTob) GetAddress() string {
	return m.addr
}

func (m *mockTob) Broadcast(msg paxostob.Message) {
	go func() {
		m.deliverCh <- paxostob.DeliveredMsg{Src: m.addr, Msg: msg}
	}()
}

func (m *mockTob) Deliver() <-chan paxostob.DeliveredMsg {
	return m.deliverCh
}

func TestKvsSinglePut(t *testing.T) {
	const NumPeers = 1
	p1 := paxostob.NewInmemTransport("peer1")
	mock := newMockTob("mock1")
	key1 := "key1"
	val1 := "val1"

	tests := []struct {
		name string
		tob  paxostob.TotalOrderBroadcast
		addr string
	}{
		{
			name: "mock",
			tob:  mock,
			addr: mock.GetAddress(),
		},
		{
			name: "tob",
			tob:  paxostob.NewTobBroadcaster(p1.GetAddress(), func() paxostob.Consensus { return paxostob.NewCons(p1, 1, NumPeers) }),
			addr: p1.GetAddress(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs := kvs.NewSimpleKvs(tt.tob)

			// Put() returns <-chan error and notifies after TOBDeliver
			done := kvs.Put(key1, val1)

			select {
			case err := <-done:
				require.Nil(t, err)

				// check if the value is available
				v, err := kvs.Get(key1)
				require.NoError(t, err)
				require.Equal(t, v, val1)

			case <-time.After(10 * time.Second):
				t.Fatal("timeout for Put()")
			}
		})
	}
}
