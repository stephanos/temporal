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
    prepared_module_with_wall(label, invocation, module, 1000)
}

fn prepared_module_with_wall(
    label: &str,
    invocation: &origin::Invocation,
    module: Vec<u8>,
    wall_millis: u64,
) -> Prepared {
    let configuration = Configuration {
        engine: gomad_wasmhost::linked::Conditions::compiled().configuration(),
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        limits: crate::evidence::record::Limits {
            fuel: 1000,
            memory_bytes: 1 << 20,
            wall_millis,
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
        compile_cache_root: None,
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
    let mut launch_ids = std::collections::BTreeSet::new();
    for (index, (path, receipt, digest)) in receipts.iter().enumerate() {
        let (_, prepared) = preparation::open_receipt(path, digest).unwrap();
        let bound = origin::from_preparation(&prepared).unwrap().unwrap();
        assert_eq!(bound.invocation, invocation);
        assert!(launch_ids.insert(bound.launch.as_ref().unwrap().invocation_id.clone()));
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

#[test]
fn retained_launch_renewal_preserves_driver_and_authenticated_preparation() {
    let invocation = invocation();
    let mut prepared = prepared("renewed", &invocation);
    let original = origin::from_preparation(&prepared).unwrap().unwrap();
    let receipt_sha256 = "6".repeat(64);
    origin::renew_launch(&mut prepared, receipt_sha256.clone()).unwrap();
    let renewed = origin::from_preparation(&prepared).unwrap().unwrap();
    assert_eq!(renewed.invocation, original.invocation);
    assert_eq!(renewed.module_sha256, original.module_sha256);
    assert_eq!(renewed.preparation_sha256, original.preparation_sha256);
    assert_ne!(renewed.launch, original.launch);
    assert_eq!(
        renewed.launch.as_ref().unwrap().preparation_receipt_sha256,
        Some(receipt_sha256)
    );
    let unchanged = prepared.provenance.clone();
    assert!(origin::renew_launch(&mut prepared, "not-a-digest".into()).is_err());
    assert_eq!(prepared.provenance, unchanged);
    for field in [
        "invocation_id",
        "process_id",
        "argv",
        "preparation_receipt_sha256",
    ] {
        let mut value = serde_json::to_value(&renewed).unwrap();
        value["launch"][field] = match field {
            "invocation_id" => json!(invocation.id),
            "process_id" => json!(0),
            "argv" => json!(["not hexadecimal"]),
            "preparation_receipt_sha256" => json!("not-a-digest"),
            _ => unreachable!(),
        };
        let invalid: origin::Intercepted = serde_json::from_value(value).unwrap();
        assert!(invalid.validate().is_err(), "{field}");
    }
}

#[test]
fn same_driver_launches_have_distinct_campaigns_and_record_commitments() {
    use crate::campaign::ledger::{Event, Journal};
    use std::{collections::BTreeSet, time::Duration};
    let root = std::env::temp_dir().join(format!(
        "gomad-shared-driver-{}",
        origin::identity().unwrap()
    ));
    let invocation = invocation();
    let run = || {
        let module = br#"(module
            (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
            (memory (export "memory") 1)
            (func (export "_start") (call $exit (i32.const 0))))"#
            .to_vec();
        let prepared = prepared_module("same-package", &invocation, module);
        let binding = origin::from_preparation(&prepared).unwrap().unwrap();
        let mut ignored = false;
        let mut options = super::super::explore::parse(
            &["--seeds=7".into(), "go-run".into(), "fixture".into()],
            &mut ignored,
        )
        .unwrap();
        options.artifacts = root.clone();
        options.json = true;
        options.parallel = 1;
        options.on_failure = "first".into();
        let prepared = retain_admission(&options, prepared).unwrap();
        let outcome =
            launcher::run(&options, &prepared, Instant::now() + Duration::from_secs(5)).unwrap();
        assert_eq!(
            (outcome.status, outcome.attempted, outcome.unexecuted),
            (0, 1, 0)
        );
        let journal = Journal::open(&outcome.campaign, false).unwrap();
        let receipt = journal
            .events()
            .iter()
            .find_map(|event| match event {
                Event::Receipt { execution } => Some(execution),
                _ => None,
            })
            .unwrap();
        let opened = crate::evidence::store::Opened::open(
            &outcome.campaign.join(receipt.artifact.as_ref().unwrap()),
        )
        .unwrap();
        opened.authenticate_all().unwrap();
        assert_eq!(origin::authenticate(&opened).unwrap().unwrap(), binding);
        let launch_id = binding.launch.unwrap().invocation_id;
        assert_eq!(
            outcome.campaign.file_name().unwrap(),
            OsStr::new(&format!("campaign-{launch_id}"))
        );
        (
            outcome.campaign,
            launch_id,
            opened.manifest().record_sha256.clone(),
        )
    };
    let first = run();
    let second = run();
    let concurrent = std::thread::scope(|scope| {
        let a = scope.spawn(run);
        let b = scope.spawn(run);
        [a.join().unwrap(), b.join().unwrap()]
    });
    let [third, fourth] = concurrent;
    let observations = [first, second, third, fourth];
    assert_eq!(
        observations
            .iter()
            .map(|value| &value.0)
            .collect::<BTreeSet<_>>()
            .len(),
        4
    );
    let launch_ids = observations
        .iter()
        .map(|value| value.1.clone())
        .collect::<BTreeSet<_>>();
    assert_eq!(launch_ids.len(), 4);
    assert_eq!(
        observations
            .iter()
            .map(|value| &value.2)
            .collect::<BTreeSet<_>>()
            .len(),
        4
    );
    authenticate_launch_namespaces(&root, &invocation, &launch_ids);
    fs::remove_dir_all(root).unwrap();
}

fn authenticate_launch_namespaces(
    root: &Path,
    invocation: &origin::Invocation,
    expected: &std::collections::BTreeSet<String>,
) {
    use crate::campaign::ledger::Journal;
    use std::collections::BTreeSet;
    let attempts = fs::read_dir(root.join("admissions").join(&invocation.id)).unwrap();
    let mut admitted = BTreeSet::new();
    let mut campaigns = BTreeSet::new();
    for attempt in attempts {
        let path = attempt.unwrap().path().join("plan.json.preparation.json");
        let digest = canonical::digest(&fs::read(&path).unwrap());
        let (receipt, prepared) = preparation::open_receipt(&path, &digest).unwrap();
        let bound = origin::from_preparation(&prepared).unwrap().unwrap();
        assert_eq!(&bound.invocation, invocation);
        let launch_id = &bound.launch.as_ref().unwrap().invocation_id;
        assert!(admitted.insert(launch_id.clone()));
        assert!(expected.contains(launch_id));
        let preparation = root
            .join("campaigns")
            .join(format!("campaign-{launch_id}-prepared"));
        let execution = root.join("campaigns").join(format!("campaign-{launch_id}"));
        assert_eq!(receipt.campaign, preparation);
        assert_eq!(receipt.guest_executions, 0);
        for (path, is_prepared) in [(preparation, true), (execution, false)] {
            let journal = Journal::open(&path, false).unwrap();
            assert_eq!(
                journal.plan()["origin"],
                serde_json::to_value(&bound).unwrap()
            );
            assert_eq!(journal.events().is_empty(), is_prepared);
            assert!(campaigns.insert(path));
        }
    }
    assert_eq!(&admitted, expected);
    assert_eq!(campaigns.len(), 8);
}

#[test]
fn legacy_intercepted_artifact_remains_readable_without_launch_extension() {
    use crate::campaign::{execution, ledger::Job};
    use gomad_wasmhost::linked;
    use std::time::Duration;
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let mut prepared = prepared_module("legacy", &invocation(), module);
    let mut provenance: serde_json::Value = serde_json::from_slice(&prepared.provenance).unwrap();
    provenance["origin"]
        .as_object_mut()
        .unwrap()
        .remove("launch");
    crate::campaign::prepare::typed_provenance(&provenance).unwrap();
    prepared.provenance = canonical::encode(&provenance).unwrap();
    assert!(
        origin::from_preparation(&prepared)
            .unwrap()
            .unwrap()
            .launch
            .is_none()
    );
    let execution = execution::run_with_replay(
        &prepared,
        &Job {
            ordinal: 0,
            seed: 7,
        },
        Duration::from_secs(1),
        &linked::Cancellation::default(),
        None,
        None,
    )
    .unwrap();
    let root = std::env::temp_dir().join(format!(
        "gomad-legacy-origin-{}",
        origin::identity().unwrap()
    ));
    let store = crate::evidence::store::Store {
        root: root.clone(),
        maximum_bytes: 64 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    };
    let published = store.publish(execution.publication).unwrap();
    let opened = crate::evidence::store::Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    assert!(
        origin::authenticate(&opened)
            .unwrap()
            .unwrap()
            .launch
            .is_none()
    );
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn detached_pooled_artifact_authenticates_after_original_store_is_removed() {
    use crate::campaign::{execution, ledger::Job};
    use gomad_wasmhost::linked;
    use std::{os::unix::fs::MetadataExt as _, time::Duration};
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let prepared = prepared_module("detached", &invocation(), module);
    let observed = execution::run_with_replay(
        &prepared,
        &Job {
            ordinal: 0,
            seed: 7,
        },
        Duration::from_secs(1),
        &linked::Cancellation::default(),
        None,
        None,
    )
    .unwrap();
    let root = std::env::temp_dir().join(format!(
        "gomad-detached-pool-{}",
        origin::identity().unwrap()
    ));
    let store = crate::evidence::store::Store {
        root: root.join("original"),
        maximum_bytes: 64 << 20,
        target_pool: Some(root.join("shared")),
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    };
    let published = store.publish(observed.publication).unwrap();
    assert!(
        fs::metadata(published.path.join("target/module.wasm"))
            .unwrap()
            .nlink()
            > 1
    );
    let opened = crate::evidence::store::Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    let detached = root.join("detached");
    for name in opened
        .manifest()
        .files
        .keys()
        .map(String::as_str)
        .chain(["manifest.json", "storage.json"])
    {
        let destination = detached.join(name);
        crate::evidence::store::durable_directory(destination.parent().unwrap(), |_| Ok(()))
            .unwrap();
        fs::copy(published.path.join(name), &destination).unwrap();
        fs::set_permissions(&destination, fs::Permissions::from_mode(0o600)).unwrap();
    }
    let original_record = opened.manifest().record_sha256.clone();
    let original_origin = origin::authenticate(&opened).unwrap();
    drop(opened);
    fs::remove_dir_all(&store.root).unwrap();
    fs::remove_dir_all(root.join("shared")).unwrap();
    assert_eq!(
        fs::metadata(detached.join("target/module.wasm"))
            .unwrap()
            .nlink(),
        1
    );
    let copied = crate::evidence::store::Opened::open(&detached).unwrap();
    copied.authenticate_all().unwrap();
    assert_eq!(copied.manifest().record_sha256, original_record);
    assert_eq!(origin::authenticate(&copied).unwrap(), original_origin);
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn poisoned_selected_cache_never_publishes_launcher_records_in_either_root() {
    use crate::campaign::{
        execution, launcher,
        ledger::{Event, Job, Journal},
    };
    use gomad_wasmhost::linked;
    use std::time::Duration;
    let root = std::env::temp_dir().join(format!(
        "gomad-cache-launcher-{}",
        origin::identity().unwrap()
    ));
    fs::create_dir(&root).unwrap();
    fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
    for name in ["first", "second"] {
        let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
        let mut prepared = prepared_module_with_wall(name, &invocation(), module, 10_000);
        let cache = root.join(name);
        prepared.compile_cache_root = Some(cache.to_str().unwrap().into());
        execution::run(
            &prepared,
            &Job {
                ordinal: 0,
                seed: 1,
            },
            Duration::from_secs(10),
            &linked::Cancellation::default(),
        )
        .unwrap();
        let entries: Vec<_> = fs::read_dir(&cache)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|entry| {
                entry
                    .extension()
                    .is_some_and(|extension| extension == "gomad-cache-v3")
            })
            .collect();
        assert_eq!(entries.len(), 1);
        let mut bytes = fs::read(&entries[0]).unwrap();
        *bytes.last_mut().unwrap() ^= 1;
        fs::write(&entries[0], bytes).unwrap();
        let mut ignored = false;
        let mut options =
            super::super::explore::parse(&["go-run".into(), "fixture".into()], &mut ignored)
                .unwrap();
        options.artifacts = root.join(format!("artifacts-{name}"));
        options.seeds = "1".into();
        options.execution_timeout = Duration::from_secs(10);
        options.json = true;
        let error = launcher::run(
            &options,
            &prepared,
            Instant::now() + Duration::from_secs(30),
        )
        .err()
        .expect("cache failure produced a launcher publication");
        assert!(error.contains("compile cache: corrupt entry"), "{error}");
        assert!(error.contains(cache.to_str().unwrap()));
        let launch = origin::from_preparation(&prepared)
            .unwrap()
            .unwrap()
            .launch
            .unwrap();
        let campaign = options
            .artifacts
            .join("campaigns")
            .join(format!("campaign-{}", launch.invocation_id));
        let journal = Journal::open(&campaign, false).unwrap();
        assert!(
            journal
                .events()
                .iter()
                .all(|event| matches!(event, Event::Admit { .. }))
        );
        let artifact = journal
            .open_artifact(journal.plan()["prepared_artifact"].as_str().unwrap())
            .unwrap();
        let record = artifact.manifest().record.bytes().unwrap();
        assert!(
            !String::from_utf8(record)
                .unwrap()
                .contains(cache.to_str().unwrap())
        );
    }
    fs::remove_dir_all(root).unwrap();
}

fn cache_installation(root: &Path, cache: &Path) -> PathBuf {
    let profile = "gomad.wasi-stock-exploratory/v1";
    let configured = crate::install::config::build(profile, None, None).unwrap();
    let compiler = root.join("unavailable-go/bin/go");
    let compiler_inputs = BTreeMap::from([(compiler.clone(), "3".repeat(64))]);
    let value = json!({
        "schema":"gomad.rust-wasm-installation/v1", "compiler_path":compiler,
        "compiler_sha256":"3".repeat(64), "go_version":"go1.27.1", "runtime_root":"",
        "cache_root":root.join("unavailable-preparation"), "profile":profile,
        "fuel":1000, "memory_bytes":1<<20, "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"), "runtime_implementation_sha256":null,
        "model":configured.model, "namespace":configured.namespace, "storage":configured.storage,
        "network":configured.network, "build_tags":[],
        "admission":{
            "compiler_inputs":compiler_inputs, "inputs_sha256":canonical::digest(&canonical::encode(&compiler_inputs).unwrap()),
            "guest_configuration":configured.original, "configuration_sha256":canonical::digest(&canonical::encode(&configured.original).unwrap()),
            "host_executable_sha256":canonical::digest(&fs::read(std::env::current_exe().unwrap()).unwrap()),
            "explicit_helper":null, "environment":configured.environment, "arguments":configured.arguments,
            "compile_cache_root":cache
        }
    });
    let path = root.join("installation.json");
    fs::write(&path, canonical::encode(&value).unwrap()).unwrap();
    path
}

#[test]
fn receipt_admission_restores_authenticated_selected_cache_and_detached_receipt_does_not() {
    let root = std::env::temp_dir().join(format!(
        "gomad-receipt-cache-{}",
        origin::identity().unwrap()
    ));
    fs::create_dir(&root).unwrap();
    fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
    let cache = root.join("engine-cache");
    let installation = cache_installation(&root, &cache);
    let selected = crate::install::manifest::resolve(Some(&installation)).unwrap();
    let mut invocation = invocation();
    invocation.installation_sha256 = selected.sha256;
    let module = br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec();
    let prepared = prepared_module_with_wall("receipt", &invocation, module.clone(), 10_000);
    let mut ignored = false;
    let mut options =
        super::super::explore::parse(&["go-run".into(), "fixture".into()], &mut ignored).unwrap();
    options.artifacts = root.join("original");
    let retained = preparation::retain_admitted(&options, prepared).unwrap();
    let (receipt, _, digest) = retained
        .publish(&root.join("plan.json"), vec!["run-guest".into()])
        .unwrap();
    assert!(
        preparation::open_receipt(&receipt, &digest)
            .unwrap()
            .1
            .compile_cache_root
            .is_none()
    );
    let arguments = Arguments {
        module: root.join("retained.wasm"),
        argv: Vec::new(),
        seeds: Some("1".into()),
        artifacts: Some(root.join("admitted")),
        json: true,
    };
    let mut deadline = Deadline::new();
    let admitted = admit_receipt(
        &arguments,
        Some(&installation),
        &module,
        receipt.as_os_str(),
        &digest,
        &mut deadline,
    )
    .unwrap();
    assert_eq!(
        admitted.prepared.compile_cache_root.as_deref(),
        cache.to_str()
    );
    assert!(
        preparation::open_receipt(&receipt, &digest)
            .unwrap()
            .1
            .compile_cache_root
            .is_none()
    );
    let mut changed = arguments;
    changed.argv.push("changed".into());
    assert!(
        admit_receipt(
            &changed,
            Some(&installation),
            &module,
            receipt.as_os_str(),
            &digest,
            &mut deadline
        )
        .is_err()
    );
    fs::remove_dir_all(root).unwrap();
}
