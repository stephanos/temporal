use super::*;
use std::os::unix::fs::DirBuilderExt;

struct Scratch(std::path::PathBuf);
impl Scratch {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-variant-custody-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .unwrap();
        Self(root)
    }
    fn native(&self) -> std::path::PathBuf {
        let module = self.0.join("module.wat");
        std::fs::write(
            &module,
            b"(module (memory (export \"memory\") 1) (func (export \"_start\")))",
        )
        .unwrap();
        let input = self.0.join("input.json");
        std::fs::write(&input,serde_json::to_vec(&json!({"kind":"standalone","guest":{
            "module":module,"module_sha256":canonical::digest(&std::fs::read(module).unwrap()),
            "limits":{"memory_bytes":65536,"fuel":500_000_000,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
            "model":null,"namespace":null,"storage":null},"network":null})).unwrap()).unwrap();
        let output = self.0.join("native");
        super::super::execution::run(&input, &output).unwrap();
        output
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn read(path: &Path) -> Value {
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}

#[test]
fn real_native_payload_mutation_cannot_become_available_canonical_evidence() {
    let scratch = Scratch::new();
    let output = scratch.native();
    let prior = read(&output.join("variant-artifacts.json"));
    assert_eq!(prior["artifacts"]["guest-stdout"]["status"], "available");
    std::fs::remove_file(output.join("variant-artifacts.json")).unwrap();
    std::fs::write(output.join("stdout"), b"forged after finalization").unwrap();
    let error = retain_execution(&output, Some("original collection failure")).unwrap_err();
    assert!(
        error.contains("size or digest mismatch") || error.contains("bound"),
        "{error}"
    );
    let catalog = read(&output.join("variant-artifacts.json"));
    assert_eq!(catalog["failure"], "original collection failure");
    for (meaning, _, _) in MEANINGS {
        assert_eq!(catalog["artifacts"][meaning]["status"], "unavailable");
        assert!(catalog["artifacts"][meaning].get("sha256").is_none());
    }
    assert_eq!(
        std::fs::read(output.join("stdout")).unwrap(),
        b"forged after finalization"
    );
}

impl Scratch {
    fn native_digest(&self) -> std::path::PathBuf {
        let module = self.0.join("digest.wat");
        std::fs::write(&module,br#"(module
            (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
            (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
            (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
            (memory (export "memory") 1)
            (func (export "_start")
                i32.const 0 i32.const 16 call $config drop
                i32.const 0 i64.const 0 i32.const 32 call $clock drop
                i32.const 0 i32.const 16 call $finish drop))"#).unwrap();
        let input = self.0.join("digest.json");
        std::fs::write(&input,serde_json::to_vec(&json!({"kind":"standalone","guest":{
            "module":module,"module_sha256":canonical::digest(&std::fs::read(&module).unwrap()),
            "limits":{"memory_bytes":65536,"fuel":500_000_000,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
            "model":{"configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/"),
                "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,"digest_only":true,
                "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},"arguments":[],"environment":["PWD=/"],"replay_trace":null},
            "namespace":{"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0},"storage":null},"network":null})).unwrap()).unwrap();
        let output = self.0.join("digest");
        super::super::execution::run(&input, &output).unwrap();
        output
    }
}
fn change_json(path: &Path, change: impl FnOnce(&mut Value)) {
    let mut value = read(path);
    change(&mut value);
    std::fs::write(path, serde_json::to_vec(&value).unwrap()).unwrap();
}
fn rebind_metadata(output: &Path, field: &str, file: &str) {
    let sha256 = canonical::digest(&std::fs::read(output.join(file)).unwrap());
    change_json(&output.join("canonical-streams.json"), |catalog| {
        catalog[field] = json!(sha256);
    });
}
fn mutate_digest_receipt(output: &Path, mutation: usize) {
    use std::os::unix::fs::PermissionsExt;
    match mutation {
        0 => change_json(&output.join("canonical-streams.json"), |catalog| {
            let mut digest = catalog["streams"]["wasi"]["sha256"]
                .as_str()
                .unwrap()
                .as_bytes()
                .to_vec();
            digest[0] = if digest[0] == b'0' { b'1' } else { b'0' };
            catalog["streams"]["wasi"]["sha256"] = json!(String::from_utf8(digest).unwrap());
        }),
        1 | 8 => {
            change_json(&output.join("request.json"), |request| {
                if mutation == 1 {
                    request["guest"]["model"]["configuration"]["digest_only"] = json!(false);
                } else {
                    request["guest"]["model"]["configuration"]
                        .as_object_mut()
                        .unwrap()
                        .remove("digest_only");
                }
            });
            let sha256 = canonical::digest(&std::fs::read(output.join("request.json")).unwrap());
            change_json(&output.join("report.json"), |report| {
                report["request_sha256"] = json!(sha256);
            });
        }
        2 => {
            std::fs::write(output.join("wasi"), b"").unwrap();
            std::fs::set_permissions(output.join("wasi"), std::fs::Permissions::from_mode(0o600))
                .unwrap();
        }
        3 | 9 => {
            change_json(&output.join("outcome.json"), |outcome| {
                outcome["local_model"]
                    .as_object_mut()
                    .unwrap()
                    .remove(if mutation == 3 {
                        "transcript_bytes"
                    } else {
                        "transcript_sha256"
                    });
            });
            rebind_metadata(output, "outcome_sha256", "outcome.json");
        }
        4 => {
            change_json(&output.join("streams.json"), |native| {
                native["transcript"] = json!({"bytes":0,"sha256":canonical::digest(b"")});
            });
            std::fs::write(output.join("transcript"), b"").unwrap();
            rebind_metadata(output, "native_streams_sha256", "streams.json");
        }
        5 => change_json(&output.join("canonical-streams.json"), |catalog| {
            catalog["streams"]["wasi"]
                .as_object_mut()
                .unwrap()
                .remove("file");
        }),
        6 => {
            change_json(&output.join("outcome.json"), |outcome| {
                outcome["local_model"]["transcript_sha256"] = json!("g".repeat(64));
            });
            change_json(&output.join("canonical-streams.json"), |catalog| {
                catalog["streams"]["wasi"]["sha256"] = json!("g".repeat(64));
            });
            rebind_metadata(output, "outcome_sha256", "outcome.json");
        }
        _ => std::fs::set_permissions(
            output.join("outcome.json"),
            std::fs::Permissions::from_mode(0o644),
        )
        .unwrap(),
    }
}

#[test]
fn actual_digest_only_wasi_receipt_refuses_commitment_flag_raw_presence_and_metadata_mutations() {
    for mutation in 0..10 {
        let scratch = Scratch::new();
        let output = scratch.native_digest();
        let prior = read(&output.join("variant-artifacts.json"));
        assert_eq!(
            prior["artifacts"]["canonical-wasi"]["status"],
            "digest-only"
        );
        assert!(prior["artifacts"]["canonical-wasi"]["file"].is_null());
        std::fs::remove_file(output.join("variant-artifacts.json")).unwrap();
        mutate_digest_receipt(&output, mutation);
        let error = retain_execution(&output, Some("original cost failure")).unwrap_err();
        assert!(!error.is_empty(), "mutation {mutation}");
        let refused = read(&output.join("variant-artifacts.json"));
        assert_eq!(refused["failure"], "original cost failure");
        assert_eq!(
            refused["artifacts"]["canonical-wasi"]["status"],
            "unavailable"
        );
        assert!(
            refused["artifacts"]["canonical-wasi"]
                .get("sha256")
                .is_none()
        );
    }
}
