use super::*;
use serde::Deserialize;
use std::path::Path;

#[test]
fn crash_failure_literals_reject_changed_original_tuples() {
    let source = fixture_bytes();
    let corpus: Corpus = serde_json::from_slice(&source).unwrap();
    literals(&corpus).unwrap();
    for (pointer, replacement) in [
        ("/schema", serde_json::json!("other")),
        ("/cases/0/name", serde_json::json!("other")),
        ("/cases/0/error/resource", serde_json::json!("volume-bytes")),
        ("/cases/0/error/required", serde_json::json!(4)),
        ("/cases/0/error/maximum", serde_json::json!(3)),
        (
            "/cases/0/expected/kind",
            serde_json::json!("infrastructure"),
        ),
        ("/cases/0/expected/message", serde_json::json!("other")),
        ("/cases/1/name", serde_json::json!("other")),
        ("/cases/1/error/message", serde_json::json!("other")),
        ("/cases/1/expected/kind", serde_json::json!("capacity")),
        ("/cases/1/expected/message", serde_json::json!("other")),
        ("/cases/1/error/type", serde_json::json!("capacity")),
        (
            "/cases/0/error",
            serde_json::json!({"type": "invalid", "message": "crash selection contains an unknown operation"}),
        ),
        (
            "/cases/1/error",
            serde_json::json!({"type": "capacity", "resource": "image-bytes", "required": 3, "maximum": 2}),
        ),
    ] {
        let mut changed: serde_json::Value = serde_json::from_slice(&source).unwrap();
        *changed.pointer_mut(pointer).unwrap() = replacement;
        if let Ok(parsed) = serde_json::from_value::<Corpus>(changed) {
            assert!(
                literals(&parsed).is_err(),
                "changed tuple accepted: {pointer}"
            );
        }
    }
    for shape in ["missing", "extra", "reordered"] {
        let mut changed: Corpus = serde_json::from_slice(&source).unwrap();
        match shape {
            "missing" => {
                changed.cases.pop();
            }
            "extra" => changed.cases.push(changed.cases[0].clone()),
            "reordered" => changed.cases.swap(0, 1),
            _ => unreachable!(),
        }
        assert!(
            literals(&changed).is_err(),
            "changed roster accepted: {shape}"
        );
    }
    let mut unknown: serde_json::Value = serde_json::from_slice(&source).unwrap();
    unknown["cases"][0]["error"]["unknown"] = serde_json::json!(true);
    assert!(serde_json::from_value::<Corpus>(unknown).is_err());
    let mut trailing = source;
    trailing.extend_from_slice(b" {}");
    assert!(serde_json::from_slice::<Corpus>(&trailing).is_err());
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Expected {
    kind: String,
    message: String,
}

#[derive(Clone, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case", deny_unknown_fields)]
enum ErrorInput {
    Capacity {
        resource: String,
        required: u64,
        maximum: u64,
    },
    Invalid {
        message: String,
    },
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    error: ErrorInput,
    expected: Expected,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    schema: String,
    cases: Vec<Case>,
}

fn literals(corpus: &Corpus) -> std::result::Result<(), &'static str> {
    if corpus.schema != "gomad.rust-storage-crash-failure/v1" || corpus.cases.len() != 2 {
        return Err("mapper schema/roster differs");
    }
    let capacity = &corpus.cases[0];
    if capacity.name != "image-capacity-preserves-classification"
        || !matches!(&capacity.error, ErrorInput::Capacity { resource, required: 3, maximum: 2 } if resource == "image-bytes")
        || capacity.expected.kind != "capacity"
        || capacity.expected.message != "WASI capacity: storage-image-bytes"
    {
        return Err("original image capacity tuple differs");
    }
    let invalid = &corpus.cases[1];
    if invalid.name != "unknown-operation-preserves-infrastructure-message"
        || !matches!(&invalid.error, ErrorInput::Invalid { message } if message == "crash selection contains an unknown operation")
        || invalid.expected.kind != "infrastructure"
        || invalid.expected.message != "crash selection contains an unknown operation"
    {
        return Err("original infrastructure tuple differs");
    }
    Ok(())
}

#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn fixture_bytes() -> Vec<u8> {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/storage_crash_failure");
    let mut names: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    names.sort();
    assert_eq!(
        names,
        ["original-error-mapping.json"],
        "exact mapper corpus file roster"
    );
    std::fs::read(directory.join(&names[0])).unwrap()
}

#[test]
fn rust_storage_crash_failure_corpus() {
    let corpus: Corpus = serde_json::from_slice(&fixture_bytes()).unwrap();
    literals(&corpus).unwrap();
    for case in corpus.cases {
        let error = match case.error {
            ErrorInput::Capacity {
                required, maximum, ..
            } => volume::Error::Capacity {
                resource: "image-bytes",
                required,
                maximum,
            },
            ErrorInput::Invalid { .. } => {
                volume::Error::Invalid("crash selection contains an unknown operation")
            }
        };
        let actual = crash_failure(error);
        assert_eq!(
            actual,
            (case.expected.kind.as_str(), case.expected.message),
            "{}",
            case.name
        );
    }
}
