package trace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv4/virtualtime"
)

const CorpusSchema = "gomadv4.virtual-time-corpus/v1"

type GenerationContract struct {
	Schema         string `json:"schema"`
	ModelIdentity  string `json:"model_identity"`
	BoundsIdentity string `json:"bounds_identity"`
}

type Corpus struct {
	Schema           string             `json:"schema"`
	Generation       GenerationContract `json:"generation"`
	BaselineIdentity string             `json:"baseline_identity"`
	SemanticDigest   string             `json:"semantic_digest"`
	CorpusDigest     string             `json:"corpus_digest"`
	Coverage         []string           `json:"coverage"`
	Traces           []BehaviorTrace    `json:"traces"`
	Rejections       []RejectionCase    `json:"rejections"`
}

type BehaviorTrace struct {
	Name         string                    `json:"name"`
	InitialState virtualtime.StateSnapshot `json:"initial_state"`
	Steps        []StepRecord              `json:"steps"`
}

type StepRecord struct {
	Ordinal           int                         `json:"ordinal"`
	Action            virtualtime.Action          `json:"action"`
	PreStateIdentity  string                      `json:"pre_state_identity"`
	PostStateIdentity string                      `json:"post_state_identity"`
	ObservableDelta   virtualtime.ObservableDelta `json:"observable_delta"`
}

type RejectionCase struct {
	Name             string                    `json:"name"`
	InitialState     virtualtime.StateSnapshot `json:"initial_state"`
	Action           virtualtime.Action        `json:"action"`
	PreStateIdentity string                    `json:"pre_state_identity"`
	Code             virtualtime.RejectionCode `json:"code"`
}

func Finalize(corpus *Corpus) error {
	if corpus == nil {
		return errors.New("virtual time corpus is nil")
	}
	if corpus.Schema == "" {
		corpus.Schema = CorpusSchema
	}
	if corpus.Schema != CorpusSchema {
		return fmt.Errorf("virtual time corpus schema %q is unsupported", corpus.Schema)
	}
	slices.Sort(corpus.Coverage)
	if err := validateUniqueStrings("coverage", corpus.Coverage); err != nil {
		return err
	}
	slices.SortFunc(corpus.Traces, func(left, right BehaviorTrace) int {
		return strings.Compare(left.Name, right.Name)
	})
	slices.SortFunc(corpus.Rejections, func(left, right RejectionCase) int {
		return strings.Compare(left.Name, right.Name)
	})
	if err := validateCorpus(*corpus); err != nil {
		return err
	}
	semantic, err := json.Marshal(semanticProjection{
		Coverage: corpus.Coverage, Traces: corpus.Traces, Rejections: corpus.Rejections,
	})
	if err != nil {
		return fmt.Errorf("encode virtual time semantic projection: %w", err)
	}
	corpus.SemanticDigest = digest("gomadv4.virtual-time-corpus-semantic/v1", semantic)
	full, err := json.Marshal(corpusProjection{
		Schema: corpus.Schema, Generation: corpus.Generation, BaselineIdentity: corpus.BaselineIdentity,
		SemanticDigest: corpus.SemanticDigest, Coverage: corpus.Coverage, Traces: corpus.Traces, Rejections: corpus.Rejections,
	})
	if err != nil {
		return fmt.Errorf("encode virtual time corpus projection: %w", err)
	}
	corpus.CorpusDigest = digest("gomadv4.virtual-time-corpus/v1", full)
	return nil
}

func Encode(corpus Corpus) ([]byte, error) {
	canonical := cloneCorpus(corpus)
	if err := Finalize(&canonical); err != nil {
		return nil, err
	}
	if corpus.SemanticDigest != canonical.SemanticDigest || corpus.CorpusDigest != canonical.CorpusDigest {
		return nil, fmt.Errorf("virtual time corpus digests are stale: semantic %q != %q or corpus %q != %q", corpus.SemanticDigest, canonical.SemanticDigest, corpus.CorpusDigest, canonical.CorpusDigest)
	}
	if !slices.Equal(corpus.Coverage, canonical.Coverage) || !traceOrderEqual(corpus.Traces, canonical.Traces) || !rejectionOrderEqual(corpus.Rejections, canonical.Rejections) {
		return nil, errors.New("virtual time corpus collections are not canonical")
	}
	encoded, err := json.Marshal(corpus)
	if err != nil {
		return nil, fmt.Errorf("encode virtual time corpus: %w", err)
	}
	return append(encoded, '\n'), nil
}

