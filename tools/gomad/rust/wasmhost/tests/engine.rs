use serde_json::{Value, json};
use sha2::{Digest, Sha256};
#[cfg(feature = "guest-profile")]
use std::fmt::Write as _;
use std::io::{BufRead, BufReader, Write};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

#[path = "common/profile_stderr.rs"]
mod profile_stderr;

fn run(wat: &str, change: Value, reply: impl FnMut(&Value) -> Value) -> Vec<Value> {
    run_authorized(wat, change, reply, AuthorizationMismatch::None)
}
#[derive(Clone, Copy)]
enum AuthorizationMismatch {
    None,
}
fn run_authorized(
    wat: &str,
    change: Value,
    reply: impl FnMut(&Value) -> Value,
    mismatch: AuthorizationMismatch,
) -> Vec<Value> {
    run_captured(wat, change, reply, mismatch).0
}
#[expect(
    clippy::needless_pass_by_value,
    reason = "The fixture transfers its constructed request overrides to the process driver."
)]
fn run_captured(
    wat: &str,
    change: Value,
    mut reply: impl FnMut(&Value) -> Value,
    _mismatch: AuthorizationMismatch,
) -> (Vec<Value>, String) {
    use std::io::Read;
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad-wasmhost-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, wat).unwrap();
    let mut request = json!({"schema":"gomad.wasm-host/v1", "type":"execute", "module_path":path, "module_sha256":format!("{:x}",Sha256::digest(wat.as_bytes())), "memory_bytes":131_072, "fuel":1_000_000, "output_bytes":1024});
    for (key, value) in change.as_object().unwrap() {
        if key == "fixture_model_inputs" || key == "fixture_namespace_inputs" {
            continue;
        }
        request[key] = value.clone();
    }
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    if let Some(frames) = change["fixture_model_inputs"].as_array() {
        for frame in frames {
            writeln!(input, "{frame}").unwrap();
        }
    } else if request["local_model"].is_object() {
        writeln!(input,"{}",json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A"})).unwrap();
        writeln!(input, "{}", json!({"type":"model_inputs_end"})).unwrap();
    }
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            let configuration = frame["engine"]["configuration"].clone();
            let module = frame["module_sha256"].clone();
            writeln!(
                input,
                "{}",
                json!({"type":"authorize","module_sha256":module,"configuration":configuration})
            )
            .unwrap();
            if let Some(inputs) = change["fixture_namespace_inputs"].as_array() {
                // A refused image ends the helper before it reads every frame.
                for input_frame in inputs {
                    if let Err(error) = writeln!(input, "{input_frame}") {
                        assert_eq!(error.kind(), std::io::ErrorKind::BrokenPipe);
                        break;
                    }
                }
            }
        }
        if frame["type"] == "call" || frame["type"] == "sync" {
            // An array scripts several frames in order (chunks, then a reply).
            match reply(&frame) {
                Value::Array(frames) => {
                    for scripted in frames {
                        writeln!(input, "{scripted}").unwrap();
                    }
                }
                single => writeln!(input, "{single}").unwrap(),
            }
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
    profile_stderr::validate(stderr.as_bytes()).unwrap();
    std::fs::remove_file(path).unwrap();
    (frames, stderr)
}
fn deny(call: &Value) -> Value {
    json!({"type":"reply", "id":call["id"], "errno":0, "output":{}, "unsupported":true, "message":"not modeled"})
}
fn termination(frames: &[Value]) -> &str {
    frames.last().unwrap()["termination"].as_str().unwrap()
}

#[test]
fn determinism_refuses_unknown_execute_fields_and_kinds_before_started() {
    for change in [
        json!({"future_field":true}),
        json!({"type":"future_execute"}),
    ] {
        let frames = run(
            "(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))",
            change,
            deny,
        );
        assert_eq!(frames.len(), 1, "{frames:?}");
        assert_eq!(termination(&frames), "unsupported");
        assert!(frames[0]["message"].as_str().unwrap().contains("future"));
    }
}

#[test]
fn determinism_refuses_unknown_remote_reply_fields_and_kinds() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"sched_yield\" (func $yield (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") call $yield drop unreachable))";
    for change in [json!({"future_field":true}), json!({"type":"future_reply"})] {
        let frames = run(guest, json!({}), |call| {
            let mut reply = json!({"type":"reply","id":call["id"],"errno":0,"output":{}});
            for (key, value) in change.as_object().unwrap() {
                reply[key] = value.clone();
            }
            reply
        });
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert!(
            frames.last().unwrap()["message"]
                .as_str()
                .unwrap()
                .contains("future")
        );
    }
}

