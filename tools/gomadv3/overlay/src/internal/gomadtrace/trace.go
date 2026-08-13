// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadtrace

import (
	"encoding/binary"
	"sync"
	"syscall"

	"internal/gomadwire"
	"internal/runtime/exithook"
)

const (
	transcriptDescriptor = 6
	terminalDescriptor   = 7
	expectedDescriptor   = 8
	transcriptBytes      = 64 << 20
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
	probeIDs        [256]uint64
	probeCount      uint16
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
	producedHeader, err := gomadwire.DecodeProducedTranscriptHeader(mapped[:gomadwire.TranscriptHeaderBytes])
	if err != nil || producedHeader.Capacity != transcriptBytes || producedHeader.NextOffset != gomadwire.TranscriptHeaderBytes || producedHeader.RecordCount != 0 {
		panic("gomadv3: invalid I/O transcript backing")
	}
	expected, err := syscall.Mmap(expectedDescriptor, 0, transcriptBytes, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic("gomadv3: map expected I/O transcript")
	}
	expectedHeader, err := gomadwire.DecodeExpectedTranscriptHeader(expected[:gomadwire.TranscriptHeaderBytes], transcriptBytes)
	if err != nil {
		panic("gomadv3: invalid expected I/O transcript backing")
	}
	expectedDigest := gomadwire.Hash(expected[gomadwire.TranscriptHeaderBytes : gomadwire.TranscriptHeaderBytes+expectedHeader.PayloadBytes])
	if expectedDigest != expectedHeader.PayloadHash {
		panic("gomadv3: invalid expected I/O transcript digest")
	}
	transcript.bytes = mapped
	transcript.expected = expected
	transcript.expectedRecords = expectedHeader.RecordCount
	transcript.offset = gomadwire.TranscriptHeaderBytes
	transcript.divergence = ^uint64(0)
	transcript.replay = expectedHeader.Replay
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
	if len(operation) > gomadwire.TranscriptOperationBytes || transcript.offset > uint64(len(transcript.bytes))-gomadwire.TranscriptRecordBytes {
		transcript.frozen = true
		transcript.overflow = true
		transcript.Unlock()
		finalize()
		syscall.Exit(125)
		return
	}
	encoded, err := gomadwire.EncodeTranscriptRecord(gomadwire.TranscriptRecord{
		Ordinal: transcript.records, Operation: operation, ArgumentHash: gomadwire.Hash(arguments), ContentHash: gomadwire.Hash(content),
		Count: count, Result: result, EntropyStart: entropyStart, EntropyEnd: entropyEnd,
	})
	if err != nil {
		transcript.frozen = true
		transcript.overflow = true
		transcript.Unlock()
		finalize()
		syscall.Exit(125)
		return
	}
	record := encoded[:]
	copy(transcript.bytes[transcript.offset:transcript.offset+gomadwire.TranscriptRecordBytes], record)
	transcript.offset += gomadwire.TranscriptRecordBytes
	transcript.records++
	if err := gomadwire.PublishProducedTranscript(transcript.bytes[:gomadwire.TranscriptHeaderBytes], transcript.offset, transcript.records); err != nil {
		transcript.Unlock()
		panic("gomadv3: publish I/O transcript")
	}
	if transcript.replay {
		ordinal := transcript.records - 1
		if ordinal >= transcript.expectedRecords || string(record) != string(transcript.expected[gomadwire.TranscriptHeaderBytes+ordinal*gomadwire.TranscriptRecordBytes:gomadwire.TranscriptHeaderBytes+(ordinal+1)*gomadwire.TranscriptRecordBytes]) {
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

func ObserveBoundary(id uint64) {
	transcript.Lock()
	if transcript.bytes == nil || transcript.finalized {
		transcript.Unlock()
		return
	}
	for index := uint16(0); index < transcript.probeCount; index++ {
		if transcript.probeIDs[index] == id {
			transcript.Unlock()
			return
		}
	}
	if int(transcript.probeCount) == len(transcript.probeIDs) {
		transcript.Unlock()
		panic("gomadv3: boundary probe capacity exceeded")
	}
	transcript.probeIDs[transcript.probeCount] = id
	transcript.probeCount++
	transcript.Unlock()
	var argument [8]byte
	binary.BigEndian.PutUint64(argument[:], id)
	Record("boundary.probe", argument[:], nil, 0, 0, 0, 0)
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
	digest := gomadwire.Hash(transcript.bytes[gomadwire.TranscriptHeaderBytes:transcript.offset])
	state := gomadwire.TerminalComplete
	if transcript.overflow {
		state = gomadwire.TerminalOverflow
	}
	divergence := uint64(0)
	if transcript.divergence != ^uint64(0) {
		state = gomadwire.TerminalReplayDivergence
		divergence = transcript.divergence
	}
	terminal := gomadwire.EncodeTerminal(gomadwire.Terminal{State: state, Records: transcript.records, MappingBytes: transcript.offset, PayloadHash: digest, DivergentOrdinal: divergence})
	transcript.Unlock()
	written, err := syscall.Write(terminalDescriptor, terminal[:])
	if err != nil || written != len(terminal) {
		panic("gomadv3: write I/O terminal frame")
	}
}
