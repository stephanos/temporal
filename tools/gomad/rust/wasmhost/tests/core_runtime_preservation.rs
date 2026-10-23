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
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(),
            stdin: Vec::new(),
            limits: linked::NamespaceLimits {
                files: 16,
                descriptors: 16,
                filesystem_bytes: 4096,
                output_bytes: 4096,
            },
            epoch_nanos: 0,
        }),
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

#[test]
fn native_preparation_refuses_invalid_inputs_before_guest_instantiation() {
    for case in [
        "module-pin",
        "zero-fuel",
        "zero-memory",
        "zero-evidence",
        "zero-wall",
    ] {
        let mut input = request("unreachable");
        match case {
            "module-pin" => input.module_sha256 = "0".repeat(64),
            "zero-fuel" => input.limits.fuel = 0,
            "zero-memory" => input.limits.memory_bytes = 0,
            "zero-evidence" => input.limits.evidence_bytes = 0,
            "zero-wall" => input.limits.wall_timeout = Duration::ZERO,
            _ => unreachable!(),
        }
        assert!(linked::prepare(input).is_err(), "{case}");
    }
    let prepared = linked::prepare(request("unreachable")).unwrap();
    let observed = linked::execute_prepared(prepared).unwrap();
    assert_eq!(observed.outcome.termination, "trap");
    assert_eq!(observed.outcome.exit_code, None);
    assert_eq!(
        linked::CONFIGURATION,
        "cranelift-fuel-nan-canonical-nounwind-epoch-v1"
    );
}

#[test]
fn native_model_admission_rejects_unknown_fields_at_each_typed_level() {
    let valid = json!({
        "seed":7,"mode":1,"limit":8192,"diagnostics":false,
        "entropy_key":vec![0_u8;32],
        "working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
        "arguments":{"bytes":0,"count":0,"chunks":0},
        "environment":{"bytes":0,"count":0,"chunks":0},
        "epoch_nanos":0,"read_step_nanos":0,"now":1,
        "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}
    });
    assert!(serde_json::from_value::<linked::ModelConfig>(valid.clone()).is_ok());
    for level in [None, Some("arguments"), Some("environment"), Some("limits")] {
        let mut malformed = valid.clone();
        let object = match level {
            Some(level) => malformed[level].as_object_mut().unwrap(),
            None => malformed.as_object_mut().unwrap(),
        };
        object.insert("future".into(), true.into());
        let error = serde_json::from_value::<linked::ModelConfig>(malformed)
            .err()
            .unwrap();
        assert!(
            error.to_string().contains("unknown field"),
            "{level:?}: {error}"
        );
    }
}

#[test]
fn rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site() {
    let cases = [
        (
            "path_readlink",
            "i32 i32 i32 i32 i32 i32",
            "(i32.const 3) (i32.const 64) (i32.const 1) (i32.const 128) (i32.const 1) (i32.const 132)",
        ),
        (
            "path_symlink",
            "i32 i32 i32 i32 i32",
            "(i32.const 64) (i32.const 1) (i32.const 3) (i32.const 65) (i32.const 1)",
        ),
        (
            "sock_accept",
            "i32 i32 i32",
            "(i32.const 3) (i32.const 0) (i32.const 0)",
        ),
        ("sock_shutdown", "i32 i32", "(i32.const 3) (i32.const 0)"),
    ];
    let inventory: serde_json::Value = serde_json::from_str(include_str!(
        "../../../internal/backends/wasm/determinism/inventory.json"
    ))
    .unwrap();
    let rejected: std::collections::BTreeSet<_> = inventory["entries"]
        .as_array()
        .unwrap()
        .iter()
        .filter(|entry| entry["layer"] == "import" && entry["disposition"] == "rejected")
        .filter_map(|entry| {
            entry["site"]
                .as_str()
                .unwrap()
                .strip_prefix("wasi_snapshot_preview1::")
        })
        .collect();
    assert_eq!(rejected, cases.iter().map(|(name, _, _)| *name).collect());
    for (name, parameters, arguments) in cases {
        let mut input = request("");
        input.module = format!(r#"(module
            (import "wasi_snapshot_preview1" "{name}" (func $rejected (param {parameters}) (result i32)))
            (memory (export "memory") 1) (data (i32.const 64) "xy")
            (func (export "_start") (drop (call $rejected {arguments}))))"#).into_bytes();
        input.module_sha256 = format!("{:x}", Sha256::digest(&input.module));
        let observed = linked::execute(input).unwrap();
        assert_eq!(observed.outcome.termination, "unsupported", "{name}");
        assert_eq!(observed.outcome.exit_code, None);
        assert!(
            observed
                .outcome
                .message
                .contains(&format!("operation outside stock WASI profile: {name}"))
        );
        assert!(observed.evidence.get("stdout").is_none_or(Vec::is_empty));
        assert!(observed.evidence.get("stderr").is_none_or(Vec::is_empty));
    }
}

#[test]
fn malformed_guest_memory_is_a_trap_instead_of_capacity_or_host_io() {
    let mut input = request("");
    input.module = br#"(module
        (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start") (drop (call $random (i32.const -1) (i32.const 2)))))"#
        .to_vec();
    input.module_sha256 = format!("{:x}", Sha256::digest(&input.module));
    let observed = linked::execute(input).unwrap();
    assert_eq!(observed.outcome.termination, "trap");
    assert_eq!(observed.outcome.exit_code, None);
    assert!(observed.evidence.get("stdout").is_none_or(Vec::is_empty));
    assert!(observed.evidence.get("stderr").is_none_or(Vec::is_empty));
}
