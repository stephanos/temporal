package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
)

func DecodeCompressedSnapshot(data []byte, limits ProfileLimits) (CompressedSnapshotSummary, error) {
	snapshot, requests, events, transitions, envelope, err := decodeCompressedSnapshot(data, limits)
	if err != nil {
		return CompressedSnapshotSummary{}, err
	}
	return CompressedSnapshotSummary{
		Config: snapshot.Config, Now: snapshot.Now, RequestCount: requests, EventCount: events, TransitionCount: transitions,
		PayloadBytes: snapshot.PayloadBytes, TranscriptDigest: snapshot.TranscriptDigest, StateDigest: snapshot.StateDigest,
		Replay: snapshot.Replay, ExpandedBytes: envelope.ExpandedBytes, CompressedBytes: envelope.CompressedBytes, ExpandedSHA256: envelope.ExpandedSHA256,
	}, nil
}

// DecodeCompressedSnapshotState returns the verified full snapshot carried by
// a compressed envelope, for consumers that replay or compare its transitions.
func DecodeCompressedSnapshotState(data []byte, limits ProfileLimits) (Snapshot, error) {
	snapshot, _, _, _, _, err := decodeCompressedSnapshot(data, limits)
	return snapshot, err
}

// WrapExpandedSnapshot accepts an expanded snapshot wire written outside Go,
// such as by the Rust cluster helper's generated writer. It decodes and replays
// it with the generated decoder, requires canonical bytes, and returns the
// compressed envelope Go's own encoder produces for that snapshot.
func WrapExpandedSnapshot(expanded []byte, limits ProfileLimits) ([]byte, error) {
	if err := validateCompressedProfile(limits); err != nil {
		return nil, err
	}
	if len(expanded) == 0 || uint64(len(expanded)) > limits.MaxExpandedBytes {
		return nil, invalidSnapshot("compressed.expanded.bytes")
	}
	snapshot, _, _, _, err := decodeCompressedSnapshotWire(&expandedWireReader{data: expanded}, limits)
	if err != nil {
		return nil, err
	}
	digest, size, err := compressedSnapshotWireDigest(snapshot, limits.MaxExpandedBytes)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(expanded)
	if size != uint64(len(expanded)) || digest != Digest(hex.EncodeToString(sum[:])) {
		return nil, invalidSnapshot("compressed.expanded.canonical")
	}
	return encodeCompressedSnapshot(snapshot, limits)
}

type expandedWireReader struct{ data []byte }

func (reader *expandedWireReader) Remaining() uint64 { return uint64(len(reader.data)) }

func (reader *expandedWireReader) Read(data []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	n := copy(data, reader.data)
	reader.data = reader.data[n:]
	return n, nil
}

func decodeCompressedSnapshot(data []byte, limits ProfileLimits) (Snapshot, uint64, uint64, uint64, compressedSnapshotEnvelope, error) {
	if err := validateCompressedProfile(limits); err != nil {
		return Snapshot{}, 0, 0, 0, compressedSnapshotEnvelope{}, err
	}
	envelope, err := decodeCompressedEnvelope(data, limits)
	if err != nil {
		return Snapshot{}, 0, 0, 0, compressedSnapshotEnvelope{}, err
	}
	var snapshot Snapshot
	var requests, events, transitions uint64
	if err := compressedGzipPass(envelope, func(reader *compressedHashReader) error {
		var decodeErr error
		snapshot, requests, events, transitions, decodeErr = decodeCompressedSnapshotWire(reader, limits)
		return decodeErr
	}); err != nil {
		return Snapshot{}, 0, 0, 0, compressedSnapshotEnvelope{}, err
	}
	wireDigest, wireSize, err := compressedSnapshotWireDigest(snapshot, limits.MaxExpandedBytes)
	if err != nil {
		return Snapshot{}, 0, 0, 0, compressedSnapshotEnvelope{}, err
	}
	if wireDigest != envelope.ExpandedSHA256 || wireSize != envelope.ExpandedBytes {
		return Snapshot{}, 0, 0, 0, compressedSnapshotEnvelope{}, invalidSnapshot("compressed.snapshot.canonical-or-state")
	}
	return snapshot, requests, events, transitions, envelope, nil
}

func decodeCompressedEnvelope(data []byte, limits ProfileLimits) (compressedSnapshotEnvelope, error) {
	if uint64(len(data)) > uint64(compressedEnvelopeHeaderBytes)+limits.MaxCompressedBytes {
		return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.bytes")
	}
	if len(data) < compressedEnvelopeHeaderBytes || !bytes.Equal(data[:8], compressedEnvelopeMagic[:]) ||
		binary.BigEndian.Uint32(data[8:12]) != compressedEnvelopeHeaderBytes || !allZero(data[12:16]) {
		return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.header")
	}
	expandedBytes := binary.BigEndian.Uint64(data[16:24])
	compressedBytes := binary.BigEndian.Uint64(data[24:32])
	if expandedBytes == 0 || expandedBytes > limits.MaxExpandedBytes || compressedBytes == 0 || compressedBytes > limits.MaxCompressedBytes ||
		compressedBytes != uint64(len(data)-compressedEnvelopeHeaderBytes) {
		return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.identity")
	}
	digest := Digest(hex.EncodeToString(data[32:64]))
	if !validDigest(digest) {
		return compressedSnapshotEnvelope{}, invalidSnapshot("compressed.envelope.digest")
	}
	return compressedSnapshotEnvelope{
		ExpandedBytes: expandedBytes, CompressedBytes: compressedBytes, ExpandedSHA256: digest,
		Gzip: data[compressedEnvelopeHeaderBytes:],
	}, nil
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}
