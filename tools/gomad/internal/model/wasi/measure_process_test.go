package wasi

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func helperPath(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("GOMAD_WASM_TEST_HELPER"); override != "" {
		if !filepath.IsAbs(override) {
			t.Fatal("GOMAD_WASM_TEST_HELPER must be absolute")
		}
		if _, err := os.Stat(override); err != nil {
			t.Fatal(err)
		}
		return override
	}
	name, err := filepath.Abs("../../../rust/target/release/gomad-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); err != nil {
		t.Skip("build pinned release wasmhost to run guest gates: ", err)
	}
	return name
}

func fileHash(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func binaryInventory(t *testing.T, data []byte) ([]Import, uint64) {
	t.Helper()
	if len(data) < 8 || string(data[:4]) != "\x00asm" {
		t.Fatal("not binary WASM")
	}
	readUint := func(buffer *[]byte) uint64 {
		var value uint64
		for shift := uint(0); shift < 64; shift += 7 {
			if len(*buffer) == 0 {
				t.Fatal("truncated WASM LEB")
			}
			b := (*buffer)[0]
			*buffer = (*buffer)[1:]
			value |= uint64(b&127) << shift
			if b < 128 {
				return value
			}
		}
		t.Fatal("overflow WASM LEB")
		return 0
	}
	readName := func(buffer *[]byte) string {
		size := readUint(buffer)
		if size > uint64(len(*buffer)) {
			t.Fatal("truncated WASM name")
		}
		name := string((*buffer)[:size])
		*buffer = (*buffer)[size:]
		return name
	}
	types := []Import{}
	imports := []Import{}
	pages := uint64(0)
	for remaining := data[8:]; len(remaining) > 0; {
		kind := remaining[0]
		remaining = remaining[1:]
		size := readUint(&remaining)
		if size > uint64(len(remaining)) {
			t.Fatal("truncated WASM section")
		}
		section := remaining[:size]
		remaining = remaining[size:]
		switch kind {
		case 1:
			count := readUint(&section)
			for i := uint64(0); i < count; i++ {
				if len(section) == 0 || section[0] != 0x60 {
					t.Fatal("fixture WASM function type")
				}
				section = section[1:]
				signature := Import{Params: []string{}, Results: []string{}}
				for j := 0; j < 2; j++ {
					count := readUint(&section)
					for k := uint64(0); k < count; k++ {
						if len(section) == 0 {
							t.Fatal("truncated value type")
						}
						name := map[byte]string{0x7f: "i32", 0x7e: "i64", 0x7d: "f32", 0x7c: "f64"}[section[0]]
						section = section[1:]
						if name == "" {
							t.Fatal("fixture WASM value type")
						}
						if j == 0 {
							signature.Params = append(signature.Params, name)
						} else {
							signature.Results = append(signature.Results, name)
						}
					}
				}
				types = append(types, signature)
			}
		case 2:
			count := readUint(&section)
			for i := uint64(0); i < count; i++ {
				module, name := readName(&section), readName(&section)
				if len(section) == 0 || section[0] != 0 {
					t.Fatal("fixture import is not function")
				}
				section = section[1:]
				index := readUint(&section)
				if index >= uint64(len(types)) {
					t.Fatal("invalid fixture type index")
				}
				signature := types[index]
				signature.Module = module
				signature.Name = name
				imports = append(imports, signature)
			}
		case 5:
			if readUint(&section) != 1 {
				t.Fatal("fixture memory count")
			}
			flags := readUint(&section)
			pages = readUint(&section)
			if flags&1 != 0 {
				readUint(&section)
			}
		}
	}
	if pages == 0 {
		t.Fatal("fixture memory missing")
	}
	return imports, pages
}
