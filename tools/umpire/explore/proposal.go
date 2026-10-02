package explore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/replay"
	"google.golang.org/protobuf/encoding/protojson"
)

type proposal struct {
	Version     int             `json:"version"`
	Exploration string          `json:"exploration"`
	Query       string          `json:"query"`
	Target      string          `json:"target"`
	Drops       []int           `json:"drops"`
	Digest      string          `json:"digest"`
	Identity    string          `json:"caseIdentity"`
	Model       json.RawMessage `json:"model"`
	Case        json.RawMessage `json:"case"`
}

func (p *Plan) Proposal(c *Candidate) (*replay.BridgeProposal, error) {
	model, err := protojson.Marshal(p.base)
	if err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err = json.Compact(&compact, model); err != nil {
		return nil, err
	}
	source, err := encodeJSON(proposal{Version: 1, Exploration: p.Name, Query: p.Query, Target: c.Key, Drops: c.Drops, Digest: c.Digest, Identity: c.Identity, Model: compact.Bytes(), Case: c.Bytes})
	if err != nil {
		return nil, err
	}
	if _, err = ReadProposal(source); err != nil {
		return nil, fmt.Errorf("re-answer proposal: %w", err)
	}
	digest := recordedrun.Digest(source)
	return &replay.BridgeProposal{Digest: c.Digest, SHA256: &digest, Path: c.Digest + "-regression.json", Source: string(source)}, nil
}

// ReadProposal re-answers a saved model recipe and verifies the exact Case bytes before replay.
func ReadProposal(source []byte) (*Candidate, error) {
	var p proposal
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("trailing proposal data")
	}
	if p.Version != 1 {
		return nil, errors.New("unsupported proposal version")
	}
	var model modelirspb.Model
	if err := protojson.Unmarshal(p.Model, &model); err != nil {
		return nil, err
	}
	plan, err := New(&model, p.Exploration)
	if err != nil {
		return nil, err
	}
	if p.Query != plan.Query {
		return nil, errors.New("crossed proposal Query")
	}
	var retained *Candidate
	for _, c := range plan.Candidates {
		if c.Key == p.Target {
			retained = c
		}
	}
	if retained == nil || retained.Rejection != "" {
		return nil, errors.New("proposal target has no Case")
	}
	if len(p.Drops) > int(plan.Edits) {
		return nil, errors.New("proposal exceeds declared edit bound")
	}
	previous := len(retained.Actions)
	for _, index := range p.Drops {
		if index >= previous {
			return nil, errors.New("proposal edits are not a bounded prefix sweep")
		}
		previous = index
		retained, err = plan.Reduce(retained, index)
		if err != nil {
			return nil, err
		}
	}
	if retained.Digest != p.Digest || retained.Identity != p.Identity || !bytes.Equal(retained.Bytes, p.Case) {
		return nil, errors.New("proposal does not regenerate its exact identities and Case")
	}
	return retained, nil
}

func encodeJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}
