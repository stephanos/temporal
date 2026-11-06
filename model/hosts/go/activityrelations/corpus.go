package activityrelations

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	realizationv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/realization/v1"
	"github.com/temporalio/umpire/hosts/go/relation"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// Row is one actual invocation in capture order. Request and response bytes are kept only for
// bound RPCs, re-encoded with every Secret-claimed value replaced by a placeholder of the same
// presence; every other row keeps only its method and status, so worker task tokens never reach
// a stored corpus. RawHMAC digests the unmodified response under a key discarded with the
// Collector, so raw distinctness survives without a recoverable or confirmable Secret.
type Row struct {
	Harness  string `json:"harness"`
	Case     string `json:"case"`
	Seed     uint64 `json:"seed,omitempty"`
	Method   string `json:"method"`
	Code     uint32 `json:"code"`
	Request  string `json:"request_hex,omitempty"`
	Response string `json:"response_hex,omitempty"`
	RawHMAC  string `json:"raw_hmac,omitempty"`
	Excluded string `json:"excluded,omitempty"`
}

const startMethod = "temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"

type caseKey struct{}

// WithCase labels the invocations made under ctx with their harness and case; unlabelled
// invocations are excluded from the corpus by policy.
func WithCase(ctx context.Context, harness, label string, seed uint64) context.Context {
	return context.WithValue(ctx, caseKey{}, Row{Harness: harness, Case: label, Seed: seed})
}

// Collector records rows from a gRPC interceptor and from judged explore Observations.
type Collector struct {
	bindings Bindings
	claims   map[secretField]bool
	digest   rawDigest
	mu       sync.Mutex
	rows     []Row
	receipts map[[3]string]string
}

// NewCollector admits the generated bindings once.
func NewCollector() (*Collector, error) {
	bindings, err := NewBindings()
	if err != nil {
		return nil, err
	}
	digest, err := newRawDigest()
	if err != nil {
		return nil, err
	}
	return &Collector{bindings: bindings, claims: secrets(), digest: digest, receipts: map[[3]string]string{}}, nil
}

// Rows returns a copy of the rows captured so far.
func (c *Collector) Rows() []Row {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.rows)
}

