package ioprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const (
	Deterministic                      = "gomadv3-deterministic/v1"
	deterministicImplementationVersion = "gomadv3.deterministic-io/v1/implementation-v1"
)

type TargetContract struct {
	GoVersion string
	GOOS      string
	GOARCH    string
}

type ProfileSpec struct {
	definition *profileDefinition
}

type Identity struct {
	Name                 string        `json:"name"`
	ImplementationSHA256 record.SHA256 `json:"implementation_sha256"`
	InventorySHA256      record.SHA256 `json:"inventory_sha256"`
}

type profileDefinition struct {
	name                  string
	target                TargetContract
	inventory             []byte
	inventorySHA256       record.SHA256
	implementationFamily  string
	implementationVersion string
	implementationSHA256  record.SHA256
	adapters              adapterRegistry
}

type inventory struct {
	Schema                  string           `json:"schema"`
	Profile                 string           `json:"profile"`
	Platform                string           `json:"platform"`
	BoundaryManifestVersion string           `json:"boundary_manifest_version"`
	BoundaryManifestSHA256  record.SHA256    `json:"boundary_manifest_sha256"`
	Entries                 []inventoryEntry `json:"entries"`
	ReservedFDs             []string         `json:"reserved_fds"`
}

type inventoryEntry struct {
	Boundary    string   `json:"boundary"`
	Disposition string   `json:"disposition"`
	Operations  []string `json:"operations"`
}

var deterministicAdapters = mustAdapterRegistry(gomadversion.Adapters[:], []adapterImplementation{{
	module: libcModulePath,
	inventory: inventoryEntry{
		Boundary: libcModulePath, Disposition: "target-adapter", Operations: []string{"filesystem", "entropy", "time"},
	},
	prepare: prepareModerncLibc,
}})

var deterministicProfile = mustProfileSpec(profileDefinition{
	name:                  Deterministic,
	target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: generatedBoundaryGOOS, GOARCH: generatedBoundaryGOARCH},
	implementationFamily:  "gomadv3.deterministic-io/v1",
	implementationVersion: deterministicImplementationVersion,
	adapters:              deterministicAdapters,
})

func mustProfileSpec(definition profileDefinition) ProfileSpec {
	entries := []inventoryEntry{
		{Boundary: "crypto/rand", Disposition: "in-memory", Operations: []string{"Reader.Read", "Read"}},
		{Boundary: "filesystem", Disposition: "in-memory", Operations: []string{"open", "read", "write", "stat", "rename", "remove", "mkdir"}},
		{Boundary: "io-transcript", Disposition: "shared-memory", Operations: []string{"expected-replay", "record", "terminal"}},
	}
	entries = append(entries, definition.adapters.inventory()...)
	entries = append(entries,
		inventoryEntry{Boundary: "net", Disposition: "in-memory", Operations: []string{"Dial", "DialTCP", "Dialer.DialContext", "Listen", "ListenConfig.Listen", "ListenTCP"}},
		inventoryEntry{Boundary: "os.read-only-mount", Disposition: "lazy-in-memory", Operations: []string{"open", "read", "stat", "readdir"}},
	)
	encoded, err := record.CanonicalJSON(inventory{
		Schema: "gomadv3.io-inventory/v1", Profile: definition.name, Platform: definition.target.GOOS + "/" + definition.target.GOARCH,
		BoundaryManifestVersion: generatedBoundaryManifestVersion, BoundaryManifestSHA256: record.SHA256(generatedBoundaryManifestSHA256),
		Entries:     entries,
		ReservedFDs: []string{"bootstrap", "expected-transcript", "io-config", "io-terminal", "stderr", "stdout", "transcript", "world-config", "world-record", "read-only-mount-request", "read-only-mount-response"},
	})
	if err != nil {
		panic(fmt.Errorf("encode deterministic I/O inventory: %w", err))
	}
	definition.inventory = encoded
	definition.inventorySHA256 = digest(encoded)
	definition.implementationSHA256 = digest([]byte(definition.implementationVersion + "\x00" + string(definition.inventorySHA256)))
	return ProfileSpec{definition: &definition}
}

func Default() ProfileSpec {
	return deterministicProfile
}

func (profile ProfileSpec) Identity() Identity {
	if profile.definition == nil {
		return Identity{}
	}
	return Identity{
		Name:                 profile.definition.name,
		ImplementationSHA256: profile.definition.implementationSHA256,
		InventorySHA256:      profile.definition.inventorySHA256,
	}
}

func (profile ProfileSpec) Matches(identity Identity) bool {
	return profile.Identity() == identity
}

func (identity Identity) MatchesRecord(candidate record.IOProfile) bool {
	return identity == (Identity{
		Name:                 candidate.Name,
		ImplementationSHA256: candidate.ImplementationSHA256,
		InventorySHA256:      candidate.InventorySHA256,
	})
}

func (profile ProfileSpec) RecordIdentity() record.IOProfile {
	identity := profile.Identity()
	return record.IOProfile{
		Name:                 identity.Name,
		ImplementationSHA256: identity.ImplementationSHA256,
		Inventory:            string(profile.Inventory()),
		InventorySHA256:      identity.InventorySHA256,
	}
}

func (profile ProfileSpec) MatchesRecord(identity record.IOProfile) bool {
	return profile.Identity().MatchesRecord(identity) && string(profile.Inventory()) == identity.Inventory
}

func (profile ProfileSpec) Name() string {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.name
}

func (profile ProfileSpec) Inventory() []byte {
	if profile.definition == nil {
		return nil
	}
	return append([]byte(nil), profile.definition.inventory...)
}

func (profile ProfileSpec) InventorySHA256() record.SHA256 {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.inventorySHA256
}

func (profile ProfileSpec) ImplementationSHA256() record.SHA256 {
	if profile.definition == nil {
		return ""
	}
	return profile.definition.implementationSHA256
}

func (profile ProfileSpec) TargetContract() TargetContract {
	if profile.definition == nil {
		return TargetContract{}
	}
	return profile.definition.target
}

func (profile ProfileSpec) validated() (*profileDefinition, error) {
	if profile.definition == nil || profile.definition != deterministicProfile.definition {
		return nil, fmt.Errorf("invalid I/O profile specification")
	}
	return profile.definition, nil
}

func (profile ProfileSpec) ValidatePreparedTarget(spec target.Spec, prepared target.Prepared, environment []string) error {
	definition, err := profile.validated()
	if err != nil {
		return err
	}
	if spec.Kind != prepared.Kind || spec.Source != prepared.Source {
		return fmt.Errorf("deterministic I/O target identity does not match its build specification")
	}
	if len(prepared.Argv) == 0 || prepared.Argv[0] != "gomadv3-target" || !equalStrings(spec.Args, prepared.Argv[1:]) {
		return fmt.Errorf("deterministic I/O target arguments do not match their build specification")
	}
	if prepared.GoVersion != definition.target.GoVersion || prepared.TargetGOOS != definition.target.GOOS || prepared.TargetGOARCH != definition.target.GOARCH {
		return fmt.Errorf("deterministic I/O requires Go 1.26.4 on darwin/arm64")
	}
	if err := profile.VerifyAdapters(prepared.Adapters); err != nil {
		return fmt.Errorf("deterministic I/O target adapters: %w", err)
	}
	return nil
}

func digest(value []byte) record.SHA256 {
	digest := sha256.Sum256(value)
	return record.SHA256("sha256:" + hex.EncodeToString(digest[:]))
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
