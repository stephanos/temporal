//! Helper transcript corpus (fn-187): every `testdata/helper_transcript/*.json`
//! case drives the release helper with its request, inputs and scripted
//! replies, runs it twice (ten times under regeneration), and must reproduce
//! its expected block exactly. `GOMAD_HELPER_CORPUS_REGENERATE=1` rewrites
//! expected blocks (and derived inputs) from the run; the reviewed corpus diff
//! is the approval. The Go `wasi` package replays the same files
//! (`TestHelperCorpus`).
//!
//! The default build compares `expected`. A diagnostic build
//! (`diagnostic-profile`, `diagnostic-fuel-off`, `guest-profile`) runs only
//! the cases listing it in `builds` and compares `expected` with the top-level
//! fields in `expected_builds[<build>]` replaced (absent when its output is
//! identical). The conformance build replaces the helper and does not run the
//! corpus.
#![cfg(not(feature = "conformance"))]
#[path = "common/residual_helper_literals.rs"]
mod residual_helper_literals;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::io::{BufRead, BufReader, Read, Write};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::mpsc;
use std::time::Duration;

#[path = "common/inspection.rs"]
mod inspection;

#[path = "common/network_inspection.rs"]
mod network_inspection;

#[path = "common/profile_stderr.rs"]
mod profile_stderr;

#[path = "common/namespace_corpus_literals.rs"]
mod namespace_corpus_literals;

#[path = "common/closed_cluster_guards.rs"]
mod closed_cluster_guards;

#[path = "common/model_corpus_literals.rs"]
mod model_corpus_literals;

#[path = "common/admission_corpus_literals.rs"]
mod admission_corpus_literals;

#[path = "common/write_corpus_literals.rs"]
mod write_corpus_literals;

#[path = "common/helper_corpus_inputs.rs"]
mod helper_corpus_inputs;

#[path = "common/helper_corpus_projection.rs"]
mod helper_corpus_projection;
use helper_corpus_projection::{project, split_profile};
#[path = "common/helper_feature_corpus.rs"]
mod helper_feature_corpus;

const SCHEMA: &str = "gomad.helper-transcript-corpus/v1";
const CASE_KEYS: [&str; 13] = [
    "schema",
    "intent",
    "replaces",
    "guest",
    "request",
    "inputs",
    "close_input",
    "replies",
    "reply_policy",
    "builds",
    "derived",
    "expected",
    "expected_builds",
];
/// The helper build under test; non-default builds run the cases that list it.
const BUILD: &str = if cfg!(all(
    feature = "guest-profile",
    feature = "diagnostic-fuel-off"
)) {
    "guest-profile-fuel-off"
} else if cfg!(feature = "diagnostic-fuel-off") {
    "diagnostic-fuel-off"
} else if cfg!(feature = "guest-profile") {
    "guest-profile"
} else if cfg!(feature = "diagnostic-profile") {
    "diagnostic-profile"
} else {
    "default"
};
const TIMEOUT: Duration = Duration::from_secs(120);

fn corpus_dir() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/helper_transcript")
}

/// Raw output of one helper run.
#[derive(Clone, PartialEq, Debug)]
struct Run {
    frames: Vec<Value>,
    raw: Vec<String>,
    stderr: String,
    exit_code: Option<i32>,
}

fn guest_bytes(guest: &Value) -> Result<Vec<u8>, String> {
    let object = guest.as_object().ok_or("guest must be an object")?;
    if object.len() != 1 {
        return Err("guest needs exactly one of wat, path, wat_parts".into());
    }
    if let Some(text) = object.get("wat") {
        return Ok(text.as_str().ok_or("guest.wat must be a string")?.into());
    }
    if let Some(path) = object.get("path") {
        let path = path.as_str().ok_or("guest.path must be a string")?;
        if path.starts_with('/') || path.split('/').any(|part| part == "..") {
            return Err(format!("guest.path {path} escapes the corpus directory"));
        }
        return std::fs::read(corpus_dir().join(path)).map_err(|error| format!("{path}: {error}"));
    }
    let parts = object
        .get("wat_parts")
        .and_then(Value::as_array)
        .ok_or("guest needs wat, path or wat_parts")?;
    let mut text = String::new();
    for part in parts {
        match part {
            Value::String(literal) => text.push_str(literal),
            Value::Object(repeat) if repeat.len() == 2 => {
                let count = repeat["repeat"].as_u64().ok_or("repeat must be a count")?;
                let piece = repeat["text"]
                    .as_str()
                    .ok_or("repeat text must be a string")?;
                text.push_str(&piece.repeat(
                    usize::try_from(count).map_err(|_| "repeat count exceeds host width")?,
                ));
            }
            _ => return Err(format!("invalid guest part {part}")),
        }
    }
    Ok(text.into_bytes())
}

