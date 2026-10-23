use super::*;
use crate::evidence::{
    inspection::{self, ChoiceEvidence},
    record::Stream,
    search_context::tests::{publication, store},
};
use serde_json::{Value, json};
use std::os::unix::fs::PermissionsExt as _;

fn trace() -> Vec<u8> {
    let mut raw = vec![0; 2 * 96];
    for (ordinal, record) in raw.as_chunks_mut::<96>().0.iter_mut().enumerate() {
        record[..8].copy_from_slice(&(ordinal as u64).to_be_bytes());
        record[8] = 1;
        record[9] = 1;
        record[12..16].copy_from_slice(&2_u32.to_be_bytes());
        record[24..32].copy_from_slice(&5383_u64.to_be_bytes());
        record[32..64].fill(1);
        record[64..96].fill(2);
    }
    raw
}

fn replace(publication: &mut Publication, name: &str, raw: Vec<u8>) {
    publication.record.streams.insert(
        name.into(),
        Stream {
            bytes: raw.len() as u64,
            sha256: canonical::digest(&raw),
            records: 1,
        },
    );
    publication.payloads.insert(name.into(), raw);
}

fn choices(choices: &mut ChoiceEvidence, raw: &[u8]) {
    choices.trace.bytes = raw.len() as u64;
    choices.trace.sha256 = canonical::digest(raw);
    choices.records = (raw.len() / 96) as u64;
    choices.tape_sha256 = Some(
        linked::inspect_choices(raw, &choices.identity)
            .unwrap()
            .tape_sha256,
    );
}

fn minimized() -> Publication {
    minimized_with_symbols(false)
}

fn candidate_hashes(raw: &[u8], identity: &linked::ChoiceIdentity) -> (String, String) {
    let mut state =
        crate::minimize::proposals::State::choice(raw.to_vec(), identity.clone(), 8).unwrap();
    state = state
        .commit(&state.next().unwrap().unwrap(), false)
        .unwrap();
    state = state.commit(&state.next().unwrap().unwrap(), true).unwrap();
    (
        state
            .original
            .sha256
            .trim_start_matches("sha256:")
            .to_owned(),
        state
            .current
            .sha256
            .trim_start_matches("sha256:")
            .to_owned(),
    )
}

fn minimized_with_symbols(symbols: bool) -> Publication {
    let mut publication = publication(0);
    publication
        .payloads
        .remove(crate::evidence::search_context::FILE);
    let mut details: Details =
        serde_json::from_slice(&publication.payloads["evidence/inspection.json"]).unwrap();
    if symbols {
        use sha2::Digest as _;
        let module = std::fs::read(
            std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../../testdata/parity/symbolize/guest.wasm"),
        )
        .unwrap();
        publication.record.identity.target = canonical::digest(&module);
        publication.record.execution_identity = publication.record.identity.digest().unwrap();
        details.choices.as_mut().unwrap().identity.target_sha256 =
            sha2::Sha256::digest(&module).into();
        publication
            .payloads
            .insert("target/module.wasm".into(), module);
    }
    let raw = trace();
    choices(details.choices.as_mut().unwrap(), &raw);
    replace(&mut publication, "evidence/choice", raw.clone());
    replace(
        &mut publication,
        "evidence/inspection.json",
        canonical::encode(&details).unwrap(),
    );
    let original = publication.record.bytes().unwrap();
    let parent_hash = publication.record.hash().unwrap();
    let signature = publication.record.failure_signature().unwrap();
    let (original_candidate, final_candidate) =
        candidate_hashes(&raw, &details.choices.as_ref().unwrap().identity);
    let raw_final = raw[..96].to_vec();
    choices(details.choices.as_mut().unwrap(), &raw_final);
    details.minimization = Some(inspection::Minimization {
        schema: "gomad.minimization-evidence/v1".into(),
        implementation_sha256: "1".repeat(64),
        parent_record_hash: parent_hash.clone(),
        parent_failure_signature: signature.clone(),
        original_candidate_sha256: original_candidate.clone(),
        final_candidate_sha256: final_candidate.clone(),
        attempt_budget: 8,
        attempts: 2,
        original_forced_decisions: 2,
        final_forced_decisions: 1,
        accepted: vec![inspection::Reduction {
            kind: "schedule_suffix".into(),
            before_sha256: original_candidate,
            after_sha256: final_candidate,
            removed: vec![inspection::Decision {
                dimension: "runtime".into(),
                ordinal: 1,
                identity: "01".repeat(32),
            }],
        }],
        predicate: inspection::Predicate {
            failure_signature: signature,
            domain: "success".into(),
            reason: String::new(),
            termination: "exit".into(),
            replay_match: true,
            choice_replay: "exact".into(),
            simulation_replay: "not_present".into(),
        },
    });
    replace(&mut publication, "evidence/choice", raw_final);
    replace(
        &mut publication,
        "evidence/inspection.json",
        canonical::encode(&details).unwrap(),
    );
    replace(
        &mut publication,
        crate::minimize::predicate::ORIGINAL,
        original,
    );
    Source {
        raw,
        identity: details.choices.unwrap().identity,
        limit: 4096,
        parent_record_sha256: parent_hash,
    }
    .attach(&mut publication)
    .unwrap();
    publication
}

