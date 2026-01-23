package paxostob

import "time"

type Transport interface {
	CreateSocket(address string) (ClosableSocket, error)
}

type Socket interface {
	Send(dest string, message Message, timeout time.Duration) error
	Recv(timeout time.Duration) (Message, error)
	GetAddress() string
	GetIns() []Message
	GteOuts() []Message
}

type ClosableSocket interface {
	Socket
	Close() error
}
