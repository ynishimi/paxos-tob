package paxostob

import "fmt"

type Message interface {
	Src() string
	Payload() []byte
	String() string
}

type SimepleMsg struct {
	src     string
	payload string
}

func NewSimpleMsg(src string, payload string) *SimepleMsg {
	return &SimepleMsg{
		src:     src,
		payload: payload,
	}
}

func (m *SimepleMsg) Src() string {
	return m.src
}

func (m *SimepleMsg) Payload() []byte {
	return []byte(m.payload)
}

func (m *SimepleMsg) String() string {
	return fmt.Sprintf("%s: %s", m.src, m.payload)
}
