#![cfg(not(feature = "conformance"))]
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::io::{BufRead, BufReader, Write};
use std::process::{Command, Stdio};

fn marker(kind: &str, replay: &Value) -> Vec<u8> {
    let mut bytes = serde_json::to_vec(&json!({"type":kind,"schema":replay["schema"],"trace_bytes":replay["trace_bytes"],"trace_frames":replay["trace_frames"]})).unwrap();
    bytes.push(b'\n');
    bytes
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The subprocess table preserves setup, wire mutation and pre-instantiation assertions together."
)]
fn replay_preflight_refuses_before_instantiation() {
    let wat = b"(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))";
    let path =
        std::env::temp_dir().join(format!("gomad-replay-preflight-{}.wat", std::process::id()));
    std::fs::write(&path, wat).unwrap();
    for name in [
        "valid-empty",
        "valid-trace",
        "valid-prefix",
        "schema",
        "record-trace",
        "trace-sha256",
        "trace-decode",
        "tape-sha256",
        "decisions",
        "identity",
        "oversized",
        "overflow",
        "frame-count",
        "zero-frame",
        "oversized-frame",
        "truncated-length",
        "truncated-payload",
        "extra-frame",
        "missing-end",
        "wrong-end",
        "wrong-count",
        "exact-control",
    ] {
        let fixture_name = match name {
            "valid-empty" => "empty",
            "valid-prefix" => "prefix-rank",
            _ => "select-readiness-origin",
        };
        let fixture: Value = serde_json::from_slice(
            &std::fs::read(
                std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                    .join(format!("tests/testdata/replay_plan/{fixture_name}.json")),
            )
            .unwrap(),
        )
        .unwrap();
        let mut trace = STANDARD.decode(fixture["trace"].as_str().unwrap()).unwrap();
        let mut replay = json!({"schema":"gomad.replay-trace/v1","execution_identity":fixture["identity"],"trace_sha256":format!("{:x}",Sha256::digest(&trace)),"tape_sha256":fixture["expected"]["sha256"],"decisions":fixture["expected"]["decisions"].as_array().unwrap().len(),"trace_bytes":trace.len(),"trace_frames":trace.len().div_ceil(65536)});
        if name == "valid-prefix" {
            replay["prefix_control"] = fixture["prefix_control"].clone();
        }
        let expected = match name {
            "valid-empty" | "valid-trace" | "valid-prefix" => None,
            "schema" => {
                replay["schema"] = "gomad.replay-trace/v99".into();
                Some("schema")
            }
            "trace-sha256" => {
                trace[0] ^= 1;
                Some("trace_sha256")
            }
            "trace-decode" => {
                trace[8] = 255;
                replay["trace_sha256"] = format!("{:x}", Sha256::digest(&trace)).into();
                Some("trace_decode")
            }
            "tape-sha256" => {
                replay["tape_sha256"] = "0".repeat(64).into();
                Some("tape_sha256")
            }
            "decisions" => {
                replay["decisions"] = u64::MAX.into();
                Some("decisions")
            }
            "identity" => {
                replay["execution_identity"]["TargetSHA256"] = json!(vec![0u8; 32]);
                Some("execution_identity")
            }
            "oversized" => {
                replay["trace_bytes"] = 8193.into();
                Some("framing")
            }
            "overflow" => {
                replay["trace_bytes"] = u64::MAX.into();
                replay["trace_frames"] = u64::MAX.into();
                Some("framing")
            }
            "frame-count" => {
                replay["trace_frames"] = 0.into();
                Some("framing")
            }
            "exact-control" => {
                replay["prefix_control"] = json!({"cutoff":0});
                Some("tape_sha256")
            }
            _ => Some("framing"),
        };
        let mut body = marker("replay_trace", &replay);
        if !trace.is_empty() {
            let length = match name {
                "zero-frame" => 0,
                "oversized-frame" => 65537,
                _ => u32::try_from(trace.len()).unwrap(),
            };
            let encoded = length.to_be_bytes();
            if name == "truncated-length" {
                body.extend(&encoded[..2]);
            } else {
                body.extend(encoded);
                if name != "truncated-payload" {
                    body.extend(&trace);
                }
            }
        }
        if !matches!(
            name,
            "truncated-length" | "truncated-payload" | "missing-end"
        ) {
            if name == "extra-frame" {
                body.extend([0, 0, 0, 1, 255]);
            }
            if name == "wrong-count" {
                replay["trace_bytes"] = 0.into();
            }
            body.extend(marker(
                if name == "wrong-end" {
                    "other"
                } else {
                    "replay_trace_end"
                },
                &replay,
            ));
            if name == "wrong-count" {
                replay["trace_bytes"] = trace.len().into();
            }
        }
        let mode = if name == "record-trace" {
            1
        } else if name == "valid-prefix" {
            3
        } else {
            2
        };
        let request = json!({"schema":"gomad.wasm-host/v1","type":"execute","module_path":path,"module_sha256":format!("{:x}",Sha256::digest(wat)),"memory_bytes":131_072,"fuel":1000,"output_bytes":1024,
            "local_model":{"seed":7,"mode":mode,"limit":8192,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":0,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":65536,"pending_events":16},"replay":replay}});
        let mut child = Command::new(env!("CARGO_BIN_EXE_gomad-wasmhost"))
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .unwrap();
        let mut input = child.stdin.take();
        let mut bootstrap = Vec::new();
        writeln!(bootstrap, "{request}").unwrap();
        writeln!(bootstrap,"{}",json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":"UFdEPS8A"})).unwrap();
        writeln!(bootstrap, "{}", json!({"type":"model_inputs_end"})).unwrap();
        input.as_mut().unwrap().write_all(&bootstrap).unwrap();
        let mut terminal = None;
        for line in BufReader::new(child.stdout.take().unwrap()).lines() {
            let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
            if frame["type"] == "started" {
                writeln!(input.as_mut().unwrap(),"{}",json!({"type":"authorize","module_sha256":frame["module_sha256"],"configuration":frame["engine"]["configuration"]})).unwrap();
                input.as_mut().unwrap().write_all(&body).unwrap();
                drop(input.take());
            }
            if frame["type"] == "result" {
                terminal = Some(frame);
            }
        }
        drop(input);
        let output = child.wait_with_output().unwrap();
        assert!(
            output.status.success(),
            "{name}: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        let terminal = terminal.unwrap();
        if let Some(check) = expected {
            assert_eq!(terminal["termination"], "invalid", "{name}: {terminal}");
            assert_eq!(terminal["error"], "replay_preflight", "{name}: {terminal}");
            assert_eq!(terminal["check"], check, "{name}: {terminal}");
            assert_eq!(
                terminal["metrics"]["instantiate_nanos"]
                    .as_u64()
                    .unwrap_or(0),
                0,
                "{name}"
            );
            assert_eq!(terminal["peak_memory_bytes"], 0, "{name}");
        } else {
            assert_eq!(terminal["termination"], "trap", "{name}: {terminal}");
            assert!(
                terminal["metrics"]["instantiate_nanos"].as_u64().unwrap() > 0,
                "{name}"
            );
        }
    }
    std::fs::remove_file(path).unwrap();
}
