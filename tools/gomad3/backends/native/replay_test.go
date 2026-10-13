package native

import (
	"runtime/debug"
	"strings"
	"testing"

	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestReplayBuildInfoRejectsMatchingCoverageInstrumentation(t *testing.T) {
	info := &debug.BuildInfo{GoVersion: "go1.27.1", Path: "example.com/target", Settings: []debug.BuildSetting{{Key: "-cover", Value: "true"}}}
	err := validateBuildInfo(info, nativebuild.ProjectBuildInfo(info))
	if err == nil || !strings.Contains(err.Error(), "coverage") {
		t.Fatalf("matching coverage build-info rejection = %v", err)
	}
}

func TestReplayBuildInfoRequiresExactSemanticIdentity(t *testing.T) {
	info := &debug.BuildInfo{
		GoVersion: "go1.27.1",
		Path:      "example.com/target",
		Main:      debug.Module{Path: "example.com/target", Version: "v1.2.3"},
		Settings:  []debug.BuildSetting{{Key: "GOARCH", Value: "arm64"}},
	}
	expected := nativebuild.ProjectBuildInfo(info)
	if err := validateBuildInfo(info, expected); err != nil {
		t.Fatalf("exact build info rejected: %v", err)
	}
	for name, changed := range map[string]record.BuildInfo{
		"scalar":  {GoVersion: "go1.27.2", Path: expected.Path, MainModule: expected.MainModule, Settings: expected.Settings},
		"setting": {GoVersion: expected.GoVersion, Path: expected.Path, MainModule: expected.MainModule, Settings: []record.BuildSetting{{Key: "GOARCH", Value: "amd64"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateBuildInfo(info, changed); err == nil {
				t.Fatal("mismatched build info admitted")
			}
		})
	}

	emptyInfo := &debug.BuildInfo{GoVersion: "go1.27.1", Path: "example.com/target"}
	emptyExpected := nativebuild.ProjectBuildInfo(emptyInfo)
	emptyExpected.Settings = nil
	if err := validateBuildInfo(emptyInfo, emptyExpected); err == nil {
		t.Fatal("nil settings admitted as equivalent to an empty settings array")
	}
}
