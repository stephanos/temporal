package wasi

import (
	"encoding/json"
	"testing"
)

func TestHelperCorpusClosedRefusalsRejectWrongStage(t *testing.T) {
	for _, name := range []string{
		"admission_mode_one_inspection",
		"admission_clock_capacity_callbacks_step_three",
		"admission_clock_capacity_transcript_one_step_three",
		"admission_clock_capacity_overflow_step_three",
		"admission_reference_decoder_wrong_id",
		"admission_reference_decoder_missing_state",
		"admission_reference_decoder_invalid_clock",
		"admission_shared_decoder_stop",
		"admission_shared_decoder_unsupported",
		"admission_shared_decoder_empty_chunk",
		"namespace_fd_fdstat_set_flags_unsupported_digest_only_limit1",
		"namespace_path_open_unsupported_digest_only_limit1",
		"namespace_path_readlink_unsupported_digest_only_limit1",
		"namespace_two_max_paths_before_transcript",
		"namespace_output_ab_x_refuse_y_at3",
		"closed_cluster_single_0-trap",
		"closed_cluster_single_1-memory",
		"closed_cluster_single_2-fuel",
		"closed_cluster_neighbors_failing",
		"closed_cluster_diagnostic_refusal",
		"closed_cluster_network_bound_bounded",
		"closed_cluster_trap_pending_peer",
		"closed_cluster_tagged_oversized",
	} {
		t.Run(name, func(t *testing.T) {
			actual := map[string]any{"frames": []any{map[string]any{
				"type": "result", "termination": "invalid", "message": "wrong rejection stage",
			}}}
			if err := helperCorpusRefusalLiteral(name, actual); err == nil {
				t.Fatal("accepted a different refusal stage")
			}
		})
	}
}

func TestHelperCorpusClosedNonnegativeCasesDoNotAcquireRefusalGuards(t *testing.T) {
	for _, name := range []string{
		"admission_handoff_boundary_documents",
		"admission_scatter_remote_plain_stdout", "admission_scatter_remote_plain_maxfd",
		"admission_scatter_remote_positioned_stdout", "admission_scatter_remote_positioned_maxfd",
		"admission_scatter_modeled_plain_stdout", "admission_scatter_modeled_plain_maxfd",
		"admission_scatter_modeled_positioned_stdout", "admission_scatter_modeled_positioned_maxfd",
		"admission_entropy_two_full_counters", "admission_runtime_forced_rank_full_answer",
		"admission_dormant_predecision_prefix", "admission_remote_chunk_write32_offset_zero",
		"closed_cluster_single_3-exit", "closed_cluster_neighbors_healthy",
		"closed_cluster_network_bound_free", "closed_cluster_manual_inspection_record",
		"closed_cluster_manual_turn_direct",
		"closed_cluster_manual_turn_delayed",
	} {
		t.Run(name, func(t *testing.T) {
			if err := helperCorpusRefusalLiteral(name, nil); err != nil {
				t.Fatalf("nonnegative case acquired refusal guard: %v", err)
			}
		})
	}
}

