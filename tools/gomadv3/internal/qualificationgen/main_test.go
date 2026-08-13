package main

import (
	"bytes"
	"context"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
)

func TestRunPassesExplicitQualificationSetPaths(t *testing.T) {
	var observed qualificationset.Config
	var stdout, stderr bytes.Buffer
	status := run([]string{
		"--manifest=qualification/temporal.json", "--gomad=.bin/gomad", "--working-dir=../..",
		"--artifacts=.toolchain/qualification", "--output=.toolchain/qualification-set.json",
	}, &stdout, &stderr, func(_ context.Context, config qualificationset.Config) (qualificationset.SetReport, error) {
		observed = config
		return qualificationset.SetReport{Name: "temporal-representative", Qualified: true, Selected: 5, Completed: 5}, nil
	}, qualificationset.LoadManifest)
	if status != 0 || stderr.Len() != 0 || observed.ManifestPath != "qualification/temporal.json" || observed.GomadPath != ".bin/gomad" || observed.WorkingDir != "../.." || observed.ArtifactRoot != ".toolchain/qualification" || observed.OutputPath != ".toolchain/qualification-set.json" || stdout.String() == "" {
		t.Fatalf("status=%d config=%#v stdout=%q stderr=%q", status, observed, stdout.String(), stderr.String())
	}
}

func TestRunCheckOnlyValidatesTheManifest(t *testing.T) {
	var loaded string
	var stdout, stderr bytes.Buffer
	status := run([]string{"--check", "--manifest=qualification/temporal.json"}, &stdout, &stderr,
		func(context.Context, qualificationset.Config) (qualificationset.SetReport, error) {
			t.Fatal("qualification set was executed during validation")
			return qualificationset.SetReport{}, nil
		},
		func(path string) (qualificationset.Manifest, error) {
			loaded = path
			return qualificationset.Manifest{Name: "temporal-representative", Suites: make([]qualificationset.Suite, 5)}, nil
		},
	)
	if status != 0 || loaded != "qualification/temporal.json" || stderr.Len() != 0 || stdout.String() == "" {
		t.Fatalf("status=%d loaded=%q stdout=%q stderr=%q", status, loaded, stdout.String(), stderr.String())
	}
}
