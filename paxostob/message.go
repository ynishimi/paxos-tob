package paxostob

type Message interface {
	Src() string
	Payload() string
	String() string
}

// type SimpleMsg struct {
// 	src     string
// 	payload string
// }

// func (m *SimpleMsg) String() string {
// 	return fmt.Sprint(m.src, m.payload)
// }
