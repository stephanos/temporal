package adapterregen

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

func TestRegenerationLockReleaseComposition(t *testing.T) {
	input := &InputError{Err: errors.New("invalid input")}
	blocked := &BlockedError{Err: errors.New("staged output failed")}
	releaseErr := errors.New("injected release failure")
	for _, test := range []struct {
		name    string
		primary error
		result  *publication
	}{
		{name: "recover success"},
		{name: "recover input", primary: input},
		{name: "recover blocked", primary: blocked},
		{name: "apply before stage", result: &publication{}},
		{name: "apply primary", primary: blocked, result: &publication{}},
		{name: "stage only", result: &publication{staged: []StagedFile{{Path: "adapter.go", Change: "changed"}}}},
		{name: "completed publication", result: &publication{staged: []StagedFile{{Path: "adapter.go", Change: "changed"}}, published: []string{"adapter.go"}, residual: []string{"leftover"}, warnings: []string{"scan warning", "stage warning"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, fault := range []bool{false, true} {
				var before publication
				if test.result != nil {
					before = *test.result
					before.warnings = append([]string(nil), before.warnings...)
				}
				calls := 0
				err := releaseRegenerationLock(test.result, test.primary, func() error {
					calls++
					if fault {
						return releaseErr
					}
					return nil
				})
				if calls != 1 {
					t.Fatalf("release calls=%d, want exactly1", calls)
				}
				published := test.result != nil && test.result.published != nil
				if !fault || published {
					if err != test.primary {
						t.Fatalf("primary identity changed: got %v want %v", err, test.primary)
					}
				} else if test.primary == nil {
					if err != releaseErr {
						t.Fatalf("sole release failure discarded or wrapped: %v", err)
					}
				} else {
					joined, ok := err.(interface{ Unwrap() []error })
					if !ok || !reflect.DeepEqual(joined.Unwrap(), []error{test.primary, releaseErr}) {
						t.Fatalf("release causes=%v, want exact primary then release", err)
					}
					var inputCause *InputError
					var blockedCause *BlockedError
					if errors.As(err, &inputCause) != (test.primary == input) || errors.As(err, &blockedCause) != (test.primary == blocked) {
						t.Fatalf("primary classification changed: %v", err)
					}
				}
				if test.result != nil {
					if !reflect.DeepEqual(test.result.staged, before.staged) || !reflect.DeepEqual(test.result.published, before.published) || !reflect.DeepEqual(test.result.residual, before.residual) {
						t.Fatalf("release changed publication: %+v -> %+v", before, *test.result)
					}
					if published && fault {
						warnings := test.result.warnings
						if len(warnings) != len(before.warnings)+1 || !reflect.DeepEqual(warnings[:len(before.warnings)], before.warnings) || !strings.Contains(warnings[len(before.warnings)], releaseErr.Error()) {
							t.Fatalf("publication warnings=%v", warnings)
						}
					} else if !reflect.DeepEqual(test.result.warnings, before.warnings) {
						t.Fatalf("unexpected warnings=%v", test.result.warnings)
					}
				}
			}
		})
	}
}

func TestPublicRegenerationReleasesLock(t *testing.T) {
	for _, mode := range []string{"stage", "apply", "failed apply", "recover"} {
		t.Run(mode, func(t *testing.T) {
			fixture := newFixture(t)
			before := fixture.snapshot()
			review := fixture.dryRun(goodVersion)
			spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
			primary := &BlockedError{Err: errors.New("stage refused")}
			spec.StageOnly = mode == "stage"
			if mode == "failed apply" {
				spec.afterStage = func() error { return primary }
			}
			if mode == "recover" {
				if err := Recover(fixture.root); err != nil {
					t.Fatal(err)
				}
			} else {
				result, err := Run(t.Context(), spec)
				if mode == "failed apply" {
					if err != primary || result.Applied || result.Published != nil {
						t.Fatalf("failed apply result=%+v error=%v, want original primary", result, err)
					}
				} else if err != nil || result.Applied != (mode == "apply") || len(result.Staged) == 0 || len(result.Warnings) != 0 {
					t.Fatalf("%s result=%+v error=%v", mode, result, err)
				}
			}
			if mode != "apply" {
				requireUnchanged(t, before, fixture.snapshot())
			}
			lock, err := hostfs.Try(filepath.Join(fixture.root, filepath.FromSlash(stateDirectory), "lock"))
			if err != nil {
				t.Fatalf("lock not released after %s: %v", mode, err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
			if err := Recover(fixture.root); err != nil {
				t.Fatalf("repeat recovery after %s: %v", mode, err)
			}
		})
	}
}
