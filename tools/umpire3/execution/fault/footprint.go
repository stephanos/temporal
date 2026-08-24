package fault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

const FootprintFormatVersion = "umpire3/learned-footprint/v1"

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type CallRole string

const (
	CallRoleSetup       CallRole = "setup"
	CallRoleClientEntry CallRole = "client-entry"
	CallRoleInternal    CallRole = "internal"
	CallRoleEvidence    CallRole = "evidence"
)

type Call struct {
	Protocol         string    `json:"protocol"`
	Service          string    `json:"service"`
	Route            string    `json:"route"`
	Direction        Direction `json:"direction"`
	Role             CallRole  `json:"role"`
	Namespace        string    `json:"namespace"`
	Participant      string    `json:"participant"`
	Attempt          int       `json:"attempt"`
	Occurrence       int       `json:"occurrence"`
	Interval         Interval  `json:"interval"`
	CausalReferences []string  `json:"causalReferences,omitempty"`
	Risk             int       `json:"risk"`
}

type Recorder struct {
	mu          sync.Mutex
	calls       []Call
	occurrences map[string]int
}

type Report struct {
	FormatVersion        string      `json:"formatVersion"`
	Calls                []Call      `json:"calls"`
	Declared             []Footprint `json:"declared"`
	AllowedNoise         []Footprint `json:"allowedNoise,omitempty"`
	Drift                Drift       `json:"drift"`
	FootprintDigest      string      `json:"footprintDigest"`
	ReconciliationDigest string      `json:"reconciliationDigest"`
	Complete             bool        `json:"complete"`
}

type FootprintProvider interface {
	FootprintReport() (Report, error)
}

func NewRecorder() *Recorder {
	return &Recorder{occurrences: make(map[string]int)}
}

func (r *Recorder) Record(call Call) error {
	if r == nil {
		return errors.New("footprint recorder is required")
	}
	call = normalizeCall(call)
	if call.Protocol == "" || call.Service == "" || call.Route == "" {
		return errors.New("footprint call requires protocol, service, and route")
	}
	if call.Direction != DirectionInbound && call.Direction != DirectionOutbound {
		return errors.New("footprint call requires a known direction")
	}
	if call.Role != CallRoleSetup && call.Role != CallRoleClientEntry &&
		call.Role != CallRoleInternal && call.Role != CallRoleEvidence {
		return errors.New("footprint call requires a known role")
	}
	if call.Namespace == "" || call.Participant == "" {
		return errors.New("footprint call requires namespace and participant identity")
	}
	if call.Attempt <= 0 {
		call.Attempt = 1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := callIdentity(call)
	r.occurrences[key]++
	call.Occurrence = r.occurrences[key]
	if call.Interval.Start == 0 && call.Interval.Stop == 0 {
		call.Interval = Interval{Start: int64(len(r.calls) + 1), Stop: int64(len(r.calls) + 2)}
	}
	if call.Interval.Start < 0 || call.Interval.Stop <= call.Interval.Start {
		return errors.New("footprint call interval must be positive and bounded")
	}
	r.calls = append(r.calls, call)
	return nil
}

func (r *Recorder) Snapshot() []Call {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]Call, len(r.calls))
	for index, call := range r.calls {
		result[index] = cloneCall(call)
	}
	return result
}

