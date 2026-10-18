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
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::{BufRead, BufReader, Read, Write};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::mpsc;
use std::time::Duration;

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
const PROFILE_PREFIX: &str = "gomad-wasm-profile-v1 ";
const TIMEOUT: Duration = Duration::from_secs(120);

fn corpus_dir() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/helper_transcript")
}

/// Raw output of one helper run.
#[derive(Clone, PartialEq, Debug)]
struct Run {
    frames: Vec<Value>,
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
                text.push_str(&piece.repeat(count as usize));
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
    let Some(instance) = frame.get("instance") else {
        return frame.to_string();
    };
    let mut rest = frame.clone();
    rest.as_object_mut().unwrap().remove("instance");
    let rest = rest.to_string();
    if rest == "{}" {
        return format!("{{\"instance\":{instance}}}");
    }
    format!("{{\"instance\":{instance},{}", &rest[1..])
}

fn write_line(input: &mut Option<std::process::ChildStdin>, frame: &Value) {
    // A helper that already refused may have closed its input; the refusal
    // is in its output, so a broken pipe is not a corpus failure.
    if let Some(stdin) = input
        && writeln!(stdin, "{}", line(frame)).is_err()
    {
        *input = None;
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
    let result = drive(case, &guest);
    let _ = std::fs::remove_dir_all(path.parent().unwrap());
    result
}

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
    let mut cursor = 0;
    let mut failure = None;
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let line = line.map_err(|error| error.to_string())?;
        let frame: Value = serde_json::from_str(&line)
            .map_err(|error| format!("helper emitted non-JSON {line:?}: {error}"))?;
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
        stderr,
        exit_code: status.code(),
    })
}

/// Host clocks and resource counters (`metrics`, cluster `scheduler`) are
/// informational: durations keep only whether they are zero, process
/// counters only that they were measured.
fn project_metrics(metrics: &Value) -> Value {
    let Some(fields) = metrics.as_object() else {
        return metrics.clone();
    };
    let mut out = Map::new();
    for (key, value) in fields {
        let projected = match (key.as_str(), value.as_u64()) {
            ("process_user_nanos" | "process_system_nanos" | "peak_rss_bytes", Some(_)) => {
                json!("measured")
            }
            (name, Some(number)) if name.ends_with("_nanos") => {
                if number == 0 {
                    json!(0)
                } else {
                    json!("nonzero")
                }
            }
            _ => project(value),
        };
        out.insert(key.clone(), projected);
    }
    Value::Object(out)
}

fn project(value: &Value) -> Value {
    match value {
        Value::Object(fields) => {
            let mut out = Map::new();
            for (key, item) in fields {
                let projected = match key.as_str() {
                    // Host telemetry outside semantic evidence.
                    "metrics" | "scheduler" => project_metrics(item),
                    "engine" if item.is_object() => {
                        let mut engine = project(item);
                        for host in ["host_os", "host_arch"] {
                            if engine[host].is_string() {
                                engine[host] = json!(format!("${host}"));
                            }
                        }
                        engine
                    }
                    _ => project(item),
                };
                out.insert(key.clone(), projected);
            }
            Value::Object(out)
        }
        Value::Array(items) => Value::Array(items.iter().map(project).collect()),
        other => other.clone(),
    }
}

/// Diagnostic builds report one `gomad-wasm-profile-v1` stderr line of host
/// timing; it is kept as its projected document, the rest as text.
fn split_profile(stderr: &str) -> Result<(String, Value), String> {
    let mut rest = String::new();
    let mut profile = Value::Null;
    for line in stderr.split_inclusive('\n') {
        match line.strip_prefix(PROFILE_PREFIX) {
            Some(document) if profile.is_null() => {
                let document: Value = serde_json::from_str(document)
                    .map_err(|error| format!("diagnostic profile: {error}"))?;
                profile = project_profile(&document);
            }
            _ => rest.push_str(line),
        }
    }
    Ok((rest, profile))
}

