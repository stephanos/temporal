package inspect

import "testing"

func TestAppendOnlyStoreInspectionUsesVersionedSchema(t *testing.T) {
	if reportSchema != "gomad3.inspect/v8" {
		t.Fatalf("append-only store report schema = %q, want gomad3.inspect/v8", reportSchema)
	}
}
