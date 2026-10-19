use super::*;
use serde::Deserialize;
use std::path::Path;

const NAME: &str = "fragmented-output-and-sticky-routing";
const OUTPUT_ERROR: &str = "cluster frame is not a JSON object";
const ROUTING: &str = "cluster frame is not addressed to the waiting instance";
const EOF: &str = "unexpected cluster protocol EOF";
const TAGGED: &str = "{\"instance\":\"node-a\",\"type\":\"call\",\"id\":1}\n{\"instance\":\"node-a\",\"type\":\"x\"}\n";
const REPLY: &str = "{\"type\":\"reply\",\"id\":1}\n";
const INPUT: &str = "{\"instance\":\"node-a\",\"type\":\"reply\",\"id\":1}\n{\"instance\":\"node-b\",\"type\":\"reply\"}\n";

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Target {
    instance: String,
    incarnation: u64,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct WriteStep {
    chunk: String,
    error: Option<String>,
    cumulative: String,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct ReadStep {
    frame: Option<String>,
    error: Option<String>,
    broken: Option<String>,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    schema: String,
    name: String,
    target: Target,
    writes: Vec<WriteStep>,
    input: String,
    reads: Vec<ReadStep>,
}

fn literals(case: &Case) -> Result<(), &'static str> {
    if case.schema != "gomad.rust-cluster-transport-trace/v1" || case.name != NAME {
        return Err("trace schema/name differs");
    }
    if case.target.instance != "node-a" || case.target.incarnation != 1 {
        return Err("trace target differs from original node-a/incarnation1");
    }
    if case.writes.len() != 3 || case.reads.len() != 3 || case.input != INPUT {
        return Err("trace step counts or complete input stream differ");
    }
    let writes = [
        ("{\"type\":\"call\",", None, ""),
        ("\"id\":1}\n{\"type\":\"x\"}\n", None, TAGGED),
        ("[]\n", Some(OUTPUT_ERROR), TAGGED),
    ];
    for (step, (chunk, error, cumulative)) in case.writes.iter().zip(writes) {
        if step.chunk != chunk || step.error.as_deref() != error || step.cumulative != cumulative {
            return Err("trace fragment, write result or cumulative raw bytes differ");
        }
    }
    let reads = [
        (Some(REPLY), None, None),
        (None, Some(ROUTING), Some(ROUTING)),
        (None, Some(EOF), Some(ROUTING)),
    ];
    for (step, (frame, error, broken)) in case.reads.iter().zip(reads) {
        if step.frame.as_deref() != frame
            || step.error.as_deref() != error
            || step.broken.as_deref() != broken
        {
            return Err("trace reply, read error stage or first-wins Broken differs");
        }
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

#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn read_case() -> Case {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/cluster_transport_trace");
    let mut names: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    names.sort();
    assert_eq!(
        names,
        vec![format!("{NAME}.json")],
        "exact Rust-only trace roster"
    );
    serde_json::from_slice(&std::fs::read(directory.join(&names[0])).unwrap()).unwrap()
}

#[test]
fn rust_cluster_transport_io_trace_corpus() {
    let case = read_case();
    literals(&case).unwrap();
    let capture = Capture::default();
    let shared: SharedOutput = Arc::new(Mutex::new(Box::new(capture.clone())));
    let mut output = TaggedOutput::new(shared, &case.target.instance);
    for (index, step) in case.writes.iter().enumerate() {
        let result = output.write_all(step.chunk.as_bytes());
        match &step.error {
            None => result.unwrap(),
            Some(message) => assert_eq!(result.unwrap_err().to_string(), *message),
        }
        assert_eq!(
            lock(&capture.0).as_slice(),
            step.cumulative.as_bytes(),
            "write step {index}"
        );
    }
    let shared: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(
        case.input.as_bytes().to_vec(),
    ))));
    let broken = Broken::default();
    let mut input = TaggedInput::new(shared, &case.target.instance, broken.clone());
    for (index, step) in case.reads.iter().enumerate() {
        let result = crate::protocol::read_frame(&mut input);
        match (&step.frame, &step.error) {
            (Some(frame), None) => {
                assert_eq!(result.unwrap(), frame.as_bytes(), "read step {index}");
            }
            (None, Some(message)) => {
                assert_eq!(
                    result.unwrap_err().to_string(),
                    *message,
                    "read step {index}"
                );
            }
            _ => panic!("invalid trace read expectation at {index}"),
        }
        assert_eq!(
            broken.get().as_deref(),
            step.broken.as_deref(),
            "Broken after read {index}"
        );
    }
}

#[test]
fn trace_literals_reject_changed_chunks_bytes_and_error_stages() {
    let case = read_case();
    literals(&case).unwrap();
    for mutation in [
        "target",
        "incarnation",
        "schema",
        "name",
        "write-count",
        "read-count",
        "first-chunk",
        "second-chunk",
        "array-chunk",
        "early-output",
        "tagged-output",
        "array-output",
        "early-write-error",
        "array-error",
        "input",
        "reply",
        "reply-error",
        "routing-error",
        "eof-error",
        "early-broken",
        "routing-broken",
        "sticky-broken",
    ] {
        let mut changed = case.clone();
        match mutation {
            "target" => changed.target.instance = "node-b".into(),
            "incarnation" => changed.target.incarnation = 2,
            "schema" => changed.schema.clear(),
            "name" => changed.name.clear(),
            "write-count" => {
                changed.writes.pop();
            }
            "read-count" => {
                changed.reads.pop();
            }
            "first-chunk" => changed.writes[0].chunk.push('x'),
            "second-chunk" => changed.writes[1].chunk.clear(),
            "array-chunk" => changed.writes[2].chunk = "{}\n".into(),
            "early-output" => changed.writes[0].cumulative = TAGGED.into(),
            "tagged-output" => changed.writes[1].cumulative = TAGGED.replace("node-a", "node-b"),
            "array-output" => changed.writes[2].cumulative.clear(),
            "early-write-error" => changed.writes[0].error = Some(OUTPUT_ERROR.into()),
            "array-error" => changed.writes[2].error = Some(ROUTING.into()),
            "input" => changed.input = INPUT.replace("node-b", "node-a"),
            "reply" => changed.reads[0].frame = Some(REPLY.replace('1', "2")),
            "reply-error" => changed.reads[0].error = Some(ROUTING.into()),
            "routing-error" => changed.reads[1].error = Some(EOF.into()),
            "eof-error" => changed.reads[2].error = Some(ROUTING.into()),
            "early-broken" => changed.reads[0].broken = Some(ROUTING.into()),
            "routing-broken" => changed.reads[1].broken = None,
            "sticky-broken" => changed.reads[2].broken = Some(EOF.into()),
            _ => unreachable!(),
        }
        assert!(
            literals(&changed).is_err(),
            "changed trace {mutation} accepted"
        );
    }
}