func Decode(data []byte) (Corpus, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var corpus Corpus
	if err := decoder.Decode(&corpus); err != nil {
		return Corpus{}, fmt.Errorf("decode virtual time corpus: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Corpus{}, err
	}
	expected := cloneCorpus(corpus)
	if err := Finalize(&expected); err != nil {
		return Corpus{}, err
	}
	if corpus.SemanticDigest != expected.SemanticDigest {
		return Corpus{}, fmt.Errorf("virtual time semantic digest = %q, want %q", corpus.SemanticDigest, expected.SemanticDigest)
	}
	if corpus.CorpusDigest != expected.CorpusDigest {
		return Corpus{}, fmt.Errorf("virtual time corpus digest = %q, want %q", corpus.CorpusDigest, expected.CorpusDigest)
	}
	canonical, err := Encode(expected)
	if err != nil {
		return Corpus{}, err
	}
	if !bytes.Equal(data, canonical) {
		return Corpus{}, errors.New("virtual time corpus is not canonical JSON")
	}
	return corpus, nil
}

type semanticProjection struct {
	Coverage   []string        `json:"coverage"`
	Traces     []BehaviorTrace `json:"traces"`
	Rejections []RejectionCase `json:"rejections"`
}

type corpusProjection struct {
	Schema           string             `json:"schema"`
	Generation       GenerationContract `json:"generation"`
	BaselineIdentity string             `json:"baseline_identity"`
	SemanticDigest   string             `json:"semantic_digest"`
	Coverage         []string           `json:"coverage"`
	Traces           []BehaviorTrace    `json:"traces"`
	Rejections       []RejectionCase    `json:"rejections"`
}

func validateCorpus(corpus Corpus) error {
	if corpus.Generation.Schema == "" || corpus.Generation.ModelIdentity == "" || corpus.Generation.BoundsIdentity == "" {
		return errors.New("virtual time generation contract is incomplete")
	}
	if corpus.BaselineIdentity == "" {
		return errors.New("virtual time baseline identity is required")
	}
	for index, behavior := range corpus.Traces {
		if behavior.Name == "" {
			return fmt.Errorf("virtual time trace %d has an empty name", index)
		}
		if index > 0 && corpus.Traces[index-1].Name == behavior.Name {
			return fmt.Errorf("virtual time trace %q is duplicated", behavior.Name)
		}
		for ordinal, step := range behavior.Steps {
			if step.Ordinal != ordinal {
				return fmt.Errorf("virtual time trace %q step ordinal = %d, want %d", behavior.Name, step.Ordinal, ordinal)
			}
			if step.PreStateIdentity == "" || step.PostStateIdentity == "" {
				return fmt.Errorf("virtual time trace %q step %d has an empty state identity", behavior.Name, ordinal)
			}
		}
	}
	for index, rejection := range corpus.Rejections {
		if rejection.Name == "" || rejection.PreStateIdentity == "" || rejection.Code == "" {
			return fmt.Errorf("virtual time rejection %d is incomplete", index)
		}
		if index > 0 && corpus.Rejections[index-1].Name == rejection.Name {
			return fmt.Errorf("virtual time rejection %q is duplicated", rejection.Name)
		}
	}
	return nil
}

func validateUniqueStrings(kind string, values []string) error {
	for index, value := range values {
		if value == "" {
			return fmt.Errorf("virtual time %s contains an empty value", kind)
		}
		if index > 0 && values[index-1] == value {
			return fmt.Errorf("virtual time %s value %q is duplicated", kind, value)
		}
	}
	return nil
}

func traceOrderEqual(left, right []BehaviorTrace) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Name != right[index].Name {
			return false
		}
	}
	return true
}

func rejectionOrderEqual(left, right []RejectionCase) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Name != right[index].Name {
			return false
		}
	}
	return true
}

func cloneCorpus(corpus Corpus) Corpus {
	cloned := corpus
	cloned.Coverage = cloneSlice(corpus.Coverage)
	cloned.Traces = cloneSlice(corpus.Traces)
	for index := range cloned.Traces {
		cloned.Traces[index].Steps = cloneSlice(corpus.Traces[index].Steps)
	}
	cloned.Rejections = cloneSlice(corpus.Rejections)
	return cloned
}

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	cloned := make([]T, len(values))
	copy(cloned, values)
	return cloned
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing virtual time corpus data: %w", err)
	}
	return errors.New("virtual time corpus has trailing data")
}

func digest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