/// Resolves `$guest_path`, `$guest_sha256` and `$trigger.<path>` strings.
fn substitute(
    value: &Value,
    guest: &(String, String),
    trigger: Option<&Value>,
) -> Result<Value, String> {
    Ok(match value {
        Value::String(text) if text.starts_with('$') => match text.as_str() {
            "$guest_path" => json!(guest.0),
            "$guest_sha256" => json!(guest.1),
            _ => {
                let path = text
                    .strip_prefix("$trigger.")
                    .ok_or_else(|| format!("unknown placeholder {text}"))?;
                let mut found = trigger.ok_or_else(|| format!("{text} outside a reply"))?;
                for key in path.split('.') {
                    found = found.get(key).ok_or_else(|| {
                        format!("{text} missing from trigger {}", trigger.unwrap())
                    })?;
                }
                found.clone()
            }
        },
        Value::Array(items) => Value::Array(
            items
                .iter()
                .map(|item| substitute(item, guest, trigger))
                .collect::<Result<_, _>>()?,
        ),
        Value::Object(fields) => {
            let mut out = Map::new();
            for (key, item) in fields {
                out.insert(key.clone(), substitute(item, guest, trigger)?);
            }
            Value::Object(out)
        }
        other => other.clone(),
    })
}

/// Pattern objects match recursively by subset; everything else exactly.
fn matches(pattern: &Value, frame: &Value) -> bool {
    match (pattern, frame) {
        (Value::Object(want), Value::Object(have)) => want
            .iter()
            .all(|(key, value)| have.get(key).is_some_and(|got| matches(value, got))),
        _ => pattern == frame,
    }
}

/// Encodes a sent frame; cluster frames carry their `instance` tag first, as
/// the helper's transport routes by that prefix.
fn line(frame: &Value) -> String {
    fn encode(value: &Value) -> String {
        match value {
            Value::Object(fields) if fields.len() == 1 && fields.contains_key("raw_json") => {
                fields["raw_json"].as_str().unwrap().to_owned()
            }
            Value::Object(fields) => format!(
                "{{{}}}",
                fields
                    .iter()
                    .map(|(key, value)| format!("{}:{}", json!(key), encode(value)))
                    .collect::<Vec<_>>()
                    .join(",")
            ),
            Value::Array(items) => format!(
                "[{}]",
                items.iter().map(encode).collect::<Vec<_>>().join(",")
            ),
            _ => value.to_string(),
        }
    }
    let Some(instance) = frame.get("instance") else {
        return encode(frame);
    };
    let mut rest = frame.clone();
    rest.as_object_mut().unwrap().remove("instance");
    let rest = encode(&rest);
    if rest == "{}" {
        return format!("{{\"instance\":{instance}}}");
    }
    format!("{{\"instance\":{instance},{}", &rest[1..])
}

fn write_line(input: &mut Option<std::process::ChildStdin>, frame: &Value) {
    // A helper that already refused may have closed its input; the refusal
    // is in its output, so a broken pipe is not a corpus failure.
    if let Some(stdin) = input {
        let result = if let Some(raw) = frame.get("raw_base64") {
            assert_eq!(
                frame.as_object().unwrap().len(),
                1,
                "raw corpus input has extra fields"
            );
            stdin.write_all(&STANDARD.decode(raw.as_str().unwrap()).unwrap())
        } else {
            writeln!(stdin, "{}", line(frame))
        };
        if result.is_err() {
            *input = None;
        }
    }
}

/// Byte length of every guest path. Helper refusals quote request columns,
/// so the path's length (not its text) would otherwise leak into output.
const GUEST_PATH_BYTES: usize = 160;

/// A fresh, unique guest path of exactly `GUEST_PATH_BYTES` under the temp
/// directory; the Go loader builds the same shape.
fn guest_path() -> Result<PathBuf, String> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let unique = u64::from(std::process::id()) * 1_000_000 + NEXT.fetch_add(1, Ordering::Relaxed);
    let base = std::env::temp_dir().join(format!("gomad-helper-corpus-{unique:020}-"));
    let base = base.to_str().ok_or("temp directory is not UTF-8")?;
    let padding = GUEST_PATH_BYTES
        .checked_sub(base.len() + "/guest.wat".len())
        .ok_or("temp directory too long for the fixed-length guest path")?;
    let directory = PathBuf::from(format!("{base}{}", "x".repeat(padding)));
    std::fs::create_dir(&directory).map_err(|error| error.to_string())?;
    Ok(directory.join("guest.wat"))
}

