package deterministicio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	Deterministic                      = "gomad3-deterministic/v1"
	deterministicImplementationVersion = "gomad3.deterministic-io/v1/implementation-v11"
)

type TargetContract struct {
	GoVersion string
	GOOS      string
	GOARCH    string
}

type Spec struct {
	definition *profileDefinition
}

type Contract struct {
	ImplementationSHA256 Digest `json:"implementation_sha256"`
	InventorySHA256      Digest `json:"inventory_sha256"`
	Name                 string `json:"name"`
}

type profileDefinition struct {
	name                  string
	target                TargetContract
	inventory             []byte
	inventorySHA256       Digest
	implementationFamily  string
	implementationVersion string
	implementationSHA256  Digest
	adapters              adapterRegistry
}

type inventory struct {
	BoundaryManifestSHA256  Digest           `json:"boundary_manifest_sha256"`
	BoundaryManifestVersion string           `json:"boundary_manifest_version"`
	Entries                 []inventoryEntry `json:"entries"`
	Platform                string           `json:"platform"`
	Profile                 string           `json:"profile"`
	ReservedFDs             []string         `json:"reserved_fds"`
	Schema                  string           `json:"schema"`
}

type inventoryEntry struct {
	Boundary    string   `json:"boundary"`
	Disposition string   `json:"disposition"`
	Operations  []string `json:"operations"`
}

var deterministicAdapters = mustAdapterRegistry(gomadversion.Adapters[:], []adapterImplementation{
	{
		module: xnetModulePath,
		inventory: inventoryEntry{
			Boundary: xnetModulePath, Disposition: "target-adapter", Operations: []string{"raw-socket-option-denial"},
		},
		prepare: prepareXNet,
	},
	{
		module: grpcModulePath,
		inventory: inventoryEntry{
			Boundary: grpcModulePath, Disposition: "target-adapter", Operations: []string{"virtual-tcp-keepalive-suppression"},
		},
		prepare: prepareGRPC,
	},
	{
		module: libcModulePath,
		inventory: inventoryEntry{
			Boundary: libcModulePath, Disposition: "target-adapter", Operations: []string{"filesystem", "entropy", "time"},
		},
		prepare: prepareModerncLibc,
	},
	{
		module: memoryModulePath,
		inventory: inventoryEntry{
			Boundary: memoryModulePath, Disposition: "target-adapter", Operations: []string{"anonymous-memory"},
		},
		prepare: prepareModerncMemory,
	},
})

var deterministicProfile = mustSpec(profileDefinition{
	name:                  Deterministic,
	target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: generatedBoundaryGOOS, GOARCH: generatedBoundaryGOARCH},
	implementationFamily:  "gomad3.deterministic-io/v1",
	implementationVersion: deterministicImplementationVersion,
	adapters:              deterministicAdapters,
})

func mustSpec(definition profileDefinition) Spec {
	entries := []inventoryEntry{
		{Boundary: "crypto/rand", Disposition: "in-memory", Operations: []string{"Reader.Read", "Read"}},
		{Boundary: "filesystem", Disposition: "in-memory", Operations: []string{"open", "read", "write", "stat", "rename", "remove", "mkdir"}},
		{Boundary: "io-transcript", Disposition: "shared-memory", Operations: []string{"expected-replay", "record", "terminal"}},
	}
	entries = append(entries, definition.adapters.inventory()...)
	entries = append(entries,
		inventoryEntry{Boundary: "net", Disposition: "in-memory", Operations: []string{"Dial", "DialTCP", "Dialer.DialContext", "Listen", "ListenConfig.Listen", "ListenTCP", "Resolver.LookupIPAddr(localhost)"}},
		inventoryEntry{Boundary: "os.read-only-mount", Disposition: "lazy-in-memory", Operations: []string{"open", "read", "stat", "readdir"}},
	)
	encoded, err := canonicalJSON(inventory{
		Schema: "gomad3.io-inventory/v1", Profile: definition.name, Platform: definition.target.GOOS + "/" + definition.target.GOARCH,
		BoundaryManifestVersion: generatedBoundaryManifestVersion, BoundaryManifestSHA256: Digest(generatedBoundaryManifestSHA256),
		Entries:     entries,
		ReservedFDs: []string{"bootstrap", "expected-transcript", "io-config", "io-terminal", "stderr", "stdout", "transcript", "world-config", "world-record", "read-only-mount-request", "read-only-mount-response"},
	})
	if err != nil {
		panic(fmt.Errorf("encode deterministic I/O inventory: %w", err))
	}
	definition.inventory = encoded
	definition.inventorySHA256 = digest(encoded)
	definition.implementationSHA256 = digest([]byte(definition.implementationVersion + "\x00" + string(definition.inventorySHA256)))
	return Spec{definition: &definition}
}

func Default() Spec {
	return deterministicProfile
}

func (profile Spec) Identity() Contract {
	if profile.definition == nil {
		return Contract{}
	}
	return Contract{
		Name:                 profile.definition.name,
		ImplementationSHA256: profile.definition.implementationSHA256,
		InventorySHA256:      profile.definition.inventorySHA256,
	}
}

func (profile Spec) Matches(identity Contract) bool {
	return profile.Identity() == identity
}

func (profile Spec) MatchesRecorded(name, implementationSHA256, inventorySHA256, encodedInventory string) bool {
	identity := profile.Identity()
	return identity.Name == name && string(identity.ImplementationSHA256) == implementationSHA256 && string(identity.InventorySHA256) == inventorySHA256 && string(profile.Inventory()) == encodedInventory
}

func (profile Spec) Name() string {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.name
}

func (profile Spec) Inventory() []byte {
	if profile.definition == nil {
		return nil
	}
	return append([]byte(nil), profile.definition.inventory...)
}

func (profile Spec) InventorySHA256() Digest {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.inventorySHA256
}

func (profile Spec) ImplementationSHA256() Digest {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.implementationSHA256
}

func (profile Spec) TargetContract() TargetContract {
	if profile.definition == nil {
		return TargetContract{}
	}
	return profile.definition.target
}

func (profile Spec) validated() (*profileDefinition, error) {
	if profile.definition == nil || profile.definition != deterministicProfile.definition {
		return nil, fmt.Errorf("invalid I/O profile specification")
	}
	return profile.definition, nil
}

func (profile Spec) ValidatePreparedTarget(spec target.Spec, prepared target.Prepared, environment []string) error {
	definition, err := profile.validated()
	if err != nil {
		return err
	}
	if spec.Kind != prepared.Kind || spec.Source != prepared.Source {
		return fmt.Errorf("deterministic I/O target identity does not match its build specification")
	}
	if len(prepared.Argv) == 0 || prepared.Argv[0] != "gomad3-target" || !equalStrings(spec.Args, prepared.Argv[1:]) {
		return fmt.Errorf("deterministic I/O target arguments do not match their build specification")
	}
	if prepared.GoVersion != definition.target.GoVersion || prepared.TargetGOOS != definition.target.GOOS || prepared.TargetGOARCH != definition.target.GOARCH {
		return fmt.Errorf("deterministic I/O requires Go 1.26.4 on darwin/arm64")
	}
	adapters := make([]Adapter, len(prepared.Adapters))
	for index, adapter := range prepared.Adapters {
		adapters[index] = Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	if err := profile.VerifyAdapters(adapters); err != nil {
		return fmt.Errorf("deterministic I/O target adapters: %w", err)
	}
	return nil
}

func digest(value []byte) Digest {
	digest := sha256.Sum256(value)
	return Digest("sha256:" + hex.EncodeToString(digest[:]))
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
