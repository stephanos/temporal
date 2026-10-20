package determinism

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityHistoryPreservesAcceptedPrefix(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		change func([]identityRecord) []identityRecord
	}{
		{"remove", func(records []identityRecord) []identityRecord { return records[1:] }},
		{"reuse", func(records []identityRecord) []identityRecord {
			records[0].Site = "import:replacement::site"
			return records
		}},
		{"swap", func(records []identityRecord) []identityRecord {
			records[0], records[1] = records[1], records[0]
			return records
		}},
		{"remove and append", func(records []identityRecord) []identityRecord {
			return append(records[1:], identityRecord{"new.id", "import:new::site"})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var records []identityRecord
			require.NoError(t, json.Unmarshal(identityHistoryJSON, &records))
			data, err := json.Marshal(test.change(records))
			require.NoError(t, err)
			require.ErrorContains(t, checkIdentityHistory(data, loadInventory(t).Entries), "accepted records")
		})
	}
}

func TestIdentityHistoryAppendsAndRetiresWithoutReuse(t *testing.T) {
	t.Parallel()
	inventory := loadInventory(t)
	var records []identityRecord
	require.NoError(t, json.Unmarshal(identityHistoryJSON, &records))
	entry := inventory.Entries[0]
	entry.ID, entry.Site = "appended.id", "retired:appended-site"
	inventory.Entries = append(inventory.Entries, entry)
	records = append(records, identityRecord{entry.ID, entry.Site})
	data, err := json.Marshal(records)
	require.NoError(t, err)
	require.NoError(t, checkIdentityHistory(data, inventory.Entries))
	entry.Disposition = "rejected"
	inventory.Entries[len(inventory.Entries)-1] = entry
	require.NoError(t, checkIdentityHistory(data, inventory.Entries))
	require.ErrorContains(t, checkIdentityHistory(data, inventory.Entries[:len(inventory.Entries)-1]), "removed")
	entry.Site = "retired:reused-site"
	inventory.Entries[len(inventory.Entries)-1] = entry
	require.ErrorContains(t, checkIdentityHistory(data, inventory.Entries), "must retain its site")
}

func TestCompletenessRetainsOnlyRejectedHistoricalSites(t *testing.T) {
	t.Parallel()
	inventory := loadInventory(t)
	row := inventory.Sites("protocol")["execute-fields-and-kinds"]
	require.Contains(t, row.Evidence, "go:internal/backends/wasm/wasi:TestDeterminismRefuseExecuteRequest")
	retired := Inventory{Entries: []Entry{row}}
	require.NoError(t, retired.CheckEvidence(moduleRoot))
	require.NoError(t, retired.Complete("protocol", nil))
	require.ErrorContains(t, retired.Complete("protocol", []string{"new-unlisted-operation"}), "has no inventory row")
	for _, test := range []struct {
		name   string
		change func(*Entry)
	}{
		{"controlled stale site", func(entry *Entry) { entry.Disposition = "controlled" }},
		{"invented rejected id", func(entry *Entry) { entry.ID = "invented.id" }},
		{"invented rejected site", func(entry *Entry) { entry.Site = "protocol:invented-site" }},
		{"missing refusal evidence", func(entry *Entry) { entry.Evidence = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			changed := retired
			changed.Entries = slices.Clone(retired.Entries)
			test.change(&changed.Entries[0])
			require.ErrorContains(t, changed.Complete("protocol", nil), "no longer has")
		})
	}
}

func TestCompletenessRejectsUnacceptedRetirement(t *testing.T) {
	t.Parallel()
	inventory := loadInventory(t)
	var records []identityRecord
	require.NoError(t, json.Unmarshal(identityHistoryJSON, &records))
	entry := inventory.Sites("import")["wasi_snapshot_preview1::path_symlink"]
	entry.ID, entry.Site = "appended.rejected.id", "import:invented::stale"
	inventory.Entries = append(inventory.Entries, entry)
	records = append(records, identityRecord{entry.ID, entry.Site})
	history, err := json.Marshal(records)
	require.NoError(t, err)
	require.NoError(t, checkIdentityHistory(history, inventory.Entries))
	var sites []string
	for site := range inventory.Sites("import") {
		if site != "invented::stale" {
			sites = append(sites, site)
		}
	}
	require.ErrorContains(t, inventory.check("import", sites, true, history), `"invented::stale" that the code no longer has`)
	inventory.Entries[len(inventory.Entries)-1].Disposition = "controlled"
	require.NoError(t, inventory.check("import", append(sites, "invented::stale"), true, history))
}
