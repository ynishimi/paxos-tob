package paxostob_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ynishimi/paxos-tob/paxostob"
	"github.com/ynishimi/paxos-tob/paxostob/testutil"
)

// mockCons
type mockCons struct {
	ch chan paxostob.Message
}

func newMockCons() *mockCons {
	return &mockCons{
		ch: make(chan paxostob.Message),
	}
}

func (c *mockCons) Propose(msg paxostob.Message) {
	c.ch <- msg
}

func (c *mockCons) Decide() <-chan paxostob.Message {
	return c.ch
}

func (c *mockCons) Close() {
	close(c.ch)
}

func TestTobBroadcasterBroadcastSolo(t *testing.T) {
	const NumPeers = 1
	p1 := paxostob.NewInmemTransport("peer1")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		c paxostob.Consensus
		// Named input parameters for target function.
		msg       paxostob.Message
		transport paxostob.Transport
	}{
		// TODO: Add test cases.
		struct {
			name      string
			c         paxostob.Consensus
			msg       paxostob.Message
			transport paxostob.Transport
		}{
			name:      "mockCons",
			c:         newMockCons(),
			msg:       testutil.NewTestMsg("peer1", "p1's message"),
			transport: p1,
		},
		{
			name:      "cons",
			c:         paxostob.NewCons(p1, 1, NumPeers),
			msg:       testutil.NewTestMsg("peer1", "p1's message"),
			transport: p1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			b := paxostob.NewTobBroadcaster(tt.transport, 1, NumPeers, tt.c)
			b.Broadcast(tt.msg)

			// should deliver the value
			select {
			case deliveredMsg := <-b.Deliver():
				require.Equal(t, deliveredMsg.Src(), p1.GetAddress())
				require.Equal(t, deliveredMsg.Payload(), tt.msg.Payload())

			case <-time.After(time.Second):
				t.Fatal("timeout waiting for message delivery")
			}
		})
	}
}
