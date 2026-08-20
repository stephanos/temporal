package fault

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type Scope struct {
	Namespaces   []string `json:"namespaces"`
	Endpoints    []string `json:"endpoints"`
	TaskQueues   []string `json:"taskQueues"`
	Services     []string `json:"services"`
	Routes       []string `json:"routes"`
	Participants []string `json:"participants"`
	Attempts     []int    `json:"attempts"`
}

type Occurrence struct {
	First int `json:"first"`
	Count int `json:"count"`
}

type Interval struct {
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

type Term struct {
	Kind       protocol.FaultKind `json:"kind"`
	Scope      Scope              `json:"scope"`
	Occurrence Occurrence         `json:"occurrence"`
	Interval   Interval           `json:"interval"`
}

type Options struct {
	Capabilities    []protocol.CapabilityID
	AllowRestricted bool
	CleanupTimeout  time.Duration
}

type Realizer interface {
	Install(context.Context, Term) (string, error)
	Activate(context.Context, string) error
	Release(context.Context, string) error
	Cleanup(context.Context, string) error
}

type Provider interface {
	FaultRealizer() Realizer
}

type RealizationEvidence struct {
	SourceIdentity string `json:"sourceIdentity"`
	Reference      string `json:"reference"`
	EntityIdentity string `json:"entityIdentity"`
}

type EvidenceProvider interface {
	RealizationEvidence(context.Context, string) (RealizationEvidence, error)
}

type Footprint struct {
	Protocol            string `json:"protocol"`
	Service             string `json:"service"`
	Route               string `json:"route"`
	Risk                int    `json:"risk"`
	RealizationEvidence bool   `json:"realizationEvidence"`
}

func Preflight(term Term, capabilities []protocol.CapabilityID, allowRestricted bool) error {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load fault catalog: %w", err)
	}
	var declaration *protocol.FaultDeclaration
	for index := range catalog.Faults {
		if protocol.FaultKind(catalog.Faults[index].Identifier) == term.Kind {
			declaration = &catalog.Faults[index]
			break
		}
	}
	if declaration == nil {
		return fmt.Errorf("unknown fault kind %q", term.Kind)
	}
	if declaration.SafetyClass == "restricted" && !allowRestricted {
		return fmt.Errorf("fault %q is restricted", term.Kind)
	}
	if err := term.validate(declaration.ScopeDimensions); err != nil {
		return err
	}
	have := make(map[protocol.CapabilityID]struct{}, len(capabilities))
	for _, capability := range capabilities {
		have[capability] = struct{}{}
	}
	var missing []protocol.CapabilityID
	for _, capability := range declaration.RequiredCapabilities {
		if _, exists := have[capability]; !exists {
			missing = append(missing, capability)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing capabilities: %v", missing)
	}
	return nil
}

func (t Term) validate(dimensions []string) error {
	if len(t.Scope.Namespaces) != 1 || t.Scope.Namespaces[0] == "" {
		return errors.New("fault requires exactly one isolated namespace")
	}
	if t.Occurrence.First <= 0 || t.Occurrence.Count <= 0 {
		return errors.New("fault occurrence requires positive first and count")
	}
	if t.Interval.Start < 0 || t.Interval.Start >= t.Interval.Stop {
		return errors.New("fault interval must be positive and bounded")
	}
	for _, dimension := range dimensions {
		var present bool
		switch dimension {
		case "namespace", "occurrence", "interval":
			present = true
		case "endpoint":
			present = nonEmpty(t.Scope.Endpoints)
		case "task-queue":
			present = nonEmpty(t.Scope.TaskQueues)
		case "service":
			present = nonEmpty(t.Scope.Services)
		case "route":
			present = nonEmpty(t.Scope.Routes)
		case "participant":
			present = nonEmpty(t.Scope.Participants)
		case "attempt":
			present = len(t.Scope.Attempts) != 0
		default:
			return fmt.Errorf("unknown fault scope dimension %q", dimension)
		}
		if !present {
			return fmt.Errorf("fault scope is missing %q", dimension)
		}
	}
	return nil
}

func Run(ctx context.Context, term Term, realizer Realizer, options Options, operation func(context.Context) error) (retErr error) {
	if realizer == nil || operation == nil {
		return errors.New("fault realizer and operation are required")
	}
	if options.CleanupTimeout <= 0 {
		return errors.New("positive independent cleanup timeout is required")
	}
	if err := Preflight(term, options.Capabilities, options.AllowRestricted); err != nil {
		return fmt.Errorf("preflight fault: %w", err)
	}
	handle, err := realizer.Install(ctx, term)
	if err != nil {
		return fmt.Errorf("install fault: %w", err)
	}
	activated := false
	defer func() {
		panicValue := recover()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), options.CleanupTimeout)
		defer cancel()
		var cleanupErrors []error
		if activated {
			if err := realizer.Release(cleanupCtx, handle); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("release fault: %w", err))
			}
		}
		if err := realizer.Cleanup(cleanupCtx, handle); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cleanup fault: %w", err))
		}
		retErr = errors.Join(retErr, errors.Join(cleanupErrors...))
		if panicValue != nil {
			panic(panicValue)
		}
	}()
	if err := realizer.Activate(ctx, handle); err != nil {
		return fmt.Errorf("activate fault: %w", err)
	}
	activated = true
	return operation(ctx)
}

func SelectFootprints(footprints []Footprint, seed int64, limit int) []Footprint {
	if limit <= 0 {
		return nil
	}
	unique := make(map[string]Footprint, len(footprints))
	for _, footprint := range footprints {
		if footprint.Protocol == "" || footprint.Service == "" || footprint.Route == "" {
			continue
		}
		footprint.RealizationEvidence = true
		key := footprint.Protocol + "\x00" + footprint.Service + "\x00" + footprint.Route
		if previous, exists := unique[key]; !exists || footprint.Risk > previous.Risk {
			unique[key] = footprint
		}
	}
	selected := make([]Footprint, 0, len(unique))
	for _, footprint := range unique {
		selected = append(selected, footprint)
	}
	slices.SortFunc(selected, func(left, right Footprint) int {
		if left.Risk != right.Risk {
			return right.Risk - left.Risk
		}
		leftHash := footprintOrder(left, seed)
		rightHash := footprintOrder(right, seed)
		if leftHash < rightHash {
			return -1
		}
		if leftHash > rightHash {
			return 1
		}
		return 0
	})
	return append([]Footprint(nil), selected[:min(limit, len(selected))]...)
}

func footprintOrder(footprint Footprint, seed int64) uint64 {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%s", seed, footprint.Protocol, footprint.Service, footprint.Route)))
	return binary.BigEndian.Uint64(digest[:8])
}

func nonEmpty(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value == "" {
			return false
		}
	}
	return true
}
