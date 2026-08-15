package deterministicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkManifestInterceptsConcreteTCPMethodsBeforeHostDispatch(t *testing.T) {
	manifestContents, err := os.ReadFile(filepath.Join("..", "toolchain", "runtime", "overlay", "src", "cmd", "compile", "internal", "gomadintercept", "spec_go126.go"))
	if err != nil {
		t.Fatal(err)
	}
	hookContents, err := os.ReadFile(filepath.Join("..", "toolchain", "runtime", "overlay", "src", "net", "gomad.go"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(manifestContents)
	hooks := string(hookContents)
	for _, method := range []string{"SetLinger", "SetKeepAlive", "SetKeepAlivePeriod", "SetKeepAliveConfig", "SetNoDelay", "MultipathTCP"} {
		hook := "gomadInterceptTCPConn" + method
		if !strings.Contains(manifest, `Function: "`+method+`", Hook: "`+hook+`"`) {
			t.Errorf("TCPConn.%s is not in the interception manifest", method)
			continue
		}
		start := strings.Index(hooks, "func "+hook+"(")
		if start < 0 {
			t.Errorf("TCPConn.%s interception hook is missing", method)
			continue
		}
		end := min(start+500, len(hooks))
		if !strings.Contains(hooks[start:end], "gomadInterceptTCPConnOption") && !strings.Contains(hooks[start:end], "gomadConnection(conn.fd)") {
			t.Errorf("TCPConn.%s can reach its host implementation", method)
		}
	}
}
