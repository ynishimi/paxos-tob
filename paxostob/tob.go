package paxostob

type tob interface {
	Broadcast(message Message) error
	Deliver() <-chan DeliveredMessage
}

type DeliveredMessage struct {
	src     string
	message Message
}

// type tobBroadcaster struct{}

// func (b *tobBroadcaster) Broadcast(message Message) error {

// }

// func (b *tobBroadcaster) Deliver() <-chan DeliveredMessage {

// }
