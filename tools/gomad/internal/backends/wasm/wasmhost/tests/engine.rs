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
        if key == "fixture_model_inputs" || key == "fixture_namespace_inputs" {
            continue;
        }
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
            if matches!(mismatch, AuthorizationMismatch::None)
                && let Some(inputs) = change["fixture_namespace_inputs"].as_array()
            {
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
fn input_model() -> Value {
    json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}})
}

fn captured_namespace_image(file_path: &[u8]) -> Vec<u8> {
    captured_namespace_image_with(file_path, b"q")
}

fn captured_namespace_image_with(file_path: &[u8], file_data: &[u8]) -> Vec<u8> {
    fn blob(bytes: &mut Vec<u8>, value: &[u8]) {
        bytes.extend_from_slice(&(value.len() as u64).to_le_bytes());
        bytes.extend_from_slice(value);
    }
    let mut bytes = b"GMWN\x01\0\0\0".to_vec();
    for value in [3u64, 16, 1024, 1024, 1000, 2, 0, 1, 1] {
        bytes.extend_from_slice(&value.to_le_bytes());
    }
    for value in [b"/capt".as_slice(), b"/capt", b"/capt", b""] {
        blob(&mut bytes, value);
    }
    for (kind, path, data) in [
        (1, b"/capt".as_slice(), b"".as_slice()),
        (2, file_path, file_data),
    ] {
        bytes.extend_from_slice(&[kind, 1]);
        blob(&mut bytes, path);
        blob(&mut bytes, data);
    }
    bytes.push(0);
    bytes
}

fn namespace_request(bytes: &[u8]) -> Value {
    namespace_request_for(bytes, "/capt", 3)
}

fn namespace_request_for(bytes: &[u8], cwd: &str, files: u64) -> Value {
    let mut config = input_model();
    let environment = format!("PWD={cwd}\0");
    config["working_directory_sha256"] = json!(format!("{:x}", Sha256::digest(cwd.as_bytes())));
    config["environment"] = json!({"bytes":environment.len(),"count":1,"chunks":1});
    let mut frames: Vec<_> = bytes.chunks(65536).enumerate().map(|(sequence, chunk)| {
        json!({"type":"namespace_input","sequence":sequence,"data_base64":STANDARD.encode(chunk)})
    }).collect();
    frames.push(json!({"type":"namespace_inputs_end"}));
    json!({"local_model":config,
        "namespace":{"chunks":bytes.len().div_ceil(65536),"files":files,"bytes":bytes.len(),"sha256":format!("{:x}",Sha256::digest(bytes))},
        "fixture_model_inputs":[{"type":"model_input","stream":"environment","sequence":0,"data_base64":STANDARD.encode(environment.as_bytes())},{"type":"model_inputs_end"}],
        "fixture_namespace_inputs":frames})
}

fn writable_namespace_request(files: u64, descriptors: u64, filesystem_bytes: u64) -> Value {
    writable_namespace_request_at(b"/tmp", files, descriptors, filesystem_bytes)
}

fn writable_namespace_request_at(
    mount: &[u8],
    files: u64,
    descriptors: u64,
    filesystem_bytes: u64,
) -> Value {
    let mut bytes = b"GMWN\x01\0\0\0".to_vec();
    for value in [files, descriptors, filesystem_bytes, 1024, 1000, 1, 1, 0, 1] {
        bytes.extend_from_slice(&value.to_le_bytes());
    }
    for value in [mount, mount, mount, b""] {
        bytes.extend_from_slice(&(value.len() as u64).to_le_bytes());
        bytes.extend_from_slice(value);
    }
    bytes.extend_from_slice(&[1, 0]);
    bytes.extend_from_slice(&(mount.len() as u64).to_le_bytes());
    bytes.extend_from_slice(mount);
    bytes.extend_from_slice(&0u64.to_le_bytes());
    bytes.push(0);
    namespace_request_for(&bytes, std::str::from_utf8(mount).unwrap(), 2)
}

