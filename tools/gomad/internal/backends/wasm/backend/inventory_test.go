package backend

import "testing"

func TestModuleInventoryRejectsMalformedBeforeExecution(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("not wasm"), []byte{0, 97, 115, 109, 1, 0, 0, 0, 5, 3, 1, 0, 128}} {
		if _, _, err := inventory(data); err == nil {
			t.Fatal("malformed module accepted")
		}
	}
	imports, pages, err := inventory([]byte{0, 97, 115, 109, 1, 0, 0, 0, 5, 3, 1, 0, 1})
	if err != nil || pages != 1 || len(imports) != 0 {
		t.Fatalf("minimal module inventory: %v %d %v", imports, pages, err)
	}
}
