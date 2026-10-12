package native

import (
	"runtime/debug"
	"strings"
	"testing"

	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
)

func TestReplayBuildInfoRejectsMatchingCoverageInstrumentation(t *testing.T) {
	info := &debug.BuildInfo{GoVersion: "go1.27.1", Path: "example.com/target", Settings: []debug.BuildSetting{{Key: "-cover", Value: "true"}}}
	err := validateBuildInfo(info, nativebuild.ProjectBuildInfo(info))
	if err == nil || !strings.Contains(err.Error(), "coverage") {
		t.Fatalf("matching coverage build-info rejection = %v", err)
	}
}
