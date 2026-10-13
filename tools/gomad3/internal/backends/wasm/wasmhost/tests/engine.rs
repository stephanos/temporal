use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::io::{BufRead, BufReader, Write};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

fn run(wat: &str, change: Value, reply: impl FnMut(&Value) -> Value) -> Vec<Value> {
    run_authorized(wat, change, reply, AuthorizationMismatch::None)
}
#[derive(Clone, Copy)]
enum AuthorizationMismatch {
    None,
    Module,
    Configuration,
}
fn run_authorized(
    wat: &str,
    change: Value,
    reply: impl FnMut(&Value) -> Value,
    mismatch: AuthorizationMismatch,
) -> Vec<Value> {
    run_captured(wat, change, reply, mismatch).0
}
fn run_captured(
    wat: &str,
    change: Value,
    mut reply: impl FnMut(&Value) -> Value,
    mismatch: AuthorizationMismatch,
) -> (Vec<Value>, String) {
    use std::io::Read;
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, wat).unwrap();
    let mut request = json!({"schema":"gomad3.wasm-host/v1", "type":"execute", "module_path":path, "module_sha256":format!("{:x}",Sha256::digest(wat.as_bytes())), "memory_bytes":131072, "fuel":1000000, "output_bytes":1024});
    for (key, value) in change.as_object().unwrap() {
        request[key] = value.clone();
    }
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            let configuration = if matches!(mismatch, AuthorizationMismatch::Configuration) {
                // Build the previous revision without preserving a live identity literal.
                Value::String(["cranelift-fuel-nan-canonical", "v1"].join("-"))
            } else {
                frame["engine"]["configuration"].clone()
            };
            let module = if matches!(mismatch, AuthorizationMismatch::Module) {
                json!("bad")
            } else {
                frame["module_sha256"].clone()
            };
            writeln!(
                input,
                "{}",
                json!({"type":"authorize","module_sha256":module,"configuration":configuration})
            )
            .unwrap();
        }
        if frame["type"] == "call" {
            writeln!(input, "{}", reply(&frame)).unwrap();
        }
        frames.push(frame);
    }
    let mut stderr = String::new();
    child
        .stderr
        .take()
        .unwrap()
        .read_to_string(&mut stderr)
        .unwrap();
    assert!(child.wait().unwrap().success(), "{stderr}");
    std::fs::remove_file(path).unwrap();
    (frames, stderr)
}
fn deny(call: &Value) -> Value {
    json!({"type":"reply", "id":call["id"], "errno":0, "output":{}, "unsupported":true, "message":"not modeled"})
}
fn termination(frames: &[Value]) -> &str {
    frames.last().unwrap()["termination"].as_str().unwrap()
}

#[cfg(not(feature = "guest-profile"))]
#[test]
fn release_refuses_guest_profile_before_loading_or_execution() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))",
        json!({"guest_profile":{"path":"/tmp/unused-gomad-profile.json","interval_micros":1000,"max_samples":1}}),
        deny,
    );
    assert_eq!(termination(&frames), "invalid");
    assert_eq!(frames.len(), 1);
    assert!(
        frames[0]["message"]
            .as_str()
            .unwrap()
            .contains("guest_profile")
    );
}

#[cfg(feature = "diagnostic-fuel-off")]
#[test]
fn diagnostic_fuel_off_reports_distinct_revision_and_preserves_offered_fuel() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({"fuel":1}),
        deny,
    );
    assert_eq!(termination(&frames), "exit");
    assert_eq!(frames.last().unwrap()["fuel_remaining"], 1);
    assert!(
        frames[0]["engine"]["configuration"]
            .as_str()
            .unwrap()
            .ends_with("-fuel-off")
    );
}

