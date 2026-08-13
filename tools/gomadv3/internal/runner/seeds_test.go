package runner

import (
	"math"
	"testing"
)

func TestParseSeedsIteratesInInputOrder(t *testing.T) {
	selection, err := ParseSeeds("7,0,11-13,18446744073709551615")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := selection.Count(), uint64(6); got != want {
		t.Fatalf("Count() = %d, want %d", got, want)
	}

	iterator := selection.Iterator()
	want := []uint64{7, 0, 11, 12, 13, math.MaxUint64}
	for index, wantSeed := range want {
		gotSeed, ok := iterator.Next()
		if !ok {
			t.Fatalf("Next() at index %d reported exhaustion", index)
		}
		if gotSeed != wantSeed {
			t.Fatalf("Next() at index %d = %d, want %d", index, gotSeed, wantSeed)
		}
	}
	if seed, ok := iterator.Next(); ok {
		t.Fatalf("Next() after exhaustion = %d, true", seed)
	}
}

func TestParseSeedsRejectsMalformedSelection(t *testing.T) {
	tests := []string{
		"",
		" ",
		"1,",
		",1",
		"1,,2",
		"+1",
		"-1",
		" 1",
		"1 ",
		"01",
		"1-",
		"-2",
		"2-1",
		"1-2-3",
		"18446744073709551616",
		"0-18446744073709551615",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseSeeds(input); err == nil {
				t.Fatalf("ParseSeeds(%q) succeeded", input)
			}
		})
	}
}

func TestParseSeedsRejectsDuplicatesAndOverlaps(t *testing.T) {
	for _, input := range []string{"1,1", "1-3,2", "3,1-3", "5-7,4-6", "9-11,10-12"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseSeeds(input); err == nil {
				t.Fatalf("ParseSeeds(%q) succeeded", input)
			}
		})
	}
}

func TestParseSeedsKeepsMaximumSizedLazySelection(t *testing.T) {
	selection, err := ParseSeeds("1-18446744073709551615")
	if err != nil {
		t.Fatal(err)
	}
	if got := selection.Count(); got != math.MaxUint64 {
		t.Fatalf("Count() = %d, want %d", got, uint64(math.MaxUint64))
	}
	iterator := selection.Iterator()
	for want := uint64(1); want <= 3; want++ {
		if got, ok := iterator.Next(); !ok || got != want {
			t.Fatalf("Next() = %d, %v, want %d, true", got, ok, want)
		}
	}
}

func TestSeedSelectionLooksUpOrdinalWithoutIteration(t *testing.T) {
	selection, err := ParseSeeds("7,0,11-13,18446744073709551615")
	if err != nil {
		t.Fatal(err)
	}
	for ordinal, want := range []uint64{7, 0, 11, 12, 13, math.MaxUint64} {
		if got, ok := selection.SeedAt(uint64(ordinal)); !ok || got != want {
			t.Fatalf("SeedAt(%d) = %d, %t, want %d, true", ordinal, got, ok, want)
		}
	}
	if seed, ok := selection.SeedAt(6); ok {
		t.Fatalf("SeedAt(6) = %d, true", seed)
	}
}
