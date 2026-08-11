package ioprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemPatchInterceptsProfileOperationsBeforeHostDispatch(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "go1.26.4.patch"))
	if err != nil {
		t.Fatal(err)
	}
	patch := string(contents)
	for _, function := range []string{"Hostname", "Mkdir", "MkdirAll", "Stat"} {
		start := strings.Index(patch, "func "+function+"(")
		if start < 0 {
			t.Errorf("os.%s is not intercepted", function)
			continue
		}
		end := min(start+500, len(patch))
		if !strings.Contains(patch[start:end], "gomadIOEnabled") {
			t.Errorf("os.%s can reach its host implementation", function)
		}
	}
}
