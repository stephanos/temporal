package determinism

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

//go:embed identity-history.json
var identityHistoryJSON []byte

// The accepted prefix anchors identities independently of the current inventory
// and Git history. New identities append records; changing this anchor requires
// explicit review of the entire accepted prefix.
const acceptedIdentityCount = 263
const acceptedIdentitySHA256 = "571a435d3eed2e5533450ef8210e3ad0f1d4e779ffc0853287d31775a1b66475"

type identityRecord struct {
	ID   string `json:"id"`
	Site string `json:"site"`
}

func (entry Entry) retiredIdentity(history []byte) bool {
	if entry.Disposition != "rejected" || entry.validate() != nil {
		return false
	}
	var records []identityRecord
	if json.Unmarshal(history, &records) != nil || len(records) < acceptedIdentityCount {
		return false
	}
	return slices.Contains(records[:acceptedIdentityCount], identityRecord{entry.ID, entry.Site})
}

func checkIdentityHistory(data []byte, entries []Entry) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var records []identityRecord
	if err := decoder.Decode(&records); err != nil {
		return fmt.Errorf("historical identity ledger: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("historical identity ledger has trailing data")
	}
	if len(records) < acceptedIdentityCount {
		return errors.New("historical identity ledger removed accepted records")
	}
	var prefix strings.Builder
	for _, record := range records[:acceptedIdentityCount] {
		prefix.WriteString(record.ID + "\t" + record.Site + "\n")
	}
	if fmt.Sprintf("%x", sha256.Sum256([]byte(prefix.String()))) != acceptedIdentitySHA256 {
		return errors.New("historical identity ledger changed accepted records; append new identities")
	}
	byID, bySite := map[string]string{}, map[string]string{}
	for _, record := range records {
		if !idPattern.MatchString(record.ID) || record.Site == "" {
			return errors.New("historical identity ledger has an invalid identity")
		}
		if _, exists := byID[record.ID]; exists {
			return fmt.Errorf("historical identity ledger reuses id %q", record.ID)
		}
		if _, exists := bySite[record.Site]; exists {
			return fmt.Errorf("historical identity ledger reuses site %q", record.Site)
		}
		byID[record.ID], bySite[record.Site] = record.Site, record.ID
	}
	for _, entry := range entries {
		if site, exists := byID[entry.ID]; !exists || site != entry.Site {
			return fmt.Errorf("historical identity %q must retain its site; append new identities to the ledger", entry.ID)
		}
		delete(byID, entry.ID)
	}
	for _, record := range records {
		if _, missing := byID[record.ID]; missing {
			return fmt.Errorf("historical identity %q removed; retain its site with disposition rejected when retired", record.ID)
		}
	}
	return nil
}
