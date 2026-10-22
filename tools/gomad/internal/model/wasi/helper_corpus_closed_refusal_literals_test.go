package wasi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func helperCorpusClosedRefusalLiteral(name string, actual any) (bool, error) {
	termination, message, resultType := "invalid", "", "result"
	started, decoder, cluster := true, false, false
	switch name {
	case "admission_handoff_boundary_documents",
		"admission_scatter_remote_plain_stdout", "admission_scatter_remote_plain_maxfd",
		"admission_scatter_remote_positioned_stdout", "admission_scatter_remote_positioned_maxfd",
		"admission_scatter_modeled_plain_stdout", "admission_scatter_modeled_plain_maxfd",
		"admission_scatter_modeled_positioned_stdout", "admission_scatter_modeled_positioned_maxfd",
		"admission_entropy_two_full_counters", "admission_runtime_forced_rank_full_answer",
		"admission_dormant_predecision_prefix", "admission_remote_chunk_write32_offset_zero",
		"closed_cluster_single_3-exit", "closed_cluster_neighbors_healthy",
		"closed_cluster_network_bound_free", "closed_cluster_manual_inspection_record",
		"closed_cluster_manual_turn_direct",
		"closed_cluster_manual_turn_delayed":
		return true, nil
	case "admission_mode_one_inspection":
		message, started = "invalid local model configuration", false
	case "admission_clock_capacity_callbacks_step_three":
		termination, message = "capacity", "WASI capacity: callbacks"
	case "admission_clock_capacity_transcript_one_step_three":
		termination, message = "capacity", "WASI capacity: transcript-bytes"
	case "admission_clock_capacity_overflow_step_three":
		termination, message = "capacity", "WASI capacity: clock-nanoseconds"
	case "admission_reference_decoder_wrong_id":
		message, decoder = "invalid callback reply type, ID or output", true
	case "admission_reference_decoder_missing_state":
		message, decoder = "missing remote model state", true
	case "admission_reference_decoder_invalid_clock":
		message, decoder = "invalid remote clock state", true
	case "admission_shared_decoder_stop":
		termination, message, decoder = "stop", "modeled", true
	case "admission_shared_decoder_unsupported":
		termination, message, decoder = "unsupported", "fd_pwrite: outside profile", true
	case "admission_shared_decoder_empty_chunk":
		message, decoder = "remote transcript chunk capacity", true
	case "namespace_fd_fdstat_set_flags_unsupported_digest_only_limit1":
		termination, message = "unsupported", "fd_fdstat_set_flags: descriptor flags outside volatile profile"
	case "namespace_path_open_unsupported_digest_only_limit1":
		termination, message = "unsupported", "path_open: open flags outside volatile profile"
	case "namespace_path_readlink_unsupported_digest_only_limit1":
		termination, message = "unsupported", "path_readlink: operation outside stock WASI profile: path_readlink"
	case "namespace_two_max_paths_before_transcript":
		termination, message = "capacity", "WASI capacity: callback-frame"
	case "namespace_output_ab_x_refuse_y_at3":
		termination, message = "capacity", "WASI capacity: offered-output-bytes"
	case "closed_cluster_single_0-trap", "closed_cluster_single_1-memory", "closed_cluster_single_2-fuel":
		cluster = true
	case "closed_cluster_diagnostic_refusal":
		cluster, resultType = true, "cluster_result"
	case "closed_cluster_neighbors_failing", "closed_cluster_network_bound_bounded",
		"closed_cluster_trap_pending_peer", "closed_cluster_tagged_oversized":
		cluster, resultType = true, "cluster_result"
	default:
		return false, nil
	}
	fail := func(detail string) (bool, error) { return true, fmt.Errorf("%s: %s", name, detail) }
	projected, ok := actual.(map[string]any)
	if !ok {
		return fail("missing projected result")
	}
	items, ok := projected["frames"].([]any)
	if !ok || len(items) == 0 {
		return fail("missing result frames")
	}
	if name == "closed_cluster_diagnostic_refusal" {
		started = len(items) != 1
	}
	var frames, callbacks []map[string]any
	starts, terminals := 0, 0
	for _, item := range items {
		frame, ok := item.(map[string]any)
		if !ok {
			return fail("invalid frame")
		}
		frames = append(frames, frame)
		if frame["type"] == "started" {
			starts++
		}
		if frame["type"] == "call" {
			callbacks = append(callbacks, frame)
		}
		if frame["type"] == resultType {
			terminals++
		}
	}
	if (!started && starts != 0) || (started && (starts != 1 || frames[0]["type"] != "started")) {
		return fail("wrong start boundary")
	}
	result := frames[len(frames)-1]
	if result["type"] != resultType || terminals != 1 {
		return fail("wrong terminal frame type")
	}
	if cluster {
		return true, helperCorpusClosedClusterStage(name, frames, callbacks)
	}
	if result["termination"] != termination || result["message"] != message {
		return fail("wrong refusal message or termination")
	}
	if !decoder {
		if len(callbacks) != 0 {
			return fail("unexpected host callback")
		}
		return true, nil
	}
	offset, data := "0", ""
	if strings.HasPrefix(name, "admission_shared_decoder_") {
		offset, data = "18446744073709551615", "AP8="
	}
	want := map[string]any{
		"type": "call", "id": json.Number("1"), "op": "fd_pwrite",
		"input":       map[string]any{"fd": json.Number("13"), "offset": json.Number(offset), "data_base64": data},
		"local_state": map[string]any{"advances": json.Number("0"), "now": json.Number("1"), "transcript_bytes": json.Number("0")},
	}
	if len(callbacks) != 1 || !reflect.DeepEqual(callbacks[0], want) {
		return fail("wrong decoder callback tuple")
	}
	return true, nil
}

