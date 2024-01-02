package log

import (
	"fmt"
	"time"
)

type Message struct {
	Content   string
	Level     string
	Timestamp time.Time
	Fields    map[string]string
	Meta      map[string]string
}

func (m *Message) AddFields(name string, val string) {
	if val, ok := m.Fields[name]; ok {
		fmt.Printf(" Message Field duplicate, name :%s,	val: %s", name, val)
	}
	m.Fields[name] = val
}

func (m *Message) AddMeta(name string, val string) {
	if val, ok := m.Fields[name]; ok {
		fmt.Printf(" Message Field duplicate, name :%s,	val: %s", name, val)
	}
	m.Meta[name] = val
}
