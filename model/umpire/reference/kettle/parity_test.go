package kettle_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modelv1 "umpire.local/model/gen/umpire/model/v1"
	model "umpire.local/model/host"
)

func guestBytes(t testing.TB) []byte {
	t.Helper()
	path := os.Getenv("UMPIRE_WASM")
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1/guest/umpire_stock_guest.wasm")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestKettleParity(t *testing.T) {
	ctx := context.Background()
	h, err := model.New(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := h.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	report, err := h.Explore(ctx, &modelv1.ExploreRequest{Budget: 10000})
	if err != nil {
		t.Fatal(err)
	}
	wantReport, err := os.ReadFile("../../checker/tests/fixtures/sample_kettle_trait/expected/report.snap.txt")
	if err != nil {
		t.Fatal(err)
	}
	if report.Report != string(wantReport) {
		t.Fatalf("report differs: %s", report.Report)
	}
	var rows strings.Builder
	for state := uint32(0); state < report.Reached; state++ {
		response, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: state})
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range response.Rows {
			rows.WriteString(row)
		}
	}
	wantRows, err := os.ReadFile("../../checker/tests/fixtures/sample_kettle_trait/expected/rows.snap.txt")
	if err != nil {
		t.Fatal(err)
	}
	if rows.String() != string(wantRows) {
		t.Fatal("ordered rows differ")
	}
}
