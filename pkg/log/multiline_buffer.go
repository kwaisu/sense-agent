package log

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

type MessageBuffer struct {
	maxBytes     int // bytes stored in content
	maxLines     int
	lastBytes    int
	numLines     int
	MessagesChan chan []Message
	lock         sync.Mutex
	message      []Message
	closed       bool
}

func NewMessageBuffer(maxBytes, maxLines int, timeout time.Duration) *MessageBuffer {
	return &MessageBuffer{
		maxBytes:     maxBytes,
		maxLines:     maxLines,
		MessagesChan: make(chan []Message),
	}
}

func (m *MessageBuffer) Add(message Message) {
	if m.closed {
		return
	}
	if !utf8.ValidString(message.Content) {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	if message.Content == "" {
		return
	}
	if m.numLines >= m.maxLines || m.lastBytes >= m.maxBytes {
		m.flushMessage()
	}
	m.message = append(m.message, message)
	if j, err := json.Marshal(m.message); err != nil {
		fmt.Println("message buffer add message error:", err)
	} else {
		m.lastBytes = len(j)
	}
	m.numLines++
}

func (m *MessageBuffer) flushMessage() {
	if m.numLines == 0 {
		return
	}
	m.MessagesChan <- m.message
	m.reset()
}

func (m *MessageBuffer) reset() {
	m.numLines = 0
	m.message = []Message{}
}