fn run_case(case: &Value) -> Result<Run, String> {
    let bytes = guest_bytes(&case["guest"])?;
    // A fresh path per run: output that names it varies and is refused.
    let path = guest_path()?;
    std::fs::write(&path, &bytes).map_err(|error| error.to_string())?;
    let guest = (
        path.to_str().unwrap().to_owned(),
        format!("{:x}", Sha256::digest(&bytes)),
    );
    let result = (|| {
        let prepared = closed_cluster_guards::assets(case, &path)?;
        let run = drive(&prepared, &guest)?;
        closed_cluster_guards::check(case, &run)?;
        helper_feature_corpus::check_raw_profile(case, &run)?;
        Ok(run)
    })();
    let _ = std::fs::remove_dir_all(path.parent().unwrap());
    result
}

fn parse_output_frame(line: &str) -> Result<Value, String> {
    let frame: Value = serde_json::from_str(line)
        .map_err(|error| format!("helper emitted non-JSON {line:?}: {error}"))?;
    if frame["type"] == "call" && matches!(frame["op"].as_str(), Some("fd_write" | "fd_pwrite")) {
        let number = |value: &Value| {
            value
                .as_u64()
                .ok_or("invalid unsigned write callback field")
        };
        let id = number(&frame["id"])?;
        let fd = number(&frame["input"]["fd"])?;
        if fd > u64::from(u32::MAX) {
            return Err("write callback fd exceeds u32".into());
        }
        let data = frame["input"]["data_base64"]
            .as_str()
            .ok_or("missing write callback bytes")?;
        let offset = if frame["op"] == "fd_pwrite" {
            format!(",\"offset\":{}", number(&frame["input"]["offset"])?)
        } else {
            String::new()
        };
        let state = if let Some(state) = frame.get("local_state") {
            let offered = if let Some(offered) = state.get("offered_output_bytes") {
                format!(",\"offered_output_bytes\":{}", number(offered)?)
            } else {
                String::new()
            };
            format!(
                ",\"local_state\":{{\"advances\":{},\"now\":{},\"transcript_bytes\":{}{offered}}}",
                number(&state["advances"])?,
                number(&state["now"])?,
                number(&state["transcript_bytes"])?
            )
        } else {
            String::new()
        };
        let mut instance = if let Some(instance) = frame.get("instance") {
            format!(
                "\"instance\":{},",
                json!(instance.as_str().ok_or("invalid callback instance")?)
            )
        } else {
            String::new()
        };
        if let Some(incarnation) = frame.get("incarnation") {
            let incarnation = number(incarnation)?;
            if instance.is_empty() || incarnation <= 1 {
                return Err("invalid restarted callback incarnation".into());
            }
            write!(instance, "\"incarnation\":{incarnation},").unwrap();
        }
        let expected = format!(
            "{{{instance}\"id\":{id},\"input\":{{\"data_base64\":{},\"fd\":{fd}{offset}}}{state},\"op\":{},\"type\":\"call\"}}",
            json!(data),
            frame["op"]
        );
        if line != expected {
            return Err("write callback raw literal key order or bytes differ".into());
        }
    }
    Ok(frame)
}

#[test]
fn write_callback_raw_lines_preserve_literal_key_order_and_bytes() {
    for literal in [
        r#"{"id":1,"input":{"data_base64":"AP8=","fd":13},"local_state":{"advances":0,"now":1,"transcript_bytes":0},"op":"fd_write","type":"call"}"#,
        r#"{"id":2,"input":{"data_base64":"","fd":4294967295,"offset":18446744073709551615},"op":"fd_pwrite","type":"call"}"#,
    ] {
        assert!(parse_output_frame(literal).is_ok(), "{literal}");
        let reordered = literal
            .replace("{\"id\":", "{\"type\":\"call\",\"id\":")
            .replace(",\"type\":\"call\"}", "}");
        assert_eq!(
            serde_json::from_str::<Value>(&reordered).unwrap(),
            serde_json::from_str::<Value>(literal).unwrap()
        );
        assert!(
            parse_output_frame(&reordered).is_err(),
            "raw key-order drift accepted"
        );
        assert!(
            parse_output_frame(&literal.replacen(':', ": ", 1)).is_err(),
            "raw whitespace drift accepted"
        );
    }
    let escaped =
        r#"{"id":1,"input":{"data_base64":"AP\u0038=","fd":13},"op":"fd_write","type":"call"}"#;
    assert!(
        parse_output_frame(escaped).is_err(),
        "alternate string escape accepted"
    );
    let duplicate =
        r#"{"id":1,"id":1,"input":{"data_base64":"","fd":13},"op":"fd_write","type":"call"}"#;
    assert!(
        parse_output_frame(duplicate).is_err(),
        "duplicate output key accepted"
    );
}

