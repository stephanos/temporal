use super::*;
use serde::Deserialize;

const MODEL: &str = r#"{"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1024}],"limits":{"PendingOperations":10,"Transitions":10}}"#;
const IDENTITY: &str = "sha256:a9ec2d754ec09e513f6d0580634e16cc9ab4c21fa9353f2d3d91cea0ced251c3";
const BODY_SHA256: &str = "aea3c5b80a202b6649d69bed5b2bb5832a0c0cdbe65b29acc039a0e1e9313f2c";

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct ContextSequence {
    schema: String,
    name: String,
    descriptor_json: String,
    input_file: String,
    input_bytes: usize,
    input_sha256: String,
    context_bytes: u64,
    minimum_context_bytes_exclusive: u64,
    reads: Vec<ContextRead>,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct ContextRead {
    input_prefix_bytes: usize,
    maximum: u64,
    expected: serde_json::Value,
}

fn context_fixture() -> (ContextSequence, Vec<u8>) {
    let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("tests/testdata/storage_transport_vectors");
    let case: ContextSequence = serde_json::from_slice(
        &std::fs::read(directory.join("pending-context-over-one-mib.json")).unwrap(),
    )
    .unwrap();
    let input = std::fs::read(directory.join(&case.input_file)).unwrap();
    (case, input)
}

fn replay_context(case: &ContextSequence, input: &[u8]) -> std::result::Result<(), String> {
    if case.schema != "gomad.rust-storage-context-input-sequence/v1"
        || case.name.is_empty()
        || case.input_bytes != input.len()
        || case.input_sha256 != format!("{:x}", Sha256::digest(input))
        || case.minimum_context_bytes_exclusive != 1 << 20
        || case.context_bytes <= case.minimum_context_bytes_exclusive
        || case.reads.len() != 3
    {
        return Err("storage context sequence identity differs".into());
    }
    let mut descriptor: StorageDescriptor =
        serde_json::from_str(&case.descriptor_json).map_err(|error| error.to_string())?;
    descriptor.validate().map_err(|error| error.to_string())?;
    if descriptor
        .crash_context
        .as_ref()
        .map(|context| context.bytes)
        != Some(case.context_bytes)
    {
        return Err("storage context byte count differs".into());
    }
    for step in &case.reads {
        let prefix = input
            .get(..step.input_prefix_bytes)
            .ok_or("storage context input prefix exceeds fixture")?;
        let actual = if let Err(error) =
            descriptor.read_context(&mut std::io::Cursor::new(prefix), step.maximum)
        {
            serde_json::json!({"status":"error", "message":error.to_string()})
        } else {
            let counts = descriptor.observed_context.as_ref().map(|contexts| {
                contexts
                    .iter()
                    .map(|(name, context)| {
                        (
                            name,
                            context.operations.as_deref().unwrap_or_default().len(),
                        )
                    })
                    .collect::<BTreeMap<_, _>>()
            });
            serde_json::json!({"status":"ok", "context_operation_counts":counts})
        };
        if actual != step.expected {
            return Err(format!(
                "{}: storage context read differs: {actual}",
                case.name
            ));
        }
    }
    Ok(())
}

#[test]
fn storage_context_sequence_rejects_changed_outcomes_and_inputs() {
    let (case, input) = context_fixture();
    replay_context(&case, &input).unwrap();
    for mutation in ["maximum", "end", "count", "size", "descriptor", "wire"] {
        let mut changed = case.clone();
        let mut wire = input.clone();
        match mutation {
            "maximum" => changed.reads[0].maximum = 18_892_800,
            "end" => changed.reads[2].input_prefix_bytes = changed.reads[1].input_prefix_bytes,
            "count" => changed.reads[2].expected["context_operation_counts"]["data"] = 399.into(),
            "size" => changed.context_bytes = 1 << 20,
            "descriptor" => changed.descriptor_json = "{}".into(),
            "wire" => {
                wire.pop();
            }
            _ => unreachable!(),
        }
        assert!(
            replay_context(&changed, &wire).is_err(),
            "changed {mutation} accepted"
        );
    }
}

#[test]
fn storage_vector_guards_reject_changed_original_contracts() {
    let source = fixture_bytes();
    let corpus: Corpus = serde_json::from_slice(&source).unwrap();
    literals(&corpus).unwrap();
    verify_outputs(&corpus).unwrap();
    let mut changed = corpus.clone();
    changed.descriptors[3].expected = Expected::Validate {
        message: "other".into(),
    };
    assert!(
        verify_outputs(&changed).is_err(),
        "changed descriptor outcome accepted"
    );
    let mut changed = corpus.clone();
    changed.descriptors[4].expected = Expected::Validate {
        message: "unknown field `extra`".into(),
    };
    assert!(
        verify_outputs(&changed).is_err(),
        "parse refusal accepted as validation"
    );
    let mut changed = corpus.clone();
    changed.envelope.expected.body_sha256 = "0".repeat(64);
    assert!(
        verify_outputs(&changed).is_err(),
        "changed envelope digest accepted"
    );
    for (pointer, value) in [
        ("/schema", serde_json::json!("other")),
        ("/model_json", serde_json::json!("{}")),
        ("/envelope/log", serde_json::json!("x")),
        ("/descriptors/0/descriptor_json", serde_json::json!("{}")),
    ] {
        let mut changed: serde_json::Value = serde_json::from_slice(&source).unwrap();
        *changed.pointer_mut(pointer).unwrap() = value;
        assert!(literals(&serde_json::from_value::<Corpus>(changed).unwrap()).is_err());
    }
    let mut changed = corpus.clone();
    changed.descriptors.swap(0, 1);
    assert!(literals(&changed).is_err());
    let mut changed = corpus;
    changed.descriptors.pop();
    assert!(literals(&changed).is_err());
    let mut unknown: serde_json::Value = serde_json::from_slice(&source).unwrap();
    unknown["envelope"]["extra"] = serde_json::json!(1);
    assert!(serde_json::from_value::<Corpus>(unknown).is_err());
    let mut trailing = source;
    trailing.extend_from_slice(b" {}");
    assert!(serde_json::from_slice::<Corpus>(&trailing).is_err());
}

#[derive(Clone, Deserialize, PartialEq, Eq)]
#[serde(tag = "stage", rename_all = "snake_case", deny_unknown_fields)]
enum Expected {
    Valid,
    Validate { message: String },
    Parse { prefix: String },
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct DescriptorCase {
    name: String,
    descriptor_json: String,
    expected: Expected,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct EnvelopeExpected {
    magic: Vec<u8>,
    header: Vec<u8>,
    body_bytes: u64,
    total_bytes: u64,
    body_sha256: String,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Envelope {
    name: String,
    snapshots: String,
    operations: String,
    enumerations: String,
    image_identity: String,
    log: String,
    expected: EnvelopeExpected,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    schema: String,
    model_json: String,
    model_identity: String,
    envelope: Envelope,
    descriptors: Vec<DescriptorCase>,
}

fn descriptor_inputs() -> Vec<String> {
    vec![
        format!(r#"{{"model":{MODEL},"model_identity":"{IDENTITY}","return_image":true}}"#),
        format!(
            r#"{{"model":{MODEL},"model_identity":"sha256:{}","return_image":true}}"#,
            "0".repeat(64)
        ),
        format!(
            r#"{{"model":{MODEL},"model_identity":"{IDENTITY}","return_image":false,"image":{{"chunks":2,"bytes":10,"sha256":"{}"}}}}"#,
            "0".repeat(64)
        ),
        format!(
            r#"{{"model":{MODEL},"model_identity":"{IDENTITY}","return_image":true,"crash_selection":{{"other":[1]}}}}"#
        ),
        format!(r#"{{"model":{MODEL},"model_identity":"x","return_image":true,"extra":1}}"#),
    ]
}

fn literals(c: &Corpus) -> std::result::Result<(), String> {
    let e = &c.envelope;
    if c.schema != "gomad.rust-storage-transport-vectors/v1"
        || c.model_json != MODEL
        || c.model_identity != IDENTITY
        || e.name != "original-v1-layout"
        || e.snapshots != "[]"
        || e.operations != "{}"
        || e.enumerations != "null"
        || e.image_identity != "a".repeat(64)
        || e.log != "x\n"
        || e.expected.magic.as_slice() != b"GMSTORE\x01"
        || e.expected.header != [1, 0, 0, 0, 5, 0, 0, 0]
        || e.expected.body_bytes != 188
        || e.expected.total_bytes != 228
        || e.expected.body_sha256 != BODY_SHA256
    {
        return Err("original model/envelope literals differ".into());
    }
    let names = [
        "valid-identity",
        "wrong-identity",
        "invalid-framed-image",
        "unknown-selection-volume",
        "unknown-extra-field",
    ];
    let expected = [
        Expected::Valid,
        Expected::Validate {
            message: "storage model identity mismatch".into(),
        },
        Expected::Validate {
            message: "invalid storage image descriptor".into(),
        },
        Expected::Validate {
            message: "WASI invalid: storage crash selection volume".into(),
        },
        Expected::Parse {
            prefix: "unknown field `extra`".into(),
        },
    ];
    if c.descriptors.len() != 5 {
        return Err("descriptor roster differs".into());
    }
    for (index, input) in descriptor_inputs().iter().enumerate() {
        let case = &c.descriptors[index];
        if case.name != names[index]
            || case.descriptor_json != *input
            || case.expected != expected[index]
        {
            return Err("original descriptor literal differs".into());
        }
    }
    Ok(())
}

fn verify_outputs(c: &Corpus) -> std::result::Result<(), String> {
    let model: StorageModel = serde_json::from_str(&c.model_json).map_err(|e| e.to_string())?;
    if model.go_json() != c.model_json || model.identity() != c.model_identity {
        return Err("Go model encoding/identity differs".into());
    }
    let e = &c.envelope;
    let bytes = encode_state(
        e.snapshots.as_bytes(),
        e.operations.as_bytes(),
        e.enumerations.as_bytes(),
        &e.image_identity,
        e.log.as_bytes(),
    );
    if bytes.len() < 40 {
        return Err("short state envelope".into());
    }
    let body = bytes.len() - 40;
    let digest = Sha256::digest(&bytes[..body]);
    if bytes.len() as u64 != e.expected.total_bytes
        || body as u64 != e.expected.body_bytes
        || &bytes[..8] != e.expected.magic.as_slice()
        || &bytes[8..16] != e.expected.header.as_slice()
        || u64::from_le_bytes(bytes[body..body + 8].try_into().unwrap()) != body as u64
        || bytes[body + 8..] != digest[..]
        || format!("{digest:x}") != e.expected.body_sha256
    {
        return Err("complete original envelope layout/trailer differs".into());
    }
    for case in &c.descriptors {
        let matches = match serde_json::from_str::<StorageDescriptor>(&case.descriptor_json) {
            Err(error) => {
                matches!(&case.expected, Expected::Parse { prefix } if error.to_string().starts_with(prefix))
            }
            Ok(descriptor) => match (descriptor.validate(), &case.expected) {
                (Ok(()), Expected::Valid) => true,
                (Err(error), Expected::Validate { message }) => error.to_string() == *message,
                _ => false,
            },
        };
        if !matches {
            return Err(format!("{}: descriptor stage/output differs", case.name));
        }
    }
    Ok(())
}

fn fixture_bytes() -> Vec<u8> {
    std::fs::read(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("tests/testdata/storage_transport_vectors/original-envelope-descriptors.json"),
    )
    .unwrap()
}

#[test]
fn rust_storage_transport_vectors_corpus() {
    let (sequence, input) = context_fixture();
    replay_context(&sequence, &input).unwrap();
    let corpus: Corpus = serde_json::from_slice(&fixture_bytes()).unwrap();
    literals(&corpus).unwrap();
    verify_outputs(&corpus).unwrap();
}