func (c *Collector) add(label Row, method string, request, response proto.Message, code uint32) error {
	row := label
	row.Method = trimSlash(method)
	row.Code = code
	_, bound := c.bindings.For(row.Method)
	if code == 0 && response != nil {
		data, err := proto.MarshalOptions{Deterministic: true}.Marshal(response)
		if err != nil {
			return err
		}
		row.RawHMAC = c.digest.of(data)
		if bound {
			if data, err = sanitize(response, c.claims); err != nil {
				return err
			}
			row.Response = hex.EncodeToString(data)
		}
	}
	if bound && request != nil {
		data, err := sanitize(request, c.claims)
		if err != nil {
			return err
		}
		row.Request = hex.EncodeToString(data)
	}
	switch {
	case row.Method == "temporal.api.workflowservice.v1.WorkflowService/TerminateActivityExecution":
		row.Excluded = "cleanup"
	case row.Harness == "":
		row.Excluded = "unlabelled"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = append(c.rows, row)
	return nil
}

// Intercept records every completed unary call made through a connection, including
// prerequisites and baselines, without changing the call or its result.
func (c *Collector) Intercept(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoke(ctx, method, request, reply, conn, opts...)
	label, _ := ctx.Value(caseKey{}).(Row)
	requestMessage, _ := request.(proto.Message)
	replyMessage, _ := reply.(proto.Message)
	code := uint32(status.Code(err))
	if recordErr := c.add(label, method, requestMessage, replyMessage, code); recordErr != nil {
		return errors.Join(err, recordErr)
	}
	return err
}

// Judged is what Observe needs from one judged explore Observation.
type Judged struct {
	Harness, Case string
	Seed          uint64
	Observation   *anypb.Any
	Explained     bool
}

// Observe records one judged Observation. The host check runs on its actual bytes; a guest that
// explained an invocation the host refuses to correlate is a disagreement and fails.
func (c *Collector) Observe(judged Judged) error {
	if judged.Observation.MessageIs(new(temporalv1.ReconciledObservation)) {
		// A reconciled observation holds two actual invocations; each keeps its own method, so
		// a cancellation acknowledgment is never counted as the Describe that followed it.
		compound := new(temporalv1.ReconciledObservation)
		if err := judged.Observation.UnmarshalTo(compound); err != nil {
			return fmt.Errorf("reconciled observation: %w", err)
		}
		for _, part := range []*anypb.Any{compound.GetAcknowledged(), compound.GetCurrent()} {
			if part == nil {
				continue
			}
			judged.Observation = part
			if err := c.Observe(judged); err != nil {
				return err
			}
		}
		return nil
	}
	observation := new(realizationv1.Observation)
	if err := judged.Observation.UnmarshalTo(observation); err != nil {
		return fmt.Errorf("observation: %w", err)
	}
	method := trimSlash(observation.GetMethod())
	label := Row{Harness: judged.Harness, Case: judged.Case, Seed: judged.Seed, Method: method}
	if failure := observation.GetError(); failure != nil {
		label.Code = uint32(failure.GetCode())
	}
	// The host check reads the actual bytes; only the sanitized row is stored.
	stored := label
	if response := observation.GetResponse(); response != nil {
		data := response.GetValue()
		label.RawHMAC = c.digest.of(data)
		stored.RawHMAC = label.RawHMAC
		if novelty, bound := c.bindings.For(method); bound {
			label.Response = hex.EncodeToString(data)
			label.Request = hex.EncodeToString(observation.GetRequest().GetValue())
			plan := novelty.Evaluator().Plan()
			for _, part := range []struct {
				name, data string
				out        *string
			}{{plan.Input, label.Request, &stored.Request}, {plan.Output, label.Response, &stored.Response}} {
				message, err := decode(part.name, part.data)
				if err != nil {
					return fmt.Errorf("observation %s: %w", part.name, err)
				}
				clean, err := sanitize(message, c.claims)
				if err != nil {
					return err
				}
				*part.out = hex.EncodeToString(clean)
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = append(c.rows, stored)
	// An empty acknowledgment encodes to no bytes but is still a response to check.
	if _, bound := c.bindings.For(method); !bound || label.Code != 0 || observation.GetResponse() == nil {
		return nil
	}
	receipt := custody(c.receipts, label)
	if result := c.classify(label, receipt); judged.Explained && result.correlation != "" {
		return fmt.Errorf("guest explained %s but the host refused: %s", method, result.correlation)
	}
	return nil
}

// custody returns the receipt a row is checked under. The first successful Start of a harness
// case and seed establishes the run and holds the receipt for every later row of that scope,
// later Starts included; that first Start is checked against the run it established, so its
// run_id Echo only proves a nonempty identity. A bound row with no receipt refuses.
func custody(receipts map[[3]string]string, row Row) relation.Receipt {
	scope := [3]string{row.Harness, row.Case, fmt.Sprint(row.Seed)}
	if _, held := receipts[scope]; !held && row.Method == startMethod {
		if run, ok := startRun(row.Response); ok {
			receipts[scope] = run
		}
	}
	if run, ok := receipts[scope]; ok {
		return Receipt(run)
	}
	return nil
}

func trimSlash(method string) string {
	if len(method) > 0 && method[0] == '/' {
		return method[1:]
	}
	return method
}

func startRun(responseHex string) (string, bool) {
	data, err := hex.DecodeString(responseHex)
	if err != nil {
		return "", false
	}
	response := new(workflowservice.StartActivityExecutionResponse)
	if proto.Unmarshal(data, response) != nil || response.GetRunId() == "" {
		return "", false
	}
	return response.GetRunId(), true
}

type outcome struct {
	key         string
	correlation string
	projection  string
}

func (c *Collector) classify(row Row, receipt relation.Receipt) outcome {
	novelty, ok := c.bindings.For(row.Method)
	if !ok {
		return outcome{}
	}
	plan := novelty.Evaluator().Plan()
	request, requestErr := decode(plan.Input, row.Request)
	response, responseErr := decode(plan.Output, row.Response)
	if requestErr != nil || responseErr != nil {
		return outcome{projection: "undecodable"}
	}
	data, _ := hex.DecodeString(row.Response)
	_, key, err := novelty.CheckAndProject(relation.Context{Request: request, Response: response, Receipt: receipt}, data)
	var check *relation.CheckError
	var refusal *relation.NoveltyRefusal
	switch {
	case err == nil:
		return outcome{key: key.Hex()}
	case errors.As(err, &check):
		return outcome{correlation: check.Class + "/" + check.Reason}
	case errors.As(err, &refusal):
		return outcome{projection: string(refusal.Reason)}
	default:
		return outcome{projection: "error"}
	}
}

func decode(name, data string) (proto.Message, error) {
	kind, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	bytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	message := kind.New().Interface()
	return message, proto.Unmarshal(bytes, message)
}

// MethodCounts are the per-RPC counts of one corpus; every attempted row lands in exactly one of
// Excluded, RPCError, NotApplicable, Refused, RefusedProjection or Success.
type MethodCounts struct {
	Attempted         int            `json:"attempted"`
	Excluded          int            `json:"excluded"`
	RPCError          int            `json:"rpc_error"`
	NotApplicable     int            `json:"not_applicable"`
	Refused           int            `json:"refused_correlation"`
	RefusedProjection int            `json:"refused_projection"`
	Success           int            `json:"success"`
	RawDistinct       int            `json:"raw_distinct"`
	ProjectedDistinct int            `json:"projected_distinct"`
	Reasons           map[string]int `json:"reasons,omitempty"`
}

// Report holds per-RPC and total counts over one corpus.
type Report struct {
	Format  string                   `json:"format"`
	Corpus  string                   `json:"corpus_sha256"`
	Methods map[string]*MethodCounts `json:"methods"`
	Total   MethodCounts             `json:"total"`
}

// Count classifies every row in order under the receipt custody of custody. Raw and projected
// distinct counts share one denominator, the successfully projected rows.
func (c *Collector) Count(rows []Row, corpusSHA256 string) Report {
	report := Report{Format: "umpire.temporal.novelty-report.v1", Corpus: corpusSHA256, Methods: map[string]*MethodCounts{}}
	receipts := map[[3]string]string{}
	raw := map[string]map[string]bool{}
	projected := map[string]map[string]bool{}
	for _, row := range rows {
		counts := report.Methods[row.Method]
		if counts == nil {
			counts = &MethodCounts{Reasons: map[string]int{}}
			report.Methods[row.Method] = counts
			raw[row.Method], projected[row.Method] = map[string]bool{}, map[string]bool{}
		}
		counts.Attempted++
		_, bound := c.bindings.For(row.Method)
		switch {
		case row.Excluded != "":
			counts.Excluded++
			counts.Reasons["excluded:"+row.Excluded]++
			continue
		case row.Code != 0:
			counts.RPCError++
			continue
		case !bound:
			counts.NotApplicable++
			continue
		}
		result := c.classify(row, custody(receipts, row))
		switch {
		case result.correlation != "":
			counts.Refused++
			counts.Reasons["correlation:"+result.correlation]++
		case result.projection != "":
			counts.RefusedProjection++
			counts.Reasons["projection:"+result.projection]++
		default:
			counts.Success++
			raw[row.Method][row.RawHMAC] = true
			projected[row.Method][result.key] = true
		}
	}
	for method, counts := range report.Methods {
		counts.RawDistinct, counts.ProjectedDistinct = len(raw[method]), len(projected[method])
		if len(counts.Reasons) == 0 {
			counts.Reasons = nil
		}
		total := &report.Total
		total.Attempted += counts.Attempted
		total.Excluded += counts.Excluded
		total.RPCError += counts.RPCError
		total.NotApplicable += counts.NotApplicable
		total.Refused += counts.Refused
		total.RefusedProjection += counts.RefusedProjection
		total.Success += counts.Success
		total.RawDistinct += counts.RawDistinct
		total.ProjectedDistinct += counts.ProjectedDistinct
	}
	return report
}

// WriteCorpus writes rows as JSON lines and returns the corpus SHA-256.
func WriteCorpus(w io.Writer, rows []Row) (string, error) {
	hash := sha256.New()
	out := io.MultiWriter(w, hash)
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			return "", err
		}
		if _, err := out.Write(append(line, '\n')); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ReadCorpus reads rows written by WriteCorpus and returns them with the corpus SHA-256.
func ReadCorpus(r io.Reader) ([]Row, string, error) {
	hash := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(r, hash))
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	var rows []Row
	for scanner.Scan() {
		var row Row
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&row); err != nil {
			return nil, "", err
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return rows, hex.EncodeToString(hash.Sum(nil)), nil
}