fn read_output_line(reader: &mut impl BufRead) -> Result<Option<String>, String> {
    let mut line = String::new();
    if reader
        .read_line(&mut line)
        .map_err(|error| error.to_string())?
        == 0
    {
        return Ok(None);
    }
    if !line.ends_with('\n') {
        return Err("helper frame is missing its literal LF delimiter".into());
    }
    line.pop();
    Ok(Some(line))
}

#[test]
fn captured_write_callback_requires_literal_lf_without_cr_normalization() {
    let literal =
        r#"{"id":1,"input":{"data_base64":"AP8=","fd":13},"op":"fd_write","type":"call"}"#;
    for suffix in ["\r\n", ""] {
        let mut reader = std::io::Cursor::new(format!("{literal}{suffix}"));
        let result = read_output_line(&mut reader)
            .and_then(|line| parse_output_frame(&line.ok_or("missing frame")?));
        assert!(
            result.is_err(),
            "capture normalized raw callback delimiter {suffix:?}"
        );
    }
    let mut reader = std::io::Cursor::new(format!("{literal}\n"));
    assert!(parse_output_frame(&read_output_line(&mut reader).unwrap().unwrap()).is_ok());
    assert!(read_output_line(&mut reader).unwrap().is_none());
}

#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
fn drive(case: &Value, guest: &(String, String)) -> Result<Run, String> {
    let empty = Vec::new();
    let replies = case["replies"].as_array().unwrap_or(&empty);
    let deny = match case["reply_policy"].as_str().unwrap_or("fail") {
        "fail" => false,
        "deny" => true,
        other => return Err(format!("unknown reply_policy {other}")),
    };
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|error| error.to_string())?;
    let (done, watchdog) = mpsc::channel::<()>();
    let pid = rustix::process::Pid::from_child(&child);
    let timer = std::thread::spawn(move || {
        if watchdog.recv_timeout(TIMEOUT).is_err() {
            let _ = rustix::process::kill_process(pid, rustix::process::Signal::KILL);
        }
    });
    let mut stderr_pipe = child.stderr.take().unwrap();
    let stderr_reader = std::thread::spawn(move || {
        let mut text = String::new();
        stderr_pipe.read_to_string(&mut text).map(|_| text)
    });
    let mut input = child.stdin.take();
    write_line(&mut input, &substitute(&case["request"], guest, None)?);
    for frame in case["inputs"].as_array().unwrap_or(&empty) {
        write_line(&mut input, &substitute(frame, guest, None)?);
    }
    // `true` closes input after the initial frames, `"after_replies"` once
    // the last scripted reply is sent; otherwise input stays open to the end.
    let close_after_replies = match &case["close_input"] {
        Value::Null | Value::Bool(false) => false,
        Value::Bool(true) => {
            input = None;
            false
        }
        Value::String(when) if when == "after_replies" => true,
        other => return Err(format!("invalid close_input {other}")),
    };
    if close_after_replies && replies.is_empty() {
        input = None;
    }
    let mut frames = Vec::new();
    let mut raw = Vec::new();
    let mut cursor = 0;
    let mut failure = None;
    let mut reader = BufReader::new(child.stdout.take().unwrap());
    while let Some(line) = read_output_line(&mut reader)? {
        let frame = parse_output_frame(&line)?;
        raw.push(line.clone());
        if failure.is_none() {
            if cursor < replies.len() && matches(&replies[cursor]["after"], &frame) {
                for send in replies[cursor]["send"].as_array().unwrap_or(&empty) {
                    write_line(&mut input, &substitute(send, guest, Some(&frame))?);
                }
                cursor += 1;
                if close_after_replies && cursor == replies.len() {
                    input = None;
                }
            } else if frame["type"] == "call" || frame["type"] == "sync" {
                if deny {
                    write_line(
                        &mut input,
                        &json!({"type":"reply","id":frame["id"],"errno":0,"output":{},"unsupported":true,"message":"not modeled"}),
                    );
                } else {
                    failure = Some(format!("unscripted helper request {frame}"));
                    input = None;
                }
            }
        }
        frames.push(frame);
    }
    drop(input);
    let status = child.wait().map_err(|error| error.to_string())?;
    let _ = done.send(());
    timer.join().unwrap();
    let stderr = stderr_reader
        .join()
        .unwrap()
        .map_err(|error| error.to_string())?;
    profile_stderr::validate(stderr.as_bytes())?;
    if let Some(failure) = failure {
        return Err(failure);
    }
    if cursor != replies.len() {
        return Err(format!(
            "scripted reply {cursor} never triggered: {}",
            replies[cursor]["after"]
        ));
    }
    if status.code().is_none() {
        return Err(format!(
            "helper did not exit normally ({status}); stderr: {stderr}"
        ));
    }
    Ok(Run {
        frames,
        raw,
        stderr,
        exit_code: status.code(),
    })
}