#[cfg(feature = "guest-profile")]
#[test]
fn guest_profile_is_symbolized_bounded_and_preserves_evidence_on_write_failure() {
    let directory =
        std::env::temp_dir().join(format!("gomad-guest-profile-{}", std::process::id()));
    std::fs::create_dir(&directory).unwrap();
    let path = directory.join("profile.json");
    let mut decision = [0u8; 192];
    decision[..4].copy_from_slice(&1u32.to_be_bytes());
    decision[4..8].copy_from_slice(&1u32.to_be_bytes());
    decision[20..24].copy_from_slice(&2u32.to_be_bytes());
    decision[32..64].fill(1);
    decision[64..96].fill(2);
    let literal: String = decision
        .iter()
        .map(|byte| format!("\\{byte:02x}"))
        .collect();
    let guest = format!(
        "(module (import \"gomad_wasm_v1\" \"config\" (func $config (param i32 i32) (result i32))) (import \"gomad_wasm_v1\" \"decision\" (func $decision (param i32 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"{literal}\") (func $hot (local $n i32) i32.const 10000000 local.set $n (loop $work local.get $n i32.const 1 i32.sub local.tee $n br_if $work)) (func (export \"_start\") i32.const 0 i32.const 16 call $config drop i32.const 32 i32.const 192 call $decision drop i32.const 1 i64.const 0 i32.const 256 call $clock drop call $hot))"
    );
    let model = json!({"seed":7,"mode":1,"limit":4096,"diagnostics":true,"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}});
    let baseline = run(&guest, json!({"fuel":1000000000,"local_model":model}), deny);
    assert!(
        baseline
            .iter()
            .any(|frame| frame["type"] == "evidence" && frame["stream"] == "choice")
    );
    assert!(
        baseline
            .iter()
            .any(|frame| frame["type"] == "evidence" && frame["stream"] == "transcript")
    );
    let config = json!({"path":path,"interval_micros":1,"max_samples":1});
    let sampled = run(
        &guest,
        json!({"fuel":1000000000,"local_model":model,"guest_profile":config}),
        deny,
    );
    assert_eq!(termination(&sampled), "exit");
    assert!(
        sampled[0]["engine"]["configuration"]
            .as_str()
            .unwrap()
            .contains("-guest-profile")
    );
    let profile: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    let metadata = &profile["meta"]["gomadGuestProfile"];
    assert_eq!(metadata["samples"], 1);
    assert_eq!(metadata["maxSamples"], 1);
    assert_eq!(metadata["truncated"], true);
    assert_eq!(
        profile["threads"][0]["samples"]["stack"]
            .as_array()
            .unwrap()
            .len(),
        1
    );
    assert!(profile.to_string().contains("hot"), "{profile}");
    // Existing files must not be overwritten, even though profiling succeeds.
    let (failed, stderr) = run_captured(
        &guest,
        json!({"fuel":1000000000,"local_model":model,"guest_profile":config}),
        deny,
        AuthorizationMismatch::None,
    );
    assert!(
        stderr.contains("gomad-wasm-guest-profile warning:"),
        "{stderr}"
    );
    for frames in [&sampled, &failed] {
        for (expected, actual) in baseline.iter().zip(frames.iter()) {
            let mut expected = expected.clone();
            let mut actual = actual.clone();
            expected.as_object_mut().unwrap().remove("metrics");
            actual.as_object_mut().unwrap().remove("metrics");
            assert_eq!(expected, actual);
        }
        assert_eq!(baseline.len(), frames.len());
    }
    assert_eq!(
        serde_json::from_slice::<Value>(&std::fs::read(&path).unwrap()).unwrap(),
        profile
    );
    std::fs::remove_dir_all(directory).unwrap();
}

#[cfg(feature = "guest-profile")]
#[test]
fn guest_profile_drops_epochs_elapsed_inside_host_calls() {
    let path = std::env::temp_dir().join(format!("gomad-host-gap-{}.json", std::process::id()));
    let frames = run(
        "(module (import \"wasi_snapshot_preview1\" \"sched_yield\" (func $yield (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") call $yield drop))",
        json!({"guest_profile":{"path":path,"interval_micros":1000,"max_samples":100}}),
        |call| {
            std::thread::sleep(std::time::Duration::from_millis(30));
            json!({"type":"reply","id":call["id"],"errno":0,"output":{}})
        },
    );
    assert_eq!(termination(&frames), "exit");
    let profile: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    assert_eq!(profile["meta"]["gomadGuestProfile"]["samples"], 0);
    std::fs::remove_file(path).unwrap();
}

#[cfg(feature = "guest-profile")]
#[test]
fn guest_profile_rejects_invalid_bounds_and_unknown_fields() {
    for config in [
        json!({"path":"relative","interval_micros":1000,"max_samples":1}),
        json!({"path":"/tmp/profile.json","interval_micros":0,"max_samples":1}),
        json!({"path":"/tmp/profile.json","interval_micros":1000001,"max_samples":1}),
        json!({"path":"/tmp/profile.json","interval_micros":1,"max_samples":100001}),
        json!({"path":"/tmp/profile.json","interval_micros":1,"max_samples":0}),
        json!({"path":"/tmp/profile.json","interval_micros":1,"max_samples":1,"unknown":true}),
    ] {
        let frames = run("(module)", json!({"guest_profile":config}), deny);
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(frames.len(), 1);
    }
}

#[cfg(feature = "diagnostic-profile")]
fn diagnostic_run(wat: &str, malformed_reply: bool) -> (Vec<Value>, Value) {
    use std::io::Read;
    let path = std::env::temp_dir().join(format!("gomad-profile-{}.wat", std::process::id()));
    std::fs::write(&path, wat).unwrap();
    let hash = format!("{:x}", Sha256::digest(wat.as_bytes()));
    let config = json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}});
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{}", json!({"schema":"gomad3.wasm-host/v1","type":"execute","module_path":path,
        "module_sha256":hash,"memory_bytes":131072,"fuel":1000000,"output_bytes":1024,"local_model":config})).unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":hash,"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "call" {
            let transcript = b"{\"type\":\"call\",\"id\":2,\"op\":\"fd_sync\",\"input\":{\"fd\":3}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{}}\n";
            writeln!(input, "{}", json!({"type":"reply","id":frame["id"],"errno":0,
                "output":if malformed_reply {json!([])} else {json!({})},
                "local_state":{"now":4,"advances":0},"transcript_base64":STANDARD.encode(transcript)})).unwrap();
        }
        frames.push(frame);
    }
    let mut stderr = Vec::new();
    child
        .stderr
        .take()
        .unwrap()
        .read_to_end(&mut stderr)
        .unwrap();
    assert!(child.wait().unwrap().success());
    std::fs::remove_file(path).unwrap();
    assert!(
        stderr.len() < 16 * 1024,
        "{}",
        String::from_utf8_lossy(&stderr)
    );
    assert_eq!(stderr.iter().filter(|&&byte| byte == b'\n').count(), 1);
    assert!(stderr.starts_with(b"gomad-wasm-profile-v1 "));
    let profile = serde_json::from_slice(&stderr[b"gomad-wasm-profile-v1 ".len()..]).unwrap();
    (frames, profile)
}