func (r *Recorder) Digest() string {
	encoded, err := json.Marshal(canonicalCalls(r.Snapshot()))
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func BuildFootprintReport(declared []Footprint, observed []Call, allowed []Footprint) (Report, error) {
	if len(declared) == 0 {
		return Report{}, errors.New("declared footprint is required")
	}
	calls := make([]Call, len(observed))
	for index, call := range observed {
		call = normalizeCall(call)
		if err := validateCall(call); err != nil {
			return Report{}, fmt.Errorf("validate observed footprint call %d: %w", index, err)
		}
		calls[index] = call
	}
	if len(calls) == 0 {
		return Report{}, errors.New("observed footprint is required")
	}
	declared = canonicalFootprints(declared)
	allowed = canonicalFootprints(allowed)
	drift := ReconcileFootprints(declared, calls, allowed)
	footprintDigest, err := digestJSON(canonicalCalls(calls))
	if err != nil {
		return Report{}, err
	}
	reconciliationDigest, err := digestJSON(struct {
		Declared        []Footprint `json:"declared"`
		AllowedNoise    []Footprint `json:"allowedNoise,omitempty"`
		Drift           Drift       `json:"drift"`
		FootprintDigest string      `json:"footprintDigest"`
	}{declared, allowed, drift, footprintDigest})
	if err != nil {
		return Report{}, err
	}
	return Report{
		FormatVersion: FootprintFormatVersion, Calls: calls, Declared: declared, AllowedNoise: allowed,
		Drift: drift, FootprintDigest: footprintDigest, ReconciliationDigest: reconciliationDigest,
		Complete: len(drift.Missing) == 0 && len(drift.Unexpected) == 0,
	}, nil
}

func (r Report) Validate() error {
	if r.FormatVersion != FootprintFormatVersion {
		return fmt.Errorf("unsupported learned footprint format %q", r.FormatVersion)
	}
	rebuilt, err := BuildFootprintReport(r.Declared, r.Calls, r.AllowedNoise)
	if err != nil {
		return err
	}
	if rebuilt.FootprintDigest != r.FootprintDigest ||
		rebuilt.ReconciliationDigest != r.ReconciliationDigest ||
		rebuilt.Complete != r.Complete ||
		!slices.Equal(rebuilt.Drift.Missing, r.Drift.Missing) ||
		!slices.Equal(rebuilt.Drift.Unexpected, r.Drift.Unexpected) {
		return errors.New("learned footprint report does not match its normalized calls and declarations")
	}
	return nil
}

func (r Report) RequireComplete() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if !r.Complete {
		return fmt.Errorf("footprint reconciliation drift: missing=%v unexpected=%v", r.Drift.Missing, r.Drift.Unexpected)
	}
	return nil
}

type Drift struct {
	Missing    []Footprint `json:"missing,omitempty"`
	Unexpected []Footprint `json:"unexpected,omitempty"`
}

func FaultTargets(calls []Call, seed int64, limit int) []Footprint {
	footprints := make([]Footprint, 0, len(calls))
	for _, call := range calls {
		call = normalizeCall(call)
		if call.Role != CallRoleInternal {
			continue
		}
		footprints = append(footprints, Footprint{
			Protocol: call.Protocol, Service: call.Service, Route: call.Route,
			Occurrence: call.Occurrence, Risk: call.Risk,
		})
	}
	return SelectFootprints(footprints, seed, limit)
}

func ReconcileFootprints(declared []Footprint, observed []Call, allowed []Footprint) Drift {
	declaredByID := footprintSet(declared)
	allowedByID := footprintSet(allowed)
	observedByID := make(map[string]Footprint, len(observed))
	for _, call := range observed {
		call = normalizeCall(call)
		if call.Protocol == "" || call.Service == "" || call.Route == "" ||
			call.Role == CallRoleSetup || call.Role == CallRoleClientEntry || call.Role == CallRoleEvidence {
			continue
		}
		footprint := Footprint{Protocol: call.Protocol, Service: call.Service, Route: call.Route}
		observedByID[footprintIdentity(footprint)] = footprint
	}
	var drift Drift
	for identity, footprint := range declaredByID {
		if _, found := observedByID[identity]; !found {
			drift.Missing = append(drift.Missing, footprintWithoutSelection(footprint))
		}
	}
	for identity, footprint := range observedByID {
		if _, declared := declaredByID[identity]; declared {
			continue
		}
		if _, permitted := allowedByID[identity]; permitted {
			continue
		}
		drift.Unexpected = append(drift.Unexpected, footprint)
	}
	sortFootprints(drift.Missing)
	sortFootprints(drift.Unexpected)
	return drift
}

func normalizeCall(call Call) Call {
	call.Protocol = strings.ToLower(strings.TrimSpace(call.Protocol))
	call.Service = strings.TrimSpace(call.Service)
	call.Route = strings.TrimSpace(call.Route)
	if route, _, found := strings.Cut(call.Route, "?"); found {
		call.Route = route
	}
	call.Namespace = strings.TrimSpace(call.Namespace)
	call.Participant = strings.TrimSpace(call.Participant)
	slices.Sort(call.CausalReferences)
	call.CausalReferences = slices.Compact(call.CausalReferences)
	return call
}

