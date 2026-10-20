package determinism

import (
	"regexp"
	"strings"
	"testing"
)

func TestInventoryReviewedVersionsMatchCandidate(t *testing.T) {
	t.Parallel()
	inventory := loadInventory(t)
	version := func(path, pattern string) string {
		t.Helper()
		matches := regexp.MustCompile(pattern).FindAllSubmatch(mustRead(t, path), -1)
		if len(matches) != 1 {
			t.Fatalf("%s: expected one version pin, found %d", path, len(matches))
		}
		return string(matches[0][1])
	}
	// The cooperative overlay's source-version guard is its compiler pin.
	want := Reviewed{
		Date:        inventory.Reviewed.Date,
		GoToolchain: version("../toolchain/overlay.go", `string\(version\), "\\n", 2\)\[0\] != "([^"]+)"`),
		Wasmtime:    version("../wasmhost/Cargo.lock", `name = "wasmtime"\nversion = "([^"]+)"`),
		Rust:        version("../wasmhost/rust-toolchain.toml", `(?m)^channel = "([^"]+)"$`),
	}
	if inventory.Reviewed != want {
		t.Fatalf("inventory reviewed versions %+v differ from candidate %+v; re-review the inventory and update versions and date together", inventory.Reviewed, want)
	}
}

func TestInventoryRequiresReviewDate(t *testing.T) {
	t.Parallel()
	reviewed := loadInventory(t).Reviewed
	for _, date := range []string{"", "2026-02-30", "20 October 2026"} {
		t.Run(date, func(t *testing.T) {
			t.Parallel()
			data := strings.Replace(string(inventoryJSON), `"date": "`+reviewed.Date+`"`, `"date": "`+date+`"`, 1)
			if _, err := Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), "reviewed date") {
				t.Fatalf("invalid review date %q: %v", date, err)
			}
		})
	}
}