func TestHelperCorpusClosedRefusalBoundaries(t *testing.T) {
	const referenceCall = `{"type":"call","id":1,"op":"fd_pwrite","input":{"fd":13,"offset":0,"data_base64":""},"local_state":{"advances":0,"now":1,"transcript_bytes":0}}`
	const sharedCall = `{"type":"call","id":1,"op":"fd_pwrite","input":{"fd":13,"offset":18446744073709551615,"data_base64":"AP8="},"local_state":{"advances":0,"now":1,"transcript_bytes":0}}`
	const singleCall = `{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}}`
	for _, row := range []struct {
		name, termination, message, callback string
		started                              bool
	}{
		{"admission_mode_one_inspection", "invalid", "invalid local model configuration", "", false},
		{"admission_clock_capacity_callbacks_step_three", "capacity", "WASI capacity: callbacks", "", true},
		{"admission_clock_capacity_transcript_one_step_three", "capacity", "WASI capacity: transcript-bytes", "", true},
		{"admission_clock_capacity_overflow_step_three", "capacity", "WASI capacity: clock-nanoseconds", "", true},
		{"admission_reference_decoder_wrong_id", "invalid", "invalid callback reply type, ID or output", referenceCall, true},
		{"admission_reference_decoder_missing_state", "invalid", "missing remote model state", referenceCall, true},
		{"admission_reference_decoder_invalid_clock", "invalid", "invalid remote clock state", referenceCall, true},
		{"admission_shared_decoder_stop", "stop", "modeled", sharedCall, true},
		{"admission_shared_decoder_unsupported", "unsupported", "fd_pwrite: outside profile", sharedCall, true},
		{"admission_shared_decoder_empty_chunk", "invalid", "remote transcript chunk capacity", sharedCall, true},
		{"namespace_fd_fdstat_set_flags_unsupported_digest_only_limit1", "unsupported", "fd_fdstat_set_flags: descriptor flags outside volatile profile", "", true},
		{"namespace_path_open_unsupported_digest_only_limit1", "unsupported", "path_open: open flags outside volatile profile", "", true},
		{"namespace_path_readlink_unsupported_digest_only_limit1", "unsupported", "path_readlink: operation outside stock WASI profile: path_readlink", "", true},
		{"namespace_two_max_paths_before_transcript", "capacity", "WASI capacity: callback-frame", "", true},
		{"namespace_output_ab_x_refuse_y_at3", "capacity", "WASI capacity: offered-output-bytes", "", true},
		{"closed_cluster_single_0-trap", "trap", "", singleCall, true},
		{"closed_cluster_single_1-memory", "capacity", "guest linear memory capacity", singleCall, true},
		{"closed_cluster_single_2-fuel", "capacity", "all fuel consumed by WebAssembly", singleCall, true},
		{"closed_cluster_diagnostic_refusal", "unsupported", "diagnostic helper builds do not run clusters", "", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			build := func() map[string]any {
				var frames []any
				if row.started {
					frames = append(frames, map[string]any{"type": "started"})
				}
				if row.callback != "" {
					value, err := decodeCorpusJSON([]byte(row.callback))
					if err != nil {
						t.Fatal(err)
					}
					frames = append(frames, value)
				}
				kind := "result"
				if row.name == "closed_cluster_diagnostic_refusal" {
					kind = "cluster_result"
				}
				frames = append(frames, map[string]any{"type": kind, "termination": row.termination, "message": row.message})
				return map[string]any{"frames": frames}
			}
			if err := helperCorpusRefusalLiteral(row.name, build()); err != nil {
				t.Fatalf("rejected intended stage: %v", err)
			}
			for _, mutation := range []string{"message", "termination", "started", "callback"} {
				if mutation == "message" && row.termination == "trap" {
					continue
				}
				t.Run(mutation, func(t *testing.T) {
					actual := build()
					frames := actual["frames"].([]any)
					last := frames[len(frames)-1].(map[string]any)
					switch mutation {
					case "message":
						last["message"] = "wrong rejection stage"
					case "termination":
						last["termination"] = "exit"
					case "started":
						if row.started {
							actual["frames"] = frames[1:]
						} else {
							actual["frames"] = append([]any{map[string]any{"type": "started"}}, frames...)
						}
					case "callback":
						if row.callback != "" {
							frames[len(frames)-2].(map[string]any)["id"] = json.Number("2")
						} else {
							actual["frames"] = append(frames[:len(frames)-1], map[string]any{"type": "call"}, last)
						}
					}
					if err := helperCorpusRefusalLiteral(row.name, actual); err == nil {
						t.Fatal("accepted changed intended-stage boundary")
					}
				})
			}
		})
	}
}