fn inspect(path: &std::path::Path) -> Value {
    let mut output = Vec::new();
    crate::cli::inspect::run(
        &["--json".into(), path.to_string_lossy().into()],
        &mut output,
    )
    .unwrap_or_else(|error| panic!("{}", error.message));
    serde_json::from_slice(&output).unwrap()
}

#[test]
fn source_choice_detached_reporting_preserves_raw_evidence_and_legacy_missing_joins() {
    let store = store();
    let artifact = store.publish(minimized()).unwrap();
    let detached = store.root.with_extension("detached");
    std::fs::rename(&artifact.path, &detached).unwrap();
    let before = std::fs::read(detached.join("evidence/inspection.json")).unwrap();
    let row = inspect(&detached);
    assert!(row["artifact"]["choices"].is_null());
    assert_eq!(
        row["artifact"]["minimization"]["accepted"][0]["removed"][0]["symbol"],
        json!({"unavailable":"no_symbols"})
    );
    assert_eq!(
        row["artifact"]["minimization"]["final_forced_by_symbol"],
        json!([{"unavailable":"no_symbols","count":1}])
    );
    assert_eq!(
        std::fs::read(detached.join("evidence/inspection.json")).unwrap(),
        before
    );
    let mut legacy = minimized();
    legacy.payloads.remove(TRACE);
    legacy.payloads.remove(DESCRIPTOR);
    let legacy = store.publish(legacy).unwrap();
    let row = inspect(&legacy.path);
    assert_eq!(
        row["artifact"]["minimization"]["accepted"][0]["removed"][0]["symbol"],
        json!({"unavailable":"site_missing"})
    );
    std::fs::remove_dir_all(detached).unwrap();
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_present_pair_and_commitments_fail_closed() {
    for missing in [TRACE, DESCRIPTOR] {
        let mut publication = minimized();
        publication.payloads.remove(missing);
        assert_invalid(publication);
    }
    for (field, value) in [
        ("schema", json!("unknown")),
        ("trace_sha256", json!("4".repeat(64))),
        ("tape_sha256", json!("4".repeat(64))),
        ("decisions_sha256", json!("4".repeat(64))),
        ("decisions", json!(3)),
        ("limit", json!(u64::MAX)),
        ("unknown", json!(true)),
    ] {
        let mut publication = minimized();
        let mut descriptor: Value =
            serde_json::from_slice(&publication.payloads[DESCRIPTOR]).unwrap();
        descriptor[field] = value;
        publication
            .payloads
            .insert(DESCRIPTOR.into(), canonical::encode(&descriptor).unwrap());
        assert_invalid(publication);
    }
}

fn assert_invalid(publication: Publication) {
    let store = store();
    match store.publish(publication) {
        Ok(artifact) => assert!(Opened::open(&artifact.path).is_err()),
        Err(error) => assert!(
            error.contains("minimization source") || error.contains("unlisted artifact payload"),
            "{error}"
        ),
    }
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_exact_identity_join_refuses_to_attribute_same_ordinal() {
    let store = store();
    let mut publication = minimized();
    let mut details: Details =
        serde_json::from_slice(&publication.payloads["evidence/inspection.json"]).unwrap();
    details.minimization.as_mut().unwrap().accepted[0].removed[0].identity = "ff".repeat(32);
    replace(
        &mut publication,
        "evidence/inspection.json",
        canonical::encode(&details).unwrap(),
    );
    let error = store
        .publish(publication)
        .err()
        .expect("changed removed identity accepted");
    assert!(
        error.contains("removed identity differs from source tape"),
        "{error}"
    );
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_symbolized_groups_and_inherited_ordering_use_retained_module() {
    let store = store();
    let mut publication = minimized_with_symbols(true);
    let details: Details =
        serde_json::from_slice(&publication.payloads["evidence/inspection.json"]).unwrap();
    let choices = details.choices.unwrap();
    let patterns = vec!["main".into()];
    let mut resolution = crate::campaign::exploration::site_scope::Resolution::default();
    resolution
        .observe(
            &publication.payloads["target/module.wasm"],
            &publication.payloads["evidence/choice"],
            &choices.identity,
            &patterns,
        )
        .unwrap();
    let context = crate::evidence::search_context::Context::new(
        patterns,
        resolution,
        "a".repeat(64),
        2,
        choices.identity,
    );
    publication.payloads.insert(
        crate::evidence::search_context::FILE.into(),
        context.bytes().unwrap(),
    );
    let artifact = store.publish(publication).unwrap();
    let opened = Opened::open(&artifact.path).unwrap();
    let inherited = crate::minimize::ordering::prepare(&opened, None).unwrap();
    assert_eq!(inherited.scope.unwrap().patterns, ["main"]);
    let explicit =
        crate::minimize::ordering::prepare(&opened, Some(vec!["gomad.invalid/no-match".into()]))
            .unwrap();
    assert_eq!(explicit.scope.unwrap().resolved_count, 0);
    let inherited = inherited.ordering.unwrap();
    let explicit = explicit.ordering.unwrap();
    assert_ne!(inherited.key, explicit.key);
    assert_eq!(inherited.key["source"], explicit.key["source"]);
    assert_eq!(inherited.key["selection"], "inherited");
    assert_eq!(explicit.key["selection"], "explicit");
    // Legacy runnable records have no authenticated alternative creation sites.
    assert!(inherited.in_scope.is_empty());
    let row = inspect(&artifact.path);
    let metadata = &row["artifact"]["minimization"];
    let symbol = &metadata["accepted"][0]["removed"][0]["symbol"];
    assert_eq!(symbol["package"], "main");
    assert_eq!(
        metadata["final_forced_by_symbol"],
        json!([{"package":"main","function":symbol["function"],"count":1}])
    );
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_resealed_retained_payloads_reach_shared_opener_admission() {
    let store = store();
    let artifact = store.publish(minimized()).unwrap();
    let manifest_path = artifact.path.join("manifest.json");
    let manifest: Value = serde_json::from_slice(&std::fs::read(&manifest_path).unwrap()).unwrap();
    let original = std::fs::read(artifact.path.join(DESCRIPTOR)).unwrap();
    let descriptor: Value = serde_json::from_slice(&original).unwrap();
    for (field, value) in [
        ("schema", json!("unknown")),
        ("trace_sha256", json!("4".repeat(64))),
        ("tape_sha256", json!("4".repeat(64))),
        ("decisions_sha256", json!("4".repeat(64))),
        ("decisions", json!(3)),
        ("unknown", json!(true)),
    ] {
        let mut changed = descriptor.clone();
        changed[field] = value;
        let bytes = canonical::encode(&changed).unwrap();
        std::fs::write(artifact.path.join(DESCRIPTOR), &bytes).unwrap();
        let mut resealed = manifest.clone();
        resealed["files"][DESCRIPTOR]["bytes"] = json!(bytes.len());
        resealed["files"][DESCRIPTOR]["sha256"] = json!(canonical::digest(&bytes));
        std::fs::write(&manifest_path, canonical::encode(&resealed).unwrap()).unwrap();
        let error = Opened::open(&artifact.path)
            .err()
            .expect("resealed invalid descriptor accepted");
        assert!(error.contains("minimization source"), "{field}: {error}");
    }
    std::fs::write(artifact.path.join(DESCRIPTOR), &original).unwrap();
    for missing in [TRACE, DESCRIPTOR] {
        let bytes = std::fs::read(artifact.path.join(missing)).unwrap();
        std::fs::remove_file(artifact.path.join(missing)).unwrap();
        let mut resealed = manifest.clone();
        resealed["files"].as_object_mut().unwrap().remove(missing);
        std::fs::write(&manifest_path, canonical::encode(&resealed).unwrap()).unwrap();
        let error = Opened::open(&artifact.path)
            .err()
            .expect("single source payload accepted");
        assert!(
            error.contains("unlisted artifact payload"),
            "{missing}: {error}"
        );
        std::fs::write(artifact.path.join(missing), bytes).unwrap();
        std::fs::set_permissions(
            artifact.path.join(missing),
            std::fs::Permissions::from_mode(0o600),
        )
        .unwrap();
    }
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_resealed_site_substitution_refuses_unchanged_record() {
    let store = store();
    let artifact = store.publish(minimized_with_symbols(true)).unwrap();
    let detached = store.root.with_extension("substituted");
    std::fs::rename(&artifact.path, &detached).unwrap();
    let opened = Opened::open(&detached).unwrap();
    let mut source = read(&opened).unwrap().unwrap();
    let original_record = opened.manifest().record.bytes().unwrap();
    let original_metadata = opened
        .payload("evidence/inspection.json", 16 << 20)
        .unwrap();
    source.raw[120..128].copy_from_slice(&5384_u64.to_be_bytes());
    let mut publication = crate::minimize::predicate::publication(&opened).unwrap();
    source.attach(&mut publication).unwrap();
    let mut manifest = opened.manifest().clone();
    for name in [TRACE, DESCRIPTOR] {
        let bytes = &publication.payloads[name];
        std::fs::write(detached.join(name), bytes).unwrap();
        manifest.files.get_mut(name).unwrap().bytes = bytes.len() as u64;
        manifest.files.get_mut(name).unwrap().sha256 = canonical::digest(bytes);
    }
    std::fs::write(
        detached.join("manifest.json"),
        canonical::encode(&manifest).unwrap(),
    )
    .unwrap();
    assert_eq!(manifest.record.bytes().unwrap(), original_record);
    assert_eq!(
        std::fs::read(detached.join("evidence/inspection.json")).unwrap(),
        original_metadata
    );
    let error = Opened::open(&detached)
        .err()
        .expect("resealed site substitution accepted");
    assert!(
        error.contains("source candidate commitment mismatch"),
        "{error}"
    );
    std::fs::remove_dir_all(detached).unwrap();
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn minimize_malformed_present_context_names_parent_before_workspace_creation() {
    let store = store();
    let artifact = store.publish(publication(0)).unwrap();
    let context_path = artifact.path.join(crate::evidence::search_context::FILE);
    let mut context: Value =
        serde_json::from_slice(&std::fs::read(&context_path).unwrap()).unwrap();
    context["schema"] = json!("unknown");
    let bytes = canonical::encode(&context).unwrap();
    std::fs::write(&context_path, &bytes).unwrap();
    let manifest_path = artifact.path.join("manifest.json");
    let mut manifest: Value =
        serde_json::from_slice(&std::fs::read(&manifest_path).unwrap()).unwrap();
    manifest["files"][crate::evidence::search_context::FILE]["bytes"] = json!(bytes.len());
    manifest["files"][crate::evidence::search_context::FILE]["sha256"] =
        json!(canonical::digest(&bytes));
    std::fs::write(manifest_path, canonical::encode(&manifest).unwrap()).unwrap();
    let output_root = store.root.join("uncreated-minimizer");
    let result = crate::cli::minimize::run(
        &[
            "--site-scope=main".into(),
            "--artifacts".into(),
            output_root.to_string_lossy().into(),
            artifact.path.to_string_lossy().into(),
        ],
        &mut Vec::new(),
    );
    let error = result.expect_err("malformed present context accepted");
    assert_eq!(error.status, 2);
    assert!(
        error
            .message
            .contains(&artifact.path.to_string_lossy().to_string()),
        "{}",
        error.message
    );
    assert!(
        error.message.contains("search context"),
        "{}",
        error.message
    );
    assert!(!output_root.exists());
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn source_choice_final_prefix_commitment_is_checked_with_original_intact() {
    let store = store();
    let mut publication = minimized();
    let mut details: Details =
        serde_json::from_slice(&publication.payloads["evidence/inspection.json"]).unwrap();
    let metadata = details.minimization.as_mut().unwrap();
    metadata.final_candidate_sha256 = "f".repeat(64);
    metadata.accepted.last_mut().unwrap().after_sha256 = metadata.final_candidate_sha256.clone();
    replace(
        &mut publication,
        "evidence/inspection.json",
        canonical::encode(&details).unwrap(),
    );
    let error = store
        .publish(publication)
        .err()
        .expect("changed final prefix commitment accepted");
    assert!(
        error.contains("source candidate commitment mismatch"),
        "{error}"
    );
    std::fs::remove_dir_all(store.root).unwrap();
}
