package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

func TestGuestExports(t *testing.T) {
	ctx := context.Background()
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2|experimental.CoreFeaturesExceptionHandling))
	defer func() {
		if err := r.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	module, err := r.CompileModule(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	i32, i64 := api.ValueTypeI32, api.ValueTypeI64
	for name, signature := range map[string][2][]api.ValueType{
		"umpire_buffer_alloc":   {{i32}, {i32}},
		"umpire_buffer_release": {{i32, i32}, nil},
		"umpire_call":           {{i32, i32, i32}, {i64}},
	} {
		function := module.ExportedFunctions()[name]
		if function == nil || !slices.Equal(function.ParamTypes(), signature[0]) || !slices.Equal(function.ResultTypes(), signature[1]) {
			t.Errorf("missing or invalid export %s", name)
		}
	}
	for _, name := range []string{"alloc", "free", "call", "umpire_free"} {
		if module.ExportedFunctions()[name] != nil {
			t.Errorf("obsolete application export %s", name)
		}
	}
	if err := validateModule(module); err != nil {
		t.Fatal(err)
	}
}

func TestBoundaryControls(t *testing.T) {
	ctx := context.Background()
	row, err := proto.Marshal(&modelv1.Response{Outcome: &modelv1.Response_Rows{Rows: &modelv1.RowsResponse{Rows: []string{"copied\n"}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name            string
		pointer, length uint32
		payload         []byte
		trap            bool
		want            string
	}{
		{"valid", 2048, uint32(len(row)), row, false, ""},
		{"trap", 2048, uint32(len(row)), row, true, "METHOD_ROWS"},
		{"exhausted", 0, 0, nil, false, "REFUSAL_CODE_ALLOCATION"},
		{"bounds", 65535, 100, nil, false, "outside memory"},
		{"huge", 2048, maxBuffer + 1, nil, false, "invalid response allocation"},
		{"overlap", 1024, 2, nil, false, "overlaps request"},
		{"malformed", 2048, 1, []byte{255}, false, "decode response"},
		{"wrong outcome", 2048, 2, []byte{10, 0}, false, "unexpected response outcome"},
		{"empty outcome", 2048, 2, []byte{32, 1}, false, "unexpected response outcome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, err := New(ctx, controlModule(tc.pointer, tc.length, tc.payload, tc.trap, false, ""))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := h.Close(ctx); err != nil {
					t.Error(err)
				}
			}()
			response, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: 1})
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v, want %s", err, tc.want)
				}
				if tc.trap {
					if _, err := h.Rows(ctx, &modelv1.RowsRequest{}); err == nil || !strings.Contains(err.Error(), "unusable") {
						t.Fatalf("trap reused: %v", err)
					}
				}
				if tc.name == "exhausted" {
					freed, ok := h.module.Memory().ReadUint32Le(16)
					if h.unusable || !ok || freed != 1 {
						t.Fatalf("exhaustion lost request cleanup: unusable=%v freed=%d", h.unusable, freed)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(response.Rows, "") != "copied\n" {
				t.Fatal(response)
			}
			freed, ok := h.module.Memory().ReadUint32Le(16)
			if !ok || freed != 2 {
				t.Fatalf("free count %d, want 2", freed)
			}
		})
	}
}

func TestRejectsModuleContract(t *testing.T) {
	for _, tc := range []struct {
		name         string
		badSignature bool
		imported     string
	}{
		{"signature", true, ""}, {"clock", false, "clock_time_get"}, {"filesystem", false, "path_open"}, {"unknown", false, "unknown"},
		{"fd signature", false, "fd_write"}, {"env signature", false, "environ_get"}, {"env sizes signature", false, "environ_sizes_get"}, {"exit signature", false, "proc_exit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, err := New(context.Background(), controlModule(2048, 2, []byte{18, 0}, false, tc.badSignature, tc.imported))
			if err == nil {
				_ = h.Close(context.Background())
				t.Fatal("invalid contract accepted")
			}
		})
	}
}