/// `gomad.wasm-profile/v1`: event counts stay exact; every duration, CPU,
/// RSS and residual figure keeps only that it was measured (nested spans can
/// read zero nanoseconds, so not even zero/nonzero is stable).
fn project_profile(value: &Value) -> Value {
    match value {
        Value::Object(fields) => {
            let mut out = Map::new();
            for (key, item) in fields {
                let projected = match (key.as_str(), item.as_u64()) {
                    ("residual", _) => json!("measured"),
                    ("host_os" | "host_arch", _) => json!(format!("${key}")),
                    ("peak_rss_bytes_high_water", Some(_)) => json!("measured"),
                    (name, Some(_)) if name.ends_with("_nanos") => json!("measured"),
                    _ => project_profile(item),
                };
                out.insert(key.clone(), projected);
            }
            Value::Object(out)
        }
        other => other.clone(),
    }
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
        expected["diagnostic_profile"] = profile;
    }
    Ok(expected)
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
    fn walk(path: String, left: &Value, right: &Value) -> Option<String> {
        match (left, right) {
            (Value::Object(a), Value::Object(b)) => {
                for key in a.keys().chain(b.keys()) {
                    let (x, y) = (
                        a.get(key).unwrap_or(&Value::Null),
                        b.get(key).unwrap_or(&Value::Null),
                    );
                    if x != y {
                        return walk(format!("{path}.{key}"), x, y);
                    }
                }
                None
            }
            (Value::Array(a), Value::Array(b)) if a.len() == b.len() => a
                .iter()
                .zip(b)
                .enumerate()
                .find(|(_, (x, y))| x != y)
                .and_then(|(index, (x, y))| walk(format!("{path}[{index}]"), x, y)),
            _ if left != right => {
                let clip = |value: &Value| value.to_string().chars().take(400).collect::<String>();
                Some(format!("{path}: {} != {}", clip(left), clip(right)))
            }
            _ => None,
        }
    }
    walk(String::new(), left, right).unwrap_or_default()
}

fn validate(case: &Value) -> Result<(), String> {
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
    Ok(())
}

fn case_files() -> Vec<PathBuf> {
    let mut files: Vec<_> = std::fs::read_dir(corpus_dir())
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| path.extension().is_some_and(|ext| ext == "json"))
        .collect();
    files.sort();
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

