// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadtrace

import (
	"encoding/binary"
	"internal/runtime/exithook"
	"sync"
	"syscall"
)

const (
	transcriptDescriptor = 6
	terminalDescriptor   = 7
	expectedDescriptor   = 8
	transcriptBytes      = 64 << 20
	transcriptHeader     = 64
	transcriptRecord     = 128
	terminalBytes        = 104
)

var (
	transcriptMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'R', 1}
	expectedMagic   = [8]byte{'G', 'O', 'M', 'A', 'D', 'X', 'T', 1}
	terminalMagic   = [8]byte{'G', 'O', 'M', 'A', 'D', 'I', 'T', 1}
)

var transcript = struct {
	sync.Mutex
	bytes           []byte
	expected        []byte
	expectedRecords uint64
	offset          uint64
	records         uint64
	divergence      uint64
	replay          bool
	frozen          bool
	finalized       bool
	overflow        bool
}{}

func Init() {
	transcript.Lock()
	defer transcript.Unlock()
	if transcript.bytes != nil {
		return
	}
	mapped, err := syscall.Mmap(transcriptDescriptor, 0, transcriptBytes, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic("gomadv3: map I/O transcript")
	}
	if string(mapped[:8]) != string(transcriptMagic[:]) || binary.BigEndian.Uint32(mapped[8:12]) != 1 || binary.BigEndian.Uint64(mapped[16:24]) != transcriptBytes || binary.BigEndian.Uint64(mapped[24:32]) != transcriptHeader {
		panic("gomadv3: invalid I/O transcript backing")
	}
	expected, err := syscall.Mmap(expectedDescriptor, 0, transcriptBytes, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic("gomadv3: map expected I/O transcript")
	}
	expectedLength := binary.BigEndian.Uint64(expected[16:24])
	expectedRecords := binary.BigEndian.Uint64(expected[24:32])
	if string(expected[:8]) != string(expectedMagic[:]) || binary.BigEndian.Uint32(expected[8:12]) != 1 || expectedLength > transcriptBytes-transcriptHeader || expectedLength%transcriptRecord != 0 || expectedRecords != expectedLength/transcriptRecord {
		panic("gomadv3: invalid expected I/O transcript backing")
	}
	expectedDigest := sum256(expected[transcriptHeader : transcriptHeader+expectedLength])
	if string(expectedDigest[:]) != string(expected[32:64]) {
		panic("gomadv3: invalid expected I/O transcript digest")
	}
	transcript.bytes = mapped
	transcript.expected = expected
	transcript.expectedRecords = expectedRecords
	transcript.offset = transcriptHeader
	transcript.divergence = ^uint64(0)
	transcript.replay = expected[12] == 1
	exithook.Add(exithook.Hook{F: finalize, RunOnFailure: true})
}

func Record(operation string, arguments, content []byte, count uint64, result uint32, entropyStart, entropyEnd uint64) {
	transcript.Lock()
	if transcript.finalized {
		transcript.Unlock()
		return
	}
	if transcript.frozen {
		transcript.Unlock()
		syscall.Exit(125)
		return
	}
	if len(operation) > 22 || transcript.offset > uint64(len(transcript.bytes))-transcriptRecord {
		transcript.frozen = true
		transcript.overflow = true
		transcript.Unlock()
		finalize()
		syscall.Exit(125)
		return
	}
	var encoded [transcriptRecord]byte
	record := encoded[:]
	binary.BigEndian.PutUint64(record[:8], transcript.records)
	binary.BigEndian.PutUint16(record[8:10], uint16(len(operation)))
	copy(record[10:32], operation)
	argumentDigest := sum256(arguments)
	contentDigest := sum256(content)
	copy(record[32:64], argumentDigest[:])
	copy(record[64:96], contentDigest[:])
	binary.BigEndian.PutUint64(record[96:104], count)
	binary.BigEndian.PutUint32(record[104:108], result)
	binary.BigEndian.PutUint64(record[112:120], entropyStart)
	binary.BigEndian.PutUint64(record[120:128], entropyEnd)
	copy(transcript.bytes[transcript.offset:transcript.offset+transcriptRecord], record)
	transcript.offset += transcriptRecord
	transcript.records++
	binary.BigEndian.PutUint64(transcript.bytes[24:32], transcript.offset)
	binary.BigEndian.PutUint64(transcript.bytes[32:40], transcript.records)
	if transcript.replay {
		ordinal := transcript.records - 1
		if ordinal >= transcript.expectedRecords || string(record) != string(transcript.expected[transcriptHeader+ordinal*transcriptRecord:transcriptHeader+(ordinal+1)*transcriptRecord]) {
			transcript.divergence = ordinal
			transcript.frozen = true
			transcript.Unlock()
			finalize()
			syscall.Exit(125)
			return
		}
	}
	transcript.Unlock()
}

