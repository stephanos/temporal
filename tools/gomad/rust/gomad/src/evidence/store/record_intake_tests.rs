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

fn native_publication(
    body: &str,
    change: impl FnOnce(&mut linked::Execution),
) -> Result<super::Publication, String> {
    native_publication_with_memory(body, 1, change)
}

fn native_publication_with_memory(
    body: &str,
    pages: u32,
    change: impl FnOnce(&mut linked::Execution),
) -> Result<super::Publication, String> {
    let module = format!(
        r#"(module
        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
        (memory (export "memory") {pages})
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
    let mut observed = linked::execute_prepared(native).unwrap();
    change(&mut observed);
    execution::publication(
        &prepared,
        &ledger::Job {
            ordinal: 0,
            seed: 7,
        },
        observed,
    )
    .map(|result| result.publication)
}

fn publish(root: &Path, body: &str) -> Published {
    let publication = native_publication(body, |_| {}).unwrap();
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

#[test]
fn resealed_unknown_nested_record_fields_refuse_before_inspection_output() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let published = publish(&root.0.join("store"), "");
    assert_genuine(&published.path);
    let manifest_path = published.path.join("manifest.json");
    let mut manifest: serde_json::Value =
        serde_json::from_slice(&std::fs::read(&manifest_path).unwrap()).unwrap();
    manifest["record"]["outcome"]["future"] = true.into();
    let bytes = canonical::encode(&manifest["record"]).unwrap();
    let digest = canonical::digest(&bytes);
    manifest["record_sha256"] = digest.clone().into();
    manifest["files"]["record.json"]["sha256"] = digest.into();
    manifest["files"]["record.json"]["bytes"] = bytes.len().into();
    std::fs::write(published.path.join("record.json"), &bytes).unwrap();
    std::fs::write(&manifest_path, canonical::encode(&manifest).unwrap()).unwrap();
    for (name, entry) in manifest["files"].as_object().unwrap() {
        let bytes = std::fs::read(published.path.join(name)).unwrap();
        assert_eq!(entry["bytes"], bytes.len());
        assert_eq!(entry["sha256"], canonical::digest(&bytes));
    }
    let error = Opened::open(&published.path).err().unwrap();
    assert!(error.contains("unknown field `future`"), "{error}");
    let mut output = Vec::new();
    let error = inspect(&published.path, &mut output).err().unwrap();
    assert!(
        error.message.contains("unknown field `future`"),
        "{}",
        error.message
    );
    assert_eq!(output, Vec::<u8>::new());
}

fn reseal_backend_counter(path: &Path, field: &str, value: u64) {
    let mut manifest: Manifest =
        serde_json::from_slice(&std::fs::read(path.join("manifest.json")).unwrap()).unwrap();
    let name = "evidence/backend-outcome.json";
    let mut backend: serde_json::Value =
        serde_json::from_slice(&std::fs::read(path.join(name)).unwrap()).unwrap();
    backend[field] = value.into();
    let bytes = canonical::encode(&backend).unwrap();
    std::fs::write(path.join(name), &bytes).unwrap();
    let entry = manifest.files.get_mut(name).unwrap();
    entry.bytes = bytes.len().try_into().unwrap();
    entry.sha256 = canonical::digest(&bytes);
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
    for (name, entry) in &manifest.files {
        let bytes = std::fs::read(path.join(name)).unwrap();
        assert_eq!(entry.bytes, u64::try_from(bytes.len()).unwrap(), "{name}");
        assert_eq!(entry.sha256, canonical::digest(&bytes), "{name}");
    }
    assert_eq!(manifest.record_sha256, canonical::digest(&record));
}

#[test]
fn resealed_backend_fuel_above_admitted_limit_refuses_public_intake() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let published = publish(&root.0.join("store"), "");
    let record = assert_genuine(&published.path);
    let backend: serde_json::Value = serde_json::from_slice(
        &std::fs::read(published.path.join("evidence/backend-outcome.json")).unwrap(),
    )
    .unwrap();
    assert!(backend["fuel_remaining"].as_u64().unwrap() <= record.limits.fuel);
    reseal_backend_counter(&published.path, "fuel_remaining", record.limits.fuel + 1);
    let authentication = Opened::open(&published.path).and_then(|opened| opened.authenticate_all());
    let mut output = Vec::new();
    let inspection = inspect(&published.path, &mut output);
    assert!(authentication.unwrap_err().contains("backend outcome fuel"));
    assert!(
        inspection
            .err()
            .unwrap()
            .message
            .contains("backend outcome fuel")
    );
    assert_eq!(output, Vec::<u8>::new());
}

