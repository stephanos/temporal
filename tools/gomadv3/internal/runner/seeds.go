package runner

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type SeedSelection struct {
	ranges []seedRange
	count  uint64
}

type SeedIterator struct {
	ranges     []seedRange
	rangeIndex int
	current    uint64
	started    bool
}

type seedRange struct {
	start uint64
	end   uint64
}

func ParseSeeds(input string) (SeedSelection, error) {
	if input == "" {
		return SeedSelection{}, fmt.Errorf("invalid seed selection: empty input")
	}
	terms := strings.Split(input, ",")
	ranges := make([]seedRange, 0, len(terms))
	var count uint64
	for _, term := range terms {
		parts := strings.Split(term, "-")
		if len(parts) > 2 {
			return SeedSelection{}, fmt.Errorf("invalid seed selection term %q", term)
		}
		start, err := parseSeed(parts[0])
		if err != nil {
			return SeedSelection{}, fmt.Errorf("invalid seed selection term %q: %w", term, err)
		}
		end := start
		if len(parts) == 2 {
			end, err = parseSeed(parts[1])
			if err != nil {
				return SeedSelection{}, fmt.Errorf("invalid seed selection term %q: %w", term, err)
			}
			if end < start {
				return SeedSelection{}, fmt.Errorf("invalid seed selection term %q: reversed range", term)
			}
		}
		width := end - start
		if width == math.MaxUint64 || math.MaxUint64-count <= width {
			return SeedSelection{}, fmt.Errorf("invalid seed selection: selected count overflows uint64")
		}
		count += width + 1
		ranges = append(ranges, seedRange{start: start, end: end})
	}

	sortedRanges := append([]seedRange(nil), ranges...)
	sort.Slice(sortedRanges, func(i, j int) bool {
		if sortedRanges[i].start != sortedRanges[j].start {
			return sortedRanges[i].start < sortedRanges[j].start
		}
		return sortedRanges[i].end < sortedRanges[j].end
	})
	for index := 1; index < len(sortedRanges); index++ {
		if sortedRanges[index].start <= sortedRanges[index-1].end {
			return SeedSelection{}, fmt.Errorf("invalid seed selection: duplicate or overlapping ranges")
		}
	}

	return SeedSelection{ranges: ranges, count: count}, nil
}

func (selection SeedSelection) Count() uint64 {
	return selection.count
}

func (selection SeedSelection) Iterator() *SeedIterator {
	return &SeedIterator{ranges: selection.ranges}
}

func (selection SeedSelection) SeedAt(ordinal uint64) (uint64, bool) {
	if ordinal >= selection.count {
		return 0, false
	}
	for _, selected := range selection.ranges {
		width := selected.end - selected.start + 1
		if ordinal < width {
			return selected.start + ordinal, true
		}
		ordinal -= width
	}
	return 0, false
}

func (iterator *SeedIterator) Next() (uint64, bool) {
	for iterator.rangeIndex < len(iterator.ranges) {
		currentRange := iterator.ranges[iterator.rangeIndex]
		if !iterator.started {
			iterator.current = currentRange.start
			iterator.started = true
			return iterator.current, true
		}
		if iterator.current < currentRange.end {
			iterator.current++
			return iterator.current, true
		}
		iterator.rangeIndex++
		iterator.started = false
	}
	return 0, false
}

func parseSeed(input string) (uint64, error) {
	if input == "" {
		return 0, fmt.Errorf("empty seed")
	}
	if len(input) > 1 && input[0] == '0' {
		return 0, fmt.Errorf("leading zero")
	}
	for _, character := range input {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("non-decimal seed")
		}
	}
	seed, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("seed out of range: %w", err)
	}
	return seed, nil
}
