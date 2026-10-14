//! fn-178.6 cluster host: N isolated stores in one helper process, compared
//! frame for frame with single-execution runs of the same instances.
// Diagnostic builds refuse clusters, leaving only the refusal test.
#![cfg_attr(feature = "diagnostic-profile", allow(dead_code))]
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::{BufRead, BufReader, Read, Write};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

/// One module for every instance; behavior differs by configuration and by the
/// test's replies: write "hello\n" (remote fd_write), read the local clock,
/// trap on a nonzero write errno, grow memory to three pages, then spin.
const GUEST: &str = r#"(module
  (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 16) "hello\n")
  (func (export "_start") (local $n i32)
    (i32.store (i32.const 0) (i32.const 16))
    (i32.store (i32.const 4) (i32.const 6))
    (if (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 8)) (then unreachable))
    (drop (call $clock (i32.const 0) (i64.const 0) (i32.const 32)))
    (if (i32.lt_s (memory.grow (i32.const 2)) (i32.const 0)) (then unreachable))
    (local.set $n (i32.const 2000))
    (loop $spin
      (local.set $n (i32.sub (local.get $n) (i32.const 1)))
      (br_if $spin (local.get $n)))))"#;

fn model() -> Value {
    json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":1000,
        "read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}})
}

fn module() -> (std::path::PathBuf, String) {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-cluster-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, GUEST).unwrap();
    (path, format!("{:x}", Sha256::digest(GUEST.as_bytes())))
}

/// The reply a test host gives an instance's fd_write; `errno` steers traps.
fn reply(call: &Value, errno: u16) -> Value {
    let mut reply = json!({"type":"reply","id":call["id"],"errno":errno,"output":{}});
    if errno == 0 {
        reply["output"] = json!({"written":6});
    }
    if let Some(state) = call.get("local_state") {
        reply["local_state"] = json!({"now":state["now"],"advances":state["advances"]});
        reply["transcript_base64"] = STANDARD.encode(format!("{call}\n{reply}\n")).into();
    }
    reply
}

/// Drives one helper process, answering calls with `answer(instance, call)`.
fn drive(
    request: Value,
    answer: impl FnMut(Option<&str>, &Value) -> Value,
    authorize: bool,
) -> Vec<Value> {
    drive_raw(request, answer, authorize).0
}

/// As `drive`, also returning each frame's exact line.
fn drive_raw(
    request: Value,
    mut answer: impl FnMut(Option<&str>, &Value) -> Value,
    authorize: bool,
) -> (Vec<Value>, Vec<String>) {
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    let mut frames = Vec::new();
    let mut lines = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let line = line.unwrap();
        let frame: Value = serde_json::from_str(&line).unwrap();
        lines.push(line);
        if frame["type"] == "started" && authorize {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "call" {
            let instance = frame.get("instance").and_then(Value::as_str);
            let mut single = frame.clone();
            single.as_object_mut().unwrap().remove("instance");
            let mut reply = answer(instance, &single);
            if let Some(id) = instance {
                // Tag the reply: the prefix replaces the opening brace.
                let text = reply.to_string();
                writeln!(input, "{{\"instance\":\"{id}\",{}", &text[1..]).unwrap();
                reply = Value::Null;
            }
            if !reply.is_null() {
                writeln!(input, "{reply}").unwrap();
            }
        }
        frames.push(frame);
    }
    drop(input);
    let mut stderr = String::new();
    child
        .stderr
        .take()
        .unwrap()
        .read_to_string(&mut stderr)
        .unwrap();
    assert!(child.wait().unwrap().success(), "{stderr}");
    (frames, lines)
}

#[derive(Clone)]
struct Instance {
    id: &'static str,
    memory_bytes: u64,
    fuel: u64,
    errno: u16,
}

fn instance_config(instance: &Instance) -> Value {
    json!({"id":instance.id,"memory_bytes":instance.memory_bytes,"fuel":instance.fuel,
        "output_bytes":1024,"local_model":model()})
}

fn single(instance: &Instance) -> Vec<Value> {
    single_raw(instance).0
}

