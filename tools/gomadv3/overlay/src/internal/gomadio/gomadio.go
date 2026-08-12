// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"sync"
	_ "unsafe"
)

const (
	configBytes = 212
)

var configMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'I', 'O', 1}

var enabled bool

var entropy = entropyReader{
	key: sha256.Sum256([]byte("gomadv3-deterministic/v1\x00entropy/v1")),
}

func init() {
	if !runtimeProfileEnabled() {
		return
	}
	configuration := runtimeConfigFrame()
	if configuration == nil || !bytes.Equal(configuration[:8], configMagic[:]) || binary.BigEndian.Uint16(configuration[8:10]) != 1 || binary.BigEndian.Uint16(configuration[10:12]) != 1 {
		panic("gomadv3: invalid I/O configuration")
	}
	checksum := sha256.Sum256(configuration[:configBytes-sha256.Size])
	if !bytes.Equal(checksum[:], configuration[configBytes-sha256.Size:]) {
		panic("gomadv3: invalid I/O configuration checksum")
	}
	initTranscript()
	enabled = true
}

//go:linkname runtimeProfileEnabled runtime.gomadIOProfileEnabled
func runtimeProfileEnabled() bool

//go:linkname runtimeConfigFrame runtime.gomadIOConfigFrame
func runtimeConfigFrame() *[configBytes]byte

//go:linkname Enabled
func Enabled() bool {
	return enabled
}

func RandomReader() io.Reader {
	return &entropy
}

type entropyReader struct {
	mu       sync.Mutex
	key      [sha256.Size]byte
	counter  uint64
	offset   int
	block    [sha256.Size]byte
	position uint64
}

func (reader *entropyReader) Read(destination []byte) (int, error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	length := len(destination)
	output := destination
	start := reader.position
	for len(destination) != 0 {
		if reader.offset == 0 {
			var input [sha256.Size + 8]byte
			copy(input[:sha256.Size], reader.key[:])
			binary.BigEndian.PutUint64(input[sha256.Size:], reader.counter)
			reader.block = sha256.Sum256(input[:])
			reader.counter++
		}
		copied := copy(destination, reader.block[reader.offset:])
		reader.offset = (reader.offset + copied) % len(reader.block)
		destination = destination[copied:]
		reader.position += uint64(copied)
	}
	argument := uint64Argument(uint64(length))
	record("entropy.read", argument[:], output, uint64(length), 0, start, reader.position)
	return length, nil
}
