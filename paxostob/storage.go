package paxostob

type Storage interface {
	Get(msg Message) error
	Put(msg Message) error
}

// todo: Inmemory implementation
// type Inmemory struct{}

// func (i *Inmemory) Get(msg Message) error {

// }

// func (i *Inmemory) Put(msg Message) error {

// }