fn single_raw(instance: &Instance) -> (Vec<Value>, Vec<String>) {
    let (path, digest) = module();
    let mut request = instance_config(instance);
    let object = request.as_object_mut().unwrap();
    object.remove("id");
    object.insert("schema".into(), "gomad3.wasm-host/v1".into());
    object.insert("type".into(), "execute".into());
    object.insert("module_path".into(), json!(path));
    object.insert("module_sha256".into(), digest.into());
    let errno = instance.errno;
    let frames = drive_raw(request, |_, call| reply(call, errno), true);
    std::fs::remove_file(path).unwrap();
    frames
}

type Split = BTreeMap<String, Vec<Value>>;

/// Runs a cluster; per instance, the de-tagged frames and their exact lines
/// with the prefix replaced by `{`.
fn cluster_raw(instances: &[Instance]) -> (Vec<Value>, Split, BTreeMap<String, Vec<String>>) {
    let (path, digest) = module();
    let request = json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
        "module_sha256":digest,"instances":instances.iter().map(instance_config).collect::<Vec<_>>()});
    let errnos: BTreeMap<&str, u16> = instances
        .iter()
        .map(|instance| (instance.id, instance.errno))
        .collect();
    let (frames, lines) = drive_raw(
        request,
        |instance, call| reply(call, errnos[instance.unwrap()]),
        true,
    );
    std::fs::remove_file(path).unwrap();
    let mut split: Split = BTreeMap::new();
    let mut raw: BTreeMap<String, Vec<String>> = BTreeMap::new();
    for (frame, line) in frames.iter().zip(&lines) {
        if let Some(id) = frame.get("instance").and_then(Value::as_str) {
            let mut frame = frame.clone();
            frame.as_object_mut().unwrap().remove("instance");
            split.entry(id.to_owned()).or_default().push(frame);
            let prefix = format!("{{\"instance\":\"{id}\",");
            assert!(line.starts_with(&prefix), "{line}");
            raw.entry(id.to_owned())
                .or_default()
                .push(format!("{{{}", &line[prefix.len()..]));
        }
    }
    (frames, split, raw)
}

/// A single run's instance frames: everything after `started`, without the
/// process-only `metrics`.
fn instance_frames(frames: &[Value]) -> Vec<Value> {
    frames
        .iter()
        .filter(|frame| frame["type"] != "started")
        .map(|frame| {
            let mut frame = frame.clone();
            frame.as_object_mut().unwrap().remove("metrics");
            frame
        })
        .collect()
}

