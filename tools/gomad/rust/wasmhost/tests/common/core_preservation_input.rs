use gomad_wasmhost::linked;
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::time::Duration;

pub fn request(body: &str) -> linked::Request {
    let module = format!(
        r#"(module
        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "decision" (func $decision (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
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
