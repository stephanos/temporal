package ioprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
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

type profileDefinition struct {
	name                  string
	target                TargetContract
	inventory             []byte
	inventorySHA256       record.SHA256
	implementationFamily  string
	implementationVersion string
	implementationSHA256  record.SHA256
	prepareBuildOverlay   func(target.Spec, string) (target.Spec, BuildOverlay, error)
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

var deterministicProfile = mustProfileSpec(profileDefinition{
	name:                  Deterministic,
	target:                TargetContract{GoVersion: generatedBoundaryGoVersion, GOOS: generatedBoundaryGOOS, GOARCH: generatedBoundaryGOARCH},
	implementationFamily:  "gomadv3.deterministic-io/v1",
	implementationVersion: deterministicImplementationVersion,
	prepareBuildOverlay:   prepareDeterministicBuildOverlay,
})

var profileRegistry = []ProfileSpec{deterministicProfile}

var profilesByName = map[string]ProfileSpec{Deterministic: deterministicProfile}

func mustProfileSpec(definition profileDefinition) ProfileSpec {
	encoded, err := record.CanonicalJSON(inventory{
		Schema: "gomadv3.io-inventory/v1", Profile: definition.name, Platform: definition.target.GOOS + "/" + definition.target.GOARCH,
		BoundaryManifestVersion: generatedBoundaryManifestVersion, BoundaryManifestSHA256: record.SHA256(generatedBoundaryManifestSHA256),
		Entries: []inventoryEntry{
			{Boundary: "crypto/rand", Disposition: "in-memory", Operations: []string{"Reader.Read", "Read"}},
			{Boundary: "filesystem", Disposition: "in-memory", Operations: []string{"open", "read", "write", "stat", "rename", "remove", "mkdir"}},
			{Boundary: "io-transcript", Disposition: "shared-memory", Operations: []string{"expected-replay", "record", "terminal"}},
			{Boundary: "modernc.org/libc", Disposition: "target-adapter", Operations: []string{"filesystem", "entropy", "time"}},
			{Boundary: "net", Disposition: "in-memory", Operations: []string{"Dial", "DialTCP", "Dialer.DialContext", "Listen", "ListenConfig.Listen", "ListenTCP"}},
			{Boundary: "os.read-only-mount", Disposition: "lazy-in-memory", Operations: []string{"open", "read", "stat", "readdir"}},
		},
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

func Resolve(name string) (ProfileSpec, error) {
	profile, found := profilesByName[name]
	if !found {
		return ProfileSpec{}, fmt.Errorf("unknown I/O profile %q", name)
	}
	return profile, nil
}

func Default() ProfileSpec {
	return deterministicProfile
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
	if profile.definition == nil {
		return nil, fmt.Errorf("invalid I/O profile specification")
	}
	registered, found := profilesByName[profile.definition.name]
	if !found || registered.definition != profile.definition {
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
