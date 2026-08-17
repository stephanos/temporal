package parity

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestCurrentManifest(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != ManifestSchema {
		t.Fatalf("schema = %q, want %q", manifest.Schema, ManifestSchema)
	}
	if manifest.HarnessSchema != HarnessSpecSchema {
		t.Fatalf("harness schema = %q, want %q", manifest.HarnessSchema, HarnessSpecSchema)
	}
	wantCases := []string{
		"different-seed-diversity",
		"enumerate-crash-states",
		"file-directory-sync",
		"fixed-link-latency",
		"graceful-stop-vs-crash-connection",
		"independent-node-identity-lifecycle",
		"nemesis-partition-restart",
		"partial-crash-persistence",
		"partition-timeout-heal-reconnect",
		"rename-truncate-crash-dependencies",
		"restart-durable-and-volatile",
		"same-seed-equality",
		"two-node-request-response",
	}
	gotCases := make([]string, 0, len(manifest.Cases))
	for _, parityCase := range manifest.Cases {
		gotCases = append(gotCases, parityCase.ID)
		if parityCase.Status != StatusPlanned {
			t.Fatalf("case %q status = %q, want %q", parityCase.ID, parityCase.Status, StatusPlanned)
		}
		if parityCase.Disposition == DispositionReplaced && parityCase.Replacement == "" {
			t.Fatalf("case %q replaces v2 behavior without an explanation", parityCase.ID)
		}
	}
	if !slices.Equal(gotCases, wantCases) {
		t.Fatalf("case IDs = %q, want %q", gotCases, wantCases)
	}

	wantPrototypes := []Prototype{
		{
			ID:       "restart",
			CaseID:   "restart-durable-and-volatile",
			Package:  "./tools/gomadv3sim",
			Test:     "TestPrototypeRestart",
			Backend:  BackendInProcess,
			Fidelity: FidelitySimulationModel,
			Status:   StatusPrototype,
		},
		{
			ID:       "two-node-request-response",
			CaseID:   "two-node-request-response",
			Package:  "./tools/gomadv3sim",
			Test:     "TestPrototypeTwoNodeRequestResponse",
			Backend:  BackendInProcess,
			Fidelity: FidelitySimulationModel,
			Status:   StatusPrototype,
		},
	}
	if !reflect.DeepEqual(manifest.Prototypes, wantPrototypes) {
		t.Fatalf("prototypes = %#v, want %#v", manifest.Prototypes, wantPrototypes)
	}
}

func TestCurrentManifestIsCanonical(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(currentManifestBytes, encoded) || !bytes.Equal(CurrentBytes(), encoded) {
		t.Fatal("embedded manifest is not canonical JSON")
	}
	decoded, err := Decode(currentManifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Cases, manifest.Cases) || !reflect.DeepEqual(decoded.Prototypes, manifest.Prototypes) {
		t.Fatal("canonical round trip changed manifest")
	}
}

