package log

type Config struct {
	Segment struct {
		// store the maximum number of bytes that can be held in the store segment
		MaxStoreBytes uint64
		// stores the maximum number of bytes that the index segment can hold
		MaxIndexBytes uint64
		// stores the initial offset value, indicate a starting point within a file or data stream
		InitialOffset uint64
	}
}
