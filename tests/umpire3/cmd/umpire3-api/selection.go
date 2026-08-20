package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const selectionFormatVersion = "umpire3/protobuf-selection/v1"

type fieldDisposition string

const (
	dispositionInterpreted            fieldDisposition = "interpreted"
	dispositionTransportOnly          fieldDisposition = "transport-only"
	dispositionSensitive              fieldDisposition = "sensitive"
	dispositionIntentionallyUnmodeled fieldDisposition = "intentionally-unmodeled"
)

type messageSelection struct {
	FullName                string                      `json:"fullName"`
	Status                  string                      `json:"status,omitempty"`
	Purpose                 string                      `json:"purpose"`
	Owner                   string                      `json:"owner"`
	Reason                  string                      `json:"reason,omitempty"`
	DefaultFieldDisposition fieldDisposition            `json:"defaultFieldDisposition,omitempty"`
	Fields                  map[string]fieldDisposition `json:"fields,omitempty"`
}

type descriptorSelection struct {
	FormatVersion  string             `json:"formatVersion"`
	PresencePolicy string             `json:"presencePolicy"`
	Messages       []messageSelection `json:"messages"`
}

func loadSelection(path string) (descriptorSelection, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return descriptorSelection{}, fmt.Errorf("read protobuf selection: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var selection descriptorSelection
	if err := decoder.Decode(&selection); err != nil {
		return descriptorSelection{}, fmt.Errorf("decode protobuf selection: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return descriptorSelection{}, errors.New("protobuf selection contains trailing JSON")
	}
	if err := selection.validate(); err != nil {
		return descriptorSelection{}, err
	}
	return selection, nil
}

func (s descriptorSelection) validate() error {
	if s.FormatVersion != selectionFormatVersion {
		return fmt.Errorf("unsupported protobuf selection format %q", s.FormatVersion)
	}
	if s.PresencePolicy != "preserve" {
		return fmt.Errorf("unsupported protobuf presence policy %q", s.PresencePolicy)
	}
	if len(s.Messages) == 0 {
		return errors.New("protobuf selection requires messages")
	}
	seen := make(map[string]struct{}, len(s.Messages))
	for _, message := range s.Messages {
		if message.FullName == "" || message.Purpose == "" || message.Owner == "" {
			return errors.New("every protobuf selection requires full name, purpose, and owner")
		}
		if _, duplicate := seen[message.FullName]; duplicate {
			return fmt.Errorf("duplicate protobuf selection %q", message.FullName)
		}
		seen[message.FullName] = struct{}{}
		if message.Status == "deferred" {
			if message.Reason == "" {
				return fmt.Errorf("deferred protobuf selection %q requires a reason", message.FullName)
			}
			continue
		}
		if message.Status != "" && message.Status != "selected" {
			return fmt.Errorf("unknown protobuf selection status %q", message.Status)
		}
		if !message.DefaultFieldDisposition.valid() {
			return fmt.Errorf("message %q requires a valid default field disposition", message.FullName)
		}
		for field, disposition := range message.Fields {
			if field == "" || !disposition.valid() {
				return fmt.Errorf("message %q has invalid field disposition", message.FullName)
			}
		}
	}
	return nil
}

func (d fieldDisposition) valid() bool {
	switch d {
	case dispositionInterpreted, dispositionTransportOnly, dispositionSensitive,
		dispositionIntentionallyUnmodeled:
		return true
	default:
		return false
	}
}
