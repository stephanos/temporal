use super::{authenticate_provenance_module, derive};
use crate::evidence::canonical;
use serde_json::{Value, json};

fn genuine() -> Value {
    let configuration = json!({
        "engine":"cranelift-fuel-nan-canonical-nounwind-epoch-v1",
        "profile":"gomad.wasi-stock-exploratory/v1",
        "limits":{"memory_bytes":268_435_456,"fuel":4_000_000_000_u64,"wall_millis":60_000,"output_bytes":1_048_576,"evidence_bytes":134_217_728},
        "arguments":["gomad-target"],"environment":{"PWD":"/workspace"},
        "runtime_sha256":null,"model":null,"namespace":null,"storage":null
    });
    let metadata = json!({"imports":[],"initial_memory_pages":1});
    let inputs = json!({
        "arguments":[],"build_flags":[],"compiler":{},"compiler_path":"/pinned/compiler",
        "compiler_sha256":"a".repeat(64),"configuration":configuration,"directory":"/source",
        "environment":{},"go_version":"go1.27.1","helper":"b".repeat(64),"host":"c".repeat(64),
        "kind":"go-run","module_metadata":metadata,"network":null,"runtime":null,"runtime_implementation":null,
        "scenario":null,"schema":"gomad.rust-wasm-build-inputs/v1","soft_assertion_report_only":false,
        "source":".","sources":{},"tags":[]
    });
    json!({
        "schema":"gomad.rust-wasm-provenance/v1","build_key":canonical::digest(&canonical::encode(&inputs).unwrap()),
        "choice_identity":null,"configuration":configuration,"inputs":inputs,
        "module_metadata":metadata,"module_sha256":canonical::digest(module()),"network":null,"scenario":null
    })
}

fn module() -> &'static [u8] {
    b"\0asm\x01\0\0\0\x05\x03\x01\0\x01"
}

#[test]
fn genuine_preparation_compiler_path_and_typed_metadata_bind_raw_module() {
    let provenance = genuine();
    let derived = derive(&provenance).unwrap();
    assert_eq!(
        derived.toolchain_sha256,
        provenance["build_key"].as_str().unwrap()
    );
    assert_eq!(derived.module_sha256, canonical::digest(module()));
    authenticate_provenance_module(&provenance, module()).unwrap();
}

#[test]
fn qualification_provenance_rejects_unknown_nested_or_unbound_fields() {
    let original = genuine();
    let mut unknown = original.clone();
    unknown["unexpected"] = json!(true);
    assert!(derive(&unknown).is_err());
    let mut nested = original.clone();
    nested["module_metadata"]["unexpected"] = json!(true);
    nested["inputs"]["module_metadata"] = nested["module_metadata"].clone();
    nested["build_key"] = json!(canonical::digest(
        &canonical::encode(&nested["inputs"]).unwrap()
    ));
    assert!(derive(&nested).is_err());
    let mut compiler = original.clone();
    compiler["inputs"]["compiler_path"] = json!("relative/compiler");
    compiler["build_key"] = json!(canonical::digest(
        &canonical::encode(&compiler["inputs"]).unwrap()
    ));
    assert!(derive(&compiler).is_err());
    let mut missing = original;
    missing["inputs"]
        .as_object_mut()
        .unwrap()
        .remove("compiler_path");
    missing["build_key"] = json!(canonical::digest(
        &canonical::encode(&missing["inputs"]).unwrap()
    ));
    assert!(derive(&missing).is_err());
}

#[test]
fn rehashed_typed_metadata_cannot_change_the_actual_wasm_memory() {
    let mut provenance = genuine();
    provenance["module_metadata"]["initial_memory_pages"] = json!(2);
    provenance["inputs"]["module_metadata"] = provenance["module_metadata"].clone();
    provenance["build_key"] = json!(canonical::digest(
        &canonical::encode(&provenance["inputs"]).unwrap()
    ));
    derive(&provenance).unwrap();
    assert!(authenticate_provenance_module(&provenance, module()).is_err());
    let mut disagreement = genuine();
    disagreement["inputs"]["module_metadata"]["initial_memory_pages"] = json!(2);
    disagreement["build_key"] = json!(canonical::digest(
        &canonical::encode(&disagreement["inputs"]).unwrap()
    ));
    assert!(derive(&disagreement).is_err());
}
