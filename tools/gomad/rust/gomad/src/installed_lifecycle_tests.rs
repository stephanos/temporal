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
    retained_module(cluster, binary_module(0), 1000)
}

fn binary_module(tables: u8) -> Vec<u8> {
    let mut module = b"\0asm\x01\0\0\0".to_vec();
    module.extend([1, 4, 1, 0x60, 0, 0, 3, 2, 1, 0]);
    if tables != 0 {
        module.extend([4, 1 + tables * 3, tables]);
        for _ in 0..tables {
            module.extend([0x70, 0, 1]);
        }
    }
    module.extend([5, 3, 1, 0, 1, 7, 19, 2, 6]);
    module.extend(b"memory");
    module.extend([2, 0, 6]);
    module.extend(b"_start");
    module.extend([0, 0, 10, 4, 1, 2, 0, 0x0b]);
    module
}

fn retained_module(cluster: bool, module: Vec<u8>, wall_millis: u64) -> Retained {
    let root = std::env::temp_dir().join(format!(
        "gomad installed ' lifecycle-{}",
        crate::campaign::origin::identity().unwrap()
    ));
    std::fs::create_dir(&root).unwrap();
    let limits = Limits {
        fuel: 1_000_000,
        memory_bytes: 64 << 20,
        wall_millis,
        output_bytes: 1 << 20,
        evidence_bytes: 16 << 20,
    };
    let configuration = crate::replay::preflight::Configuration {
        engine: linked::Conditions::compiled().configuration(),
        profile: "gomad.wasi-stock-exploratory/v1".into(),
        limits: limits.clone(),
        wall_nanos: None,
        world_transition_bytes: None,
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
                    "interrupted":true,"offered_output_bytes":stdout.len()+stderr.len(),
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
fn installed_lifecycle_corrupt_retained_cluster_is_status_three_before_execution() {
    let fixture = retained(true);
    let path = fixture.published.path.join("evidence/cluster_summary");
    let mut bytes = std::fs::read(&path).unwrap();
    *bytes.last_mut().unwrap() ^= 1;
    std::fs::write(path, bytes).unwrap();
    let error = crate::cli::replay::run(
        &[fixture.published.path.to_str().unwrap().into()],
        &mut Vec::new(),
    )
    .expect_err("corrupt retained cluster was admitted");
    assert_eq!(error.status, 3, "{}", error.message);
}

#[test]
fn installed_lifecycle_foreign_storage_lineage_remains_status_two() {
    let fixture = retained(false);
    std::fs::write(
        fixture.published.path.join("storage.json"),
        br#"{"format":"foreign/v1"}"#,
    )
    .unwrap();
    let error = crate::cli::replay::run(
        &[fixture.published.path.to_str().unwrap().into()],
        &mut Vec::new(),
    )
    .expect_err("foreign storage lineage was admitted");
    assert_eq!(error.status, 2, "{}", error.message);
    assert!(
        error
            .message
            .contains("unsupported artifact storage format")
    );
}

#[test]
fn installed_lifecycle_authenticated_foreign_replay_context_is_status_two() {
    let fixture = retained(false);
    let opened = crate::evidence::store::Opened::open(&fixture.published.path).unwrap();
    let mut payloads = BTreeMap::new();
    for (name, entry) in &opened.manifest().files {
        if name != "record.json" {
            payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
        }
    }
    let mut inputs: crate::replay::preflight::Inputs =
        serde_json::from_slice(&payloads[crate::replay::preflight::FILE]).unwrap();
    inputs.host_sha256 = "0".repeat(64);
    payloads.insert(
        crate::replay::preflight::FILE.into(),
        canonical::encode(&inputs).unwrap(),
    );
    let fixture = publish_cluster_payloads(fixture, payloads);
    let error = crate::cli::replay::run(
        &[fixture.published.path.to_str().unwrap().into()],
        &mut Vec::new(),
    )
    .expect_err("foreign implementation context was admitted");
    assert_eq!(error.status, 2, "{}", error.message);
    assert!(error.message.contains("incompatible replay artifact"));
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

#[test]
fn installed_lifecycle_selected_cache_is_used_without_old_build_inputs() {
    let fixture = retained_module(false, binary_module(0), 10_000);
    let path = installation(&fixture, false);
    let mut installed: crate::install::manifest::Installation =
        serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    let configured = crate::install::config::build(&installed.profile, None, None).unwrap();
    let compiler_inputs = BTreeMap::from([(
        installed.compiler_path.clone(),
        installed.compiler_sha256.clone(),
    )]);
    let cache = fixture.root.join("engine-cache");
    installed.admission = Some(crate::install::manifest::Admission {
        inputs_sha256: canonical::digest(&canonical::encode(&compiler_inputs).unwrap()),
        compiler_inputs,
        configuration_sha256: canonical::digest(&canonical::encode(&configured.original).unwrap()),
        guest_configuration: configured.original,
        host_executable_sha256: canonical::digest(
            &std::fs::read(std::env::current_exe().unwrap()).unwrap(),
        ),
        explicit_helper: None,
        environment: configured.environment,
        arguments: configured.arguments,
        compile_cache_root: Some(cache.clone()),
    });
    installed.model = configured.model;
    installed.namespace = configured.namespace;
    installed.storage = configured.storage;
    installed.network = configured.network;
    std::fs::write(&path, canonical::encode(&installed).unwrap()).unwrap();
    let arguments = vec![fixture.published.path.to_str().unwrap().to_owned()];
    assert_eq!(
        crate::cli::replay::run_with_installation(&arguments, &mut Vec::new(), Some(&path))
            .unwrap_or_else(|error| panic!("{}", error.message)),
        1
    );
    let entries: Vec<_> = std::fs::read_dir(&cache)
        .expect("selected installation cache was ignored")
        .map(|entry| entry.unwrap().path())
        .filter(|entry| {
            entry
                .extension()
                .is_some_and(|extension| extension == "gomad-cache-v3")
        })
        .collect();
    assert_eq!(entries.len(), 1);
    let entry = &entries[0];
    let mut bytes = std::fs::read(entry).unwrap();
    *bytes.last_mut().unwrap() ^= 1;
    std::fs::write(entry, bytes).unwrap();
    let error = crate::cli::replay::run_with_installation(&arguments, &mut Vec::new(), Some(&path))
        .expect_err("poisoned installed cache was ignored");
    assert_eq!(error.status, 3);
    assert!(
        error.message.contains("compile cache: corrupt entry"),
        "{}",
        error.message
    );
}

#[test]
fn installed_lifecycle_noncache_capacity_remains_a_replay_mismatch() {
    let module = binary_module(9);
    let fixture = retained_module(false, module, 1000);
    let arguments = vec![fixture.published.path.to_str().unwrap().to_owned()];
    let mut output = Vec::new();
    assert_eq!(
        crate::cli::replay::run(&arguments, &mut output)
            .unwrap_or_else(|error| panic!("{}", error.message)),
        1
    );
    assert!(
        String::from_utf8(output)
            .unwrap()
            .contains("reproduced=false")
    );
}

fn retained_cluster_streams(
    declared: &[&str],
    included: &[&str],
    identity_matches: bool,
) -> Retained {
    let fixture = retained(true);
    let opened = crate::evidence::store::Opened::open(&fixture.published.path).unwrap();
    let mut payloads = BTreeMap::new();
    for (name, entry) in &opened.manifest().files {
        if name != "record.json"
            && (!name.starts_with("evidence/cluster/a/1/")
                || included.contains(&name.rsplit('/').next().unwrap()))
        {
            payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
        }
    }
    let mut summary: serde_json::Value =
        serde_json::from_slice(&payloads["evidence/cluster_summary"]).unwrap();
    let streams = summary["incarnations"][0]["streams"]
        .as_object_mut()
        .unwrap();
    let stdout = streams["stdout"].clone();
    streams.retain(|name, _| declared.contains(&name.as_str()));
    for stream in declared {
        let descriptor = streams.entry(*stream).or_insert_with(|| stdout.clone());
        if !identity_matches {
            descriptor["sha256"] = json!("0".repeat(64));
        }
    }
    payloads.insert(
        "evidence/cluster_summary".into(),
        canonical::encode(&summary).unwrap(),
    );
    publish_cluster_payloads(fixture, payloads)
}

fn publish_cluster_payloads(
    mut fixture: Retained,
    mut payloads: BTreeMap<String, Vec<u8>>,
) -> Retained {
    fixture.record.streams.clear();
    for (name, raw) in &payloads {
        if name != "target/module.wasm" {
            fixture.record.streams.insert(
                name.clone(),
                Stream {
                    bytes: raw.len() as u64,
                    sha256: canonical::digest(raw),
                    records: 1,
                },
            );
        }
    }
    payloads.insert("record.json".into(), fixture.record.bytes().unwrap());
    fixture.published = Store {
        root: fixture.root.join("sparse-stream-store"),
        maximum_bytes: 64 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(Publication {
        record: fixture.record.clone(),
        payloads,
    })
    .unwrap();
    fixture
}

#[test]
fn installed_lifecycle_inspect_accepts_silent_and_single_stream_cluster_instances() {
    for streams in [&[][..], &["stdout"][..], &["stderr"][..]] {
        let fixture = retained_cluster_streams(streams, streams, true);
        let mut output = Vec::new();
        crate::cli::inspect::run(
            &[fixture.published.path.to_str().unwrap().to_owned()],
            &mut output,
        )
        .unwrap_or_else(|error| panic!("{}", error.message));
        let text = String::from_utf8(output).unwrap();
        let stdout = if streams.contains(&"stdout") { 13 } else { 0 };
        let stderr = if streams.contains(&"stderr") { 22 } else { 0 };
        assert!(
            text.contains(&format!("stdout-bytes={stdout} stderr-bytes={stderr}")),
            "{text}"
        );
    }
}

#[test]
fn installed_lifecycle_inspect_refuses_inconsistent_cluster_streams() {
    for (declared, included, matches) in [
        (&["stdout"][..], &[][..], true),
        (&[][..], &["stdout"][..], true),
        (&["stdout"][..], &["stdout"][..], false),
    ] {
        let fixture = retained_cluster_streams(declared, included, matches);
        assert!(
            crate::cli::inspect::run(
                &[fixture.published.path.to_str().unwrap().to_owned()],
                &mut Vec::new()
            )
            .is_err()
        );
    }
}

fn retained_projected_cluster(damage: &str) -> Retained {
    use gomad_wasmhost::linked_cluster as helper;
    let mut fixture = retained(true);
    let output_limit = usize::try_from(fixture.record.limits.output_bytes).unwrap();
    let evidence_limit = usize::try_from(fixture.record.limits.evidence_bytes).unwrap();
    let mut evidence: BTreeMap<String, Vec<u8>> = BTreeMap::from([
        (
            "stdout".into(),
            [vec![b'x'; 16_384], b"actual output tail".to_vec()].concat(),
        ),
        ("choice".into(), b"actual choice bytes".to_vec()),
        ("transcript".into(), vec![b't'; output_limit + 1]),
        ("network".into(), b"actual network bytes".to_vec()),
        ("storage".into(), b"actual storage bytes".to_vec()),
        (
            "storage_image".into(),
            b"actual storage image bytes".to_vec(),
        ),
    ]);
    if damage == "evidence-budget" {
        evidence.insert("transcript".into(), vec![b't'; evidence_limit / 2 + 1]);
        evidence.insert("network".into(), vec![b'n'; evidence_limit / 2 + 1]);
    } else if damage == "output-budget" {
        evidence.insert("stdout".into(), vec![b'x'; output_limit / 2 + 1]);
        evidence.insert("stderr".into(), vec![b'e'; output_limit / 2 + 1]);
    }
    let mut outcome = linked::Outcome::new("exit", "");
    outcome.exit_code = Some(0);
    let offered_output_bytes = evidence
        .iter()
        .filter(|(name, _)| matches!(name.as_str(), "stdout" | "stderr"))
        .map(|(_, raw)| raw.len() as u64)
        .sum();
    let mut incarnation_outcome = linked::Outcome::new("exit", "");
    incarnation_outcome.output_bytes = offered_output_bytes;
    let projected = crate::replay::cluster::project_encoded(
        helper::Execution {
            outcome,
            executed_steps: vec![],
            incarnations: vec![helper::IncarnationResult {
                handle: helper::Handle {
                    node: "a".into(),
                    incarnation: 1,
                },
                execution: linked::Execution {
                    authorization_to_terminal_nanos: None,
                    outcome: incarnation_outcome,
                    evidence,
                    event_order: None,
                },
                interrupted: false,
            }],
            world: vec![],
            network: vec![],
            inspection: vec![],
            virtual_elapsed_nanos: 42,
            event_order: None,
        },
        "",
    )
    .unwrap();
    let opened = crate::evidence::store::Opened::open(&fixture.published.path).unwrap();
    let mut payloads = BTreeMap::new();
    for (name, entry) in &opened.manifest().files {
        if name == "target/module.wasm" {
            payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
        }
    }
    for (name, raw) in projected.evidence {
        payloads.insert(format!("evidence/{name}"), raw);
    }
    damage_projected_payloads(&mut payloads, damage);
    fixture.record.outcome = Outcome {
        class: "success".into(),
        termination: "exit".into(),
        exit_code: Some(0),
        reason: String::new(),
    };
    publish_cluster_payloads(fixture, payloads)
}

fn damage_projected_payloads(payloads: &mut BTreeMap<String, Vec<u8>>, damage: &str) {
    if matches!(
        damage,
        "hash"
            | "size"
            | "descriptor"
            | "malformed"
            | "duplicate-instance"
            | "missing-instance"
            | "offered-underflow"
    ) {
        let mut summary: serde_json::Value =
            serde_json::from_slice(&payloads["evidence/cluster_summary"]).unwrap();
        let streams = summary["incarnations"][0]["streams"]
            .as_object_mut()
            .unwrap();
        match damage {
            "hash" => streams.get_mut("storage").unwrap()["sha256"] = json!("0".repeat(64)),
            "size" => streams.get_mut("storage").unwrap()["bytes"] = json!(1),
            "descriptor" => {
                streams.remove("network");
            }
            "malformed" => {
                streams.insert("../escape".into(), streams["choice"].clone());
            }
            "duplicate-instance" | "missing-instance" | "offered-underflow" => {}
            _ => unreachable!(),
        }
        if damage == "duplicate-instance" {
            let duplicate = summary["incarnations"][0].clone();
            summary["incarnations"]
                .as_array_mut()
                .unwrap()
                .push(duplicate);
        } else if damage == "missing-instance" {
            summary["incarnations"] = json!([]);
        } else if damage == "offered-underflow" {
            summary["incarnations"][0]["offered_output_bytes"] = json!(0);
        }
        payloads.insert(
            "evidence/cluster_summary".into(),
            canonical::encode(&summary).unwrap(),
        );
    } else if damage == "missing-summary" {
        payloads.remove("evidence/cluster_summary");
    } else if damage == "orphan-instance" {
        payloads.insert(
            "evidence/cluster/foreign/1/stdout".into(),
            b"orphan".to_vec(),
        );
    } else if damage == "payload" {
        payloads.remove("evidence/cluster/a/1/storage");
    }
}

#[test]
fn installed_lifecycle_inspect_accepts_actual_project_encoded_nonoutput_streams() {
    let fixture = retained_projected_cluster("");
    let opened = crate::evidence::store::Opened::open(&fixture.published.path).unwrap();
    let tail = opened
        .payload_tail(
            "evidence/cluster/a/1/stdout",
            fixture.record.limits.output_bytes,
            4096,
        )
        .unwrap();
    assert_eq!(tail.len(), 4096);
    assert!(tail.ends_with(b"actual output tail"));
    let mut output = Vec::new();
    crate::cli::inspect::run(
        &[fixture.published.path.to_str().unwrap().to_owned()],
        &mut output,
    )
    .unwrap_or_else(|error| panic!("{}", error.message));
    let text = String::from_utf8(output).unwrap();
    assert!(text.contains("stdout-bytes=16402 stderr-bytes=0"), "{text}");
    assert!(text.contains("actual output tail"));
    assert!(text.len() < 16_384);
    let mut output = Vec::new();
    crate::cli::inspect::run(
        &[
            "--json".into(),
            fixture.published.path.to_str().unwrap().into(),
        ],
        &mut output,
    )
    .unwrap_or_else(|error| panic!("{}", error.message));
    let report: serde_json::Value = serde_json::from_slice(&output).unwrap();
    assert_eq!(report["artifact"]["cluster"]["stdout_bytes"], 16402);
    assert_eq!(report["artifact"]["cluster"]["stderr_bytes"], 0);
    assert_eq!(report["artifact"]["cluster"]["offered_output_bytes"], 16402);
    assert_eq!(report["artifact"]["cluster"]["discarded_output_bytes"], 0);
    assert_eq!(report["artifact"]["cluster"]["truncated"], false);

    assert_eq!(
        report["artifact"]["cluster"]["instances"][0]["stdout_tail"],
        String::from_utf8_lossy(&tail).as_ref()
    );
}

#[test]
fn installed_lifecycle_inspect_authenticates_actual_project_encoded_catalogue_and_budgets() {
    for damage in [
        "hash",
        "size",
        "payload",
        "descriptor",
        "malformed",
        "evidence-budget",
        "output-budget",
        "duplicate-instance",
        "missing-instance",
        "orphan-instance",
        "missing-summary",
        "offered-underflow",
    ] {
        let fixture = retained_projected_cluster(damage);
        for json in [false, true] {
            let mut arguments = vec![fixture.published.path.to_str().unwrap().to_owned()];
            if json {
                arguments.insert(0, "--json".into());
            }
            let error = crate::cli::inspect::run(&arguments, &mut Vec::new())
                .expect_err("inconsistent producer evidence was accepted");
            if damage.ends_with("budget") {
                assert!(error.message.contains("budget"), "{}", error.message);
            }
        }
    }
}
