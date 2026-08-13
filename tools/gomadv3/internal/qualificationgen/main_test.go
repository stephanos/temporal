package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
)

func TestRunRequiresExplicitManifestAndWorkingDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run(nil, &stdout, &stderr,
		func(context.Context, qualificationset.Config) (qualificationset.SetReport, error) {
			t.Fatal("qualification set executed without explicit inputs")
			return qualificationset.SetReport{}, nil
		},
		func(string) (qualificationset.Manifest, error) {
			t.Fatal("qualification manifest loaded without an explicit path")
			return qualificationset.Manifest{}, nil
		},
	)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "manifest") || !strings.Contains(stderr.String(), "working-dir") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRunPassesExplicitQualificationSetPaths(t *testing.T) {
	var observed qualificationset.Config
	var stdout, stderr bytes.Buffer
	status := run([]string{
		"--manifest=qualification/core.json", "--gomad=.bin/gomad", "--working-dir=./consumer",
		"--artifacts=.toolchain/qualification", "--output=.toolchain/qualification-set.json",
	}, &stdout, &stderr, func(_ context.Context, config qualificationset.Config) (qualificationset.SetReport, error) {
		observed = config
		return qualificationset.SetReport{Name: "example-corpus", ExpectationsMet: true, Selected: 5, Completed: 5, Supported: 3, Unsupported: 2}, nil
	}, qualificationset.LoadManifest)
	if status != 0 || stderr.Len() != 0 || observed.ManifestPath != "qualification/core.json" || observed.GomadPath != ".bin/gomad" || observed.WorkingDir != "./consumer" || observed.ArtifactRoot != ".toolchain/qualification" || observed.OutputPath != ".toolchain/qualification-set.json" || !strings.Contains(stdout.String(), "expectations-met=true supported=3 unsupported=2") {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, observed, stdout.String(), stderr.String())
	}
}

func TestRunCheckOnlyValidatesTheManifest(t *testing.T) {
	var loaded string
	var stdout, stderr bytes.Buffer
	status := run([]string{"--check", "--manifest=qualification/core.json", "--working-dir=./consumer"}, &stdout, &stderr,
		func(context.Context, qualificationset.Config) (qualificationset.SetReport, error) {
			t.Fatal("qualification set was executed during validation")
			return qualificationset.SetReport{}, nil
		},
		func(path string) (qualificationset.Manifest, error) {
			loaded = path
			return qualificationset.Manifest{Name: "example-corpus", Suites: make([]qualificationset.Suite, 5)}, nil
		},
	)
	if status != 0 || loaded != "qualification/core.json" || stderr.Len() != 0 || stdout.String() == "" {
		t.Fatalf("status=%d loaded=%q stdout=%q stderr=%q", status, loaded, stdout.String(), stderr.String())
	}
}
