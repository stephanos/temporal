package volume

import (
	"errors"
	"syscall"
)

type RecorderLimits struct {
	Records    uint64
	Operations uint64
}

type RecorderFault struct {
	Ordinal uint64
	Errno   syscall.Errno
}

type VolumeRecord struct {
	Volume             string
	Kind               string
	Operations         []Operation
	SelectedOperations []uint64
	Errno              syscall.Errno
}

// Recorder retains bounded detached decisions and owns an immutable ordinal
// fault plan. It never dispatches application callbacks.
type Recorder struct {
	limits     RecorderLimits
	faults     map[uint64]syscall.Errno
	records    []VolumeRecord
	operations uint64
}

func NewRecorder(limits RecorderLimits, faults []RecorderFault) (*Recorder, error) {
	if limits.Records == 0 || limits.Records > maximumNodes || limits.Operations == 0 || limits.Operations > maximumNodes {
		return nil, errors.New("volume recorder limits are invalid")
	}
	if uint64(len(faults)) > limits.Records {
		return nil, syscall.ENOSPC
	}
	recorder := &Recorder{limits: limits, faults: make(map[uint64]syscall.Errno, len(faults))}
	var previous uint64
	for _, fault := range faults {
		if fault.Ordinal <= previous || fault.Ordinal > limits.Records || fault.Errno == 0 {
			return nil, errors.New("volume recorder faults are invalid")
		}
		recorder.faults[fault.Ordinal] = fault.Errno
		previous = fault.Ordinal
	}
	return recorder, nil
}

func (recorder *Recorder) BeforeVolumeOperations(id string, operations []Operation) error {
	return recorder.record(VolumeRecord{Volume: id, Kind: "operations", Operations: operations})
}

func (recorder *Recorder) BeforeVolumeControl(id, kind string, selected []uint64) error {
	return recorder.record(VolumeRecord{Volume: id, Kind: kind, SelectedOperations: selected})
}

func (recorder *Recorder) record(record VolumeRecord) error {
	ordinal := uint64(len(recorder.records)) + 1
	if ordinal > recorder.limits.Records {
		return &VolumeCapacityError{Resource: "recorder_records", Required: ordinal, Maximum: recorder.limits.Records}
	}
	operations := uint64(len(record.Operations)) + uint64(len(record.SelectedOperations))
	for _, operation := range record.Operations {
		operations += uint64(len(operation.Dependencies))
	}
	if operations > recorder.limits.Operations-recorder.operations {
		return &VolumeCapacityError{Resource: "recorder_operations", Required: recorder.operations + operations, Maximum: recorder.limits.Operations}
	}
	record.Errno = recorder.faults[ordinal]
	recorder.records = append(recorder.records, cloneVolumeRecord(record))
	recorder.operations += operations
	if record.Errno != 0 {
		return record.Errno
	}
	return nil
}

func (recorder *Recorder) Records() []VolumeRecord {
	result := make([]VolumeRecord, len(recorder.records))
	for index, record := range recorder.records {
		result[index] = cloneVolumeRecord(record)
	}
	return result
}

func cloneVolumeRecord(record VolumeRecord) VolumeRecord {
	record.Operations = append([]Operation(nil), record.Operations...)
	for index := range record.Operations {
		record.Operations[index].Dependencies = append([]uint64(nil), record.Operations[index].Dependencies...)
	}
	record.SelectedOperations = append([]uint64(nil), record.SelectedOperations...)
	return record
}
