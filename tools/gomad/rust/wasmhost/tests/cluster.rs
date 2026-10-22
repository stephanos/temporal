//! fn-178.6 cluster host: N isolated stores in one helper process, compared
//! frame for frame with single-execution runs of the same instances.
#![cfg(not(any(feature = "diagnostic-fuel-off", feature = "guest-profile")))]
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::fmt::Write as _;
use std::io::{BufRead, BufReader, Read, Write};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

#[path = "common/inspection.rs"]
mod inspection;

#[path = "common/profile_stderr.rs"]
mod profile_stderr;

/// One module for every instance; behavior differs by configuration and by the
/// test's replies: write "hello\n" (remote `fd_write`), read the local clock,
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
    (local.set $n (i32.const 4000))
    (loop $spin
      (local.set $n (i32.sub (local.get $n) (i32.const 1)))
      (br_if $spin (local.get $n)))))"#;

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The scenario retains its complete ordered setup and contract assertions."
)]
fn manual_inspection_reaps_waiting_peer_and_preserves_never_started_node() {
    let mut pending = vec![0u8; 240];
    for (offset, value) in [(0, 1u32), (4, 1), (12, 224)] {
        pending[offset..offset + 4].copy_from_slice(&value.to_le_bytes());
    }
    for (offset, value) in [(16, 1u32), (20, 1), (36, 2), (40, 1)] {
        pending[offset..offset + 4].copy_from_slice(&value.to_be_bytes());
    }
    pending[24..32].copy_from_slice(&42u64.to_be_bytes());
    pending[48..80].fill(2);
    pending[80..112].fill(1);
    pending[224..232].copy_from_slice(&1u64.to_be_bytes());
    let data: String = pending.iter().fold(String::new(), |mut text, byte| {
        write!(text, "\\{byte:02x}").unwrap();
        text
    });
    let guest = format!(
        r#"(module
      (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
      (import "gomad_debug_v1" "exchange" (func $debug (param i32 i32) (result i32)))
      (memory (export "memory") 1)
      (data (i32.const 256) "{data}")
      (data (i32.const 1024) "\01\00\00\00\06\00\00\00\00\00\00\00\00\00\00\00")
      (func (export "_start")
        (drop (call $clock (i32.const 1) (i64.const 0) (i32.const 0)))
        (drop (call $debug (i32.const 256) (i32.const 240)))
        (if (i32.load (i32.const 264))
          (then (drop (call $debug (i32.const 1024) (i32.const 16))) unreachable))))"#
    );
    let (path, _) = module();
    std::fs::write(&path, &guest).unwrap();
    let request = json!({"schema":"gomad.wasm-host/v1","type":"cluster_execute",
      "module_path":path,"module_sha256":format!("{:x}",Sha256::digest(guest.as_bytes())),
      "manual_start":true,"broker":broker_config(),"instances":[
        {"id":"a","memory_bytes":65536,"fuel":1_000_000,"output_bytes":1024,"local_model":model()},
        {"id":"b","memory_bytes":65536,"fuel":1_000_000,"output_bytes":1024,"local_model":model()},
        {"id":"c","memory_bytes":65536,"fuel":1_000_000,"output_bytes":1024,"local_model":model()}]});
    let execute = |control: Option<Value>| {
        let inspecting = control.is_some();
        let mut child = Command::new(env!("CARGO_BIN_EXE_gomad-wasmhost"))
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .spawn()
            .unwrap();
        let mut input = child.stdin.take().unwrap();
        let mut output = BufReader::new(child.stdout.take().unwrap());
        let mut frames = Vec::new();
        let read = |output: &mut BufReader<std::process::ChildStdout>, frames: &mut Vec<Value>| {
            let mut line = String::new();
            assert_ne!(output.read_line(&mut line).unwrap(), 0);
            let value: Value = serde_json::from_str(&line).unwrap();
            frames.push(value.clone());
            value
        };
        writeln!(input, "{request}").unwrap();
        for line in model_inputs(&request) {
            writeln!(input, "{line}").unwrap();
        }
        let mut started = read(&mut output, &mut frames);
        while started["type"] == "boot_started" {
            started = read(&mut output, &mut frames);
        }
        assert_eq!(started["type"], "started", "{started}");
        writeln!(input, "{}", json!({"type":"authorize","module_sha256":started["module_sha256"],"configuration":started["engine"]["configuration"]})).unwrap();
        let mut id = 0;
        for node in ["a", "b"] {
            id += 1;
            let mut start = json!({"type":"start","id":id,"node":node,"incarnation":1,"boot":node});
            if node == "a"
                && let Some(control) = &control
            {
                let retained: Value =
                    serde_json::from_str(include_str!("testdata/replay_plan/helper-runtime.json"))
                        .unwrap();
                let trace = STANDARD
                    .decode(retained["trace"].as_str().unwrap())
                    .unwrap();
                start["runtime"] = json!({"mode":2,"replay":{"schema":"gomad.replay-trace/v1","execution_identity":retained["identity"],"trace_sha256":format!("{:x}",Sha256::digest(&trace)),"tape_sha256":retained["expected"]["sha256"],"decisions":1,"trace_bytes":trace.len(),"trace_frames":1},"inspection":control});
                writeln!(input, "{start}").unwrap();
                for kind in ["replay_trace", "replay_trace_end"] {
                    writeln!(input,"{}",json!({"instance":"a","incarnation":1,"type":kind,"schema":"gomad.replay-trace/v1","trace_bytes":trace.len(),"trace_frames":1})).unwrap();
                    if kind == "replay_trace" {
                        input
                            .write_all(&u32::try_from(trace.len()).unwrap().to_be_bytes())
                            .unwrap();
                        input.write_all(&trace).unwrap();
                    }
                }
            } else {
                writeln!(input, "{start}").unwrap();
            }
            let ack = read(&mut output, &mut frames);
            assert_eq!(ack["termination"], "", "{ack}");
        }
        for node in if inspecting {
            vec!["a"]
        } else {
            vec!["a", "b"]
        } {
            id += 1;
            writeln!(
                input,
                "{}",
                json!({"type":"run","id":id,"node":node,"incarnation":1})
            )
            .unwrap();
            let mut terminal = read(&mut output, &mut frames);
            assert_eq!(terminal["type"], "control_result", "{terminal}");
            loop {
                terminal = read(&mut output, &mut frames);
                if terminal["type"] == "result" && terminal["instance"] == node {
                    break;
                }
            }
        }
        id += 1;
        writeln!(input, "{}", json!({"type":"finish","id":id})).unwrap();
        while frames.last().unwrap()["type"] != "cluster_result" {
            read(&mut output, &mut frames);
        }
        assert!(child.wait().unwrap().success());
        frames
    };
    let recorded = execute(None);
    assert_eq!(recorded.last().unwrap()["termination"], "exit");
    let recorded_results: Vec<_> = recorded
        .iter()
        .filter(|frame| frame["type"] == "result")
        .map(|frame| (frame["instance"].clone(), frame["termination"].clone()))
        .collect();
    assert_eq!(
        recorded_results,
        vec![(json!("a"), json!("exit")), (json!("b"), json!("exit"))],
        "{recorded:?}"
    );
    let raw: Vec<u8> = recorded
        .iter()
        .filter(|f| f["stream"] == "inspection")
        .flat_map(|f| STANDARD.decode(f["data_base64"].as_str().unwrap()).unwrap())
        .collect();
    let evidence: Value = serde_json::from_slice(&raw).unwrap();
    let boundaries = inspection::boundaries(&evidence);
    let boundary = &boundaries[0];
    let cut = &boundary["cuts"][0];
    let hash = |v: &[u8]| format!("{:x}", Sha256::digest(v));
    let decode = |v: &Value| STANDARD.decode(v.as_str().unwrap()).unwrap();
    let mut hashes = serde_json::Map::new();
    for (name, size) in [
        ("choice", "choice_bytes"),
        ("wasi", "wasi_bytes"),
        ("network", "network_bytes"),
        ("storage", "storage_bytes"),
        ("stdout", "stdout_bytes"),
        ("stderr", "stderr_bytes"),
    ] {
        let bytes = decode(&evidence["incarnations"][0]["streams"][name]);
        hashes.insert(
            name.into(),
            json!(hash(
                &bytes[..usize::try_from(cut[size].as_u64().unwrap()).unwrap()]
            )),
        );
    }
    let world = decode(&evidence["world"]);
    let lifecycle = decode(&evidence["lifecycle"]);
    let control = json!({"target":{"node":"a","incarnation":1},"decision":0,"expected":boundary,
      "world_sha256":hash(&world[..usize::try_from(boundary["world_bytes"].as_u64().unwrap()).unwrap()]),
      "lifecycle_sha256":hash(&lifecycle[..usize::try_from(boundary["lifecycle_bytes"].as_u64().unwrap()).unwrap()]),
      "stream_sha256":[hashes],"probes":[],"commands":[],"registries":[evidence["incarnations"][0]["registry"]]});
    let inspected = execute(Some(control));
    let result = inspected.last().unwrap();
    assert_eq!(result["termination"], "inspected_prefix", "{inspected:?}");
    assert!(inspected.iter().any(|f| f["type"] == "result"
        && f["instance"] == "b"
        && f["termination"] == "inspection_aborted"));
    assert!(
        !inspected
            .iter()
            .any(|f| f["type"] == "result" && f["instance"] == "c")
    );
    assert!(!inspected.iter().any(|f| f["stream"] == "world"));
    std::fs::remove_file(path).unwrap();
}

