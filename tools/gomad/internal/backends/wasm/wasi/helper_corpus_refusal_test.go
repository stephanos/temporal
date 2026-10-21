package wasi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func helperCorpusRefusalLiteral(name string, actual any) error {
	if handled, err := helperCorpusClosedRefusalLiteral(name, actual); handled {
		return err
	}
	var want string
	prefix := false
	switch name {
	case "model_inputs_refused_environment_count":
		want = "invalid local model string count"
	case "model_inputs_refused_invalid_utf8":
		want = "invalid utf-8 sequence of 1 bytes from index 0"
	case "model_inputs_refused_end_wrong_type":
		want = "invalid local model input end"
	case "model_inputs_refused_end_unknown_field":
		want, prefix = "unknown field `extra`, expected `type`", true
	case "model_inputs_refused_pwd_digest_mismatch", "model_inputs_refused_environment_shape_3":
		want = "invalid local model working directory identity"
	case "model_inputs_refused_arguments_over_bound", "model_inputs_refused_environment_wrong_chunks":
		want = "invalid local model configuration"
	case "model_inputs_refused_retired_network_field":
		want, prefix = "unknown field `network`,", true
	case "model_inputs_refused_environment_shape_0", "model_inputs_refused_environment_shape_2":
		want = "invalid local model environment names"
	case "model_inputs_refused_environment_shape_1":
		want = "invalid local model environment"
	case "model_inputs_refused_tape_capacity":
		want = "replay preflight framing: invalid replay trace byte/frame declaration"
	case "model_clock_first_run_51", "model_entropy_reservation_35", "model_unsupported_flags_74", "model_unsupported_open_95", "model_unsupported_readlink_91", "model_namespace_completed_before_capacity", "model_completed_clock_before_capacity", "model_remote_transcript_19":
		want = "WASI capacity: transcript-bytes"
	case "model_clock_callback_limit":
		want = "WASI capacity: callbacks"
	case "model_clock_overflow_atomic":
		want = "WASI capacity: clock-nanoseconds"
	case "model_runtime_choice_capacity":
		want = "WASI capacity: runtime-choice-bytes"
	default:
		switch {
		case name == "admission_storage_model_identity":
			want = "storage model identity mismatch"
		case name == "admission_storage_image_descriptor":
			want = "invalid storage image descriptor"
		case name == "admission_storage_unknown_field":
			want, prefix = "unknown field `extra`,", true
		case name == "admission_storage_image_capacity":
			want = "WASI capacity: storage-image-bytes"
		case name == "admission_storage_context_400":
			want = "storage crash context diverged from prior observation"
		case name == "admission_storage_context_399":
			want = "WASI invalid: storage crash context inventory"
		case name == "admission_storage_context_missing_end":
			want = "unexpected protocol EOF"
		case name == "admission_model_input_missing_nul":
			want = "invalid local model input chunk length"
		case name == "admission_model_input_sequence_gap" || name == "admission_model_input_undeclared_arguments":
			want = "invalid local model input chunk"
		case name == "admission_model_input_truncated_missing_end" || name == "admission_model_input_truncated_no_inputs":
			want = "unexpected protocol EOF"
		case name == "admission_model_input_unknown_field":
			want, prefix = "unknown field `unknown`, expected one of `type`, `stream`, `sequence`, `data_base64`", true
		case name == "admission_exact_cpu_max_id" || name == "admission_exact_cpu_two":
			want = "clock_time_get: CPU clocks outside stock WASI profile"
		case name == "admission_exact_clock_remote_tuple":
			want = "clock_time_get: CPU clocks outside stock WASI profile"
		case name == "admission_exact_remote_json_short":
			want = "WASI capacity: transcript-bytes"
		case name == "admission_protocol_namespace_without_owner":
			want = "namespace requires local model ownership"
		case name == "admission_debug_invalid_frame":
			want = "invalid debug exchange frame"
		case strings.HasPrefix(name, "admission_debug_"):
			want = "inspection unsupported without helper-owned runtime"
		case strings.HasPrefix(name, "admission_namespace_descriptor_"):
			want = "invalid namespace descriptor"
		case name == "admission_namespace_cwd_identity":
			want = "invalid namespace image digest or length"
		case strings.HasPrefix(name, "admission_namespace_") && name != "admission_namespace_shared_literal_hi":
			want = "invalid namespace image"
		case name == "admission_remote_invalid_clock":
			want = "invalid remote clock state"
		default:
			return nil
		}
	}
	projected, ok := actual.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: missing projected result", name)
	}
	frames, ok := projected["frames"].([]any)
	if !ok || len(frames) == 0 {
		return fmt.Errorf("%s: missing result frames", name)
	}
	result, ok := frames[len(frames)-1].(map[string]any)
	if !ok {
		return fmt.Errorf("%s: missing terminal result", name)
	}
	message, ok := result["message"].(string)
	if !ok || result["type"] != "result" || (!prefix && message != want) || (prefix && !strings.HasPrefix(message, want)) {
		return fmt.Errorf("%s: wrong refusal stage: want %q, got %q", name, want, message)
	}
	if name == "model_inputs_refused_tape_capacity" && (result["error"] != "replay_preflight" || result["check"] != "framing") {
		return fmt.Errorf("%s: missing typed replay framing refusal", name)
	}
	if strings.HasPrefix(name, "admission_") {
		termination := "invalid"
		if strings.HasPrefix(name, "admission_debug_") {
			termination = "debug_error"
		} else if name == "admission_storage_image_capacity" || name == "admission_exact_remote_json_short" {
			termination = "capacity"
		} else if name == "admission_storage_unknown_field" || name == "admission_exact_cpu_max_id" || name == "admission_exact_cpu_two" || name == "admission_exact_clock_remote_tuple" {
			termination = "unsupported"
		}
		if result["termination"] != termination {
			return fmt.Errorf("%s: wrong admission termination: %v", name, result["termination"])
		}
		started := false
		for _, item := range frames {
			frame, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("%s: invalid frame", name)
			}
			started = started || frame["type"] == "started"
			if frame["type"] == "call" && name != "admission_remote_invalid_clock" && name != "admission_exact_clock_remote_tuple" && name != "admission_exact_remote_json_short" {
				return fmt.Errorf("%s: admission refusal called host", name)
			}
		}
		if strings.HasPrefix(name, "admission_namespace_") && started == strings.HasPrefix(name, "admission_namespace_descriptor_") {
			return fmt.Errorf("%s: wrong namespace admission boundary", name)
		}
	}
	return nil
}