#[test]
fn namespace_capacity_and_unsupported_boundaries_stay_typed() {
    let open = "i32.const 3 i32.const 0 i32.const 0 i32.const 1 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 8 call $open drop";
    let second = "i32.const 3 i32.const 0 i32.const 0 i32.const 1 i32.const 0 i64.const 2 i64.const 0 i32.const 0 i32.const 8 call $open drop";
    let write = "i32.const 32 i32.const 16 i32.store i32.const 36 i32.const 2 i32.store i32.const 4 i32.const 32 i32.const 1 i32.const 40 call $write drop";
    for (dimension, files, descriptors, filesystem_bytes, body, completed) in [
        ("files", 2, 16, 1024, open.to_string(), 0),
        ("descriptors", 10, 5, 1024, format!("{open} {second}"), 1),
        ("filesystem-bytes", 10, 16, 1, format!("{open} {write}"), 1),
        ("transcript-bytes", 10, 16, 1024, open.to_string(), 0),
        (
            "unsupported",
            10,
            16,
            1024,
            format!("{open} i32.const 4 i32.const 2 call $flags drop"),
            2,
        ),
    ] {
        let wat = format!(
            r#"(module
          (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
          (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
          (import "wasi_snapshot_preview1" "fd_fdstat_set_flags" (func $flags (param i32 i32) (result i32)))
          (memory (export "memory") 1) (data (i32.const 0) "f") (data (i32.const 16) "ab")
          (func (export "_start") {body}))"#
        );
        let mut request = writable_namespace_request(files, descriptors, filesystem_bytes);
        if dimension == "transcript-bytes" {
            request["local_model"]["limits"]["transcript_bytes"] = json!(1);
        }
        let frames = run(&wat, request, |call| {
            panic!("namespace boundary escaped: {call}")
        });
        let terminal = frames.last().unwrap();
        if dimension == "unsupported" {
            assert_eq!(termination(&frames), "unsupported");
            assert_eq!(
                terminal["message"],
                "fd_fdstat_set_flags: descriptor flags outside volatile profile"
            );
        } else {
            assert_eq!(termination(&frames), "capacity", "{dimension}: {frames:?}");
            assert_eq!(terminal["message"], format!("WASI capacity: {dimension}"));
        }
        assert_eq!(terminal["local_model"]["calls"], completed);
    }
}

#[test]
fn namespace_oversized_guest_write_refuses_before_model_admission() {
    let mut request = writable_namespace_request(10, 16, 1024);
    request["memory_bytes"] = json!(65 * 65536);
    let frames = run(
        r#"(module
      (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
      (memory (export "memory") 65)
      (func (export "_start")
        i32.const 0 i32.const 16 i32.store i32.const 4 i32.const 4194305 i32.store
        i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop))"#,
        request,
        |call| panic!("oversized namespace write escaped: {call}"),
    );
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.last().unwrap()["message"], "ABI buffer capacity");
    assert_eq!(frames.last().unwrap()["local_model"]["calls"], 0);
    assert_eq!(frames.last().unwrap()["output_bytes"], 0);
}

#[test]
fn namespace_images_refuse_before_guest_start() {
    let image = captured_namespace_image(b"/capt/file");
    for (name, mut bytes, digest_mismatch) in [
        ("truncated", image[..image.len() - 1].to_vec(), false),
        ("files", image.clone(), false),
        ("bytes", image.clone(), false),
        ("bytes-ceiling", image.clone(), false),
        (
            "bytes-data",
            captured_namespace_image_with(b"/capt/file", b"qq"),
            false,
        ),
        ("descriptors-ceiling", image.clone(), false),
        ("duplicate", captured_namespace_image(b"/capt"), false),
        ("kind", image.clone(), false),
        ("digest", image.clone(), true),
    ] {
        match name {
            "files" => bytes[8..16].copy_from_slice(&2u64.to_le_bytes()),
            "bytes" => bytes[24..32].copy_from_slice(&0u64.to_le_bytes()),
            "bytes-ceiling" => bytes[24..32].copy_from_slice(&((1u64 << 30) + 1).to_le_bytes()),
            // A valid nonzero limit that the captured file data exceeds.
            "bytes-data" => bytes[24..32].copy_from_slice(&1u64.to_le_bytes()),
            "descriptors-ceiling" => {
                bytes[16..24].copy_from_slice(&((1u64 << 20) + 1).to_le_bytes())
            }
            "kind" => bytes[127] = 3,
            _ => (),
        }
        let mut request = namespace_request(&bytes);
        if digest_mismatch {
            request["namespace"]["sha256"] = json!("0".repeat(64));
        }
        let frames = run(
            r#"(module (memory (export "memory") 1) (func (export "_start") unreachable))"#,
            request,
            |call| panic!("{name}: {call}"),
        );
        assert_eq!(termination(&frames), "invalid", "{name}: {frames:?}");
        assert_eq!(frames.last().unwrap()["local_model"]["calls"], 0, "{name}");
        assert_eq!(
            frames.last().unwrap()["metrics"]["execute_nanos"],
            0,
            "{name}"
        );
        assert_eq!(frames.last().unwrap()["peak_memory_bytes"], 0, "{name}");
    }
}

