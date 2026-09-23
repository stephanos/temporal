// Package policy is the production canary's fixed policy: which Case it runs and under which
// Profiles, the authority it runs with, the digests of the target it may touch, the lease it holds,
// the ref and workflow it trusts, and its Limits. The policy is data under tools/canary, never an
// Umpire type, and no flag or environment variable changes it in the untagged build.
//
// The coordinate digests are the operator's to commit: the repository cannot know production's
// names, so the committed policy starts with each coordinate `unconfigured`, which preflight
// refuses. Each digest is the hex SHA-256 of the whole environment value it stands for.
package policy

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Version is the policy format this package reads.
const Version = 1

// Unconfigured is a coordinate the operator has not committed a digest for yet.
const Unconfigured = "unconfigured"

// The authority classes a canary runs under: the protected workflow, or the harness build.
const (
	AuthorityProtectedWorkflow = "protected-workflow"
	AuthorityHarness           = "harness"
)

// AuthorityClasses is the one set of authority classes, which the provenance decoder also uses;
// each call returns a fresh copy, so no importer can widen it.
func AuthorityClasses() []string {
	return []string{AuthorityProtectedWorkflow, AuthorityHarness}
}

// The ceilings each limit is held under, so a limit is always a sane duration and never overflows.
const (
	maxIterations    = 16
	maxBoundSeconds  = 7 * 24 * 60 * 60
	maxLeaseSeconds  = 30 * 24 * 60 * 60
	maxProgressBytes = 16 << 20
)

// Coordinates are the digests of the target's names, each the hex SHA-256 of the whole
// environment value (the gRPC one is the dial target, `host:port`), or Unconfigured.
type Coordinates struct {
	GRPC          string `json:"grpc"`
	Namespace     string `json:"namespace"`
	TaskQueue     string `json:"taskQueue"`
	HandlerQueue  string `json:"handlerQueue"`
	NexusEndpoint string `json:"nexusEndpoint"`
}

// Lease names the lease workflow: its fixed ID, its type, and a task queue no worker polls.
type Lease struct {
	WorkflowID   string `json:"workflowId"`
	WorkflowType string `json:"workflowType"`
	TaskQueue    string `json:"taskQueue"`
}

// Limits are the canary's own bounds; a Run's own duration and events are the Temporal Profile's,
// and the recorded-Run and receipt caps are fn-26's.
type Limits struct {
	Iterations             int `json:"iterations"`
	InvocationSeconds      int `json:"invocationSeconds"`
	CleanupReserveSeconds  int `json:"cleanupReserveSeconds"`
	LeaseRunTimeoutSeconds int `json:"leaseRunTimeoutSeconds"`
	ProgressBytes          int `json:"progressBytes"`
}

// Invocation is the bound on one `run`, from its start.
func (l Limits) Invocation() time.Duration { return time.Duration(l.InvocationSeconds) * time.Second }

// CleanupReserve is the bound on cleanup, under its own context.
func (l Limits) CleanupReserve() time.Duration {
	return time.Duration(l.CleanupReserveSeconds) * time.Second
}

// LeaseRunTimeout is the lease run's server-side timeout, the backstop for a lost process.
func (l Limits) LeaseRunTimeout() time.Duration {
	return time.Duration(l.LeaseRunTimeoutSeconds) * time.Second
}

// Policy is one decoded, validated canary policy.
type Policy struct {
	Version           int         `json:"version"`
	CaseIdentity      string      `json:"caseIdentity"`
	CaseProfile       string      `json:"caseProfile"`
	EvaluationProfile string      `json:"evaluationProfile"`
	AuthorityClass    string      `json:"authorityClass"`
	Coordinates       Coordinates `json:"coordinates"`
	Lease             Lease       `json:"lease"`
	Repository        string      `json:"repository"`
	TrustedRef        string      `json:"trustedRef"`
	WorkflowPath      string      `json:"workflowPath"`
	Limits            Limits      `json:"limits"`
}

//go:embed production-canary.json
var embedded []byte

// Embedded is the committed policy, the untagged build's only policy.
func Embedded() (*Policy, error) {
	return Decode(embedded)
}

// Digest is the hex SHA-256 of a coordinate's whole value, as the policy records it.
func Digest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func isDigest(value string) bool {
	return len(value) == sha256.Size*2 && strings.Trim(value, "0123456789abcdef") == ""
}

// Configured reports whether every coordinate has a committed digest.
func (p *Policy) Configured() bool {
	for _, coordinate := range p.Coordinates.named() {
		if coordinate.value == Unconfigured {
			return false
		}
	}
	return true
}

type namedCoordinate struct{ name, value string }

// named is the one list of coordinates, in the policy's order, with their JSON names.
func (c Coordinates) named() []namedCoordinate {
	return []namedCoordinate{
		{"grpc", c.GRPC}, {"namespace", c.Namespace},
		{"taskQueue", c.TaskQueue}, {"handlerQueue", c.HandlerQueue},
		{"nexusEndpoint", c.NexusEndpoint},
	}
}

