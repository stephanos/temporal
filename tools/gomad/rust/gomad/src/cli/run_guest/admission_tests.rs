use super::*;
use crate::{campaign::prepare::Prepared, replay::preflight::Configuration};
use serde_json::json;
use std::{collections::BTreeMap, fs, os::unix::fs::PermissionsExt as _};

fn prepared(label: &str, invocation: &origin::Invocation) -> Prepared {
    let mut module = b"\0asm\x01\0\0\0".to_vec();
    module.extend_from_slice(&[0, u8::try_from(label.len() + 1).unwrap(), 0]);
    module.extend_from_slice(label.as_bytes());
    prepared_module(label, invocation, module)
}

fn prepared_module(label: &str, invocation: &origin::Invocation, module: Vec<u8>) -> Prepared {
    let configuration = Configuration {
        engine: gomad_wasmhost::linked::Conditions::compiled().configuration(),
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        limits: crate::evidence::record::Limits {
            fuel: 1000,
            memory_bytes: 1 << 20,
            wall_millis: 1000,
            output_bytes: 1 << 20,
            evidence_bytes: 16 << 20,
        },
        wall_nanos: None,
        arguments: vec![label.into()],
        environment: BTreeMap::from([("PWD".into(), "/".into())]),
        derive_seed_environment: true,
        network: None,
        runtime_sha256: None,
        model: Some(json!({
            "entropy_key":vec![0_u8;32], "working_directory_sha256":canonical::digest(b"/"),
            "arguments":{"bytes":0,"count":0,"chunks":0}, "environment":{"bytes":0,"count":0,"chunks":0},
            "mode":1,"limit":1_u64<<20,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":1_u64<<20,"pending_events":16}
        })),
        namespace: Some(gomad_wasmhost::linked::NamespaceImage {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(),
            stdin: Vec::new(),
            epoch_nanos: 0,
            limits: gomad_wasmhost::linked::NamespaceLimits {
                files: 16,
                descriptors: 16,
                filesystem_bytes: 4096,
                output_bytes: 1 << 20,
            },
        }),
        storage: None,
    };
    let inputs = json!({
        "schema":"gomad.rust-wasm-build-inputs/v1", "compiler_path":"/fixture/go",
        "compiler_sha256":"2".repeat(64), "go_version":"go1.27.1", "configuration":configuration,
        "runtime_implementation":null, "host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "helper":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "compiler":{"/fixture/go":"2".repeat(64)}, "sources":{"/fixture/source":"3".repeat(64)}
    });
    let toolchain = canonical::digest(&canonical::encode(&inputs).unwrap());
    let provenance = canonical::encode(&json!({
        "schema":"gomad.rust-wasm-provenance/v1", "inputs":inputs, "build_key":toolchain,
        "module_sha256":canonical::digest(&module), "configuration":configuration,
        "choice_identity":null, "scenario":null, "network":null
    }))
    .unwrap();
    let mut prepared = Prepared {
        module,
        configuration,
        toolchain,
        choice_identity: None,
        scenario: None,
        provenance,
        network: None,
        soft_assertion_report_only: false,
    };
    origin::intercept(&mut prepared, invocation.clone()).unwrap();
    prepared
}

fn invocation() -> origin::Invocation {
    origin::Invocation {
        schema: "gomad.intercepted-invocation/v1".into(),
        wrapper_run_id: "1".repeat(64),
        id: "2".repeat(64),
        original_argv: vec![origin::argument(OsStr::new("go"))],
        driver_argv: vec![origin::argument(OsStr::new("go"))],
        compiler_sha256: "3".repeat(64),
        installation_sha256: "4".repeat(64),
        recipe_sha256: "5".repeat(64),
        recipe_file: "driver-recipe.json".into(),
    }
}

