package deterministicio

import (
	"crypto/sha256"
	"fmt"

	iowire "go.temporal.io/server/tools/gomad3/deterministicio/internal/wire"
)

type Operation struct {
	Ordinal      uint64
	Name         string
	ArgumentHash [sha256.Size]byte
	ContentHash  [sha256.Size]byte
	Count        uint64
	Result       uint32
	EntropyStart uint64
	EntropyEnd   uint64
}

func EncodeTranscript(operations []Operation) ([]byte, error) {
	encoded := make([]byte, 0, len(operations)*iowire.TranscriptRecordBytes)
	for index, operation := range operations {
		if operation.Ordinal != uint64(index) {
			return nil, fmt.Errorf("I/O transcript ordinal %d at record %d", operation.Ordinal, index)
		}
		record, err := encodeOperation(operation)
		if err != nil {
			return nil, fmt.Errorf("encode I/O transcript record %d: %w", index, err)
		}
		encoded = append(encoded, record[:]...)
	}
	return encoded, nil
}

func EncodeOperation(operation Operation) ([]byte, error) {
	record, err := encodeOperation(operation)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), record[:]...), nil
}

func encodeOperation(operation Operation) ([iowire.TranscriptRecordBytes]byte, error) {
	return iowire.EncodeTranscriptRecord(iowire.TranscriptRecord{
		Ordinal: operation.Ordinal, Operation: operation.Name, ArgumentHash: operation.ArgumentHash,
		ContentHash: operation.ContentHash, Count: operation.Count, Result: operation.Result,
		EntropyStart: operation.EntropyStart, EntropyEnd: operation.EntropyEnd,
	})
}

func DecodeTranscript(encoded []byte) ([]Operation, error) {
	if len(encoded)%iowire.TranscriptRecordBytes != 0 {
		return nil, fmt.Errorf("I/O transcript has invalid length %d", len(encoded))
	}
	operations := make([]Operation, 0, len(encoded)/iowire.TranscriptRecordBytes)
	for offset := 0; offset < len(encoded); offset += iowire.TranscriptRecordBytes {
		record, err := iowire.DecodeTranscriptRecord(encoded[offset : offset+iowire.TranscriptRecordBytes])
		if err != nil {
			return nil, fmt.Errorf("decode I/O transcript record %d: %w", offset/iowire.TranscriptRecordBytes, err)
		}
		if record.Ordinal != uint64(len(operations)) {
			return nil, fmt.Errorf("I/O transcript ordinal %d at record %d", record.Ordinal, len(operations))
		}
		operations = append(operations, Operation{
			Ordinal: record.Ordinal, Name: record.Operation, ArgumentHash: record.ArgumentHash,
			ContentHash: record.ContentHash, Count: record.Count, Result: record.Result,
			EntropyStart: record.EntropyStart, EntropyEnd: record.EntropyEnd,
		})
	}
	return operations, nil
}

func TranscriptRecordCount(encoded []byte) (uint64, error) {
	if len(encoded)%iowire.TranscriptRecordBytes != 0 {
		return 0, fmt.Errorf("I/O transcript has invalid length %d", len(encoded))
	}
	return uint64(len(encoded) / iowire.TranscriptRecordBytes), nil
}

func HashArgument(encoded []byte) [sha256.Size]byte {
	return iowire.Hash(encoded)
}
