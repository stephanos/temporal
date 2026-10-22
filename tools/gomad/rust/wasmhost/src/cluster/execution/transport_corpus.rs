use super::*;
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::path::Path;

const NAME: &str = "restarted-a-incarnation-two-rejects-old-prefix";
const SCHEMA: &str = "gomad.cluster-transport-codec-corpus/v1";
const ROUTING_ERROR: &str = "cluster frame is not addressed to the waiting instance";

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Target {
    node: String,
    incarnation: u64,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct GoError {
    kind: String,
    dimension: String,
    message: String,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Expected {
    tagged_result: String,
    go_attribution_error: GoError,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    schema: String,
    name: String,
    go_implementation_sha256: String,
    target: Target,
    result_payload: String,
    incoming_frame: String,
    expected: Expected,
}
fn go_implementation() -> String {
    let source = std::fs::read(
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/reference/cluster.go.txt"),
    )
    .unwrap();
    format!("{:x}", Sha256::digest(source))
}
fn literals(case: &Case) -> Result<(), &'static str> {
    if case.schema != SCHEMA
        || case.name != NAME
        || case.go_implementation_sha256 != go_implementation()
    {
        return Err("codec schema/name/Go implementation differs");
    }
    if case.target.node != "a" || case.target.incarnation != 2 {
        return Err("codec target differs from original a/incarnation2");
    }
    if case.result_payload != "{\"type\":\"result\"}\n" {
        return Err("codec result payload differs from original minimal result");
    }
    if case.incoming_frame != "{\"instance\":\"a\",\"type\":\"reply\"}\n" {
        return Err("codec old tagged reply differs");
    }
    if case.expected.tagged_result != "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"result\"}\n"
    {
        return Err("codec raw tagged result differs");
    }
    let error = &case.expected.go_attribution_error;
    if error.kind != "invalid"
        || error.dimension != "unattributable cluster frame"
        || error.message != "WASI invalid: unattributable cluster frame"
    {
        return Err("codec Go attribution-stage error differs");
    }
    Ok(())
}
#[derive(Clone, Default)]
struct Capture(Arc<Mutex<Vec<u8>>>);
impl Write for Capture {
    fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
        lock(&self.0).extend_from_slice(bytes);
        Ok(bytes.len())
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}
fn reader(frame: &str, target: &Target) -> (TaggedInput, Broken) {
    let shared: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(
        frame.as_bytes().to_vec(),
    ))));
    let broken = Broken::default();
    (
        TaggedInput::incarnation(shared, &target.node, target.incarnation, broken.clone()),
        broken,
    )
}
fn is_stale_attribution(error: &wasmtime::Error, broken: &Broken) -> bool {
    error.to_string() == ROUTING_ERROR && broken.get().as_deref() == Some(ROUTING_ERROR)
}
#[test]
#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn go_cluster_transport_codec_corpus() {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/cluster_transport_codec");
    let mut names: Vec<_> = std::fs::read_dir(&directory)
        .expect("generate shared codec JSON with TestClusterTransportCodecCorpusIsCurrent")
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    names.sort();
    assert_eq!(
        names,
        vec![format!("{NAME}.json")],
        "exact one-case codec roster"
    );
    let case: Case =
        serde_json::from_slice(&std::fs::read(directory.join(&names[0])).unwrap()).unwrap();
    literals(&case).unwrap();
    let capture = Capture::default();
    let shared: SharedOutput = Arc::new(Mutex::new(Box::new(capture.clone())));
    let mut output = TaggedOutput::incarnation(shared, &case.target.node, case.target.incarnation);
    output.write_all(case.result_payload.as_bytes()).unwrap();
    assert_eq!(
        lock(&capture.0).as_slice(),
        case.expected.tagged_result.as_bytes(),
        "complete original raw minimal RESULT2 bytes"
    );
    let (mut input, broken) = reader(&case.incoming_frame, &case.target);
    let result = crate::protocol::read_frame(&mut input);
    assert!(result.is_err(), "original stale read must fail");
    assert!(broken.get().is_some(), "original Broken marker must be set");
    let error = result.unwrap_err();
    assert!(
        is_stale_attribution(&error, &broken),
        "not exact routing-stage refusal: {error}"
    );
}
fn template() -> Case {
    Case {
        schema: SCHEMA.into(),
        name: NAME.into(),
        go_implementation_sha256: go_implementation(),
        target: Target {
            node: "a".into(),
            incarnation: 2,
        },
        result_payload: "{\"type\":\"result\"}\n".into(),
        incoming_frame: "{\"instance\":\"a\",\"type\":\"reply\"}\n".into(),
        expected: Expected {
            tagged_result: "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"result\"}\n".into(),
            go_attribution_error: GoError {
                kind: "invalid".into(),
                dimension: "unattributable cluster frame".into(),
                message: "WASI invalid: unattributable cluster frame".into(),
            },
        },
    }
}
#[test]
fn codec_literal_stage_guards_reject_target_kind_and_nonrouting_errors() {
    let case = template();
    literals(&case).unwrap();
    let mut target = case.clone();
    target.target.node = "b".into();
    assert!(literals(&target).is_err());
    let mut kind = case.clone();
    kind.expected.tagged_result = kind
        .expected
        .tagged_result
        .replace("\"result\"", "\"call\"");
    assert!(literals(&kind).is_err());
    let mut payload = case.clone();
    payload.result_payload = "{\"type\":\"call\"}\n".into();
    assert!(literals(&payload).is_err());
    let current = "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"reply\"}\n";
    let (mut input, broken) = reader(current, &case.target);
    assert_eq!(
        crate::protocol::read_frame(&mut input).unwrap(),
        b"{\"type\":\"reply\"}\n"
    );
    assert!(
        broken.get().is_none(),
        "current tag must not poison transport"
    );
    let mut current_case = case.clone();
    current_case.incoming_frame = current.into();
    assert!(
        literals(&current_case).is_err(),
        "current tag cannot replace original stale input"
    );
    let (mut input, broken) = reader("{\"instance\":\"a\",\"incarnation\":2,", &case.target);
    let eof = crate::protocol::read_frame(&mut input).unwrap_err();
    assert!(
        !is_stale_attribution(&eof, &broken),
        "EOF/non-routing failure must not count as stale attribution"
    );
}