/// The comparable record of a run: projected frames, classified outcome,
/// per-stream evidence digests, helper stderr and process exit status.
fn expected_of(run: &Run) -> Result<Value, String> {
    let mut streams: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    for frame in &run.frames {
        if frame["type"] != "evidence" {
            continue;
        }
        let stream = frame["stream"].as_str().ok_or("evidence without stream")?;
        let name = match frame["instance"].as_str() {
            Some(instance) => format!("{instance}/{stream}"),
            None => stream.to_owned(),
        };
        let data = STANDARD
            .decode(
                frame["data_base64"]
                    .as_str()
                    .ok_or("evidence without data")?,
            )
            .map_err(|error| error.to_string())?;
        streams.entry(name).or_default().extend(data);
    }
    let evidence: Map<String, Value> = streams
        .into_iter()
        .map(|(name, bytes)| {
            let mut summary =
                json!({"bytes":bytes.len(),"sha256":format!("{:x}", Sha256::digest(&bytes))});
            // Text streams (the storage operations log, guest stdout/stderr)
            // are also kept readable so corpus diffs can be reviewed.
            let stream = name.rsplit('/').next().unwrap_or_default();
            if matches!(stream, "storage" | "stdout" | "stderr")
                && let Ok(text) = std::str::from_utf8(&bytes)
            {
                summary["text"] = json!(text);
            }
            (name, summary)
        })
        .collect();
    let (stderr, profile) = split_profile(&run.stderr)?;
    let mut expected = json!({
        "exit_code": run.exit_code,
        "termination": run.frames.last().map_or(Value::Null, |frame| frame["termination"].clone()),
        "frames": run.frames.iter().map(project).collect::<Vec<_>>(),
        "evidence": evidence,
        "stderr": stderr,
    });
    if !profile.is_null() {
        profile_invariants(&profile)?;
        expected["diagnostic_profile"] = profile;
    }
    Ok(expected)
}

/// fn-185: every import of every case is framed and attributed to an owner,
/// and the exact op table agrees with the v1 kind buckets it refines.
fn profile_invariants(profile: &Value) -> Result<(), String> {
    for counter in [
        "unattributed_imports",
        "unframed_imports",
        "span_errors",
        "abi_overlap_errors",
    ] {
        if profile[counter] != 0 {
            return Err(format!(
                "diagnostic profile {counter}: {}",
                profile[counter]
            ));
        }
    }
    let count = |value: &Value| value["count"].as_u64().unwrap_or(u64::MAX);
    let ops = profile["abi_ops"]
        .as_object()
        .ok_or("profile without abi_ops")?;
    let mut total = 0;
    for (name, op) in ops {
        let served: u64 = op["served_by"]
            .as_object()
            .ok_or("op without served_by")?
            .values()
            .map(|count| count.as_u64().unwrap_or(u64::MAX))
            .sum();
        if served != count(op) || count(op) == 0 {
            return Err(format!(
                "diagnostic op {name}: served_by does not cover its calls"
            ));
        }
        total += count(op);
    }
    let legacy: u64 = profile["abi"]
        .as_object()
        .ok_or("profile without abi")?
        .values()
        .map(count)
        .sum::<u64>()
        + count(&profile["abi_excluded"]);
    if total != legacy {
        return Err(format!(
            "diagnostic op calls {total} differ from abi buckets {legacy}"
        ));
    }
    Ok(())
}

/// Runs a case `runs` times and returns its projected record, refusing a
/// case whose projected output differs between any two runs.
fn stable_expected(
    case: &Value,
    projection: fn(&Run) -> Result<Value, String>,
    runs: usize,
) -> Result<Value, String> {
    let first = projection(&run_case(case)?)?;
    for _ in 1..runs {
        let next = projection(&run_case(case)?)?;
        if first != next {
            return Err(format!(
                "output varies across runs: {}",
                first_difference(&first, &next)
            ));
        }
    }
    Ok(first)
}

fn first_difference(left: &Value, right: &Value) -> String {
    fn walk(path: &str, left: &Value, right: &Value) -> Option<String> {
        match (left, right) {
            (Value::Object(a), Value::Object(b)) => {
                for key in a.keys().chain(b.keys()) {
                    let (x, y) = (
                        a.get(key).unwrap_or(&Value::Null),
                        b.get(key).unwrap_or(&Value::Null),
                    );
                    if x != y {
                        return walk(&format!("{path}.{key}"), x, y);
                    }
                }
                None
            }
            (Value::Array(a), Value::Array(b)) if a.len() == b.len() => a
                .iter()
                .zip(b)
                .enumerate()
                .find(|(_, (x, y))| x != y)
                .and_then(|(index, (x, y))| walk(&format!("{path}[{index}]"), x, y)),
            _ if left != right => {
                let clip = |value: &Value| value.to_string().chars().take(400).collect::<String>();
                Some(format!("{path}: {} != {}", clip(left), clip(right)))
            }
            _ => None,
        }
    }
    walk("", left, right).unwrap_or_default()
}