#[test]
fn determinism_refuses_unknown_split_remote_reply_fields_and_kinds() {
    let fixture: Value = serde_json::from_str(include_str!(
        "testdata/helper_transcript/split_sync_before_local_call_continued.json"
    ))
    .unwrap();
    let mut request = fixture["request"].clone();
    request.as_object_mut().unwrap().remove("module_path");
    request.as_object_mut().unwrap().remove("module_sha256");
    request["fixture_model_inputs"] = fixture["inputs"].clone();
    request["fixture_namespace_inputs"] =
        Value::Array(fixture["replies"][0]["send"].as_array().unwrap()[1..].to_vec());
    for change in [
        json!({"future_field":true}),
        json!({"type":"future_reply"}),
        json!({"local_state":{"now":1,"advances":0,"future_field":true}}),
    ] {
        let frames = run(
            fixture["guest"]["wat"].as_str().unwrap(),
            request.clone(),
            |call| {
                let mut reply = json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2},"local_state":{"now":1,"advances":0},"transcript_base64":""});
                for (key, value) in change.as_object().unwrap() {
                    reply[key] = value.clone();
                }
                reply
            },
        );
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert!(
            frames.last().unwrap()["message"]
                .as_str()
                .unwrap()
                .contains("future")
        );
    }
    for refused in [
        json!({"type":"remote_transcript","data_base64":"AQ==","future":true}),
        json!({"type":"interrupt","termination":"stop","message":"","future":true}),
    ] {
        let frames = run(
            fixture["guest"]["wat"].as_str().unwrap(),
            request.clone(),
            |_| refused.clone(),
        );
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert!(
            frames.last().unwrap()["message"]
                .as_str()
                .unwrap()
                .contains("future")
        );
    }
    for refused in [
        json!({"type":"future_sync"}),
        json!({"type":"continue","now":7,"future":true}),
    ] {
        let frames = run(
            fixture["guest"]["wat"].as_str().unwrap(),
            request.clone(),
            |call| {
                if call["type"] == "sync" {
                    return refused.clone();
                }
                json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2},"local_state":{"now":5,"advances":0},"transcript_base64":"","revocable":true})
            },
        );
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert!(
            frames.last().unwrap()["message"]
                .as_str()
                .unwrap()
                .contains("future")
        );
    }
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
    let literal: String = decision.iter().fold(String::new(), |mut text, byte| {
        write!(text, "\\{byte:02x}").unwrap();
        text
    });
    let guest = format!(
        "(module (import \"gomad_wasm_v1\" \"config\" (func $config (param i32 i32) (result i32))) (import \"gomad_wasm_v1\" \"decision\" (func $decision (param i32 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"{literal}\") (func $hot (local $n i32) i32.const 10000000 local.set $n (loop $work local.get $n i32.const 1 i32.sub local.tee $n br_if $work)) (func $hot_entry (export \"_start\") i32.const 0 i32.const 16 call $config drop i32.const 32 i32.const 192 call $decision drop i32.const 1 i64.const 0 i32.const 256 call $clock drop call $hot))"
    );
    let model = json!({"seed":7,"mode":1,"limit":4096,"diagnostics":true,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}});
    let baseline = run(
        &guest,
        json!({"fuel":1_000_000_000,"local_model":model}),
        deny,
    );
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
        json!({"fuel":1_000_000_000,"local_model":model,"guest_profile":config}),
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
        json!({"fuel":1_000_000_000,"local_model":model,"guest_profile":config}),
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