func helperCorpusClosedClusterStage(name string, frames, callbacks []map[string]any) error {
	fail := func(detail string) error { return fmt.Errorf("%s: %s", name, detail) }
	last := frames[len(frames)-1]
	result := func(instance string) (map[string]any, error) {
		var found map[string]any
		for _, frame := range frames {
			if frame["type"] == "result" && frame["instance"] == instance {
				if found != nil {
					return nil, fail("duplicate instance result: " + instance)
				}
				found = frame
			}
		}
		if found == nil {
			return nil, fail("missing affected instance result: " + instance)
		}
		return found, nil
	}
	stage := func(frame map[string]any, termination, message string, substring bool) error {
		text, _ := frame["message"].(string)
		if frame["termination"] != termination || (message != "" && ((!substring && text != message) || (substring && !strings.Contains(text, message)))) {
			return fail("wrong affected-instance refusal stage")
		}
		return nil
	}
	call := func(instance any, incarnation any) bool {
		for _, frame := range callbacks {
			input, _ := frame["input"].(map[string]any)
			if frame["instance"] == instance && frame["incarnation"] == incarnation && frame["id"] == json.Number("1") && frame["op"] == "fd_write" && input["fd"] == json.Number("1") && input["data_base64"] == "aGVsbG8K" {
				return true
			}
		}
		return false
	}
	if name == "closed_cluster_diagnostic_refusal" {
		if len(frames) == 1 {
			return stage(last, "unsupported", "diagnostic helper builds do not run clusters", false)
		}
		frame, err := result("a")
		if err != nil {
			return err
		}
		if last["termination"] != "exit" || len(callbacks) != 2 || !call("a", nil) || frame["peak_memory_bytes"] != json.Number("196608") {
			return fail("wrong diagnostic conditional-public-modes boundary")
		}
		clock := callbacks[1]
		if clock["instance"] != "a" || clock["id"] != json.Number("2") || clock["op"] != "clock_time_get" || !reflect.DeepEqual(clock["input"], map[string]any{"clock_id": json.Number("0"), "precision": json.Number("0")}) {
			return fail("wrong diagnostic default clock callback")
		}
		return stage(frame, "exit", "", false)
	}
	if strings.HasPrefix(name, "closed_cluster_single_") {
		if len(callbacks) != 1 || !call(nil, nil) {
			return fail("wrong single-instance callback boundary")
		}
		switch name {
		case "closed_cluster_single_0-trap":
			return stage(last, "trap", "", false)
		case "closed_cluster_single_1-memory":
			return stage(last, "capacity", "guest linear memory capacity", true)
		default:
			return stage(last, "capacity", "fuel", true)
		}
	}
	if last["termination"] != "exit" {
		return fail("wrong cluster completion")
	}
	type expectedStage struct {
		instance, termination, message string
		substring                      bool
	}
	var wants []expectedStage
	switch name {
	case "closed_cluster_neighbors_failing":
		wants = []expectedStage{{"0-trap", "trap", "", false}, {"1-memory", "capacity", "guest linear memory capacity", true}, {"2-fuel", "capacity", "fuel", true}, {"3-exit", "exit", "", false}}
		vector := []any{map[string]any{"id": "0-trap", "termination": "trap"}, map[string]any{"id": "1-memory", "termination": "capacity"}, map[string]any{"id": "2-fuel", "termination": "capacity"}, map[string]any{"id": "3-exit", "termination": "exit"}}
		if !reflect.DeepEqual(last["instances"], vector) || len(callbacks) != 4 {
			return fail("wrong complete neighbor terminal vector or callback count")
		}
		for _, want := range wants {
			if !call(want.instance, nil) {
				return fail("missing neighbor callback boundary")
			}
		}
	case "closed_cluster_network_bound_bounded":
		wants = []expectedStage{{"1-bound", "capacity", "WASI capacity: network-v2-evidence-bytes", false}}
		if len(callbacks) != 0 {
			return fail("owned network refusal called host")
		}
		frame, err := result("1-bound")
		if err != nil {
			return err
		}
		if _, exists := frame["exit_code"]; exists {
			return fail("capacity instance published an exit code")
		}
	case "closed_cluster_trap_pending_peer":
		wants = []expectedStage{{"a", "trap", "", false}, {"b", "exit", "", false}}
		if len(callbacks) != 0 {
			return fail("owned pending-peer execution called host")
		}
	case "closed_cluster_tagged_oversized":
		wants = []expectedStage{{"a", "capacity", "", false}, {"b", "exit", "", false}}
		if !call("a", nil) || !call("b", nil) {
			return fail("missing tagged callback boundary")
		}
	default:
		return fail("unhandled closed negative cluster case")
	}
	for _, want := range wants {
		frame, err := result(want.instance)
		if err != nil {
			return err
		}
		if err := stage(frame, want.termination, want.message, want.substring); err != nil {
			return err
		}
	}
	return nil
}
