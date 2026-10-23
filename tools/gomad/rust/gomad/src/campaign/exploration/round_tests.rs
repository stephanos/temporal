use super::*;
use crate::{
    campaign::{
        exploration::{
            choice::{self, ChoiceResult},
            site_scope::Resolution,
        },
        prepare::Prepared,
    },
    evidence::{
        inspection::Details,
        search_context,
        store::{Publication, Store},
    },
};
use serde_json::json;
use sha2::Digest as _;

fn prepared() -> (Prepared, Publication) {
    let mut publication = search_context::tests::publication(0);
    let module = std::fs::read(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../testdata/parity/symbolize/guest.wasm"),
    )
    .unwrap();
    let mut details: Details =
        serde_json::from_slice(&publication.payloads["evidence/inspection.json"]).unwrap();
    let identity = &mut details.choices.as_mut().unwrap().identity;
    identity.target_sha256 = sha2::Sha256::digest(&module).into();
    let identity = identity.clone();
    details.choices.as_mut().unwrap().tape_sha256 = Some(
        gomad_wasmhost::linked::inspect_choices(&[], &identity)
            .unwrap()
            .tape_sha256,
    );
    let configuration = json!({"engine":"fixture","profile":"gomad.wasi-cooperative/v1", "limits": publication.record.limits,
        "arguments":[],"environment":{},"runtime_sha256":choice::hex(&identity.implementation_sha256),"model":null,"namespace":null});
    publication.record.identity.target = canonical::digest(&module);
    let configuration: crate::replay::preflight::Configuration =
        serde_json::from_value(configuration).unwrap();
    publication.record.identity.configuration =
        canonical::digest(&canonical::encode(&configuration).unwrap());
    publication.record.identity.platform = crate::replay::preflight::platform();
    publication.record.identity.implementation =
        crate::replay::preflight::implementation().unwrap();
    publication.record.execution_identity = publication.record.identity.digest().unwrap();
    let bytes = canonical::encode(&details).unwrap();
    let stream = publication
        .record
        .streams
        .get_mut("evidence/inspection.json")
        .unwrap();
    stream.bytes = bytes.len() as u64;
    stream.sha256 = canonical::digest(&bytes);
    publication
        .payloads
        .insert("evidence/inspection.json".into(), bytes);
    publication
        .payloads
        .insert("target/module.wasm".into(), module.clone());
    publication.payloads.remove(search_context::FILE);
    let prepared = Prepared {
        auxiliary: std::collections::BTreeMap::new(),
        compile_cache_root: None,
        module,
        configuration,
        toolchain: identity.toolchain_build_key.clone(),
        choice_identity: Some(identity),
        scenario: None,
        provenance: b"{}".to_vec(),
        network: None,
        soft_assertion_report_only: false,
    };
    (prepared, publication)
}