func TestRejectsApplicationExports(t *testing.T) {
	names := []string{"umpire_buffer_alloc", "umpire_buffer_release", "umpire_call"}
	for index, name := range names {
		for _, fault := range []string{"missing", "signature"} {
			t.Run(name+"/"+fault, func(t *testing.T) {
				exports := slices.Clone(names)
				if fault == "missing" {
					exports[index] = ""
				} else {
					next := (index + 1) % len(exports)
					exports[index], exports[next] = exports[next], exports[index]
				}
				h, err := New(context.Background(), controlModuleExports(2048, 2, []byte{18, 0}, false, false, "", exports))
				if err == nil {
					_ = h.Close(context.Background())
				}
				if err == nil || !strings.Contains(err.Error(), "invalid signature") {
					t.Fatalf("invalid export accepted or wrong failure: %v", err)
				}
			})
		}
	}
	for _, name := range []string{"alloc", "free", "call", "umpire_free", "extra"} {
		t.Run("unexpected/"+name, func(t *testing.T) {
			exports := append(slices.Clone(names), name)
			h, err := New(context.Background(), controlModuleExports(2048, 2, []byte{18, 0}, false, false, "", exports))
			if err == nil {
				_ = h.Close(context.Background())
			}
			if err == nil || !strings.Contains(err.Error(), "unexpected model function export "+name) {
				t.Fatalf("unexpected export accepted or wrong failure: %v", err)
			}
		})
	}
}

func TestTrapIdentifiesABIAndLogicalMethod(t *testing.T) {
	ctx := context.Background()
	for _, function := range []string{"umpire_buffer_alloc", "umpire_buffer_release", "umpire_call"} {
		t.Run(function, func(t *testing.T) {
			h, err := New(ctx, controlModule(2048, 2, []byte{18, 0}, function == "umpire_call", false, "", function))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := h.Close(ctx); err != nil {
					t.Error(err)
				}
			}()
			_, err = h.Rows(ctx, &modelv1.RowsRequest{StateId: 1})
			if err == nil || !strings.Contains(err.Error(), "METHOD_ROWS: "+function+":") {
				t.Fatalf("missing method or ABI context: %v", err)
			}
		})
	}
}

func TestGuestAllocationGuards(t *testing.T) {
	ctx := context.Background()
	for _, scenario := range []string{"zero", "oversize", "initialized", "inexact call", "invalid free", "double free"} {
		t.Run(scenario, func(t *testing.T) {
			h, err := New(ctx, guestBytes(t))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := h.Close(ctx); err != nil {
					t.Error(err)
				}
			}()
			allocate := func(size uint64) uint64 {
				t.Helper()
				result, err := h.alloc.Call(ctx, size)
				if err != nil {
					t.Fatal(err)
				}
				return result[0]
			}
			switch scenario {
			case "zero":
				if pointer := allocate(0); pointer != 0 {
					t.Fatalf("alloc(0) = %d", pointer)
				}
				if _, err := h.free.Call(ctx, 0, 0); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if pointer := allocate(maxBuffer + 1); pointer != 0 {
					t.Fatalf("oversize allocation = %d", pointer)
				}
			case "initialized":
				pointer := allocate(16)
				if pointer == 0 {
					t.Fatal("allocation exhausted")
				}
				data, ok := h.module.Memory().Read(uint32(pointer), 16)
				if !ok || string(data) != string(make([]byte, 16)) {
					t.Fatal("allocation is not zero initialized")
				}
				if _, err := h.free.Call(ctx, pointer, 16); err != nil {
					t.Fatal(err)
				}
			case "inexact call":
				pointer := allocate(16)
				if pointer == 0 {
					t.Fatal("allocation exhausted")
				}
				result, err := h.call.Call(ctx, uint64(modelv1.Method_METHOD_ROWS), pointer+1, 15)
				if err != nil {
					t.Fatal(err)
				}
				responsePtr, length := uint32(result[0]), uint32(result[0]>>32)
				data, ok := h.module.Memory().Read(responsePtr, length)
				if !ok {
					t.Fatal("refusal outside memory")
				}
				response := new(modelv1.Response)
				if err := proto.Unmarshal(data, response); err != nil {
					t.Fatal(err)
				}
				if refusal := response.GetRefusal(); refusal == nil || refusal.Code != modelv1.RefusalCode_REFUSAL_CODE_ALLOCATION {
					t.Fatalf("inexact call response %v", response)
				}
				if _, err := h.free.Call(ctx, uint64(responsePtr), uint64(length)); err != nil {
					t.Fatal(err)
				}
				if _, err := h.free.Call(ctx, pointer, 16); err != nil {
					t.Fatal(err)
				}
			case "invalid free", "double free":
				pointer := allocate(16)
				if pointer == 0 {
					t.Fatal("allocation exhausted")
				}
				length := uint64(15)
				if scenario == "double free" {
					if _, err := h.free.Call(ctx, pointer, 16); err != nil {
						t.Fatal(err)
					}
					length = 16
				}
				if _, err := h.free.Call(ctx, pointer, length); err == nil || !strings.Contains(err.Error(), "unreachable") {
					t.Fatalf("invalid free did not trap: %v", err)
				}
			}
		})
	}
}