fn validate(case: &Value) -> Result<(), String> {
    fn raw_frames(value: &Value) -> Result<(), String> {
        match value {
            Value::Object(fields) if fields.contains_key("raw_json") => {
                let text = fields["raw_json"]
                    .as_str()
                    .ok_or("raw_json must be a string")?;
                if fields.len() != 1
                    || text.contains(['\n', '\r'])
                    || !serde_json::from_str::<Value>(text).is_ok_and(|value| value.is_object())
                {
                    return Err("raw_json must contain exactly one JSON object on one line".into());
                }
            }
            Value::Object(fields) => {
                for value in fields.values() {
                    raw_frames(value)?;
                }
            }
            Value::Array(items) => {
                for value in items {
                    raw_frames(value)?;
                }
            }
            _ => {}
        }
        Ok(())
    }
    let object = case.as_object().ok_or("case must be an object")?;
    if object["schema"] != SCHEMA {
        return Err(format!("schema must be {SCHEMA}"));
    }
    for key in object.keys() {
        if !CASE_KEYS.contains(&key.as_str()) {
            return Err(format!("unknown case field {key}"));
        }
    }
    if !object.get("intent").is_some_and(Value::is_string) {
        return Err("case needs an intent".into());
    }
    for reply in case["replies"].as_array().into_iter().flatten() {
        if !reply["after"].is_object() || !reply["send"].is_array() {
            return Err(format!("reply needs after and send: {reply}"));
        }
    }
    for field in ["request", "inputs", "replies"] {
        raw_frames(&case[field])?;
    }
    Ok(())
}

#[test]
fn raw_wire_frames_preserve_duplicate_keys_without_accepting_extra_frames() {
    let duplicate = r#"{"type":"namespace_inputs_end","type":"namespace_inputs_end"}"#;
    let case = json!({"schema":SCHEMA,"intent":"raw duplicate-key parser probes","inputs":[{"raw_json":duplicate}]});
    validate(&case).unwrap();
    assert_eq!(line(&case["inputs"][0]), duplicate);
    assert_eq!(
        line(&json!({"namespace":{"raw_json":"{\"files\":1,\"files\":2}"}})),
        "{\"namespace\":{\"files\":1,\"files\":2}}"
    );
    for raw in [
        json!(0),
        json!("[]"),
        json!("{}\n{}"),
        json!("{}\r"),
        json!("{"),
    ] {
        let mut bad = case.clone();
        bad["inputs"][0]["raw_json"] = raw;
        assert!(validate(&bad).is_err());
    }
    let mut bad = case;
    bad["inputs"][0]["extra"] = json!(true);
    assert!(validate(&bad).is_err());
}

fn case_files() -> Vec<PathBuf> {
    let mut files: Vec<_> = std::fs::read_dir(corpus_dir())
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| path.extension().is_some_and(|ext| ext == "json"))
        .collect();
    files.sort();
    helper_feature_corpus::append_cases(&mut files);
    files
}

/// Canonical file encoding: sorted keys, two levels of indentation, and every
/// deeper value (a frame, a reply pattern) on one line so diffs stay per frame.
fn encode(case: &Value) -> String {
    fn write(out: &mut String, value: &Value, depth: usize) {
        let pad = |out: &mut String, depth: usize| out.extend(std::iter::repeat_n("  ", depth));
        match value {
            Value::Object(fields) if depth < 3 && !fields.is_empty() => {
                out.push_str("{\n");
                let mut keys: Vec<_> = fields.keys().collect();
                keys.sort();
                for (index, key) in keys.iter().enumerate() {
                    pad(out, depth + 1);
                    out.push_str(&serde_json::to_string(key).unwrap());
                    out.push_str(": ");
                    write(out, &fields[*key], depth + 1);
                    out.push_str(if index + 1 < keys.len() { ",\n" } else { "\n" });
                }
                pad(out, depth);
                out.push('}');
            }
            Value::Array(items) if depth < 3 && !items.is_empty() => {
                out.push_str("[\n");
                for (index, item) in items.iter().enumerate() {
                    pad(out, depth + 1);
                    write(out, item, depth + 1);
                    out.push_str(if index + 1 < items.len() { ",\n" } else { "\n" });
                }
                pad(out, depth);
                out.push(']');
            }
            _ => out.push_str(&serde_json::to_string(value).unwrap()),
        }
    }
    let mut out = String::new();
    write(&mut out, case, 0);
    out.push('\n');
    out
}

