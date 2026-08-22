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

	"go.temporal.io/server/tools/common/formal/model"
)

type GenerationContract struct {
	Schema         string `json:"schema"`
	ModelIdentity  string `json:"model_identity"`
	BoundsIdentity string `json:"bounds_identity"`
}

type Corpus[S, A, D any, C ~string] struct {
	Schema           string                   `json:"schema"`
	Generation       GenerationContract       `json:"generation"`
	BaselineIdentity string                   `json:"baseline_identity"`
	SemanticDigest   string                   `json:"semantic_digest"`
	CorpusDigest     string                   `json:"corpus_digest"`
	Coverage         []string                 `json:"coverage"`
	Traces           []BehaviorTrace[S, A, D] `json:"traces"`
	Rejections       []RejectionCase[S, A, C] `json:"rejections"`
}

type BehaviorTrace[S, A, D any] struct {
	Name         string             `json:"name"`
	InitialState S                  `json:"initial_state"`
	Steps        []StepRecord[A, D] `json:"steps"`
}

type StepRecord[A, D any] struct {
	Ordinal           int    `json:"ordinal"`
	Action            A      `json:"action"`
	PreStateIdentity  string `json:"pre_state_identity"`
	PostStateIdentity string `json:"post_state_identity"`
	ObservableDelta   D      `json:"observable_delta"`
}

func (record StepRecord[A, D]) Observation() model.Observation[A, D] {
	return model.Observation[A, D]{
		Action: record.Action, PreStateIdentity: record.PreStateIdentity,
		PostStateIdentity: record.PostStateIdentity, ObservableDelta: record.ObservableDelta,
	}
}

type RejectionCase[S, A any, C ~string] struct {
	Name             string `json:"name"`
	InitialState     S      `json:"initial_state"`
	Action           A      `json:"action"`
	PreStateIdentity string `json:"pre_state_identity"`
	Code             C      `json:"code"`
}

type Codec[S, A, D any, C ~string] struct {
	Subject              string
	CorpusSchema         string
	SemanticDigestDomain string
	CorpusDigestDomain   string
}

func (codec Codec[S, A, D, C]) Finalize(corpus *Corpus[S, A, D, C]) error {
	if err := codec.validate(); err != nil {
		return err
	}
	if corpus == nil {
		return fmt.Errorf("%s corpus is nil", codec.Subject)
	}
	if corpus.Schema == "" {
		corpus.Schema = codec.CorpusSchema
	}
	if corpus.Schema != codec.CorpusSchema {
		return fmt.Errorf("%s corpus schema %q is unsupported", codec.Subject, corpus.Schema)
	}
	slices.Sort(corpus.Coverage)
	if err := validateUniqueStrings(codec.Subject, "coverage", corpus.Coverage); err != nil {
		return err
	}
	slices.SortFunc(corpus.Traces, func(left, right BehaviorTrace[S, A, D]) int {
		return strings.Compare(left.Name, right.Name)
	})
	slices.SortFunc(corpus.Rejections, func(left, right RejectionCase[S, A, C]) int {
		return strings.Compare(left.Name, right.Name)
	})
	if err := codec.validateCorpus(*corpus); err != nil {
		return err
	}
	semantic, err := json.Marshal(semanticProjection[S, A, D, C]{
		Coverage: corpus.Coverage, Traces: corpus.Traces, Rejections: corpus.Rejections,
	})
	if err != nil {
		return fmt.Errorf("encode %s semantic projection: %w", codec.Subject, err)
	}
	corpus.SemanticDigest = digest(codec.SemanticDigestDomain, semantic)
	full, err := json.Marshal(corpusProjection[S, A, D, C]{
		Schema: corpus.Schema, Generation: corpus.Generation, BaselineIdentity: corpus.BaselineIdentity,
		SemanticDigest: corpus.SemanticDigest, Coverage: corpus.Coverage, Traces: corpus.Traces, Rejections: corpus.Rejections,
	})
	if err != nil {
		return fmt.Errorf("encode %s corpus projection: %w", codec.Subject, err)
	}
	corpus.CorpusDigest = digest(codec.CorpusDigestDomain, full)
	return nil
}

func (codec Codec[S, A, D, C]) Encode(corpus Corpus[S, A, D, C]) ([]byte, error) {
	canonical := cloneCorpus(corpus)
	if err := codec.Finalize(&canonical); err != nil {
		return nil, err
	}
	if corpus.SemanticDigest != canonical.SemanticDigest || corpus.CorpusDigest != canonical.CorpusDigest {
		return nil, fmt.Errorf("%s corpus digests are stale: semantic %q != %q or corpus %q != %q", codec.Subject, corpus.SemanticDigest, canonical.SemanticDigest, corpus.CorpusDigest, canonical.CorpusDigest)
	}
	if !slices.Equal(corpus.Coverage, canonical.Coverage) || !traceOrderEqual(corpus.Traces, canonical.Traces) || !rejectionOrderEqual(corpus.Rejections, canonical.Rejections) {
		return nil, fmt.Errorf("%s corpus collections are not canonical", codec.Subject)
	}
	encoded, err := json.Marshal(corpus)
	if err != nil {
		return nil, fmt.Errorf("encode %s corpus: %w", codec.Subject, err)
	}
	return append(encoded, '\n'), nil
}