#[cfg(feature = "diagnostic-profile")]
#[test]
fn diagnostic_hooks_measure_real_boundaries_without_changing_evidence_or_failures() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"fd_sync\" (func $sync (param i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i64.const 0 i32.const 8 call $clock drop i32.const 3 call $sync drop))";
    let clock = b"{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":0,\"precision\":0}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"timestamp\":1001}}\n";
    for (wat, bad_reply, want, remote_count) in [
        (guest.to_owned(), false, "exit", 1),
        (guest.to_owned(), true, "invalid", 1),
        (
            guest.replace("i32.const 8", "i32.const 65535"),
            false,
            "trap",
            0,
        ),
    ] {
        let (frames, profile) = diagnostic_run(&wat, bad_reply);
        assert_eq!(termination(&frames), want, "{frames:?}");
        assert_eq!(profile["schema"], "gomad3.wasm-profile/v1");
        assert_eq!(profile["abi_overlap_errors"], 0);
        for phase in [
            "module_read",
            "module_hash",
            "engine_init",
            "compile",
            "instantiate",
            "execute",
        ] {
            assert_eq!(profile["phases"][phase]["count"], 1, "missing {phase}");
            assert!(profile["phases"][phase]["wall_nanos"].as_u64().unwrap() > 0);
        }
        assert_eq!(profile["abi"]["clock"]["count"], 1);
        assert_eq!(profile["abi"]["remote_callback"]["count"], remote_count);
        assert_eq!(profile["nested"]["remote_write"]["count"], remote_count);
        assert_eq!(profile["nested"]["remote_read_wait"]["count"], remote_count);
        let mut transcript = Vec::new();
        for frame in frames.iter().filter(|frame| frame["type"] == "evidence") {
            assert_eq!(frame["stream"], "transcript");
            transcript.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap())
                    .unwrap(),
            );
        }
        let mut expected: Vec<u8> = Vec::new();
        if want != "trap" {
            expected.extend(clock);
        }
        if want == "exit" {
            expected.extend(b"{\"type\":\"call\",\"id\":2,\"op\":\"fd_sync\",\"input\":{\"fd\":3}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{}}\n");
        }
        assert_eq!(transcript, expected);
        assert_eq!(
            frames.last().unwrap()["local_model"]["now"],
            if want == "trap" { 1 } else { 4 }
        );
    }
}

#[test]
fn local_clock_abi_preserves_literal_transcript_and_pointer_validation_priority() {
    let config = json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}});
    let guest = "(module (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i64.const 0 i32.const 8 call $clock drop i32.const 8 i64.load i64.const 1001 i64.ne if unreachable end i32.const 1 i64.const -1 i32.const 16 call $clock drop i32.const 16 i64.load i64.const 4 i64.ne if unreachable end))";
    let frames = run(guest, config.clone(), deny);
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    let expected = b"{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":0,\"precision\":0}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"timestamp\":1001}}\n{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":18446744073709551615}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{\"timestamp\":4}}\n";
    let mut transcript = Vec::new();
    for frame in frames
        .iter()
        .filter(|frame| frame["type"] == "evidence" && frame["stream"] == "transcript")
    {
        transcript.extend(
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap(),
        );
    }
    assert_eq!(transcript, expected);
    let result = frames.last().unwrap();
    assert_eq!(result["local_model"]["calls"], 2);
    assert_eq!(result["local_model"]["now"], 7);
    assert_eq!(
        result["local_model"]["transcript_sha256"],
        format!("{:x}", Sha256::digest(expected))
    );
    for (pointer, want, calls) in [(8, "unsupported", 1), (65535, "trap", 0)] {
        let guest = format!(
            "(module (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 2 i64.const 0 i32.const {pointer} call $clock drop unreachable))"
        );
        let frames = run(&guest, config.clone(), deny);
        assert_eq!(termination(&frames), want, "{frames:?}");
        let result = frames.last().unwrap();
        assert_eq!(result["local_model"]["calls"], calls);
        assert_eq!(result["local_model"]["now"], 1);
        if calls == 0 {
            assert_eq!(result["local_model"]["transcript_bytes"], 0);
        }
    }
}

#[test]
fn forwards_bounded_v2_stream_import_to_remote_model() {
    let guest = "(module (import \"gomad_wasm_v2\" \"net_stream\" (func $net (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 64 call $net drop))";
    for local in [false, true] {
        let config = if local {
            json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,
        "epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}})
        } else {
            json!({})
        };
        let frames = run(guest, config, |call| {
            assert_eq!(call["op"], "runtime_net_stream");
            assert_eq!(
                STANDARD
                    .decode(call["input"]["data_base64"].as_str().unwrap())
                    .unwrap()
                    .len(),
                64
            );
            let mut reply =
                json!({"type":"reply","id":call["id"],"errno":0,"output":call["input"]});
            if local {
                reply["local_state"] = json!({"now":1,"advances":0});
                reply["transcript_base64"] = json!("");
            }
            reply
        });
        assert_eq!(termination(&frames), "exit", "{frames:?}");
        if local {
            assert_eq!(frames.last().unwrap()["local_model"]["calls"], 1);
            assert_eq!(frames.last().unwrap()["local_model"]["transcript_bytes"], 0);
            assert!(!frames.iter().any(|frame| frame["type"] == "evidence"));
        }
    }
}

