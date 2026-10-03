package deterministicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestManifestInterceptsBeforeHostDispatch requires each listed operation to be
// in the interception manifest with its hook, and the hook to consult the
// profile before it can reach the host implementation.
func TestManifestInterceptsBeforeHostDispatch(t *testing.T) {
	manifestContents, err := os.ReadFile(filepath.Join("..", "toolchain", "runtime", "overlay", "src", "cmd", "compile", "internal", "gomadintercept", "spec_go127.go"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(manifestContents)
	for _, test := range []struct {
		name, hooks, label, hookPrefix string
		functions                      []string
		// guarded reports whether the start of a hook's body keeps it
		// from its host implementation.
		guarded func(body string) bool
	}{
		{
			name: "filesystem profile operations", hooks: "os", label: "os.", hookPrefix: "gomadIntercept",
			functions: []string{"Hostname", "Mkdir", "MkdirAll", "Stat"},
			guarded:   func(body string) bool { return strings.Contains(body, "gomadIOEnabled") },
		},
		{
			name: "concrete TCP methods", hooks: "net", label: "TCPConn.", hookPrefix: "gomadInterceptTCPConn",
			functions: []string{"SetLinger", "SetKeepAlive", "SetKeepAlivePeriod", "SetKeepAliveConfig", "SetNoDelay", "MultipathTCP"},
			guarded: func(body string) bool {
				return strings.Contains(body, "gomadInterceptTCPConnOption") || strings.Contains(body, "gomadConnection(conn.fd)")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			hookContents, err := os.ReadFile(filepath.Join("..", "toolchain", "runtime", "overlay", "src", test.hooks, "gomad.go"))
			if err != nil {
				t.Fatal(err)
			}
			hooks := string(hookContents)
			for _, function := range test.functions {
				hook := test.hookPrefix + function
				if !strings.Contains(manifest, `Function: "`+function+`", Hook: "`+hook+`"`) {
					t.Errorf("%s%s is not in the interception manifest", test.label, function)
					continue
				}
				start := strings.Index(hooks, "func "+hook+"(")
				if start < 0 {
					t.Errorf("%s%s interception hook is missing", test.label, function)
					continue
				}
				end := min(start+500, len(hooks))
				if !test.guarded(hooks[start:end]) {
					t.Errorf("%s%s can reach its host implementation", test.label, function)
				}
			}
		})
	}
}
