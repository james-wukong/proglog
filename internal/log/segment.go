// the abstraction that ties a store and an index together
// The segment wraps the index and store types to coordinate operations across the two.
// For example, when the log appends a record to the active segment,
// the segment needs to write the data to its store and add a new entry in the index.
// Similarly for reads, the segment needs to look up the entry
// from the index and then fetch the data from the store.
package log

import (
	"fmt"
	"os"
	"path"

	api "proglog/api/v1"

	"google.golang.org/protobuf/proto"
)

type segment struct {
	// needs to call its store and index files
	store *store
	index *index
	// need to know what offset to append new records under
	// and calculate the relative offset for the index entries
	// The starting offset for the log entries in this segment
	// The offset for the next log entry to be appended to this segment
	baseOffset, nextOffset uint64
	config                 Config
}

// The log calls newSegment when it needs to add a new segment,
// such as when the current active segment hits its max size.
// - open the store and index files
// - create our index and store with these files
// - set the segment’s next offset to prepare for the next appended record
func newSegment(dir string, baseOffset uint64, c Config) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     c,
	}

	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}

	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE,
		0644,
	)

	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}

// writes the record to the segment
// returns the newly appended record's offset
func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	cur := s.nextOffset
	record.Offset = cur
	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}
	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}
	if err = s.index.Write(
		// index offsets are relative to base offset
		uint32(s.nextOffset-uint64(s.baseOffset)),
		pos,
	); err != nil {
		return 0, err
	}
	s.nextOffset++
	return cur, nil
}

// returns the record for the given offset
// to read a record the segment must first translate the absolute index
// into a relative offset
// gett he ssociated index entry
func (s *segment) Read(off uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}
	p, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}
	record := &api.Record{}
	err = proto.Unmarshal(p, record)

	return record, err
}

// returns whether the segment has reached its max
// either by writing too much to the store
// or the index
func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes || s.index.size >= s.config.Segment.MaxStoreBytes
}

// closes the segment
// removes the index and store files
func (s *segment) Remove() error {
	if err = s.index.Close(); err != nil {
		return err
	}
	if err = os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err = os.Remove(s.store.Name()); err != nil {
		return err
	}
	return nil
}

func (s *segment) CLose() error {
	if err = s.index.Close(); err != nil {
		return err
	}

	if err = s.store.Close(); err != nil {
		return err
	}
	return nil
}

// returns the nearest and lesser multiple of k in j
// for example: nearestMultiple(9, 4) == 8
func nearestMultiple(j, k uint64) uint64 {
	if j >= 0 {
		return (j / k) * k
	}

	return ((j - k + 1) / k) * k
}
