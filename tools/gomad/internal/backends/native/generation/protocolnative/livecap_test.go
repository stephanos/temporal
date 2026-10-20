package protocolnative

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLiveCapabilitySchemaPinsClosedProtocol(t *testing.T) {
	definition, err := readLiveCapabilitySchema(filepath.Join("..", "..", "..", "..", "..", "internal", "backends", "native", "build", "internal", "livecap", "livecap.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 2 || definition.Schema != "gomad.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" {
		t.Fatalf("live capability identity = version %d schema %q symbol %q", definition.Version, definition.Schema, definition.Symbol)
	}
	if definition.Header.Magic != "GOMADCAPABILITY\x00" || definition.Header.Bytes != 112 {
		t.Fatalf("live capability header = magic %q bytes %d", definition.Header.Magic, definition.Header.Bytes)
	}
	if definition.Limits.PayloadBytes != 16<<20 || definition.Limits.Facts != 100_000 || definition.Limits.StringBytes != 4<<10 || definition.Limits.OwnerFacts != 4_096 {
		t.Fatalf("live capability limits = %+v", definition.Limits)
	}
	wantKinds := []string{"boundary", "capability", "foreign", "guard", "linkname"}
	if strings.Join(definition.FactKinds, ",") != strings.Join(wantKinds, ",") {
		t.Fatalf("live capability fact kinds = %v, want %v", definition.FactKinds, wantKinds)
	}
	wantDispositions := []string{"denied", "guarded", "modeled", "pack"}
	if strings.Join(definition.Dispositions, ",") != strings.Join(wantDispositions, ",") {
		t.Fatalf("live capability dispositions = %v, want %v", definition.Dispositions, wantDispositions)
	}
	if strings.Join(definition.GuardExemptions, ",") != "syscall.Clearenv,syscall.Environ,syscall.Errno.Error,syscall.Errno.Is,syscall.Errno.Temporary,syscall.Errno.Timeout,syscall.Getenv,syscall.Setenv,syscall.Unsetenv,syscall.Write" {
		t.Fatalf("live capability guard exemptions = %v", definition.GuardExemptions)
	}
	wantForbidden := []string{"os/exec", "os/signal", "os/user", "plugin", "runtime/cgo", "syscall"}
	if strings.Join(definition.ForbiddenImports, ",") != strings.Join(wantForbidden, ",") || strings.Join(definition.ForbiddenPrefixes, ",") != "golang.org/x/sys" {
		t.Fatalf("live capability forbidden vocabulary = imports %v prefixes %v", definition.ForbiddenImports, definition.ForbiddenPrefixes)
	}
}

func TestLiveCapabilityUniverseIdentityBindsBoundaryAndForbiddenVocabulary(t *testing.T) {
	definition := liveCapabilitySchema{
		Version: 1, Schema: "gomad.live-capability-manifest/v1",
		ForbiddenImports: []string{"syscall"}, ForbiddenPrefixes: []string{"golang.org/x/sys"},
	}
	want, err := liveCapabilityUniverseIdentity(definition, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	changed := definition
	changed.ForbiddenImports = []string{"os/exec", "syscall"}
	got, err := liveCapabilityUniverseIdentity(changed, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted forbidden imports")
	}
	changed = definition
	changed.GuardExemptions = []string{"syscall.rsaAlignOf"}
	got, err = liveCapabilityUniverseIdentity(changed, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted guard exemptions")
	}
	got, err = liveCapabilityUniverseIdentity(definition, "sha256:"+strings.Repeat("2", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted boundary manifest")
	}
}

func TestLiveCapabilityImplementationIdentityBindsProducerAndValidatorInputs(t *testing.T) {
	inputs := liveCapabilityImplementationInputs{
		Schema: []byte("schema"), CodecTemplate: []byte("codec"), CompilerEmitter: []byte("compiler"), LinkerProjector: []byte("linker"),
		Encoder: []byte("encoder"), GuardFlag: []byte("guard-flag"), GuardSource: []byte("guard-source"), RuntimeGuard: []byte("runtime-guard"),
		InterceptionSource: []byte("interception"), BoundaryTable: []byte("boundary"), HostValidator: []byte("validator"), ProjectionContract: []byte("projection"),
	}
	want := liveCapabilityImplementationIdentity(inputs)
	if want == ([32]byte{}) {
		t.Fatal("live capability implementation identity is empty")
	}
	mutations := [][]byte{
		inputs.Schema, inputs.CodecTemplate, inputs.CompilerEmitter, inputs.LinkerProjector, inputs.Encoder,
		inputs.GuardFlag, inputs.GuardSource, inputs.RuntimeGuard, inputs.InterceptionSource, inputs.BoundaryTable, inputs.HostValidator, inputs.ProjectionContract,
	}
	for index, original := range mutations {
		changed := inputs
		value := append(bytes.Clone(original), '!')
		switch index {
		case 0:
			changed.Schema = value
		case 1:
			changed.CodecTemplate = value
		case 2:
			changed.CompilerEmitter = value
		case 3:
			changed.LinkerProjector = value
		case 4:
			changed.Encoder = value
		case 5:
			changed.GuardFlag = value
		case 6:
			changed.GuardSource = value
		case 7:
			changed.RuntimeGuard = value
		case 8:
			changed.InterceptionSource = value
		case 9:
			changed.BoundaryTable = value
		case 10:
			changed.HostValidator = value
		case 11:
			changed.ProjectionContract = value
		}
		if got := liveCapabilityImplementationIdentity(changed); got == want {
			t.Fatalf("identity did not bind input %d", index)
		}
	}
}