func (codec Codec[S, A, D, C]) Decode(data []byte) (Corpus[S, A, D, C], error) {
	if err := codec.validate(); err != nil {
		return Corpus[S, A, D, C]{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var corpus Corpus[S, A, D, C]
	if err := decoder.Decode(&corpus); err != nil {
		return Corpus[S, A, D, C]{}, fmt.Errorf("decode %s corpus: %w", codec.Subject, err)
	}
	if err := ensureEOF(codec.Subject, decoder); err != nil {
		return Corpus[S, A, D, C]{}, err
	}
	expected := cloneCorpus(corpus)
	if err := codec.Finalize(&expected); err != nil {
		return Corpus[S, A, D, C]{}, err
	}
	if corpus.SemanticDigest != expected.SemanticDigest {
		return Corpus[S, A, D, C]{}, fmt.Errorf("%s semantic digest = %q, want %q", codec.Subject, corpus.SemanticDigest, expected.SemanticDigest)
	}
	if corpus.CorpusDigest != expected.CorpusDigest {
		return Corpus[S, A, D, C]{}, fmt.Errorf("%s corpus digest = %q, want %q", codec.Subject, corpus.CorpusDigest, expected.CorpusDigest)
	}
	canonical, err := codec.Encode(expected)
	if err != nil {
		return Corpus[S, A, D, C]{}, err
	}
	if !bytes.Equal(data, canonical) {
		return Corpus[S, A, D, C]{}, fmt.Errorf("%s corpus is not canonical JSON", codec.Subject)
	}
	return corpus, nil
}

type semanticProjection[S, A, D any, C ~string] struct {
	Coverage   []string                 `json:"coverage"`
	Traces     []BehaviorTrace[S, A, D] `json:"traces"`
	Rejections []RejectionCase[S, A, C] `json:"rejections"`
}

type corpusProjection[S, A, D any, C ~string] struct {
	Schema           string                   `json:"schema"`
	Generation       GenerationContract       `json:"generation"`
	BaselineIdentity string                   `json:"baseline_identity"`
	SemanticDigest   string                   `json:"semantic_digest"`
	Coverage         []string                 `json:"coverage"`
	Traces           []BehaviorTrace[S, A, D] `json:"traces"`
	Rejections       []RejectionCase[S, A, C] `json:"rejections"`
}

func (codec Codec[S, A, D, C]) validate() error {
	if codec.Subject == "" || codec.CorpusSchema == "" || codec.SemanticDigestDomain == "" || codec.CorpusDigestDomain == "" {
		return errors.New("formal trace codec configuration is incomplete")
	}
	return nil
}

func (codec Codec[S, A, D, C]) validateCorpus(corpus Corpus[S, A, D, C]) error {
	if corpus.Generation.Schema == "" || corpus.Generation.ModelIdentity == "" || corpus.Generation.BoundsIdentity == "" {
		return fmt.Errorf("%s generation contract is incomplete", codec.Subject)
	}
	if corpus.BaselineIdentity == "" {
		return fmt.Errorf("%s baseline identity is required", codec.Subject)
	}
	for index, behavior := range corpus.Traces {
		if behavior.Name == "" {
			return fmt.Errorf("%s trace %d has an empty name", codec.Subject, index)
		}
		if index > 0 && corpus.Traces[index-1].Name == behavior.Name {
			return fmt.Errorf("%s trace %q is duplicated", codec.Subject, behavior.Name)
		}
		for ordinal, step := range behavior.Steps {
			if step.Ordinal != ordinal {
				return fmt.Errorf("%s trace %q step ordinal = %d, want %d", codec.Subject, behavior.Name, step.Ordinal, ordinal)
			}
			if step.PreStateIdentity == "" || step.PostStateIdentity == "" {
				return fmt.Errorf("%s trace %q step %d has an empty state identity", codec.Subject, behavior.Name, ordinal)
			}
		}
	}
	for index, rejection := range corpus.Rejections {
		if rejection.Name == "" || rejection.PreStateIdentity == "" || rejection.Code == "" {
			return fmt.Errorf("%s rejection %d is incomplete", codec.Subject, index)
		}
		if index > 0 && corpus.Rejections[index-1].Name == rejection.Name {
			return fmt.Errorf("%s rejection %q is duplicated", codec.Subject, rejection.Name)
		}
	}
	return nil
}

func validateUniqueStrings(subject, kind string, values []string) error {
	for index, value := range values {
		if value == "" {
			return fmt.Errorf("%s %s contains an empty value", subject, kind)
		}
		if index > 0 && values[index-1] == value {
			return fmt.Errorf("%s %s value %q is duplicated", subject, kind, value)
		}
	}
	return nil
}

func traceOrderEqual[S, A, D any](left, right []BehaviorTrace[S, A, D]) bool {
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

func rejectionOrderEqual[S, A any, C ~string](left, right []RejectionCase[S, A, C]) bool {
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

func cloneCorpus[S, A, D any, C ~string](corpus Corpus[S, A, D, C]) Corpus[S, A, D, C] {
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

func ensureEOF(subject string, decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing %s corpus data: %w", subject, err)
	}
	return fmt.Errorf("%s corpus has trailing data", subject)
}

func digest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
