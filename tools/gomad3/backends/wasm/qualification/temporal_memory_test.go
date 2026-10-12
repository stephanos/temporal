package qualification

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/simulation"
	"go.temporal.io/server/tools/gomad3/hostfs"
	"go.temporal.io/server/tools/gomad3sim"
)

type temporalMemoryObservation struct {
	MinimumBytes, AvailableBytes uint64
}

// This opt-in observation is not a reservation or modeled execution input.
func temporalMemoryAdmission(platform, path, minimum string) (temporalMemoryObservation, error) {
	observation := temporalMemoryObservation{}
	if minimum == "" {
		return observation, nil
	}
	value, err := temporalMemoryDecimal(minimum)
	if err != nil || value == 0 {
		return observation, errors.New("positive decimal memory admission threshold required")
	}
	observation.MinimumBytes = value
	if platform != "linux" {
		return observation, errors.New("explicit memory admission requires Linux MemAvailable")
	}
	data, err := hostfs.ReadBounded(path, 8<<10)
	if err != nil {
		return observation, err
	}
	seen := false
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
			return observation, errors.New("unique exact MemAvailable field in kB required")
		}
		seen = true
		kilobytes, err := temporalMemoryDecimal(fields[1])
		if err != nil || kilobytes > ^uint64(0)/1024 {
			return observation, errors.New("MemAvailable value invalid or byte conversion overflow")
		}
		observation.AvailableBytes = kilobytes * 1024
	}
	if !seen {
		return observation, errors.New("MemAvailable observation missing")
	}
	if observation.AvailableBytes < observation.MinimumBytes {
		return observation, errors.New("insufficient available memory before helper dispatch")
	}
	return observation, nil
}

func temporalMemoryDecimal(value string) (uint64, error) {
	if len(value) == 0 || len(value) > 20 {
		return 0, errors.New("bounded decimal value required")
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, errors.New("decimal digits required")
		}
	}
	return strconv.ParseUint(value, 10, 64)
}

func temporalRunAdmittedCluster(t *testing.T, ctx context.Context, spec Spec, cluster gomad3sim.Spec, scenario gomad3sim.Scenario, provider *simulation.Provider, platform, path, minimum string) (Result, temporalMemoryObservation, error) {
	t.Helper()
	observation, err := temporalMemoryAdmission(platform, path, minimum)
	if minimum != "" {
		t.Logf("pre-helper safety observation available-bytes=%d minimum-bytes=%d admitted=%t (not a reservation or measured budget pass)", observation.AvailableBytes, observation.MinimumBytes, err == nil)
	}
	if err != nil {
		return Result{}, observation, err
	}
	result, err := RunCluster(ctx, spec, cluster, scenario, provider)
	return result, observation, err
}

func TestTemporalMemoryAdmissionChecksBoundaryBeforeDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meminfo")
	if err := os.WriteFile(path, []byte("MemTotal: 100 kB\nMemAvailable: 8 kB\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		minimum string
		admit   bool
	}{{"8193", false}, {"8192", true}, {"8191", true}} {
		result, observation, err := temporalRunAdmittedCluster(t, t.Context(), Spec{Manifest: Manifest{ID: "reached-existing-owner"}}, gomad3sim.Spec{}, nil, nil, "linux", path, test.minimum)
		if observation.AvailableBytes != 8192 || observation.MinimumBytes == 0 || err == nil {
			t.Fatalf("boundary observation missing: %+v %v", observation, err)
		}
		if (result.Manifest.ID == "reached-existing-owner") != test.admit {
			t.Fatalf("dispatch decision at %s: result=%+v observation=%+v error=%v", test.minimum, result.Manifest, observation, err)
		}
	}
}

func TestTemporalMemoryAdmissionReadsActualBoundedLinuxObservation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux MemAvailable observation is opt-in only")
	}
	observation, err := temporalMemoryAdmission(runtime.GOOS, "/proc/meminfo", "1")
	if err != nil || observation.AvailableBytes == 0 || observation.MinimumBytes != 1 {
		t.Fatalf("actual bounded MemAvailable unavailable: %+v %v", observation, err)
	}
}

func TestTemporalMemoryAdmissionRejectsMalformedAndOversizedReadings(t *testing.T) {
	for _, data := range []string{
		"", "MemTotal: 100 kB\n", "MemAvailable: 8 kB\nMemAvailable: 8 kB\n",
		"memavailable: 8 kB\n", "MemAvailable: -1 kB\n", "MemAvailable: +1 kB\n",
		"MemAvailable: 1 MB\n", "MemAvailable: 1 kB extra\n", "MemAvailable: 1\n",
		"MemAvailable: 8 kB\nMemAvailable : 8 kB\n", "MemAvailable: 8 kB\nMemAvailable:8 kB\n",
		"MemAvailable: 18446744073709551616 kB\n", "MemAvailable: 18014398509481984 kB\n",
		"MemAvailable: 8 kB\n" + strings.Repeat(" ", 8<<10),
	} {
		path := filepath.Join(t.TempDir(), "meminfo")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := temporalMemoryAdmission("linux", path, "1"); err == nil {
			t.Fatalf("malformed memory observation accepted: %q", data)
		}
	}
}

func TestTemporalMemoryAdmissionIsExplicitAndPlatformBounded(t *testing.T) {
	for _, minimum := range []string{"0", "-1", "+1", " 1", "18446744073709551616"} {
		if _, err := temporalMemoryAdmission("linux", "unreadable", minimum); err == nil {
			t.Fatalf("invalid threshold accepted: %q", minimum)
		}
	}
	if _, err := temporalMemoryAdmission("darwin", "unreadable", "1"); err == nil {
		t.Fatal("explicit Linux safety check silently admitted unknown platform")
	}
	if observation, err := temporalMemoryAdmission("darwin", "unreadable", ""); err != nil || observation.MinimumBytes != 0 || observation.AvailableBytes != 0 {
		t.Fatalf("absent opt-in changed default host behavior: %+v %v", observation, err)
	}
}
