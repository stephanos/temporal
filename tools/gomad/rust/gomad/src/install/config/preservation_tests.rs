use super::*;
use crate::{campaign::prepare::process::Temporary, evidence::canonical};

fn execute(config: &Configuration, imports: &str, body: &str) -> linked::Execution {
    let module = format!(
        r#"(module {imports}
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start") {body}))"#
    )
    .into_bytes();
    linked::execute(linked::Request {
        module_sha256: digest(&module),
        module,
        limits: linked::Limits {
            memory_bytes: 65536,
            fuel: 10000,
            output_bytes: config.namespace.limits.output_bytes,
            evidence_bytes: 1 << 20,
            wall_timeout: std::time::Duration::from_secs(5),
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(config.model.clone()).unwrap(),
            arguments: config.arguments.clone(),
            environment: config
                .environment
                .iter()
                .map(|(k, v)| format!("{k}={v}"))
                .collect(),
            replay_trace: None,
        }),
        namespace: Some(config.namespace.clone()),
        storage: config.storage.clone(),
    })
    .unwrap()
}

fn original() -> Value {
    let mut value = default_value(STOCK);
    value["Args"] = serde_json::json!(["guest", "literal $arg"]);
    value["Environment"] = serde_json::json!(["PWD=/workspace", "TZ=UTC", "ONLY=captured"]);
    value["EntropyKey"] = serde_json::json!([
        1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0
    ]);
    value
}

