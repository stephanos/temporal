package qualification

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
)

func qualificationGuestBuildTags(profile string) ([]string, error) {
	tags := backend.TemporalGuestBuildTags()
	switch profile {
	case "", "slim":
		return tags, nil
	case "full":
		return slices.DeleteFunc(tags, func(tag string) bool { return tag == backend.SlimGuestBuildTag }), nil
	default:
		return nil, errors.New("qualification guest profile must be full or slim")
	}
}

func qualificationHelperPath() (string, error) {
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helper == "" {
		helper = "../wasmhost/target/release/gomad-wasmhost"
	}
	return filepath.Abs(helper)
}

func TestQualificationGuestProfilesBindOnlySlimSelection(t *testing.T) {
	for _, test := range []struct {
		profile string
		want    []string
	}{
		{"", []string{"disable_grpc_modules", "gomad_wasm_slim", "sqlite3_dotlk", "test_dep"}},
		{"slim", []string{"disable_grpc_modules", "gomad_wasm_slim", "sqlite3_dotlk", "test_dep"}},
		{"full", []string{"disable_grpc_modules", "sqlite3_dotlk", "test_dep"}},
	} {
		tags, err := qualificationGuestBuildTags(test.profile)
		if err != nil || !slices.Equal(tags, test.want) {
			t.Fatalf("profile=%q tags=%q want=%q err=%v", test.profile, tags, test.want, err)
		}
	}
	if _, err := qualificationGuestBuildTags("unknown"); err == nil {
		t.Fatal("unknown qualification guest selection was admitted")
	}
}