fn journal(root: &Path, prepared: &Prepared) -> RoundJournal {
    let config = ChoiceConfig {
        execution: prepared.choice_identity.clone().unwrap(),
        controller_sha256: choice::implementation(),
        base_seed: 7,
        parallel: 1,
        max_executions: 2,
        max_choice_depth: 2,
        start_ordinal: 0,
        max_exploration_bytes: 1 << 20,
        failure_policy: "all".into(),
        failure_budget: 1,
        site_scope: Some(Resolution::default().scope(&["main".into()])),
    };
    let mut metadata = json!({"journal_bytes":1 << 20,"overall_nanos":1_000_000_000_u64,"overall_millis":1000,
        "module_sha256":canonical::digest(&prepared.module),"toolchain":prepared.toolchain,"configuration":prepared.configuration,
        "choice_identity":prepared.choice_identity,"host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,"scenario":null,"network":null,"soft_assertion_report_only":false});
    crate::campaign::retain_prepared(root, prepared, &mut metadata).unwrap();
    RoundJournal::create(
        root,
        &RoundPlan {
            prepared: metadata,
            strategy: RoundStrategy::Choice(config),
        },
        1 << 20,
    )
    .unwrap()
}

fn evidence(
    root: &Path,
    mut publication: Publication,
    context: &search_context::Context,
) -> (Execution, choice::ChoiceResult) {
    publication
        .payloads
        .insert(search_context::FILE.into(), context.bytes().unwrap());
    let record = publication.record.clone();
    let published = Store {
        root: root.join("evidence"),
        maximum_bytes: 0,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(publication)
    .unwrap();
    let execution = Execution {
        ordinal: 0,
        seed: 7,
        record,
        artifact: Some(
            published
                .path
                .strip_prefix(root)
                .unwrap()
                .to_string_lossy()
                .into(),
        ),
        stored_bytes: published.stored_bytes,
        semantic_probes: Vec::new(),
        choice_features: Vec::new(),
        divergence: None,
        soft_assertions: None,
        scenario_steps: None,
    };
    let result = ChoiceResult {
        candidate_sha256: String::new(),
        outcome_sha256: super::super::hash(
            "gomad-choice-exploration-outcome/v2",
            &execution.record,
        )
        .unwrap(),
        failed: false,
        failure_sha256: String::new(),
        divergence: None,
        source_trace: Some(Vec::new()),
    };
    (execution, result)
}

#[test]
fn scoped_context_publication_appends_reopens_and_refuses_wrong_round_and_suffix() {
    let temporary = search_context::tests::store();
    let root = temporary.root.join("campaign");
    let (prepared, publication) = prepared();
    let mut journal = journal(&root, &prepared);
    let context = journal.search_context(Some(&[])).unwrap().unwrap();
    let mut wrong_context = context.clone();
    wrong_context.round = 1;
    let (wrong_execution, mut result) = evidence(
        &root,
        Publication {
            record: publication.record.clone(),
            payloads: publication.payloads.clone(),
        },
        &wrong_context,
    );
    let (execution, _) = evidence(&root, publication, &context);
    let RoundState::Choice(state) = journal.state() else {
        panic!("choice state missing")
    };
    let round = state.next_round().unwrap();
    result.candidate_sha256 = round.candidates[0].sha256.clone();
    let (_, segment) = state.commit(&round, std::slice::from_ref(&result)).unwrap();
    let before = std::fs::read(root.join("journal.jsonl")).unwrap();
    let error = journal
        .append(RoundEvent::Choice {
            executions: vec![wrong_execution],
            segment: segment.clone(),
        })
        .unwrap_err();
    assert!(error.contains("pre-round snapshot"), "{error}");
    assert_eq!(std::fs::read(root.join("journal.jsonl")).unwrap(), before);
    journal
        .append(RoundEvent::Choice {
            executions: vec![execution.clone()],
            segment,
        })
        .unwrap();
    drop(journal);
    let journal = RoundJournal::open(&root, false).unwrap();
    assert_eq!(journal.events().len(), 1);
    assert_eq!(
        journal
            .open_artifact(execution.artifact.as_deref().unwrap())
            .unwrap()
            .manifest()
            .record,
        execution.record
    );
    let reference = execution.artifact.unwrap();
    let (_, name) = reference.split_once('/').unwrap();
    let mismatched = format!(
        "execution-{}-context-{}",
        execution.record.hash().unwrap(),
        "f".repeat(64)
    );
    std::fs::rename(
        root.join(&reference),
        root.join("evidence").join(&mismatched),
    )
    .unwrap();
    let error = ledger::open_artifact(
        &open::pin(&root).unwrap(),
        &format!("evidence/{mismatched}"),
    )
    .err()
    .expect("wrong context suffix accepted");
    assert!(error.contains("name does not match"), "{name}: {error}");
    drop(journal);
    assert!(RoundJournal::open(&root, false).is_err());
    std::fs::remove_dir_all(temporary.root).unwrap();
}

fn observed_preparation(root: &Path) -> Prepared {
    use crate::campaign::prepare::{observations::Observations, process};
    let (mut prepared, _) = prepared();
    let inputs = json!({"compiler":{"/layout-only/compiler":"1".repeat(64)},"sources":{"/layout-only/source":"2".repeat(64)}});
    prepared.toolchain = canonical::digest(&canonical::encode(&inputs).unwrap());
    prepared
        .choice_identity
        .as_mut()
        .unwrap()
        .toolchain_build_key
        .clone_from(&prepared.toolchain);
    prepared.provenance =
        canonical::encode(&json!({"build_key":prepared.toolchain,"inputs":inputs})).unwrap();
    let cache = root.join("observation-cache");
    let scratch = process::Temporary::new(&cache).unwrap();
    for phase in ["compile", "admission"] {
        let mut observations = Observations::default();
        let command = process::observed_command(
            Path::new("/usr/bin/true"),
            &[],
            root,
            &process::environment().unwrap(),
            &scratch.0,
            4096,
            std::time::Instant::now() + std::time::Duration::from_secs(5),
        )
        .unwrap();
        observations
            .record("layout-only-process", command, &cache)
            .unwrap();
        prepared
            .auxiliary
            .extend(observations.payloads(phase).unwrap());
    }
    prepared
}

#[test]
fn seed_journal_accepts_and_reopens_authentic_owned_preparation_observations() {
    let temporary = search_context::tests::store();
    let prepared = observed_preparation(&temporary.root);
    let mut options = crate::cli::explore::parse(
        &["--choices".into(), "go-run".into(), ".".into()],
        &mut false,
    )
    .unwrap();
    options.artifacts = temporary.root.join("seed");
    let journal = crate::campaign::seed_journal(&options, &prepared).unwrap();
    let path = journal.path().to_path_buf();
    assert_eq!(
        crate::campaign::restore(&journal).unwrap().auxiliary,
        prepared.auxiliary
    );
    drop(journal);
    let journal = crate::campaign::ledger::Journal::open(&path, false).unwrap();
    assert_eq!(
        crate::campaign::restore(&journal).unwrap().auxiliary,
        prepared.auxiliary
    );
    drop(journal);
    std::fs::remove_dir_all(temporary.root).unwrap();
}

#[test]
fn round_journal_accepts_and_reopens_authentic_owned_preparation_observations() {
    let temporary = search_context::tests::store();
    let prepared = observed_preparation(&temporary.root);
    let path = temporary.root.join("round");
    let journal = journal(&path, &prepared);
    assert_eq!(
        crate::campaign::prepare::open_observations(
            &path,
            &prepared,
            &journal
                .open_artifact(
                    journal.plan().prepared["prepared_artifact"]
                        .as_str()
                        .unwrap()
                )
                .unwrap()
                .manifest()
                .record_sha256
        )
        .unwrap(),
        prepared.auxiliary
    );
    drop(journal);
    let journal = RoundJournal::open(&path, false).unwrap();
    assert_eq!(
        crate::campaign::exploration::plan::restore(&journal)
            .unwrap()
            .auxiliary,
        prepared.auxiliary
    );
    drop(journal);
    std::fs::remove_dir_all(temporary.root).unwrap();
}

#[test]
fn seed_and_round_reopen_with_execution_metrics_and_refuse_unsafe_directory() {
    use std::os::unix::fs::PermissionsExt as _;
    for round in [false, true] {
        let temporary = search_context::tests::store();
        let prepared = observed_preparation(&temporary.root);
        let path = if round {
            let path = temporary.root.join("round");
            drop(journal(&path, &prepared));
            path
        } else {
            let mut options = crate::cli::explore::parse(
                &["--choices".into(), "go-run".into(), ".".into()],
                &mut false,
            )
            .unwrap();
            options.artifacts = temporary.root.join("seed");
            let journal = crate::campaign::seed_journal(&options, &prepared).unwrap();
            journal.path().to_path_buf()
        };
        let execution = Execution {
            ordinal: 0,
            seed: 7,
            record: self::prepared().1.record,
            artifact: None,
            stored_bytes: 0,
            semantic_probes: vec![],
            choice_features: vec![],
            divergence: None,
            soft_assertions: None,
            scenario_steps: None,
        };
        crate::campaign::execution::retain_backend_metrics(
            &path,
            &execution,
            &crate::campaign::execution::Metrics {
                raw_backend: Some(canonical::encode(&json!({"execute_nanos":10})).unwrap()),
                ..Default::default()
            },
        )
        .unwrap();
        let reopen = |writable| {
            if round {
                RoundJournal::open(&path, writable)
                    .and_then(|journal| crate::campaign::exploration::plan::restore(&journal))
            } else {
                crate::campaign::ledger::Journal::open(&path, writable)
                    .and_then(|journal| crate::campaign::restore(&journal))
            }
        };
        for writable in [false, true] {
            assert_eq!(reopen(writable).unwrap().auxiliary, prepared.auxiliary);
        }
        let observations = path.join("execution-observations");
        assert_eq!(std::fs::read_dir(&observations).unwrap().count(), 1);
        std::fs::set_permissions(&observations, std::fs::Permissions::from_mode(0o755)).unwrap();
        let bytes = std::fs::read(path.join("journal.jsonl")).unwrap();
        assert!(reopen(true).is_err());
        assert_eq!(std::fs::read(path.join("journal.jsonl")).unwrap(), bytes);
        std::fs::set_permissions(&observations, std::fs::Permissions::from_mode(0o700)).unwrap();
        let moved = temporary.root.join("moved-observations");
        std::fs::rename(&observations, &moved).unwrap();
        std::os::unix::fs::symlink(moved, &observations).unwrap();
        assert!(reopen(true).is_err());
        assert_eq!(std::fs::read(path.join("journal.jsonl")).unwrap(), bytes);
        std::fs::remove_dir_all(temporary.root).unwrap();
    }
}

fn layout_create(path: &Path, metadata: &Value, round: Option<&RoundPlan>) -> Result<(), String> {
    let maximum = metadata["journal_bytes"].as_u64().unwrap();
    if let Some(round) = round {
        let mut plan = round.clone();
        plan.prepared = metadata.clone();
        RoundJournal::create(path, &plan, maximum).map(|_| ())
    } else {
        crate::campaign::ledger::Journal::create(path, metadata, maximum).map(|_| ())
    }
}

fn corrupt_preparation_layout(path: &Path, mutation: &str) {
    use std::os::unix::fs::PermissionsExt as _;
    let sidecar = path.join(crate::campaign::prepare::OBSERVATION_DIRECTORY);
    let metadata = sidecar.join("target/preparation-inputs/admission/metadata.json");
    match mutation {
        "unknown-root" => std::fs::create_dir(path.join("unexpected")).unwrap(),
        "unknown-sidecar" => std::fs::create_dir(sidecar.join("unexpected")).unwrap(),
        "sidecar-link" => {
            let moved = path.parent().unwrap().join(format!(
                "{}-sidecar",
                path.file_name().unwrap().to_str().unwrap()
            ));
            std::fs::rename(&sidecar, &moved).unwrap();
            std::os::unix::fs::symlink(moved, sidecar).unwrap();
        }
        "payload-link" => {
            std::fs::remove_file(&metadata).unwrap();
            std::os::unix::fs::symlink(sidecar.join("receipt.json"), metadata).unwrap();
        }
        "payload-mode" => {
            std::fs::set_permissions(metadata, std::fs::Permissions::from_mode(0o644)).unwrap();
        }
        "sidecar-mode" => {
            std::fs::set_permissions(sidecar, std::fs::Permissions::from_mode(0o755)).unwrap();
        }
        "payload-bytes" => std::fs::write(metadata, b"changed observed command").unwrap(),
        "receipt-duplicate" => {
            std::fs::write(sidecar.join("receipt.json"), b"{\"schema\":1,\"schema\":2}").unwrap();
        }
        "receipt-binding" => {
            let receipt = sidecar.join("receipt.json");
            let mut value: Value =
                serde_json::from_slice(&std::fs::read(&receipt).unwrap()).unwrap();
            value["prepared_record_sha256"] = json!("f".repeat(64));
            std::fs::write(receipt, canonical::encode(&value).unwrap()).unwrap();
        }
        "missing-prepared" => std::fs::remove_dir_all(path.join("prepared")).unwrap(),
        _ => unreachable!(),
    }
}

#[test]
fn journal_preparation_layout_rejects_unsafe_entries_before_creation_or_tail_repair() {
    let temporary = search_context::tests::store();
    let prepared = observed_preparation(&temporary.root);
    let mut options = crate::cli::explore::parse(
        &["--choices".into(), "go-run".into(), ".".into()],
        &mut false,
    )
    .unwrap();
    options.artifacts = temporary.root.join("seed-base");
    let seed = crate::campaign::seed_journal(&options, &prepared).unwrap();
    let seed_plan = seed.plan().clone();
    drop(seed);
    let round = journal(&temporary.root.join("round-base"), &prepared);
    let round_plan = round.plan().clone();
    drop(round);
    for (name, metadata, round) in [
        ("seed", &seed_plan, None),
        ("round", &round_plan.prepared, Some(&round_plan)),
    ] {
        for mutation in [
            "unknown-root",
            "unknown-sidecar",
            "sidecar-link",
            "payload-link",
            "payload-mode",
            "sidecar-mode",
            "payload-bytes",
            "receipt-duplicate",
            "receipt-binding",
            "missing-prepared",
        ] {
            for existing in [false, true] {
                let path = temporary.root.join(format!("{name}-{mutation}-{existing}"));
                let mut plan = metadata.clone();
                crate::campaign::retain_prepared(&path, &prepared, &mut plan).unwrap();
                if existing {
                    layout_create(&path, &plan, round).unwrap();
                    std::fs::write(path.join("journal.jsonl"), b"torn").unwrap();
                }
                corrupt_preparation_layout(&path, mutation);
                let result = if existing {
                    if round.is_some() {
                        RoundJournal::open(&path, true).map(|_| ())
                    } else {
                        crate::campaign::ledger::Journal::open(&path, true).map(|_| ())
                    }
                } else {
                    layout_create(&path, &plan, round)
                };
                assert!(result.is_err(), "{name} {mutation} existing={existing}");
                if existing {
                    assert_eq!(std::fs::read(path.join("journal.jsonl")).unwrap(), b"torn");
                } else {
                    assert!(!path.join(".lock").exists());
                    assert!(!path.join("storage.json").exists());
                    assert!(!path.join("journal.jsonl").exists());
                }
            }
        }
        let path = temporary.root.join(format!("{name}-duplicate"));
        let mut plan = metadata.clone();
        crate::campaign::retain_prepared(&path, &prepared, &mut plan).unwrap();
        layout_create(&path, &plan, round).unwrap();
        assert!(layout_create(&path, &plan, round).is_err());
    }
    std::fs::remove_dir_all(temporary.root).unwrap();
}
