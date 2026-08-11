package ioprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkPatchInterceptsConcreteTCPMethodsBeforeHostDispatch(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "go1.26.4.patch"))
	if err != nil {
		t.Fatal(err)
	}
	patch := string(contents)
	for _, method := range []string{"SetLinger", "SetKeepAlive", "SetKeepAlivePeriod", "SetKeepAliveConfig", "SetNoDelay", "MultipathTCP"} {
		start := strings.Index(patch, "func (c *TCPConn) "+method)
		if start < 0 {
			t.Errorf("TCPConn.%s is not intercepted", method)
			continue
		}
		end := min(start+500, len(patch))
		if !strings.Contains(patch[start:end], "gomadConnection(c.fd)") {
			t.Errorf("TCPConn.%s can reach its host implementation", method)
		}
	}
}