#[test]
fn namespace_rofs_and_mixed_poll_stay_local_and_share_recorder() {
    let frames = run(
        r#"(module
      (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "poll_oneoff" (func $poll (param i32 i32 i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
      (memory (export "memory") 1) (data (i32.const 0) "file") (data (i32.const 16) "ok")
      (func (export "_start")
        i32.const 3 i32.const 0 i32.const 0 i32.const 4 i32.const 0 i64.const 64 i64.const 0 i32.const 0 i32.const 8 call $open
        i32.const 69 i32.ne if unreachable end
        i32.const 64 i64.const 11 i64.store
        i32.const 72 i32.const 1 i32.store8
        i32.const 80 i32.const 99 i32.store
        i32.const 112 i64.const 12 i64.store
        i32.const 128 i32.const 1 i32.store
        i32.const 136 i64.const 10 i64.store
        i32.const 64 i32.const 192 i32.const 2 i32.const 256 call $poll if unreachable end
        i32.const 256 i32.load i32.const 1 i32.ne if unreachable end
        i32.const 192 i64.load i64.const 11 i64.ne if unreachable end
        i32.const 200 i32.load16_u i32.const 8 i32.ne if unreachable end
        i32.const 32 i32.const 16 i32.store i32.const 36 i32.const 2 i32.store
        i32.const 1 i32.const 32 i32.const 1 i32.const 40 call $write if unreachable end))"#,
        namespace_request(&captured_namespace_image(b"/capt/file")),
        |call| panic!("namespace import escaped: {call}"),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    assert_eq!(frames.last().unwrap()["local_model"]["calls"], 3);
    let transcript: Vec<_> = frames
        .iter()
        .filter(|frame| frame["stream"] == "transcript")
        .flat_map(|frame| {
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
        })
        .collect();
    assert_eq!(&transcript[..8], b"GMWT\x01\0\0\0");
    let mut at = 8;
    for (id, tag) in [(1u64, 26u8), (2, 32), (3, 8)] {
        assert_eq!(transcript[at], tag);
        assert_eq!(&transcript[at + 4..at + 12], &id.to_le_bytes());
        at += 16 + u32::from_le_bytes(transcript[at + 12..at + 16].try_into().unwrap()) as usize;
    }
    assert_eq!(at, transcript.len());
    let stdout: Vec<_> = frames
        .iter()
        .filter(|frame| frame["stream"] == "stdout")
        .flat_map(|frame| {
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
        })
        .collect();
    assert_eq!(stdout, b"ok");
}

#[test]
fn namespace_trap_flushes_completed_output_and_transcript() {
    let frames = run(
        r#"(module
      (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
      (memory (export "memory") 1) (data (i32.const 16) "ok")
      (func (export "_start")
        i32.const 32 i32.const 16 i32.store i32.const 36 i32.const 2 i32.store
        i32.const 1 i32.const 32 i32.const 1 i32.const 40 call $write if unreachable end
        unreachable))"#,
        namespace_request(&captured_namespace_image(b"/capt/file")),
        |call| panic!("namespace write escaped: {call}"),
    );
    assert_eq!(termination(&frames), "trap");
    assert_eq!(frames.last().unwrap()["output_bytes"], 2);
    let stdout: Vec<_> = frames
        .iter()
        .filter(|frame| frame["stream"] == "stdout")
        .flat_map(|frame| {
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
        })
        .collect();
    assert_eq!(stdout, b"ok");
    let transcript: Vec<_> = frames
        .iter()
        .filter(|frame| frame["stream"] == "transcript")
        .flat_map(|frame| {
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
        })
        .collect();
    assert_eq!(&transcript[..8], b"GMWT\x01\0\0\0");
    assert_eq!(transcript[8], 8);
}
#[test]
fn local_random_and_strings_never_request_remote_callbacks() {
    let mut config = input_model();
    config["arguments"] = json!({"bytes":5,"count":2,"chunks":1});
    let frames = run(
        r#"(module
      (import "wasi_snapshot_preview1" "args_sizes_get" (func $as (param i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "args_get" (func $ag (param i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "environ_sizes_get" (func $es (param i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "environ_get" (func $eg (param i32 i32) (result i32)))
      (import "wasi_snapshot_preview1" "random_get" (func $r (param i32 i32) (result i32)))
      (memory (export "memory") 1)
      (func (export "_start")
        i32.const 0 i32.const 4 call $as drop
        i32.const 0 i32.load i32.const 2 i32.ne if unreachable end
        i32.const 4 i32.load i32.const 5 i32.ne if unreachable end
        i32.const 8 i32.const 16 call $ag drop
        i32.const 8 i32.load i32.const 16 i32.ne if unreachable end
        i32.const 12 i32.load i32.const 19 i32.ne if unreachable end
        i32.const 16 i32.load16_u i32.const 0x6261 i32.ne if unreachable end
        i32.const 18 i32.load8_u if unreachable end
        i32.const 19 i32.load16_u i32.const 0x63 i32.ne if unreachable end
        i32.const 24 i32.const 28 call $es drop
        i32.const 24 i32.load i32.const 1 i32.ne if unreachable end
        i32.const 28 i32.load i32.const 6 i32.ne if unreachable end
        i32.const 32 i32.const 40 call $eg drop
        i32.const 32 i32.load i32.const 40 i32.ne if unreachable end
        i32.const 40 i32.load i32.const 0x3d445750 i32.ne if unreachable end
        i32.const 64 i32.const 30 call $r drop
        i32.const 64 i32.const 6 call $r drop
        i32.const 64 i32.load i32.const 0x646a4748 i32.ne if unreachable end
        i32.const 68 i32.load16_u i32.const 0x20d7 i32.ne if unreachable end
        i32.const 65536 i32.const 0 call $r drop))"#,
        json!({"local_model":config,"fixture_model_inputs":[{"type":"model_input","stream":"arguments","sequence":0,"data_base64":"YWIAYwA="},{"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A"},{"type":"model_inputs_end"}]}),
        |call| panic!("local import escaped to Go: {call}"),
    );
    assert_eq!(termination(&frames), "exit");
    assert_eq!(frames.last().unwrap()["local_model"]["calls"], 7);
    let mut bytes = Vec::new();
    for frame in &frames {
        if frame["stream"] == "transcript" {
            bytes.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap())
                    .unwrap(),
            );
        }
    }
    let mut expected = b"GMWT\x01\0\0\0".to_vec();
    for (index, tag) in [3u8, 4, 5, 6].into_iter().enumerate() {
        expected.extend([tag, 0, 0, 0]);
        expected.extend(((index + 1) as u64).to_le_bytes());
        expected.extend([0; 4]);
    }
    for (id, length, offset) in [(5u64, 30u32, 0u64), (6, 6, 30), (7, 0, 36)] {
        expected.extend([7, 0, 0, 0]);
        expected.extend(id.to_le_bytes());
        expected.extend([12, 0, 0, 0]);
        expected.extend(length.to_le_bytes());
        expected.extend(offset.to_le_bytes());
    }
    assert_eq!(bytes, expected);
}
#[test]
fn malformed_local_inputs_refuse_before_started_and_get_requires_sizes() {
    let config = input_model();
    for field in [
        "entropy_key",
        "working_directory_sha256",
        "arguments",
        "environment",
    ] {
        let mut missing = config.clone();
        missing.as_object_mut().unwrap().remove(field);
        let frames = run("(module)", json!({"local_model":missing}), |call| {
            panic!("{call}")
        });
        assert_eq!(frames.len(), 1);
        assert_eq!(termination(&frames), "invalid");
    }
    for input in [
        json!({"type":"model_input","stream":"environment","sequence":1,"data_base64":"UFdEPS8A"}),
        json!({"type":"model_input","stream":"arguments","sequence":0,"data_base64":"UFdEPS8A"}),
        json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8="}),
        json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A","unknown":true}),
    ] {
        let frames = run(
            "(module (memory (export \"memory\") 1))",
            json!({"local_model":config,"fixture_model_inputs":[input]}),
            |call| panic!("{call}"),
        );
        assert_eq!(frames.len(), 1);
        assert_eq!(termination(&frames), "invalid");
    }
    for name in ["args_get", "environ_get"] {
        let frames = run(
            &format!(
                "(module (import \"wasi_snapshot_preview1\" \"{name}\" (func $get (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const -1 i32.const -1 call $get drop))"
            ),
            json!({"local_model":config}),
            |call| panic!("{call}"),
        );
        assert_eq!(termination(&frames), "trap");
        assert_eq!(frames.last().unwrap()["local_model"]["calls"], 0);
        assert!(
            frames.last().unwrap()["message"]
                .as_str()
                .unwrap()
                .contains("string sizes must precede get")
        );
    }
    for (name, arguments, termination_kind) in [
        ("args_sizes_get", "i32.const -1 i32.const -1", "trap"),
        ("environ_sizes_get", "i32.const -1 i32.const -1", "trap"),
        ("random_get", "i32.const -1 i32.const 1", "trap"),
        ("random_get", "i32.const -1 i32.const 4194305", "capacity"),
    ] {
        let frames = run(
            &format!(
                "(module (import \"wasi_snapshot_preview1\" \"{name}\" (func $call (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") {arguments} call $call drop))"
            ),
            json!({"local_model":config}),
            |call| panic!("{call}"),
        );
        assert_eq!(termination(&frames), termination_kind);
        assert_eq!(frames.last().unwrap()["local_model"]["calls"], 0);
        assert_eq!(frames.last().unwrap()["local_model"]["transcript_bytes"], 0);
    }
}
#[test]
fn truncated_local_input_carriage_is_invalid_before_module_loading() {
    for suffix in [
        "",
        "{\"type\":\"model_input\",\"stream\":\"environment\",\"sequence\":0,\"data_base64\":\"UFdEPS8A\"}\n",
    ] {
        let request = json!({"schema":"gomad3.wasm-host/v1","type":"execute","module_path":"/not-loaded-for-input-refusal.wasm","module_sha256":"0".repeat(64),"memory_bytes":65536,"fuel":1,"output_bytes":1,"local_model":input_model()});
        let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .unwrap();
        {
            let mut input = child.stdin.take().unwrap();
            writeln!(input, "{request}").unwrap();
            write!(input, "{suffix}").unwrap();
        }
        let output = child.wait_with_output().unwrap();
        assert!(output.status.success());
        let frames: Vec<Value> = String::from_utf8(output.stdout)
            .unwrap()
            .lines()
            .map(|line| serde_json::from_str(line).unwrap())
            .collect();
        assert_eq!(frames.len(), 1);
        assert_eq!(termination(&frames), "invalid");
        assert!(
            !frames[0]["message"]
                .as_str()
                .unwrap()
                .contains("not-loaded-for-input-refusal")
        );
    }
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
    let model = json!({"seed":7,"mode":1,"limit":4096,"diagnostics":true,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}});
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
    let config = json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}});
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{}", json!({"schema":"gomad3.wasm-host/v1","type":"execute","module_path":path,
        "module_sha256":hash,"memory_bytes":131072,"fuel":1000000,"output_bytes":1024,"local_model":config})).unwrap();
    writeln!(
        input,
        "{}",
        json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A"})
    )
    .unwrap();
    writeln!(input, "{}", json!({"type":"model_inputs_end"})).unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}", json!({"type":"authorize","module_sha256":hash,"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "call" {
            let transcript =
                b"\x0d\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x03\x00\x00\x00";
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
    let clock = b"\x47\x4d\x57\x54\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00";
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
            expected.extend(
                b"\x0d\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x03\x00\x00\x00",
            );
        }
        assert_eq!(transcript, expected);
        assert_eq!(
            frames.last().unwrap()["local_model"]["now"],
            if want == "trap" { 1 } else { 4 }
        );
    }
}

