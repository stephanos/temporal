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

func (selection SeedSelection) String() string {
	terms := make([]string, len(selection.ranges))
	for index, selected := range selection.ranges {
		terms[index] = strconv.FormatUint(selected.start, 10)
		if selected.end != selected.start {
			terms[index] += "-" + strconv.FormatUint(selected.end, 10)
		}
	}
	return strings.Join(terms, ",")
}

func mixGuidedSelection(base SeedSelection, prioritized []uint64) (SeedSelection, error) {
	unguided := base.count / 4
	if base.count%4 != 0 {
		unguided++
	}
	maximumGuided := base.count - unguided
	guided := make([]uint64, 0, min(uint64(len(prioritized)), maximumGuided))
	guidedSet := make(map[uint64]struct{}, cap(guided))
	for _, seed := range prioritized {
		if uint64(len(guided)) == maximumGuided {
			break
		}
		if _, found := guidedSet[seed]; found {
			continue
		}
		guidedSet[seed] = struct{}{}
		guided = append(guided, seed)
	}
	if len(guided) == 0 {
		return base, nil
	}
	ranges := make([]seedRange, 0, len(guided)+len(base.ranges))
	for _, seed := range guided {
		ranges = append(ranges, seedRange{start: seed, end: seed})
	}
	remaining := base.count - uint64(len(guided))
	for _, selected := range base.ranges {
		excluded := make([]uint64, 0)
		for seed := range guidedSet {
			if seed >= selected.start && seed <= selected.end {
				excluded = append(excluded, seed)
			}
		}
		sort.Slice(excluded, func(i, j int) bool { return excluded[i] < excluded[j] })
		start := selected.start
		for _, seed := range excluded {
			if start < seed {
				appendGuidedRange(&ranges, start, seed-1, &remaining)
			}
			if remaining == 0 {
				break
			}
			if seed != math.MaxUint64 {
				start = seed + 1
			}
		}
		if remaining != 0 && (len(excluded) == 0 || excluded[len(excluded)-1] != math.MaxUint64) && start <= selected.end {
			appendGuidedRange(&ranges, start, selected.end, &remaining)
		}
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		return SeedSelection{}, fmt.Errorf("guided seed selection could not reserve its unguided fraction")
	}
	return SeedSelection{ranges: ranges, count: base.count}, nil
}

func appendGuidedRange(ranges *[]seedRange, start, end uint64, remaining *uint64) {
	width := end - start + 1
	if width > *remaining {
		end = start + *remaining - 1
		width = *remaining
	}
	*ranges = append(*ranges, seedRange{start: start, end: end})
	*remaining -= width
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
