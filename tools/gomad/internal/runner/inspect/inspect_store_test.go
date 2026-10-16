package inspect

import "testing"

func TestAppendOnlyStoreInspectionUsesVersionedSchema(t *testing.T) {
	if reportSchema != "gomad.inspect/v8" {
		t.Fatalf("append-only store report schema = %q, want gomad.inspect/v8", reportSchema)
	}
}