func TestingComplete() {
	if transcript.bytes != nil {
		finalize()
	}
}

func finalize() {
	transcript.Lock()
	if transcript.finalized {
		transcript.Unlock()
		return
	}
	transcript.frozen = true
	transcript.finalized = true
	if transcript.replay && transcript.divergence == ^uint64(0) && transcript.records != transcript.expectedRecords {
		transcript.divergence = transcript.records
	}
	digest := sum256(transcript.bytes[transcriptHeader:transcript.offset])
	terminal := make([]byte, terminalBytes)
	copy(terminal[:8], terminalMagic[:])
	binary.BigEndian.PutUint32(terminal[8:12], 1)
	terminal[12] = 1
	if transcript.overflow {
		terminal[12] = 2
	}
	if transcript.divergence != ^uint64(0) {
		terminal[12] = 3
		binary.BigEndian.PutUint64(terminal[64:72], transcript.divergence)
	}
	binary.BigEndian.PutUint64(terminal[16:24], transcript.records)
	binary.BigEndian.PutUint64(terminal[24:32], transcript.offset)
	copy(terminal[32:64], digest[:])
	checksum := sum256(terminal[:72])
	copy(terminal[72:], checksum[:])
	transcript.Unlock()
	written, err := syscall.Write(terminalDescriptor, terminal)
	if err != nil || written != len(terminal) {
		panic("gomadv3: write I/O terminal frame")
	}
}

func sum256(input []byte) [32]byte {
	state := [8]uint32{0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19}
	length := uint64(len(input)) * 8
	paddedLength := (len(input) + 9 + 63) &^ 63
	padded := make([]byte, paddedLength)
	copy(padded, input)
	padded[len(input)] = 0x80
	binary.BigEndian.PutUint64(padded[len(padded)-8:], length)
	for len(padded) != 0 {
		var words [64]uint32
		for index := 0; index < 16; index++ {
			words[index] = binary.BigEndian.Uint32(padded[index*4 : index*4+4])
		}
		for index := 16; index < 64; index++ {
			s0 := rotateRight(words[index-15], 7) ^ rotateRight(words[index-15], 18) ^ words[index-15]>>3
			s1 := rotateRight(words[index-2], 17) ^ rotateRight(words[index-2], 19) ^ words[index-2]>>10
			words[index] = words[index-16] + s0 + words[index-7] + s1
		}
		a, b, c, d := state[0], state[1], state[2], state[3]
		e, f, g, h := state[4], state[5], state[6], state[7]
		for index := 0; index < 64; index++ {
			s1 := rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
			choice := e&f ^ (^e)&g
			temporary1 := h + s1 + choice + sha256Constants[index] + words[index]
			s0 := rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
			majority := a&b ^ a&c ^ b&c
			temporary2 := s0 + majority
			h, g, f, e, d, c, b, a = g, f, e, d+temporary1, c, b, a, temporary1+temporary2
		}
		state[0] += a
		state[1] += b
		state[2] += c
		state[3] += d
		state[4] += e
		state[5] += f
		state[6] += g
		state[7] += h
		padded = padded[64:]
	}
	var result [32]byte
	for index, value := range state {
		binary.BigEndian.PutUint32(result[index*4:index*4+4], value)
	}
	return result
}

func rotateRight(value uint32, bits uint) uint32 {
	return value>>bits | value<<(32-bits)
}

var sha256Constants = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}