#[test]
fn typed_interrupt_stops_before_syscall_resume_and_flushes_actual_state() {
    let mut decision = [0u8; 192];
    decision[..4].copy_from_slice(&1u32.to_be_bytes());
    decision[4..8].copy_from_slice(&1u32.to_be_bytes());
    decision[20..24].copy_from_slice(&2u32.to_be_bytes());
    decision[32..64].fill(1);
    decision[64..96].fill(2);
    let literal: String = decision
        .iter()
        .map(|byte| format!("\\{byte:02x}"))
        .collect();
    for reason in ["storage_crash", "crash", "stop"] {
        for local in [false, true] {
            let guest = if local {
                format!(
                    "(module (import \"gomad_wasm_v1\" \"config\" (func $config (param i32 i32) (result i32))) (import \"gomad_wasm_v1\" \"decision\" (func $decision (param i32 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"fd_sync\" (func $sync (param i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"{literal}\") (func (export \"_start\") i32.const 0 i32.const 16 call $config drop i32.const 32 i32.const 192 call $decision drop i32.const 1 i64.const 0 i32.const 256 call $clock drop i32.const 3 call $sync drop unreachable))"
                )
            } else {
                "(module (import \"wasi_snapshot_preview1\" \"fd_sync\" (func $sync (param i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 3 call $sync drop unreachable))".into()
            };
            let config = if local {
                json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":true,
            "epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}})
            } else {
                json!({})
            };
            let frames = run(&guest, config, |call| {
                assert_eq!(call["op"], "fd_sync");
                json!({"type":"interrupt","termination":reason,"message":"modeled interruption: fault"})
            });
            assert_eq!(termination(&frames), reason, "{frames:?}");
            let result = frames.last().unwrap();
            assert_eq!(result["message"], "modeled interruption: fault");
            assert_eq!(result["peak_memory_bytes"], 65536);
            let fuel = result["fuel_remaining"].as_u64().unwrap();
            if cfg!(feature = "diagnostic-fuel-off") {
                assert_eq!(fuel, 1000000);
            } else {
                assert!(fuel > 0 && fuel < 1000000);
            }
            assert!(result["metrics"]["instantiate_nanos"].as_u64().unwrap() > 0);
            assert!(result["metrics"]["execute_nanos"].as_u64().unwrap() > 0);
            if local {
                assert_eq!(result["local_model"]["calls"], 3);
                assert_eq!(result["local_model"]["records"], 1);
                assert_eq!(result["local_model"]["finished"], false);
                let choice = frames
                    .iter()
                    .find(|frame| frame["type"] == "evidence" && frame["stream"] == "choice")
                    .unwrap();
                assert_eq!(
                    STANDARD
                        .decode(choice["data_base64"].as_str().unwrap())
                        .unwrap()
                        .len(),
                    96
                );
                assert!(
                frames
                    .iter()
                    .any(|frame| frame["type"] == "evidence" && frame["stream"] == "transcript")
            );
                let diagnostic = frames
                    .iter()
                    .find(|frame| frame["type"] == "evidence" && frame["stream"] == "diagnostic")
                    .unwrap();
                assert_eq!(
                    STANDARD
                        .decode(diagnostic["data_base64"].as_str().unwrap())
                        .unwrap()
                        .len(),
                    96
                );
            }
        }
    }
}

#[test]
fn typed_interrupt_rejects_unknown_termination_fields_and_oversized_message() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_sync\" (func $sync (param i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 3 call $sync drop))";
    for frame in [
        json!({"type":"interrupt","termination":"watchdog","message":"fault"}),
        json!({"type":"interrupt","termination":"CRASH","message":"fault"}),
        json!({"type":"interrupt","termination":null,"message":"fault"}),
        json!({"type":"interrupt","termination":"storage_crash","message":"fault","extra":true}),
        json!({"type":"interrupt","termination":"storage_crash","message":"x".repeat(65537)}),
    ] {
        let frames = run(guest, json!({}), |_| frame.clone());
        assert_eq!(termination(&frames), "invalid");
    }
}

#[test]
fn executes_tiny_guest_in_fresh_instances() {
    let guest = "(module (memory (export \"memory\") 1) (global $g (mut i32) (i32.const 0)) (func (export \"_start\") global.get $g if unreachable end i32.const 1 global.set $g))";
    for _ in 0..2 {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "exit", "{frames:?}");
        assert_eq!(frames[0]["engine"]["version"], "47.0.3");
    }
}

