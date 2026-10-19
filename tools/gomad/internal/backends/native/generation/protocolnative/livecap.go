package protocolnative

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/native/generation/boundary"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/generation/protocol"
)

type liveCapabilitySchema struct {
	Version     uint32 `json:"version"`
	Schema      string `json:"schema"`
	Symbol      string `json:"symbol"`
	GuardSymbol string `json:"guard_symbol"`
	Header      struct {
		Magic string `json:"magic"`
		Bytes uint32 `json:"bytes"`
	} `json:"header"`
	Limits struct {
		PayloadBytes uint64 `json:"payload_bytes"`
		Facts        uint64 `json:"facts"`
		StringBytes  uint64 `json:"string_bytes"`
		OwnerFacts   uint64 `json:"owner_facts"`
	} `json:"limits"`
	FactKinds         []string `json:"fact_kinds"`
	Dispositions      []string `json:"dispositions"`
	GuardExemptions   []string `json:"guard_exemptions"`
	ForbiddenImports  []string `json:"forbidden_imports"`
	ForbiddenPrefixes []string `json:"forbidden_prefixes"`
}

type liveCapabilityImplementationInputs struct {
	Schema             []byte
	CodecTemplate      []byte
	CompilerEmitter    []byte
	LinkerProjector    []byte
	Encoder            []byte
	GuardFlag          []byte
	GuardSource        []byte
	RuntimeGuard       []byte
	InterceptionSource []byte
	BoundaryTable      []byte
	HostValidator      []byte
	ProjectionContract []byte
}

type liveCapabilityTemplateData struct {
	Package                   string
	Schema                    liveCapabilitySchema
	ImplementationDigest      string
	GuardImplementationDigest string
	UniverseDigest            string
	BoundaryManifestDigest    string
	Boundaries                []liveCapabilityBoundary
	Platforms                 []string
}

type liveCapabilityBoundary struct {
	Package     string
	Target      string
	Hook        string
	Operation   string
	Probe       string
	ProbeID     uint64
	Disposition string
}

func generateLiveCapabilityProtocols(root string, check bool) ([]string, error) {
	definition, err := readLiveCapabilitySchema(filepath.Join(root, "internal/backends", "native", "build", "internal", "livecap", "livecap.json"))
	if err != nil {
		return nil, err
	}
	boundaryDigest, projectedBoundaries, err := boundary.CapabilityProjection(root)
	if err != nil {
		return nil, err
	}
	platforms, err := boundary.Platforms(root)
	if err != nil {
		return nil, err
	}
	universeDigest, err := liveCapabilityUniverseIdentity(definition, boundaryDigest)
	if err != nil {
		return nil, err
	}
	identityInputs, err := readLiveCapabilityImplementationInputs(root)
	if err != nil {
		return nil, err
	}
	implementation := liveCapabilityImplementationIdentity(identityInputs)
	implementationDigest := fmt.Sprintf("sha256:%x", implementation)
	boundaries := make([]liveCapabilityBoundary, len(projectedBoundaries))
	for index, entry := range projectedBoundaries {
		boundaries[index] = liveCapabilityBoundary{
			Package: entry.Package, Target: entry.Target, Hook: entry.Hook, Operation: entry.Operation,
			Probe: entry.Probe, ProbeID: entry.ProbeID, Disposition: entry.Disposition,
		}
	}
	var stale []string
	for _, target := range []struct{ Package, Path string }{
		{Package: "livecap", Path: "internal/backends/native/build/internal/livecap/protocol_generated.go"},
		{Package: "gomadcap", Path: "internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go"},
	} {
		generated, generateErr := protocol.RenderGo(filepath.Join(root, "internal/backends", "native", "build", "internal", "livecap", "livecap.go.tmpl"), liveCapabilityTemplateData{
			Package: target.Package, Schema: definition, ImplementationDigest: implementationDigest, GuardImplementationDigest: implementationDigest,
			UniverseDigest: universeDigest, BoundaryManifestDigest: boundaryDigest, Boundaries: boundaries, Platforms: platforms,
		})
		if generateErr != nil {
			return nil, generateErr
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated) {
				stale = append(stale, target.Path)
			}
			continue
		}
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return nil, fmt.Errorf("write generated live capability protocol: %w", err)
		}
	}
	return stale, nil
}

