package bufferpool

import (
	"bytes"
	"sync"
)

// Buffer pools to reduce memory allocations
var (
	// BufferPool is a pool of byte slices for general use
	BufferPool = sync.Pool{
		New: func() interface{} {
			// Create a buffer with a reasonable initial capacity
			return make([]byte, 0, 1024)
		},
	}

	// ReaderPool is a pool of bytes.Reader for converting byte slices to io.ReadCloser
	ReaderPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Reader{}
		},
	}

	// BufferPool4K is a pool of 4K byte slices for larger payloads
	BufferPool4K = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 4096)
		},
	}

	// BufferPool16K is a pool of 16K byte slices for even larger payloads
	BufferPool16K = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 16384)
		},
	}
)

// GetBuffer returns a byte slice from the pool
func GetBuffer(size int) []byte {
	// For small buffers, use the default pool
	if size <= 1024 {
		buf := BufferPool.Get().([]byte)
		return buf[:size]
	}

	// For medium buffers, use the 4K pool
	if size <= 4096 {
		buf := BufferPool4K.Get().([]byte)
		return buf[:size]
	}

	// For large buffers, use the 16K pool
	if size <= 16384 {
		buf := BufferPool16K.Get().([]byte)
		return buf[:size]
	}

	// For very large buffers, allocate directly
	return make([]byte, size)
}

// PutBuffer returns a byte slice to the appropriate pool
func PutBuffer(buf []byte) {
	capacity := cap(buf)

	// Reset the buffer length to 0 before putting it back
	buf = buf[:0]

	// Return to appropriate pool based on capacity
	if capacity <= 1024 {
		BufferPool.Put(buf)
	} else if capacity <= 4096 {
		BufferPool4K.Put(buf)
	} else if capacity <= 16384 {
		BufferPool16K.Put(buf)
	}
	// Very large buffers are not returned to the pool
}

// GetReader returns a bytes.Reader from the pool
func GetReader() *bytes.Reader {
	return ReaderPool.Get().(*bytes.Reader)
}

// PutReader returns a bytes.Reader to the pool
func PutReader(reader *bytes.Reader) {
	// Reset the reader before putting it back
	reader.Reset(nil)
	ReaderPool.Put(reader)
}