fn failing() -> Vec<Instance> {
    vec![
        Instance {
            id: "0-trap",
            memory_bytes: 262144,
            fuel: 1_000_000,
            errno: 8,
        },
        Instance {
            id: "1-memory",
            memory_bytes: 131072,
            fuel: 1_000_000,
            errno: 0,
        },
        Instance {
            id: "2-fuel",
            memory_bytes: 262144,
            fuel: 2_000,
            errno: 0,
        },
        Instance {
            id: "3-exit",
            memory_bytes: 262144,
            fuel: 1_000_000,
            errno: 0,
        },
    ]
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_instances_match_their_single_runs_and_isolate_failures() {
    let instances = failing();
    let (frames, split, raw) = cluster_raw(&instances);
    let started: Vec<_> = frames
        .iter()
        .filter(|frame| frame["type"] == "started")
        .collect();
    assert_eq!(started.len(), 1, "one compilation and one started frame");
    let last = frames.last().unwrap();
    assert_eq!(last["type"], "cluster_result", "{last}");
    assert_eq!(last["termination"], "exit", "{last}");
    assert_eq!(
        last["instances"],
        json!([{"id":"0-trap","termination":"trap"},{"id":"1-memory","termination":"capacity"},
            {"id":"2-fuel","termination":"capacity"},{"id":"3-exit","termination":"exit"}])
    );
    assert!(last["metrics"]["peak_rss_bytes"].as_u64().unwrap() > 0);
    for instance in &instances {
        let (alone, lines) = single_raw(instance);
        assert_eq!(
            split[instance.id],
            instance_frames(&alone),
            "{} differs from its single run",
            instance.id
        );
        // Byte-for-byte: every frame but the result (whose single-run form
        // also carries `metrics`) is the identical line.
        let single_lines: Vec<&String> = alone
            .iter()
            .zip(&lines)
            .filter(|(frame, _)| frame["type"] != "started" && frame["type"] != "result")
            .map(|(_, line)| line)
            .collect();
        let cluster_lines: Vec<&String> = raw[instance.id]
            .iter()
            .filter(|line| !line.starts_with("{\"type\":\"result\""))
            .collect();
        assert_eq!(cluster_lines, single_lines, "{} line bytes", instance.id);
        assert!(!single_lines.is_empty());
    }
    let memory = split["1-memory"].last().unwrap();
    // The limiter refused growth past memory_bytes; the peak stays at the
    // initial page, and the survivors' peak is their grown three pages.
    assert_eq!(memory["peak_memory_bytes"], 65536, "{memory}");
    assert!(
        memory["message"]
            .as_str()
            .unwrap()
            .contains("guest linear memory capacity")
    );
    assert_eq!(split["3-exit"].last().unwrap()["peak_memory_bytes"], 196608);
    assert!(
        split["2-fuel"].last().unwrap()["message"]
            .as_str()
            .unwrap()
            .contains("fuel")
    );
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_survivor_evidence_is_unaffected_by_failing_neighbors() {
    let healthy: Vec<Instance> = failing()
        .into_iter()
        .map(|mut instance| {
            instance.memory_bytes = 262144;
            instance.fuel = 1_000_000;
            instance.errno = 0;
            instance
        })
        .collect();
    let (frames, clean, clean_lines) = cluster_raw(&healthy);
    assert_eq!(frames.last().unwrap()["termination"], "exit");
    let (_, _, failed_lines) = cluster_raw(&failing());
    assert_eq!(
        failed_lines["3-exit"], clean_lines["3-exit"],
        "survivor bytes changed next to failing instances"
    );
    assert!(
        clean["3-exit"]
            .iter()
            .any(|frame| frame["type"] == "evidence" && frame["stream"] == "transcript")
    );
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_refuses_invalid_configuration_before_started() {
    let (path, digest) = module();
    let base = |instances: Value| {
        json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
            "module_sha256":digest,"instances":instances})
    };
    let one = |id: &str| json!({"id":id,"memory_bytes":131072,"fuel":1000,"output_bytes":1024});
    let mut unknown = one("a");
    unknown["network_v2"] = json!({});
    let mut bad_limit = one("a");
    bad_limit["memory_bytes"] = json!(1000);
    let many: Vec<Value> = (0..65).map(|index| one(&format!("n{index}"))).collect();
    let mut cases = vec![
        base(json!([])),
        base(json!([one("a"), one("a")])),
        base(json!([one("bad id")])),
        base(json!([one("")])),
        base(json!([unknown])),
        base(json!([bad_limit])),
        base(Value::Array(many)),
    ];
    let mut extra = base(json!([one("a")]));
    extra["broker"] = json!({});
    cases.push(extra);
    for request in cases {
        let frames = drive(request.clone(), |_, _| unreachable!(), true);
        assert_eq!(frames.len(), 1, "{request}: {frames:?}");
        assert_eq!(frames[0]["type"], "cluster_result");
        assert_eq!(frames[0]["termination"], "invalid", "{request}");
    }
    std::fs::remove_file(path).unwrap();
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_unattributable_reply_ends_the_whole_run() {
    let (path, digest) = module();
    let instance = json!({"id":"a","memory_bytes":262144,"fuel":1000000,"output_bytes":1024});
    let other = json!({"id":"b","memory_bytes":262144,"fuel":1000000,"output_bytes":1024});
    let request = json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
        "module_sha256":digest,"instances":[instance, other]});
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "call" {
            // Addressed to the wrong instance.
            writeln!(
                input,
                "{{\"instance\":\"b\",\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{{\"written\":6}}}}"
            )
            .unwrap();
        }
        frames.push(frame);
    }
    assert!(child.wait().unwrap().success());
    let last = frames.last().unwrap();
    assert_eq!(last["type"], "cluster_result");
    assert_eq!(last["termination"], "invalid", "{last}");
    assert_eq!(last["instances"], json!([]));
    assert!(!frames.iter().any(|frame| frame["type"] == "result"
        || frame.get("termination").is_some() && frame["type"] != "cluster_result"));
    std::fs::remove_file(path).unwrap();
}

