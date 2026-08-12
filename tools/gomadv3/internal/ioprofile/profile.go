package ioprofile

import (
	"bytes"
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

type Profile struct {
	Name                 string
	Ready                bool
	Inventory            []byte
	InventorySHA256      record.SHA256
	ImplementationSHA256 record.SHA256
}

type inventory struct {
	Schema      string           `json:"schema"`
	Profile     string           `json:"profile"`
	Platform    string           `json:"platform"`
	Entries     []inventoryEntry `json:"entries"`
	ReservedFDs []string         `json:"reserved_fds"`
}

type inventoryEntry struct {
	Boundary    string   `json:"boundary"`
	Disposition string   `json:"disposition"`
	Operations  []string `json:"operations"`
}

func Resolve(name string) (Profile, error) {
	if name != Deterministic {
		return Profile{}, fmt.Errorf("unknown I/O profile %q", name)
	}
	encoded, err := record.CanonicalJSON(inventory{
		Schema: "gomadv3.io-inventory/v1", Profile: Deterministic, Platform: "darwin/arm64",
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
		return Profile{}, fmt.Errorf("encode deterministic I/O inventory: %w", err)
	}
	inventoryDigest := digest(encoded)
	implementationDigest := digest([]byte(deterministicImplementationVersion + "\x00" + string(inventoryDigest)))
	return Profile{Name: Deterministic, Ready: true, Inventory: encoded, InventorySHA256: inventoryDigest, ImplementationSHA256: implementationDigest}, nil
}

func Default() Profile {
	profile, err := Resolve(Deterministic)
	if err != nil {
		panic(err)
	}
	return profile
}

func (profile Profile) ValidatePreparedTarget(spec target.Spec, prepared target.Prepared, environment []string) error {
	resolved, err := Resolve(profile.Name)
	if err != nil {
		return err
	}
	if spec.Kind != prepared.Kind || spec.Source != prepared.Source {
		return fmt.Errorf("deterministic I/O target identity does not match its build specification")
	}
	if len(prepared.Argv) == 0 || prepared.Argv[0] != "gomadv3-target" || !equalStrings(spec.Args, prepared.Argv[1:]) {
		return fmt.Errorf("deterministic I/O target arguments do not match their build specification")
	}
	if prepared.GoVersion != "go1.26.4" || prepared.TargetGOOS != "darwin" || prepared.TargetGOARCH != "arm64" {
		return fmt.Errorf("deterministic I/O requires Go 1.26.4 on darwin/arm64")
	}
	if !bytes.Equal(profile.Inventory, resolved.Inventory) || profile.InventorySHA256 != resolved.InventorySHA256 || profile.ImplementationSHA256 != resolved.ImplementationSHA256 {
		return fmt.Errorf("deterministic I/O identity is invalid")
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
