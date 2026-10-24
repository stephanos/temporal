package host

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/sys"
	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
)

const maxBuffer = 1 << 20

type RefusalError struct {
	Code    modelv1.RefusalCode
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
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2|experimental.CoreFeaturesExceptionHandling).
		WithMemoryLimitPages(1024).WithCloseOnContextDone(true))
	h := &Host{runtime: r}
	var success bool
	defer func() {
		if !success {
			_ = r.Close(ctx)
		}
	}()
	compiled, err := r.CompileModule(ctx, wasm)
	if err != nil {
		return nil, fmt.Errorf("compile model: %w", err)
	}
	if err := validateModule(compiled); err != nil {
		return nil, err
	}
	if err := instantiateSupport(ctx, r); err != nil {
		return nil, err
	}
	module, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName("model").WithStartFunctions())
	if err != nil {
		return nil, fmt.Errorf("instantiate model: %w", err)
	}
	h.module = module
	h.alloc, h.free, h.call = module.ExportedFunction("umpire_buffer_alloc"), module.ExportedFunction("umpire_buffer_release"), module.ExportedFunction("umpire_call")
	success = true
	return h, nil
}

func validateModule(module wazero.CompiledModule) error {
	for _, imported := range module.ImportedFunctions() {
		namespace, name, _ := imported.Import()
		params, results, ok := supportSignature(name)
		if namespace != "wasi_snapshot_preview1" || !ok || !slices.Equal(imported.ParamTypes(), params) || !slices.Equal(imported.ResultTypes(), results) {
			return fmt.Errorf("model import forbidden: %s.%s %v -> %v", namespace, name, imported.ParamTypes(), imported.ResultTypes())
		}
	}
	if len(module.ImportedMemories()) != 0 {
		return errors.New("model imported memory is forbidden")
	}
	if module.ExportedMemories()["memory"] == nil {
		return errors.New("model must export memory")
	}
	i32, i64 := api.ValueTypeI32, api.ValueTypeI64
	for name, signature := range map[string][2][]api.ValueType{
		"umpire_buffer_alloc": {{i32}, {i32}}, "umpire_buffer_release": {{i32, i32}, nil}, "umpire_call": {{i32, i32, i32}, {i64}},
	} {
		function := module.ExportedFunctions()[name]
		if function == nil || !slices.Equal(function.ParamTypes(), signature[0]) || !slices.Equal(function.ResultTypes(), signature[1]) {
			return fmt.Errorf("model export %s has invalid signature", name)
		}
	}
	for name := range module.ExportedFunctions() {
		switch name {
		case "umpire_buffer_alloc", "umpire_buffer_release", "umpire_call", "_initialize", "__wasm_call_ctors":
		default:
			return fmt.Errorf("unexpected model function export %s", name)
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
	case "fd_close":
		return []api.ValueType{i32}, []api.ValueType{i32}, true
	case "fd_seek":
		return []api.ValueType{i32, api.ValueTypeI64, i32, i32}, []api.ValueType{i32}, true
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
	b.NewFunctionBuilder().WithFunc(func(uint32) uint32 { return 8 }).Export("fd_close")
	b.NewFunctionBuilder().WithFunc(func(uint32, uint64, uint32, uint32) uint32 { return 8 }).Export("fd_seek")
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

func (h *Host) Explore(ctx context.Context, request *modelv1.ExploreRequest) (*modelv1.ExploreResponse, error) {
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_EXPLORE, data)
	if err != nil {
		return nil, err
	}
	return response.GetExplore(), nil
}

func (h *Host) Rows(ctx context.Context, request *modelv1.RowsRequest) (*modelv1.RowsResponse, error) {
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_ROWS, data)
	if err != nil {
		return nil, err
	}
	return response.GetRows(), nil
}

func (h *Host) Reset(ctx context.Context, request *modelv1.ResetRequest) (*modelv1.ResetResponse, error) {
	if request == nil {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Reset requires a request"}
	}
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_RESET, data)
	if err != nil {
		return nil, err
	}
	return response.GetReset_(), nil
}

func (h *Host) Judge(ctx context.Context, request *modelv1.JudgeRequest) (*modelv1.JudgeResponse, error) {
	if request.GetObservation() == nil || request.GetObservation().TypeUrl == "" {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Judge requires an observation Any type URL"}
	}
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_JUDGE, data)
	if err != nil {
		return nil, err
	}
	return response.GetJudge(), nil
}

func (h *Host) Answer(ctx context.Context, request *modelv1.AnswerRequest) (*modelv1.AnswerResponse, error) {
	if request.GetCallback() == nil || request.GetCallback().TypeUrl == "" {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Answer requires a callback Any type URL"}
	}
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_ANSWER, data)
	if err != nil {
		return nil, err
	}
	return response.GetAnswer(), nil
}