fn model() -> Value {
    json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],
        "working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
        "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},
        "epoch_nanos":1000,"read_step_nanos":3,"now":1,
        "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}})
}

/// The model-input frames (environment `PWD=/`, then the end marker) that follow
/// the execute frame for every cooperative instance, tagged in a cluster.
fn model_inputs(request: &Value) -> Vec<String> {
    let frames = [
        json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A"}),
        json!({"type":"model_inputs_end"}),
    ];
    if request["type"] == "execute" {
        if request["local_model"].is_object() {
            return frames.iter().map(Value::to_string).collect();
        }
        return Vec::new();
    }
    let mut instances: Vec<&Value> = request["instances"]
        .as_array()
        .map(|instances| instances.iter().collect())
        .unwrap_or_default();
    instances.sort_by_key(|instance| instance["id"].as_str().unwrap_or_default().to_owned());
    instances.extend(request["boots"].as_array().into_iter().flatten());
    let mut lines = Vec::new();
    for instance in instances {
        if instance["local_model"].is_object() {
            let id = instance["id"].as_str().unwrap();
            for frame in &frames {
                let text = frame.to_string();
                lines.push(format!("{{\"instance\":\"{id}\",{}", &text[1..]));
            }
        }
    }
    lines
}

fn module() -> (std::path::PathBuf, String) {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad-wasmhost-cluster-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, GUEST).unwrap();
    (path, format!("{:x}", Sha256::digest(GUEST.as_bytes())))
}