#[test]
fn captured_arguments_environment_and_entropy_keep_complete_immutable_configuration_identity() {
    let mut caller = original();
    let captured = convert(caller.clone(), STOCK).unwrap();
    caller["Args"][0] = "mutated".into();
    let imports = r#"
        (import "wasi_snapshot_preview1" "args_get" (func $args (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "args_sizes_get" (func $sizes (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "environ_sizes_get" (func $env_sizes (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "environ_get" (func $env (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))"#;
    let check_inputs = r"
        (if (call $sizes (i32.const 0) (i32.const 4)) (then unreachable))
        (if (i32.ne (i32.load (i32.const 0)) (i32.const 2)) (then unreachable))
        (if (i32.ne (i32.load (i32.const 4)) (i32.const 19)) (then unreachable))
        (if (call $args (i32.const 8) (i32.const 32)) (then unreachable))
        (if (call $env_sizes (i32.const 16) (i32.const 20)) (then unreachable))
        (if (i32.ne (i32.load (i32.const 16)) (i32.const 3)) (then unreachable))
        (if (i32.ne (i32.load (i32.const 20)) (i32.const 36)) (then unreachable))
        (if (call $env (i32.const 64) (i32.const 96)) (then unreachable))
        (i32.store (i32.const 160) (i32.const 32))
        (i32.store (i32.const 164) (i32.const 19))
        (drop (call $write (i32.const 1) (i32.const 160) (i32.const 1) (i32.const 168)))
        (i32.store (i32.const 160) (i32.const 96))
        (i32.store (i32.const 164) (i32.const 36))
        (drop (call $write (i32.const 1) (i32.const 160) (i32.const 1) (i32.const 168)))
    ";
    let emit_random = r"
        (i32.store (i32.const 160) (i32.const 256))
        (i32.store (i32.const 164) (i32.const 64))
        (drop (call $write (i32.const 1) (i32.const 160) (i32.const 1) (i32.const 168)))
    ";
    let observed = execute(
        &captured,
        imports,
        &format!(
            r"{check_inputs}
        (drop (call $random (i32.const 256) (i32.const 7)))
        (drop (call $random (i32.const 263) (i32.const 57))) {emit_random}"
        ),
    );
    let fresh = convert(original(), STOCK).unwrap();
    let whole = execute(
        &fresh,
        imports,
        &format!(
            r"{check_inputs}
        (drop (call $random (i32.const 256) (i32.const 64))) {emit_random}"
        ),
    );
    assert_eq!(
        observed.outcome.termination, "exit",
        "{}",
        observed.outcome.message
    );
    assert_eq!(
        whole.outcome.termination, "exit",
        "{}",
        whole.outcome.message
    );
    let bytes = &observed.evidence["stdout"];
    assert_eq!(bytes, &whole.evidence["stdout"]);
    assert_eq!(&bytes[..19], b"guest\0literal $arg\0");
    assert_eq!(&bytes[19..55], b"ONLY=captured\0PWD=/workspace\0TZ=UTC\0");
    assert_ne!(&bytes[55..62], &bytes[62..69]);
    assert_eq!(captured.original, fresh.original);
    assert_eq!(
        canonical::digest(&canonical::encode(&captured.original).unwrap()),
        canonical::digest(&canonical::encode(&fresh.original).unwrap())
    );
    assert_ne!(captured.original, caller);
}

#[test]
fn captured_namespace_reads_after_host_and_capture_buffers_are_removed_and_infers_readonly_absence()
{
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/preservation-tests"))
            .unwrap();
    let source = root.0.join("schema/sqlite");
    std::fs::create_dir_all(&source).unwrap();
    std::fs::write(source.join("input"), b"frozen").unwrap();
    let mut caller = temporal_policy(&root.0, "./tests", TemporalPolicy::Qualification).unwrap();
    let mut captured = convert(caller.clone(), COOPERATIVE).unwrap();
    for value in [&mut caller, &mut captured.original] {
        value["CapturedInputs"]["Payloads"] = serde_json::json!({});
        value["CapturedInputs"]["Descriptor"] = Value::Null;
        value["CapturedInputs"]["Manifest"]["Mappings"] = serde_json::json!([]);
    }
    std::fs::remove_dir_all(&source).unwrap();
    let preopen = captured
        .namespace
        .preopens
        .iter()
        .position(|path| path == b"/workspace/schema/sqlite")
        .unwrap()
        + 3;
    let imports = r#"
        (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "fd_read" (func $read (param i32 i32 i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "path_filestat_get" (func $stat (param i32 i32 i32 i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "path_create_directory" (func $mkdir (param i32 i32 i32) (result i32)))"#;
    let observed = execute(
        &captured,
        imports,
        &format!(
            r"
        (i64.store (i32.const 64) (i64.const 0x7475706e69))
        (if (call $open (i32.const {preopen}) (i32.const 0) (i32.const 64) (i32.const 5) (i32.const 0) (i64.const 2) (i64.const 0) (i32.const 0) (i32.const 0)) (then unreachable))
        (i32.store (i32.const 16) (i32.const 128))
        (i32.store (i32.const 20) (i32.const 20))
        (if (call $read (i32.load (i32.const 0)) (i32.const 16) (i32.const 1) (i32.const 24)) (then unreachable))
        (if (i32.ne (i32.load (i32.const 24)) (i32.const 6)) (then unreachable))
        (i32.store (i32.const 20) (i32.const 6))
        (drop (call $write (i32.const 1) (i32.const 16) (i32.const 1) (i32.const 24)))
        (i64.store (i32.const 64) (i64.const 0x676e697373696d))
        (if (i32.ne (call $stat (i32.const {preopen}) (i32.const 0) (i32.const 64) (i32.const 7) (i32.const 256)) (i32.const 44)) (then unreachable))
        (i32.store (i32.const 64) (i32.const 7824750))
        (if (i32.ne (call $mkdir (i32.const {preopen}) (i32.const 64) (i32.const 3)) (i32.const 69)) (then unreachable))"
        ),
    );
    assert_eq!(
        observed.outcome.termination, "exit",
        "{}",
        observed.outcome.message
    );
    assert_eq!(observed.outcome.exit_code, Some(0));
    assert_eq!(observed.evidence["stdout"], b"frozen");
    assert!(!source.exists());
}

#[test]
fn environment_rejects_each_original_ambiguous_configuration() {
    for case in ["pwd", "duplicate", "nul", "limits", "profile"] {
        let mut value = original();
        match case {
            "pwd" => value["WorkingDirectory"] = "/other".into(),
            "duplicate" => value["Environment"]
                .as_array_mut()
                .unwrap()
                .push("ONLY=again".into()),
            "nul" => value["Args"][0] = "bad\0arg".into(),
            "limits" => value["Limits"]["Descriptors"] = 0.into(),
            "profile" => value["Profile"] = "strict".into(),
            _ => unreachable!(),
        }
        assert!(convert(value, STOCK).is_err(), "{case}");
    }
}
