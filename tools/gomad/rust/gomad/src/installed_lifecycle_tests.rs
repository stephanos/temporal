use crate::evidence::{
    canonical,
    identity::Identity,
    record::{Limits, Outcome, Record, Stream},
    store::{Publication, Published, Store},
};
use gomad_wasmhost::linked;
use serde_json::json;
use std::{collections::BTreeMap, path::PathBuf};

pub(crate) struct Retained {
    pub root: PathBuf,
    pub published: Published,
    pub record: Record,
}

impl Drop for Retained {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}

pub(crate) fn retained(cluster: bool) -> Retained {
    let root = std::env::temp_dir().join(format!(
        "gomad installed ' lifecycle-{}",
        crate::campaign::origin::identity().unwrap()
    ));
    std::fs::create_dir(&root).unwrap();
    let module = b"\0asm\x01\0\0\0".to_vec();
    let limits = Limits {
        fuel: 1_000_000,
        memory_bytes: 64 << 20,
        wall_millis: 1000,
        output_bytes: 1 << 20,
        evidence_bytes: 16 << 20,
    };
    let configuration = crate::replay::preflight::Configuration {
        engine: linked::Conditions::compiled().configuration(),
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        limits: limits.clone(),
        wall_nanos: None,
        arguments: vec![],
        environment: BTreeMap::new(),
        derive_seed_environment: false,
        network: None,
        runtime_sha256: None,
        model: None,
        namespace: None,
        storage: None,
    };
    let identity = Identity {
        target: canonical::digest(&module),
        toolchain: "1".repeat(64),
        platform: crate::replay::preflight::platform(),
        implementation: crate::replay::preflight::implementation().unwrap(),
        configuration: canonical::digest(&canonical::encode(&configuration).unwrap()),
    };
    let mut payloads = payloads(cluster, module, configuration, &identity);
    let mut record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest().unwrap(),
        identity,
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        seed: 7,
        arguments: vec![],
        environment: BTreeMap::new(),
        limits,
        virtual_time_nanos: 42,
        outcome: Outcome {
            class: if cluster { "runner_failure" } else { "success" }.into(),
            termination: if cluster { "capacity" } else { "exit" }.into(),
            exit_code: if cluster { None } else { Some(0) },
            reason: if cluster {
                "WASI capacity: payload-bytes limit=4096 used=4096 delta=64"
            } else {
                ""
            }
            .into(),
        },
        streams: BTreeMap::new(),
    };
    for (name, raw) in &payloads {
        if name != "target/module.wasm" {
            record.streams.insert(
                name.clone(),
                Stream {
                    bytes: raw.len() as u64,
                    sha256: canonical::digest(raw),
                    records: 1,
                },
            );
        }
    }
    payloads.insert("record.json".into(), record.bytes().unwrap());
    let store = Store {
        root: root.join("store"),
        maximum_bytes: 64 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    };
    let published = store
        .publish(Publication {
            record: record.clone(),
            payloads,
        })
        .unwrap();
    Retained {
        root,
        published,
        record,
    }
}

fn payloads(
    cluster: bool,
    module: Vec<u8>,
    configuration: crate::replay::preflight::Configuration,
    identity: &Identity,
) -> BTreeMap<String, Vec<u8>> {
    let mut payloads: BTreeMap<String, Vec<u8>> = BTreeMap::from([
        ("target/module.wasm".into(), module),
        ("output/stdout".into(), b"ping round 2\n".to_vec()),
        ("output/stderr".into(), b"echo server listening\n".to_vec()),
    ]);
    if cluster {
        let stdout = b"ping round 2\n".to_vec();
        let stderr = b"echo server listening\n".to_vec();
        let descriptor =
            |bytes: &[u8]| json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)});
        payloads.insert("evidence/cluster/a/1/stdout".into(), stdout.clone());
        payloads.insert("evidence/cluster/a/1/stderr".into(), stderr.clone());
        payloads.insert(
            "evidence/cluster_summary".into(),
            canonical::encode(&json!({
                "schema":"gomad.cluster-observation/v1","virtual_elapsed_nanos":42,
                "incarnations":[{"handle":{"node":"a","incarnation":1},
                    "termination":"capacity","exit_code":null,"reason":"payload-bytes",
                    "streams":{"stdout":descriptor(&stdout),"stderr":descriptor(&stderr)}}]
            }))
            .unwrap(),
        );
    } else {
        let inputs = crate::replay::preflight::Inputs {
            schema: "gomad.replay-inputs/v1".into(),
            helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
            host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
            toolchain_sha256: identity.toolchain.clone(),
            runtime_implementation_sha256: None,
            configuration,
        };
        payloads.insert(
            "evidence/replay.json".into(),
            canonical::encode(&inputs).unwrap(),
        );
    }
    payloads
}

#[test]
fn installed_lifecycle_inspect_capacity_displays_authenticated_instance_tail_and_bound() {
    let fixture = retained(true);
    let arguments = vec![fixture.published.path.to_str().unwrap().to_owned()];
    let mut output = Vec::new();
    crate::cli::inspect::run(&arguments, &mut output)
        .unwrap_or_else(|error| panic!("{}", error.message));
    let text = String::from_utf8(output).unwrap();
    assert!(
        text.contains("ping round 2"),
        "missing retained instance tail: {text}"
    );
    assert!(text.contains("exhausted=payload-bytes limit=4096 used=4096 delta=64"));
    assert!(text.contains("cluster-instance: node=a incarnation=1"));
    let path = fixture.published.path.join("evidence/cluster/a/1/stdout");
    std::fs::write(path, b"forged tail").unwrap();
    assert!(crate::cli::inspect::run(&arguments, &mut Vec::new()).is_err());
}