func (h *Host) Propose(ctx context.Context, request *modelv1.ProposeRequest) (*modelv1.ProposeResponse, error) {
	if request == nil {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Propose requires a request"}
	}
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_PROPOSE, data)
	if err != nil {
		return nil, err
	}
	return response.GetPropose(), nil
}

func validateProposal(proposal *modelv1.ProposeResponse) error {
	if proposal == nil || proposal.GetRequest().GetTypeUrl() == "" {
		return &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Propose response requires a request"}
	}
	switch proposal.GetKind() {
	case modelv1.ProposalKind_PROPOSAL_KIND_ACTION:
		if proposal.GetAction() == "" {
			return &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Action proposal requires an action"}
		}
	case modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP, modelv1.ProposalKind_PROPOSAL_KIND_READ:
		if proposal.GetAction() != "" {
			return &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Bootstrap and Read actions must be empty"}
		}
	default:
		return &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, Message: "unknown proposal kind"}
	}
	return nil
}

func (h *Host) Annotate(ctx context.Context, request *modelv1.AnnotateRequest) (*modelv1.AnnotateResponse, error) {
	if request.GetInput() == nil || request.GetInput().TypeUrl == "" {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, Message: "Annotate requires an input Any type URL"}
	}
	data, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := h.Call(ctx, modelv1.Method_METHOD_ANNOTATE, data)
	if err != nil {
		return nil, err
	}
	return response.GetAnnotate(), nil
}

func (h *Host) Call(ctx context.Context, method modelv1.Method, request []byte) (response *modelv1.Response, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	name := method.String()
	if h.unusable {
		return nil, fmt.Errorf("%s: model instance is unusable", name)
	}
	if len(request) > maxBuffer {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "request exceeds 1 MiB"}
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
			if _, freeErr := invoke("umpire_buffer_release", h.free, uint64(allocation[0]), uint64(allocation[1])); freeErr != nil {
				err = errors.Join(err, freeErr)
				response = nil
				return
			}
		}
	}()
	invalid := func(message string) (*modelv1.Response, error) {
		h.unusable = true
		_ = h.module.Close(ctx)
		return nil, fmt.Errorf("%s: invalid model ABI: %s", name, message)
	}
	if len(request) != 0 {
		result, allocErr := invoke("umpire_buffer_alloc", h.alloc, uint64(len(request)))
		if allocErr != nil {
			return nil, allocErr
		}
		requestPtr = uint32(result[0])
		if requestPtr == 0 {
			return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "guest allocation exhausted"}
		}
		if !h.module.Memory().Write(requestPtr, request) {
			return invalid("request allocation outside memory")
		}
	}
	result, err := invoke("umpire_call", h.call, uint64(method), uint64(requestPtr), uint64(len(request)))
	if err != nil {
		return nil, err
	}
	responsePtr, responseLen = uint32(result[0]), uint32(result[0]>>32)
	if result[0] == 0 {
		return nil, &RefusalError{Code: modelv1.RefusalCode_REFUSAL_CODE_ALLOCATION, Message: "guest response allocation exhausted"}
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
	response = new(modelv1.Response)
	if err := proto.Unmarshal(owned, response); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", name, err)
	}
	if refusal := response.GetRefusal(); refusal != nil {
		if _, known := modelv1.RefusalCode_name[int32(refusal.Code)]; !known {
			return nil, fmt.Errorf("%s: unknown refusal code %d", name, refusal.Code)
		}
		return nil, &RefusalError{Code: refusal.Code, Message: refusal.Message}
	}
	if method == modelv1.Method_METHOD_EXPLORE && response.GetExplore() != nil ||
		method == modelv1.Method_METHOD_ROWS && response.GetRows() != nil ||
		method == modelv1.Method_METHOD_RESET && response.GetReset_() != nil {
		return response, nil
	}
	if method == modelv1.Method_METHOD_JUDGE && response.GetJudge() != nil {
		if response.GetJudge().GetExplained() == nil && response.GetJudge().GetUnexplained() == nil {
			return nil, fmt.Errorf("%s: missing Judge judgment", name)
		}
		return response, nil
	}
	if method == modelv1.Method_METHOD_ANSWER && response.GetAnswer() != nil {
		if reply := response.GetAnswer().GetReply(); reply == nil || reply.TypeUrl == "" {
			return nil, fmt.Errorf("%s: missing Answer reply Any type URL", name)
		}
		return response, nil
	}
	if method == modelv1.Method_METHOD_PROPOSE && response.GetPropose() != nil {
		if err := validateProposal(response.GetPropose()); err != nil {
			return nil, err
		}
		return response, nil
	}
	if method == modelv1.Method_METHOD_ANNOTATE && response.GetAnnotate() != nil {
		if output := response.GetAnnotate().GetOutput(); output == nil || output.TypeUrl == "" {
			return nil, fmt.Errorf("%s: missing Annotate output Any type URL", name)
		}
		return response, nil
	}
	return nil, fmt.Errorf("%s: unexpected response outcome %T", name, response.Outcome)
}
