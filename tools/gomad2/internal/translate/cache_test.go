package translate

import (
	"testing"

	"github.com/temporalio/gomad/internal/gomadtool"
)

func TestTranslateToolHashIncludesUserTags(t *testing.T) {
	base := gomadtool.BuildConfig{
		GOOS:   "linux",
		GOARCH: "amd64",
	}
	tagged := base
	tagged.UserTags = gomadtool.ParseBuildTags("test_dep")

	if computeTranslateToolHash(base) == computeTranslateToolHash(tagged) {
		t.Fatal("computeTranslateToolHash() ignored user build tags")
	}
	if base.AsDirname() != tagged.AsDirname() {
		t.Fatal("user build tags changed the stable artifact directory")
	}
}