/// Drives one helper process, answering calls with `answer(instance, call)`
/// and broker pauses with `answer(None, pause)`, an array of control frames.
fn drive(
    request: Value,
    answer: impl FnMut(Option<&str>, &Value) -> Value,
    authorize: bool,
) -> Vec<Value> {
    drive_raw(request, answer, authorize).0
}

/// As `drive`, also returning each frame's exact line.
#[expect(
    clippy::needless_pass_by_value,
    reason = "The fixture transfers its constructed request to the process driver."
)]
fn drive_raw(
    request: Value,
    mut answer: impl FnMut(Option<&str>, &Value) -> Value,
    authorize: bool,
) -> (Vec<Value>, Vec<String>) {
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    for line in model_inputs(&request) {
        // A request refused before its inputs may close stdin first.
        if let Err(error) = writeln!(input, "{line}") {
            assert_eq!(error.kind(), std::io::ErrorKind::BrokenPipe, "{error}");
            break;
        }
    }
    let mut frames = Vec::new();
    let mut lines = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let line = line.unwrap();
        let frame: Value = serde_json::from_str(&line).unwrap();
        lines.push(line);
        if frame["type"] == "started" && authorize {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "pause" {
            // A broker settle pause: the answer is the control frames, then resume.
            let answer = answer(None, &frame);
            for control in answer.as_array().unwrap() {
                writeln!(input, "{control}").unwrap();
            }
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
    profile_stderr::validate(stderr.as_bytes()).unwrap();
    (frames, lines)
}

/// Go `brokerTestConfig`'s World.
fn broker_config() -> Value {
    json!({"seed":7,"limits":{"max_requests":10000,"max_events":10000,"max_queued_events":1000,
        "max_transitions":40000,"max_payload_bytes":33_554_432,"max_string_bytes":1024}})
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

/// One recorded Go broker order (`testdata/broker_orders`) as a guest: each
/// node is a straight-line function selected by the instance's seed, whose
/// call data is the recorded step's (references resolved to Go's recorded
/// outputs), trapping unless each output equals Go's byte for byte.
fn order_guest(scenario: &Value) -> String {
    order_guest_terminal(scenario, None)
}

#[expect(
    clippy::too_many_lines,
    reason = "The scenario retains its complete ordered setup and contract assertions."
)]
fn order_guest_terminal(scenario: &Value, trapped: Option<&str>) -> String {
    let outputs = &scenario["outputs"];
    let output = |node: &str, step: usize| -> Vec<u8> {
        STANDARD
            .decode(outputs[node][step]["data_base64"].as_str().unwrap())
            .unwrap()
    };
    let resolve = |step: &Value, key: &str, at: usize, width: usize| -> Vec<u8> {
        match step.get(key).filter(|value| !value.is_null()) {
            None => vec![0; width],
            Some(reference) => output(
                reference["node"].as_str().unwrap(),
                usize::try_from(reference["step"].as_u64().unwrap()).unwrap(),
            )[at..at + width]
                .to_vec(),
        }
    };
    let bytes = |data: &[u8]| -> String {
        data.iter().fold(String::new(), |mut text, byte| {
            write!(text, "\\{byte:02x}").unwrap();
            text
        })
    };
    let mut segments = String::new();
    let mut functions = String::new();
    let mut dispatch = String::new();
    let mut next = 4096usize;
    for (index, node) in scenario["nodes"].as_array().unwrap().iter().enumerate() {
        let node = node.as_str().unwrap();
        let mut body = String::new();
        for (at, step) in scenario["steps"][node]
            .as_array()
            .unwrap()
            .iter()
            .enumerate()
        {
            let field = |key: &str| step.get(key).and_then(Value::as_u64).unwrap_or(0);
            let (data, import) = if step["op"] == "idle" {
                let mut data = vec![0u8; 16];
                data[..4].copy_from_slice(&1u32.to_be_bytes());
                data[8..].copy_from_slice(&field("deadline").to_be_bytes());
                (data, "$idle")
            } else {
                let (code, count) = (
                    u32::try_from(field("code")).unwrap(),
                    u32::try_from(field("count")).unwrap(),
                );
                let mut data = vec![0u8; 64];
                data[..4].copy_from_slice(&2u32.to_be_bytes());
                data[4..8].copy_from_slice(&code.to_be_bytes());
                data[8..16].copy_from_slice(&field("id").to_be_bytes());
                data[16..24].copy_from_slice(&resolve(step, "handle", 16, 8));
                data[32..36].copy_from_slice(&resolve(step, "port", 48, 4));
                let header = if code == 12 { 0 } else { count };
                data[36..40].copy_from_slice(&header.to_be_bytes());
                for (offset, byte) in step["address"].as_array().unwrap().iter().enumerate() {
                    data[56 + offset] = u8::try_from(byte.as_u64().unwrap()).unwrap();
                }
                match step["payload"].as_str() {
                    Some(payload) if code == 5 => data.extend(payload.as_bytes()),
                    _ => data.extend(vec![0; count as usize]),
                }
                (data, "$stream")
            };
            let want = output(node, at);
            assert_eq!(want.len(), data.len(), "{node} step {at}");
            let (input, expected) = (next, next + data.len());
            next = expected + want.len();
            write!(
                segments,
                "  (data (i32.const {input}) \"{}\")\n  (data (i32.const {expected}) \"{}\")\n",
                bytes(&data),
                bytes(&want)
            )
            .unwrap();
            write!(body,
                "    (memory.copy (i32.const 0) (i32.const {input}) (i32.const {len}))\n    (if (call {import} (i32.const 0) (i32.const {len})) (then unreachable))\n    (call $same (i32.const {expected}) (i32.const {len}))\n",
                len = data.len()
            ).unwrap();
        }
        if trapped == Some(node) {
            body.push_str("    unreachable\n");
        }
        writeln!(functions, "  (func $node{index}\n{body}  )").unwrap();
        writeln!(dispatch,
            "    (if (i32.eq (local.get $seed) (i32.const {})) (then (call $node{index}) (return)))",
            index + 1
        ).unwrap();
    }
    assert!(next < 65536, "recorded order exceeds one page");
    format!(
        r#"(module
  (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "idle" (func $idle (param i32 i32) (result i32)))
  (import "gomad_wasm_v2" "net_stream" (func $stream (param i32 i32) (result i32)))
  (memory (export "memory") 1)
{segments}  (func $same (param $want i32) (param $len i32) (local $at i32)
    (loop $next
      (if (i32.lt_u (local.get $at) (local.get $len)) (then
        (if (i32.ne (i32.load8_u (local.get $at)) (i32.load8_u (i32.add (local.get $want) (local.get $at))))
          (then unreachable))
        (local.set $at (i32.add (local.get $at) (i32.const 1)))
        (br $next)))))
{functions}  (func (export "_start") (local $seed i32)
    (if (call $config (i32.const 1024) (i32.const 16)) (then unreachable))
    ;; The seed is big-endian at [8:16]; every node's seed is below 256.
    (if (i64.ne (i64.load (i32.const 1032)) (i64.and (i64.load (i32.const 1032)) (i64.const 0xff00000000000000)))
      (then unreachable))
    (local.set $seed (i32.load8_u (i32.const 1039)))
{dispatch}    unreachable))"#
    )
}

