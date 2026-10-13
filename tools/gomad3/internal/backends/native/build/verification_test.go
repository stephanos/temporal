package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func TestVerifyPreparedPreservesEmbeddedManifest(t *testing.T) {
	key := strings.Repeat("7", 64)
	object := sourceLinkedObject(t, key)
	path := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(path, object, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest, err := ReadCapabilityManifest(path, target.ToolchainIdentity{
		GoVersion: "go1.27.1", BuildKey: key, TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}
	base := target.Prepared{
		Path: path, SHA256: string(record.HashBytes(object)), Size: uint64(len(object)), Compatibility: []record.CompatibilityPack{},
		GoVersion: "go1.27.1", BuildKey: key, TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH,
		CapabilityMode: target.CapabilityModeLinked, CapabilityManifest: manifest,
	}
	for _, test := range []struct {
		name   string
		change func(*target.Prepared)
		want   string
	}{
		{name: "unchanged"},
		{name: "missing manifest", change: func(prepared *target.Prepared) { prepared.CapabilityManifest = nil }, want: "missing linked manifest"},
		{name: "changed manifest", change: func(prepared *target.Prepared) { prepared.CapabilityManifest.Facts++ }, want: "embedded record changed after preparation"},
		{name: "changed payload", change: func(prepared *target.Prepared) { prepared.CapabilityManifest.Payload[0] ^= 1 }, want: "embedded record changed after preparation"},
		{name: "changed toolchain", change: func(prepared *target.Prepared) { prepared.BuildKey = strings.Repeat("8", 64) }, want: "target identity does not match"},
		{name: "missing compatibility", change: func(prepared *target.Prepared) { prepared.Compatibility = nil }, want: "compatibility pack identity is missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared := base.CloneBackend()
			if test.change != nil {
				test.change(&prepared)
			}
			err := VerifyPrepared(prepared)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("VerifyPrepared() = %v, want %q", err, test.want)
			}
		})
	}
}