func TestRefusals(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := h.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	check := func(code modelv1.RefusalCode, err error) {
		t.Helper()
		var refusal *RefusalError
		if !errors.As(err, &refusal) || refusal.Code != code || refusal.Message == "" {
			t.Fatalf("got %v, want %s", err, code)
		}
	}
	_, err = h.Rows(ctx, &modelv1.RowsRequest{})
	check(modelv1.RefusalCode_REFUSAL_CODE_NOT_EXPLORED, err)
	_, err = h.Call(ctx, modelv1.Method(999), nil)
	check(modelv1.RefusalCode_REFUSAL_CODE_UNKNOWN_METHOD, err)
	_, err = h.Call(ctx, modelv1.Method_METHOD_EXPLORE, []byte{255})
	check(modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, err)
	_, err = h.Explore(ctx, &modelv1.ExploreRequest{Budget: 0})
	check(modelv1.RefusalCode_REFUSAL_CODE_BUDGET, err)
	if _, err = h.Explore(ctx, &modelv1.ExploreRequest{Budget: 10000}); err != nil {
		t.Fatal(err)
	}
	_, err = h.Rows(ctx, &modelv1.RowsRequest{StateId: ^uint32(0)})
	check(modelv1.RefusalCode_REFUSAL_CODE_STATE, err)
}

func TestRepeatedAndConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := h.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	report, err := h.Explore(ctx, &modelv1.ExploreRequest{Budget: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if report.Reached != 283 {
		t.Fatalf("states: %d", report.Reached)
	}
	for i := uint32(0); i < report.Reached; i++ {
		if _, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: i}); err != nil {
			t.Fatal(err)
		}
	}
	before := h.Pages()
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Go(func() {
			for i := 0; i < 128; i++ {
				if _, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: uint32(i % 283)}); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	group.Wait()
	if after := h.Pages(); after != before {
		t.Fatalf("memory grew after warmup: %d -> %d pages", before, after)
	}
}

func TestGuestExhaustionPreservesExploration(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := h.Close(ctx); err != nil {
			t.Error(err)
		}
	}()
	if _, err := h.Explore(ctx, &modelv1.ExploreRequest{Budget: 10000}); err != nil {
		t.Fatal(err)
	}
	before, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: 282})
	if err != nil {
		t.Fatal(err)
	}
	request, err := proto.Marshal(&modelv1.ExploreRequest{Budget: 1})
	if err != nil {
		t.Fatal(err)
	}
	var allocations [][2]uint64
	allocate := func(size uint64) uint64 {
		t.Helper()
		result, err := h.alloc.Call(ctx, size)
		if err != nil || result[0] == 0 {
			t.Fatalf("alloc(%d): %v %v", size, result, err)
		}
		allocations = append(allocations, [2]uint64{result[0], size})
		return result[0]
	}
	requestPtr := allocate(uint64(len(request)))
	if !h.module.Memory().Write(uint32(requestPtr), request) {
		t.Fatal("write request")
	}
	for i := 0; i < 15; i++ {
		allocate(maxBuffer)
	}
	allocate(maxBuffer - uint64(len(request)))
	result, err := h.call.Call(ctx, uint64(modelv1.Method_METHOD_EXPLORE), requestPtr, uint64(len(request)))
	if err != nil || result[0] != 0 {
		t.Fatalf("exhausted call: %v %v", result, err)
	}
	for _, allocation := range allocations {
		if _, err := h.free.Call(ctx, allocation[0], allocation[1]); err != nil {
			t.Fatal(err)
		}
	}
	after, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: 282})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(before, after) {
		t.Fatal("response exhaustion replaced previous exploration")
	}
}

// The control uses only the production ABI; unreachable supplies an actual engine trap.
func controlModule(pointer, length uint32, payload []byte, trap, badSignature bool, imported string, trapAt ...string) []byte {
	return controlModuleExports(pointer, length, payload, trap, badSignature, imported,
		[]string{"umpire_buffer_alloc", "umpire_buffer_release", "umpire_call"}, trapAt...)
}

