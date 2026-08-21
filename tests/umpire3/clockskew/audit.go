package clockskew

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tests/umpire3/evidence"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const FormatVersion = "umpire3/clock-skew-audit/v1"

type Report struct {
	FormatVersion         string   `json:"formatVersion"`
	FaultLifecycle        []string `json:"faultLifecycle"`
	ClockDomains          []string `json:"clockDomains"`
	InvertedTimestamps    bool     `json:"invertedTimestamps"`
	CausalOrderAccepted   bool     `json:"causalOrderAccepted"`
	TimestampOnlyRejected bool     `json:"timestampOnlyRejected"`
	CausalGraphDigest     string   `json:"causalGraphDigest"`
	NegativeGraphDigest   string   `json:"negativeGraphDigest"`
	ArtifactDigest        string   `json:"artifactDigest"`
}

func RunAudit() (Report, error) {
	realizer := &auditRealizer{}
	term := umpire3fault.Term{
		Kind: protocol.FaultKindClockSkew,
		Scope: umpire3fault.Scope{
			Namespaces: []string{"isolated-namespace"}, Participants: []string{"worker-b"},
		},
		Occurrence: umpire3fault.Occurrence{First: 1, Count: 1},
		Interval:   umpire3fault.Interval{Start: 1, Stop: 2},
	}
	if err := umpire3fault.Run(context.Background(), term, realizer, umpire3fault.Options{
		Capabilities:    []protocol.CapabilityID{protocol.CapabilityIDFaultClock},
		AllowRestricted: true,
		CleanupTimeout:  time.Second,
	}, func(context.Context) error { return nil }); err != nil {
		return Report{}, fmt.Errorf("execute clock-skew fault: %w", err)
	}
	causal := skewedGraph(true)
	causalOrder, err := causal.Before("before", "after")
	if err != nil {
		return Report{}, fmt.Errorf("evaluate causal clock-skew order: %w", err)
	}
	causalJSON, err := causal.CanonicalJSON()
	if err != nil {
		return Report{}, err
	}
	negative := skewedGraph(false)
	timestampOrder, err := negative.Before("before", "after")
	if err != nil {
		return Report{}, fmt.Errorf("evaluate timestamp-only clock-skew order: %w", err)
	}
	negativeJSON, err := negative.CanonicalJSON()
	if err != nil {
		return Report{}, err
	}
	report := Report{
		FormatVersion:      FormatVersion,
		FaultLifecycle:     append([]string(nil), realizer.calls...),
		ClockDomains:       []string{"clock-a", "clock-b"},
		InvertedTimestamps: true, CausalOrderAccepted: causalOrder,
		TimestampOnlyRejected: !timestampOrder,
		CausalGraphDigest:     digest(causalJSON), NegativeGraphDigest: digest(negativeJSON),
	}
	report.ArtifactDigest, err = report.computedDigest()
	if err != nil {
		return Report{}, err
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) Validate() error {
	if r.FormatVersion != FormatVersion ||
		!slices.Equal(r.FaultLifecycle, []string{"install", "activate", "release", "cleanup"}) ||
		!slices.Equal(r.ClockDomains, []string{"clock-a", "clock-b"}) {
		return errors.New("clock-skew audit requires a complete isolated fault lifecycle and two clock domains")
	}
	if !r.InvertedTimestamps || !r.CausalOrderAccepted || !r.TimestampOnlyRejected {
		return errors.New("clock-skew audit must accept causal order and reject timestamp-only order")
	}
	if !validDigest(r.CausalGraphDigest) || !validDigest(r.NegativeGraphDigest) ||
		r.CausalGraphDigest == r.NegativeGraphDigest {
		return errors.New("clock-skew audit requires distinct causal and negative graph digests")
	}
	expected, err := r.computedDigest()
	if err != nil {
		return err
	}
	if r.ArtifactDigest != expected {
		return errors.New("clock-skew audit digest does not match its contents")
	}
	return nil
}

func skewedGraph(causal bool) evidence.Graph {
	afterReferences := []string(nil)
	if causal {
		afterReferences = []string{"event-a"}
	}
	return evidence.Graph{
		FormatVersion: evidence.FormatVersion,
		Facts: []evidence.Fact{
			{
				Identifier: "before", Kind: "distributed-step", Value: true,
				SourceIdentity: "worker-a", ClockDomain: "clock-a", SourceSequence: 10,
				ObservedAtUnixNano: 2_000, Reference: "event-a", EntityIdentity: "workflow",
				Lineage: []string{"namespace", "workflow"},
			},
			{
				Identifier: "after", Kind: "distributed-step", Value: true,
				SourceIdentity: "worker-b", ClockDomain: "clock-b", SourceSequence: 1,
				ObservedAtUnixNano: 1_000, Reference: "event-b", CausalReferences: afterReferences,
				EntityIdentity: "workflow", Lineage: []string{"namespace", "workflow"},
			},
		},
	}
}

type auditRealizer struct {
	calls []string
}

func (r *auditRealizer) Install(context.Context, umpire3fault.Term) (string, error) {
	r.calls = append(r.calls, "install")
	return "clock-skew-handle", nil
}

func (r *auditRealizer) Activate(context.Context, string) error {
	r.calls = append(r.calls, "activate")
	return nil
}

func (r *auditRealizer) Release(context.Context, string) error {
	r.calls = append(r.calls, "release")
	return nil
}

func (r *auditRealizer) Cleanup(context.Context, string) error {
	r.calls = append(r.calls, "cleanup")
	return nil
}

func (r Report) computedDigest() (string, error) {
	r.ArtifactDigest = ""
	encoded, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("encode clock-skew audit: %w", err)
	}
	return digest(encoded), nil
}

func digest(value []byte) string {
	encoded := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(encoded[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