fn read_case(file: &Path) -> Result<(String, Value), String> {
    let text = std::fs::read_to_string(file).map_err(|error| error.to_string())?;
    let case: Value = serde_json::from_str(&text).map_err(|error| error.to_string())?;
    validate(&case)?;
    Ok((text, case))
}

fn write_case(file: &Path, text: &str, case: &Value) -> Result<(), String> {
    let encoded = encode(case);
    if encoded == text {
        return Ok(());
    }
    let staged = file.with_extension("json.regenerating");
    std::fs::write(&staged, encoded)
        .and_then(|()| std::fs::rename(&staged, file))
        .map_err(|error| error.to_string())
}

fn runs_in_this_build(case: &Value) -> bool {
    BUILD == "default"
        || case["builds"]
            .as_array()
            .is_some_and(|builds| builds.iter().any(|build| build == BUILD))
}

/// Checks (or regenerates) one case against this build's expected block.
fn check_case(file: &Path, regenerate: bool) -> Result<(), String> {
    let (mut text, mut case) = read_case(file)?;
    if !runs_in_this_build(&case) {
        return Ok(());
    }
    if let Some(derived) = case.get("derived").cloned() {
        let rebuilt = helper_corpus_inputs::derive_inputs(&case, &derived)?;
        if rebuilt != case {
            if !regenerate || BUILD != "default" {
                return Err(format!(
                    "derived inputs are stale against {} (GOMAD_HELPER_CORPUS_REGENERATE=1)",
                    derived["from"]
                ));
            }
            case = rebuilt;
            write_case(file, &text, &case)?;
            text = encode(&case);
        }
    }
    admission_corpus_literals::check_storage_context_input(
        file.file_stem().unwrap().to_str().unwrap(),
        &case,
    )?;
    residual_helper_literals::input(file.file_stem().unwrap().to_str().unwrap(), &case)?;
    if file.file_stem().unwrap() == "admission_mode_one_inspection" {
        let (_, positive) = read_case(&corpus_dir().join("model_debug_pending_inspect.json"))?;
        let mut configuration = positive["request"]["local_model"].clone();
        configuration["mode"] = json!(1);
        configuration.as_object_mut().unwrap().remove("replay");
        if case["request"]["local_model"] != configuration {
            return Err(
                "mode-one inspection case has a confounding configuration difference".into(),
            );
        }
    }
    // Regeneration repeats each case ten times before it records anything.
    let actual = stable_expected(&case, expected_of, if regenerate { 10 } else { 2 })?;
    helper_feature_corpus::check_literals(file, &actual)?;
    model_corpus_literals::check_model_literals(
        file.file_stem().unwrap().to_str().unwrap(),
        &actual,
    )?;
    write_corpus_literals::check_write_literals(
        file.file_stem().unwrap().to_str().unwrap(),
        &actual,
    )?;
    admission_corpus_literals::check_admission_literals(
        file.file_stem().unwrap().to_str().unwrap(),
        &actual,
    )?;
    residual_helper_literals::actual(file.file_stem().unwrap().to_str().unwrap(), &actual)?;
    namespace_corpus_literals::check(file.file_stem().unwrap().to_str().unwrap(), &case, &actual)?;
    if BUILD != "default" {
        return helper_feature_corpus::check_expected(file, &text, &case, &actual, regenerate);
    }
    if regenerate {
        case["expected"] = actual;
        return write_case(file, &text, &case);
    }
    if case["expected"].is_null() {
        return Err("no expected block; regenerate and review".into());
    }
    let mut expected = case["expected"].clone();
    for (key, value) in case["expected_builds"][BUILD]
        .as_object()
        .into_iter()
        .flatten()
    {
        expected[key] = value.clone();
    }
    let expected = &expected;
    if *expected != actual {
        return Err(format!(
            "differs from the {BUILD} expected block (review, then GOMAD_HELPER_CORPUS_REGENERATE=1): {}",
            first_difference(expected, &actual)
        ));
    }
    if encode(&case) != text {
        return Err("not canonically encoded; regenerate".into());
    }
    Ok(())
}

#[test]
fn residual_regeneration_selection_is_exact_and_generation_only() {
    residual_helper_literals::residual_regeneration_selection_is_exact_and_generation_only();
}

#[cfg(feature = "diagnostic-profile")]
#[test]
fn runtime_choice_hash_is_attributed_to_transcript() {
    let (_, case) = read_case(&corpus_dir().join("model_runtime_record.json")).unwrap();
    let actual = stable_expected(&case, expected_of, 2).unwrap();
    let profile = &actual["diagnostic_profile"];
    assert_eq!(profile["owners"]["transcript"]["count"], 2);
    assert_eq!(profile["nested"]["alternative_hash"]["count"], 1);
    assert_eq!(profile["nested"]["transcript_hash"]["count"], 0);
    assert_eq!(profile["span_errors"], 0);
    assert_eq!(profile["abi_overlap_errors"], 0);
}