func TestHelperCorpusRefusalLiteralsRejectWrongStageMessages(t *testing.T) {
	for _, name := range []string{
		"model_inputs_refused_environment_count", "model_inputs_refused_invalid_utf8",
		"model_inputs_refused_end_wrong_type", "model_inputs_refused_end_unknown_field",
		"model_inputs_refused_pwd_digest_mismatch", "model_inputs_refused_arguments_over_bound",
		"model_inputs_refused_environment_wrong_chunks", "model_inputs_refused_retired_network_field",
		"model_inputs_refused_environment_shape_0", "model_inputs_refused_environment_shape_1",
		"model_inputs_refused_environment_shape_2", "model_inputs_refused_environment_shape_3",
		"model_inputs_refused_tape_capacity", "model_clock_first_run_51",
		"model_entropy_reservation_35", "model_unsupported_flags_74",
		"model_unsupported_open_95", "model_unsupported_readlink_91",
		"model_namespace_completed_before_capacity", "model_completed_clock_before_capacity",
		"model_clock_callback_limit", "model_clock_overflow_atomic",
		"model_remote_transcript_19", "model_runtime_choice_capacity",
	} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(helperCorpusDirectory, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			value, err := decodeCorpusJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			actual := value.(map[string]any)["expected"].(map[string]any)
			if err := helperCorpusRefusalLiteral(name, actual); err != nil {
				t.Fatal(err)
			}
			frames := actual["frames"].([]any)
			frames[len(frames)-1].(map[string]any)["message"] = "wrong admission stage"
			if err := helperCorpusRefusalLiteral(name, actual); err == nil {
				t.Fatal("accepted a different refusal stage")
			}
		})
	}
}

func TestHelperCorpusAdmissionRefusalsRejectWrongStageMessages(t *testing.T) {
	for _, name := range []string{
		"admission_debug_invalid_frame", "admission_debug_without_runtime_4",
		"admission_namespace_descriptor_chunks_zero", "admission_namespace_chunk_sequence",
		"admission_namespace_cwd_identity", "admission_remote_invalid_clock",
		"admission_storage_model_identity", "admission_storage_image_descriptor",
		"admission_storage_unknown_field", "admission_storage_image_capacity",
		"admission_storage_context_400", "admission_storage_context_399", "admission_storage_context_missing_end",
		"admission_model_input_missing_nul", "admission_model_input_sequence_gap",
		"admission_model_input_truncated_missing_end", "admission_model_input_truncated_no_inputs",
		"admission_model_input_undeclared_arguments", "admission_model_input_unknown_field",
		"admission_exact_cpu_max_id", "admission_exact_cpu_two", "admission_exact_clock_remote_tuple",
		"admission_exact_remote_json_short", "admission_protocol_namespace_without_owner",
	} {
		t.Run(name, func(t *testing.T) {
			actual := map[string]any{"frames": []any{map[string]any{
				"type": "result", "termination": "invalid", "message": "wrong rejection stage",
			}}}
			if err := helperCorpusRefusalLiteral(name, actual); err == nil {
				t.Fatal("wrong-stage admission refusal accepted")
			}
		})
	}
}

func TestHelperCorpusStorageUnknownFieldTermination(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		termination string
		wantError   bool
	}{
		{termination: "unsupported"},
		{termination: "invalid", wantError: true},
		{termination: "capacity", wantError: true},
	} {
		t.Run(test.termination, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(helperCorpusDirectory, "admission_storage_unknown_field.json"))
			if err != nil {
				t.Fatal(err)
			}
			value, err := decodeCorpusJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			actual := value.(map[string]any)["expected"].(map[string]any)
			frames := actual["frames"].([]any)
			frames[len(frames)-1].(map[string]any)["termination"] = test.termination
			err = helperCorpusRefusalLiteral("admission_storage_unknown_field", actual)
			if (err != nil) != test.wantError {
				t.Fatalf("termination %s: error = %v, want error = %v", test.termination, err, test.wantError)
			}
		})
	}
}