/// A build's expected block is the default block with top-level fields
/// replaced, usually only `diagnostic_profile` (and `frames` under fuel-off).
fn build_overrides(default: &Value, actual: &Value) -> Result<Map<String, Value>, String> {
    let default = default
        .as_object()
        .ok_or("regenerate the default build first")?;
    let actual = actual.as_object().unwrap();
    if let Some(missing) = default.keys().find(|key| !actual.contains_key(*key)) {
        return Err(format!("{BUILD} output lacks default field {missing}"));
    }
    Ok(actual
        .iter()
        .filter(|(key, value)| default.get(*key) != Some(value))
        .map(|(key, value)| (key.clone(), value.clone()))
        .collect())
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
        let rebuilt = derive_inputs(&case, &derived)?;
        if rebuilt != case {
            if !regenerate {
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
    // Regeneration repeats each case ten times before it records anything.
    let actual = stable_expected(&case, expected_of, if regenerate { 10 } else { 2 })?;
    if regenerate {
        if BUILD == "default" {
            case["expected"] = actual;
        } else {
            // A build whose output equals the default block shares it.
            let builds = case.as_object_mut().unwrap();
            let mut separate = builds
                .remove("expected_builds")
                .and_then(|value| value.as_object().cloned())
                .unwrap_or_default();
            let overrides = build_overrides(&builds["expected"], &actual)?;
            if overrides.is_empty() {
                separate.remove(BUILD);
            } else {
                separate.insert(BUILD.into(), Value::Object(overrides));
            }
            if !separate.is_empty() {
                builds.insert("expected_builds".into(), Value::Object(separate));
            }
        }
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

/// Producers run before the cases whose inputs derive from their expected
/// blocks, so one regeneration pass refreshes both.
#[test]
fn helper_transcript_corpus_reproduces_expected_outcomes() {
    let regenerate = std::env::var("GOMAD_HELPER_CORPUS_REGENERATE").as_deref() == Ok("1");
    let files = case_files();
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

/// Concatenated evidence stream of a producer's default expected block.
fn producer_stream(name: &str, stream: &str) -> Result<Vec<u8>, String> {
    let (_, producer) = read_case(&corpus_dir().join(format!("{name}.json")))?;
    let mut bytes = Vec::new();
    for frame in producer["expected"]["frames"]
        .as_array()
        .ok_or("producer has no expected frames")?
    {
        if frame["type"] == "evidence" && frame["stream"] == stream {
            bytes.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap_or_default())
                    .map_err(|error| error.to_string())?,
            );
        }
    }
    Ok(bytes)
}

/// The JSON crash context section (tag 6) of a `GMSTORE` storage state.
fn crash_context(state: &[u8]) -> Result<Value, String> {
    let word = |at: usize| {
        state
            .get(at..at + 4)
            .map(|b| u32::from_le_bytes(b.try_into().unwrap()))
    };
    let long = |at: usize| {
        state
            .get(at..at + 8)
            .map(|b| u64::from_le_bytes(b.try_into().unwrap()))
    };
    let mut offset = 16;
    for tag in 1..=6u32 {
        if word(offset) != Some(tag) {
            return Err(format!("storage state section {tag} missing"));
        }
        let length = long(offset + 4).ok_or("storage state truncated")? as usize;
        offset += 12;
        let body = state
            .get(offset..offset + length)
            .ok_or("storage state truncated")?;
        if tag == 6 {
            return serde_json::from_slice(body).map_err(|error| error.to_string());
        }
        offset += length;
    }
    unreachable!()
}

fn chunk_frames(kind: &str, bytes: &[u8]) -> Vec<Value> {
    let mut frames: Vec<Value> = bytes
        .chunks(65536)
        .enumerate()
        .map(|(sequence, chunk)| json!({"type":format!("{kind}_input"),"sequence":sequence,"data_base64":STANDARD.encode(chunk)}))
        .collect();
    frames.push(json!({"type":format!("{kind}_inputs_end")}));
    frames
}

fn declaration(bytes: &[u8]) -> Value {
    json!({"chunks":bytes.len().div_ceil(65536),"bytes":bytes.len(),"sha256":format!("{:x}", Sha256::digest(bytes))})
}

/// Rebuilds a derived case's inputs from its producer: `storage_image`
/// embeds the producer's returned recovery image; `crash_context` embeds the
/// crash context recorded in its storage state, with JSON-pointer `set`
/// edits and, under `select`, a crash selection of every recorded operation.
/// The inputs follow the authorize frame and namespace inputs, as Go sends
/// them.
fn derive_inputs(case: &Value, derived: &Value) -> Result<Value, String> {
    let from = derived["from"]
        .as_str()
        .ok_or("derived.from must name a case")?;
    let mut case = case.clone();
    let mut inputs = Vec::new();
    if derived["storage_image"] == true {
        let image = producer_stream(from, "storage_image")?;
        if image.is_empty() {
            return Err(format!("{from} returned no storage image"));
        }
        inputs.extend(chunk_frames("storage_image", &image));
        case["request"]["storage"]["image"] = declaration(&image);
    }
    if let Some(edit) = derived.get("crash_context") {
        let mut context = crash_context(&producer_stream(from, "storage_state")?)?;
        for (pointer, value) in edit["set"].as_object().into_iter().flatten() {
            *context
                .pointer_mut(pointer)
                .ok_or_else(|| format!("crash context has no {pointer}"))? = value.clone();
        }
        let bytes = serde_json::to_vec(&context).unwrap();
        inputs.extend(chunk_frames("storage_context", &bytes));
        case["request"]["storage"]["crash_context"] = declaration(&bytes);
        if edit["select"] == true {
            let selection: Map<String, Value> = context
                .as_object()
                .ok_or("crash context must be an object")?
                .iter()
                .map(|(volume, state)| {
                    let ids = state["Operations"]
                        .as_array()
                        .into_iter()
                        .flatten()
                        .map(|operation| operation["ID"].clone());
                    (volume.clone(), Value::Array(ids.collect()))
                })
                .collect();
            case["request"]["storage"]["crash_selection"] = Value::Object(selection);
        }
    }
    let send = case["replies"][0]["send"]
        .as_array_mut()
        .ok_or("derived case needs an authorize reply")?;
    send.retain(|frame| {
        !frame["type"].as_str().is_some_and(|kind| {
            kind.starts_with("storage_image_input") || kind.starts_with("storage_context_input")
        })
    });
    send.extend(inputs);
    Ok(case)
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
        "request":{"schema":"gomad.wasm-host/v1","type":"execute","module_path":"$guest_path","module_sha256":"$guest_sha256","memory_bytes":131072,"fuel":1000000,"output_bytes":1024},
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

/// Derived inputs are a function of their producer: a corrupted copy is
/// detected and rebuilt to the committed bytes.
#[test]
fn derived_inputs_follow_their_producer() {
    let mut derived = 0;
    for file in case_files() {
        let (_, case) = read_case(&file).unwrap();
        let Some(spec) = case.get("derived") else {
            continue;
        };
        derived += 1;
        let mut corrupted = case.clone();
        corrupted["replies"][0]["send"]
            .as_array_mut()
            .unwrap()
            .push(json!({"type":"storage_context_input","sequence":9,"data_base64":""}));
        corrupted["request"]["storage"]["image"] = json!(null);
        corrupted["request"]["storage"]["crash_context"] = json!(null);
        assert_ne!(corrupted, case);
        let rebuilt = derive_inputs(&corrupted, spec).unwrap();
        let mut want = case.clone();
        for key in ["image", "crash_context"] {
            if want["request"]["storage"].get(key).is_none() {
                want["request"]["storage"][key] = json!(null);
            }
        }
        assert_eq!(rebuilt, want, "{}", file.display());
    }
    assert!(derived >= 5, "{derived} derived cases");
}

/// Relations between committed cases that the deleted engine.rs scenarios
/// asserted, checked over the expected blocks without running the helper.
#[test]
fn committed_cases_keep_their_cross_case_relations() {
    let stream = |case: &str, name: &str| producer_stream(case, name).unwrap();
    // Split revocation preserves the same storage whether the revoked call
    // was replied to or completed by the interrupt (split_revocation_*).
    for kind in ["crash", "stop"] {
        for effect in ["remote_only", "local_write"] {
            for name in ["storage", "storage_state", "storage_image"] {
                let replied = stream(
                    &format!("split_revocation_{kind}_after_reply_{effect}"),
                    name,
                );
                let completed = stream(
                    &format!("split_revocation_{kind}_completed_interrupt_{effect}"),
                    name,
                );
                assert!(!replied.is_empty(), "{kind}/{effect}/{name} is empty");
                assert_eq!(replied, completed, "{kind}/{effect}/{name}");
            }
        }
    }
    // The sealed storage state binds the identity of the returned image.
    let state = stream("storage_volume_operations_local", "storage_state");
    let image = stream("storage_volume_operations_local", "storage_image");
    assert!(image.starts_with(b"GMVOLIMG"));
    let identity = format!("{:x}", Sha256::digest(&image));
    assert!(
        state
            .windows(identity.len())
            .any(|window| window == identity.as_bytes()),
        "storage state does not bind the image identity"
    );
    // Recovery from that image starts with the durable file.
    const DURABLE: &[u8] = br#""Path":"/db""#;
    let recovered = stream("storage_recovery_from_returned_image", "storage_state");
    assert!(
        recovered
            .windows(DURABLE.len())
            .any(|window| window == DURABLE),
        "recovered state lacks /db"
    );
    // The read-only namespace run records path_open, poll_oneoff and fd_write
    // as calls 1, 2 and 3 in one transcript.
    let transcript = stream("namespace_rofs_and_mixed_poll_stay_local", "transcript");
    assert_eq!(&transcript[..8], b"GMWT\x01\0\0\0");
    let mut at = 8;
    for (id, tag) in [(1u64, 26u8), (2, 32), (3, 8)] {
        assert_eq!(transcript[at], tag, "record {id} tag");
        assert_eq!(
            &transcript[at + 4..at + 12],
            &id.to_le_bytes(),
            "record {id} id"
        );
        at += 16 + u32::from_le_bytes(transcript[at + 12..at + 16].try_into().unwrap()) as usize;
    }
    assert_eq!(at, transcript.len(), "trailing transcript records");
}
