package main

import (
	"bytes"
	"testing"
)

func TestByteSizeFlagParsesBinaryUnitsCanonically(t *testing.T) {
	for input, want := range map[string]uint64{"1": 1, "8KiB": 8 << 10, "8MiB": 8 << 20, "2GiB": 2 << 30} {
		var value byteSize
		if err := value.Set(input); err != nil {
			t.Fatalf("Set(%q): %v", input, err)
		}
		if uint64(value) != want {
			t.Fatalf("Set(%q) = %d, want %d", input, value, want)
		}
	}
	for _, input := range []string{"", "0", "1MB", "-1", "01", "18446744073709551615GiB"} {
		var value byteSize
		if err := value.Set(input); err == nil {
			t.Fatalf("Set(%q) succeeded", input)
		}
	}
}

func TestRunRejectsUnknownCommandWithUsageStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"unknown"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestParseTargetPreservesArgumentVector(t *testing.T) {
	spec, err := parseTarget([]string{"go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.source != "./pkg" || len(spec.arguments) != 2 || spec.arguments[0] != "-test.run=Test Name" || spec.arguments[1] != "literal;$value" {
		t.Fatalf("target = %#v", spec)
	}
}
