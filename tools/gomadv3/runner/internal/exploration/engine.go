package exploration

import (
	"errors"
	"slices"

	"go.temporal.io/server/tools/gomadv3/record"
)

func NextRound[C any](queue []C, parallel int, stopped bool, clone func(C) C) ([]C, bool) {
	if stopped || len(queue) == 0 {
		return nil, false
	}
	count := min(parallel, len(queue))
	round := make([]C, count)
	for index := range round {
		round[index] = clone(queue[index])
	}
	return round, true
}

func InsertIdentity(values []record.SHA256, value record.SHA256) []record.SHA256 {
	index, found := slices.BinarySearch(values, value)
	if found {
		return values
	}
	values = append(values, "")
	copy(values[index+1:], values[index:])
	values[index] = value
	return values
}

func ContainsIdentity(values []record.SHA256, value record.SHA256) bool {
	_, found := slices.BinarySearch(values, value)
	return found
}

func SumBytes[C any](values []C, size func(C) (uint64, error)) (uint64, error) {
	var total uint64
	for _, value := range values {
		bytes, err := size(value)
		if err != nil {
			return 0, err
		}
		if bytes > ^uint64(0)-total {
			return 0, errors.New("exploration byte accounting overflow")
		}
		total += bytes
	}
	return total, nil
}