func readLiveCapabilityImplementationInputs(root string) (liveCapabilityImplementationInputs, error) {
	paths := []string{
		"internal/backends/native/build/internal/livecap/livecap.json",
		"internal/backends/native/build/internal/livecap/livecap.go.tmpl",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"internal/backends/native/transcript/boundary_generated.go",
		"internal/backends/native/build/internal/livecap/livecap.go",
		"internal/backends/native/build/internal/livecap/project.go",
	}
	values := make([][]byte, len(paths))
	for index, relative := range paths {
		value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return liveCapabilityImplementationInputs{}, fmt.Errorf("read live capability implementation input %s: %w", relative, err)
		}
		values[index] = value
	}
	semantics, err := os.ReadFile(filepath.Join(root, "internal/model/runtimesource", "gomad_semantics.go.txt"))
	if err != nil {
		return liveCapabilityImplementationInputs{}, fmt.Errorf("read live capability runtime semantics: %w", err)
	}
	values[7] = append(append(values[7], '\n'), semantics...)
	return liveCapabilityImplementationInputs{
		Schema: values[0], CodecTemplate: values[1], CompilerEmitter: values[2], LinkerProjector: values[3],
		Encoder: values[4], GuardFlag: values[5], GuardSource: values[6], RuntimeGuard: values[7],
		InterceptionSource: values[8], BoundaryTable: values[9], HostValidator: values[10], ProjectionContract: values[11],
	}, nil
}

func liveCapabilityImplementationIdentity(inputs liveCapabilityImplementationInputs) [sha256.Size]byte {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad-live-capability-implementation-source-v2"))
	for _, input := range [][]byte{
		inputs.Schema, inputs.CodecTemplate, inputs.CompilerEmitter, inputs.LinkerProjector, inputs.Encoder,
		inputs.GuardFlag, inputs.GuardSource, inputs.RuntimeGuard, inputs.InterceptionSource,
		inputs.BoundaryTable, inputs.HostValidator, inputs.ProjectionContract,
	} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(input)))
		_, _ = hasher.Write(size[:])
		_, _ = hasher.Write(input)
	}
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

func liveCapabilityUniverseIdentity(definition liveCapabilitySchema, boundaryManifestSHA256 string) (string, error) {
	if len(boundaryManifestSHA256) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(boundaryManifestSHA256, "sha256:") {
		return "", errors.New("live capability boundary manifest identity is invalid")
	}
	projection := struct {
		Schema                 string   `json:"schema"`
		Version                uint32   `json:"version"`
		GuardSymbol            string   `json:"guard_symbol"`
		BoundaryManifestSHA256 string   `json:"boundary_manifest_sha256"`
		FactKinds              []string `json:"fact_kinds"`
		Dispositions           []string `json:"dispositions"`
		GuardExemptions        []string `json:"guard_exemptions"`
		ForbiddenImports       []string `json:"forbidden_imports"`
		ForbiddenPrefixes      []string `json:"forbidden_prefixes"`
	}{
		Schema: definition.Schema, Version: definition.Version, GuardSymbol: definition.GuardSymbol, BoundaryManifestSHA256: boundaryManifestSHA256,
		FactKinds: definition.FactKinds, Dispositions: definition.Dispositions, GuardExemptions: definition.GuardExemptions,
		ForbiddenImports: definition.ForbiddenImports, ForbiddenPrefixes: definition.ForbiddenPrefixes,
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("encode live capability universe: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", digest), nil
}

func readLiveCapabilitySchema(path string) (liveCapabilitySchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return liveCapabilitySchema{}, fmt.Errorf("read live capability schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition liveCapabilitySchema
	if err := decoder.Decode(&definition); err != nil {
		return liveCapabilitySchema{}, fmt.Errorf("decode live capability schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return liveCapabilitySchema{}, errors.New("live capability schema has trailing data")
	}
	if definition.Version != 2 || definition.Schema != "gomad.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" ||
		definition.Header.Magic != "GOMADCAPABILITY\x00" || definition.Header.Bytes != 112 ||
		definition.Limits.PayloadBytes != 16<<20 || definition.Limits.Facts != 100_000 || definition.Limits.StringBytes != 4<<10 || definition.Limits.OwnerFacts != 4_096 ||
		!slices.Equal(definition.FactKinds, []string{"boundary", "capability", "foreign", "guard", "linkname"}) ||
		!slices.Equal(definition.Dispositions, []string{"denied", "guarded", "modeled", "pack"}) ||
		!slices.Equal(definition.GuardExemptions, []string{"syscall.Clearenv", "syscall.Environ", "syscall.Errno.Error", "syscall.Errno.Is", "syscall.Errno.Temporary", "syscall.Errno.Timeout", "syscall.Getenv", "syscall.Setenv", "syscall.Unsetenv", "syscall.Write"}) ||
		!slices.Equal(definition.ForbiddenImports, []string{"os/exec", "os/signal", "os/user", "plugin", "runtime/cgo", "syscall"}) ||
		!slices.Equal(definition.ForbiddenPrefixes, []string{"golang.org/x/sys"}) {
		return liveCapabilitySchema{}, errors.New("live capability schema is unsupported by this generator")
	}
	return definition, nil
}
