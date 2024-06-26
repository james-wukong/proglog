package server_test

import (
	"fmt"
	"proglog/internal/server"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	emptyLog *server.Log = &server.Log{}
	emptyRecord server.Record = server.Record{}
	ok bool
	ErrOffsetNotFound = fmt.Errorf("offset not found")
)

func TestNewLog(t *testing.T){
	testCases := []struct {
		name string
		wanted *server.Log
	}{
		{
			name: "create new Log",
			wanted: &server.Log{},
		},
		// {
		// 	name: "second",
		// 	wanted: fmt.Errorf("the value of %s caused an error: %v\n", "12345467890", nil),
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := server.NewLog()
			ok := assert.Equal(t, tc.wanted, got, "The two words should be the same.")
			if !ok {
				// if !strings.Contains(tc.wanted.Error(), got.Error()) {
				// 	t.Errorf("\nGot    :%v\n wanted:%v", got, tc.wanted)
				// }
				t.Errorf("\nGot    :%v\n wanted:%v", got, tc.wanted)
			}
		})
	}
}

func TestLogRead(t *testing.T) {
	testCases := [] struct {
		name string
		log *server.Log
		expected server.Record
		err error
		offset uint64
	} {
		{
			name: "empty log",
			log: emptyLog,
			expected: emptyRecord,
			offset: 0,
			err: ErrOffsetNotFound,
		},
	}
	for _, tc := range testCases {
		got, err := tc.log.Read(tc.offset)
		ok = assert.Equal(t, tc.expected, got)
		if !ok {
			t.Errorf("\nGot    :%v\n wanted:%v\n error: %v", got, tc.expected, err)
		}
		ok = assert.Equal(t, tc.err, err)
		if !ok {
			t.Errorf("\nGot    :%v\n wanted:%v\n error: %v", got, tc.expected, err)
		}
	}
}