#[test]
fn helper_transcript_corpus_reproduces_expected_outcomes() {
    let regenerate = std::env::var("GOMAD_HELPER_CORPUS_REGENERATE").as_deref() == Ok("1");
    let selection = match std::env::var("GOMAD_HELPER_CORPUS_RESIDUAL_CASES") {
        Ok(value) => Some(value),
        Err(std::env::VarError::NotPresent) => None,
        Err(error) => panic!("invalid residual case selection: {error}"),
    };
    let files = residual_helper_literals::residual_regeneration_files(
        case_files(),
        regenerate,
        selection.as_deref(),
    )
    .unwrap();
    assert!(!files.is_empty(), "empty helper transcript corpus");
    let (derived, producers): (Vec<_>, Vec<_>) = files
        .iter()
        .partition(|file| read_case(file).is_ok_and(|(_, case)| case.get("derived").is_some()));
    let mut failures = Vec::new();
    for pass in [producers, derived] {
        if pass.is_empty() {
            continue;
        }
        for file in pass {
            if let Err(error) = check_case(file, regenerate) {
                failures.push(format!(
                    "{}: {error}",
                    file.file_name().unwrap().to_string_lossy()
                ));
            }
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}

#[test]
fn namespace_ownership_literal_rejects_wrong_admission_stage() {
    admission_corpus_literals::namespace_ownership_literal_rejects_wrong_admission_stage();
}

/// The determinism gate is real: without the metrics projection, host clock
/// readings make two runs of the same case differ and the loader refuses it.
#[test]
fn loader_refuses_output_that_varies_across_runs() {
    let file = corpus_dir().join("fresh_instance_execution.json");
    let case: Value = serde_json::from_str(&std::fs::read_to_string(file).unwrap()).unwrap();
    stable_expected(&case, expected_of, 2).unwrap();
    let raw = |run: &Run| Ok(json!(run.frames));
    let error = stable_expected(&case, raw, 2).unwrap_err();
    assert!(error.starts_with("output varies across runs"), "{error}");
    assert!(error.contains("metrics"), "{error}");
}

#[test]
fn loader_refuses_unscripted_requests_and_unused_replies() {
    let mut case = json!({"schema":SCHEMA,"intent":"self-test",
        "guest":{"wat":"(module (import \"wasi_snapshot_preview1\" \"sched_yield\" (func $y (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") call $y drop))"},
        "request":{"schema":"gomad.wasm-host/v1","type":"execute","module_path":"$guest_path","module_sha256":"$guest_sha256","memory_bytes":131_072,"fuel":1_000_000,"output_bytes":1024},
        "replies":[{"after":{"type":"started"},"send":[{"type":"authorize","module_sha256":"$trigger.module_sha256","configuration":"$trigger.engine.configuration"}]}]});
    validate(&case).unwrap();
    let error = run_case(&case).unwrap_err();
    assert!(error.starts_with("unscripted helper request"), "{error}");
    case["replies"].as_array_mut().unwrap().push(json!({"after":{"type":"call"},"send":[{"type":"reply","id":"$trigger.id","errno":0,"output":{}}]}));
    case["replies"]
        .as_array_mut()
        .unwrap()
        .push(json!({"after":{"type":"call"},"send":[]}));
    let error = run_case(&case).unwrap_err();
    assert!(
        error.starts_with("scripted reply 2 never triggered"),
        "{error}"
    );
}

#[test]
fn derived_inputs_follow_their_producer() {
    helper_corpus_inputs::derived_inputs_follow_their_producer();
}

#[test]
fn committed_cases_keep_their_cross_case_relations() {
    helper_corpus_inputs::committed_cases_keep_their_cross_case_relations();
}

#[test]
fn model_refusal_literals_reject_wrong_stage_messages() {
    model_corpus_literals::model_refusal_literals_reject_wrong_stage_messages();
}

#[test]
fn scatter_tuple_literals_reject_changed_payload_and_callback_ids() {
    write_corpus_literals::scatter_tuple_literals_reject_changed_payload_and_callback_ids();
}

#[test]
fn inspection_schema_literals_reject_wrong_schema_and_world_encoding() {
    model_corpus_literals::inspection_schema_literals_reject_wrong_schema_and_world_encoding();
}

#[test]
fn max_buffer_literals_reject_wrong_callback_identity_and_local_state() {
    write_corpus_literals::max_buffer_literals_reject_wrong_callback_identity_and_local_state();
}
