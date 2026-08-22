package qualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const QualificationEventSchema = "gomadv3.qualify-event/v1"

type QualificationResultEvent struct {
	Classification string
	ReportPath     string
}

type event struct {
	Schema         string               `json:"schema"`
	Type           string               `json:"type"`
	Classification string               `json:"classification,omitempty"`
	Message        string               `json:"message,omitempty"`
	Iteration      uint64               `json:"iteration,omitempty"`
	Repeat         uint64               `json:"repeat,omitempty"`
	ReportPath     string               `json:"report_path,omitempty"`
	Report         *QualificationReport `json:"report,omitempty"`
}

func WriteProgressEvent(writer io.Writer, iteration, repeat uint64) error {
	return writeEvent(writer, event{Schema: QualificationEventSchema, Type: "progress", Iteration: iteration, Repeat: repeat})
}

func WriteErrorEvent(writer io.Writer, classification string, err error) error {
	return writeEvent(writer, event{Schema: QualificationEventSchema, Type: "error", Classification: classification, Message: err.Error()})
}

func WriteResultEvent(writer io.Writer, report QualificationReport, path string) error {
	return writeEvent(writer, event{Schema: QualificationEventSchema, Type: "result", Classification: ClassifyQualification(report), ReportPath: path, Report: &report})
}

func DecodeResultEvent(contents []byte) (QualificationResultEvent, error) {
	var result QualificationResultEvent
	for _, line := range bytes.Split(bytes.TrimSuffix(contents, []byte{'\n'}), []byte{'\n'}) {
		if len(line) == 0 {
			return QualificationResultEvent{}, errors.New("qualification event stream contains an empty record")
		}
		var decoded event
		if err := json.Unmarshal(line, &decoded); err != nil {
			return QualificationResultEvent{}, fmt.Errorf("decode qualification event: %w", err)
		}
		if decoded.Schema != QualificationEventSchema {
			return QualificationResultEvent{}, fmt.Errorf("unsupported qualification event schema %q", decoded.Schema)
		}
		switch decoded.Type {
		case "progress":
		case "result":
			if result.ReportPath != "" || decoded.ReportPath == "" || decoded.Classification == "" {
				return QualificationResultEvent{}, errors.New("qualification result event is invalid or duplicated")
			}
			result = QualificationResultEvent{Classification: decoded.Classification, ReportPath: decoded.ReportPath}
		case "error":
			return QualificationResultEvent{}, fmt.Errorf("unretained qualification error %s: %s", decoded.Classification, decoded.Message)
		default:
			return QualificationResultEvent{}, fmt.Errorf("unknown qualification event type %q", decoded.Type)
		}
	}
	if result.ReportPath == "" {
		return QualificationResultEvent{}, errors.New("qualification event stream has no retained result")
	}
	return result, nil
}

func ClassifyQualification(report QualificationReport) string {
	if report.Failure != nil {
		return report.Failure.Classification
	}
	if !report.Deterministic {
		return "nondeterministic"
	}
	for _, run := range report.Executions {
		if run.Replay != nil && !run.Replay.Match {
			return "replay_divergence"
		}
	}
	if !report.TargetSuccess {
		return "target_failure"
	}
	return "qualified"
}

func ExitStatus(classification string) int {
	switch classification {
	case "qualified":
		return 0
	case "target_failure", "nondeterministic", "replay_divergence", "semantic_coverage_failure":
		return 1
	case "unsupported_target", "invalid_input":
		return 2
	default:
		return 3
	}
}

func writeEvent(writer io.Writer, value event) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode qualification event: %w", err)
	}
	_, err = fmt.Fprintf(writer, "%s\n", encoded)
	return err
}
