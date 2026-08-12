package ioprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemManifestInterceptsProfileOperationsBeforeHostDispatch(t *testing.T) {
	manifestContents, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", "cmd", "compile", "internal", "gomadintercept", "spec_go126.go"))
	if err != nil {
		t.Fatal(err)
	}
	hookContents, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", "os", "gomad.go"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(manifestContents)
	hooks := string(hookContents)
	for _, function := range []string{"Hostname", "Mkdir", "MkdirAll", "Stat"} {
		hook := "gomadIntercept" + function
		if !strings.Contains(manifest, `Function: "`+function+`", Hook: "`+hook+`"`) {
			t.Errorf("os.%s is not in the interception manifest", function)
			continue
		}
		start := strings.Index(hooks, "func "+hook+"(")
		if start < 0 {
			t.Errorf("os.%s interception hook is missing", function)
			continue
		}
		end := min(start+500, len(hooks))
		if !strings.Contains(hooks[start:end], "gomadIOEnabled") {
			t.Errorf("os.%s can reach its host implementation", function)
		}
	}
}