#[test]
fn native_terminal_snapshots_offered_output_before_public_outcome_is_mutable() {
    let module = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))".to_vec();
    let mut prepared = linked::prepare(linked::Request {
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
    prepared.conditions(linked::Conditions::compiled());
    let mut observed = std::thread::scope(|scope| {
        scope
            .spawn(move || linked::execute_prepared(prepared))
            .join()
            .unwrap()
            .unwrap()
    });
    assert_eq!(observed.outcome.termination, "exit");
    assert_eq!(observed.outcome.output_bytes, 0);
    let raw = observed
        .evidence
        .get("native-output-accounting")
        .expect("native terminal omitted its independent offered-output snapshot")
        .clone();
    let accounting: serde_json::Value = serde_json::from_slice(&raw).unwrap();
    assert_eq!(accounting["schema"], "gomad.native-output-accounting/v1");
    assert_eq!(accounting["offered_output_bytes"], 0);
    observed.outcome.output_bytes = 1;
    assert_eq!(observed.evidence["native-output-accounting"], raw);
    assert!(observed.evidence.get("stdout").is_none_or(Vec::is_empty));
    assert!(observed.evidence.get("stderr").is_none_or(Vec::is_empty));
    let timing = observed.timing.unwrap();
    assert!(timing.authorization_started <= timing.terminal_classified);
    assert!(timing.terminal_classified <= timing.returned);
}

#[test]
fn resealed_backend_invented_output_refuses_public_intake() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let published = publish(&root.0.join("store"), "");
    let record = assert_genuine(&published.path);
    assert!(!record.streams.contains_key("output/stdout"));
    assert!(!record.streams.contains_key("output/stderr"));
    let backend: serde_json::Value = serde_json::from_slice(
        &std::fs::read(published.path.join("evidence/backend-outcome.json")).unwrap(),
    )
    .unwrap();
    assert_eq!(backend["output_bytes"], 0);
    reseal_backend_counter(&published.path, "output_bytes", 1);
    let authentication = Opened::open(&published.path).and_then(|opened| opened.authenticate_all());
    let mut output = Vec::new();
    let inspection = inspect(&published.path, &mut output);
    assert!(
        authentication
            .unwrap_err()
            .contains("backend outcome output")
    );
    assert!(
        inspection
            .err()
            .unwrap()
            .message
            .contains("backend outcome output")
    );
    assert_eq!(output, Vec::<u8>::new());
}

#[test]
fn publication_rejects_mutated_native_output_claim_before_writing_an_artifact() {
    let error = native_publication("", |execution| execution.outcome.output_bytes = 1)
        .err()
        .expect("publication accepted a claim contradicting the native store snapshot");
    assert!(error.contains("backend outcome output"), "{error}");
}

#[test]
fn native_publication_versions_and_binds_the_independent_output_snapshot() {
    let publication = native_publication("", |_| {}).unwrap();
    let raw = &publication.payloads["evidence/native-output-accounting"];
    let backend: serde_json::Value =
        serde_json::from_slice(&publication.payloads["evidence/backend-outcome.json"]).unwrap();
    assert_eq!(backend["schema"], "gomad.backend-outcome/v2");
    assert_eq!(backend["output_accounting_sha256"], canonical::digest(raw));
    let accounting: serde_json::Value = serde_json::from_slice(raw).unwrap();
    assert_eq!(accounting["offered_output_bytes"], backend["output_bytes"]);
}

#[test]
fn preinstantiation_capacity_keeps_zero_output_accounting_without_model_metadata() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let publication = native_publication_with_memory("", 2, |observed| {
        assert_eq!(observed.outcome.termination, "capacity");
        assert!(observed.outcome.message.contains("memory"));
        assert!(observed.outcome.local_model.is_none());
        assert_eq!(observed.outcome.output_bytes, 0);
    })
    .unwrap();
    let published = Store {
        root: root.0.join("preinstantiation-capacity"),
        maximum_bytes: 1 << 24,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(publication)
    .unwrap();
    assert_genuine(&published.path);
}

fn legacy_publication(mut publication: super::Publication) -> super::Publication {
    let name = "evidence/native-output-accounting";
    publication.payloads.remove(name);
    publication.record.streams.remove(name);
    let name = "evidence/backend-outcome.json";
    let mut backend: serde_json::Value =
        serde_json::from_slice(&publication.payloads[name]).unwrap();
    backend["schema"] = "gomad.backend-outcome/v1".into();
    backend
        .as_object_mut()
        .unwrap()
        .remove("output_accounting_sha256");
    replace_publication_backend(&mut publication, &backend);
    publication
}

