package explore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/campaign"
	"go.temporal.io/server/common/testing/testpilot/replay"
	"google.golang.org/protobuf/encoding/protojson"
)

type request struct {
	Frame           string          `json:"frame"`
	Seq             int             `json:"seq"`
	Set             string          `json:"set"`
	Profile         string          `json:"profile"`
	Query           string          `json:"query"`
	Target          string          `json:"target"`
	Identity        string          `json:"identity"`
	Candidate       string          `json:"candidate"`
	Class           replay.Class    `json:"class"`
	Run             json.RawMessage `json:"run"`
	PrepareRejected *string         `json:"prepareRejected"`
	Status          string          `json:"status"`
}

type server struct {
	models               []*modelirspb.Model
	plan                 *Plan
	mode, profile        string
	seq, index, selected int
	outstanding          *Candidate
	retained, subject    *Candidate
	edits                []replay.Edit
	settled              []replay.Settled
	ledger               []campaign.TargetStatus
	capped, undecided    bool
}

// Serve implements the existing campaign and replay protocols over checked IR declarations.
func Serve(input io.Reader, output io.Writer, models []*modelirspb.Model) error {
	s := &server{models: models}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), campaign.DefaultMaxFrameBytes)
	for scanner.Scan() {
		var r request
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&r); err != nil {
			return err
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return errors.New("trailing bridge data")
		}
		reply, err := s.handle(r)
		if err != nil {
			reply = map[string]any{"frame": "rejected", "reason": err.Error()}
		} else {
			s.seq = r.Seq
		}
		reply["seq"], reply["set"], reply["profile"] = r.Seq, r.Set, s.profile
		if s.profile == "" {
			reply["profile"] = r.Profile
		}
		encoded, err := encodeJSON(reply)
		if err != nil {
			return err
		}
		if len(encoded)+1 > campaign.DefaultMaxFrameBytes {
			return campaign.ErrFrameTooLarge
		}
		if _, err = output.Write(append(encoded, '\n')); err != nil {
			return err
		}
		if reply["frame"] == "finished" || reply["frame"] == "crossed" {
			return nil
		}
	}
	return scanner.Err()
}

func (s *server) handle(r request) (map[string]any, error) {
	if r.Seq != s.seq+1 {
		return nil, errors.New("crossed sequence")
	}
	if s.plan == nil {
		return s.start(r)
	}
	if r.Set != s.plan.Name || (r.Profile != "" && r.Profile != s.profile) {
		return nil, errors.New("crossed set or Profile")
	}
	switch r.Frame {
	case "next":
		if s.outstanding != nil {
			return nil, campaign.ErrCandidateOutstanding
		}
		if s.mode == "admit" {
			return s.nextReduction()
		}
		return s.nextCampaign(), nil
	case "observe":
		if s.outstanding == nil {
			return nil, campaign.ErrNoCandidate
		}
		if r.Candidate != s.outstanding.Digest {
			return nil, campaign.ErrCrossedCandidate
		}
		if s.mode == "admit" {
			return s.observeReduction(r)
		}
		return s.observeCampaign(r)
	case "finish":
		if r.Status != "" && r.Status != "stopped" && r.Status != "limit-reached" {
			return nil, errors.New("unknown finish status")
		}
		if s.mode == "admit" {
			return s.finishReduction(r.Status)
		}
		return s.finishCampaign(r.Status), nil
	default:
		return nil, fmt.Errorf("unexpected frame %s", r.Frame)
	}
}

