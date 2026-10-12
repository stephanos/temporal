package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestPreparedRecordProjectionDefensivelyCopiesIdentity(t *testing.T) {
	prepared := Prepared{
		Kind: KindGoTest, Source: "./pkg", SHA256: string(record.HashBytes([]byte("target"))), Size: 6,
		Argv: []string{"target", "argument"}, BuildTags: []string{"tag"},
		Adapters: []record.TargetAdapter{{Module: "adapter"}}, Compatibility: []record.CompatibilityPack{{ID: "pack"}},
		BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target", Settings: []record.BuildSetting{{Key: "setting", Value: "value"}}},
		GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}
	targetRecord := prepared.RecordTarget()
	toolchainRecord := prepared.RecordToolchain()
	prepared.Argv[0] = "changed"
	prepared.BuildTags[0] = "changed"
	prepared.Adapters[0].Module = "changed"
	prepared.Compatibility[0].ID = "changed"
	prepared.BuildInfo.Settings[0].Value = "changed"
	if targetRecord.Kind != "go-test" || targetRecord.Argv[0] != "target" || targetRecord.BuildTags[0] != "tag" || targetRecord.Adapters[0].Module != "adapter" || targetRecord.Compatibility[0].ID != "pack" || targetRecord.BuildInfo.Settings[0].Value != "value" {
		t.Fatalf("RecordTarget() = %#v", targetRecord)
	}
	if toolchainRecord.GoVersion != "go1.26.4" || toolchainRecord.BuildKey != strings.Repeat("a", 64) || toolchainRecord.TargetGOOS != "darwin" || toolchainRecord.TargetGOARCH != "arm64" {
		t.Fatalf("RecordToolchain() = %#v", toolchainRecord)
	}
}

func TestPreparedVerifyCommonFileIdentity(t *testing.T) {
	data := []byte("prepared executable")
	for _, test := range []struct {
		name   string
		change func(*testing.T, *Prepared)
		want   string
	}{
		{name: "unchanged"},
		{name: "changed bytes", change: func(t *testing.T, prepared *Prepared) {
			if err := os.WriteFile(prepared.Path, []byte("changed executable"), 0o700); err != nil {
				t.Fatal(err)
			}
		}, want: "changed after preparation"},
		{name: "changed size", change: func(_ *testing.T, prepared *Prepared) { prepared.Size++ }, want: "changed after preparation"},
		{name: "non-executable", change: func(t *testing.T, prepared *Prepared) {
			if err := os.Chmod(prepared.Path, 0o600); err != nil {
				t.Fatal(err)
			}
		}, want: "not a regular executable"},
		{name: "symlink", change: func(t *testing.T, prepared *Prepared) {
			link := prepared.Path + ".link"
			if err := os.Symlink(prepared.Path, link); err != nil {
				t.Fatal(err)
			}
			prepared.Path = link
		}, want: "not a regular executable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared := Prepared{Path: filepath.Join(t.TempDir(), "target"), SHA256: string(record.HashBytes(data)), Size: uint64(len(data))}
			if err := os.WriteFile(prepared.Path, data, 0o700); err != nil {
				t.Fatal(err)
			}
			if test.change != nil {
				test.change(t, &prepared)
			}
			err := prepared.Verify()
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("Verify() = %v, want %q", err, test.want)
			}
		})
	}
}