fn replace_publication_backend(publication: &mut super::Publication, backend: &serde_json::Value) {
    let name = "evidence/backend-outcome.json";
    let bytes = canonical::encode(backend).unwrap();
    if let Some(stream) = publication.record.streams.get_mut(name) {
        stream.bytes = bytes.len().try_into().unwrap();
        stream.sha256 = canonical::digest(&bytes);
    }
    publication.payloads.insert(name.into(), bytes);
    publication
        .payloads
        .insert("record.json".into(), publication.record.bytes().unwrap());
}

#[test]
fn store_publication_rejects_contradictory_native_backend_counters() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    for (field, value, diagnostic) in [
        ("fuel_remaining", 10001, "backend outcome fuel"),
        ("output_bytes", 1, "backend outcome output"),
    ] {
        let mut publication = native_publication("", |_| {}).unwrap();
        let mut backend: serde_json::Value =
            serde_json::from_slice(&publication.payloads["evidence/backend-outcome.json"]).unwrap();
        backend[field] = value.into();
        replace_publication_backend(&mut publication, &backend);
        let store = Store {
            root: root.0.join(field),
            maximum_bytes: 1 << 24,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        };
        let error = store
            .publish(publication)
            .err()
            .expect("invalid counters were committed");
        assert!(error.contains(diagnostic), "{error}");
        let committed = std::fs::read_dir(&store.root)
            .unwrap()
            .map(Result::unwrap)
            .filter(|entry| super::artifact_name(&entry.file_name().to_string_lossy()))
            .count();
        assert_eq!(committed, 0);
    }
}

#[test]
fn backend_terminal_claims_must_match_the_valid_record() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    for (index, termination, exit_code) in [
        (0, "trap", Some(0)),
        (1, "exit", None),
        (2, "exit", Some(7)),
    ] {
        let mut publication = native_publication("", |_| {}).unwrap();
        let mut backend: serde_json::Value =
            serde_json::from_slice(&publication.payloads["evidence/backend-outcome.json"]).unwrap();
        backend["termination"] = termination.into();
        backend["exit_code"] = serde_json::to_value(exit_code).unwrap();
        replace_publication_backend(&mut publication, &backend);
        publication.record.validate().unwrap();
        let error = Store {
            root: root.0.join(index.to_string()),
            maximum_bytes: 1 << 24,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
        .publish(publication)
        .err()
        .expect("contradictory backend terminal was committed");
        assert!(error.contains("backend outcome terminal"), "{error}");
    }
}

#[test]
fn legacy_v1_backend_records_remain_readable_without_rewriting_canonical_bytes() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let publication = legacy_publication(native_publication("", |_| {}).unwrap());
    let name = "evidence/backend-outcome.json";
    let legacy_bytes = publication.payloads[name].clone();
    let legacy_record = publication.record.bytes().unwrap();
    let published = Store {
        root: root.0.join("legacy-store"),
        maximum_bytes: 1 << 24,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(publication)
    .unwrap();
    assert_genuine(&published.path);
    assert_eq!(
        std::fs::read(published.path.join(name)).unwrap(),
        legacy_bytes
    );
    assert_eq!(
        std::fs::read(published.path.join("record.json")).unwrap(),
        legacy_record
    );
}

#[test]
fn legacy_v1_cancelled_output_remains_readable_with_the_historical_zero_counter() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    let (publication, _, retained) = offered_output_publication(1 << 20, true);
    assert!(retained > 0);
    let mut publication = legacy_publication(publication);
    let name = "evidence/backend-outcome.json";
    let mut backend: serde_json::Value =
        serde_json::from_slice(&publication.payloads[name]).unwrap();
    backend["output_bytes"] = 0.into();
    replace_publication_backend(&mut publication, &backend);
    let legacy_bytes = publication.payloads[name].clone();
    let legacy_record = publication.record.bytes().unwrap();
    assert_eq!(publication.record.outcome.termination, "cancelled");
    let published = Store {
        root: root.0.join("legacy-cancelled"),
        maximum_bytes: 1 << 24,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(publication)
    .unwrap();
    assert_genuine(&published.path);
    assert_eq!(
        std::fs::read(published.path.join(name)).unwrap(),
        legacy_bytes
    );
    assert_eq!(
        std::fs::read(published.path.join("record.json")).unwrap(),
        legacy_record
    );
}

