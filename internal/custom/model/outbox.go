package model

import "time"

type OutboxRecord struct {
	GateID    string
	ChatKey   string
	MessageID string
	Payload   []byte
	NextAt    time.Time
}

type OutboxTask struct {
	ID        int64
	ChatKey   string
	MessageID string
	Payload   []byte
	Attempt   int32
	Gate      *CustomGate
}
