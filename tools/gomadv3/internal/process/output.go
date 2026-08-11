package process

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"math"
)

type OutputCapture struct {
	limit     uint64
	headLimit int
	tailLimit int
	head      []byte
	tail      []byte
	total     uint64
	hasher    hash.Hash
}

type Output struct {
	Bytes          []byte
	FullSHA256     [sha256.Size]byte
	RetainedSHA256 [sha256.Size]byte
	TotalBytes     uint64
	RetainedBytes  uint64
	DiscardedBytes uint64
	Truncated      bool
}

func NewOutputCapture(limit uint64) (*OutputCapture, error) {
	if limit == 0 {
		return nil, fmt.Errorf("output limit must be positive")
	}
	maxInt := uint64(^uint(0) >> 1)
	if limit > maxInt {
		return nil, fmt.Errorf("output limit %d exceeds host addressable memory", limit)
	}
	tailLimit := int(limit / 4)
	headLimit := int(limit) - tailLimit
	return &OutputCapture{
		limit:     limit,
		headLimit: headLimit,
		tailLimit: tailLimit,
		head:      make([]byte, 0, headLimit),
		tail:      make([]byte, 0, tailLimit),
		hasher:    sha256.New(),
	}, nil
}

func (capture *OutputCapture) Write(data []byte) (int, error) {
	if math.MaxUint64-capture.total < uint64(len(data)) {
		return 0, fmt.Errorf("output byte count overflow")
	}
	written, err := capture.hasher.Write(data)
	if err != nil {
		return written, fmt.Errorf("hash output: %w", err)
	}
	if written != len(data) {
		return written, fmt.Errorf("hash output wrote %d bytes, want %d", written, len(data))
	}
	capture.total += uint64(len(data))

	remainingHead := capture.headLimit - len(capture.head)
	if remainingHead > 0 {
		toHead := min(remainingHead, len(data))
		capture.head = append(capture.head, data[:toHead]...)
		data = data[toHead:]
	}
	capture.appendTail(data)
	return written, nil
}

func (capture *OutputCapture) Result() Output {
	retainedBytes := uint64(len(capture.head) + len(capture.tail))
	discardedBytes := capture.total - retainedBytes
	truncated := capture.total > capture.limit
	retained := make([]byte, 0, len(capture.head)+len(capture.tail)+64)
	retained = append(retained, capture.head...)
	if truncated {
		retained = fmt.Appendf(retained, "\n--- gomadv3 output truncated: %d bytes discarded ---\n", discardedBytes)
	}
	retained = append(retained, capture.tail...)
	var fullHash [sha256.Size]byte
	copy(fullHash[:], capture.hasher.Sum(nil))
	return Output{
		Bytes:          retained,
		FullSHA256:     fullHash,
		RetainedSHA256: sha256.Sum256(retained),
		TotalBytes:     capture.total,
		RetainedBytes:  retainedBytes,
		DiscardedBytes: discardedBytes,
		Truncated:      truncated,
	}
}

func (capture *OutputCapture) appendTail(data []byte) {
	if capture.tailLimit == 0 || len(data) == 0 {
		return
	}
	if len(data) >= capture.tailLimit {
		capture.tail = capture.tail[:capture.tailLimit]
		copy(capture.tail, data[len(data)-capture.tailLimit:])
		return
	}
	if excess := len(capture.tail) + len(data) - capture.tailLimit; excess > 0 {
		copy(capture.tail, capture.tail[excess:])
		capture.tail = capture.tail[:len(capture.tail)-excess]
	}
	capture.tail = append(capture.tail, data...)
}
