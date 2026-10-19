use super::*;
use serde::Deserialize;
use std::path::Path;

const NAME: &str = "unterminated-current-incarnation-reply";
const EOF: &str = "unexpected cluster protocol EOF";

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct ReadSequence {
    schema: String,
    name: String,
    target: Target,
    input_file: String,
    input_bytes: usize,
    input_sha256: String,
    reads: Vec<SequenceRead>,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct SequenceRead {
    reader: String,
    expected: serde_json::Value,
}

fn sequence_fixture() -> (ReadSequence, Vec<u8>) {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/cluster_transport_input");
    let case: ReadSequence = serde_json::from_slice(
        &std::fs::read(directory.join("oversized-tagged-frame-followup.json")).unwrap(),
    )
    .unwrap();
    let input = std::fs::read(directory.join(&case.input_file)).unwrap();
    (case, input)
}

fn replay_sequence(case: &ReadSequence, input: &[u8]) -> Result<(), String> {
    use sha2::{Digest, Sha256};

    if case.schema != "gomad.rust-cluster-transport-input-sequence/v1"
        || case.name.is_empty()
        || case.input_bytes != input.len()
        || case.input_sha256 != format!("{:x}", Sha256::digest(input))
        || case.reads.len() != 2
    {
        return Err("cluster input sequence identity differs".into());
    }
    let shared: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(input.to_vec()))));
    let broken = Broken::default();
    for step in &case.reads {
        if step.reader != "fresh" {
            return Err("unsupported cluster reader lifecycle".into());
        }
        let mut reader = TaggedInput::incarnation(
            shared.clone(),
            &case.target.instance,
            case.target.incarnation,
            broken.clone(),
        );
        let outcome = match crate::protocol::read_frame(&mut reader) {
            Ok(bytes) => serde_json::json!({"status":"frame", "bytes":bytes}),
            Err(error) => serde_json::json!({"status":"error",
                "classification":crate::protocol::classification(&error),
                "message":error.to_string()}),
        };
        let actual = serde_json::json!({"outcome":outcome, "broken":broken.get()});
        if actual != step.expected {
            return Err(format!("{}: cluster read differs: {actual}", case.name));
        }
    }
    Ok(())
}

#[test]
fn cluster_input_sequence_rejects_changed_observations_and_wire() {
    let (case, input) = sequence_fixture();
    replay_sequence(&case, &input).unwrap();
    for mutation in [
        "classification",
        "broken",
        "frame",
        "reader",
        "target",
        "input",
    ] {
        let mut changed = case.clone();
        let mut wire = input.clone();
        match mutation {
            "classification" => {
                changed.reads[0].expected["outcome"]["classification"] = "invalid".into();
            }
            "broken" => changed.reads[0].expected["broken"] = "poisoned".into(),
            "frame" => {
                changed.reads[1].expected["outcome"]["bytes"]
                    .as_array_mut()
                    .unwrap()
                    .pop();
            }
            "reader" => changed.reads[1].reader = "reuse".into(),
            "target" => changed.target.instance = "b".into(),
            "input" => {
                wire.pop();
            }
            _ => unreachable!(),
        }
        assert!(
            replay_sequence(&changed, &wire).is_err(),
            "changed {mutation} accepted"
        );
    }
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Target {
    instance: String,
    incarnation: u64,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Expected {
    error: String,
    broken: String,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    schema: String,
    name: String,
    target: Target,
    input: String,
    expected: Expected,
}

fn literals(case: &Case) -> Result<(), &'static str> {
    if case.schema != "gomad.rust-cluster-transport-input-corpus/v1" || case.name != NAME {
        return Err("EOF schema/name differs");
    }
    if case.target.instance != "a"
        || case.target.incarnation != 2
        || case.input != "{\"instance\":\"a\",\"incarnation\":2,"
    {
        return Err("EOF requires exact unterminated current-incarnation prefix");
    }
    if case.expected.error != EOF || case.expected.broken != EOF {
        return Err("EOF requires exact error AND Broken observation");
    }
    Ok(())
}

fn read(input: &str, target: &Target) -> (wasmtime::Result<Vec<u8>>, Option<String>) {
    let shared: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(
        input.as_bytes().to_vec(),
    ))));
    let broken = Broken::default();
    let mut input =
        TaggedInput::incarnation(shared, &target.instance, target.incarnation, broken.clone());
    (crate::protocol::read_frame(&mut input), broken.get())
}

fn is_eof(error: &wasmtime::Error, broken: Option<&str>) -> bool {
    error.to_string() == EOF && broken == Some(EOF)
}

#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn read_case() -> Case {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/cluster_transport_input");
    let mut names: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    names.sort();
    assert_eq!(
        names,
        vec![
            "oversized-tagged-frame-followup.json".to_owned(),
            format!("{NAME}.json")
        ],
        "exact Rust-only input corpus roster"
    );
    serde_json::from_slice(&std::fs::read(directory.join(format!("{NAME}.json"))).unwrap()).unwrap()
}

#[test]
fn rust_cluster_input_eof_corpus() {
    let (sequence, input) = sequence_fixture();
    replay_sequence(&sequence, &input).unwrap();
    let case = read_case();
    literals(&case).unwrap();
    let (result, broken) = read(&case.input, &case.target);
    let error = result.unwrap_err();
    assert_eq!(error.to_string(), case.expected.error);
    assert_eq!(broken.as_deref(), Some(case.expected.broken.as_str()));
    assert!(
        is_eof(&error, broken.as_deref()),
        "another refusal cannot replace actual EOF"
    );
}

#[test]
fn eof_literals_and_stage_controls_reject_stale_or_completed_input() {
    let case = read_case();
    literals(&case).unwrap();
    for mutation in [
        "target",
        "incarnation",
        "complete-input",
        "stale-input",
        "error",
        "broken",
    ] {
        let mut changed = case.clone();
        match mutation {
            "target" => changed.target.instance = "b".into(),
            "incarnation" => changed.target.incarnation = 1,
            "complete-input" => {
                changed.input =
                    "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"reply\"}\n".into();
            }
            "stale-input" => changed.input = "{\"instance\":\"a\",\"type\":\"reply\"}\n".into(),
            "error" => {
                changed.expected.error =
                    "cluster frame is not addressed to the waiting instance".into();
            }
            "broken" => changed.expected.broken.clear(),
            _ => unreachable!(),
        }
        assert!(
            literals(&changed).is_err(),
            "changed EOF {mutation} accepted"
        );
    }
    let (current, broken) = read(
        "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"reply\"}\n",
        &case.target,
    );
    assert_eq!(current.unwrap(), b"{\"type\":\"reply\"}\n");
    assert!(
        broken.is_none(),
        "complete current frame must not poison transport"
    );
    let (stale, broken) = read("{\"instance\":\"a\",\"type\":\"reply\"}\n", &case.target);
    let error = stale.unwrap_err();
    assert_eq!(
        error.to_string(),
        "cluster frame is not addressed to the waiting instance"
    );
    assert_eq!(
        broken.as_deref(),
        Some("cluster frame is not addressed to the waiting instance")
    );
    assert!(
        !is_eof(&error, broken.as_deref()),
        "stale routing failure counted as EOF"
    );
}
