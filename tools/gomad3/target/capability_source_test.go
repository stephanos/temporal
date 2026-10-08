package target

import (
	"bytes"
	"context"
	"debug/elf"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
)

func sourceLinkedObject(t *testing.T, key string) []byte {
	t.Helper()
	payload, err := canonicaljson.CanonicalJSON(livecap.Manifest{
		Schema: livecap.ManifestSchema, GoVersion: "go1.27.1", ToolchainBuildKey: key, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		ProducerImplementationSHA256: livecap.ProducerImplementationSHA256, GuardImplementationSHA256: livecap.GuardImplementationSHA256,
		CapabilityUniverseSHA256: livecap.CapabilityUniverseSHA256, Facts: []livecap.Fact{},
		Limits: livecap.Limits{Facts: livecap.MaximumFacts, OwnerFacts: livecap.MaximumOwnerFacts, PayloadBytes: livecap.MaximumPayloadBytes, StringBytes: livecap.MaximumStringBytes},
	})
	if err != nil {
		t.Fatal(err)
	}
	header := make([]byte, livecap.HeaderBytes)
	copy(header, livecap.HeaderMagic[:])
	binary.LittleEndian.PutUint32(header[16:], livecap.ProtocolVersion)
	binary.LittleEndian.PutUint32(header[20:], livecap.HeaderBytes)
	binary.LittleEndian.PutUint64(header[24:], uint64(len(payload)))
	digest, err := record.HashBytes(payload).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	copy(header[40:], digest[:])
	producer, err := record.ParseSHA256(livecap.ProducerImplementationSHA256)
	if err != nil {
		t.Fatal(err)
	}
	producerBytes, err := producer.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	copy(header[72:], producerBytes[:])
	data := append(header, payload...)
	stringsTable := []byte("\x00" + livecap.ReservedSymbol + "\x00")
	var symbols bytes.Buffer
	for _, symbol := range []elf.Sym64{{}, {Name: 1, Info: byte(elf.STB_GLOBAL)<<4 | byte(elf.STT_OBJECT), Shndx: 1, Value: 0x1000, Size: uint64(len(data))}} {
		if err := binary.Write(&symbols, binary.LittleEndian, symbol); err != nil {
			t.Fatal(err)
		}
	}
	names := []byte("\x00.rodata\x00.strtab\x00.symtab\x00.shstrtab\x00")
	dataOffset := uint64(64)
	stringsOffset := dataOffset + uint64(len(data))
	symbolOffset := stringsOffset + uint64(len(stringsTable))
	namesOffset := symbolOffset + uint64(symbols.Len())
	sectionOffset := namesOffset + uint64(len(names))
	var object bytes.Buffer
	fileHeader := elf.Header64{Ident: [16]byte{0x7f, 'E', 'L', 'F', 2, 1, 1}, Type: uint16(elf.ET_REL), Machine: uint16(elf.EM_X86_64), Version: 1, Shoff: sectionOffset, Ehsize: 64, Shentsize: 64, Shnum: 5, Shstrndx: 4}
	if err := binary.Write(&object, binary.LittleEndian, fileHeader); err != nil {
		t.Fatal(err)
	}
	object.Write(data)
	object.Write(stringsTable)
	object.Write(symbols.Bytes())
	object.Write(names)
	for _, section := range []elf.Section64{{}, {Name: 1, Type: uint32(elf.SHT_PROGBITS), Flags: uint64(elf.SHF_ALLOC), Addr: 0x1000, Off: dataOffset, Size: uint64(len(data)), Addralign: 1}, {Name: 9, Type: uint32(elf.SHT_STRTAB), Off: stringsOffset, Size: uint64(len(stringsTable)), Addralign: 1}, {Name: 17, Type: uint32(elf.SHT_SYMTAB), Off: symbolOffset, Size: uint64(symbols.Len()), Link: 2, Info: 1, Addralign: 8, Entsize: 24}, {Name: 25, Type: uint32(elf.SHT_STRTAB), Off: namesOffset, Size: uint64(len(names)), Addralign: 1}} {
		if err := binary.Write(&object, binary.LittleEndian, section); err != nil {
			t.Fatal(err)
		}
	}
	return object.Bytes()
}

