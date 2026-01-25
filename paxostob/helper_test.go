package paxostob_test

import "fmt"

type TestMsg struct {
	src     string
	payload string
}

func (m *TestMsg) Src() string {
	return m.src
}

func (m *TestMsg) Payload() string {
	return m.payload
}

func (m *TestMsg) String() string {
	return fmt.Sprintf("%s: %s", m.src, m.payload)
}
