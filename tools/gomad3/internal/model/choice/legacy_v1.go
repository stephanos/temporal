package choice

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	LegacyProfile        = "gomad3-choice-trace/v1"
	legacyHeaderBytes    = 64
	legacyRecordBytes    = 48
	legacyTerminalBytes  = 96
	legacyChecksumOffset = 64
)

var legacyTerminalMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'C', 'T', 1}

func DecodeLegacyV1Trace(payload, terminalFrame []byte, mappingLimit uint64) (Trace, error) {
	if len(terminalFrame) == 0 {
		return Trace{}, ErrUnterminated
	}
	if len(terminalFrame) != legacyTerminalBytes || !bytes.Equal(terminalFrame[:8], legacyTerminalMagic[:]) || binary.BigEndian.Uint32(terminalFrame[8:12]) != Version1 || !zeroBytes(terminalFrame[13:16]) {
		return Trace{}, errors.Join(ErrMalformed, errors.New("invalid legacy choice terminal frame"))
	}
	checksum := sha256.Sum256(terminalFrame[:legacyChecksumOffset])
	if !bytes.Equal(checksum[:], terminalFrame[legacyChecksumOffset:]) {
		return Trace{}, errors.Join(ErrMalformed, errors.New("legacy choice terminal checksum mismatch"))
	}
	state := TerminalState(terminalFrame[12])
	records := binary.BigEndian.Uint64(terminalFrame[16:24])
	mappingBytes := binary.BigEndian.Uint64(terminalFrame[24:32])
	if state != TerminalComplete && state != TerminalOverflow || mappingBytes > mappingLimit || mappingBytes != legacyHeaderBytes+uint64(len(payload)) || len(payload)%legacyRecordBytes != 0 || records != uint64(len(payload))/legacyRecordBytes {
		return Trace{}, errors.Join(ErrMalformed, errors.New("legacy choice terminal bounds do not match payload"))
	}
	digest := sha256.Sum256(payload)
	if !bytes.Equal(digest[:], terminalFrame[32:64]) {
		return Trace{}, errors.Join(ErrMalformed, errors.New("legacy choice trace digest mismatch"))
	}
	result := Trace{
		Version: Version1, Bytes: append([]byte(nil), payload...), SHA256: digest,
		Records: make([]Record, 0, records), Summary: Summary{Records: records, Terminal: state},
	}
	for offset := 0; offset < len(payload); offset += legacyRecordBytes {
		record, err := decodeLegacyV1Record(payload[offset : offset+legacyRecordBytes])
		if err != nil {
			return Trace{}, errors.Join(ErrMalformed, err)
		}
		if record.Ordinal != uint64(len(result.Records)) {
			return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("legacy choice trace ordinal %d at record %d", record.Ordinal, len(result.Records)))
		}
		result.Records = append(result.Records, record)
		if record.Flags&FlagDecision != 0 && record.Alternatives > 1 {
			result.Summary.Branching++
		}
		switch record.Kind {
		case KindRunnable:
			result.Summary.Runnable++
		case KindSelectPoll:
			result.Summary.SelectPoll++
		case KindSelectResult:
			result.Summary.SelectResult++
		}
	}
	if state == TerminalOverflow {
		return result, ErrOverflow
	}
	return result, nil
}

func decodeStoredLegacyV1Trace(payload []byte, metadata TerminalMetadata) (Trace, error) {
	frame := make([]byte, legacyTerminalBytes)
	copy(frame[:8], legacyTerminalMagic[:])
	binary.BigEndian.PutUint32(frame[8:12], Version1)
	frame[12] = byte(metadata.State)
	binary.BigEndian.PutUint64(frame[16:24], metadata.Records)
	binary.BigEndian.PutUint64(frame[24:32], legacyHeaderBytes+uint64(len(payload)))
	copy(frame[32:64], metadata.SHA256[:])
	checksum := sha256.Sum256(frame[:legacyChecksumOffset])
	copy(frame[legacyChecksumOffset:], checksum[:])
	return DecodeLegacyV1Trace(payload, frame, metadata.Limit)
}

func decodeLegacyV1Record(encoded []byte) (Record, error) {
	if len(encoded) != legacyRecordBytes || !zeroBytes(encoded[10:12]) || !zeroBytes(encoded[32:48]) {
		return Record{}, errors.New("invalid legacy choice trace record")
	}
	record := Record{
		Ordinal: binary.BigEndian.Uint64(encoded[:8]), Kind: Kind(encoded[8]), Flags: Flags(encoded[9]),
		Alternatives: binary.BigEndian.Uint32(encoded[12:16]), Selected: binary.BigEndian.Uint32(encoded[16:20]),
		Data: binary.BigEndian.Uint32(encoded[20:24]), SiteOffset: binary.BigEndian.Uint64(encoded[24:32]),
	}
	if record.Kind < KindRunnable || record.Kind > KindSelectResult || record.Flags&^(FlagDecision|FlagObservation|FlagSiteMissing) != 0 || record.Flags&(FlagDecision|FlagObservation) == 0 || record.Flags&(FlagDecision|FlagObservation) == FlagDecision|FlagObservation || record.Kind == KindSelectResult && record.Flags&FlagObservation == 0 || record.Kind != KindSelectResult && record.Flags&FlagDecision == 0 || record.Alternatives == 0 || record.Selected >= record.Alternatives || record.Flags&FlagSiteMissing != 0 && record.SiteOffset != 0 {
		return Record{}, errors.New("invalid legacy choice trace record values")
	}
	return record, nil
}

func zeroBytes(values []byte) bool {
	for _, value := range values {
		if value != 0 {
			return false
		}
	}
	return true
}
