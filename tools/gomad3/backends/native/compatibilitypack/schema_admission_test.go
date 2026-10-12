package compatibility

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

func TestPackAdmissionRejectsFiveHostImports(t *testing.T) {
	for _, test := range []struct{ capability, want string }{
		{"import:os/exec", "compatibility pack rule 0: capability import:os/exec is never admitted"},
		{"import:os/signal", "compatibility pack rule 0: capability import:os/signal is never admitted"},
		{"import:os/user", "compatibility pack rule 0: capability import:os/user is never admitted"},
		{"import:plugin", "compatibility pack rule 0: capability import:plugin is never admitted"},
		{"import:runtime/cgo", "compatibility pack rule 0: capability import:runtime/cgo is never admitted"},
	} {
		t.Run(test.capability, func(t *testing.T) {
			pack := admissionTestPack(t)
			pack.Rules[0].Capabilities = []string{test.capability}
			requireTestNoError(t, ValidatePackStructure(pack))
			requireAdmissionError(t, ValidatePack(pack), test.want)
			encoded, err := canonicaljson.CanonicalJSON(pack)
			requireTestNoError(t, err)
			decoded, err := DecodePack(encoded)
			requireAdmissionError(t, err, test.want)
			requireTestEqual(t, Pack{}, decoded)
			loaded, err := LoadPack(encoded)
			requireAdmissionError(t, err, test.want)
			requireTestEqual(t, ValidatedPack{}, loaded)
		})
	}
}

func TestPackAdmissionPreservesStructuralAndRulePriority(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Pack)
		want   string
	}{
		{"later-invalid-inventory", func(pack *Pack) {
			later := pack.Rules[0]
			later.ImportPath = "example.com/dependency/zzz"
			later.GoSources = nil
			pack.Rules[0].Capabilities = []string{"import:plugin"}
			pack.Rules = append(pack.Rules, later)
		}, "compatibility pack rule 1: source inventory count is invalid"},
		{"later-prohibited-rule", func(pack *Pack) {
			later := pack.Rules[0]
			later.ImportPath = "example.com/dependency/zzz"
			later.Capabilities = []string{"import:runtime/cgo"}
			pack.Rules = append(pack.Rules, later)
		}, "compatibility pack rule 1: capability import:runtime/cgo is never admitted"},
		{"first-prohibited-rule", func(pack *Pack) {
			later := pack.Rules[0]
			later.ImportPath = "example.com/dependency/zzz"
			later.Capabilities = []string{"import:plugin"}
			pack.Rules[0].Capabilities = []string{"import:runtime/cgo"}
			pack.Rules = append(pack.Rules, later)
		}, "compatibility pack rule 0: capability import:runtime/cgo is never admitted"},
		{"capability-order", func(pack *Pack) {
			pack.Rules[0].Capabilities = []string{"import:plugin", "import:runtime/cgo"}
		}, "compatibility pack rule 0: capability import:plugin is never admitted"},
		{"existing-ban-before-new-ban", func(pack *Pack) {
			pack.Rules[0].Capabilities = []string{"import:os/user", "import:plugin"}
		}, "compatibility pack rule 0: capability import:os/user is never admitted"},
		{"unsorted-capabilities", func(pack *Pack) {
			pack.Rules[0].Capabilities = []string{"import:runtime/cgo", "import:plugin"}
		}, "compatibility pack rule 0: capability inventory is not canonical"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pack := admissionTestPack(t)
			test.change(&pack)
			requireAdmissionError(t, ValidatePack(pack), test.want)
			encoded, err := canonicaljson.CanonicalJSON(pack)
			requireTestNoError(t, err)
			_, err = DecodePack(encoded)
			requireAdmissionError(t, err, test.want)
			_, err = LoadPack(encoded)
			requireAdmissionError(t, err, test.want)
		})
	}
}

func TestPackAdmissionRevalidatesUnselectedTokens(t *testing.T) {
	for _, test := range []struct{ capability, want string }{
		{"import:os/exec", "compatibility pack rule 0: capability import:os/exec is never admitted"},
		{"import:os/signal", "compatibility pack rule 0: capability import:os/signal is never admitted"},
		{"import:os/user", "compatibility pack rule 0: capability import:os/user is never admitted"},
		{"import:plugin", "compatibility pack rule 0: capability import:plugin is never admitted"},
		{"import:runtime/cgo", "compatibility pack rule 0: capability import:runtime/cgo is never admitted"},
	} {
		t.Run(test.capability, func(t *testing.T) {
			pack := admissionTestPack(t)
			pack.Rules[0].Capabilities = []string{test.capability}
			forged := ValidatedPack{pack: pack, digest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}
			selection, err := SelectPacksForPlatform([]ValidatedPack{forged}, nil, "linux/amd64")
			requireAdmissionError(t, err, test.want)
			requireTestEqual(t, Selection{}, selection)
			requireAdmissionError(t, VerifyPackIdentities([]ValidatedPack{forged}, nil), test.want)
		})
	}
}

func TestPackAdmissionPreservesEarlierPackErrors(t *testing.T) {
	valid := ValidatedPack{pack: admissionTestPack(t), digest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	valid.pack.ID = "a-pack"
	invalid := ValidatedPack{pack: admissionTestPack(t), digest: valid.digest}
	invalid.pack.ID = "z-pack"
	invalid.pack.Rules[0].Capabilities = []string{"import:plugin"}
	for _, test := range []struct {
		name  string
		packs []ValidatedPack
		want  string
	}{
		{"digest", []ValidatedPack{{pack: valid.pack}, invalid}, "validated compatibility pack has no digest"},
		{"duplicate", []ValidatedPack{valid, valid, invalid}, "compatibility pack ID is duplicated: a-pack"},
	} {
		t.Run(test.name, func(t *testing.T) {
			requireAdmissionError(t, VerifyPackIdentities(test.packs, nil), test.want)
			ordered := append([]ValidatedPack{invalid}, test.packs[:len(test.packs)-1]...)
			selection, err := SelectPacksForPlatform(ordered, nil, "linux/amd64")
			requireAdmissionError(t, err, test.want)
			requireTestEqual(t, Selection{}, selection)
		})
	}
}

func TestPackAdmissionPreservesOtherCapabilities(t *testing.T) {
	for _, capability := range []string{"import:syscall", "import:future/capability", "import:plugin/subpackage", "foreign:cgo:native.c"} {
		t.Run(capability, func(t *testing.T) {
			pack := admissionTestPack(t)
			pack.Rules[0].Capabilities = []string{capability}
			requireTestNoError(t, ValidatePack(pack))
			encoded, err := canonicaljson.CanonicalJSON(pack)
			requireTestNoError(t, err)
			loaded, err := LoadPack(encoded)
			requireTestNoError(t, err)
			requireTestEqual(t, pack, loaded.Pack())
		})
	}
}

func admissionTestPack(t *testing.T) Pack {
	t.Helper()
	pack, err := DecodePack([]byte(validPackV2))
	requireTestNoError(t, err)
	return pack
}

func requireAdmissionError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