fn write_module(prefix: &str, module: &str) -> std::path::PathBuf {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad-wasmhost-{prefix}-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, module).unwrap();
    path
}

/// The helper's broker World expanded wire, from its untagged evidence frames.
fn world_evidence(frames: &[Value]) -> Vec<u8> {
    let mut world = Vec::new();
    for frame in frames
        .iter()
        .filter(|frame| frame["stream"] == "world" && frame.get("instance").is_none())
    {
        world.extend(
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap(),
        );
    }
    world
}

#[test]
fn cluster_broker_runs_recorded_go_orders_with_central_faults() {
    let directory =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/broker_orders");
    let mut ran = Vec::new();
    for entry in std::fs::read_dir(&directory).unwrap() {
        let path = entry.unwrap().path();
        if path.extension().and_then(|extension| extension.to_str()) != Some("json") {
            continue;
        }
        let scenario: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        let name = scenario["name"].as_str().unwrap().to_owned();
        // A guest cannot move its own clock; that order stays a replay test.
        let steps = scenario["steps"].as_object().unwrap();
        if steps
            .values()
            .flat_map(|steps| steps.as_array().unwrap())
            .any(|step| step["op"] == "advance_now")
        {
            continue;
        }
        let module = order_guest(&scenario);
        let module_path = write_module("cluster-order", &module);
        let network = scenario.get("network").filter(|network| !network.is_null());
        let nodes: Vec<&str> = scenario["nodes"]
            .as_array()
            .unwrap()
            .iter()
            .map(|node| node.as_str().unwrap())
            .collect();
        let instances: Vec<Value> = nodes
            .iter()
            .enumerate()
            .map(|(index, id)| {
                let mut model = model();
                model["seed"] = (index as u64 + 1).into();
                model["read_step_nanos"] = 0.into();
                model["limits"]["transcript_bytes"] = (1u64 << 20).into();
                let mut instance = json!({"id":id,"memory_bytes":65536,"fuel":100_000_000,
                    "output_bytes":1024,"local_model":model});
                if let Some(network) = network {
                    instance["network_v2"] = network.clone();
                }
                instance
            })
            .collect();
        let mut request = json!({"schema":"gomad.wasm-host/v1","type":"cluster_execute",
            "module_path":module_path,"module_sha256":format!("{:x}", Sha256::digest(module.as_bytes())),
            "instances":instances,"broker":broker_config()});
        if let Some(network) = network {
            let faults: Vec<Value> = scenario["faults"]
                .as_array()
                .map(|faults| {
                    faults
                        .iter()
                        .map(|fault| {
                            json!({"name":fault["Name"],"action":fault["Action"],"from":fault["From"],
                                "to":fault["To"],"at_nanos":fault["AtNanos"],
                                "delay_nanos":fault["DelayNanos"],"symmetric":fault["Symmetric"]})
                        })
                        .collect()
                })
                .unwrap_or_default();
            request["network"] = json!({"model":network,"faults":faults});
        }
        let frames = drive(
            request,
            |_, call| panic!("{name}: broker cluster called back to Go: {call}"),
            true,
        );
        std::fs::remove_file(module_path).unwrap();
        let last = frames.last().unwrap();
        assert_eq!(last["termination"], "exit", "{name}: {frames:?}");
        for id in &nodes {
            assert_eq!(result_of(&frames, id)["termination"], "exit", "{name} {id}");
            if network.is_some() {
                let want = std::fs::read(directory.join(format!("{name}.{id}.gmnetev1"))).unwrap();
                assert!(
                    network_evidence(&frames, id) == want,
                    "{name} {id}: network evidence differs from the recorded order"
                );
            }
        }
        assert_eq!(
            STANDARD.encode(world_evidence(&frames)),
            scenario["world_base64"].as_str().unwrap(),
            "{name}: World evidence differs from the Go broker's"
        );
        ran.push(name);
    }
    ran.sort();
    assert!(
        ran.contains(&"ping-pong-partition-heal".to_owned())
            && ran.contains(&"heal-before-stall-expiry".to_owned())
            && ran.len() >= 5,
        "{ran:?}"
    );
}
