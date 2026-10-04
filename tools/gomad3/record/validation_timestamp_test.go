package record

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTimestampValidationPreservesGrammarAndParseErrors(t *testing.T) {
	for _, text := range []string{"2026-01-02T03:04:05Z", "2026-01-02T03:04:05+00:00", "2026-01-02T03:04:05-00:00", "2026-01-02T03:04:05+01:30", "2026-01-02T03:04:05-08:00", "2026-01-02T03:04:05.123456789Z", "2026-01-02T03:04:05,25Z", "2026-01-02T3:04:05Z", "2026-01-02T03:04:05+24:00", "2026-01-02T03:04:05+00:60", "2026-01-02T25:04:05Z", "2026-02-30T03:04:05Z", "2026-01-02T03:04:05Z trailing", "", "not-a-time"} {
		t.Run(text, func(t *testing.T) {
			_, originalErr := time.Parse(time.RFC3339Nano, text)
			for _, field := range []string{"created", "started", "finished"} {
				input := manifestFixture()
				switch field {
				case "created":
					input.CreatedAt = text
				case "started":
					input.Host.StartedAt = text
				case "finished":
					input.Host.FinishedAt = text
				}
				err := validateManifest(input, false)
				if text == "" && field == "created" {
					if err == nil || err.Error() != "manifest creation time and batch ID are required" {
						t.Fatalf("empty creation precedence: %v", err)
					}
					continue
				}
				if originalErr == nil {
					if err != nil {
						t.Fatalf("%s rejected %q: %v", field, text, err)
					}
					continue
				}
				var original, actual *time.ParseError
				if !errors.As(originalErr, &original) || !errors.As(err, &actual) || !reflect.DeepEqual(actual, original) {
					t.Fatalf("%s parse error = %#v (%v), want %#v", field, actual, err, original)
				}
				prefix := map[string]string{"created": "invalid manifest creation time: ", "started": "invalid host start time: ", "finished": "invalid host finish time: "}[field]
				if err.Error() != prefix+originalErr.Error() {
					t.Fatalf("%s text = %q", field, err.Error())
				}
			}
		})
	}
}

func TestTimestampFinalizeRetainsOriginalStringsAndIdentities(t *testing.T) {
	input := manifestFixture()
	input.CreatedAt = "2026-01-02T03:04:05,25+01:30"
	input.Host.StartedAt = "2026-01-02T03:04:05-08:00"
	input.Host.FinishedAt = "2026-01-02T3:04:06+00:60"
	finalized, encoded := finalizedManifest(t, input)
	decoded, err := DecodeExecutionRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []ExecutionRecord{finalized, decoded} {
		if value.CreatedAt != input.CreatedAt || value.Host.StartedAt != input.Host.StartedAt || value.Host.FinishedAt != input.Host.FinishedAt {
			t.Fatalf("timestamp strings normalized: %+v", value.Host)
		}
	}
	if finalized.RecordHash != "sha256:f5aca3e0f9f1c145a93497aed4bbc25a875bd0823ebb2ab1fe76e091382e6c64" || finalized.Outcome.FailureSignature != "sha256:f43359d4976521234ed6a3127432da4924f9c1ac00e0b26b3c4b06f40118d39a" || HashBytes(encoded) != "sha256:b16b23482df24de9227e1279756eac2027946cdb2e8d05644ad6f189d18a850a" {
		t.Fatalf("timestamp identity changed: record=%s failure=%s bytes=%s", finalized.RecordHash, finalized.Outcome.FailureSignature, HashBytes(encoded))
	}
	input.CreatedAt = "bad creation"
	input.Host.StartedAt = "bad start"
	input.Host.FinishedAt = "bad finish"
	if _, _, err := FinalizeExecutionRecord(input); err == nil || !strings.HasPrefix(err.Error(), "invalid manifest creation time:") {
		t.Fatalf("creation precedence: %v", err)
	}
	input.CreatedAt = "2026-01-02T03:04:05Z"
	if _, _, err := FinalizeExecutionRecord(input); err == nil || !strings.HasPrefix(err.Error(), "invalid host start time:") {
		t.Fatalf("start precedence: %v", err)
	}
}