func TestHelperCorpusClosedClusterRefusalsSelectAffectedInstance(t *testing.T) {
	for _, row := range []struct{ name, actual, affected string }{
		{"closed_cluster_diagnostic_refusal", `{"frames":[{"type":"started"},{"type":"call","instance":"a","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"call","instance":"a","id":2,"op":"clock_time_get","input":{"clock_id":0,"precision":0}},{"type":"result","instance":"a","termination":"exit","peak_memory_bytes":196608},{"type":"cluster_result","termination":"exit"}]}`, "a"},
		{"closed_cluster_neighbors_failing", `{"frames":[{"type":"started"},{"type":"call","instance":"0-trap","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"call","instance":"1-memory","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"call","instance":"2-fuel","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"call","instance":"3-exit","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"result","instance":"0-trap","termination":"trap"},{"type":"result","instance":"1-memory","termination":"capacity","message":"guest linear memory capacity"},{"type":"result","instance":"2-fuel","termination":"capacity","message":"all fuel consumed by WebAssembly"},{"type":"result","instance":"3-exit","termination":"exit"},{"type":"cluster_result","termination":"exit","instances":[{"id":"0-trap","termination":"trap"},{"id":"1-memory","termination":"capacity"},{"id":"2-fuel","termination":"capacity"},{"id":"3-exit","termination":"exit"}]}]}`, "1-memory"},
		{"closed_cluster_network_bound_bounded", `{"frames":[{"type":"started"},{"type":"result","instance":"1-bound","termination":"capacity","message":"WASI capacity: network-v2-evidence-bytes"},{"type":"cluster_result","termination":"exit"}]}`, "1-bound"},
		{"closed_cluster_trap_pending_peer", `{"frames":[{"type":"started"},{"type":"result","instance":"a","termination":"trap"},{"type":"result","instance":"b","termination":"exit"},{"type":"cluster_result","termination":"exit"}]}`, "a"},
		{"closed_cluster_tagged_oversized", `{"frames":[{"type":"started"},{"type":"call","instance":"a","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"call","instance":"b","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"aGVsbG8K"}},{"type":"result","instance":"a","termination":"capacity","message":"protocol frame exceeds 8 MiB"},{"type":"result","instance":"b","termination":"exit"},{"type":"cluster_result","termination":"exit"}]}`, "a"},
	} {
		t.Run(row.name, func(t *testing.T) {
			for _, mutate := range []bool{false, true} {
				actual, err := decodeCorpusJSON([]byte(row.actual))
				if err != nil {
					t.Fatal(err)
				}
				if mutate {
					for _, item := range actual.(map[string]any)["frames"].([]any) {
						frame := item.(map[string]any)
						if row.affected == "" && frame["type"] == "call" {
							frame["incarnation"] = json.Number("1")
						} else if frame["type"] == "result" && frame["instance"] == row.affected {
							if frame["termination"] == "exit" {
								frame["termination"] = "capacity"
							} else {
								frame["termination"] = "exit"
							}
						}
					}
				}
				err = helperCorpusRefusalLiteral(row.name, actual)
				if (!mutate && err != nil) || (mutate && err == nil) {
					t.Fatalf("affected-instance mutation=%v: %v", mutate, err)
				}
			}
			if row.name == "closed_cluster_diagnostic_refusal" {
				for _, field := range []string{"clock_id", "callback_id", "peak", "missing_clock"} {
					t.Run(field, func(t *testing.T) {
						actual, err := decodeCorpusJSON([]byte(row.actual))
						if err != nil {
							t.Fatal(err)
						}
						frames := actual.(map[string]any)["frames"].([]any)
						switch field {
						case "clock_id":
							frames[2].(map[string]any)["input"].(map[string]any)["clock_id"] = json.Number("1")
						case "callback_id":
							frames[2].(map[string]any)["id"] = json.Number("3")
						case "peak":
							frames[3].(map[string]any)["peak_memory_bytes"] = json.Number("65536")
						case "missing_clock":
							actual.(map[string]any)["frames"] = append(frames[:2], frames[3:]...)
						}
						if err := helperCorpusRefusalLiteral(row.name, actual); err == nil {
							t.Fatal("accepted changed benign diagnostic boundary")
						}
					})
				}
			}
		})
	}
}
