package projection

import (
	"bytes"
	"os"
	"testing"
)

func TestProjectionRejectsProcessBranchDrift(t *testing.T) {
	input, err := os.ReadFile("../../../../" + overlay + "gomadfs/fs.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string][]byte{
		"missing":  bytes.Replace(input, []byte("if fs.process {"), []byte("if false {"), 1),
		"guard":    bytes.Replace(input, []byte("if fs.process {"), []byte("if !fs.process {"), 1),
		"body":     bytes.Replace(input, []byte("return processResolve(name)"), []byte("return processResolve(\"changed\")"), 1),
		"receiver": bytes.Replace(input, []byte("func (fs *FS) Resolve"), []byte("func (fs *OtherFS) Resolve"), 1),
		"else":     bytes.Replace(input, []byte("return processResolve(name)\n\t}"), []byte("return processResolve(name)\n\t} else {}"), 1),
		"extra":    append(append([]byte(nil), input...), []byte("\nfunc (fs *FS) extra() { if fs.process {} }\n")...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := rewrite(mutation, "volume", true, false); err == nil {
				t.Fatal("process branch drift was accepted")
			}
		})
	}
}

func TestProjectionRemovesOnlyDispatchBranches(t *testing.T) {
	input, err := os.ReadFile("../../../../" + overlay + "gomadfs/fs.go")
	if err != nil {
		t.Fatal(err)
	}
	output, err := rewrite(input, "volume", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(output, []byte("if fs.process")) || bytes.Contains(output, []byte("processResolve")) {
		t.Fatal("process dispatch survived projection")
	}
	// The source equality gate checks the complete generated output; this witness
	// also ensures existing native comments and capacity bounds remain present.
	for _, witness := range [][]byte{[]byte("MaximumFileBytes   = 256 << 20"), []byte("A WAL-mode SQLite database"), []byte("fs.recordAllocationAndLinkLocked")} {
		if !bytes.Contains(output, witness) {
			t.Fatalf("native witness was removed: %s", witness)
		}
	}
}
