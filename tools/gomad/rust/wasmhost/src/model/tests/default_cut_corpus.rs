use super::*;
use serde::Deserialize;
use std::path::Path;

#[test]
fn default_cut_literals_reject_snapshot_and_changed_stage_contracts() {
    let source = fixture_bytes();
    let case: Case = serde_json::from_slice(&source).unwrap();
    literals(&case).unwrap();
    for (pointer, replacement) in [
        ("/schema", json!("other")),
        ("/name", json!("other")),
        ("/builder", json!("other")),
        ("/debug_input", json!("other")),
        ("/runtime_input", json!("other")),
        ("/mode", json!(1)),
        ("/tape", json!(null)),
        ("/tape", json!([1])),
        (
            "/actions",
            json!([
                "bootstrap",
                "snapshot",
                "debug_predecision",
                "runtime_decision"
            ]),
        ),
        (
            "/actions",
            json!(["bootstrap", "runtime_decision", "debug_predecision"]),
        ),
        ("/actions", json!(["bootstrap", "debug_predecision"])),
        (
            "/actions",
            json!([
                "bootstrap",
                "bootstrap",
                "debug_predecision",
                "runtime_decision"
            ]),
        ),
        ("/expected/debug_success", json!(false)),
        ("/expected/divergence_after_debug", json!(true)),
        ("/expected/runtime_error", json!(false)),
        ("/expected/divergence_after_runtime", json!(false)),
    ] {
        let mut changed: serde_json::Value = serde_json::from_slice(&source).unwrap();
        *changed.pointer_mut(pointer).unwrap() = replacement;
        if let Ok(parsed) = serde_json::from_value::<Case>(changed) {
            assert!(
                literals(&parsed).is_err(),
                "changed no-snapshot contract accepted: {pointer}"
            );
        }
    }
    let mut nonempty: Case = serde_json::from_slice(&source).unwrap();
    nonempty.tape = Some(vec![Decision {
        ordinal: 0,
        kind: 1,
        site_offset: 0,
        site_missing: false,
        rank_override: false,
        alternatives: 2,
        selected: 0,
        data: 0,
        selected_identity: [0; 32],
        alternative_set_digest: [0; 32],
        creation_sites: Vec::new(),
    }]);
    assert!(literals(&nonempty).is_err(), "nonempty typed tape accepted");
    let mut missing: serde_json::Value = serde_json::from_slice(&source).unwrap();
    missing.as_object_mut().unwrap().remove("tape");
    let missing: Case = serde_json::from_value(missing).unwrap();
    assert!(
        literals(&missing).is_err(),
        "missing Some(empty) tape accepted"
    );
    let mut unknown: serde_json::Value = serde_json::from_slice(&source).unwrap();
    unknown["expected"]["unknown"] = json!(true);
    assert!(serde_json::from_value::<Case>(unknown).is_err());
    let mut trailing = source;
    trailing.extend_from_slice(b" {}");
    assert!(serde_json::from_slice::<Case>(&trailing).is_err());
}

#[derive(Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
enum Action {
    Bootstrap,
    DebugPredecision,
    RuntimeDecision,
}

#[derive(Debug, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
#[expect(
    clippy::struct_excessive_bools,
    reason = "Corpus flags independently select success, refusal and divergence observations."
)]
struct Expected {
    debug_success: bool,
    divergence_after_debug: bool,
    runtime_error: bool,
    divergence_after_runtime: bool,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    schema: String,
    name: String,
    builder: String,
    debug_input: String,
    runtime_input: String,
    mode: u8,
    tape: Option<Vec<Decision>>,
    actions: Vec<Action>,
    expected: Expected,
}

fn literals(case: &Case) -> std::result::Result<(), &'static str> {
    if case.schema != "gomad.rust-model-default-cut/v1"
        || case.name != "dormant-predecision-before-any-snapshot"
        || case.builder != "model::tests::model"
        || case.debug_input != "model::tests::debug_predecision"
        || case.runtime_input != "model::tests::decision"
        || case.mode != 2
        || case.tape.as_ref().map(Vec::len) != Some(0)
        || case.actions
            != [
                Action::Bootstrap,
                Action::DebugPredecision,
                Action::RuntimeDecision,
            ]
    {
        return Err("exact no-snapshot setup/action sequence differs");
    }
    if !case.expected.debug_success
        || case.expected.divergence_after_debug
        || !case.expected.runtime_error
        || !case.expected.divergence_after_runtime
    {
        return Err("exact original stage observations differ");
    }
    Ok(())
}

#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn fixture_bytes() -> Vec<u8> {
    let directory = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/model_default_cut");
    let mut names: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    names.sort();
    assert_eq!(
        names,
        ["dormant-predecision-before-any-snapshot.json"],
        "exact defaultCut corpus roster"
    );
    std::fs::read(directory.join(&names[0])).unwrap()
}

#[test]
fn rust_model_default_cut_corpus() {
    let case: Case = serde_json::from_slice(&fixture_bytes()).unwrap();
    literals(&case).unwrap();
    let mut replay = model();
    replay.config.mode = case.mode;
    replay.config.tape = case.tape;
    bootstrap(&mut replay);
    let debug_success = replay.debug_exchange(&mut debug_predecision()).is_ok();
    assert_eq!(debug_success, case.expected.debug_success);
    let divergence_after_debug = replay.divergence.is_some();
    assert_eq!(divergence_after_debug, case.expected.divergence_after_debug);
    let runtime_error = replay.runtime("runtime_decision", decision()).is_err();
    assert_eq!(runtime_error, case.expected.runtime_error);
    let divergence_after_runtime = replay.divergence.is_some();
    assert_eq!(
        divergence_after_runtime,
        case.expected.divergence_after_runtime
    );
    assert_eq!(
        Expected {
            debug_success,
            divergence_after_debug,
            runtime_error,
            divergence_after_runtime
        },
        case.expected
    );
}