func (s *server) start(r request) (map[string]any, error) {
	if (r.Frame != "initialize" && r.Frame != "admit") || r.Set == "" || r.Profile == "" {
		return nil, errors.New("initialize or admit needs a set and Profile")
	}
	selected, err := s.model(r.Set)
	if err != nil {
		return nil, err
	}
	p, err := New(selected, r.Set)
	if err != nil {
		return nil, err
	}
	if r.Frame == "admit" {
		if r.Target == "" || r.Query != "" || r.Identity == "" {
			return nil, errors.New("IR replay names an exploration target and Case identity")
		}
		for _, c := range p.Candidates {
			if c.Key == r.Target && c.Rejection == "" && c.Identity == r.Identity {
				s.retained, s.subject = c, c
			}
		}
		if s.subject == nil {
			return map[string]any{"frame": "crossed", "reason": "the declared target does not produce this Case"}, nil
		}
	}
	s.plan, s.profile, s.mode = p, r.Profile, r.Frame
	if r.Frame == "initialize" {
		targets := []string{}
		for _, c := range p.Candidates {
			targets = append(targets, c.Key)
			s.ledger = append(s.ledger, campaign.TargetStatus{Target: c.Key, Status: "pending"})
		}
		l := p.query.GetLimits()
		return map[string]any{"frame": "initialized", "machine": p.scenario.GetMachine(), "budget": l.GetName(), "limits": campaign.Limits{Steps: int(l.GetSteps()), Actions: int(l.GetActions()), Search: int(l.GetSearch())}, "targets": targets}, nil
	}
	s.edits = []replay.Edit{}
	if p.query.GetExploration().GetDropPrefix() {
		for i := len(s.subject.Actions) - 2; i >= 0; i-- {
			s.edits = append(s.edits, replay.Edit{Edit: "dropPrefixStep", Index: i, Action: s.subject.Actions[i].GetAction()})
		}
	}
	if len(s.edits) > int(p.Edits) {
		s.edits = s.edits[:p.Edits]
		s.capped = true
	}
	return map[string]any{"frame": "admitted", "subject": s.subject.Digest, "caseId": s.subject.Case.GetCaseId(), "identity": s.subject.Identity, "edits": s.edits, "capped": s.capped}, nil
}

func candidateFrame(c *Candidate) map[string]any {
	return map[string]any{"frame": "candidate", "candidate": c.Digest, "caseId": c.Case.GetCaseId(), "identity": c.Identity, "case": c.Bytes, "target": c.Key, "covers": []string{c.Key}}
}

func (s *server) nextReduction() (map[string]any, error) {
	skipped := []replay.Settled{}
	for s.index < len(s.edits) && !s.undecided {
		edit := s.edits[s.index]
		s.index++
		c, err := s.plan.Reduce(s.retained, edit.Index)
		if err != nil {
			entry := replay.Settled{Edit: edit, Fate: "invalid", Reason: err.Error()}
			s.settled = append(s.settled, entry)
			skipped = append(skipped, entry)
			continue
		}
		s.outstanding = c
		out := candidateFrame(c)
		out["edit"], out["index"], out["action"], out["skipped"] = edit.Edit, edit.Index, edit.Action, skipped
		return out, nil
	}
	return map[string]any{"frame": "exhausted", "skipped": skipped}, nil
}

func (s *server) observeReduction(r request) (map[string]any, error) {
	if (r.Class == "") == (r.PrepareRejected == nil) {
		return nil, errors.New("one class or preparation rejection is required")
	}
	fate, reason := "rejected", "the candidate did not reproduce the failure key"
	switch {
	case r.PrepareRejected != nil:
		reason = *r.PrepareRejected
	case r.Class == replay.ClassReproduced:
		fate, reason = "retained", "two fresh Runs reproduced the same key"
		s.retained = s.outstanding
	case r.Class == replay.ClassIndeterminate:
		fate, reason = "undecided", "the candidate's Runs were indeterminate"
		s.undecided = true
	case r.Class == replay.ClassNotReproduced:
	default:
		return nil, errors.New("unknown reproduction class")
	}
	edit := s.edits[s.index-1]
	digest := s.outstanding.Digest
	s.settled = append(s.settled, replay.Settled{Edit: edit, Fate: fate, Reason: reason, Candidate: &digest})
	s.outstanding = nil
	return map[string]any{"frame": "settled", "candidate": digest, "edit": edit.Edit, "index": edit.Index, "action": edit.Action, "fate": fate, "reason": reason, "retained": s.retained.Digest}, nil
}

