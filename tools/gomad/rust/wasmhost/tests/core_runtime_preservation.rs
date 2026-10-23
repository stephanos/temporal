#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::linked;
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::time::Duration;

fn request(body: &str) -> linked::Request {
    let module = format!(
        r#"(module
        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "decision" (func $decision (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start") {body}))"#
    )
    .into_bytes();
    linked::Request {
        module_sha256: format!("{:x}", Sha256::digest(&module)),
        module,
        limits: linked::Limits {
            memory_bytes: 65536,
            fuel: 10000,
            output_bytes: 4096,
            evidence_bytes: 1 << 20,
            wall_timeout: Duration::from_secs(5),
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(json!({
                "seed":7,"mode":1,"limit":8192,"diagnostics":false,
                "entropy_key":vec![0_u8;32],
                "working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
                "arguments":{"bytes":0,"count":0,"chunks":0},
                "environment":{"bytes":0,"count":0,"chunks":0},
                "epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}
            }))
            .unwrap(),
            arguments: Vec::new(),
            environment: vec!["PWD=/".into()],
            replay_trace: None,
        }),
        namespace: None,
        storage: None,
    }
}

const CONFIGURE_AND_DECIDE: &str = r#"
    (drop (call $config (i32.const 0) (i32.const 16)))
    (i32.store (i32.const 32) (i32.const 1))
    (i32.store (i32.const 36) (i32.const 1))
    (i64.store (i32.const 40) (i64.const 10))
    (i32.store (i32.const 52) (i32.const 2))
    (i32.store8 (i32.const 64) (i32.const 1))
    (i32.store8 (i32.const 96) (i32.const 2))
    (drop (call $decision (i32.const 32) (i32.const 96)))
"#;

#[test]
fn ordinary_linked_exit_requires_the_configured_runtime_finish() {
    for (suffix, expected, code) in [
        ("", "invalid", None),
        (
            "(i32.store (i32.const 160) (i32.const 1)) (drop (call $finish (i32.const 160) (i32.const 16)))",
            "exit",
            Some(0),
        ),
        ("unreachable", "trap", None),
        ("(loop $spin (br $spin))", "capacity", None),
    ] {
        let observed =
            linked::execute(request(&format!("{CONFIGURE_AND_DECIDE}{suffix}"))).unwrap();
        assert_eq!(observed.outcome.termination, expected, "suffix {suffix:?}");
        assert_eq!(observed.outcome.exit_code, code);
        let metadata = observed.outcome.local_model.as_ref().unwrap();
        assert_eq!(metadata["configured"], true);
        assert_eq!(metadata["records"], 1);
        assert_eq!(metadata["finished"], expected == "exit");
        assert_eq!(observed.evidence["choice"].len(), 96);
        if expected == "invalid" {
            assert!(
                observed
                    .outcome
                    .message
                    .contains("runtime trace unfinished")
            );
        }
    }
    let ordinary = linked::execute(request("")).unwrap();
    assert_eq!(ordinary.outcome.termination, "exit");
    assert_eq!(ordinary.outcome.exit_code, Some(0));
    assert_eq!(
        ordinary.outcome.local_model.as_ref().unwrap()["configured"],
        false
    );
}

#[test]
fn native_outcomes_keep_real_exit_trap_fuel_and_output_claims_within_admitted_limits() {
    for (body, expected, exit_code) in [
        ("(call $exit (i32.const 7))", "exit", Some(7)),
        ("unreachable", "trap", None),
        ("(loop $spin (br $spin))", "capacity", None),
        (
            "(i32.store (i32.const 0) (i32.const 64)) (i32.store (i32.const 4) (i32.const 5)) (drop (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 16)))",
            "capacity",
            None,
        ),
    ] {
        let mut input = request("");
        input.module = format!(r#"(module
            (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
            (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
            (memory (export "memory") 1)
            (data (i32.const 64) "hello")
            (func (export "_start") {body}))"#).into_bytes();
        input.module_sha256 = format!("{:x}", Sha256::digest(&input.module));
        input.limits.output_bytes = 4;
        let observed = linked::execute(input).unwrap();
        assert_eq!(observed.outcome.termination, expected, "{body}");
        assert_eq!(observed.outcome.exit_code, exit_code, "{body}");
        assert!(observed.outcome.fuel_remaining <= 10000);
        assert!(observed.outcome.peak_memory_bytes <= 65536);
        let retained: usize = ["stdout", "stderr"]
            .into_iter()
            .map(|stream| observed.evidence.get(stream).map_or(0, Vec::len))
            .sum();
        assert!(retained <= 4);
        assert!(observed.authorization_to_terminal_nanos.is_some());
        if body.contains("$write") {
            assert_eq!(observed.outcome.output_bytes, 5);
            assert_eq!(retained, 0);
        } else {
            assert_eq!(observed.outcome.output_bytes, 0);
        }
    }
}
