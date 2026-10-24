package host

import (
	"bytes"
	"context"
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestDeniedDescriptorImports(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		params []api.ValueType
	}{
		{"fd_close", []api.ValueType{api.ValueTypeI32}},
		{"fd_seek", []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32, api.ValueTypeI32}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wasm := descriptorModule(tc.name, "wasi_snapshot_preview1", tc.params, []api.ValueType{api.ValueTypeI32}, false)
			h, err := New(ctx, wasm)
			if err != nil {
				t.Fatal(err)
			}
			if err := h.Close(ctx); err != nil {
				t.Fatal(err)
			}
			r := wazero.NewRuntime(ctx)
			t.Cleanup(func() {
				if err := r.Close(ctx); err != nil {
					t.Error(err)
				}
			})
			if err := instantiateSupport(ctx, r); err != nil {
				t.Fatal(err)
			}
			module, err := r.Instantiate(ctx, descriptorModule(tc.name, "wasi_snapshot_preview1", tc.params, []api.ValueType{api.ValueTypeI32}, true))
			if err != nil {
				t.Fatal(err)
			}
			memory := module.Memory()
			before := bytes.Repeat([]byte{0xa5}, int(memory.Size()))
			if !memory.Write(0, before) {
				t.Fatal("initialize memory")
			}
			for _, fd := range []uint64{0, 1, 2, 3, 1<<32 - 1} {
				arguments := [][]uint64{{fd}}
				if tc.name == "fd_seek" {
					arguments = nil
					for _, seek := range [][2]uint64{{0, 0}, {1, 1}, {^uint64(0), 2}, {^uint64(0), 1<<32 - 1}} {
						for _, pointer := range []uint64{0, 8, uint64(memory.Size()) - 4, 1<<32 - 1} {
							arguments = append(arguments, []uint64{fd, seek[0], seek[1], pointer})
						}
					}
				}
				for _, args := range arguments {
					result, err := module.ExportedFunction("support").Call(ctx, args...)
					if err != nil || !slices.Equal(result, []uint64{8}) {
						t.Fatalf("args=%v: got %v %v, want BADF", args, result, err)
					}
					after, ok := memory.Read(0, memory.Size())
					if !ok || !bytes.Equal(after, before) {
						t.Fatalf("args=%v: denied import changed memory", args)
					}
				}
			}
		})
	}
}

func TestRejectsDescriptorImportContracts(t *testing.T) {
	i32, i64 := api.ValueTypeI32, api.ValueTypeI64
	for _, tc := range []struct {
		name, namespace string
		params, results []api.ValueType
	}{
		{"fd_close", "other", []api.ValueType{i32}, []api.ValueType{i32}},
		{"fd_close", "wasi_snapshot_preview1", []api.ValueType{i64}, []api.ValueType{i32}},
		{"fd_close", "wasi_snapshot_preview1", []api.ValueType{i32}, []api.ValueType{i64}},
		{"fd_close", "wasi_snapshot_preview1", nil, []api.ValueType{i32}},
		{"fd_seek", "other", []api.ValueType{i32, i64, i32, i32}, []api.ValueType{i32}},
		{"fd_seek", "wasi_snapshot_preview1", []api.ValueType{i32, i32, i32, i32}, []api.ValueType{i32}},
		{"fd_seek", "wasi_snapshot_preview1", []api.ValueType{i32, i64, i32}, []api.ValueType{i32}},
		{"fd_seek", "wasi_snapshot_preview1", []api.ValueType{i32, i64, i32, i32}, nil},
	} {
		t.Run(tc.name+"/"+tc.namespace, func(t *testing.T) {
			h, err := New(context.Background(), descriptorModule(tc.name, tc.namespace, tc.params, tc.results, false))
			if err == nil {
				_ = h.Close(context.Background())
			}
			if err == nil || !strings.Contains(err.Error(), "model import forbidden:") {
				t.Fatalf("invalid import accepted or wrong failure: %v", err)
			}
		})
	}
}

func descriptorModule(name, namespace string, params, results []api.ValueType, exportImport bool) []byte {
	base := controlModule(2048, 2, []byte{18, 0}, false, false, name)
	wasm := slices.Clone(base[:8])
	str := func(value string) []byte {
		return append(binary.AppendUvarint(nil, uint64(len(value))), value...)
	}
	for offset := 8; offset < len(base); {
		id := base[offset]
		size, n := binary.Uvarint(base[offset+1:])
		start := offset + 1 + n
		body := slices.Clone(base[start : start+int(size)])
		offset = start + int(size)
		switch id {
		case 1:
			body[0]++
			body = append(body, 0x60, byte(len(params)))
			body = append(body, params...)
			body = append(body, byte(len(results)))
			body = append(body, results...)
		case 2:
			body = append([]byte{1}, str(namespace)...)
			body = append(body, str(name)...)
			body = append(body, 0, 3)
		case 7:
			if exportImport {
				body[0]++
				body = append(body, str("support")...)
				body = append(body, 0, 0)
			}
		}
		wasm = append(wasm, id)
		wasm = binary.AppendUvarint(wasm, uint64(len(body)))
		wasm = append(wasm, body...)
	}
	return wasm
}