#[test]
fn reports_compile_cache_execution_and_resource_metrics_separately() {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let cache = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-metrics-{}-{}",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::create_dir(&cache).unwrap();
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
    }
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    for warm in [false, true] {
        let frames = run(guest, json!({"compile_cache":cache}), deny);
        assert_eq!(termination(&frames), "exit", "{frames:?}");
        let metrics = &frames.last().unwrap()["metrics"];
        assert_eq!(metrics["cache_hit"], warm);
        let number = |name: &str| metrics[name].as_u64().unwrap();
        assert!(
            number("prepare_nanos")
                >= number("compile_nanos")
                    + number("cache_load_nanos")
                    + number("cache_write_nanos")
        );
        assert!(number("cache_load_nanos") > 0);
        assert!(number("instantiate_nanos") > 0);
        assert!(number("execute_nanos") > 0);
        if warm {
            assert_eq!(number("compile_nanos"), 0);
            assert_eq!(number("cache_write_nanos"), 0);
        } else {
            assert!(number("compile_nanos") > 0);
            assert!(number("cache_write_nanos") > 0);
        }
        if cfg!(all(
            target_pointer_width = "64",
            any(target_arch = "aarch64", target_arch = "x86_64"),
            any(target_os = "linux", target_os = "macos")
        )) {
            assert_eq!(metrics["resource_usage_available"], true);
            assert!(number("peak_rss_bytes") > 0);
        }
    }
    std::fs::remove_dir_all(cache).unwrap();
}
#[cfg(unix)]
#[test]
fn cache_rejects_poison_stale_links_and_permissive_storage() {
    use std::os::unix::fs::{PermissionsExt, symlink};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    for case in [
        "corrupt",
        "stale",
        "symlink",
        "permissive",
        "hardlink",
        "directory_mode",
        "directory_link",
        "oversized",
    ] {
        let cache = std::env::temp_dir().join(format!(
            "gomad-cache-security-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&cache).unwrap();
        std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
        let initial = run(guest, json!({"compile_cache":cache}), deny);
        assert_eq!(termination(&initial), "exit", "{initial:?}");
        let entry = std::fs::read_dir(&cache)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .find(|path| {
                path.extension()
                    .is_some_and(|extension| extension == "cwasm" || extension == "gomad-cache-v2")
            })
            .unwrap();
        let other = cache.with_extension("other");
        let mut requested = cache.clone();
        match case {
            "corrupt" | "stale" => {
                let mut bytes = std::fs::read(&entry).unwrap();
                let offset = if case == "stale" { 60 } else { bytes.len() - 1 };
                bytes[offset] ^= 1;
                std::fs::write(&entry, bytes).unwrap();
            }
            "symlink" => {
                std::fs::rename(&entry, &other).unwrap();
                symlink(&other, &entry).unwrap();
            }
            "permissive" => {
                std::fs::set_permissions(&entry, std::fs::Permissions::from_mode(0o666)).unwrap()
            }
            "hardlink" => std::fs::hard_link(&entry, &other).unwrap(),
            "directory_mode" => {
                std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o777)).unwrap()
            }
            "directory_link" => {
                symlink(&cache, &other).unwrap();
                requested = other.clone();
            }
            "oversized" => std::fs::OpenOptions::new()
                .write(true)
                .open(&entry)
                .unwrap()
                .set_len(1024 * 1024 * 1024 + 1024 * 1024)
                .unwrap(),
            _ => unreachable!(),
        }
        let frames = run(guest, json!({"compile_cache":requested}), deny);
        assert_eq!(
            termination(&frames),
            if case == "oversized" {
                "capacity"
            } else {
                "infrastructure"
            },
            "case {case}: {frames:?}"
        );
        if matches!(
            case,
            "corrupt" | "stale" | "symlink" | "permissive" | "hardlink"
        ) {
            let message = frames.last().unwrap()["message"].as_str().unwrap();
            assert!(
                message.contains(&format!("corrupt entry {}", entry.display())),
                "case {case}: {message}"
            );
        }
        assert!(!frames.iter().any(|frame| frame["type"] == "started"));
        std::fs::remove_dir_all(cache).unwrap();
        if other.exists() || std::fs::symlink_metadata(&other).is_ok() {
            std::fs::remove_file(other).unwrap();
        }
    }
}
#[cfg(unix)]
#[test]
fn cache_serializes_parallel_publication_without_preserving_guest_heap() {
    use std::os::unix::fs::PermissionsExt;
    let cache = std::env::temp_dir().join(format!("gomad-cache-parallel-{}", std::process::id()));
    std::fs::create_dir(&cache).unwrap();
    std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
    let guest = "(module (memory (export \"memory\") 1) (data (i32.const 0) \"\\07\") (func (export \"_start\") i32.const 0 i32.load8_u i32.const 7 i32.ne if unreachable end i32.const 0 i32.const 18 i32.store8))";
    let old = cache.join(format!(
        "{:x}-wasmtime-47.0.3-cranelift-fuel-nan-canonical-v1.cwasm",
        Sha256::digest(guest.as_bytes())
    ));
    std::fs::write(&old, b"legacy poison never deserialized").unwrap();
    let threads: Vec<_> = (0..8)
        .map(|_| {
            let cache = cache.clone();
            std::thread::spawn(move || run(guest, json!({"compile_cache":cache}), deny))
        })
        .collect();
    let frames: Vec<_> = threads
        .into_iter()
        .map(|thread| thread.join().unwrap())
        .collect();
    for frames in &frames {
        assert_eq!(termination(frames), "exit", "{frames:?}");
    }
    assert_eq!(
        frames
            .iter()
            .filter(|frames| frames.last().unwrap()["metrics"]["cache_hit"] == false)
            .count(),
        1
    );
    assert_eq!(
        std::fs::read(&old).unwrap(),
        b"legacy poison never deserialized"
    );
    let entries: Vec<_> = std::fs::read_dir(&cache)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| {
            path.extension()
                .is_some_and(|extension| extension == "gomad-cache-v2")
        })
        .collect();
    assert_eq!(entries.len(), 1);
    let different = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") nop))",
        json!({"compile_cache":cache}),
        deny,
    );
    assert_eq!(termination(&different), "exit");
    assert_eq!(different.last().unwrap()["metrics"]["cache_hit"], false);
    std::fs::remove_dir_all(cache).unwrap();
}
#[cfg(target_os = "linux")]
#[test]
fn cache_identity_uses_running_inode_after_atomic_helper_replacement() {
    use std::os::unix::fs::PermissionsExt;
    let cache = std::env::temp_dir().join(format!("gomad-cache-exe-race-{}", std::process::id()));
    std::fs::create_dir(&cache).unwrap();
    std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
    let executable = cache.join("helper");
    std::fs::copy(env!("CARGO_BIN_EXE_gomad3-wasmhost"), &executable).unwrap();
    let original = format!("{:x}", Sha256::digest(std::fs::read(&executable).unwrap()));
    let mut child = Command::new(&executable)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .unwrap();
    let replacement = cache.join("replacement");
    std::fs::write(&replacement, b"not the running helper").unwrap();
    std::fs::rename(&replacement, &executable).unwrap();
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    let source = cache.join("guest.wat");
    std::fs::write(&source, guest).unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{}", json!({"schema":"gomad3.wasm-host/v1","type":"execute","module_path":source,"module_sha256":format!("{:x}", Sha256::digest(guest.as_bytes())),"memory_bytes":65536,"fuel":10000,"output_bytes":1024,"compile_cache":cache})).unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        frames.push(frame);
    }
    assert!(child.wait().unwrap().success());
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    let entry = std::fs::read_dir(&cache)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .find(|path| {
            path.extension()
                .is_some_and(|extension| extension == "gomad-cache-v2")
        })
        .unwrap();
    let envelope = std::fs::read(entry).unwrap();
    let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
    let manifest: Value = serde_json::from_slice(&envelope[52..52 + manifest_len]).unwrap();
    assert_eq!(manifest["helper_sha256"], original);
    std::fs::remove_dir_all(cache).unwrap();
}
#[cfg(not(feature = "diagnostic-fuel-off"))]
#[test]
fn bounds_infinite_loop_with_fuel() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") (loop $l br $l)))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.last().unwrap()["fuel_remaining"], 0);
}
#[test]
fn rejects_unknown_import_before_started() {
    let frames = run(
        "(module (import \"env\" \"clock\" (func)) (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "unsupported");
    assert_eq!(frames.len(), 1);
}
#[test]
fn rejects_changed_module_identity() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({"module_sha256":"0000000000000000000000000000000000000000000000000000000000000000"}),
        deny,
    );
    assert_eq!(termination(&frames), "invalid");
    assert_eq!(frames.len(), 1);
}
#[test]
fn bounds_initial_memory() {
    let frames = run(
        "(module (memory (export \"memory\") 3) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
}
#[test]
fn routes_guest_output_without_polluting_protocol() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_write\" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop i32.const 8 i32.load i32.const 2 i32.ne if unreachable end))";
    let frames = run(guest, json!({}), |call| {
        assert_eq!(call["op"], "fd_write");
        assert_eq!(call["input"], json!({"fd":1,"data_base64":"aGk="}));
        json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2}})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_wrong_import_signature() {
    let frames = run(
        "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func (param i64) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "unsupported");
    assert_eq!(frames.len(), 1);
}
#[test]
fn rejects_malformed_execute_and_unknown_fields() {
    for change in [
        json!({"schema":"other"}),
        json!({"fuel":0}),
        json!({"memory_bytes":1}),
        json!({"surprise":true}),
    ] {
        let frames = run(
            "(module (memory (export \"memory\") 1) (func (export \"_start\")))",
            change,
            deny,
        );
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn traps_growth_at_declared_limit() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") i32.const 2 memory.grow drop))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
}
#[test]
fn checks_abi_range_before_callback() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const -1 i32.const 16 call $random drop))";
    let frames = run(guest, json!({}), |_| {
        panic!("invalid memory reached host callback")
    });
    assert_eq!(termination(&frames), "trap");
}
#[test]
fn writes_host_entropy_into_guest_memory() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 2 call $random drop i32.const 0 i32.load16_u i32.const 513 i32.ne if unreachable end))";
    let frames = run(guest, json!({}), |call| {
        assert_eq!(call["input"], json!({"length":2}));
        json!({"type":"reply","id":call["id"],"errno":0,"output":{"data_base64":"AQI="}})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_malformed_or_mismatched_callback_reply() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 2 call $random drop))";
    for output in [
        json!({"data_base64":"AQ=="}),
        json!({"data_base64":"%%%"}),
        json!({"data_base64":"AQI=","extra":1}),
    ] {
        let frames = run(
            guest,
            json!({}),
            |call| json!({"type":"reply","id":call["id"],"errno":0,"output":output}),
        );
        assert_eq!(termination(&frames), "invalid");
    }
    let frames = run(
        guest,
        json!({}),
        |_| json!({"type":"reply","id":99,"errno":0,"output":{"data_base64":"AQI="}}),
    );
    assert_eq!(termination(&frames), "invalid");
}
#[test]
fn does_not_write_memory_for_errno() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 0) \"zz\") (func (export \"_start\") i32.const 0 i32.const 2 call $random i32.const 8 i32.ne if unreachable end i32.const 0 i32.load16_u i32.const 31354 i32.ne if unreachable end))";
    let frames = run(
        guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":8,"output":{}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn denies_unsupported_callback_explicitly() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"sock_shutdown\" (func $shutdown (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 3 i32.const 2 call $shutdown drop))";
    let frames = run(guest, json!({}), deny);
    assert_eq!(termination(&frames), "unsupported");
}
#[test]
fn enforces_output_limit_before_delivery() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_write\" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop))";
    let frames = run(guest, json!({"output_bytes":1}), |_| {
        panic!("over-limit output reached host")
    });
    assert_eq!(termination(&frames), "capacity");
}