#[test]
fn single_execution_protocol_is_unchanged_by_cluster_dispatch() {
    let instance = Instance {
        id: "unused",
        memory_bytes: 262144,
        fuel: 1_000_000,
        errno: 0,
    };
    let frames = single(&instance);
    assert_eq!(frames.last().unwrap()["type"], "result");
    assert_eq!(frames.last().unwrap()["termination"], "exit");
    assert!(frames.iter().all(|frame| frame.get("instance").is_none()));
}

#[cfg(feature = "diagnostic-profile")]
#[test]
fn diagnostic_builds_refuse_cluster_execution() {
    let (path, digest) = module();
    let request = json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
        "module_sha256":digest,"instances":[{"id":"a","memory_bytes":131072,"fuel":1000,"output_bytes":1024}]});
    let frames = drive(request, |_, _| unreachable!(), true);
    assert_eq!(frames.len(), 1);
    assert_eq!(frames[0]["termination"], "unsupported");
    std::fs::remove_file(path).unwrap();
}

/// Runs a cluster of `module` with explicit tape lines sent after `authorize`.
fn cluster_with(module: &str, instances: Value, tapes: &[String]) -> Vec<Value> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-cluster-custom-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, module).unwrap();
    let request = json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
        "module_sha256":format!("{:x}", Sha256::digest(module.as_bytes())),"instances":instances});
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
            for tape in tapes {
                writeln!(input, "{tape}").unwrap();
            }
        }
        frames.push(frame);
    }
    assert!(child.wait().unwrap().success());
    std::fs::remove_file(path).unwrap();
    frames
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_initial_memory_is_judged_per_instance() {
    let module = "(module (memory (export \"memory\") 2) (func (export \"_start\")))";
    let instances = json!([
        {"id":"big","memory_bytes":196608,"fuel":1000,"output_bytes":1024},
        {"id":"small","memory_bytes":65536,"fuel":1000,"output_bytes":1024}
    ]);
    let frames = cluster_with(module, instances, &[]);
    let last = frames.last().unwrap();
    assert_eq!(
        last["instances"],
        json!([{"id":"big","termination":"exit"},{"id":"small","termination":"capacity"}]),
        "{frames:?}"
    );
    let small = frames
        .iter()
        .find(|frame| frame["instance"] == "small" && frame["type"] == "result")
        .unwrap();
    assert!(
        small["message"]
            .as_str()
            .unwrap()
            .contains("guest linear memory capacity")
    );
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_invalid_tape_fails_only_its_instance() {
    let module = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
    let replay = |chunks: u64| {
        let mut model = model();
        model["mode"] = 2.into();
        model["tape_chunks"] = chunks.into();
        model
    };
    let instances = json!([
        {"id":"a","memory_bytes":65536,"fuel":1000,"output_bytes":1024,"local_model":replay(2)},
        {"id":"b","memory_bytes":65536,"fuel":1000,"output_bytes":1024,"local_model":replay(0)}
    ]);
    // Both of a's declared chunks arrive; the first is invalid (no decisions).
    let tape = "{\"instance\":\"a\",\"type\":\"tape\",\"decisions\":[]}".to_owned();
    let frames = cluster_with(module, instances.clone(), &[tape.clone(), tape]);
    let last = frames.last().unwrap();
    assert_eq!(last["termination"], "exit", "{frames:?}");
    assert_eq!(
        last["instances"],
        json!([{"id":"a","termination":"invalid"},{"id":"b","termination":"exit"}])
    );
    let refused = frames
        .iter()
        .find(|frame| frame["instance"] == "a" && frame["type"] == "result")
        .unwrap();
    assert_eq!(refused["message"], "invalid tape chunk");
    // A tape frame addressed to another instance cannot be attributed.
    let foreign = "{\"instance\":\"b\",\"type\":\"tape\",\"decisions\":[]}".to_owned();
    let frames = cluster_with(module, instances, &[foreign]);
    assert_eq!(frames.last().unwrap()["termination"], "invalid");
    assert_eq!(frames.last().unwrap()["instances"], json!([]));
}

/// A cooperative guest that listens on its own node's server, dials it, accepts,
/// writes "hello" from the client side and reads it on the server side, all
/// through `gomad_wasm_v2::net_stream` and without idling. `TAIL` follows.
fn network_guest(tail: &str) -> String {
    format!(
        r#"(module
  (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
  (import "gomad_wasm_v2" "net_stream" (func $stream (param i32 i32) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 512) "hello")
  (func $be32 (param $at i32) (param $v i32)
    (i32.store8 (local.get $at) (i32.shr_u (local.get $v) (i32.const 24)))
    (i32.store8 (i32.add (local.get $at) (i32.const 1)) (i32.shr_u (local.get $v) (i32.const 16)))
    (i32.store8 (i32.add (local.get $at) (i32.const 2)) (i32.shr_u (local.get $v) (i32.const 8)))
    (i32.store8 (i32.add (local.get $at) (i32.const 3)) (local.get $v)))
  (func $be64 (param $at i32) (param $v i64)
    (call $be32 (local.get $at) (i32.wrap_i64 (i64.shr_u (local.get $v) (i64.const 32))))
    (call $be32 (i32.add (local.get $at) (i32.const 4)) (i32.wrap_i64 (local.get $v))))
  (func $get32 (param $at i32) (result i32)
    (i32.or
      (i32.or (i32.shl (i32.load8_u (local.get $at)) (i32.const 24))
              (i32.shl (i32.load8_u (i32.add (local.get $at) (i32.const 1))) (i32.const 16)))
      (i32.or (i32.shl (i32.load8_u (i32.add (local.get $at) (i32.const 2))) (i32.const 8))
              (i32.load8_u (i32.add (local.get $at) (i32.const 3))))))
  (func $get64 (param $at i32) (result i64)
    (i64.or (i64.shl (i64.extend_i32_u (call $get32 (local.get $at))) (i64.const 32))
            (i64.extend_i32_u (call $get32 (i32.add (local.get $at) (i32.const 4))))))
  (func $net (param $op i32) (param $id i64) (param $handle i64) (param $port i32) (param $count i32) (param $len i32)
    (memory.fill (i32.const 0) (i32.const 0) (i32.const 64))
    (call $be32 (i32.const 0) (i32.const 2))
    (call $be32 (i32.const 4) (local.get $op))
    (call $be64 (i32.const 8) (local.get $id))
    (call $be64 (i32.const 16) (local.get $handle))
    (call $be32 (i32.const 32) (local.get $port))
    (call $be32 (i32.const 36) (local.get $count))
    (if (call $stream (i32.const 0) (local.get $len)) (then unreachable))
    (if (call $get32 (i32.const 40)) (then unreachable))
    (if (call $get32 (i32.const 44)) (then unreachable)))
  (func (export "_start") (local $listener i64) (local $port i32) (local $client i64) (local $server i64) (local $n i32)
    (if (call $config (i32.const 1024) (i32.const 16)) (then unreachable))
    (call $net (i32.const 1) (i64.const 1) (i64.const 0) (i32.const 0) (i32.const 0) (i32.const 64))
    (local.set $listener (call $get64 (i32.const 16)))
    (local.set $port (call $get32 (i32.const 48)))
    (call $net (i32.const 2) (i64.const 2) (i64.const 0) (local.get $port) (i32.const 0) (i32.const 64))
    (local.set $client (call $get64 (i32.const 16)))
    (call $net (i32.const 3) (i64.const 3) (local.get $listener) (i32.const 0) (i32.const 0) (i32.const 64))
    (local.set $server (call $get64 (i32.const 16)))
    (memory.copy (i32.const 64) (i32.const 512) (i32.const 5))
    (call $net (i32.const 5) (i64.const 4) (local.get $client) (i32.const 0) (i32.const 5) (i32.const 69))
    (if (i32.ne (call $get32 (i32.const 36)) (i32.const 5)) (then unreachable))
    (memory.fill (i32.const 64) (i32.const 0) (i32.const 5))
    (call $net (i32.const 4) (i64.const 5) (local.get $server) (i32.const 0) (i32.const 5) (i32.const 69))
    (if (i32.ne (call $get32 (i32.const 36)) (i32.const 5)) (then unreachable))
    (if (i32.ne (i32.load (i32.const 64)) (i32.load (i32.const 512))) (then unreachable))
    {tail}))"#
    )
}

fn network_limits() -> Value {
    json!({"listeners":128,"connections":128,"bytes":8388608,"connection_bytes":4194304,
        "stall_nanos":10000000000u64,"transitions":8192})
}

fn network_instance(id: &str, memory_bytes: u64, fuel: u64, transcript: u64) -> Value {
    let mut model = model();
    model["limits"]["transcript_bytes"] = transcript.into();
    json!({"id":id,"memory_bytes":memory_bytes,"fuel":fuel,"output_bytes":1024,"local_model":model,
        "network_v2":{"profile":"gomad3.simulation-network-stream/v3","limits":network_limits()}})
}

fn network_cluster(module: &str, instances: Value) -> Vec<Value> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-cluster-network-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, module).unwrap();
    let request = json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":path,
        "module_sha256":format!("{:x}", Sha256::digest(module.as_bytes())),"instances":instances,
        "network":{"model":{"profile":"gomad3.simulation-network-stream/v3","limits":network_limits()}}});
    let frames = drive(
        request,
        |_, call| panic!("network cluster called back to Go: {call}"),
        true,
    );
    std::fs::remove_file(path).unwrap();
    frames
}

