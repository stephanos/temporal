//go:build test_dep

package xsys

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/sha3"
	"golang.org/x/term"
)

// TestXSysCompatibilityClosure reaches golang.org/x/sys/unix through x/term and
// golang.org/x/sys/cpu through x/crypto/sha3, without the modernc libc modules:
// the closure an ordinary dependency outside SQLite presents to the standalone
// x/sys pack.
func TestXSysCompatibilityClosure(t *testing.T) {
	var screen bytes.Buffer
	if term.NewTerminal(&screen, "> ") == nil {
		t.Fatal("term.NewTerminal returned nil")
	}
	digest := sha3.Sum256([]byte("gomad"))
	if digest == ([32]byte{}) {
		t.Fatal("sha3.Sum256 returned a zero digest")
	}
}
