// the file store index entries in

// Assume we have a log file where entries need to be indexed:
// Log Entry: "Hello, World!" stored at byte position 1024.
// Offset: Logical position 1 (first entry in the log).
// Position: Byte position 1024 in the log file.

// To store this in the index:
// Offset (offWidth): 4 bytes to store the value 1.
// Position (posWidth): 8 bytes to store the value 1024.

// Each index entry in the file is 12 bytes (entWidth):
// Bytes 0-3: Offset (1). 1 Byte -> 8 bits
// Bytes 4-11: Position (1024).

// When retrieving the entry:
// Read the first 12 bytes from the index.
// Extract the offset (first 4 bytes) and position (next 8 bytes).
// Use the position to seek to byte 1024 in the log file and read the entry.
package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

var (
	// -----------------------
	// |offwidth | poswidth  |
	// |---------+------------
	// | 4(bytes)| 8(bytes)  |  ==> [an entry]
	// |---------+------------
	// |      entwidth       |
	// -----------------------
	// offset, takes up to 4 bytes (uint32), relative
	// the width (size in bytes) allocated for storing the offset of a log entry
	// To store a relative offset,which is an identifier that represents the logical position of an entry in the log
	// Used to quickly locate an entry in a sequence
	offWidth uint64 = 4
	// position, takes up to 8 bytes (uint64)
	// the width (size in bytes) allocated for storing the position of a log entry
	// To store the actual byte position of the entry in the log file.
	// Used to directly access the byte offset in the log file
	posWidth uint64 = 8
	// position of an entry given its offset
	// the total width (size in bytes) of an index entry, which includes both the offset and position
	// To combine the offset and position into a single, fixed-size entry for the index.
	// Each entry in the index array has a fixed size
	entWidth = offWidth + posWidth
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

// creates an index for the given file
// save the current size of the file
// grow the file to the max index size before memory-mapping the file
// then return the created index to the caller
func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{
		file: f,
	}
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(fi.Size())
	// Truncate the size of the given file to (size) bytes
	// Using Truncate() function
	if err = os.Truncate(
		f.Name(),
		int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}
	// memory-mapped file operations are being performed using the gommap package
	// Memory-mapped files allow file contents to be mapped directly
	// into the process's address space, enabling efficient file I/O operations.

	// map the file represented by idx.file into memory.
	if idx.mmap, err = gommap.Map(
		// This retrieves the file descriptor (an integer handle)
		idx.file.Fd(),
		// desired memory protection for the mapping, allows reading and writing
		gommap.PROT_READ|gommap.PROT_WRITE,
		// updates to the mapping are shared with other processes that map this file
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}
	return idx, nil
}

// make sure the memory-mapped file has synced its data to the persisted file
// and that the persisted file has flushed its contents to stable storage.
// truncate the persisted file to the amount of data that's actually in it
// and close the file

// The reason we resize them now is that,
// once they're memory-mapped, we can't resize them
func (i *index) Close() error {
	// Synchronize the Memory-Mapped File
	// all updates to the mapping should be written to the underlying file
	if err = i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	// flushes any buffered data to the underlying storage device
	// ensuring that all in-memory changes are physically written to disk.
	if err = i.file.Sync(); err != nil {
		return err
	}
	// resizes the file to the specified length
	if err = i.file.Truncate(int64(i.size)); err != nil {
		return err
	}

	return i.file.Close()
}

// take in an offset and returns the associated record's position in the store
// offset is relative to the segment's base offset
// 0 is always the offset of the index's first entry, 1 is the second
// Returns the relative offset (out) and the byte position (pos) of the log entry
func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}
	if in == -1 {
		// it sets out to the last index entry (i.e., the last record in the index)
		out = uint32((i.size / entWidth) - 1)
	} else {
		// sets the out offset
		out = uint32(in)
	}
	// Calculate Position in Memory-Mapped File
	pos = uint64(out) * entWidth
	// Checks if this position is beyond the current size of the index
	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}
	// Retrieve Offset and Position
	// Reads the relative offset from the memory-mapped file by slicing it from pos to pos+offWidth
	// takes a byte slice ([]byte) as input and interprets it as a 32-bit unsigned integer (uint32)
	out = enc.Uint32(i.mmap[pos : pos+offWidth])
	// Reads the byte position from the memory-mapped file by slicing it from pos+offWidth to pos+entWidth
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])

	return out, pos, nil
}

// func (binary.BigEndian) Uint32(b []byte) uint32 {
// 	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
// 	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
// }

// appends the given offset and position to the index
func (i *index) Write(off uint32, pos uint64) error {
	// validate space to write the entry
	if uint64(len(i.mmap)) < i.size+entWidth {
		return io.EOF
	}
	// encode the offset and position
	// write them to the memory-mapped file
	// takes a byte slice ([]byte) and a 32-bit unsigned integer (uint32) as inputs.
	// It writes the 32-bit integer into the byte slice in big-endian byte order.
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)
	// increment the position for the next write
	i.size += uint64(entWidth)

	return nil
}

//	func (binary.BigEndian) PutUint32(b []byte, v uint32) {
//		_ = b[3] // early bounds check to guarantee safety of writes below
//		b[0] = byte(v)
//		b[1] = byte(v >> 8)
//		b[2] = byte(v >> 16)
//		b[3] = byte(v >> 24)
//	}

// return the index's file path
func (i *index) Name() string {
	return i.file.Name()
}