fn offered_output_fixture(evidence_bytes: u64) -> (Prepared, linked::Prepared) {
    let profile = "gomad.wasi-stock-exploratory/v1";
    let converted = crate::install::config::convert(
        json!({
            "Profile":profile,"Args":["guest"],"Environment":["PWD=/workspace"],
            "WorkingDirectory":"/workspace","WritableDirectories":["/workspace","/tmp"],
            "Clock":{"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":1000},
            "Limits":{"OutputBytes":1024,"TranscriptBytes":65536,"Calls":10000,"Descriptors":64,
                "Files":1000,"FilesystemBytes":1_048_576,"PendingEvents":4096}
        }),
        profile,
    )
    .unwrap();
    let module = br#"(module
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "fd_close" (func $close (param i32) (result i32)))
        (memory (export "memory") 1) (data (i32.const 32) "x")
        (func (export "_start")
            (i32.store (i32.const 0) (i32.const 32))
            (i32.store (i32.const 4) (i32.const 1))
            (if (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 8)) (then unreachable))
            (if (call $close (i32.const 1)) (then unreachable))
            (if (i32.ne (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 8)) (i32.const 8)) (then unreachable))
            (if (call $write (i32.const 2) (i32.const 0) (i32.const 1) (i32.const 8)) (then unreachable))))"#.to_vec();
    let configuration = serde_json::from_value(json!({
        "engine":linked::Conditions::compiled().configuration(),"profile":profile,
        "limits":{"fuel":10000,"memory_bytes":65536,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":evidence_bytes},
        "arguments":converted.arguments,"environment":converted.environment,"runtime_sha256":null,
        "model":converted.model,"namespace":converted.namespace
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
    let native = linked::prepare(linked::Request {
        module_sha256: canonical::digest(&module),
        module,
        limits: linked::Limits {
            fuel: 10000,
            memory_bytes: 65536,
            wall_timeout: Duration::from_secs(5),
            output_bytes: 1024,
            evidence_bytes,
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(converted.model).unwrap(),
            arguments: converted.arguments,
            environment: converted
                .environment
                .into_iter()
                .map(|(k, v)| format!("{k}={v}"))
                .collect(),
            replay_trace: None,
        }),
        namespace: Some(converted.namespace),
        storage: None,
    })
    .unwrap();
    (prepared, native)
}

struct CancelAfterOutput(linked::Cancellation);
impl linked::OutputConsumer for CancelAfterOutput {
    fn write(&self, _: linked::OutputStream, _: &[u8]) -> std::io::Result<()> {
        self.0.cancel();
        Ok(())
    }
}

fn offered_output_publication(evidence_bytes: u64, cancel: bool) -> (super::Publication, u64, u64) {
    let (prepared, mut native) = offered_output_fixture(evidence_bytes);
    native.conditions(linked::Conditions::compiled());
    if cancel {
        let cancellation = linked::Cancellation::default();
        native.cancellation(cancellation.clone());
        native.output(std::sync::Arc::new(CancelAfterOutput(cancellation)));
    }
    let observed = std::thread::scope(|scope| {
        scope
            .spawn(move || linked::execute_prepared(native))
            .join()
            .unwrap()
            .unwrap()
    });
    if cancel {
        assert!(observed.outcome.peak_memory_bytes > 0);
        assert!(observed.outcome.local_model.is_some());
        assert!(observed.outcome.metrics.is_some());
    }
    let offered = observed.outcome.output_bytes;
    let retained = ["stdout", "stderr"]
        .into_iter()
        .map(|name| {
            observed
                .evidence
                .get(name)
                .map_or(0, |bytes| bytes.len() as u64)
        })
        .sum();
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
    (publication, offered, retained)
}

#[test]
fn closed_descriptor_attempts_and_capacity_prefixes_keep_distinct_offered_output_counts() {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/record-intake-tests"))
            .unwrap();
    for (name, evidence_bytes, termination) in [
        ("closed", 1 << 20, "exit"),
        ("capacity", 1, "capacity"),
        ("cancelled", 1 << 20, "cancelled"),
    ] {
        let (publication, offered, retained) =
            offered_output_publication(evidence_bytes, name == "cancelled");
        assert_eq!(
            publication.record.outcome.termination, termination,
            "{name}"
        );
        if name == "cancelled" {
            assert!(
                offered >= retained && retained > 0,
                "offered={offered} retained={retained}"
            );
        } else {
            assert_eq!(offered, 3, "{name}");
            assert!(
                offered > retained,
                "{name}: offered={offered} retained={retained}"
            );
        }
        if name == "closed" {
            assert_eq!(retained, 2);
        }
        let published = Store {
            root: root.0.join(name),
            maximum_bytes: 1 << 24,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
        .publish(publication)
        .unwrap();
        assert_genuine(&published.path);
    }
}
