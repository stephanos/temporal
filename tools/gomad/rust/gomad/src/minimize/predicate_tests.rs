use super::*;
use crate::{
    campaign::ledger,
    evidence::{
        record::Stream,
        search_context::tests::{publication as fixture, store},
    },
};

const PROVENANCE: &str = "target/provenance.json";

fn bind(publication: &mut Publication, name: &str, raw: &[u8]) {
    publication.record.streams.insert(
        name.into(),
        Stream {
            bytes: raw.len() as u64,
            sha256: canonical::digest(raw),
            records: 1,
        },
    );
    publication.payloads.insert(name.into(), raw.to_vec());
    publication
        .payloads
        .insert("record.json".into(), publication.record.bytes().unwrap());
}

fn trial(parent: &Opened) -> execution::Result {
    let mut publication = publication(parent).unwrap();
    if !publication.payloads.contains_key(PROVENANCE) {
        bind(&mut publication, PROVENANCE, b"{}");
    }
    execution::Result {
        execution: ledger::Execution {
            ordinal: 0,
            seed: publication.record.seed,
            record: publication.record.clone(),
            artifact: None,
            stored_bytes: 0,
            semantic_probes: Vec::new(),
            choice_features: Vec::new(),
            divergence: None,
            scenario_steps: None,
            soft_assertions: None,
        },
        publication,
        metrics: execution::Metrics::default(),
    }
}

#[test]
fn absent_provenance_preserves_full_record_and_publication_consistency() {
    let store = store();
    let parent = store.publish(fixture(0)).unwrap();
    let parent = Opened::open(&parent.path).unwrap();
    let mut result = trial(&parent);
    assert_ne!(result.publication.record, parent.manifest().record);
    retain_provenance_presence(&parent, &mut result).unwrap();
    assert_eq!(result.publication.record, parent.manifest().record);
    assert_eq!(result.execution.record, result.publication.record);
    assert_eq!(
        result.publication.payloads["record.json"],
        parent.manifest().record.bytes().unwrap()
    );
    assert!(!result.publication.payloads.contains_key(PROVENANCE));
    assert!(!result.publication.record.streams.contains_key(PROVENANCE));
    drop(parent);
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn present_provenance_remains_byte_identical_and_invalid_schema_still_refuses() {
    let store = store();
    let original = fixture(0);
    let configuration = preflight::Configuration {
        world_transition_bytes: None,
        engine: linked::Conditions::compiled().configuration(),
        profile: original.record.profile.clone(),
        limits: original.record.limits.clone(),
        wall_nanos: None,
        arguments: original.record.arguments.clone(),
        environment: original.record.environment.clone(),
        derive_seed_environment: false,
        network: None,
        runtime_sha256: Some("0".repeat(64)),
        model: None,
        namespace: None,
        storage: None,
    };
    let valid = serde_json::json!({
        "schema":"gomad.rust-wasm-provenance/v1", "inputs":{"kind":"go-run"},
        "build_key":original.record.identity.toolchain, "module_sha256":original.record.identity.target,
        "configuration":configuration, "choice_identity":null, "scenario":null, "network":null
    });
    crate::campaign::prepare::typed_provenance(&valid).unwrap();
    for bytes in [b"{}".to_vec(), canonical::encode(&valid).unwrap()] {
        let mut original = fixture(0);
        bind(&mut original, PROVENANCE, &bytes);
        let published = store.publish(original).unwrap();
        let parent = Opened::open(&published.path).unwrap();
        parent.authenticate_all().unwrap();
        let mut result = trial(&parent);
        let before = result.publication.record.bytes().unwrap();
        retain_provenance_presence(&parent, &mut result).unwrap();
        assert_eq!(result.publication.record.bytes().unwrap(), before);
        assert_eq!(result.publication.payloads[PROVENANCE], bytes);
        assert_eq!(result.execution.record, result.publication.record);
        let typed =
            crate::campaign::prepare::typed_provenance(&serde_json::from_slice(&bytes).unwrap());
        assert_eq!(typed.is_ok(), bytes != b"{}");
    }
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn absent_parent_refuses_unexpected_placeholder_descriptor_or_inconsistent_records() {
    let store = store();
    let parent = store.publish(fixture(0)).unwrap();
    let parent = Opened::open(&parent.path).unwrap();
    for mutation in 0..6 {
        let mut result = trial(&parent);
        match mutation {
            0 => {
                result
                    .publication
                    .payloads
                    .insert(PROVENANCE.into(), b"unexpected".to_vec());
            }
            1 => {
                result
                    .publication
                    .record
                    .streams
                    .get_mut(PROVENANCE)
                    .unwrap()
                    .bytes = 3;
            }
            2 => {
                result
                    .publication
                    .record
                    .streams
                    .get_mut(PROVENANCE)
                    .unwrap()
                    .sha256 = "0".repeat(64);
            }
            3 => {
                result
                    .publication
                    .record
                    .streams
                    .get_mut(PROVENANCE)
                    .unwrap()
                    .records = 2;
            }
            4 => {
                result.execution.record.outcome.reason = "changed".into();
            }
            _ => {
                result
                    .publication
                    .payloads
                    .insert("record.json".into(), b"stale".to_vec());
            }
        }
        let before = result.publication.record.bytes().unwrap();
        assert!(retain_provenance_presence(&parent, &mut result).is_err());
        assert_eq!(result.publication.record.bytes().unwrap(), before);
        assert!(result.publication.payloads.contains_key(PROVENANCE));
    }
    drop(parent);
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn absence_propagation_does_not_exempt_other_bound_metadata_from_predicate() {
    let store = store();
    let parent = store.publish(fixture(0)).unwrap();
    let parent = Opened::open(&parent.path).unwrap();
    let mut result = trial(&parent);
    bind(&mut result.publication, "evidence/other.json", b"{}");
    result
        .execution
        .record
        .clone_from(&result.publication.record);
    retain_provenance_presence(&parent, &mut result).unwrap();
    assert!(
        result
            .publication
            .record
            .streams
            .contains_key("evidence/other.json")
    );
    assert_ne!(
        result.publication.record.failure_signature().unwrap(),
        parent.manifest().record.failure_signature().unwrap()
    );
    drop(parent);
    std::fs::remove_dir_all(store.root).unwrap();
}

#[test]
fn authenticated_present_provenance_payload_tampering_refuses() {
    let store = store();
    let mut original = fixture(0);
    bind(&mut original, PROVENANCE, b"{}");
    let published = store.publish(original).unwrap();
    Opened::open(&published.path)
        .unwrap()
        .authenticate_all()
        .unwrap();
    std::fs::write(published.path.join(PROVENANCE), b"[]").unwrap();
    assert!(
        Opened::open(&published.path)
            .and_then(|parent| parent.authenticate_all())
            .is_err()
    );
    std::fs::remove_dir_all(store.root).unwrap();
}
