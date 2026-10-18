package oracle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"
	oraclev1 "umpire.local/oracle/umpire/oracle/v1"
)

func TestBoundaryControls(t *testing.T) {
	ctx := context.Background()
	row, err := proto.Marshal(&oraclev1.Response{Outcome: &oraclev1.Response_Rows{Rows: &oraclev1.RowsResponse{Rows: []string{"copied\n"}}}})
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
			response, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: 1})
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v, want %s", err, tc.want)
				}
				if tc.trap {
					if _, err := h.Rows(ctx, &oraclev1.RowsRequest{}); err == nil || !strings.Contains(err.Error(), "unusable") {
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
	check := func(code oraclev1.RefusalCode, err error) {
		t.Helper()
		var refusal *RefusalError
		if !errors.As(err, &refusal) || refusal.Code != code || refusal.Message == "" {
			t.Fatalf("got %v, want %s", err, code)
		}
	}
	_, err = h.Rows(ctx, &oraclev1.RowsRequest{})
	check(oraclev1.RefusalCode_REFUSAL_CODE_NOT_EXPLORED, err)
	_, err = h.Call(ctx, oraclev1.Method(999), nil)
	check(oraclev1.RefusalCode_REFUSAL_CODE_UNKNOWN_METHOD, err)
	_, err = h.Call(ctx, oraclev1.Method_METHOD_EXPLORE, []byte{255})
	check(oraclev1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, err)
	_, err = h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 0})
	check(oraclev1.RefusalCode_REFUSAL_CODE_BUDGET, err)
	if _, err = h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 10000}); err != nil {
		t.Fatal(err)
	}
	_, err = h.Rows(ctx, &oraclev1.RowsRequest{StateId: ^uint32(0)})
	check(oraclev1.RefusalCode_REFUSAL_CODE_STATE, err)
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
	report, err := h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if report.Reached != 283 {
		t.Fatalf("states: %d", report.Reached)
	}
	for i := uint32(0); i < report.Reached; i++ {
		if _, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: i}); err != nil {
			t.Fatal(err)
		}
	}
	before := h.Pages()
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Go(func() {
			for i := 0; i < 128; i++ {
				if _, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: uint32(i % 283)}); err != nil {
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
	if _, err := h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 10000}); err != nil {
		t.Fatal(err)
	}
	before, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: 282})
	if err != nil {
		t.Fatal(err)
	}
	request, err := proto.Marshal(&oraclev1.ExploreRequest{Budget: 1})
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
	result, err := h.call.Call(ctx, uint64(oraclev1.Method_METHOD_EXPLORE), requestPtr, uint64(len(request)))
	if err != nil || result[0] != 0 {
		t.Fatalf("exhausted call: %v %v", result, err)
	}
	for _, allocation := range allocations {
		if _, err := h.free.Call(ctx, allocation[0], allocation[1]); err != nil {
			t.Fatal(err)
		}
	}
	after, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: 282})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(before, after) {
		t.Fatal("response exhaustion replaced previous exploration")
	}
}

// The control uses only the production ABI; unreachable supplies an actual engine trap.
func controlModule(pointer, length uint32, payload []byte, trap, badSignature bool, imported string) []byte {
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
	exports := []byte{4}
	for index, name := range []string{"alloc", "free", "call"} {
		exports = append(exports, str(name)...)
		exports = append(exports, 0, byte(index)+offset)
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
	if _, err := h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 10000}); err != nil {
		b.Fatal(err)
	}
	before := h.Pages()
	b.ResetTimer()
	var bytes uint64
	for i := 0; i < b.N; i++ {
		rows, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: uint32(i % 283)})
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
			target = "../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1/release/umpire_wasm.wasm")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestKettleParity(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, guestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := h.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	report, err := h.Explore(ctx, &oraclev1.ExploreRequest{Budget: 10000})
	if err != nil {
		t.Fatal(err)
	}
	wantReport, err := os.ReadFile("../../umpire/tests/fixtures/kettle_trait/expected/report.txt")
	if err != nil {
		t.Fatal(err)
	}
	if report.Report != string(wantReport) {
		t.Fatalf("report differs: %s", report.Report)
	}
	var rows strings.Builder
	for state := uint32(0); state < report.Reached; state++ {
		response, err := h.Rows(ctx, &oraclev1.RowsRequest{StateId: state})
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range response.Rows {
			rows.WriteString(row)
		}
	}
	wantRows, err := os.ReadFile("../../umpire/tests/fixtures/kettle_trait/expected/rows.txt")
	if err != nil {
		t.Fatal(err)
	}
	if rows.String() != string(wantRows) {
		t.Fatal("ordered rows differ")
	}
}
