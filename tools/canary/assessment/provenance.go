package assessment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// ProvenanceFormatVersion is the provenance format this package writes and reads.
const ProvenanceFormatVersion = 1

// MaxProvenanceBytes bounds one provenance document.
const MaxProvenanceBytes = 64 << 10

// The invocation's cleanup outcome, known because provenance is published after the cleanup
// attempt: the lease released, or uncertain with the lease held.
const (
	CleanupReleased  = "released"
	CleanupUncertain = "uncertain"
)

// Isolation is the one isolation statement a canary provenance makes: what the scope is, not a
// claim about the rest of production.
const Isolation = "a dedicated canary namespace, task queue, handler queue and Nexus endpoint; " +
	"one lease-fenced Run at a time; no customer traffic, deployment or configuration change"

// Provenance is what a canary receipt was produced under, beside the fn-26 receipt it names: the
// authority, the workflow context, the target as digests, the lease and fence, the invocation and
// iteration, the Limits, the iteration's recorded cleanup and the invocation's cleanup outcome, the
// isolation statement and every workflow the lease fenced. It carries no credential, raw
// coordinate or payload, and is never release evidence.
type Provenance struct {
	Version int `json:"version"`
	// Receipt is the fn-26 receipt identity this provenance accompanies.
	Receipt string `json:"receipt"`
	// EvaluationProfile is the Evaluation Profile's identity.
	EvaluationProfile string               `json:"evaluationProfile"`
	AuthorityClass    string               `json:"authorityClass"`
	Workflow          ProvenanceWorkflow   `json:"workflow"`
	Coordinates       policy.Coordinates   `json:"coordinates"`
	Lease             ProvenanceLease      `json:"lease"`
	Invocation        ProvenanceInvocation `json:"invocation"`
	Limits            policy.Limits        `json:"limits"`
	Cleanup           ProvenanceCleanup    `json:"cleanup"`
	Isolation         string               `json:"isolation"`
	// Fenced is every workflow ID the lease fenced, so they outlive the namespace's retention.
	Fenced             []string           `json:"fenced"`
	ReleaseEligibility releaseEligibility `json:"releaseEligibility"`
}

// ProvenanceWorkflow is the GitHub workflow run the invocation was: its workflow ref and run ID.
type ProvenanceWorkflow struct {
	Ref   string `json:"ref"`
	RunID string `json:"runId"`
}

// ProvenanceLease is the lease: its workflow ID's digest and the fence, the lease run's ID.
type ProvenanceLease struct {
	WorkflowIDDigest string `json:"workflowIdDigest"`
	Fence            string `json:"fence"`
}

// ProvenanceInvocation is the invocation, which iteration of it this was, and that iteration's Run.
type ProvenanceInvocation struct {
	ID        string `json:"id"`
	Iteration int    `json:"iteration"`
	RunID     string `json:"runId"`
}

// ProvenanceCleanup is the iteration's recorded cleanup status and the invocation's outcome.
type ProvenanceCleanup struct {
	Iteration  string `json:"iteration"`
	Invocation string `json:"invocation"`
}

// releaseEligibility is always false: it has no other value to hold, renders only as false, and
// decodes nothing else, so no provenance can be release evidence.
type releaseEligibility struct{}

func (releaseEligibility) MarshalJSON() ([]byte, error) { return []byte("false"), nil }

func (*releaseEligibility) UnmarshalJSON(encoded []byte) error {
	if string(encoded) != "false" {
		return fmt.Errorf("releaseEligibility is %s; a canary provenance is never release evidence", encoded)
	}
	return nil
}

func isHexDigest(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}