func validateCall(call Call) error {
	if call.Protocol == "" || call.Service == "" || call.Route == "" {
		return errors.New("footprint call requires protocol, service, and route")
	}
	if call.Direction != DirectionInbound && call.Direction != DirectionOutbound {
		return errors.New("footprint call requires a known direction")
	}
	if call.Role != CallRoleSetup && call.Role != CallRoleClientEntry &&
		call.Role != CallRoleInternal && call.Role != CallRoleEvidence {
		return errors.New("footprint call requires a known role")
	}
	if call.Namespace == "" || call.Participant == "" {
		return errors.New("footprint call requires namespace and participant identity")
	}
	if call.Attempt <= 0 || call.Occurrence <= 0 {
		return errors.New("footprint call requires positive attempt and occurrence")
	}
	if call.Interval.Start < 0 || call.Interval.Stop <= call.Interval.Start {
		return errors.New("footprint call interval must be positive and bounded")
	}
	return nil
}

func cloneCall(call Call) Call {
	call.CausalReferences = append([]string(nil), call.CausalReferences...)
	return call
}

func callIdentity(call Call) string {
	return strings.Join([]string{
		call.Protocol, call.Service, call.Route, string(call.Direction), call.Namespace, call.Participant,
	}, "\x00")
}

func footprintSet(footprints []Footprint) map[string]Footprint {
	result := make(map[string]Footprint, len(footprints))
	for _, footprint := range footprints {
		footprint.Protocol = strings.ToLower(strings.TrimSpace(footprint.Protocol))
		footprint.Service = strings.TrimSpace(footprint.Service)
		footprint.Route = strings.TrimSpace(footprint.Route)
		if footprint.Protocol == "" || footprint.Service == "" || footprint.Route == "" {
			continue
		}
		footprint.Occurrence = 0
		result[footprintIdentity(footprint)] = footprint
	}
	return result
}

func canonicalFootprints(footprints []Footprint) []Footprint {
	set := footprintSet(footprints)
	result := make([]Footprint, 0, len(set))
	for _, footprint := range set {
		result = append(result, footprint)
	}
	sortFootprints(result)
	return result
}

type canonicalCall struct {
	Protocol   string    `json:"protocol"`
	Service    string    `json:"service"`
	Route      string    `json:"route"`
	Direction  Direction `json:"direction"`
	Role       CallRole  `json:"role"`
	Attempt    int       `json:"attempt"`
	Occurrence int       `json:"occurrence"`
	Risk       int       `json:"risk"`
}

func canonicalCalls(calls []Call) []canonicalCall {
	result := make([]canonicalCall, len(calls))
	for index, call := range calls {
		call = normalizeCall(call)
		result[index] = canonicalCall{
			Protocol: call.Protocol, Service: call.Service, Route: call.Route, Direction: call.Direction,
			Role: call.Role, Attempt: call.Attempt,
			Occurrence: call.Occurrence, Risk: call.Risk,
		}
	}
	slices.SortFunc(result, func(left, right canonicalCall) int {
		leftIdentity := strings.Join([]string{
			left.Protocol, left.Service, left.Route, string(left.Direction), string(left.Role),
			fmt.Sprint(left.Attempt), fmt.Sprint(left.Occurrence), fmt.Sprint(left.Risk),
		}, "\x00")
		rightIdentity := strings.Join([]string{
			right.Protocol, right.Service, right.Route, string(right.Direction), string(right.Role),
			fmt.Sprint(right.Attempt), fmt.Sprint(right.Occurrence), fmt.Sprint(right.Risk),
		}, "\x00")
		return strings.Compare(leftIdentity, rightIdentity)
	})
	return result
}

func digestJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode learned footprint digest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func footprintIdentity(footprint Footprint) string {
	return strings.Join([]string{footprint.Protocol, footprint.Service, footprint.Route}, "\x00")
}

func footprintWithoutSelection(footprint Footprint) Footprint {
	footprint.Risk = 0
	footprint.RealizationEvidence = false
	return footprint
}

func sortFootprints(footprints []Footprint) {
	slices.SortFunc(footprints, func(left, right Footprint) int {
		return strings.Compare(footprintIdentity(left), footprintIdentity(right))
	})
}
