// Package recordedrun is the recorded Run: the file shape every writer of a closed Run shares, and
// the checks that replay admission and qualification admission both make of it before either
// decides anything of its own. A recorded Run carries the Run with its Verdict, the Profile
// identity it was prepared under, and the identity of the canonical Case it was prepared from,
// none of which the Run proto itself carries.
//
// It is a leaf: it imports the Case Runtime's public types and the canonical Case form, never the
// replay bridge, a Driver or a deployment, so an admission that must not execute imports it freely.
// It is public, outside tools/umpire/internal, so code beyond Umpire -- the canary -- computes a
// Case's identity and encodes a recorded Run exactly as Umpire does.
package recordedrun

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/casefile"
	"google.golang.org/protobuf/encoding/protojson"
)

// Identity is the Profile identity a Run was prepared under: the Profile's name, the catalog
// fingerprint and the environment-binding fingerprint, none of them secret.
type Identity struct {
	Profile  string `json:"profile"`
	Catalog  string `json:"catalog"`
	Bindings string `json:"bindings"`
}

// Record is one closed Run with the identity of the canonical Case it ran and the Profile identity
// it was prepared under, as umpire-run --record, umpire-fuzz --record-root and the live suite
// write it: a local file, not an artifact family.
type Record struct {
	// Case is the hex SHA-256 of the canonical Case bytes the Run was prepared from.
	Case     string          `json:"case"`
	Identity Identity        `json:"identity"`
	Run      json.RawMessage `json:"run"`
}

// Decoded is a recorded Run read back: the Case identity, the Profile identity and the Run.
type Decoded struct {
	Case   string
	Driver testpilot.DriverIdentity
	Run    *testpilotspb.Run
}

// ErrNoCase says a record names no Case: a record from before the Case identity was recorded,
// another format rather than a malformed one.
var ErrNoCase = errors.New("the recorded Run names no Case identity")

// CaseIdentity is the identity of a Case: the hex SHA-256 of its canonical bytes, recovered from
// the canonical or persisted form; any other form has no identity.
func CaseIdentity(input []byte) (string, error) {
	canonical, err := casefile.Canonical(input)
	if err != nil {
		return "", err
	}
	return Digest(canonical), nil
}

// Digest is the hex SHA-256 of bytes, the identity of a canonical Case or record.
func Digest(canonical []byte) string {
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}

func isDigest(value string) bool {
	return len(value) == sha256.Size*2 && strings.Trim(value, "0123456789abcdef") == ""
}

// Encode renders a closed Run with the Case identity and the Profile identity it was prepared
// under, one JSON document ended with a newline. The Run is compact ProtoJSON, so the same values
// always encode to the same bytes.
func Encode(caseIdentity string, identity testpilot.DriverIdentity, run *testpilotspb.Run) ([]byte, error) {
	if run == nil {
		return nil, errors.New("closed Run required")
	}
	if !isDigest(caseIdentity) {
		return nil, fmt.Errorf("case identity %q is not a hex SHA-256", caseIdentity)
	}
	encoded, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(run)
	if err != nil {
		return nil, fmt.Errorf("encode Run: %w", err)
	}
	// protojson's spacing is deliberately unstable; the record carries the compact form.
	var compact bytes.Buffer
	if err := json.Compact(&compact, encoded); err != nil {
		return nil, fmt.Errorf("compact Run: %w", err)
	}
	document, err := json.Marshal(Record{
		Case:     caseIdentity,
		Identity: Identity{Profile: identity.Profile, Catalog: identity.Catalog, Bindings: identity.Bindings},
		Run:      json.RawMessage(compact.Bytes()),
	})
	if err != nil {
		return nil, err
	}
	return append(document, '\n'), nil
}