// Mismatch names the first coordinate whose digest differs from other's, or reports that none does.
func (c Coordinates) Mismatch(other Coordinates) (string, bool) {
	theirs := other.named()
	for index, coordinate := range c.named() {
		if coordinate.value != theirs[index].value {
			return coordinate.name, true
		}
	}
	return "", false
}

// DigestsOf is the policy form of raw coordinates: each one's Digest.
func DigestsOf(grpc, namespace, taskQueue, handlerQueue, nexusEndpoint string) Coordinates {
	return Coordinates{
		GRPC: Digest(grpc), Namespace: Digest(namespace), TaskQueue: Digest(taskQueue),
		HandlerQueue: Digest(handlerQueue), NexusEndpoint: Digest(nexusEndpoint),
	}
}

// Decode reads a policy strictly: every field present and valid, and the bytes exactly the
// two-space-indented rendering of what they decode to, so an unknown, repeated or case-folded key,
// another order or other spacing is refused.
func Decode(encoded []byte) (*Policy, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var policy Policy
	if err := decoder.Decode(&policy); err != nil {
		return nil, fmt.Errorf("decode canary policy: %w", err)
	}
	if err := policy.validate(); err != nil {
		return nil, err
	}
	canonical, err := json.MarshalIndent(&policy, "", "  ")
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(append(canonical, '\n'), encoded) {
		return nil, errors.New("the canary policy is not in its canonical form")
	}
	return &policy, nil
}

func (p *Policy) validate() error {
	if p.Version != Version {
		return fmt.Errorf("canary policy format version %d, not %d", p.Version, Version)
	}
	if !isDigest(p.CaseIdentity) {
		return fmt.Errorf("caseIdentity %q is not a hex SHA-256", p.CaseIdentity)
	}
	for _, field := range []struct{ name, value string }{
		{"caseProfile", p.CaseProfile}, {"evaluationProfile", p.EvaluationProfile},
		{"lease.workflowId", p.Lease.WorkflowID}, {"lease.workflowType", p.Lease.WorkflowType},
		{"lease.taskQueue", p.Lease.TaskQueue}, {"repository", p.Repository},
	} {
		if field.value == "" {
			return fmt.Errorf("%s is missing", field.name)
		}
	}
	if !slices.Contains(AuthorityClasses(), p.AuthorityClass) {
		return fmt.Errorf("authorityClass %q is not one of %s", p.AuthorityClass, strings.Join(AuthorityClasses(), ", "))
	}
	for _, coordinate := range p.Coordinates.named() {
		if coordinate.value != Unconfigured && !isDigest(coordinate.value) {
			return fmt.Errorf("coordinate %s is neither a hex SHA-256 nor %q", coordinate.name, Unconfigured)
		}
	}
	if owner, name, ok := strings.Cut(p.Repository, "/"); !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return fmt.Errorf("repository %q is not owner/name", p.Repository)
	}
	if !strings.HasPrefix(p.TrustedRef, "refs/heads/") || p.TrustedRef == "refs/heads/" {
		return fmt.Errorf("trustedRef %q is not a branch ref", p.TrustedRef)
	}
	if !strings.HasPrefix(p.WorkflowPath, ".github/workflows/") || !strings.HasSuffix(p.WorkflowPath, ".yml") {
		return fmt.Errorf("workflowPath %q is not a workflow file", p.WorkflowPath)
	}
	for _, limit := range []struct {
		name       string
		value, max int
	}{
		{"iterations", p.Limits.Iterations, maxIterations},
		{"invocationSeconds", p.Limits.InvocationSeconds, maxBoundSeconds},
		{"cleanupReserveSeconds", p.Limits.CleanupReserveSeconds, maxBoundSeconds},
		{"leaseRunTimeoutSeconds", p.Limits.LeaseRunTimeoutSeconds, maxLeaseSeconds},
		{"progressBytes", p.Limits.ProgressBytes, maxProgressBytes},
	} {
		if limit.value <= 0 {
			return fmt.Errorf("limit %s must be positive, is %d", limit.name, limit.value)
		}
		if limit.value > limit.max {
			return fmt.Errorf("limit %s must be at most %d, is %d", limit.name, limit.max, limit.value)
		}
	}
	if p.Limits.LeaseRunTimeoutSeconds <= p.Limits.InvocationSeconds+p.Limits.CleanupReserveSeconds {
		return fmt.Errorf("leaseRunTimeoutSeconds %d must exceed the invocation limit plus the cleanup reserve (%d)",
			p.Limits.LeaseRunTimeoutSeconds, p.Limits.InvocationSeconds+p.Limits.CleanupReserveSeconds)
	}
	return nil
}
