use super::{Manifest, Opened, Published, Store, canonical};
use crate::{
    campaign::{
        execution, ledger,
        prepare::{Prepared, process::Temporary},
    },
    evidence::record::{Outcome, Record},
};
use gomad_wasmhost::linked;
use serde_json::json;
use std::{collections::BTreeMap, path::Path, time::Duration};

fn publish(root: &Path, body: &str) -> Published {
    let module = format!(
        r#"(module
        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
        (memory (export "memory") 1)
        (func (export "_start") {body}))"#
    )
    .into_bytes();
    let configuration = serde_json::from_value(json!({
        "engine": linked::Conditions::compiled().configuration(),
        "profile": "gomad.wasi-stock-exploratory/v1",
        "limits":{"fuel":10000,"memory_bytes":65536,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
        "arguments":[],"environment":{},"runtime_sha256":null,"model":null,"namespace":null
    })).unwrap();
    let prepared = Prepared {
        module: module.clone(),
        configuration,
        toolchain: "a".repeat(64),
        choice_identity: None,
        scenario: None,
        provenance: b"{}".to_vec(),
        network: None,
        soft_assertion_report_only: false,
        compile_cache_root: None,
        auxiliary: BTreeMap::new(),
    };
    let mut native = linked::prepare(linked::Request {
        module_sha256: canonical::digest(&module),
        module,
        limits: linked::Limits {
            fuel: 10000,
            memory_bytes: 65536,
            wall_timeout: Duration::from_secs(5),
            output_bytes: 1024,
            evidence_bytes: 1 << 20,
        },
        model: None,
        namespace: None,
        storage: None,
    })
    .unwrap();
    native.conditions(linked::Conditions::compiled());
    let observed = linked::execute_prepared(native).unwrap();
    let publication = execution::publication(
        &prepared,
        &ledger::Job {
            ordinal: 0,
            seed: 7,
        },
        observed,
    )
    .unwrap()
    .publication;
    Store {
        root: root.to_path_buf(),
        maximum_bytes: 1 << 24,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(publication)
    .unwrap()
}

fn inspect(path: &Path, output: &mut Vec<u8>) -> Result<(), crate::cli::Error> {
    crate::cli::inspect::run(&["--json".into(), path.to_str().unwrap().into()], output)
}

fn assert_genuine(path: &Path) -> Record {
    let opened = Opened::open(path).unwrap();
    opened.authenticate_all().unwrap();
    let record = opened.manifest().record.clone();
    let mut output = Vec::new();
    inspect(path, &mut output).unwrap_or_else(|error| panic!("{}", error.message));
    let report: serde_json::Value = serde_json::from_slice(&output).unwrap();
    assert_eq!(
        report["artifact"]["record"],
        serde_json::to_value(&record).unwrap()
    );
    record
}

fn reseal(path: &Path, outcome: Outcome) {
    let mut manifest: Manifest =
        serde_json::from_slice(&std::fs::read(path.join("manifest.json")).unwrap()).unwrap();
    manifest.record.outcome = outcome;
    let name = "evidence/backend-outcome.json";
    let mut backend: serde_json::Value =
        serde_json::from_slice(&std::fs::read(path.join(name)).unwrap()).unwrap();
    backend["termination"] = manifest.record.outcome.termination.clone().into();
    backend["exit_code"] = serde_json::to_value(manifest.record.outcome.exit_code).unwrap();
    let backend = canonical::encode(&backend).unwrap();
    std::fs::write(path.join(name), &backend).unwrap();
    let entry = manifest.files.get_mut(name).unwrap();
    entry.bytes = backend.len().try_into().unwrap();
    entry.sha256 = canonical::digest(&backend);
    if let Some(stream) = manifest.record.streams.get_mut(name) {
        stream.bytes = entry.bytes;
        stream.sha256.clone_from(&entry.sha256);
    }
    let record = canonical::encode(&manifest.record).unwrap();
    manifest.record_sha256 = canonical::digest(&record);
    let entry = manifest.files.get_mut("record.json").unwrap();
    entry.bytes = record.len().try_into().unwrap();
    entry.sha256.clone_from(&manifest.record_sha256);
    std::fs::write(path.join("record.json"), &record).unwrap();
    std::fs::write(
        path.join("manifest.json"),
        canonical::encode(&manifest).unwrap(),
    )
    .unwrap();

    assert_eq!(canonical::digest(&record), manifest.record_sha256);
    assert_eq!(
        serde_json::from_slice::<Record>(&record).unwrap(),
        manifest.record
    );
    for (name, entry) in &manifest.files {
        let bytes = std::fs::read(path.join(name)).unwrap();
        assert_eq!(entry.bytes, u64::try_from(bytes.len()).unwrap(), "{name}");
        assert_eq!(entry.sha256, canonical::digest(&bytes), "{name}");
    }
    for (name, stream) in &manifest.record.streams {
        let entry = &manifest.files[name];
        assert_eq!(stream.bytes, entry.bytes, "{name}");
        assert_eq!(stream.sha256, entry.sha256, "{name}");
    }
}

fn rejects_resealed(outcome: Outcome) {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let published = publish(&root.0.join("store"), "");
    assert_eq!(assert_genuine(&published.path).outcome.termination, "exit");
    reseal(&published.path, outcome);
    let authentication = Opened::open(&published.path).and_then(|opened| opened.authenticate_all());
    let mut output = Vec::new();
    let inspection = inspect(&published.path, &mut output);
    assert!(
        authentication.is_err(),
        "public intake accepted a coherently resealed invalid record outcome"
    );
    assert!(
        inspection.is_err(),
        "public inspect accepted a coherently resealed invalid record outcome"
    );
    assert!(
        authentication
            .unwrap_err()
            .contains("invalid record outcome")
    );
    assert!(
        inspection
            .err()
            .unwrap()
            .message
            .contains("invalid record outcome")
    );
    assert_eq!(output, Vec::<u8>::new());
}

#[test]
fn genuine_native_exit_trap_and_failure_records_authenticate_and_inspect() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    for (index, body, class, termination, code) in [
        (0, "", "success", "exit", Some(0)),
        (1, "unreachable", "target_failure", "trap", None),
        (
            2,
            "(call $exit (i32.const 7))",
            "target_failure",
            "exit",
            Some(7),
        ),
    ] {
        let published = publish(&root.0.join(index.to_string()), body);
        let record = assert_genuine(&published.path);
        assert_eq!(
            (
                record.outcome.class.as_str(),
                record.outcome.termination.as_str(),
                record.outcome.exit_code
            ),
            (class, termination, code)
        );
    }
}

#[test]
fn resealed_unknown_record_termination_is_rejected_by_public_intake() {
    rejects_resealed(Outcome {
        class: "success".into(),
        termination: "future_reply".into(),
        exit_code: Some(0),
        reason: String::new(),
    });
}

#[test]
fn resealed_trap_record_with_exit_status_is_rejected_by_public_intake() {
    rejects_resealed(Outcome {
        class: "target_failure".into(),
        termination: "trap".into(),
        exit_code: Some(0),
        reason: String::new(),
    });
}

#[test]
fn resealed_exit_record_without_exit_status_is_rejected_by_public_intake() {
    rejects_resealed(Outcome {
        class: "target_failure".into(),
        termination: "exit".into(),
        exit_code: None,
        reason: String::new(),
    });
}
