package os_test

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
	_ "unsafe"

	"internal/gomadsim"
)

//go:linkname gomadObserveWrite os.gomadObserveWrite
func gomadObserveWrite(uint8, []byte, func([]byte) (int, error)) (int, error, bool)

func TestGomadObservedWriteRetainsOnlySuccessfulPrefix(t *testing.T) {
	run := gomadsim.Begin(64, 1)
	if run == 0 {
		t.Fatal("Begin returned token zero")
	}
	domain := gomadsim.Register(run, "node", "10.0.0.1", 1)
	if domain == 0 {
		t.Fatal("Register returned token zero")
	}
	previous, ok := gomadsim.Enter(domain)
	if !ok {
		t.Fatal("Enter rejected an active domain")
	}
	defer gomadsim.Leave(previous)

	source := []byte("partial")
	written, err, handled := gomadObserveWrite(gomadsim.OutputStdout, source, func([]byte) (int, error) {
		return 3, io.ErrShortWrite
	})
	if written != 3 || !errors.Is(err, io.ErrShortWrite) || !handled {
		t.Fatalf("observed write = (%d, %v, %t), want (3, short write, true)", written, err, handled)
	}
	encoded, ok := gomadsim.Finish(run)
	if !ok {
		t.Fatal("Finish rejected an active run")
	}
	if count := binary.LittleEndian.Uint64(encoded[8:16]); count != 1 {
		t.Fatalf("output count = %d, want 1", count)
	}
	if total := binary.LittleEndian.Uint64(encoded[48:56]); total != 3 {
		t.Fatalf("total bytes = %d, want 3", total)
	}
	if retained := binary.LittleEndian.Uint64(encoded[40:48]); retained != 3 {
		t.Fatalf("retained bytes = %d, want 3", retained)
	}
}