func TestDecodeManifestRejectsInvalidInput(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	encode := func(value Manifest) []byte {
		t.Helper()
		encoded, encodeErr := evidence.CanonicalJSON(value)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		return encoded
	}
	tests := map[string][]byte{
		"empty":          nil,
		"oversized":      bytes.Repeat([]byte{'x'}, MaximumManifestBytes+1),
		"unknown field":  bytes.Replace(CurrentBytes(), []byte{'{'}, []byte(`{"unknown":true,`), 1),
		"trailing input": append(append([]byte(nil), CurrentBytes()...), '\n'),
	}

	unsorted := manifest
	unsorted.Cases = append([]Case(nil), manifest.Cases...)
	unsorted.Cases[0], unsorted.Cases[1] = unsorted.Cases[1], unsorted.Cases[0]
	tests["unsorted cases"] = encode(unsorted)

	duplicate := manifest
	duplicate.Cases = append([]Case(nil), manifest.Cases...)
	duplicate.Cases[1] = duplicate.Cases[0]
	tests["duplicate cases"] = encode(duplicate)

	badStage := manifest
	badStage.Cases = append([]Case(nil), manifest.Cases...)
	badStage.Cases[0].Stage = "SIM-9"
	tests["invalid stage"] = encode(badStage)

	missingReplacement := manifest
	missingReplacement.Cases = append([]Case(nil), manifest.Cases...)
	for index := range missingReplacement.Cases {
		if missingReplacement.Cases[index].Disposition == DispositionReplaced {
			missingReplacement.Cases[index].Replacement = ""
			break
		}
	}
	tests["missing replacement"] = encode(missingReplacement)

	unsafeSource := manifest
	unsafeSource.Cases = append([]Case(nil), manifest.Cases...)
	unsafeSource.Cases[0].Sources = append([]SourceReference(nil), manifest.Cases[0].Sources...)
	unsafeSource.Cases[0].Sources[0].Path = "../outside.go"
	tests["unsafe source"] = encode(unsafeSource)

	hardIsolationInProcess := manifest
	hardIsolationInProcess.Cases = append([]Case(nil), manifest.Cases...)
	hardIsolationInProcess.Cases[0].Requirements = []Requirement{{
		Fidelity: FidelityHardIsolation,
		Backends: []Backend{BackendInProcess},
	}}
	tests["hard isolation in process"] = encode(hardIsolationInProcess)

	badDisposition := manifest
	badDisposition.Cases = append([]Case(nil), manifest.Cases...)
	badDisposition.Cases[0].Disposition = "copied"
	tests["invalid disposition"] = encode(badDisposition)

	badStatus := manifest
	badStatus.Cases = append([]Case(nil), manifest.Cases...)
	badStatus.Cases[0].Status = StatusPrototype
	tests["invalid case status"] = encode(badStatus)

	badBackend := manifest
	badBackend.Cases = append([]Case(nil), manifest.Cases...)
	badBackend.Cases[0].Requirements = []Requirement{{Fidelity: FidelitySimulationModel, Backends: []Backend{"host"}}}
	tests["invalid backend"] = encode(badBackend)

	badFidelity := manifest
	badFidelity.Cases = append([]Case(nil), manifest.Cases...)
	badFidelity.Cases[0].Requirements = []Requirement{{Fidelity: "best_effort", Backends: []Backend{BackendProcess}}}
	tests["invalid fidelity"] = encode(badFidelity)

	widenedPrototype := manifest
	widenedPrototype.Prototypes = append([]Prototype(nil), manifest.Prototypes...)
	widenedPrototype.Prototypes[1].Backend = BackendProcess
	widenedPrototype.Prototypes[1].Fidelity = FidelityHardIsolation
	tests["prototype widens case fidelity"] = encode(widenedPrototype)

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(data); err == nil {
				t.Fatal("Decode succeeded")
			}
		})
	}
}

func TestManifestValidateRejectsInvalidUTF8(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	manifest.Cases = append([]Case(nil), manifest.Cases...)
	manifest.Cases[0].Contract = string([]byte{0xff})
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate succeeded")
	}
}

func TestManifestValidateRejectsUnboundedTestNames(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	manifest.Cases = append([]Case(nil), manifest.Cases...)
	manifest.Cases[0].Sources = append([]SourceReference(nil), manifest.Cases[0].Sources...)
	manifest.Cases[0].Sources[0].Tests = []string{"Test" + strings.Repeat("A", maximumTestNameBytes)}
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate accepted an unbounded source test name")
	}

	manifest, err = Current()
	if err != nil {
		t.Fatal(err)
	}
	manifest.Prototypes = append([]Prototype(nil), manifest.Prototypes...)
	manifest.Prototypes[0].Test = "Test" + strings.Repeat("A", maximumTestNameBytes)
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate accepted an unbounded prototype test name")
	}
}

func TestCurrentManifestV2SourcesExist(t *testing.T) {
	manifest, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory(t), "..", "..", "..", ".."))
	for _, parityCase := range manifest.Cases {
		for _, source := range parityCase.Sources {
			path := filepath.Join(repositoryRoot, filepath.FromSlash(source.Path))
			parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				t.Fatalf("case %q parse %q: %v", parityCase.ID, source.Path, parseErr)
			}
			functions := make(map[string]struct{})
			for _, declaration := range parsed.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if ok && function.Recv == nil {
					functions[function.Name.Name] = struct{}{}
				}
			}
			for _, testName := range source.Tests {
				if _, ok := functions[testName]; !ok {
					t.Fatalf("case %q source %q does not define %q", parityCase.ID, source.Path, testName)
				}
			}
		}
	}
}

func packageDirectory(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return directory
}
