package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestSoakRejectsInvalidInvocationsAsInvalidInput(t *testing.T) {
	directory := t.TempDir()
	for name, arguments := range map[string][]string{
		"missing directories": {"--manifest=" + filepath.Join(directory, "soak.json")},
		"missing manifest": {
			"--manifest=" + filepath.Join(directory, "absent.json"), "--gomad=gomad",
			"--work=" + filepath.Join(directory, "work"), "--ledger=" + filepath.Join(directory, "ledger"), "--output=" + filepath.Join(directory, "output"),
		},
		"positional argument": {"extra"},
		"invalid seed":        {"--seed=eleven"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := run(append([]string{"soak"}, arguments...), &stdout, &stderr); status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status %d, stdout %q, stderr %q", status, stdout.String(), stderr.String())
			}
		})
	}
}
