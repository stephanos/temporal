package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func TestCompatibilityPackSourceRequiresDeclaredPlatformAndExplicitInputs(t *testing.T) {
	t.Parallel()
	compatibilityRoot, err := filepath.Abs("../../compatibilitypack")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		flags []string
	}{
		{"other platform", []string{"--source-platform=linux/amd64", "--source-go=/missing/go", "--source-inventory=/missing/inventory.json"}},
		{"relative Go", []string{"--source-platform=darwin/arm64", "--source-go=go", "--source-inventory=/missing/inventory.json"}},
		{"missing inventory", []string{"--source-platform=darwin/arm64", "--source-go=/missing/go"}},
		{"implicit platform", []string{"--source-go=/missing/go"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			arguments := []string{"--root=" + t.TempDir(), "--compatibility-root=" + compatibilityRoot, "--request=requests/temporal-functional-compute-darwin-arm64.json", "--working-dir=" + t.TempDir()}
			arguments = append(arguments, test.flags...)
			var stdout, stderr bytes.Buffer
			if status := runCompatibilityPackDiscover(arguments, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
				t.Fatalf("source admission = %d, stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestCompatibilityPackSourceInventoryRecordsNoExecutionQualification(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := publishCompatibilitySourceInventory(t.Context(), path, "darwin/arm64", target.CapabilityReview{Schema: target.CapabilityReviewSchema}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var inventory map[string]json.RawMessage
	if err := json.Unmarshal(contents, &inventory); err != nil {
		t.Fatal(err)
	}
	if string(inventory["native_execution_qualified"]) != "false" || string(inventory["platform"]) != `"darwin/arm64"` || string(inventory["schema"]) != `"gomad.compatibility-pack-source-review/v1"` || len(inventory["review"]) == 0 {
		t.Fatalf("source provenance = %s", contents)
	}
}
