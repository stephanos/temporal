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
