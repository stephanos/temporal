package wasi

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/hostfs"
)

// The frontend suite stays opt-in; a selection is a comma-separated list.
func matchedSuiteSelected(selection, suite string) bool {
	if selection == "" {
		return suite != "frontend"
	}
	for _, name := range strings.Split(selection, ",") {
		if strings.TrimSpace(name) == suite {
			return true
		}
	}
	return false
}

func matchedBaselineCache(baseline, candidate string) error {
	if !filepath.IsAbs(baseline) || filepath.Clean(baseline) == filepath.Clean(candidate) {
		return errors.New("GOMAD_WASM_MATCHED_BASELINE_CACHE must be absolute and distinct from the candidate cache")
	}
	return nil
}

// Even samples run the baseline first and odd samples the candidate first.
func matchedBaselineTurn(sample, turn int) bool {
	return (sample+turn)%2 == 0
}

func TestMatchedSuiteSelectionCacheAndOrder(t *testing.T) {
	for _, test := range []struct {
		selection, suite string
		want             bool
	}{{"", "workflow", true}, {"", "frontend", false}, {"frontend", "frontend", true}, {"workflow, frontend", "frontend", true}, {"workflow,update", "update", true}, {"workflow,update", "timers", false}} {
		if got := matchedSuiteSelected(test.selection, test.suite); got != test.want {
			t.Fatalf("selection %q suite %s = %t", test.selection, test.suite, got)
		}
	}
	candidate := filepath.Join(t.TempDir(), "candidate")
	for _, baseline := range []string{"", "relative", candidate, candidate + "/"} {
		if matchedBaselineCache(baseline, candidate) == nil {
			t.Fatalf("baseline cache %q admitted", baseline)
		}
	}
	if err := matchedBaselineCache(filepath.Join(t.TempDir(), "baseline"), candidate); err != nil {
		t.Fatal(err)
	}
	for sample, want := range []bool{true, false, true, false} {
		if matchedBaselineTurn(sample, 0) != want || matchedBaselineTurn(sample, 1) == want {
			t.Fatalf("sample %d order is not alternating", sample)
		}
	}
}

func matchedPrepareAdmitted(platform, path, mode string, dispatch func() (Result, error)) (Result, error) {
	if mode == "" {
		return dispatch()
	}
	if mode != "1" || platform != "linux" {
		return Result{}, errors.New("explicit matched memory admission requires mode 1 on Linux")
	}
	data, err := hostfs.ReadBounded(path, 8<<10)
	if err != nil {
		return Result{}, err
	}
	seen := false
	var available uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		key, _, _ := strings.Cut(fields[0], ":")
		if !strings.EqualFold(key, "MemAvailable") {
			continue
		}
		if seen || fields[0] != "MemAvailable:" || len(fields) != 3 || fields[2] != "kB" {
			return Result{}, errors.New("unique exact MemAvailable field in kB required")
		}
		seen = true
		value, err := matchedMemoryDecimal(fields[1])
		if err != nil || value > ^uint64(0)/1024 {
			return Result{}, errors.New("invalid or overflowing MemAvailable value")
		}
		available = value * 1024
	}
	if !seen || available < 8<<30 {
		return Result{}, errors.New("at least 8GiB MemAvailable required before matched preparation dispatch")
	}
	return dispatch()
}

func matchedMemoryDecimal(value string) (uint64, error) {
	if len(value) == 0 || len(value) > 20 {
		return 0, errors.New("bounded memory decimal required")
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, errors.New("memory decimal digits required")
		}
	}
	return strconv.ParseUint(value, 10, 64)
}

func TestMatchedPreparationMemoryBoundaryBeforeBothOwners(t *testing.T) {
	for _, owner := range []string{"original", "candidate"} {
		t.Run(owner, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "meminfo")
			for _, test := range []struct {
				data string
				want bool
			}{
				{"MemAvailable: 8388607 kB\n", false},
				{"MemAvailable: 8388608 kB\n", true},
				{"MemAvailable: 8388609 kB\n", true},
			} {
				if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
					t.Fatal(err)
				}
				called := false
				result, err := matchedPrepareAdmitted("linux", path, "1", func() (Result, error) {
					called = true
					return Result{Termination: owner}, nil
				})
				if called != test.want || (err == nil) != test.want || (result.Termination == owner) != test.want {
					t.Fatalf("owner=%s observation=%q dispatched=%t want=%t error=%v", owner, test.data, called, test.want, err)
				}
			}
		})
	}
}

func TestMatchedPreparationMemoryRejectsMalformedBeforeDispatch(t *testing.T) {
	for _, data := range []string{
		"", "MemTotal: 100 kB\n", "MemAvailable: 8388608 kB\nMemAvailable: 8388608 kB\n",
		"memavailable: 8388608 kB\n", "MemAvailable: -1 kB\n", "MemAvailable: +1 kB\n",
		"MemAvailable: 8388608 MB\n", "MemAvailable: 8388608 kB extra\n", "MemAvailable: 8388608\n",
		"MemAvailable: 8388608 kB\nMemAvailable : 8388608 kB\n", "MemAvailable: 8388608 kB\nMemAvailable:8388608 kB\n",
		"MemAvailable: 18446744073709551616 kB\n", "MemAvailable: 18014398509481984 kB\n",
		"MemAvailable: 8388608 kB\n" + strings.Repeat(" ", 8<<10),
	} {
		path := filepath.Join(t.TempDir(), "meminfo")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		for _, owner := range []string{"original", "candidate"} {
			called := false
			_, err := matchedPrepareAdmitted("linux", path, "1", func() (Result, error) {
				called = true
				return Result{Termination: owner}, nil
			})
			if called || err == nil {
				t.Fatalf("owner=%s malformed observation dispatched: %q error=%v", owner, data, err)
			}
		}
	}
}

func TestMatchedPreparationMemoryIsExplicitAndPlatformBound(t *testing.T) {
	for _, test := range []struct {
		platform, mode string
		want           bool
	}{
		{"darwin", "", true}, {"linux", "", true},
		{"darwin", "1", false}, {"linux", "1", false},
		{"linux", "0", false}, {"linux", "true", false}, {"linux", " 1", false},
	} {
		called := false
		_, err := matchedPrepareAdmitted(test.platform, "unreadable", test.mode, func() (Result, error) {
			called = true
			return Result{}, nil
		})
		if called != test.want || (err == nil) != test.want {
			t.Fatalf("platform=%s mode=%q called=%t error=%v", test.platform, test.mode, called, err)
		}
	}
}
