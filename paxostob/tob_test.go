package paxostob_test

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
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
		ch: make(chan paxostob.Message, 1),
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
		name    string
		factory func() paxostob.Consensus
		msg     paxostob.Message
		addr    string
	}{
		{
			name:    "mockCons",
			factory: func() paxostob.Consensus { return newMockCons() },
			msg:     testutil.NewTestMsg("peer1", "p1's message"),
			addr:    p1.GetAddress(),
		},
		{
			name:    "cons",
			factory: func() paxostob.Consensus { return paxostob.NewCons(p1, 1, NumPeers) },
			msg:     testutil.NewTestMsg("peer1", "p1's message"),
			addr:    p1.GetAddress(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			b := paxostob.NewTobBroadcaster(tt.addr, tt.factory)
			b.Broadcast(tt.msg)

			// should deliver the value
			select {
			case deliveredMsg := <-b.Deliver():
				require.Equal(t, deliveredMsg.Src, p1.GetAddress())
				require.Equal(t, deliveredMsg.Payload(), tt.msg.Payload())

			case <-time.After(time.Second):
				t.Fatal("timeout waiting for message delivery")
			}
		})
	}
}

// todo: easy scenario with 2 peers
func TestTobBroadcasterBroadcastTwoPeers(t *testing.T) {
	p1 := paxostob.NewInmemTransport("peer1")
	p2 := paxostob.NewInmemTransport("peer2")
	p1.AddPeer(p2)
	p2.AddPeer(p1)

	const NumPeers = 2

	p1tob := paxostob.NewTobBroadcaster(p1.GetAddress(), func() paxostob.Consensus {
		return paxostob.NewCons(p1, 1, NumPeers)
	})
	p2tob := paxostob.NewTobBroadcaster(p2.GetAddress(), func() paxostob.Consensus {
		return paxostob.NewCons(p2, 2, NumPeers)
	})

	// todo: src addr should be automatically assigned
	msg1 := testutil.NewTestMsg(p1tob.GetAddress(), "p1's message")

	msg2 := testutil.NewTestMsg(p2tob.GetAddress(), "p2's message")

	// send prepare msg
	p1tob.Broadcast(msg1)
	p2tob.Broadcast(msg2)

	// both peers should deliver msgs satisfying total order property

	// 1st msg
	d1 := <-p1tob.Deliver()
	log.Debug().Msgf("tob_test p1 deliv: %v", d1)

	d2 := <-p2tob.Deliver()
	log.Debug().Msgf("tob_test p2 deliv: %v", d2)

	require.Equal(t, d1, d2)

	// 2nd msg
	d1 = <-p1tob.Deliver()
	log.Debug().Msgf("tob_test p1 deliv: %v", d1)

	d2 = <-p2tob.Deliver()
	log.Debug().Msgf("tob_test p2 deliv: %v", d2)

	require.Equal(t, d1, d2)
}

// todo:
// p1, p2.
// p1 broadcasts m1, and p2 broadcasts m2. Both in cons1.
// Only one of them will be decided on cons1.
// The message which wewe not decided on cons1 should be proposed in cons2.
// Eventually, both messages should be delivered in the same order.

// todo: p1 is currently in cons1.
// then p1 receives cons3's result.
// p1 should change its counter to 3 and wait for cons1 and cons2's result.