#[test]
fn reports_compile_cache_execution_and_resource_metrics_separately() {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let cache = std::env::temp_dir().join(format!(
        "gomad-wasmhost-metrics-{}-{}",
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
fn cache_hit_ignores_missing_or_replaced_module_bytes() {
    use std::os::unix::fs::PermissionsExt;
    let cache = std::env::temp_dir().join(format!("gomad-cache-module-hit-{}", std::process::id()));
    std::fs::create_dir(&cache).unwrap();
    std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    let initial = run(guest, json!({"compile_cache":cache}), deny);
    assert_eq!(termination(&initial), "exit", "{initial:?}");
    assert_eq!(initial.last().unwrap()["metrics"]["cache_hit"], false);
    let source = cache.join("unused-module.wat");
    for replacement in [
        None,
        Some("(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))"),
    ] {
        if let Some(bytes) = replacement {
            std::fs::write(&source, bytes).unwrap();
        }
        let (frames, stderr) = run_captured(
            guest,
            json!({"compile_cache":cache,"module_path":source}),
            deny,
            AuthorizationMismatch::None,
        );
        assert_eq!(termination(&frames), "exit", "{frames:?}");
        assert_eq!(frames.last().unwrap()["metrics"]["cache_hit"], true);
        #[cfg(feature = "diagnostic-profile")]
        {
            let profile: Value = serde_json::from_str(
                stderr
                    .lines()
                    .find_map(|line| line.strip_prefix("gomad-wasm-profile-v1 "))
                    .unwrap(),
            )
            .unwrap();
            assert_eq!(profile["phases"]["module_read"]["count"], 0);
            assert_eq!(profile["phases"]["module_hash"]["count"], 0);
        }
        #[cfg(not(feature = "diagnostic-profile"))]
        assert_eq!(stderr, "");
    }
    std::fs::remove_dir_all(cache).unwrap();
}
#[cfg(unix)]
#[test]
fn cache_miss_verifies_module_before_compilation_or_publication() {
    use std::os::unix::fs::PermissionsExt;
    let cache =
        std::env::temp_dir().join(format!("gomad-cache-module-miss-{}", std::process::id()));
    std::fs::create_dir(&cache).unwrap();
    std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o700)).unwrap();
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    for configured_cache in [json!(cache), Value::Null] {
        let mismatch = run(
            guest,
            json!({"compile_cache":configured_cache,"module_sha256":"0".repeat(64)}),
            deny,
        );
        assert_eq!(termination(&mismatch), "invalid", "{mismatch:?}");
        assert_eq!(mismatch.len(), 1);
        assert_eq!(mismatch[0]["message"], "module SHA-256 mismatch");
        let missing = run(
            guest,
            json!({"compile_cache":configured_cache,"module_path":cache.join("missing-module.wat")}),
            deny,
        );
        assert_eq!(termination(&missing), "infrastructure", "{missing:?}");
        assert_eq!(missing.len(), 1);
        assert!(std::fs::read_dir(&cache).unwrap().all(|entry| {
            entry
                .unwrap()
                .path()
                .extension()
                .is_some_and(|extension| extension == "lock")
        }));
    }
    std::fs::remove_dir_all(cache).unwrap();
}
#[cfg(unix)]
#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The scenario retains its complete ordered setup and contract assertions."
)]
fn cache_rejects_poison_stale_links_and_permissive_storage() {
    use std::os::unix::fs::{PermissionsExt, symlink};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let guest = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    for case in [
        "corrupt",
        "stale",
        "truncated",
        "magic",
        "length",
        "manifest_oversized",
        "artifact_oversized",
        "length_overflow",
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
                    .is_some_and(|extension| extension == "gomad-cache-v3")
            })
            .unwrap();
        let other = cache.with_extension("other");
        let mut requested = cache.clone();
        match case {
            "corrupt" | "stale" => {
                let mut bytes = std::fs::read(&entry).unwrap();
                let offset = if case == "stale" { 24 } else { bytes.len() - 1 };
                bytes[offset] ^= 1;
                std::fs::write(&entry, bytes).unwrap();
            }
            "truncated" => std::fs::write(&entry, b"GOMADC03").unwrap(),
            "magic" => {
                let mut bytes = std::fs::read(&entry).unwrap();
                bytes[7] = b'2';
                std::fs::write(&entry, bytes).unwrap();
            }
            "length" => {
                let mut bytes = std::fs::read(&entry).unwrap();
                bytes[12..20].copy_from_slice(&0u64.to_le_bytes());
                std::fs::write(&entry, bytes).unwrap();
            }
            "manifest_oversized" | "artifact_oversized" | "length_overflow" => {
                let mut bytes = std::fs::read(&entry).unwrap();
                if case != "artifact_oversized" {
                    bytes[8..12].copy_from_slice(&u32::MAX.to_le_bytes());
                }
                if case != "manifest_oversized" {
                    bytes[12..20].copy_from_slice(&u64::MAX.to_le_bytes());
                }
                std::fs::write(&entry, bytes).unwrap();
            }
            "symlink" => {
                std::fs::rename(&entry, &other).unwrap();
                symlink(&other, &entry).unwrap();
            }
            "permissive" => {
                std::fs::set_permissions(&entry, std::fs::Permissions::from_mode(0o666)).unwrap();
            }
            "hardlink" => std::fs::hard_link(&entry, &other).unwrap(),
            "directory_mode" => {
                std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o777)).unwrap();
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
            "infrastructure",
            "case {case}: {frames:?}"
        );
        if matches!(
            case,
            "corrupt"
                | "stale"
                | "truncated"
                | "magic"
                | "length"
                | "manifest_oversized"
                | "artifact_oversized"
                | "length_overflow"
                | "oversized"
                | "symlink"
                | "permissive"
                | "hardlink"
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
    #[expect(
        clippy::needless_collect,
        reason = "All publishers must start before any thread is joined."
    )]
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
                .is_some_and(|extension| extension == "gomad-cache-v3")
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
    std::fs::copy(env!("CARGO_BIN_EXE_gomad-wasmhost"), &executable).unwrap();
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
    writeln!(input, "{}", json!({"schema":"gomad.wasm-host/v1","type":"execute","module_path":source,"module_sha256":format!("{:x}", Sha256::digest(guest.as_bytes())),"memory_bytes":65536,"fuel":10000,"output_bytes":1024,"compile_cache":cache})).unwrap();
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
                .is_some_and(|extension| extension == "gomad-cache-v3")
        })
        .unwrap();
    let envelope = std::fs::read(entry).unwrap();
    let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
    let manifest: Value = serde_json::from_slice(&envelope[24..24 + manifest_len]).unwrap();
    assert_eq!(manifest["helper_sha256"], original);
    std::fs::remove_dir_all(cache).unwrap();
}