#[test]
fn repeated_and_concurrent_admissions_have_private_distinct_bindings() {
    let root =
        std::env::temp_dir().join(format!("gomad-admission-{}", origin::identity().unwrap()));
    let invocation = invocation();
    let publish = |label: &str| {
        let mut ignored = false;
        let mut options =
            super::super::explore::parse(&["go-run".into(), "fixture".into()], &mut ignored)
                .unwrap();
        options.artifacts = root.clone();
        let prepared = prepared(label, &invocation);
        let path = admission_attempt(&root, &invocation.id).unwrap();
        let retained = preparation::retain_admitted(&options, prepared).unwrap();
        let (path, receipt, digest) = retained
            .publish(
                &path.join("plan.json"),
                vec!["run-guest".into(), label.into()],
            )
            .unwrap();
        (path, receipt, digest)
    };
    let first = publish("repeated");
    let second = publish("repeated");
    let concurrent = std::thread::scope(|scope| {
        let a = scope.spawn(|| publish("parallel-a"));
        let b = scope.spawn(|| publish("parallel-b"));
        [a.join().unwrap(), b.join().unwrap()]
    });
    let [third, fourth] = concurrent;
    let receipts = [first, second, third, fourth];
    for (index, (path, receipt, digest)) in receipts.iter().enumerate() {
        let (_, prepared) = preparation::open_receipt(path, digest).unwrap();
        let bound = origin::from_preparation(&prepared).unwrap().unwrap();
        assert_eq!(bound.invocation, invocation);
        assert_eq!(bound.module_sha256, receipt.module_sha256);
        assert_eq!(
            canonical::digest(&prepared.provenance),
            receipt.provenance_sha256
        );
        assert_eq!(receipt.process_id, std::process::id());
        for field in ["module_sha256", "provenance_sha256", "record_sha256"] {
            let mut forged = serde_json::to_value(receipt).unwrap();
            forged[field] = "9".repeat(64).into();
            let forged = canonical::encode(&forged).unwrap();
            let forged_path = path.with_file_name(format!("forged-{field}.json"));
            fs::write(&forged_path, &forged).unwrap();
            fs::set_permissions(&forged_path, fs::Permissions::from_mode(0o600)).unwrap();
            let error = preparation::open_receipt(&forged_path, &canonical::digest(&forged))
                .err()
                .expect("forged receipt accepted");
            assert!(
                error.contains("receipt") && error.contains("mismatch"),
                "{error}"
            );
        }
        for (other, (other_path, _, other_digest)) in receipts.iter().enumerate() {
            if index != other {
                assert_ne!(path, other_path);
                assert!(preparation::open_receipt(path, other_digest).is_err());
            }
        }
    }
    assert!(admission_attempt(&root, "../unbound").is_err());
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn compiled_output_callback_failure_retains_partial_publication_and_origin() {
    use crate::campaign::{execution, ledger::Job};
    use gomad_wasmhost::linked;
    use std::{
        io,
        sync::{
            Arc, Mutex,
            atomic::{AtomicUsize, Ordering},
        },
        time::Duration,
    };
    struct Refuse {
        calls: AtomicUsize,
        bytes: Mutex<Vec<u8>>,
    }
    impl linked::OutputConsumer for Refuse {
        fn write(&self, _stream: linked::OutputStream, bytes: &[u8]) -> io::Result<()> {
            self.calls.fetch_add(1, Ordering::Relaxed);
            self.bytes.lock().unwrap().extend_from_slice(bytes);
            Err(io::Error::new(
                io::ErrorKind::BrokenPipe,
                "consumer refused partial output",
            ))
        }
    }
    let module = br#"(module
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (memory (export "memory") 1) (data (i32.const 32) "prefix")
        (func (export "_start")
            (i32.store (i32.const 0) (i32.const 32))
            (i32.store (i32.const 4) (i32.const 6))
            (drop (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 16)))))"#.to_vec();
    let invocation = invocation();
    let prepared = prepared_module("partial-output", &invocation, module);
    let consumer = Arc::new(Refuse {
        calls: AtomicUsize::new(0),
        bytes: Mutex::new(Vec::new()),
    });
    let observed = execution::run_with_output(
        &prepared,
        &Job {
            ordinal: 0,
            seed: 7,
        },
        Duration::from_secs(1),
        &linked::Cancellation::default(),
        None,
        None,
        Some(consumer.clone()),
    )
    .unwrap();
    assert_eq!(
        consumer.calls.load(Ordering::Relaxed),
        1,
        "outcome={:?}",
        observed.execution.record.outcome
    );
    assert_eq!(*consumer.bytes.lock().unwrap(), b"prefix");
    assert_eq!(
        observed.execution.record.outcome.termination,
        "infrastructure"
    );
    assert_eq!(launcher::status(&observed.execution.record.outcome), 3);
    assert_eq!(observed.publication.payloads["output/stdout"], b"prefix");
    let root = std::env::temp_dir().join(format!(
        "gomad-callback-publication-{}",
        origin::identity().unwrap()
    ));
    let store = crate::evidence::store::Store {
        root: root.clone(),
        maximum_bytes: 64 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    };
    let published = store.publish(observed.publication).unwrap();
    let row = launcher::outcome_row(&observed.execution, &published).unwrap();
    assert_eq!(row["type"], "outcome");
    assert_eq!(row["status"], 3);
    assert_eq!(row["class"], "runner_failure");
    assert_eq!(row["termination"], "infrastructure");
    assert_eq!(row["record_sha256"], published.record_sha256);
    assert_eq!(row["seed"], 7);
    let opened = crate::evidence::store::Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    assert_eq!(opened.payload("output/stdout", 1024).unwrap(), b"prefix");
    assert_eq!(
        origin::authenticate(&opened).unwrap().unwrap().invocation,
        invocation
    );
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn first_failure_leaves_a_later_would_pass_seed_unexecuted() {
    use crate::campaign::{
        execution,
        ledger::{Event, Job, Journal},
    };
    use gomad_wasmhost::linked;
    use std::time::Duration;
    let module = br#"(module
        (import "wasi_snapshot_preview1" "environ_sizes_get" (func $sizes (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "environ_get" (func $env (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
        (memory (export "memory") 1)
        (func (export "_start")
            (drop (call $sizes (i32.const 64) (i32.const 68)))
            (drop (call $env (i32.const 80) (i32.const 128)))
            (if (i32.eq (i32.load8_u (i32.add (i32.load (i32.const 84)) (i32.const 10))) (i32.const 55))
                (then (call $exit (i32.const 17))))))"#.to_vec();
    let prepared = prepared_module("first-failure", &invocation(), module);
    let later = execution::run_with_replay(
        &prepared,
        &Job {
            ordinal: 1,
            seed: 8,
        },
        Duration::from_secs(1),
        &linked::Cancellation::default(),
        None,
        None,
    )
    .unwrap();
    assert_eq!(later.execution.record.outcome.exit_code, Some(0));
    let root = std::env::temp_dir().join(format!(
        "gomad-first-failure-{}",
        origin::identity().unwrap()
    ));
    let mut ignored = false;
    let mut options = super::super::explore::parse(
        &["--seeds=7,8".into(), "go-run".into(), "fixture".into()],
        &mut ignored,
    )
    .unwrap();
    options.artifacts = root.clone();
    options.parallel = 1;
    options.on_failure = "first".into();
    options.json = true;
    let outcome =
        launcher::run(&options, &prepared, Instant::now() + Duration::from_secs(5)).unwrap();
    assert_eq!(
        (outcome.status, outcome.attempted, outcome.unexecuted),
        (17, 1, 1)
    );
    let journal = Journal::open(&outcome.campaign, false).unwrap();
    let receipts = journal
        .events()
        .iter()
        .filter_map(|event| match event {
            Event::Receipt { execution } => Some(execution),
            _ => None,
        })
        .collect::<Vec<_>>();
    assert_eq!(receipts.len(), 1);
    assert_eq!((receipts[0].ordinal, receipts[0].seed), (0, 7));
    let mut output = Vec::new();
    let mut diagnostics = Vec::new();
    launcher::report(
        &outcome,
        true,
        Path::new("installation"),
        &mut output,
        &mut diagnostics,
    )
    .unwrap();
    let output = json_rows(&output);
    assert_eq!(output.len(), 2, "one attempted outcome and one summary");
    assert_eq!(output[0]["type"], "outcome");
    assert_eq!(
        (output[0]["ordinal"].as_u64(), output[0]["seed"].as_u64()),
        (Some(0), Some(7))
    );
    assert_eq!(output[0]["status"], 17);
    assert_eq!(output[0]["class"], "target_failure");
    assert_eq!(output[0]["termination"], "exit");
    assert_eq!(output[1]["type"], "result");
    assert_eq!(output[1]["unexecuted"], 1);
    assert_eq!(output[1]["status"], 17);
    assert!(diagnostics.ends_with(b"no replayable artifact was retained\n"));
    fs::remove_dir_all(root).unwrap();
}

fn json_rows(bytes: &[u8]) -> Vec<serde_json::Value> {
    bytes
        .split(|byte| *byte == b'\n')
        .filter(|line| !line.is_empty())
        .map(|line| serde_json::from_slice(line).unwrap())
        .collect()
}

#[test]
fn structured_all_pass_reports_both_authenticated_outcomes_then_summary() {
    use std::time::Duration;
    let module = br#"(module
        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
        (memory (export "memory") 1)
        (func (export "_start") (call $exit (i32.const 0))))"#
        .to_vec();
    let prepared = prepared_module("all-pass", &invocation(), module);
    let root = std::env::temp_dir().join(format!("gomad-all-pass-{}", origin::identity().unwrap()));
    let mut ignored = false;
    let mut options = super::super::explore::parse(
        &["--seeds=7,8".into(), "go-run".into(), "fixture".into()],
        &mut ignored,
    )
    .unwrap();
    options.artifacts = root.clone();
    options.parallel = 1;
    options.on_failure = "first".into();
    options.json = true;
    let outcome =
        launcher::run(&options, &prepared, Instant::now() + Duration::from_secs(5)).unwrap();
    assert_eq!(
        (outcome.status, outcome.attempted, outcome.unexecuted),
        (0, 2, 0),
        "{:?}",
        outcome.failure
    );
    let mut output = Vec::new();
    let mut diagnostics = Vec::new();
    launcher::report(
        &outcome,
        true,
        Path::new("installation"),
        &mut output,
        &mut diagnostics,
    )
    .unwrap();
    let rows = json_rows(&output);
    assert_eq!(rows.len(), 3, "two attempted outcomes and one summary");
    for (index, seed) in [7_u64, 8].iter().enumerate() {
        assert_eq!(rows[index]["type"], "outcome");
        assert_eq!(rows[index]["ordinal"], index);
        assert_eq!(rows[index]["seed"], *seed);
        assert_eq!(rows[index]["status"], 0);
        assert_eq!(rows[index]["class"], "success");
        let path = Path::new(rows[index]["artifact_path"].as_str().unwrap());
        let opened = crate::evidence::store::Opened::open(path).unwrap();
        opened.authenticate_all().unwrap();
        assert_eq!(
            rows[index]["record_sha256"],
            opened.manifest().record_sha256
        );
        assert_eq!(
            rows[index]["module_sha256"],
            opened.manifest().record.identity.target
        );
        assert_eq!(
            rows[index]["execution_identity"],
            opened.manifest().record.execution_identity
        );
        assert_eq!(
            rows[index]["origin_sha256"],
            opened.manifest().record.streams[origin::FILE].sha256
        );
    }
    assert_eq!(rows[2]["type"], "result");
    assert_eq!(rows[2]["attempted"], 2);
    assert_eq!(rows[2]["unexecuted"], 0);
    assert_eq!(diagnostics.as_slice(), b"");
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn qualification_source_probe_reaches_target_refusal_after_valid_digest() {
    let value = json!({
        "Manifest":{"Provisional":true,"IntegratedSourceSHA256":format!("sha256:{}", "0".repeat(64))},
        "Campaign":{"Target":{"Backend":"wasm","Kind":"exec","Source":"/sealed/intercepted-artifact"}},
        "ArtifactRoot":"/unused/qualification-result"
    });
    let spec = crate::qualify::spec::strict::<crate::qualify::spec::Spec>(
        &canonical::encode(&value).unwrap(),
    )
    .unwrap();
    assert_eq!(
        crate::qualify::spec::declared_digest(&spec.manifest.integrated_source_sha256).unwrap(),
        Some("0".repeat(64))
    );
    let error = crate::qualify::cells::freeze(
        &spec,
        Some(Path::new("/must-not-resolve-installation")),
        vec!["qualify".into()],
        None,
    )
    .err()
    .expect("prepared artifact source unexpectedly admitted");
    assert_eq!(error, "unsupported WASM source preparation request");
}
