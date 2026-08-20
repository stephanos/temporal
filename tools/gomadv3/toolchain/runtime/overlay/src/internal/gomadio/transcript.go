// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"encoding/binary"
	"io"
	"os"

	"internal/gomadtrace"
)

func initTranscript() {
	gomadtrace.Init()
}

func record(operation string, arguments, content []byte, count uint64, result uint32, entropyStart, entropyEnd uint64) {
	if !enabled {
		return
	}
	gomadtrace.Record(operation, arguments, content, count, result, entropyStart, entropyEnd)
}

func TestingComplete() {
	if enabled {
		gomadtrace.TestingComplete()
	}
}

func uint64Argument(value uint64) [8]byte {
	var result [8]byte
	binary.BigEndian.PutUint64(result[:], value)
	return result
}

func networkArguments(network string, values ...int) []byte {
	result := make([]byte, 2+len(network)+8*len(values))
	binary.BigEndian.PutUint16(result[:2], uint16(len(network)))
	copy(result[2:], network)
	offset := 2 + len(network)
	for _, value := range values {
		binary.BigEndian.PutUint64(result[offset:offset+8], uint64(value))
		offset += 8
	}
	return result
}

func resultClass(err error) uint32 {
	switch err {
	case nil:
		return 0
	case ErrClosed:
		return 1
	case ErrAddressInUse:
		return 2
	case ErrConnectionRefused:
		return 3
	case ErrUnsupported:
		return 4
	case ErrResourceExhausted:
		return 8
	case io.EOF:
		return 5
	case os.ErrDeadlineExceeded:
		return 6
	default:
		return 7
	}
}
