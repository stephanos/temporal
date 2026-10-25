package kettle_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"umpire.local/model/snapshot"
)

var snapshotCandidates = flag.String("umpire-snapshot-candidates", "", "export live snapshot candidates to this directory; comparisons still run")

func readSnapshot(t testing.TB, path string) string {
	t.Helper()
	payload, err := snapshot.Read(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return payload
}

func compareSnapshot(t *testing.T, name, actual string) {
	t.Helper()
	if *snapshotCandidates != "" {
		if err := os.MkdirAll(*snapshotCandidates, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*snapshotCandidates, name), []byte(actual), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("Live candidate exported. From model/: UMPIRE_SNAPSHOT_CANDIDATES=%q INSTA_UPDATE=new mise exec -- cargo test -p umpire-reference-kettle --test live_snapshots --locked -- --ignored", *snapshotCandidates)
	}
	expected := readSnapshot(t, "expected/"+name+".snap")
	if actual != expected {
		t.Fatalf("Kettle %s differs:\n%s", name, actual)
	}
}
