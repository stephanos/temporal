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
	TemporalActivityAPIBatchCancel = "temporal-activity-api-batch-cancel/v1"
	implementationVersion          = "gomadv3.io-profile/temporal-activity-api-batch-cancel/v1/implementation-v3"
	targetArgument                 = "-test.run=^TestActivityAPIBatchCancelClientTestSuite$"
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
	Package     string           `json:"package"`
	Argument    string           `json:"argument"`
	Entries     []inventoryEntry `json:"entries"`
	ReservedFDs []string         `json:"reserved_fds"`
}

type inventoryEntry struct {
	Boundary    string   `json:"boundary"`
	Disposition string   `json:"disposition"`
	Operations  []string `json:"operations"`
}

func Resolve(name string) (Profile, error) {
	if name != TemporalActivityAPIBatchCancel {
		return Profile{}, fmt.Errorf("unknown I/O profile %q", name)
	}
	encoded, err := record.CanonicalJSON(inventory{
		Schema:   "gomadv3.io-inventory/v1",
		Profile:  TemporalActivityAPIBatchCancel,
		Platform: "darwin/arm64",
		Package:  "go.temporal.io/server/tests",
		Argument: targetArgument,
		Entries: []inventoryEntry{
			{Boundary: "crypto/rand", Disposition: "in-memory", Operations: []string{"Reader.Read", "Read"}},
			{Boundary: "io-transcript", Disposition: "shared-memory", Operations: []string{"expected-replay", "record", "terminal"}},
			{Boundary: "modernc.org/sqlite", Disposition: "target-overlay", Operations: []string{"vfs-entropy", "vfs-time"}},
			{Boundary: "net", Disposition: "in-memory", Operations: []string{"Dial", "DialTCP", "Dialer.DialContext", "Listen", "ListenConfig.Listen", "ListenTCP"}},
			{Boundary: "os", Disposition: "in-memory", Operations: []string{"Hostname", "Mkdir", "MkdirAll", "Stat"}},
		},
		ReservedFDs: []string{"bootstrap", "expected-transcript", "io-config", "io-terminal", "stderr", "stdout", "transcript", "world-config", "world-record"},
	})
	if err != nil {
		return Profile{}, fmt.Errorf("encode I/O profile inventory: %w", err)
	}
	inventoryDigest := digest(encoded)
	implementationDigest := digest([]byte(implementationVersion + "\x00" + string(inventoryDigest)))
	return Profile{
		Name: name, Ready: true, Inventory: encoded, InventorySHA256: inventoryDigest, ImplementationSHA256: implementationDigest,
	}, nil
}

func (profile Profile) ValidatePreparedTarget(spec target.Spec, prepared target.Prepared, environment []string) error {
	if profile.Name != TemporalActivityAPIBatchCancel {
		return fmt.Errorf("unknown I/O profile %q", profile.Name)
	}
	if spec.Kind != target.KindGoTest || prepared.Kind != target.KindGoTest {
		return fmt.Errorf("I/O profile %q requires a go-test target", profile.Name)
	}
	if spec.Source != "./tests" || prepared.Source != "./tests" || prepared.BuildInfo.Path != "go.temporal.io/server/tests.test" {
		return fmt.Errorf("I/O profile %q requires package go.temporal.io/server/tests selected as ./tests", profile.Name)
	}
	if !equalStrings(spec.Args, []string{targetArgument}) || !equalStrings(prepared.Argv, []string{"gomadv3-target", targetArgument}) {
		return fmt.Errorf("I/O profile %q requires exactly %s", profile.Name, targetArgument)
	}
	if len(environment) != 0 {
		return fmt.Errorf("I/O profile %q does not accept target environment additions", profile.Name)
	}
	if !equalStrings(prepared.BuildTags, []string{"test_dep"}) {
		return fmt.Errorf("I/O profile %q requires exactly the test_dep build tag", profile.Name)
	}
	if prepared.GoVersion != "go1.26.4" || prepared.TargetGOOS != "darwin" || prepared.TargetGOARCH != "arm64" {
		return fmt.Errorf("I/O profile %q requires Go 1.26.4 on darwin/arm64", profile.Name)
	}
	resolved, err := Resolve(profile.Name)
	if err != nil {
		return err
	}
	if !bytes.Equal(profile.Inventory, resolved.Inventory) || profile.InventorySHA256 != resolved.InventorySHA256 || profile.ImplementationSHA256 != resolved.ImplementationSHA256 {
		return fmt.Errorf("I/O profile %q identity is invalid", profile.Name)
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
