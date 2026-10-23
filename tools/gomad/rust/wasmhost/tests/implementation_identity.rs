#[path = "../build_identity.rs"]
mod build_identity;

use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, fs, path::Path};

type Captured = (BTreeMap<String, Vec<u8>>, BTreeMap<String, String>, Vec<u8>);

fn captured() -> Captured {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let mut files = BTreeMap::new();
    for line in include_str!(concat!(env!("OUT_DIR"), "/measurement-source-inputs.tsv")).lines() {
        let (_, name) = line.split_once('\t').expect("captured source identity");
        files.insert(
            name.into(),
            fs::read(root.join(name)).expect("captured source"),
        );
    }
    let bindings = include_str!(concat!(env!("OUT_DIR"), "/identity-bindings.tsv"))
        .lines()
        .map(|line| {
            let (name, value) = line.split_once('\t').expect("captured build binding");
            (name.into(), value.into())
        })
        .collect();
    let compiler = include_bytes!(concat!(env!("OUT_DIR"), "/identity-compiler.txt")).to_vec();
    (files, bindings, compiler)
}

fn observe(case: &str, inputs: &Value, observed: &Value) {
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        json!({
            "schema":"gomad.core-predicate-observation/v1", "case":case,
            "inputs":inputs, "observed":observed,
        })
    );
}

fn source_inputs(
    files: &BTreeMap<String, Vec<u8>>,
    bindings: &BTreeMap<String, String>,
    compiler: &[u8],
) -> Value {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let files: Vec<_> = files
        .iter()
        .map(|(name, bytes)| {
            json!({
                "name":name,"path":root.join(name),"sha256":format!("{:x}",Sha256::digest(bytes)),
                "bytes":bytes.len(),
            })
        })
        .collect();
    json!({"files":files,"bindings":bindings,"compiler_utf8":std::str::from_utf8(compiler).unwrap(),
        "required_owners":build_identity::REQUIRED_OWNERS})
}

#[test]
fn compiled_identity_matches_the_exact_captured_native_source_and_build_context() {
    let (files, bindings, compiler) = captured();
    observe(
        "identity.compiled-native",
        &source_inputs(&files, &bindings, &compiler),
        &json!({"recomputed":build_identity::implementation(&files, &bindings, &compiler).unwrap(),
            "compiled":env!("GOMAD_LINKED_IMPLEMENTATION_SHA256")}),
    );
    assert_eq!(
        build_identity::implementation(&files, &bindings, &compiler).unwrap(),
        env!("GOMAD_LINKED_IMPLEMENTATION_SHA256")
    );
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
    let mut discovered = BTreeMap::new();
    build_identity::sources(manifest, &manifest.join("src"), &mut discovered);
    for (name, bytes) in discovered {
        assert_eq!(files.get(&name), Some(&bytes), "uncaptured owner {name}");
    }
}

fn mutation_and_missing_refusal(case: &str, owners: &[&str]) {
    let (files, bindings, compiler) = captured();
    let before = build_identity::implementation(&files, &bindings, &compiler).unwrap();
    assert_eq!(before.len(), 64);
    for name in owners {
        let mut changed = files.clone();
        changed
            .get_mut(*name)
            .expect("required native owner")
            .extend_from_slice(b"changed implementation");
        let after = build_identity::implementation(&changed, &bindings, &compiler).unwrap();
        assert_ne!(
            build_identity::implementation(&changed, &bindings, &compiler).unwrap(),
            before,
            "{name}"
        );
        changed.remove(*name);
        let missing = build_identity::implementation(&changed, &bindings, &compiler).unwrap_err();
        let mut inputs = source_inputs(&files, &bindings, &compiler);
        inputs["owner"] = json!(name);
        inputs["mutation_append_utf8"] = json!("changed implementation");
        observe(
            case,
            &inputs,
            &json!({"before":before,"after":after,"missing_error":missing}),
        );
        assert_eq!(
            build_identity::implementation(&changed, &bindings, &compiler).unwrap_err(),
            format!("missing implementation owner {name}")
        );
    }
}

#[test]
fn native_storage_evidence_owners_rotate_identity_and_missing_owners_refuse() {
    mutation_and_missing_refusal(
        "identity.storage-owner",
        &[
            "src/linked_cluster.rs",
            "src/linked.rs",
            "src/linked_storage.rs",
            "src/storage_transport.rs",
            "src/volume/mod.rs",
            "src/namespace/image.rs",
            "src/namespace/call.rs",
            "src/metrics.rs",
            "src/volume/image.rs",
        ],
    );
}

#[test]
fn native_inspection_owners_rotate_identity_and_missing_owners_refuse() {
    mutation_and_missing_refusal(
        "identity.inspection-owner",
        &[
            "src/linked_inspection.rs",
            "src/linked_storage_inspection.rs",
            "src/model/inspection.rs",
            "src/cluster/network/evidence.rs",
        ],
    );
}

#[test]
fn strict_typed_json_owners_rotate_identity_and_missing_owners_refuse() {
    mutation_and_missing_refusal(
        "identity.strict-json-owner",
        &["src/protocol.rs", "src/model.rs", "src/linked_storage.rs"],
    );
}

#[test]
fn empty_source_closure_refuses_before_producing_an_identity() {
    let (_, bindings, compiler) = captured();
    let error = build_identity::implementation(&BTreeMap::new(), &bindings, &compiler).unwrap_err();
    observe(
        "identity.empty-closure",
        &source_inputs(&BTreeMap::new(), &bindings, &compiler),
        &json!({"error":error}),
    );
    assert!(build_identity::implementation(&BTreeMap::new(), &bindings, &compiler).is_err());
}
