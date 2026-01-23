package paxostob

type Message interface {
	GetValue() (string, error)
	String() string
}
