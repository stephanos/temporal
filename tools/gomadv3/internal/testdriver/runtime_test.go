package testdriver

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestValidateRandomContract(t *testing.T) {
	valid := strings.Repeat("0123456789abcdef 01234567\n", 8)
	if err := validateRandomContract("1", valid); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{
		strings.Repeat("0123456789abcdef 01234567\n", 7),
		strings.Repeat("0123456789abcdef invalid\n", 8),
	} {
		if err := validateRandomContract("1", output); err == nil {
			t.Fatalf("validateRandomContract(%q) succeeded", output)
		}
	}
}

func TestValidateClockRaceOutput(t *testing.T) {
	if err := validateClockRaceOutput("[3 1 0 2]\n", 4); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"[3 1 0 1]\n", "[3 1 0]\n", "[3 1 nope 2]\n"} {
		if err := validateClockRaceOutput(output, 4); err == nil {
			t.Fatalf("validateClockRaceOutput(%q) succeeded", output)
		}
	}
}

func TestBenchmarkMedianNS(t *testing.T) {
	var output strings.Builder
	for value := 1; value <= 14; value++ {
		output.WriteString("BenchmarkDisabledClockNow-8 100 ")
		output.WriteString(string(rune('0' + value/10)))
		output.WriteString(string(rune('0' + value%10)))
		output.WriteString(" ns/op\n")
	}
	median, err := benchmarkMedianNS(output.String())
	if err != nil {
		t.Fatal(err)
	}
	if median != 7.5 {
		t.Fatalf("benchmarkMedianNS() = %v", median)
	}
	if _, err := benchmarkMedianNS("BenchmarkDisabledClockNow-8 100 1 ns/op\n"); err == nil {
		t.Fatal("benchmarkMedianNS() accepted an incomplete sample")
	}
}

func TestRepeatabilityMismatchRetainsDivergentEvidence(t *testing.T) {
	report := Report{Cases: []CaseResult{{Name: "actual", Passed: true, Stdout: []byte("actual\n")}}}
	campaign := runtimeCampaign{report: &report}
	err := campaign.repeatabilityMismatch("same-seed sync output diverged", "expected", "actual")
	expectedDigest := sha256.Sum256([]byte("expected"))
	actualDigest := sha256.Sum256([]byte("actual"))
	for _, want := range []string{
		"same-seed sync output diverged",
		fmt.Sprintf("expected sha256:%x", expectedDigest),
		fmt.Sprintf("actual sha256:%x", actualDigest),
		`expected-output="expected"`,
		`actual-output="actual"`,
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("repeatabilityMismatch() error = %v, want %q", err, want)
		}
	}
	if report.Cases[0].Passed {
		t.Fatal("divergent case remained passed")
	}
}
