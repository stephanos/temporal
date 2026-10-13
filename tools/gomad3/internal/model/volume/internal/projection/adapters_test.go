package projection

import (
	"bytes"
	"os"
	"testing"
)

func TestOwnerAdaptationRejectsCallbackAndAliasDrift(t *testing.T) {
	input, err := os.ReadFile("../../../../../" + overlay + "gomadfs/fs.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string][]byte{
		"clock_signature":  bytes.Replace(input, []byte("SetClock(clock func() int64)"), []byte("SetClock(clock func() uint64)"), 1),
		"clock_call":       bytes.Replace(input, []byte("fs.clock()"), []byte("fs.clock(1)"), 1),
		"loader_signature": bytes.Replace(input, []byte("type Loader func(string)"), []byte("type Loader func(string, int)"), 1),
		"loader_call":      bytes.Replace(input, []byte("fs.loader(path)"), []byte("fs.loader(path + \"\")"), 1),
		"public_alias":     bytes.Replace(input, []byte("type Kind = gomadwire.MountKind"), []byte("type Kind gomadwire.MountKind"), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := rewrite(mutation, "volume", true, false); err == nil {
				t.Fatal("owner adapter drift was accepted")
			}
		})
	}
	volume, err := os.ReadFile("../../../../../" + overlay + "gomadfs/volume.go")
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(volume, []byte("BeforeVolumeControl(string, string, []uint64) error"), []byte("BeforeVolumeControl(string, string) error"), 1)
	if _, err := rewrite(changed, "volume", false, true); err == nil {
		t.Fatal("observer interface drift was accepted")
	}
	guard := bytes.Replace(volume, []byte("now >= started && uint64(now-started) >= limits.WallNanos"), []byte("uint64(now-started) >= limits.WallNanos"), 1)
	if _, err := rewrite(guard, "volume", false, true); err == nil {
		t.Fatal("enumeration guard drift was accepted")
	}
}

func TestNamedMirrorsComeFromAuthoritativeWireTypes(t *testing.T) {
	input := []byte("package wire\ntype MountKind uint8\ntype MountStatus uint16\ntype MountChild struct { Name string; Kind MountKind; Mode uint32 }\ntype MountEntry struct { Kind MountKind; Mode uint32; Data []byte; Children []MountChild }\n")
	output, err := mirrorTypes(input, "wire.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, witness := range []string{"type Kind uint8", "type MountStatus uint16", "type Child struct", "type LoadEntry struct", "Kind Kind", "Children []Child"} {
		// gofmt aligns struct fields, so compare token spellings independently of spacing.
		compact := bytes.Join(bytes.Fields(output), []byte(" "))
		if !bytes.Contains(compact, []byte(witness)) {
			t.Fatalf("authoritative mirror missing %s in %s", witness, output)
		}
	}
	duplicate := append(append([]byte(nil), input...), []byte("type MountKind uint16\n")...)
	if _, err := mirrorTypes(duplicate, "wire.go"); err == nil {
		t.Fatal("duplicate authoritative type accepted")
	}
	alias := bytes.Replace(input, []byte("type MountKind uint8"), []byte("type MountKind = uint8"), 1)
	if _, err := mirrorTypes(alias, "wire.go"); err == nil {
		t.Fatal("authoritative alias accepted")
	}
}