#[test]
fn rejects_authorization_before_guest_start_section() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"sched_yield\" (func $yield (result i32))) (memory (export \"memory\") 1) (func $init call $yield drop) (start $init) (func (export \"_start\")))";
    for mismatch in [
        AuthorizationMismatch::Module,
        AuthorizationMismatch::Configuration,
    ] {
        let frames = run_authorized(guest, json!({}), deny, mismatch);
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(
            frames.len(),
            2,
            "guest code ran before authorization: {frames:?}"
        );
    }
}

fn checked_call(
    name: &str,
    signature: &str,
    body: &str,
    expected_input: Value,
    output: Value,
    check: &str,
) {
    let guest = format!(
        "(module (import \"wasi_snapshot_preview1\" \"{name}\" (func $call {signature} (result i32))) (memory (export \"memory\") 1) (data (i32.const 256) \"file\") (func (export \"_start\") {body} call $call drop {check}))"
    );
    let frames = run(&guest, json!({}), |call| {
        assert_eq!(call["op"], name);
        assert_eq!(call["input"], expected_input);
        json!({"type":"reply","id":call["id"],"errno":0,"output":output})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn marshals_clock_and_file_metadata_into_preview1_layouts() {
    checked_call(
        "clock_time_get",
        "(param i32 i64 i32)",
        "i32.const 1 i64.const 7 i32.const 64",
        json!({"clock_id":1,"precision":7}),
        json!({"timestamp":42}),
        "i32.const 64 i64.load i64.const 42 i64.ne if unreachable end",
    );
    checked_call(
        "fd_fdstat_get",
        "(param i32 i32)",
        "i32.const 4 i32.const 64",
        json!({"fd":4}),
        json!({"filetype":4,"flags":2,"rights_base":9,"rights_inheriting":7}),
        "i32.const 64 i32.load8_u i32.const 4 i32.ne if unreachable end i32.const 66 i32.load16_u i32.const 2 i32.ne if unreachable end i32.const 72 i64.load i64.const 9 i64.ne if unreachable end i32.const 80 i64.load i64.const 7 i64.ne if unreachable end",
    );
    checked_call(
        "fd_filestat_get",
        "(param i32 i32)",
        "i32.const 4 i32.const 64",
        json!({"fd":4}),
        json!({"dev":1,"ino":2,"filetype":4,"nlink":3,"size":9,"atim":11,"mtim":12,"ctim":13}),
        "i32.const 80 i32.load8_u i32.const 4 i32.ne if unreachable end i32.const 96 i64.load i64.const 9 i64.ne if unreachable end i32.const 120 i64.load i64.const 13 i64.ne if unreachable end",
    );
}
#[test]
fn decodes_path_open_rights_without_exposing_guest_pointers() {
    checked_call(
        "path_open",
        "(param i32 i32 i32 i32 i32 i64 i64 i32 i32)",
        "i32.const 3 i32.const 1 i32.const 256 i32.const 4 i32.const 9 i64.const 8192 i64.const 64 i32.const 2 i32.const 64",
        json!({"fd":3,"dirflags":1,"path_base64":"ZmlsZQ==","oflags":9,"rights_base":8192,"rights_inheriting":64,"fdflags":2}),
        json!({"fd":8}),
        "i32.const 64 i32.load i32.const 8 i32.ne if unreachable end",
    );
}
#[test]
fn scatters_partial_positional_read_into_iovecs() {
    checked_call(
        "fd_pread",
        "(param i32 i32 i32 i64 i32)",
        "i32.const 0 i32.const 64 i32.store i32.const 4 i32.const 2 i32.store i32.const 8 i32.const 80 i32.store i32.const 12 i32.const 2 i32.store i32.const 4 i32.const 0 i32.const 2 i64.const 9 i32.const 32",
        json!({"fd":4,"length":4,"offset":9}),
        json!({"data_base64":"AQID"}),
        "i32.const 32 i32.load i32.const 3 i32.ne if unreachable end i32.const 64 i32.load16_u i32.const 513 i32.ne if unreachable end i32.const 80 i32.load16_u i32.const 3 i32.ne if unreachable end",
    );
}
#[test]
fn marshals_ordered_poll_subscriptions_and_events() {
    checked_call(
        "poll_oneoff",
        "(param i32 i32 i32 i32)",
        "i32.const 0 i64.const 91 i64.store i32.const 8 i32.const 0 i32.store8 i32.const 16 i32.const 1 i32.store i32.const 24 i64.const 123 i64.store i32.const 32 i64.const 7 i64.store i32.const 40 i32.const 1 i32.store16 i32.const 0 i32.const 64 i32.const 1 i32.const 128",
        json!({"subscriptions":[{"userdata":91,"type":0,"clock_id":1,"timeout":123,"precision":7,"flags":1}]}),
        json!({"events":[{"userdata":91,"errno":0,"type":0,"nbytes":0,"flags":0}]}),
        "i32.const 64 i64.load i64.const 91 i64.ne if unreachable end i32.const 128 i32.load i32.const 1 i32.ne if unreachable end",
    );
}
#[test]
fn packs_captured_arguments_using_preceding_sizes() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"args_sizes_get\" (func $sizes (param i32 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"args_get\" (func $get (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 4 call $sizes drop i32.const 8 i32.const 32 call $get drop i32.const 8 i32.load i32.const 32 i32.ne if unreachable end i32.const 32 i32.load16_u i32.const 97 i32.ne if unreachable end i32.const 12 i32.load i32.const 34 i32.ne if unreachable end))";
    let frames = run(
        guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":0,"output":if call["op"]=="args_sizes_get"{json!({"count":2,"bytes":3})}else{json!({"values":["YQ==",""]})}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_out_of_bounds_iovec_without_callback() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_read\" (func $read (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const -1 i32.store i32.const 4 i32.const 2 i32.store i32.const 0 i32.const 0 i32.const 1 i32.const 8 call $read drop))";
    let frames = run(guest, json!({}), |_| panic!("invalid iovec reached host"));
    assert_eq!(termination(&frames), "trap");
}

#[test]
fn bounds_positional_writes_to_standard_output() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_pwrite\" (func $write (param i32 i32 i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i64.const 0 i32.const 8 call $write drop))";
    let frames = run(
        guest,
        json!({"output_bytes":1}),
        |call| json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2}}),
    );
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 2, "over-limit positional output reached host");
}
#[test]
fn classifies_initial_table_count_as_capacity() {
    let guest = format!(
        "(module {} (memory (export \"memory\") 1) (func (export \"_start\")))",
        "(table 0 funcref)".repeat(9)
    );
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 1);
}
#[test]
fn bounds_aggregate_callback_frame() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"path_rename\" (func $rename (param i32 i32 i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 112) (func (export \"_start\") i32.const 3 i32.const 0 i32.const 3145728 i32.const 3 i32.const 3145728 i32.const 3145728 call $rename drop))";
    let frames = run(guest, json!({"memory_bytes":8388608}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 2);
}
#[test]
fn classifies_disabled_valid_proposals_as_unsupported() {
    for guest in [
        "(module (memory (export \"memory\") i64 1) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1 2 shared) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1) (memory 1) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1) (func (export \"_start\") v128.const f32x4 0 0 0 0 v128.const f32x4 0 0 0 0 v128.const f32x4 0 0 0 0 f32x4.relaxed_madd drop))",
    ] {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn keeps_malformed_wat_and_binary_as_invalid() {
    for guest in [
        "(module (memory",
        "(module (memory (export \"memory\") 1) (func (export \"_start\") i32.const 1))",
        "\0asm\x01\0\0\0\x05\x01\x01",
    ] {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn classifies_missing_module_as_infrastructure() {
    let frames = run(
        "(module)",
        json!({"module_path":"/missing-gomad3-wasmhost-module"}),
        deny,
    );
    assert_eq!(termination(&frames), "infrastructure");
}
#[test]
fn bounds_started_import_inventory() {
    let guest=format!("(module {} (memory (export \"memory\") 1) (func (export \"_start\")))","(import \"wasi_snapshot_preview1\" \"fd_write\" (func (param i32 i32 i32 i32) (result i32)))".repeat(100000));
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 1);
}

#[test]
fn bounds_unsupported_import_diagnostics() {
    let guest = format!(
        "(module (import \"env\" \"{}\" (func)) (memory (export \"memory\") 1) (func (export \"_start\")))",
        "x".repeat(90000)
    );
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "unsupported");
    assert!(frames.last().unwrap()["message"].as_str().unwrap().len() <= 65536);
}

#[test]
fn classifies_disabled_atomic_instructions_as_unsupported() {
    for instruction in ["atomic.fence", "i32.const 0 i32.atomic.load drop"] {
        let guest = format!(
            "(module (memory (export \"memory\") 1) (func (export \"_start\") {instruction}))"
        );
        let frames = run(&guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn classifies_real_guest_unreachable_as_trap() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "trap");
}

fn runtime_guest(name: &str, pointer: i32, length: i32, data: &[u8], check: &str) -> String {
    let data: String = data.iter().map(|byte| format!("\\{byte:02x}")).collect();
    format!(
        "(module (import \"gomad_wasm_v1\" \"{name}\" (func $call (param i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 64) \"{data}\") (func (export \"_start\") i32.const {pointer} i32.const {length} call $call {check}))"
    )
}

#[test]
fn runtime_imports_copy_bounded_buffers_back_into_guest_memory() {
    for (name, length, count) in [
        ("config", 16, 0),
        ("finish", 16, 0),
        ("idle", 16, 0),
        ("decision", 96, 2),
        ("decision", 192, 2),
        ("decision", 8224, 256),
        ("decision", 8320, 256),
        ("observation", 96, 0),
        ("observation", 192, 0),
    ] {
        let mut bytes = vec![0u8; length];
        if count != 0 {
            bytes[20..24].copy_from_slice(&(count as u32).to_be_bytes());
        }
        let guest = runtime_guest(
            name,
            64,
            length as i32,
            &bytes,
            "if unreachable end i32.const 64 i32.load8_u i32.const 42 i32.ne if unreachable end",
        );
        let frames = run(&guest, json!({}), |call| {
            assert_eq!(call["op"], format!("runtime_{name}"));
            assert_eq!(
                call["input"],
                json!({"data_base64":STANDARD.encode(&bytes)})
            );
            let mut output = bytes.clone();
            output[0] = 42;
            json!({"type":"reply","id":call["id"],"errno":0,"output":{"data_base64":STANDARD.encode(output)}})
        });
        assert_eq!(termination(&frames), "exit", "{name}/{length}: {frames:?}");
    }
}

#[test]
fn runtime_imports_require_exact_module_name_and_signature_before_started() {
    for (module, name, signature) in [
        ("gomad_wasm_v1", "other", "(param i32 i32) (result i32)"),
        (
            "gomad_wasm_v1",
            "runtime_config",
            "(param i32 i32) (result i32)",
        ),
        (
            "gomad_wasm_v1",
            "random_get",
            "(param i32 i32) (result i32)",
        ),
        (
            "wasi_snapshot_preview1",
            "config",
            "(param i32 i32) (result i32)",
        ),
        ("other", "config", "(param i32 i32) (result i32)"),
        ("gomad_wasm_v1", "config", "(param i64 i32) (result i32)"),
        ("gomad_wasm_v1", "config", "(param i32 i32)"),
    ] {
        let guest = format!(
            "(module (import \"{module}\" \"{name}\" (func {signature})) (memory (export \"memory\") 1) (func (export \"_start\")))"
        );
        let frames = run(&guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1, "{frames:?}");
    }
}

#[test]
fn runtime_imports_reject_invalid_ranges_and_layouts_before_callback() {
    for (name, pointer, length, count, expected) in [
        ("config", -1, 16, 0, "trap"),
        ("config", 65528, 16, 0, "trap"),
        ("config", 64, 15, 0, "trap"),
        ("finish", 64, 17, 0, "capacity"),
        ("idle", 64, 0, 0, "trap"),
        ("observation", 64, 97, 0, "trap"),
        ("decision", 64, 96, 1, "trap"),
        ("decision", 64, 96, 257, "trap"),
        ("decision", 64, 128, 2, "trap"),
        ("decision", 64, 8321, 256, "capacity"),
        ("decision", 64, -1, 2, "capacity"),
    ] {
        let mut bytes = [0u8; 32];
        bytes[20..24].copy_from_slice(&(count as u32).to_be_bytes());
        let guest = runtime_guest(name, pointer, length, &bytes, "drop");
        let frames = run(&guest, json!({}), |_| {
            panic!("invalid {name} {pointer}/{length}/{count} reached host")
        });
        assert_eq!(
            termination(&frames),
            expected,
            "{name}/{length}: {frames:?}"
        );
        assert_eq!(frames.len(), 2, "{frames:?}");
    }
}

#[test]
fn runtime_imports_reject_malformed_and_wrong_length_replies() {
    let guest = runtime_guest("config", 64, 16, &[0u8; 16], "drop");
    for output in [
        json!({"data_base64":STANDARD.encode([0u8; 15])}),
        json!({"data_base64":STANDARD.encode([0u8; 17])}),
        json!({"data_base64":"%%%"}),
        json!({"data_base64":42}),
        json!({"data_base64":STANDARD.encode([0u8; 16]),"extra":1}),
        json!({"data_base64":"A".repeat(16384)}),
    ] {
        let frames = run(
            &guest,
            json!({}),
            |call| json!({"type":"reply","id":call["id"],"errno":0,"output":output}),
        );
        assert_eq!(termination(&frames), "invalid", "{frames:?}");
        assert_eq!(frames.len(), 3, "{frames:?}");
    }
}

#[test]
fn runtime_imports_preserve_guest_buffer_for_errno() {
    let guest = runtime_guest(
        "idle",
        64,
        16,
        &[42u8; 16],
        "i32.const 8 i32.ne if unreachable end i32.const 64 i32.load8_u i32.const 42 i32.ne if unreachable end",
    );
    let frames = run(
        &guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":8,"output":{}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
