package paxostob

type TotalOrderBroadcast interface {
	Broadcast(message Message) error
	Deliver() <-chan DeliveredMsg
}

type DeliveredMsg struct {
	src     string
	payload string
}

// type TobBroadcaster struct{}

// func (b *TobBroadcaster) Broadcast(message Message) error {

// }

// func (b *TobBroadcaster) Deliver() <-chan DeliveredMsg {

// }
