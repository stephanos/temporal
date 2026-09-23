package assessment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

// testpilotRunPrefix begins every Run ID Testpilot chooses, and so every workflow ID the fence
// names, since the canary Case's workflow ID is its Run ID.
const testpilotRunPrefix = "testpilot.run."

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
	if !policy.IsDigest(p.Receipt) {
		return fmt.Errorf("receipt %q is not a receipt identity", p.Receipt)
	}
	if identity, ok := strings.CutPrefix(p.EvaluationProfile, "sha256:"); !ok || !policy.IsDigest(identity) {
		return fmt.Errorf("evaluationProfile %q is not a Profile identity", p.EvaluationProfile)
	}
	if !slices.Contains(policy.AuthorityClasses(), p.AuthorityClass) {
		return fmt.Errorf("authorityClass %q is not one of %s", p.AuthorityClass, strings.Join(policy.AuthorityClasses(), ", "))
	}
	repositoryAndPath, ref, ok := strings.Cut(p.Workflow.Ref, "@")
	repository, path, _ := strings.Cut(repositoryAndPath, "/.github/workflows/")
	if !ok || repository == "" || !strings.HasSuffix(path, ".yml") || strings.Contains(path, "/") ||
		!strings.HasPrefix(ref, "refs/heads/") || ref == "refs/heads/" {
		return fmt.Errorf("workflow ref %q is not <repository>/.github/workflows/<file>.yml@<branch ref>", p.Workflow.Ref)
	}
	if !positiveNumber(p.Workflow.RunID) {
		return fmt.Errorf("workflow run ID %q is not a positive number", p.Workflow.RunID)
	}
	// The invocation is the workflow run's attempt, as preflight names it.
	if attempt, ok := strings.CutPrefix(p.Invocation.ID, p.Workflow.RunID+"-"); !ok || !positiveNumber(attempt) {
		return fmt.Errorf("invocation %q is not an attempt of workflow run %s", p.Invocation.ID, p.Workflow.RunID)
	}
	return nil
}

// canonicalUUID reports whether value is a UUID in its one canonical form: lower-case and dashed.
func canonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

// IsTestpilotRunID reports whether id is a Run ID as Testpilot chooses one: its prefix and a
// canonical UUID, and nothing else.
func IsTestpilotRunID(id string) bool {
	suffix, ok := strings.CutPrefix(id, testpilotRunPrefix)
	return ok && canonicalUUID(suffix)
}

// positiveNumber reports whether value is a positive decimal number written canonically.
func positiveNumber(value string) bool {
	number, err := strconv.ParseUint(value, 10, 64)
	return err == nil && number > 0 && strconv.FormatUint(number, 10) == value
}

// validateScope checks the target and the lease are digests, the fence is named, the Limits are
// positive and the isolation statement is the canary's.
func (p *Provenance) validateScope() error {
	coordinates := p.Coordinates
	for _, digest := range []struct{ name, value string }{
		{"grpc", coordinates.GRPC}, {"namespace", coordinates.Namespace}, {"taskQueue", coordinates.TaskQueue},
		{"handlerQueue", coordinates.HandlerQueue}, {"nexusEndpoint", coordinates.NexusEndpoint},
		{"lease.workflowIdDigest", p.Lease.WorkflowIDDigest},
	} {
		if !policy.IsDigest(digest.value) {
			return fmt.Errorf("%s is not a digest", digest.name)
		}
	}
	if !canonicalUUID(p.Lease.Fence) {
		return errors.New("the fence is not a lease run ID")
	}
	if err := p.Limits.Validate(); err != nil {
		return err
	}
	if p.Isolation != Isolation {
		return errors.New("the isolation statement is not the canary's")
	}
	return nil
}

// validateIteration checks the iteration against its invocation: its number within the limit, its
// Run the fence's in that position, and both cleanups recorded ones.
func (p *Provenance) validateIteration() error {
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
	// Each iteration fences its one Run before it opens it, and a fence that fails ends the
	// invocation, so iteration N's Run is the Nth the lease fenced.
	if len(p.Fenced) < p.Invocation.Iteration || p.Fenced[p.Invocation.Iteration-1] != p.Invocation.RunID {
		return fmt.Errorf("iteration %d's Run is not the lease's %d fenced workflow", p.Invocation.Iteration, p.Invocation.Iteration)
	}
	seen := map[string]bool{}
	for _, id := range p.Fenced {
		if !IsTestpilotRunID(id) || seen[id] {
			return errors.New("the fenced workflow IDs are not distinct Testpilot Run IDs")
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
