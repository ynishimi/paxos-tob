package paxostob_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ynishimi/paxos-tob/paxostob"
)

type testMsg struct {
	src     string
	payload string
}

func (m *testMsg) String() string {
	return fmt.Sprintf("%s: %s", m.src, m.payload)
}

func TestInmemoryTransportSingle(t *testing.T) {

	p1 := paxostob.NewTransport("peer1")
	p2 := paxostob.NewTransport("peer2")

	// let them know each other
	p1.AddPeer(p2)

	// p1 -> p2
	p1.Send(p2.GetAddress(), &testMsg{p1.GetAddress(), "hi from peer1!"})

	select {
	case msg := <-p2.Deliver():
		// success
		fmt.Print(msg)

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message delivery")
	}
}

func TestInmemoryTransport(t *testing.T) {

	p1 := paxostob.NewTransport("peer1")
	p2 := paxostob.NewTransport("peer2")

	// let them know each other
	p1.AddPeer(p2)

	// let p2 wait for messages
	msgs := make([]paxostob.Message, 0)

	go func() {
		for {
			// print msg upon receiving it
			msg := <-p2.Deliver()
			fmt.Println(msg)
			msgs = append(msgs, msg)
		}
	}()

	// p1 -> p2
	sendMsgs := []paxostob.Message{
		&testMsg{p1.GetAddress(), "hi"},
		&testMsg{p1.GetAddress(), "hi again"},
		&testMsg{p1.GetAddress(), "hi again XD"},
	}

	for _, sendMsg := range sendMsgs {
		p1.Send(p2.GetAddress(), sendMsg)
	}

	time.Sleep(time.Second)
	if !reflect.DeepEqual(sendMsgs, msgs) {
		t.Error("msgs list not equal")
	}
}
