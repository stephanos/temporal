package oracle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
	"google.golang.org/protobuf/proto"
	oraclev1 "umpire.local/oracle/umpire/oracle/v1"
)

const maxBuffer = 1 << 20

type RefusalError struct {
	Code    oraclev1.RefusalCode
	Message string
}

func (e *RefusalError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

type Host struct {
	mu                sync.Mutex
	runtime           wazero.Runtime
	module            api.Module
	alloc, free, call api.Function
	unusable          bool
}

func New(ctx context.Context, wasm []byte) (*Host, error) {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(1024).WithCloseOnContextDone(true))
	h := &Host{runtime: r}
	var success bool
	defer func() {
		if !success {
			_ = r.Close(ctx)
		}
	}()
	compiled, err := r.CompileModule(ctx, wasm)
	if err != nil {
		return nil, fmt.Errorf("compile oracle: %w", err)
	}
	if err := validateModule(compiled); err != nil {
		return nil, err
	}
	if err := instantiateSupport(ctx, r); err != nil {
		return nil, err
	}
	module, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName("oracle").WithStartFunctions())
	if err != nil {
		return nil, fmt.Errorf("instantiate oracle: %w", err)
	}
	h.module = module
	h.alloc, h.free, h.call = module.ExportedFunction("alloc"), module.ExportedFunction("free"), module.ExportedFunction("call")
	success = true
	return h, nil
}

func validateModule(module wazero.CompiledModule) error {
	for _, imported := range module.ImportedFunctions() {
		namespace, name, _ := imported.Import()
		params, results, ok := supportSignature(name)
		if namespace != "wasi_snapshot_preview1" || !ok || !slices.Equal(imported.ParamTypes(), params) || !slices.Equal(imported.ResultTypes(), results) {
			return fmt.Errorf("oracle import forbidden: %s.%s %v -> %v", namespace, name, imported.ParamTypes(), imported.ResultTypes())
		}
	}
	if len(module.ImportedMemories()) != 0 {
		return errors.New("oracle imported memory is forbidden")
	}
	if module.ExportedMemories()["memory"] == nil {
		return errors.New("oracle must export memory")
	}
	i32, i64 := api.ValueTypeI32, api.ValueTypeI64
	for name, signature := range map[string][2][]api.ValueType{
		"alloc": {{i32}, {i32}}, "free": {{i32, i32}, nil}, "call": {{i32, i32, i32}, {i64}},
	} {
		function := module.ExportedFunctions()[name]
		if function == nil || !slices.Equal(function.ParamTypes(), signature[0]) || !slices.Equal(function.ResultTypes(), signature[1]) {
			return fmt.Errorf("oracle export %s has invalid signature", name)
		}
	}
	for name := range module.ExportedFunctions() {
		switch name {
		case "alloc", "free", "call", "_initialize", "__wasm_call_ctors":
		default:
			return fmt.Errorf("unexpected oracle function export %s", name)
		}
	}
	return nil
}

func supportSignature(name string) (params, results []api.ValueType, ok bool) {
	i32 := api.ValueTypeI32
	switch name {
	case "environ_get", "environ_sizes_get":
		return []api.ValueType{i32, i32}, []api.ValueType{i32}, true
	case "fd_write":
		return []api.ValueType{i32, i32, i32, i32}, []api.ValueType{i32}, true
	case "proc_exit":
		return []api.ValueType{i32}, nil, true
	default:
		return nil, nil, false
	}
}

// Stock Rust's WASI startup queries an empty environment; panic formatting needs
// a bounded sink and exit. No general WASI implementation is installed.
func instantiateSupport(ctx context.Context, runtime wazero.Runtime) error {
	b := runtime.NewHostModuleBuilder("wasi_snapshot_preview1")
	b.NewFunctionBuilder().WithFunc(func(uint32, uint32) uint32 { return 0 }).Export("environ_get")
	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, count, size uint32) uint32 {
		if !m.Memory().WriteUint32Le(count, 0) || !m.Memory().WriteUint32Le(size, 0) {
			return 21
		}
		return 0
	}).Export("environ_sizes_get")
	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, fd, iov, count, written uint32) uint32 {
		if fd != 1 && fd != 2 {
			return 8
		}
		if count > 1024 {
			return 28
		}
		var total uint32
		for i := uint32(0); i < count; i++ {
			offset := uint64(iov) + uint64(i)*8
			if offset+8 > uint64(m.Memory().Size()) {
				return 21
			}
			ptr, okPtr := m.Memory().ReadUint32Le(uint32(offset))
			length, okLen := m.Memory().ReadUint32Le(uint32(offset) + 4)
			if !okPtr || !okLen || length > maxBuffer-total {
				return 21
			}
			if _, ok := m.Memory().Read(ptr, length); !ok {
				return 21
			}
			total += length
		}
		if !m.Memory().WriteUint32Le(written, total) {
			return 21
		}
		return 0
	}).Export("fd_write")
	b.NewFunctionBuilder().WithFunc(func(code uint32) { panic(sys.NewExitError(code)) }).Export("proc_exit")
	_, err := b.Instantiate(ctx)
	return err
}

