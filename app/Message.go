package main

import (
	"fmt"

	"github.com/google/uuid"
)

type Message struct {
	UserId    uuid.UUID
	SessionId uuid.UUID
	Timestamp int64
	Message   string
}

func (m *Message) StreamName() string {
	return m.UserId.String() + "-" + m.SessionId.String()
}

func (m *Message) StreamID() string {
	return fmt.Sprintf("%d-0", m.Timestamp)
}