func (s *server) finishReduction(status string) (map[string]any, error) {
	reason := "bounded sweep of declared prefix edits"
	var proposal *replay.BridgeProposal
	if status == "" {
		if s.outstanding != nil || s.undecided || s.index < len(s.edits) || s.capped {
			status = "incomplete"
		} else {
			status = "irreducible"
			if s.subject.Digest != s.retained.Digest {
				status = "minimized"
			}
			var err error
			proposal, err = s.plan.Proposal(s.retained)
			if err != nil {
				return nil, err
			}
		}
	}
	return map[string]any{"frame": "finished", "status": status, "reason": reason, "subject": s.subject.Digest, "retained": s.retained.Digest, "edits": s.settled, "proposal": proposal}, nil
}

func (s *server) nextCampaign() map[string]any {
	skipped := []campaign.Skipped{}
	for s.index < len(s.plan.Candidates) && s.selected < int(s.plan.Runs) {
		c := s.plan.Candidates[s.index]
		s.index++
		if c.Rejection != "" {
			s.ledger[s.index-1].Status = "unrealizable"
			skipped = append(skipped, campaign.Skipped{Candidate: c.Digest, Target: c.Key, Reason: c.Rejection})
			continue
		}
		s.selected++
		s.outstanding = c
		out := candidateFrame(c)
		out["skipped"] = skipped
		return out
	}
	return map[string]any{"frame": "exhausted", "skipped": skipped}
}

func (s *server) observeCampaign(r request) (map[string]any, error) {
	if (len(r.Run) == 0) == (r.PrepareRejected == nil) {
		return nil, errors.New("one Run or preparation rejection is required")
	}
	observation, status, detail := "prepare-rejected", "unrealizable", ""
	if r.PrepareRejected != nil {
		detail = *r.PrepareRejected
	} else {
		var run testpilotspb.Run
		if err := protojson.Unmarshal(r.Run, &run); err != nil {
			return nil, err
		}
		if run.GetCaseId() != s.outstanding.Case.GetCaseId() || run.GetProgramId() != s.outstanding.Case.GetProgram().GetProgramId() {
			return nil, errors.New("crossed Run")
		}
		observation, status = "inconclusive", "attempted"
		if run.GetCleanup().GetStatus() == testpilotspb.CLEANUP_STATUS_SUCCEEDED {
			switch run.GetVerdict().GetStatus() {
			case testpilotspb.VERDICT_STATUS_SATISFIED:
				observation, status = "satisfied", "covered"
			case testpilotspb.VERDICT_STATUS_VIOLATED:
				observation, status = "violated", "violated"
			default:
			}
		}
	}
	c := s.outstanding
	s.outstanding = nil
	s.ledger[s.index-1].Status = status
	credited := []string{}
	if status == "covered" {
		credited = append(credited, c.Key)
	}
	return map[string]any{"frame": "credited", "candidate": c.Digest, "observation": observation, "detail": detail, "credited": credited, "statuses": []campaign.TargetStatus{s.ledger[s.index-1]}}, nil
}

func (s *server) finishCampaign(status string) map[string]any {
	sum := campaign.Summary{Targets: len(s.ledger), Selected: s.selected}
	for _, entry := range s.ledger {
		switch entry.Status {
		case "covered":
			sum.Covered++
		case "violated":
			sum.Violated++
		case "attempted":
			sum.Attempted++
		case "unrealizable":
			sum.Unrealizable++
		default:
			sum.Pending++
		}
	}
	sum.Exhausted = sum.Pending == 0
	if status == "" {
		status = "exhausted"
		if sum.Pending > 0 {
			status = "limit-reached"
		}
	}
	return map[string]any{"frame": "finished", "status": status, "summary": sum, "ledger": s.ledger, "counterexamples": []campaign.Counterexample{}}
}

func (s *server) model(name string) (*modelirspb.Model, error) {
	var selected *modelirspb.Model
	for _, m := range s.models {
		for _, q := range m.GetQueries() {
			if q.GetExploration().GetName() == name {
				if selected != nil {
					return nil, errors.New("ambiguous exploration")
				}
				selected = m
			}
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no exploration %s", name)
	}
	return selected, nil
}