#[cfg(feature = "diagnostic-profile")]
#[test]
fn local_yield_and_poll_encode_only_terminal_evidence_frames() {
    let guest = r#"(module
        (import "wasi_snapshot_preview1" "sched_yield" (func $yield (result i32)))
        (import "wasi_snapshot_preview1" "poll_oneoff" (func $poll (param i32 i32 i32 i32) (result i32)))
        (memory (export "memory") 1)
        (data (i32.const 16) "\01\00\00\00")
        (func (export "_start")
            call $yield drop
            i32.const 0 i32.const 64 i32.const 1 i32.const 128 call $poll drop
            i32.const 128 i32.load i32.const 1 i32.ne if unreachable end))"#;
    let (frames, stderr) = run_captured(
        guest,
        json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":8,"transcript_bytes":65536,"pending_events":16}}}),
        deny,
        AuthorizationMismatch::None,
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    let profile: Value = serde_json::from_str(
        stderr
            .trim()
            .strip_prefix("gomad-wasm-profile-v1 ")
            .unwrap(),
    )
    .unwrap();
    // Started, final evidence and result are the only JSON encoding spans.
    assert_eq!(profile["nested"]["encoding"]["count"], 3);
    assert_eq!(profile["abi"]["remote_callback"]["count"], 0);
}