fn network_evidence(frames: &[Value], id: &str) -> Vec<u8> {
    let mut evidence = Vec::new();
    for frame in frames
        .iter()
        .filter(|frame| frame["instance"] == id && frame["stream"] == "network")
    {
        evidence.extend(
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap(),
        );
    }
    evidence
}

fn result_of<'a>(frames: &'a [Value], id: &str) -> &'a Value {
    frames
        .iter()
        .find(|frame| frame["instance"] == id && frame["type"] == "result")
        .unwrap()
}

fn read_varint(data: &[u8], at: &mut usize) -> u64 {
    let mut value = 0u64;
    for shift in (0..64).step_by(7) {
        let byte = data[*at];
        *at += 1;
        value |= u64::from(byte & 0x7f) << shift;
        if byte & 0x80 == 0 {
            break;
        }
    }
    value
}

/// The helper's Go-JSON accounting of the operations log, from the header.
fn log_json_bytes(evidence: &[u8]) -> u64 {
    assert_eq!(&evidence[..8], b"GMNETEV1");
    let mut at = 8;
    assert_eq!(read_varint(evidence, &mut at), 1);
    let profile = read_varint(evidence, &mut at) as usize;
    at += profile;
    read_varint(evidence, &mut at)
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_network_streams_are_served_in_the_helper() {
    let module = network_guest("");
    let frames = network_cluster(
        &module,
        json!([network_instance("a", 65536, 10_000_000, 1 << 20)]),
    );
    let last = frames.last().unwrap();
    assert_eq!(last["type"], "cluster_result");
    assert_eq!(
        last["instances"],
        json!([{"id":"a","termination":"exit"}]),
        "{frames:?}"
    );
    assert!(!frames.iter().any(|frame| frame["type"] == "call"));
    let evidence = network_evidence(&frames, "a");
    assert!(log_json_bytes(&evidence) > 0);
    assert!(evidence.len() as u64 > 64);
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_network_evidence_follows_every_terminal_and_its_bound() {
    // Exit, then trap and fuel/memory capacity after the same exchange.
    let spin = "(local.set $n (i32.const 100000)) (loop $spin (local.set $n (i32.sub (local.get $n) (i32.const 1))) (br_if $spin (local.get $n)))";
    let grow = "(drop (memory.grow (i32.const 4)))";
    let module = network_guest(&format!("{grow} {spin}"));
    let instances = json!([
        network_instance("a-exit", 393216, 10_000_000, 1 << 20),
        network_instance("b-memory", 65536, 10_000_000, 1 << 20),
        network_instance("c-fuel", 393216, 60_000, 1 << 20)
    ]);
    let frames = network_cluster(&module, instances);
    assert_eq!(
        frames.last().unwrap()["instances"],
        json!([{"id":"a-exit","termination":"exit"},{"id":"b-memory","termination":"capacity"},
            {"id":"c-fuel","termination":"capacity"}]),
        "{frames:?}"
    );
    for id in ["a-exit", "b-memory", "c-fuel"] {
        assert!(
            network_evidence(&frames, id).starts_with(b"GMNETEV1"),
            "{id}"
        );
    }
    let trap = network_guest("unreachable");
    let frames = network_cluster(
        &trap,
        json!([network_instance("t", 65536, 10_000_000, 1 << 20)]),
    );
    assert_eq!(result_of(&frames, "t")["termination"], "trap");
    assert!(network_evidence(&frames, "t").starts_with(b"GMNETEV1"));

    // An evidence bound between an instance's own log and its document (which
    // also carries the shared Record, including its neighbors' transitions)
    // turns the outcome into network-v2-evidence-bytes capacity, without
    // touching a later neighbor's evidence.
    let clean = network_guest("");
    let three = |bound| {
        network_cluster(
            &clean,
            json!([
                network_instance("0-free", 65536, 10_000_000, 1 << 20),
                network_instance("1-bound", 65536, 10_000_000, bound),
                network_instance("2-free", 65536, 10_000_000, 1 << 20)
            ]),
        )
    };
    let free = three(1 << 20);
    let document = network_evidence(&free, "1-bound");
    let log = log_json_bytes(&document);
    assert!(
        log < document.len() as u64,
        "log {log} document {}",
        document.len()
    );
    let bounded = three((log + document.len() as u64) / 2);
    let result = result_of(&bounded, "1-bound");
    assert_eq!(result["termination"], "capacity", "{result}");
    assert_eq!(
        result["message"],
        "WASI capacity: network-v2-evidence-bytes"
    );
    assert!(result.get("exit_code").is_none());
    assert!(network_evidence(&bounded, "1-bound").is_empty());
    assert_eq!(
        network_evidence(&bounded, "2-free"),
        network_evidence(&free, "2-free")
    );
    assert_eq!(result_of(&free, "1-bound")["termination"], "exit");
}

#[cfg(not(feature = "diagnostic-profile"))]
#[test]
fn cluster_network_configuration_is_refused_before_started() {
    let module = network_guest("");
    let base = |network: Value, instances: Value| {
        json!({"schema":"gomad3.wasm-host/v1","type":"cluster_execute","module_path":"/nonexistent.wat",
            "module_sha256":format!("{:x}", Sha256::digest(module.as_bytes())),"instances":instances,
            "network":network})
    };
    let model = json!({"profile":"gomad3.simulation-network-stream/v3","limits":network_limits()});
    let one = network_instance("a", 65536, 1000, 1 << 20);
    let mut local_fault = one.clone();
    local_fault["network_v2"]["faults"] = json!([{"name":"x","action":"partition","at_nanos":1}]);
    let mut without = one.clone();
    without.as_object_mut().unwrap().remove("network_v2");
    let cases = [
        (
            base(
                json!({"model":model,"addresses":{"a":"300.0.0.1"}}),
                json!([one]),
            ),
            "WASI invalid: broker IPv4 address",
        ),
        (
            base(
                json!({"model":model,"links":[{"from":"a","to":"z","enabled":true}]}),
                json!([one]),
            ),
            "WASI invalid: broker link endpoint",
        ),
        (
            base(
                json!({"model":model,"faults":[{"name":"f","action":"reset","from":"a","to":"a","at_nanos":1}]}),
                json!([one]),
            ),
            "WASI invalid: broker fault action",
        ),
        (
            base(json!({"model":model}), json!([local_fault])),
            "WASI invalid: broker node local fault schedule",
        ),
        (
            base(json!({"model":model}), json!([without])),
            "WASI invalid: broker node requires stream network profile",
        ),
    ];
    for (request, message) in cases {
        let frames = drive(request.clone(), |_, _| unreachable!(), true);
        assert_eq!(frames.len(), 1, "{request}");
        assert_eq!(frames[0]["termination"], "invalid");
        assert_eq!(frames[0]["message"], message);
    }
    let mut no_network = base(
        Value::Null,
        json!([network_instance("a", 65536, 1000, 1 << 20)]),
    );
    no_network.as_object_mut().unwrap().remove("network");
    let frames = drive(no_network, |_, _| unreachable!(), true);
    assert_eq!(
        frames[0]["message"],
        "WASI invalid: broker node network without cluster network"
    );
}