// Decode reads a recorded Run strictly. Its outer and identity keys must be spelled exactly, once
// each: Go's decoder matches keys without case and keeps the last of a repeated key, so either
// would let two different documents decode to one record. An unknown field, a trailing document or
// a Run that does not decode is another protocol, not a Run to admit. A record without a Case
// identity is ErrNoCase, returned beside everything else it decoded, so a caller can still measure
// the Run before it decides.
func Decode(document []byte) (Decoded, error) {
	if len(document) == 0 {
		return Decoded{}, errors.New("recorded Run is required")
	}
	fields, err := exactObject(document, "case", "identity", "run")
	if err != nil {
		return Decoded{}, fmt.Errorf("decode recorded Run: %w", err)
	}
	identityFields, ok := fields["identity"]
	if !ok {
		return Decoded{}, errors.New("decode recorded Run: no identity")
	}
	identity, err := exactObject(identityFields, "profile", "catalog", "bindings")
	if err != nil {
		return Decoded{}, fmt.Errorf("decode recorded Run identity: %w", err)
	}
	var driver testpilot.DriverIdentity
	for name, target := range map[string]*string{"profile": &driver.Profile, "catalog": &driver.Catalog, "bindings": &driver.Bindings} {
		if raw, ok := identity[name]; ok {
			if err := json.Unmarshal(raw, target); err != nil {
				return Decoded{}, fmt.Errorf("decode recorded Run identity %s: %w", name, err)
			}
		}
	}
	runFields, ok := fields["run"]
	if !ok {
		return Decoded{}, errors.New("recorded Run carries no Run")
	}
	run := new(testpilotspb.Run)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(runFields, run); err != nil {
		return Decoded{}, fmt.Errorf("decode recorded Run: %w", err)
	}
	caseField, ok := fields["case"]
	if !ok {
		return Decoded{Driver: driver, Run: run}, ErrNoCase
	}
	var caseIdentity string
	if err := json.Unmarshal(caseField, &caseIdentity); err != nil {
		return Decoded{}, fmt.Errorf("decode recorded Run case: %w", err)
	}
	if !isDigest(caseIdentity) {
		return Decoded{}, fmt.Errorf("decode recorded Run: case identity %q is not a hex SHA-256", caseIdentity)
	}
	return Decoded{Case: caseIdentity, Driver: driver, Run: run}, nil
}

// exactObject reads one JSON object and nothing after it, whose keys are among allowed, spelled
// exactly and each present at most once.
func exactObject(document []byte, allowed ...string) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(document))
	open, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := open.(json.Delim); !ok || delimiter != '{' {
		return nil, errors.New("not a JSON object")
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("an object key is not a string")
		}
		known := false
		for _, name := range allowed {
			known = known || name == key
		}
		if !known {
			return nil, fmt.Errorf("unknown field %q", key)
		}
		if _, repeated := fields[key]; repeated {
			return nil, fmt.Errorf("field %q appears twice", key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	// One document and nothing after it: a second document or trailing bytes are not a record.
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("bytes after the document")
	}
	return fields, nil
}

// Crossed names why a Run is not the Case's by its IDs, or returns "" when its Case and Program
// IDs are the Case's.
func Crossed(source *testpilotspb.Case, run *testpilotspb.Run) string {
	if run.GetCaseId() != source.GetCaseId() {
		return fmt.Sprintf("the Run names Case %q, the Case is %q", run.GetCaseId(), source.GetCaseId())
	}
	if run.GetProgramId() != source.GetProgram().GetProgramId() {
		return fmt.Sprintf("the Run names Program %q, the Case's is %q", run.GetProgramId(), source.GetProgram().GetProgramId())
	}
	return ""
}

// SupportProblem is what is wrong with a supporting sequence.
type SupportProblem int

const (
	// SupportUnknown: a sequence names no event of the Run.
	SupportUnknown SupportProblem = iota + 1
	// SupportRepeated: a sequence names one event twice.
	SupportRepeated
)