func TestReviewCapabilitiesSourceCommandBoundary(t *testing.T) {
	for _, test := range []struct {
		name      string
		kind      Kind
		mode      CapabilityMode
		malformed bool
		capacity  bool
	}{
		{name: "closure", kind: KindGoRun, mode: CapabilityModeClosure},
		{name: "linked run", kind: KindGoRun, mode: CapabilityModeLinked},
		{name: "linked test", kind: KindGoTest, mode: CapabilityModeLinked},
		{name: "guarded", kind: KindGoTest, mode: CapabilityModeGuarded},
		{name: "malformed linked", kind: KindGoRun, mode: CapabilityModeLinked, malformed: true},
		{name: "unsupported linked capacity", kind: KindGoRun, mode: CapabilityModeLinked, capacity: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			module := writeModule(t, map[string]string{"go.mod": "module example.com/inspection\n\ngo 1.27.1\n", "main.go": "package main\nfunc main() { panic(\"SOURCE target launched\") }\n"})
			root, work := t.TempDir(), t.TempDir()
			key := strings.Repeat("7", 64)
			writeToolchainInstallation(t, root, key)
			listing, err := json.Marshal(listedPackage{ImportPath: "example.com/inspection", Name: "main", Dir: module, GoFiles: []string{"main.go"}, Module: &listedModule{Path: "example.com/inspection", Main: true}})
			if err != nil {
				t.Fatal(err)
			}
			var commands [][]string
			var outputPath string
			runner := gocommand.New(func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
				commands = append(commands, slices.Clone(request.Command))
				result := hostexec.Result{Termination: hostexec.TerminationExit}
				switch request.Command[1] {
				case "list":
					want := []string{filepath.Join(root, "bin/go"), "list", "-deps", "-json", "-mod=readonly"}
					if test.kind == KindGoTest {
						want = append(want, "-test")
					}
					want = append(want, "-tags", "test_dep", ".")
					if !reflect.DeepEqual(request.Command, want) || request.Dir != module {
						t.Fatalf("listing = %#v", request)
					}
					result.Stdout.RawBytes = listing
				case "env":
					if !reflect.DeepEqual(request.Command, []string{filepath.Join(root, "bin/go"), "env", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED"}) {
						t.Fatalf("identity = %#v", request)
					}
					result.Stdout.RawBytes = []byte("go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n0\n")
				case "build", "test":
					outputPath = request.Command[slices.Index(request.Command, "-o")+1]
					want := []string{filepath.Join(root, "bin/go"), "build"}
					if test.kind == KindGoTest {
						want = []string{filepath.Join(root, "bin/go"), "test", "-c"}
					}
					gcflags := "-gcflags=all=-gomadcap"
					if test.mode == CapabilityModeGuarded {
						gcflags += " -gomadguard"
					}
					want = append(want, "-trimpath", "-buildvcs=false", "-o", outputPath, gcflags, "-ldflags=-linkmode=internal -gomadcap="+key, "-tags", "test_dep", ".")
					if !reflect.DeepEqual(request.Command, want) || request.Dir != module {
						t.Fatalf("build = %#v", request)
					}
					if test.capacity {
						result.ExitCode = 1
						result.Stderr.RawBytes = []byte("live capability facts requires 5000, maximum is 4096\n")
						result.Stderr.Bytes = slices.Clone(result.Stderr.RawBytes)
						return result, nil
					}
					data := sourceLinkedObject(t, key)
					if test.malformed {
						data = []byte("malformed linked object")
					}
					if err := os.WriteFile(outputPath, data, 0o600); err != nil {
						return hostexec.Result{}, err
					}
				default:
					t.Fatalf("unexpected command/target launch = %#v", request.Command)
				}
				return result, nil
			})
			review, err := reviewCapabilitiesWith(t.Context(), Spec{Kind: test.kind, Source: ".", WorkingDir: module, ToolchainRoot: root, PreparationRoot: work, BuildTags: []string{"test_dep"}, CapabilityMode: test.mode}, runner)
			if test.capacity {
				if !IsUnsupportedCapability(err) || IsInvalidCapabilityReview(err) || !strings.Contains(err.Error(), "facts requires 5000, maximum is 4096") {
					t.Fatalf("unsupported linked capacity = %T %v", err, err)
				}
			} else if test.malformed {
				if err == nil || !strings.Contains(err.Error(), "extract linked target capability manifest") || IsInvalidCapabilityReview(err) || IsUnsupportedCapability(err) {
					t.Fatalf("malformed = %T %v", err, err)
				}
			} else if err != nil || review.Schema != CapabilityReviewSchema || review.CapabilityMode != test.mode || (review.CapabilityManifest != nil) != (test.mode != CapabilityModeClosure) {
				t.Fatalf("review = %#v, %v", review, err)
			}
			wantCommands := 1
			if test.mode != CapabilityModeClosure {
				wantCommands = 3
			}
			if len(commands) != wantCommands {
				t.Fatalf("commands = %v", commands)
			}
			entries, err := os.ReadDir(work)
			if err != nil || len(entries) != 0 {
				t.Fatalf("linked workspace retained = %v, %v", entries, err)
			}
		})
	}
}
