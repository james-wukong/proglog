package server

import (
	"fmt"
	"sync"
)

type Log struct {
	mu      sync.Mutex
	records []Record
}

// the data stored in log, represents individual log entries with a value and an offset
type Record struct {
	// a byte slice that holds the data of the log entry
	Value []byte `json:"value"`
	// a uint64 that holds the position of the log entry within the log
	Offset uint64 `json:"offset"`
}

func NewLog() *Log {
	return &Log{}
}

func (c *Log) Append(record Record) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	record.Offset = uint64(len(c.records))
	c.records = append(c.records, record)

	return record.Offset, nil
}

func (c *Log) Read(offset uint64) (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if (offset >= uint64(len(c.records))) || (c == &Log{}) {
		return Record{}, ErrOffsetNotFound
	}

	return c.records[offset], nil
}

var ErrOffsetNotFound = fmt.Errorf("offset not found")