// SupportError says which supporting sequence of which owner is wrong, and how.
type SupportError struct {
	Problem  SupportProblem
	Owner    string
	Sequence int64
}

func (e *SupportError) Error() string {
	if e.Problem == SupportRepeated {
		return fmt.Sprintf("%s names supporting event %d twice", e.Owner, e.Sequence)
	}
	return fmt.Sprintf("%s names supporting event %d, which the Run does not carry", e.Owner, e.Sequence)
}

// CheckSupport requires every supporting sequence, of the Verdict and of each rule, to name one
// event of the Run, once.
func CheckSupport(run *testpilotspb.Run, verdict *testpilotspb.Verdict) *SupportError {
	check := func(owner string, sequences []int64) *SupportError {
		seen := map[int64]bool{}
		for _, sequence := range sequences {
			if sequence <= 0 || sequence > int64(len(run.GetEvents())) || run.GetEvents()[sequence-1].GetSequence() != sequence {
				return &SupportError{Problem: SupportUnknown, Owner: owner, Sequence: sequence}
			}
			if seen[sequence] {
				return &SupportError{Problem: SupportRepeated, Owner: owner, Sequence: sequence}
			}
			seen[sequence] = true
		}
		return nil
	}
	if problem := check("the Verdict", verdict.GetSupportingEventSequences()); problem != nil {
		return problem
	}
	for _, rule := range verdict.GetRules() {
		if problem := check("rule "+rule.GetRuleId(), rule.GetSupportingEventSequences()); problem != nil {
			return problem
		}
	}
	return nil
}

// Agreement decides whether a closed Run's disposition, its Verdict's status and its rules' statuses
// agree both ways, as the Monitor and the evaluator produce them: the Verdict is violated exactly
// when some rule is violated, and the Run is stopped by its Monitor exactly then; it is satisfied
// exactly when every rule is satisfied on a completed Run; otherwise it is inconclusive. A status
// left unspecified, or a rule still pending, agrees with nothing. The detail names the first
// disagreement.
func Agreement(run *testpilotspb.Run, verdict *testpilotspb.Verdict) (bool, string) {
	status := verdict.GetStatus()
	if status == testpilotspb.VERDICT_STATUS_UNSPECIFIED {
		return false, "the Verdict's status is unspecified"
	}
	anyViolated, allSatisfied := false, true
	for _, rule := range verdict.GetRules() {
		switch rule.GetStatus() {
		case testpilotspb.RULE_VERDICT_STATUS_UNSPECIFIED, testpilotspb.RULE_VERDICT_STATUS_PENDING:
			return false, fmt.Sprintf("rule %s is %s in a closed Verdict", rule.GetRuleId(), rule.GetStatus())
		case testpilotspb.RULE_VERDICT_STATUS_VIOLATED:
			anyViolated = true
			allSatisfied = false
		case testpilotspb.RULE_VERDICT_STATUS_SATISFIED:
		default:
			allSatisfied = false
		}
	}
	violated := status == testpilotspb.VERDICT_STATUS_VIOLATED
	if violated != anyViolated {
		return false, fmt.Sprintf("verdict %s beside rules of which violated: %t", status, anyViolated)
	}
	stopped := run.GetDisposition() == testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	if stopped != violated {
		return false, fmt.Sprintf("disposition %s beside verdict %s", run.GetDisposition(), status)
	}
	satisfied := run.GetDisposition() == testpilotspb.RUN_DISPOSITION_COMPLETED && allSatisfied
	switch {
	case violated:
		return true, ""
	case satisfied && status != testpilotspb.VERDICT_STATUS_SATISFIED:
		return false, fmt.Sprintf("verdict %s where every rule is satisfied on a completed Run", status)
	case !satisfied && status != testpilotspb.VERDICT_STATUS_INCONCLUSIVE:
		return false, fmt.Sprintf("verdict %s on disposition %s with a rule not satisfied", status, run.GetDisposition())
	default:
		return true, ""
	}
}
