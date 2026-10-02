// Package explore expands finite IR declarations and re-lowers legal model edits into whole Cases.
package explore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	producer "go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Candidate struct {
	Drops     []int                    `json:"drops,omitempty"`
	Key       string                   `json:"key"`
	Priority  int64                    `json:"priority"`
	Digest    string                   `json:"digest"`
	Identity  string                   `json:"identity,omitempty"`
	Rejection string                   `json:"rejection,omitempty"`
	Bytes     json.RawMessage          `json:"-"`
	Model     *umpirespb.Model         `json:"-"`
	Case      *testpilotspb.Case       `json:"-"`
	Actions   []*umpirespb.ActionClass `json:"-"`
}

type Plan struct {
	Name       string       `json:"name"`
	Query      string       `json:"query"`
	Runs       int32        `json:"runs"`
	Edits      int32        `json:"edits"`
	Candidates []*Candidate `json:"candidates"`
	base       *umpirespb.Model
	query      *umpirespb.Query
	scenario   *umpirespb.Scenario
}

func New(m *umpirespb.Model, name string) (*Plan, error) {
	m = proto.CloneOf(m)
	if err := umpiremodel.Validate(m); err != nil {
		return nil, err
	}
	p := &Plan{Name: name, base: m}
	for _, q := range m.GetQueries() {
		if q.GetExploration() != nil && q.GetExploration().GetName() == name {
			p.query = q
		}
	}
	if p.query == nil {
		return nil, fmt.Errorf("no declared exploration %q", name)
	}
	for _, s := range m.GetScenarios() {
		if s.GetMachine() == p.query.GetScenario().GetMachine() && s.GetName() == p.query.GetScenario().GetName() {
			p.scenario = s
		}
	}
	p.Query, p.Runs, p.Edits = p.query.GetName(), p.query.GetExploration().GetRuns(), p.query.GetExploration().GetEdits()
	axes := slices.Clone(p.query.GetExploration().GetVariations())
	slices.SortFunc(axes, func(a, b *umpirespb.Variation) int { return int(a.GetIndex() - b.GetIndex()) })
	p.enumerate(axes)
	slices.SortFunc(p.Candidates, func(a, b *Candidate) int {
		if a.Priority > b.Priority {
			return -1
		}
		if a.Priority < b.Priority {
			return 1
		}
		return strings.Compare(a.Key, b.Key)
	})
	return p, nil
}

func (p *Plan) Reduce(c *Candidate, index int) (*Candidate, error) {
	if !p.query.GetExploration().GetDropPrefix() || index < 0 || index >= len(c.Actions)-1 {
		return nil, fmt.Errorf("undeclared or non-prefix reduction %d", index)
	}
	actions := slices.Delete(slices.Clone(c.Actions), index, index+1)
	next := p.lower(c.Key, c.Priority, actions)
	if next.Rejection != "" {
		return nil, fmt.Errorf("reduced Query: %s", next.Rejection)
	}
	next.Drops = append(slices.Clone(c.Drops), index)
	return next, nil
}

func (p *Plan) lower(key string, priority int64, actions []*umpirespb.ActionClass) *Candidate {
	m := proto.CloneOf(p.base)
	for _, q := range m.Queries {
		q.Exploration = nil
	}
	for _, s := range m.Scenarios {
		if s.Name == p.scenario.Name && s.Machine == p.scenario.Machine {
			s.Actions = actions
		}
	}
	c := &Candidate{Key: key, Priority: priority, Model: m, Actions: actions}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		c.Rejection = err.Error()
		return c
	}
	sum := sha256.Sum256(encoded)
	c.Digest = hex.EncodeToString(sum[:])
	lower, err := producer.NewProducer(m)
	if err != nil {
		c.Rejection = err.Error()
		return c
	}
	result, err := lower.Lower(p.Query, producer.IdentityFor("temporal.case", "scala.explore."+p.Name, c.Digest))
	if err != nil {
		c.Rejection = err.Error()
		return c
	}
	if result.Standing != producer.Lowered {
		c.Rejection = fmt.Sprintf("%s: %v", result.Standing, result.Unsupported)
		return c
	}
	c.Case = result.Case
	encoded, err = protojson.Marshal(c.Case)
	if err != nil {
		c.Rejection = err.Error()
		return c
	}
	var compact bytes.Buffer
	if err = json.Compact(&compact, encoded); err != nil {
		c.Rejection = err.Error()
		return c
	}
	c.Bytes = compact.Bytes()
	identity, err := recordedrun.CaseIdentity(c.Bytes)
	if err != nil {
		c.Rejection = err.Error()
		return c
	}
	c.Identity = identity
	return c
}

func (p *Plan) enumerate(axes []*umpirespb.Variation) {
	var expand func(int, []string, int64, map[int][]*umpirespb.ActionClass)
	expand = func(i int, names []string, priority int64, replacements map[int][]*umpirespb.ActionClass) {
		if i < len(axes) {
			axis := axes[i]
			for _, choice := range axis.GetChoices() {
				replacements[int(axis.GetIndex())] = choice.GetActions()
				expand(i+1, append(slices.Clone(names), choice.GetName()), priority+int64(choice.GetPriority()), replacements)
			}
			return
		}
		var actions []*umpirespb.ActionClass
		for index, a := range p.scenario.GetActions() {
			if as, ok := replacements[index]; ok {
				actions = append(actions, as...)
			} else {
				actions = append(actions, a)
			}
		}
		c := p.lower(strings.Join(names, "+"), priority, actions)
		p.Candidates = append(p.Candidates, c)
	}
	expand(0, nil, 0, map[int][]*umpirespb.ActionClass{})
}
