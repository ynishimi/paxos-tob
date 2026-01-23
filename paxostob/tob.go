package paxostob

type tob interface {
	Broadcast(message Message) error
	Deliver() <-chan DeliveredMessage
}

type DeliveredMessage struct {
	src     string
	message Message
}

type broadcaster struct{}

func (b *broadcaster) Broadcast(message Message) error {

}

func (b *broadcaster) Deliver() <-chan DeliveredMessage {

}