#[test]
fn installed_lifecycle_replay_explicit_missing_installation_fails_before_execution() {
    let fixture = retained(false);
    let arguments = vec![
        "--verify-only".into(),
        fixture.published.path.to_str().unwrap().into(),
    ];
    let error = crate::cli::replay::run_with_installation(
        &arguments,
        &mut Vec::new(),
        Some(&fixture.root.join("absent-installation")),
    )
    .expect_err("explicit missing installation was ignored");
    assert_eq!(error.status, 3);
    assert!(error.message.contains("runner_failure:"));
    let wrong = vec![
        "--unknown".into(),
        fixture.published.path.to_str().unwrap().into(),
    ];
    let error = crate::cli::replay::run_with_installation(
        &wrong,
        &mut Vec::new(),
        Some(&fixture.root.join("absent-installation")),
    )
    .err()
    .unwrap();
    assert_eq!(error.status, 2);
}

#[test]
fn installed_lifecycle_resume_missing_installation_keeps_committed_journal_unchanged() {
    let fixture = retained(false);
    let path = fixture.root.join("campaign");
    let plan = json!({"schema":"gomad.seed-campaign-plan/v2","selection":"7", "selection_count":1,
        "parallel":1,"on_failure":"all","failure_budget":1,"journal_bytes":1_048_576});
    drop(crate::campaign::ledger::Journal::create(&path, &plan, 1 << 20).unwrap());
    let before = std::fs::read(path.join("journal.jsonl")).unwrap();
    let arguments = vec!["--json".into(), path.to_str().unwrap().into()];
    let mut output = Vec::new();
    let status = crate::cli::resume::run_with_installation(
        &arguments,
        &mut output,
        Some(&fixture.root.join("absent-installation")),
    )
    .unwrap_or_else(|error| panic!("{}", error.message));
    assert_eq!(status, 3);
    let event: serde_json::Value = serde_json::from_slice(&output).unwrap();
    assert_eq!(event["classification"], "runner_failure");
    assert_eq!(std::fs::read(path.join("journal.jsonl")).unwrap(), before);
}

fn installation(fixture: &Retained, cooperative: bool) -> PathBuf {
    let path = fixture.root.join(if cooperative {
        "cooperative.json"
    } else {
        "stock.json"
    });
    let value = json!({"schema":"gomad.rust-wasm-installation/v1",
        "compiler_path":fixture.root.join("unavailable-go/bin/go"),"compiler_sha256":"1".repeat(64),"go_version":"go1.27.1",
        "runtime_root":if cooperative {fixture.root.join("unavailable-runtime")} else {PathBuf::new()},
        "cache_root":fixture.root.join("unavailable-cache"),"profile":if cooperative {"gomad.wasi-cooperative/v1"} else {"gomad.wasi-stock-exploratory/v1"},
        "fuel":1_000_000,"memory_bytes":67_108_864,"helper_sha256":linked::IMPLEMENTATION_SHA256,
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "runtime_implementation_sha256":if cooperative {Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"))} else {None},
        "model":{},"namespace":{"entries":[],"writable_directories":[],"preopens":[],"working_directory":[],"stdin":[],
            "limits":{"files":16,"descriptors":16,"filesystem_bytes":1_048_576,"output_bytes":1_048_576},"epoch_nanos":0},
        "storage":null,"network":null,"build_tags":[]});
    std::fs::write(&path, canonical::encode(&value).unwrap()).unwrap();
    path
}

#[test]
fn installed_lifecycle_replay_selected_context_works_without_compiler_and_refuses_other_profile() {
    let fixture = retained(false);
    let stock = installation(&fixture, false);
    let arguments = vec![
        "--verify-only".into(),
        fixture.published.path.to_str().unwrap().into(),
    ];
    let mut output = Vec::new();
    assert_eq!(
        crate::cli::replay::run_with_installation(&arguments, &mut output, Some(&stock))
            .unwrap_or_else(|error| panic!("{}", error.message)),
        0
    );
    assert!(String::from_utf8(output).unwrap().contains("verified"));
    assert!(!fixture.root.join("unavailable-go").exists());
    assert!(!fixture.root.join("unavailable-runtime").exists());
    let cooperative = installation(&fixture, true);
    let error =
        crate::cli::replay::run_with_installation(&arguments, &mut Vec::new(), Some(&cooperative))
            .expect_err("a self-consistent different profile was accepted");
    assert_eq!(error.status, 3);
    assert!(error.message.contains("retained implementation context"));
    let raw = std::fs::read(&stock).unwrap();
    let mut changed: serde_json::Value = serde_json::from_slice(&raw).unwrap();
    changed["helper_sha256"] = "0".repeat(64).into();
    std::fs::write(&stock, canonical::encode(&changed).unwrap()).unwrap();
    let error =
        crate::cli::replay::run_with_installation(&arguments, &mut Vec::new(), Some(&stock))
            .err()
            .unwrap();
    assert_eq!(error.status, 3);
}
