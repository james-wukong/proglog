// the file store records in
package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	// In big-endian byte order, the most significant byte (the "big end")
	// is stored at the lowest memory address.
	// For example, if we have a 4-byte sequence representing a 32-bit integer,
	// the first byte is the most significant byte,
	// and the last byte is the least significant byte.
	enc = binary.BigEndian
	err error
)

const (
	lenWidth = 8
)

type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64 // in bytes
}

func newStore(f *os.File) (*store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	size := uint64(fi.Size())
	return &store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil
}

func (s *store) Append(p []byte) (n, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos = s.size
	// Writes the length of the data (len(p)) as a uint64 value into the buffe
	if err = binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}
	// Writing the Actual Data
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	// Updating the Write Count
	w += lenWidth
	s.size += uint64(w)
	return uint64(w), pos, nil
}

func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// The buffer (s.buf) is flushed to ensure all buffered data
	// is written to the file before reading
	if err = s.buf.Flush(); err != nil {
		return nil, err
	}
	size := make([]byte, lenWidth)
	// Reading the Length of the Data
	if _, err = s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}
	b := make([]byte, enc.Uint64(size))
	// Reading the Data
	if _, err = s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err = s.buf.Flush(); err != nil {
		return 0, err
	}
	return s.File.ReadAt(p, off)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