func (h *Host) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unusable = true
	return h.runtime.Close(ctx)
}

func (h *Host) Pages() uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.unusable {
		return 0
	}
	return h.module.Memory().Size() / 65536
}

func (h *Host) Explore(ctx context.Context, request *oraclev1.ExploreRequest) (*oraclev1.ExploreResponse, error) {
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, oraclev1.Method_METHOD_EXPLORE, data)
	if err != nil {
		return nil, err
	}
	return response.GetExplore(), nil
}

func (h *Host) Rows(ctx context.Context, request *oraclev1.RowsRequest) (*oraclev1.RowsResponse, error) {
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, oraclev1.Method_METHOD_ROWS, data)
	if err != nil {
		return nil, err
	}
	return response.GetRows(), nil
}

func (h *Host) Call(ctx context.Context, method oraclev1.Method, request []byte) (response *oraclev1.Response, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	name := method.String()
	if h.unusable {
		return nil, fmt.Errorf("%s: oracle instance is unusable", name)
	}
	if len(request) > maxBuffer {
		return nil, &RefusalError{Code: oraclev1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "request exceeds 1 MiB"}
	}
	invoke := func(abiName string, function api.Function, args ...uint64) ([]uint64, error) {
		result, callErr := function.Call(ctx, args...)
		if callErr != nil {
			h.unusable = true
			_ = h.module.Close(ctx)
			return nil, fmt.Errorf("%s: %s: %w", name, abiName, callErr)
		}
		return result, nil
	}
	var requestPtr, responsePtr, responseLen uint32
	defer func() {
		if h.unusable {
			return
		}
		for _, allocation := range [][2]uint32{{responsePtr, responseLen}, {requestPtr, uint32(len(request))}} {
			if allocation[0] == 0 {
				continue
			}
			if _, freeErr := invoke("free", h.free, uint64(allocation[0]), uint64(allocation[1])); freeErr != nil {
				err = errors.Join(err, freeErr)
				response = nil
				return
			}
		}
	}()
	invalid := func(message string) (*oraclev1.Response, error) {
		h.unusable = true
		_ = h.module.Close(ctx)
		return nil, fmt.Errorf("%s: invalid oracle ABI: %s", name, message)
	}
	if len(request) != 0 {
		result, allocErr := invoke("alloc", h.alloc, uint64(len(request)))
		if allocErr != nil {
			return nil, allocErr
		}
		requestPtr = uint32(result[0])
		if requestPtr == 0 {
			return nil, &RefusalError{Code: oraclev1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "guest allocation exhausted"}
		}
		if !h.module.Memory().Write(requestPtr, request) {
			return invalid("request allocation outside memory")
		}
	}
	result, err := invoke("call", h.call, uint64(method), uint64(requestPtr), uint64(len(request)))
	if err != nil {
		return nil, err
	}
	responsePtr, responseLen = uint32(result[0]), uint32(result[0]>>32)
	if result[0] == 0 {
		return nil, &RefusalError{Code: oraclev1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "guest response allocation exhausted"}
	}
	if responsePtr == 0 || responseLen == 0 || responseLen > maxBuffer {
		return invalid("invalid response allocation")
	}
	if requestPtr != 0 && uint64(responsePtr) < uint64(requestPtr)+uint64(len(request)) && uint64(requestPtr) < uint64(responsePtr)+uint64(responseLen) {
		return invalid("response overlaps request")
	}
	data, ok := h.module.Memory().Read(responsePtr, responseLen)
	if !ok {
		return invalid("response allocation outside memory")
	}
	owned := slices.Clone(data)
	response = new(oraclev1.Response)
	if err := proto.Unmarshal(owned, response); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", name, err)
	}
	if refusal := response.GetRefusal(); refusal != nil {
		return nil, &RefusalError{Code: refusal.Code, Message: refusal.Message}
	}
	if method == oraclev1.Method_METHOD_EXPLORE && response.GetExplore() != nil || method == oraclev1.Method_METHOD_ROWS && response.GetRows() != nil {
		return response, nil
	}
	return nil, fmt.Errorf("%s: unexpected response outcome %T", name, response.Outcome)
}
