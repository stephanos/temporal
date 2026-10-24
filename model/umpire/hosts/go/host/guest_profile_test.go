package host

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestGuestProfileCaughtPanicParity(t *testing.T) {
	for _, tc := range []struct {
		name, wasmEnv, failure string
	}{
		{"effect_panic", "UMPIRE_CAUGHT_PANIC_WASM", "broken heater"},
		{"effect_overflow", "UMPIRE_OVERFLOW_WASM", "attempt to add with overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			native, err := os.ReadFile(filepath.Join(os.Getenv("UMPIRE_PANIC_NATIVE"), tc.name+".pb"))
			if err != nil {
				t.Fatal(err)
			}
			want := new(modelv1.Response)
			if err := proto.Unmarshal(native, want); err != nil {
				t.Fatal(err)
			}
			if report := want.GetExplore(); report == nil || !strings.Contains(report.Report, "outcome: Refuted\n") || !strings.Contains(report.Report, "panicked: "+tc.failure+"\n") {
				t.Fatalf("native control did not report the expected caught defect: %v", want)
			}
			wasm, err := os.ReadFile(os.Getenv(tc.wasmEnv))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			host, err := New(ctx, wasm)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := host.Close(ctx); err != nil {
					t.Error(err)
				}
			})
			for attempt := range 3 {
				if attempt == 2 {
					if _, err := host.Reset(ctx, &modelv1.ResetRequest{}); err != nil {
						t.Fatalf("Reset after caught defect: %v", err)
					}
				}
				got, err := host.Explore(ctx, &modelv1.ExploreRequest{Budget: 3})
				if err != nil {
					t.Fatalf("Explore attempt %d trapped instead of reporting a caught defect: %v", attempt, err)
				}
				if !proto.Equal(got, want.GetExplore()) {
					t.Fatalf("Explore attempt %d differs from native: got %v, want %v", attempt, got, want.GetExplore())
				}
			}
		})
	}
}

func TestGuestProfileUncaughtPanicsTrap(t *testing.T) {
	for name, value := range map[string]byte{"observation": 14, "outcome comparison": 15} {
		t.Run(name, func(t *testing.T) {
			host := statefulHost(t)
			_, err := host.Judge(context.Background(), &modelv1.JudgeRequest{Observation: controlAny(value)})
			if err == nil || !strings.Contains(err.Error(), "METHOD_JUDGE: umpire_call:") {
				t.Fatalf("panic did not trap with method and ABI context: %v", err)
			}
			if !host.unusable {
				t.Fatal("panicking guest remained usable")
			}
			if _, err := host.Reset(context.Background(), &modelv1.ResetRequest{}); err == nil {
				t.Fatal("panicking instance accepted Reset")
			}
		})
	}
}