func (p *Provenance) validate() error {
	if p.Version != ProvenanceFormatVersion {
		return fmt.Errorf("provenance format version %d, not %d", p.Version, ProvenanceFormatVersion)
	}
	for _, check := range []func() error{p.validateIdentities, p.validateScope, p.validateIteration} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

// validateIdentities checks what the provenance names: the receipt, the Profile, the authority
// class and the workflow run.
func (p *Provenance) validateIdentities() error {
	if !isHexDigest(p.Receipt) {
		return fmt.Errorf("receipt %q is not a receipt identity", p.Receipt)
	}
	if identity, ok := strings.CutPrefix(p.EvaluationProfile, "sha256:"); !ok || !isHexDigest(identity) {
		return fmt.Errorf("evaluationProfile %q is not a Profile identity", p.EvaluationProfile)
	}
	if !slices.Contains(policy.AuthorityClasses(), p.AuthorityClass) {
		return fmt.Errorf("authorityClass %q is not one of %s", p.AuthorityClass, strings.Join(policy.AuthorityClasses(), ", "))
	}
	if repositoryAndPath, ref, ok := strings.Cut(p.Workflow.Ref, "@"); !ok || repositoryAndPath == "" || !strings.HasPrefix(ref, "refs/") {
		return fmt.Errorf("workflow ref %q is not <repository>/<path>@<ref>", p.Workflow.Ref)
	}
	if number, err := strconv.ParseUint(p.Workflow.RunID, 10, 64); err != nil || number == 0 || strconv.FormatUint(number, 10) != p.Workflow.RunID {
		return fmt.Errorf("workflow run ID %q is not a positive number", p.Workflow.RunID)
	}
	return nil
}

// validateScope checks the target and the lease are digests, the fence is named, the Limits are
// positive and the isolation statement is the canary's.
func (p *Provenance) validateScope() error {
	coordinates := p.Coordinates
	for name, digest := range map[string]string{
		"grpc": coordinates.GRPC, "namespace": coordinates.Namespace, "taskQueue": coordinates.TaskQueue,
		"handlerQueue": coordinates.HandlerQueue, "nexusEndpoint": coordinates.NexusEndpoint,
		"lease.workflowIdDigest": p.Lease.WorkflowIDDigest,
	} {
		if !isHexDigest(digest) {
			return fmt.Errorf("%s is not a digest", name)
		}
	}
	if p.Lease.Fence == "" {
		return errors.New("the provenance names no fence")
	}
	limits := p.Limits
	if limits.Iterations <= 0 || limits.InvocationSeconds <= 0 || limits.CleanupReserveSeconds <= 0 ||
		limits.LeaseRunTimeoutSeconds <= 0 || limits.ProgressBytes <= 0 {
		return errors.New("every limit is positive")
	}
	if p.Isolation != Isolation {
		return errors.New("the isolation statement is not the canary's")
	}
	return nil
}

// validateIteration checks the iteration against its invocation: its number within the limit, its
// Run among the fenced workflows, and both cleanups recorded ones.
func (p *Provenance) validateIteration() error {
	if p.Invocation.ID == "" || p.Invocation.RunID == "" {
		return errors.New("the provenance needs the invocation ID and the iteration's Run ID")
	}
	iterations := p.Limits.Iterations
	if p.Invocation.Iteration < 1 || p.Invocation.Iteration > iterations {
		return fmt.Errorf("iteration %d is not one of the invocation's %d", p.Invocation.Iteration, iterations)
	}
	status, declared := testpilotspb.CleanupStatus_value[p.Cleanup.Iteration]
	if !declared || status == int32(testpilotspb.CLEANUP_STATUS_UNSPECIFIED) {
		return fmt.Errorf("the iteration's cleanup %q is not a recorded cleanup status", p.Cleanup.Iteration)
	}
	if p.Cleanup.Invocation != CleanupReleased && p.Cleanup.Invocation != CleanupUncertain {
		return fmt.Errorf("the invocation's cleanup %q is neither %s nor %s", p.Cleanup.Invocation, CleanupReleased, CleanupUncertain)
	}
	if !slices.Contains(p.Fenced, p.Invocation.RunID) {
		return errors.New("the iteration's Run is not one the lease fenced")
	}
	seen := map[string]bool{}
	for _, id := range p.Fenced {
		if id == "" || seen[id] {
			return errors.New("the fenced workflow IDs are not distinct and non-empty")
		}
		seen[id] = true
	}
	if len(p.Fenced) > iterations {
		return fmt.Errorf("the lease fenced %d workflows, more than the invocation's %d iterations", len(p.Fenced), iterations)
	}
	return nil
}

func checkProvenanceSize(size int) error {
	if size > MaxProvenanceBytes {
		return fmt.Errorf("the provenance is %d bytes, over the %d-byte cap", size, MaxProvenanceBytes)
	}
	return nil
}

func encodeProvenance(provenance *Provenance) ([]byte, error) {
	var rendered bytes.Buffer
	encoder := json.NewEncoder(&rendered)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(provenance); err != nil {
		return nil, err
	}
	return rendered.Bytes(), nil
}

// RenderProvenance is the canonical provenance: one compact JSON document in the fixed key order
// with one trailing newline, refused when invalid or over the cap.
func RenderProvenance(provenance *Provenance) ([]byte, error) {
	if provenance == nil {
		return nil, errors.New("no provenance")
	}
	if err := provenance.validate(); err != nil {
		return nil, err
	}
	rendered, err := encodeProvenance(provenance)
	if err != nil {
		return nil, err
	}
	if err := checkProvenanceSize(len(rendered)); err != nil {
		return nil, err
	}
	return rendered, nil
}

// ProvenanceIdentity is the hex SHA-256 of a provenance's bytes, the name it is published under.
func ProvenanceIdentity(rendered []byte) string {
	return recordedrun.Digest(rendered)
}

// DecodeProvenance reads a provenance back strictly: at most the cap, this format version, valid,
// and exactly the canonical rendering of what it decodes to, so an unknown, repeated or
// case-folded key, a null list, a trailing document or other spacing is refused.
func DecodeProvenance(encoded []byte) (*Provenance, error) {
	if err := checkProvenanceSize(len(encoded)); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var provenance Provenance
	if err := decoder.Decode(&provenance); err != nil {
		return nil, fmt.Errorf("decode provenance: %w", err)
	}
	if err := provenance.validate(); err != nil {
		return nil, err
	}
	canonical, err := encodeProvenance(&provenance)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, encoded) {
		return nil, errors.New("the provenance is not in its canonical form")
	}
	return &provenance, nil
}
