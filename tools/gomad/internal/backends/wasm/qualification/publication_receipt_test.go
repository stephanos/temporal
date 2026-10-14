package qualification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gomadqualification "go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

func TestPublicationReceiptBindsTerminalObservation(t *testing.T) {
	result := Result{OperatorProcessID: 17, OperatorPreparationBefore: &hostexec.ResourceUsage{UserNanos: 2, SystemNanos: 3, PeakRSSBytes: 10}, OperatorExecutionAfter: &hostexec.ResourceUsage{UserNanos: 10, SystemNanos: 20, PeakRSSBytes: 30}}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	hash := record.HashBytes(append(encoded, '\n'))
	path := filepath.Join(t.TempDir(), "result-"+strings.TrimPrefix(string(hash), "sha256:")+".json")
	valid := PublicationReceipt{ResultSHA256: hash, OperatorProcessID: 17, OperatorAfter: &hostexec.ResourceUsage{UserNanos: 15, SystemNanos: 27, PeakRSSBytes: 40}}
	if delta, err := ValidatePublicationReceipt(path, result, valid); err != nil || delta != 12 {
		t.Fatalf("actual terminal tail rejected: delta=%d error=%v", delta, err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*PublicationReceipt)
	}{
		{"missing", func(r *PublicationReceipt) { *r = PublicationReceipt{} }},
		{"hash", func(r *PublicationReceipt) { r.ResultSHA256 = record.HashBytes([]byte("foreign")) }},
		{"pid", func(r *PublicationReceipt) { r.OperatorProcessID++ }},
		{"missing-observation", func(r *PublicationReceipt) { r.OperatorAfter = nil }},
		{"user-backward", func(r *PublicationReceipt) { r.OperatorAfter.UserNanos = 9 }},
		{"system-backward", func(r *PublicationReceipt) { r.OperatorAfter.SystemNanos = 19 }},
		{"rss-backward", func(r *PublicationReceipt) { r.OperatorAfter.PeakRSSBytes = 29 }},
		{"rss-missing", func(r *PublicationReceipt) { r.OperatorAfter.PeakRSSBytes = 0 }},
		{"over-budget", func(r *PublicationReceipt) { r.OperatorAfter.PeakRSSBytes = 2<<30 + 1 }},
		{"cpu-overflow", func(r *PublicationReceipt) {
			r.OperatorAfter.UserNanos = ^uint64(0)
			r.OperatorAfter.SystemNanos = ^uint64(0)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := valid
			usage := *valid.OperatorAfter
			copy.OperatorAfter = &usage
			test.mutate(&copy)
			if _, err := ValidatePublicationReceipt(path, result, copy); err == nil {
				t.Fatal("invalid terminal receipt accepted")
			}
		})
	}
	if _, err := ValidatePublicationReceipt(path+".changed", result, valid); err == nil {
		t.Fatal("changed published identity accepted")
	}
}

func TestObservedPublicationRetainsActualSelfAndStrictReceipt(t *testing.T) {
	start := hostexec.CurrentProcessResourceUsage()
	if start == nil {
		t.Skip("current process resource usage unavailable on this platform")
	}
	report, err := gomadqualification.BuildQualificationFailure([]string{"qualify"}, 7, 2, nil, gomadqualification.QualificationFailure{Classification: "runner_failure", Message: "not started", Iteration: 1})
	if err != nil {
		t.Fatal(err)
	}
	reportPath, err := gomadqualification.WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{Manifest: testManifest(), Qualification: report, QualificationBytes: encoded, CostFailure: "actual preparation and compiler measurements are required", OperatorProcessID: uint64(os.Getpid()), OperatorPreparationBefore: start, OperatorExecutionAfter: hostexec.CurrentProcessResourceUsage()}
	root := t.TempDir()
	var stages []string
	var lastObservation *hostexec.ResourceUsage
	path, observed, err := writeResultObserved(root, result, func(stage string) {
		stages = append(stages, stage)
		lastObservation = hostexec.CurrentProcessResourceUsage()
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(stages, ",") != "result.validate.before,result.validate.after,result.encode.before,result.encode.after,result.publish.before,result.publish.after,result.publication.ready" || lastObservation == nil || observed.OperatorAfter == nil || observed.OperatorAfter.UserNanos < lastObservation.UserNanos || observed.OperatorAfter.SystemNanos < lastObservation.SystemNanos || observed.OperatorAfter.PeakRSSBytes < lastObservation.PeakRSSBytes {
		t.Fatal("host-stage observation occurred after terminal receipt or lost publication boundaries")
	}
	publicPath, err := WriteResult(root, result)
	if err != nil || publicPath != path {
		t.Fatalf("host observations changed immutable public Result publication: %s %v", publicPath, err)
	}
	opened, err := OpenPublicationReceipt(path)
	if err != nil {
		t.Fatal(err)
	}
	if observed.ResultSHA256 != opened.ResultSHA256 || observed.OperatorAfter == nil || opened.OperatorAfter == nil || *observed.OperatorAfter != *opened.OperatorAfter {
		t.Fatal("actual terminal receipt changed")
	}
	if _, err := ValidatePublicationReceipt(path, result, opened); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`{"OperatorProcessID":1,"operatorprocessid":2}`, `{ "unknown":1 }`, `{} {}`, strings.Repeat(" ", publicationReceiptLimit+1)} {
		if err := os.WriteFile(path+".publication.json", []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenPublicationReceipt(path); err == nil {
			t.Fatal("hostile publication receipt accepted")
		}
	}
}