func controlModuleExports(pointer, length uint32, payload []byte, trap, badSignature bool, imported string, exportNames []string, trapAt ...string) []byte {
	u := func(value uint64) []byte {
		var result []byte
		for {
			b := byte(value & 127)
			value >>= 7
			if value != 0 {
				b |= 128
			}
			result = append(result, b)
			if value == 0 {
				return result
			}
		}
	}
	s := func(value int64) []byte {
		var result []byte
		for {
			b := byte(value & 127)
			value >>= 7
			done := value == 0 && b&64 == 0 || value == -1 && b&64 != 0
			if !done {
				b |= 128
			}
			result = append(result, b)
			if done {
				return result
			}
		}
	}
	str := func(value string) []byte { return append(u(uint64(len(value))), []byte(value)...) }
	wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	section := func(id byte, body []byte) {
		wasm = append(wasm, id)
		wasm = append(wasm, u(uint64(len(body)))...)
		wasm = append(wasm, body...)
	}
	types := []byte{3, 0x60, 1, 0x7f, 1, 0x7f, 0x60, 2, 0x7f, 0x7f, 0, 0x60, 3, 0x7f, 0x7f, 0x7f, 1, 0x7e}
	if badSignature {
		types[5] = 0x7e
	}
	section(1, types)
	var offset byte
	if imported != "" {
		body := append([]byte{1}, str("wasi_snapshot_preview1")...)
		body = append(body, str(imported)...)
		body = append(body, 0, 0)
		section(2, body)
		offset = 1
	}
	section(3, []byte{3, 0, 1, 2})
	section(5, []byte{1, 0, 1})
	exports := []byte{1}
	for index, name := range exportNames {
		if name == "" {
			continue
		}
		exports[0]++
		exports = append(exports, str(name)...)
		exports = append(exports, 0, byte(index%3)+offset)
	}
	exports = append(exports, str("memory")...)
	exports = append(exports, 2, 0)
	section(7, exports)
	alloc := append([]byte{0, 0x41}, s(1024)...)
	if badSignature {
		alloc[1] = 0x42
	}
	alloc = append(alloc, 0x0b)
	free := []byte{0, 0x41, 16, 0x41, 16, 0x28, 2, 0, 0x41, 1, 0x6a, 0x36, 2, 0, 0x0b}
	if len(trapAt) != 0 {
		if trapAt[0] == "umpire_buffer_alloc" {
			alloc = []byte{0, 0, 0x0b}
		}
		if trapAt[0] == "umpire_buffer_release" {
			free = []byte{0, 0, 0x0b}
		}
	}
	call := append([]byte{0, 0x42}, s(int64(uint64(length)<<32|uint64(pointer)))...)
	call = append(call, 0x0b)
	if trap {
		call = []byte{0, 0, 0x0b}
	}
	code := []byte{3}
	for _, body := range [][]byte{alloc, free, call} {
		code = append(code, u(uint64(len(body)))...)
		code = append(code, body...)
	}
	section(10, code)
	data := append([]byte{1, 0, 0x41}, s(2048)...)
	data = append(data, 0x0b)
	data = append(data, u(uint64(len(payload)))...)
	data = append(data, payload...)
	section(11, data)
	return wasm
}

var benchmarkSink uint64

func BenchmarkColdLoad(b *testing.B) {
	ctx := context.Background()
	wasm := guestBytes(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h, err := New(ctx, wasm)
		if err != nil {
			b.Fatal(err)
		}
		if err := h.Close(ctx); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(len(wasm)), "wasm-bytes")
}

func BenchmarkRows(b *testing.B) {
	ctx := context.Background()
	h, err := New(ctx, guestBytes(b))
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := h.Close(ctx); err != nil {
			b.Error(err)
		}
	}()
	if _, err := h.Explore(ctx, &modelv1.ExploreRequest{Budget: 10000}); err != nil {
		b.Fatal(err)
	}
	before := h.Pages()
	b.ResetTimer()
	var bytes uint64
	for i := 0; i < b.N; i++ {
		rows, err := h.Rows(ctx, &modelv1.RowsRequest{StateId: uint32(i % 283)})
		if err != nil {
			b.Fatal(err)
		}
		for _, row := range rows.Rows {
			bytes += uint64(len(row))
		}
	}
	b.StopTimer()
	benchmarkSink = bytes
	b.ReportMetric(float64(before), "pages-before")
	b.ReportMetric(float64(h.Pages()), "pages-after")
	b.ReportMetric(float64(bytes), "sink-bytes")
}

func guestBytes(t testing.TB) []byte {
	t.Helper()
	path := os.Getenv("UMPIRE_WASM")
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1/guest/umpire_stock_guest.wasm")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