#[test]
fn local_clock_abi_preserves_literal_transcript_and_pointer_validation_priority() {
    let config = json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}});
    let guest = "(module (import \"wasi_snapshot_preview1\" \"clock_time_get\" (func $clock (param i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i64.const 0 i32.const 8 call $clock drop i32.const 8 i64.load i64.const 1001 i64.ne if unreachable end i32.const 1 i64.const -1 i32.const 16 call $clock drop i32.const 16 i64.load i64.const 4 i64.ne if unreachable end))";
    let frames = run(guest, config.clone(), deny);
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    let expected = b"\x47\x4d\x57\x54\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x21\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\xff\xff\xff\xff\xff\xff\xff\xff\x04\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00";
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
fn clock_runs_close_before_remote_splices_interrupts_and_terminal() {
    let guest = r#"(module
        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
        (import "wasi_snapshot_preview1" "fd_sync" (func $sync (param i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start")
            i32.const 1 i64.const 7 i32.const 8 call $clock drop
            i32.const 1 i64.const 7 i32.const 8 call $clock drop
            i32.const 1 i64.const 7 i32.const 8 call $clock drop
            i32.const 3 call $sync drop
            i32.const 1 i64.const 7 i32.const 16 call $clock drop
            i32.const 16 i64.load i64.const 10 i64.ne if unreachable end))"#;
    let first = b"GMWT\x01\0\0\0\x21\0\0\0\x01\0\0\0\0\0\0\0\x1c\0\0\0\x01\0\0\0\x07\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0\x03\0\0\0\0\0\0\0";
    let remote = b"\x0d\0\0\0\x04\0\0\0\0\0\0\0\x04\0\0\0\x03\0\0\0";
    let last = b"\x21\0\0\0\x05\0\0\0\0\0\0\0\x1c\0\0\0\x01\0\0\0\x07\0\0\0\0\0\0\0\x0a\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0";
    for interrupted in [false, true] {
        let frames = run(
            guest,
            json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}}),
            |call| {
                assert_eq!(call["id"], 4);
                assert_eq!(
                    call["local_state"],
                    json!({"now":10,"advances":0,"transcript_bytes":52})
                );
                if interrupted {
                    json!({"type":"interrupt","termination":"storage_crash","message":"modeled storage crash"})
                } else {
                    json!({"type":"reply","id":4,"errno":0,"output":{},"local_state":{"now":10,"advances":0},"transcript_base64":STANDARD.encode(remote)})
                }
            },
        );
        assert_eq!(
            termination(&frames),
            if interrupted { "storage_crash" } else { "exit" }
        );
        let mut expected = first.to_vec();
        if !interrupted {
            expected.extend(remote);
            expected.extend(last);
        }
        let mut actual = Vec::new();
        for frame in frames
            .iter()
            .filter(|frame| frame["type"] == "evidence" && frame["stream"] == "transcript")
        {
            actual.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap())
                    .unwrap(),
            );
        }
        assert_eq!(actual, expected);
        assert_eq!(
            frames.last().unwrap()["local_model"]["calls"],
            if interrupted { 3 } else { 5 }
        );
        assert_eq!(
            frames.last().unwrap()["local_model"]["transcript_sha256"],
            format!("{:x}", Sha256::digest(expected))
        );
    }
}

#[test]
fn forwards_bounded_v2_stream_import_to_remote_model() {
    let guest = "(module (import \"gomad_wasm_v2\" \"net_stream\" (func $net (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 64 call $net drop))";
    for local in [false, true] {
        let config = if local {
            json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,
        "entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}})
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
            "entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}})
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
        assert!(stderr.is_empty());
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
                .is_some_and(|extension| extension == "gomad-cache-v3")
        })
        .unwrap();
    let envelope = std::fs::read(entry).unwrap();
    let manifest_len = u32::from_le_bytes(envelope[8..12].try_into().unwrap()) as usize;
    let manifest: Value = serde_json::from_slice(&envelope[24..24 + manifest_len]).unwrap();
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
fn clock_import_during_wasm_start_uses_instance_memory() {
    let frames = run(
        r#"(module
            (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
            (memory (export "memory") 1)
            (func $init i32.const 1 i64.const 0 i32.const 8 call $clock drop)
            (start $init)
            (func (export "_start") i32.const 8 i64.load i64.const 1 i64.ne if unreachable end))"#,
        json!({"local_model":{"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":3,"now":1,"limits":{"calls":8,"transcript_bytes":65536,"pending_events":16}}}),
        deny,
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    assert_eq!(frames.last().unwrap()["local_model"]["calls"], 1);
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
        ("gomad_wasm_v1", "net", "(param i32 i32) (result i32)"),
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

const STORAGE_GUEST: &str = r#"(module
  (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "fd_sync" (func $sync (param i32) (result i32)))
  (memory (export "memory") 1) (data (i32.const 0) "db") (data (i32.const 16) "ok")
  (func (export "_start")
    i32.const 3 i32.const 0 i32.const 0 i32.const 2 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 8 call $open
    if unreachable end
    i32.const 32 i32.const 16 i32.store i32.const 36 i32.const 2 i32.store
    i32.const 8 i32.load i32.const 32 i32.const 1 i32.const 40 call $write if unreachable end
    i32.const 8 i32.load call $sync if unreachable end))"#;

fn storage_model(faults: &str) -> String {
    format!(
        r#"{{"volumes":[{{"ID":"data","Path":"/data","CapacityBytes":512}}],"limits":{{"PendingOperations":100,"Transitions":100}}{faults}}}"#
    )
}

fn storage_request(model: &str, image: Option<&[u8]>, digest: Option<&str>) -> Value {
    let mut request = writable_namespace_request_at(b"/data", 10, 16, 1024);
    request["storage"] = json!({
        "model": serde_json::from_str::<Value>(model).unwrap(),
        "model_identity": format!("sha256:{:x}", Sha256::digest(model.as_bytes())),
        "crash_limits": {"States":8,"Operations":8,"Depth":8,"Bytes":4096,"Work":4096},
        "return_image": true,
    });
    if let Some(image) = image {
        request["storage"]["image"] = json!({"chunks":image.len().div_ceil(65536),"bytes":image.len(),
            "sha256":digest.map(str::to_owned).unwrap_or_else(|| format!("{:x}", Sha256::digest(image)))});
        let frames = request["fixture_namespace_inputs"].as_array_mut().unwrap();
        for (sequence, chunk) in image.chunks(65536).enumerate() {
            frames.push(json!({"type":"storage_image_input","sequence":sequence,"data_base64":STANDARD.encode(chunk)}));
        }
        frames.push(json!({"type":"storage_image_inputs_end"}));
    }
    request
}

fn stream(frames: &[Value], name: &str) -> Vec<u8> {
    frames
        .iter()
        .filter(|frame| frame["stream"] == name)
        .flat_map(|frame| {
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
        })
        .collect()
}

#[test]
fn storage_runs_answer_volume_operations_locally_and_return_evidence() {
    let model = storage_model("");
    let frames = run(STORAGE_GUEST, storage_request(&model, None, None), |call| {
        panic!("storage import escaped: {call}")
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
    let log = String::from_utf8(stream(&frames, "storage")).unwrap();
    assert_eq!(
        log,
        concat!(
            r#"{"sequence":0,"resource":"/data/db","operation":"open","occurrence":1,"action":"complete"}"#,
            "\n",
            r#"{"sequence":1,"resource":"/data/db","operation":"write","occurrence":1,"action":"complete","bytes":2}"#,
            "\n",
            r#"{"sequence":2,"resource":"/data/db","operation":"sync","occurrence":1,"action":"complete"}"#,
            "\n",
        )
    );
    let state = stream(&frames, "storage_state");
    let body = state.len() - 40;
    assert_eq!(&state[..8], b"GMSTORE\x01");
    assert_eq!(
        &state[body + 8..],
        Sha256::digest(&state[..body]).as_slice()
    );
    let image = stream(&frames, "storage_image");
    assert!(image.starts_with(b"GMVOLIMG"));
    let identity = format!("{:x}", Sha256::digest(&image));
    assert!(
        state
            .windows(64)
            .any(|window| window == identity.as_bytes())
    );

    // Recovery from the returned image starts with the durable file.
    let recovered = run(
        r#"(module (memory (export "memory") 1) (func (export "_start")))"#,
        storage_request(&model, Some(&image), None),
        |call| panic!("{call}"),
    );
    assert_eq!(termination(&recovered), "exit", "{recovered:?}");
    let state = stream(&recovered, "storage_state");
    let text = String::from_utf8_lossy(&state);
    assert!(text.contains(r#""Path":"/db""#), "{text}");
}

#[test]
fn storage_crash_before_is_helper_originated_with_evidence() {
    let model = storage_model(
        r#","faults":[{"name":"stop-sync","resource":"/data/db","operation":"sync","occurrence":1,"action":"crash-before"}]"#,
    );
    let frames = run(STORAGE_GUEST, storage_request(&model, None, None), |call| {
        panic!("storage import escaped: {call}")
    });
    let terminal = frames.last().unwrap();
    assert_eq!(termination(&frames), "storage_crash", "{frames:?}");
    assert_eq!(terminal["message"], "modeled storage crash: stop-sync");
    assert_eq!(terminal["local_model"]["calls"], 2);
    let log = String::from_utf8(stream(&frames, "storage")).unwrap();
    assert!(
        log.ends_with("\"operation\":\"sync\",\"occurrence\":1,\"fault\":\"stop-sync\",\"action\":\"crash-before\"}\n"),
        "{log}"
    );
    assert!(!stream(&frames, "storage_state").is_empty());
    assert!(!stream(&frames, "storage_image").is_empty());
}

#[test]
fn invalid_storage_configuration_and_images_refuse_before_guest_start() {
    let model = storage_model("");
    let unmounted = model.replace("/data", "/other");
    let after = storage_model(
        r#","faults":[{"name":"a","resource":"/data/db","operation":"write","occurrence":1,"action":"crash-after"}]"#,
    );
    let image = b"GMVOLIMG\x01\0\0\0\x03\0\0\0".to_vec();
    // Identity mismatches refuse at request parse (storage_transport tests).
    for (name, request) in [
        ("unmounted", storage_request(&unmounted, None, None)),
        ("crash-after", storage_request(&after, None, None)),
        (
            "truncated-image",
            storage_request(&model, Some(&image), None),
        ),
        (
            "digest",
            storage_request(&model, Some(&image), Some(&"0".repeat(64))),
        ),
    ] {
        let frames = run(
            r#"(module (memory (export "memory") 1) (func (export "_start") unreachable))"#,
            request,
            |call| panic!("{name}: {call}"),
        );
        assert_eq!(termination(&frames), "invalid", "{name}: {frames:?}");
        assert_eq!(frames.last().unwrap()["peak_memory_bytes"], 0, "{name}");
        assert!(stream(&frames, "storage_state").is_empty(), "{name}");
    }
}

const SPLIT_GUEST: &str = r#"(module
  (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
  (memory (export "memory") 1) (data (i32.const 0) "db") (data (i32.const 16) "ok")
  (func (export "_start")
    i32.const 32 i32.const 16 i32.store i32.const 36 i32.const 2 i32.store
    i32.const 1 i32.const 32 i32.const 1 i32.const 40 call $write if unreachable end
    i32.const 3 i32.const 0 i32.const 0 i32.const 2 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 8 call $open
    if unreachable end))"#;

fn split_request() -> Value {
    let mut request = storage_request(&storage_model(""), None, None);
    request["local_model"]["split"] = json!(true);
    request
}

/// Split ownership: guest fd 1 writes are revocable remote calls; the next
/// local call first confirms them with `sync` (fn-176.6).
#[test]
fn split_ownership_syncs_after_revocable_replies_before_local_calls() {
    for interrupt in [false, true] {
        let mut syncs = Vec::new();
        let frames = run(SPLIT_GUEST, split_request(), |frame| {
            if frame["type"] == "sync" {
                syncs.push(frame["id"].as_u64().unwrap());
                return if interrupt && syncs.len() == 1 {
                    json!({"type":"interrupt","termination":"crash","message":"WASI crash: broker incarnation revoked"})
                } else {
                    json!({"type":"continue","now":7})
                };
            }
            assert_eq!(
                frame["op"], "fd_write",
                "only guest output is remote: {frame}"
            );
            assert_eq!(frame["input"]["fd"], 1);
            json!({"type":"reply","id":frame["id"],"errno":0,"output":{"written":2},
                "local_state":{"now":5,"advances":0},"transcript_base64":"","revocable":true})
        });
        // The first local call is the open after the remote write.
        assert_eq!(syncs, vec![1], "{frames:?}");
        if interrupt {
            assert_eq!(termination(&frames), "crash", "{frames:?}");
            assert!(
                stream(&frames, "storage").is_empty(),
                "local open ran after revocation"
            );
            assert!(!stream(&frames, "storage_state").is_empty());
        } else {
            assert_eq!(termination(&frames), "exit", "{frames:?}");
            assert_eq!(frames.last().unwrap()["local_model"]["now"], 7);
            assert!(
                String::from_utf8(stream(&frames, "storage"))
                    .unwrap()
                    .contains(r#""operation":"open""#)
            );
        }
    }
}

/// A remote call Go completed before the broker revoked the incarnation is
/// counted by the helper with its clock state (and record, when it has one),
/// so split terminal accounting matches the Go owner's (fn-176.6).
#[test]
fn split_interrupt_for_a_completed_call_counts_it_and_adopts_its_clock() {
    for record in [false, true] {
        let frames = run(SPLIT_GUEST, split_request(), |frame| {
            assert_eq!(frame["type"], "call", "{frame}");
            let mut interrupt = json!({"type":"interrupt","termination":"crash",
                "message":"WASI crash: broker output trigger","completed":{"now":9,"advances":1}});
            if record {
                interrupt["transcript_base64"] = json!(STANDARD.encode(b"record"));
            }
            interrupt
        });
        assert_eq!(termination(&frames), "crash", "{frames:?}");
        let model = &frames.last().unwrap()["local_model"];
        assert_eq!(model["calls"], 1, "{frames:?}");
        assert_eq!(model["now"], 9);
        assert_eq!(model["advances"], 1);
        assert!(!stream(&frames, "storage_state").is_empty());
        assert!(!stream(&frames, "storage_image").is_empty());
    }
    // A record without a completed call is refused.
    let frames = run(
        SPLIT_GUEST,
        split_request(),
        |_| json!({"type":"interrupt","termination":"crash","message":"m","transcript_base64":"eA=="}),
    );
    assert_eq!(termination(&frames), "invalid", "{frames:?}");
}
