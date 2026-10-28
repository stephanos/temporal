//! Counts observed by native runners, without expected-count substitution.
#[cfg(test)]
mod go_tests;
mod tlc_observations;

use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet, VecDeque},
    fs,
    io::{self, Write},
    os::unix::ffi::{OsStrExt, OsStringExt},
    path::Path,
    process::Command,
    sync::OnceLock,
};

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
/// Actual completed producer counts; absent observations remain unknown.
pub struct WorkCounts {
    /// Tests discovered, including filtered inventory.
    pub discovered: Option<u64>,
    /// Tests completed with passed or failed status.
    pub executed: Option<u64>,
    /// Discovered tests reported ignored by the runner.
    pub ignored: Option<u64>,
    /// Discovered tests filtered from execution.
    pub skipped: Option<u64>,
    /// TLC starts confirmed by checker startup output.
    pub tlc_launches: Option<u64>,
    /// Distinct confirmed source/configuration identities.
    pub tlc_configurations: Option<u64>,
    /// Linked child authorities, excluded from this stage's own counts.
    #[serde(skip)]
    pub child_receipts: Vec<serde_json::Value>,
}

fn add(target: &mut u64, value: u64) -> io::Result<()> {
    *target = target.checked_add(value).ok_or_else(|| io::Error::other("work count overflow"))?;
    Ok(())
}

fn event_paths(directory: &Path) -> io::Result<Vec<std::path::PathBuf>> {
    let mut paths = Vec::new();
    let mut entries = 0;
    for entry in fs::read_dir(directory)? {
        entries += 1;
        if entries > umpire_backend_process::work::MAX_EVENTS * 2 {
            return Err(io::Error::other("work observation entry limit exceeded"));
        }
        let entry = entry?;
        let path = entry.path();
        let name = path
            .file_name()
            .and_then(|name| name.to_str())
            .ok_or_else(|| io::Error::other("invalid work observation filename"))?;
        let (slot, kind) = name
            .split_once('.')
            .ok_or_else(|| io::Error::other("invalid work observation filename"))?;
        if slot
            .parse::<u64>()
            .ok()
            .is_none_or(|slot| slot >= umpire_backend_process::work::MAX_EVENTS)
            || !["claim", "event"].contains(&kind)
        {
            return Err(io::Error::other("partial or unexpected work observation"));
        }
        let metadata = fs::symlink_metadata(&path)?;
        if !metadata.is_file()
            || metadata.is_symlink()
            || metadata.len() > umpire_backend_process::work::MAX_EVENT_BYTES as u64
            || !directory.join(format!("{slot}.claim")).is_file()
            || !directory.join(format!("{slot}.event")).is_file()
        {
            return Err(io::Error::other("incomplete work observation"));
        }
        if kind == "event" {
            paths.push(path);
        }
    }
    Ok(paths)
}

/// Encode an OS path without record separators or Unicode conversion.
pub(super) fn encode_path(path: &Path) -> String {
    let bytes = path.as_os_str().as_bytes();
    let mut encoded = String::with_capacity(bytes.len() * 2);
    for &byte in bytes {
        encoded.push(char::from(b"0123456789abcdef"[usize::from(byte >> 4)]));
        encoded.push(char::from(b"0123456789abcdef"[usize::from(byte & 15)]));
    }
    encoded
}

fn decode_hex(encoded: &str) -> io::Result<Vec<u8>> {
    if encoded.is_empty() || !encoded.len().is_multiple_of(2) {
        return Err(io::Error::other("invalid encoded child path"));
    }
    let bytes = encoded
        .as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            let text = std::str::from_utf8(pair).map_err(io::Error::other)?;
            u8::from_str_radix(text, 16).map_err(io::Error::other)
        })
        .collect::<io::Result<Vec<_>>>()?;
    Ok(bytes)
}

fn decode_path(encoded: &str) -> io::Result<std::path::PathBuf> {
    raw_path(decode_hex(encoded)?)
}

fn raw_path(bytes: Vec<u8>) -> io::Result<std::path::PathBuf> {
    if bytes.contains(&0) {
        return Err(io::Error::other("NUL in child path"));
    }
    let path = std::path::PathBuf::from(std::ffi::OsString::from_vec(bytes));
    if !path.is_absolute() {
        return Err(io::Error::other("relative child path"));
    }
    Ok(path)
}

fn child_receipt(path: &Path, digest: &str) -> io::Result<serde_json::Value> {
    let records = fs::read_to_string(path)?;
    if umpire_backend_process::bytes_sha256(records.as_bytes()) != digest {
        return Err(io::Error::other("child count authority digest mismatch"));
    }
    let stages = records
        .lines()
        .map(serde_json::from_str::<serde_json::Value>)
        .collect::<Result<Vec<_>, _>>()?;
    if stages.is_empty() {
        return Err(io::Error::other("empty child count authority"));
    }
    let value = serde_json::json!({"path": path.to_str(), "path_bytes": path.as_os_str().as_bytes(), "sha256": digest, "records": records, "stages": stages});
    validate_child(&value)?;
    Ok(value)
}

/// Validate the structured count boundary shared by publishers, collectors and child authorities.
pub fn validate_receipt(receipt: &serde_json::Value) -> io::Result<()> {
    if receipt["count_authority"] != "exclusive-stage" {
        return Err(io::Error::other("missing exclusive count authority"));
    }
    let successful = receipt["state"] == "success";
    let complete = successful || receipt["state"] == "skipped";
    let error = receipt
        .get("count_error")
        .ok_or_else(|| io::Error::other("missing count observation status"))?;
    if !error.is_null() && !error.is_string() {
        return Err(io::Error::other("invalid count observation status"));
    }
    let value = receipt.get("counts").ok_or_else(|| io::Error::other("missing count object"))?;
    if value.is_null() {
        if complete || error.as_str().is_none_or(str::is_empty) {
            return Err(io::Error::other("missing completed count object"));
        }
        return Ok(());
    }
    let fields = value.as_object().ok_or_else(|| io::Error::other("invalid count object"))?;
    if fields.len() != 6 {
        return Err(io::Error::other("incomplete count object"));
    }
    let counts: WorkCounts = serde_json::from_value(value.clone())?;
    validate_totals(&counts, successful && receipt["stage"] == "tests")?;
    if complete && !error.is_null() {
        return Err(io::Error::other("successful receipt has incomplete observations"));
    }
    let children = receipt.get("child_count_receipts").and_then(serde_json::Value::as_array);
    if successful && receipt["stage"] == "standalone" && children.is_none_or(Vec::is_empty) {
        return Err(io::Error::other("missing standalone child count authority"));
    }
    if let Some(children) = children {
        let mut paths = BTreeSet::new();
        for child in children {
            if !paths.insert(serde_json::from_value::<Vec<u8>>(child["path_bytes"].clone())?) {
                return Err(io::Error::other("duplicate child count authority"));
            }
            validate_child(child)?;
        }
    }
    Ok(())
}

fn validate_totals(counts: &WorkCounts, required: bool) -> io::Result<()> {
    match (counts.discovered, counts.executed, counts.ignored, counts.skipped) {
        (None, None, None, None) if !required => (),
        (Some(discovered), Some(executed), Some(ignored), Some(skipped))
            if executed.checked_add(ignored).and_then(|count| count.checked_add(skipped))
                == Some(discovered)
                && (!required || executed > 0) => {}
        _ => return Err(io::Error::other("incomplete or inconsistent test count totals")),
    }
    if counts.tlc_configurations.is_some_and(|configurations| {
        counts.tlc_launches.is_none_or(|launches| configurations > launches)
    }) {
        return Err(io::Error::other("inconsistent TLC count totals"));
    }
    Ok(())
}

fn validate_child(child: &serde_json::Value) -> io::Result<()> {
    let records =
        child["records"].as_str().ok_or_else(|| io::Error::other("missing child count records"))?;
    let path = raw_path(serde_json::from_value::<Vec<u8>>(child["path_bytes"].clone())?)?;
    if child["path"] != serde_json::json!(path.to_str())
        || child["sha256"] != umpire_backend_process::bytes_sha256(records.as_bytes())
    {
        return Err(io::Error::other("invalid child count provenance"));
    }
    let stages = records
        .lines()
        .map(serde_json::from_str::<serde_json::Value>)
        .collect::<Result<Vec<_>, _>>()?;
    if stages.is_empty() || child["stages"] != serde_json::json!(stages) {
        return Err(io::Error::other("incomplete child count records"));
    }
    let mut identities = BTreeSet::new();
    for stage in stages {
        let name = stage["stage"]
            .as_str()
            .filter(|name| {
                !name.is_empty()
                    && name.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
            })
            .ok_or_else(|| io::Error::other("missing child stage identity"))?;
        let scope = stage["scope"]
            .as_str()
            .filter(|scope| scope.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-'))
            .ok_or_else(|| io::Error::other("missing child scope identity"))?;
        if !identities.insert((scope.to_owned(), name.to_owned()))
            || !["success", "skipped"].iter().any(|state| stage["state"] == *state)
            || stage["exit"] != 0
            || !stage["seconds"]
                .as_f64()
                .is_some_and(|seconds| seconds.is_finite() && seconds >= 0.0)
            || (stage["state"] == "skipped" && stage["reason"].as_str().is_none_or(str::is_empty))
        {
            return Err(io::Error::other("incomplete or duplicate child stage authority"));
        }
        validate_receipt(&stage)?;
    }
    Ok(())
}

/// Refuse incomplete, duplicated or malformed producer records.
pub fn collect(directory: &Path) -> io::Result<WorkCounts> {
    let mut counts = WorkCounts::default();
    let mut tests = [0; 4];
    let mut observed_tests = false;
    let mut test_started = BTreeSet::new();
    let mut test_finished = BTreeSet::new();
    let mut tlc = tlc_observations::Collector::default();
    let mut passed = BTreeMap::new();
    for path in event_paths(directory)? {
        let record = fs::read_to_string(path)?;
        let fields = record
            .strip_suffix('\n')
            .ok_or_else(|| io::Error::other("partial work observation"))?
            .split('\t')
            .collect::<Vec<_>>();
        match fields.as_slice() {
            ["1", "tests-begin", id] if valid_id(id) => {
                if !test_started.insert(id.to_string()) {
                    return Err(io::Error::other("duplicate test producer"));
                }
            }
            ["1", "tests-finish", id, discovered, executed, ignored, skipped] if valid_id(id) => {
                if !test_finished.insert(id.to_string()) {
                    return Err(io::Error::other("duplicate test completion"));
                }
                for (total, value) in tests.iter_mut().zip([discovered, executed, ignored, skipped])
                {
                    add(total, value.parse().map_err(|_| io::Error::other("invalid test count"))?)?;
                }
                let discovered = discovered.parse::<u64>().map_err(io::Error::other)?;
                let executed = executed.parse::<u64>().map_err(io::Error::other)?;
                let ignored = ignored.parse::<u64>().map_err(io::Error::other)?;
                let skipped = skipped.parse::<u64>().map_err(io::Error::other)?;
                if executed.checked_add(ignored).and_then(|count| count.checked_add(skipped))
                    != Some(discovered)
                {
                    return Err(io::Error::other("inconsistent observed test counts"));
                }
                observed_tests = true;
            }
            ["1", "tests-pass", id, name, digest] if valid_id(id) => {
                let name = String::from_utf8(decode_hex(name)?).map_err(io::Error::other)?;
                if passed.insert((id.to_string(), name), digest.to_string()).is_some() {
                    return Err(io::Error::other("duplicate successful test identity observation"));
                }
            }
            ["1", "child-receipts", path, digest] => {
                let path = decode_path(path)?;
                if counts.child_receipts.iter().any(|value| {
                    value["path_bytes"] == serde_json::json!(path.as_os_str().as_bytes())
                }) {
                    return Err(io::Error::other("invalid or duplicate child count authority"));
                }
                counts.child_receipts.push(child_receipt(&path, digest)?);
            }
            _ => {
                if !tlc.ingest(&fields)? {
                    return Err(io::Error::other("unknown work observation"));
                }
            }
        }
    }
    if test_started != test_finished {
        return Err(io::Error::other("missing test producer or completion"));
    }
    if observed_tests {
        counts.discovered = Some(tests[0]);
        counts.executed = Some(tests[1]);
        counts.ignored = Some(tests[2]);
        counts.skipped = Some(tests[3]);
    }
    for ((id, name), digest) in passed {
        if !test_started.contains(&id) {
            return Err(io::Error::other("successful test identity has no completed producer"));
        }
        tlc.record_pass(name, digest)?;
    }
    (counts.tlc_launches, counts.tlc_configurations) = tlc.finish(directory)?;
    Ok(counts)
}

#[derive(Default)]
struct TestOutput {
    discovered: u64,
    inventory: bool,
    expected: BTreeSet<String>,
    completed: BTreeSet<String>,
    binary_paths: BTreeMap<String, std::path::PathBuf>,
    binary_hashes: BTreeMap<std::path::PathBuf, String>,
    passed: BTreeMap<String, String>,
    executed: u64,
    ignored: u64,
    skipped: u64,
    started: u64,
    finished: u64,
    doctest_started: VecDeque<u64>,
}

impl TestOutput {
    fn inventory(bytes: &[u8]) -> io::Result<Self> {
        let value: serde_json::Value = serde_json::from_slice(bytes)?;
        let suites = value["rust-suites"]
            .as_object()
            .ok_or_else(|| io::Error::other("missing nextest inventory"))?;
        let mut counts = Self { inventory: true, ..Self::default() };
        for suite in suites.values() {
            if suite["status"] == "skipped" {
                continue;
            }
            if suite["status"] != "listed" {
                return Err(io::Error::other("unknown nextest suite inventory status"));
            }
            let package = suite["package-name"]
                .as_str()
                .ok_or_else(|| io::Error::other("missing nextest package"))?;
            let binary = suite["binary-name"]
                .as_str()
                .ok_or_else(|| io::Error::other("missing nextest binary"))?;
            let tests = suite["testcases"]
                .as_object()
                .ok_or_else(|| io::Error::other("missing nextest tests"))?;
            for (name, test) in tests {
                if !test["ignored"].is_boolean() {
                    return Err(io::Error::other("missing nextest ignored policy"));
                }
                add(&mut counts.discovered, 1)?;
                match test["filter-match"]["status"].as_str() {
                    Some("matches") => {
                        let identity = format!("{package}::{binary}${name}");
                        if !counts.expected.insert(identity.clone()) {
                            return Err(io::Error::other("duplicate nextest inventory identity"));
                        }
                        if package == "umpire-tlc"
                            && binary == "umpire_tlc"
                            && let Some(path) = suite["binary-path"].as_str()
                        {
                            counts.binary_paths.insert(identity, path.into());
                        }
                    }
                    Some("mismatch") if test["filter-match"]["reason"] == "ignored" => {
                        add(&mut counts.ignored, 1)?;
                    }
                    Some("mismatch") => add(&mut counts.skipped, 1)?,
                    _ => return Err(io::Error::other("unknown nextest test inventory status")),
                }
            }
        }
        if value["test-count"].as_u64() != Some(counts.discovered) {
            return Err(io::Error::other("incomplete nextest inventory"));
        }
        Ok(counts)
    }

    fn nextest(&mut self, line: &[u8]) -> io::Result<()> {
        let value: serde_json::Value = serde_json::from_slice(line)?;
        if value["type"] != "test" || !["ok", "failed"].iter().any(|event| value["event"] == *event)
        {
            return Ok(());
        }
        let name = value["name"]
            .as_str()
            .ok_or_else(|| io::Error::other("missing nextest completion identity"))?;
        if !self.inventory
            || !self.expected.contains(name)
            || !self.completed.insert(name.to_owned())
        {
            return Err(io::Error::other("unexpected or duplicate nextest completion"));
        }
        add(&mut self.executed, 1)?;
        if value["event"] == "ok"
            && let Some(path) = self.binary_paths.get(name)
        {
            let digest = if let Some(digest) = self.binary_hashes.get(path) {
                digest.clone()
            } else {
                let digest = umpire_backend_process::bytes_sha256(&fs::read(path)?);
                self.binary_hashes.insert(path.clone(), digest.clone());
                digest
            };
            self.passed.insert(name.to_owned(), digest);
        }
        Ok(())
    }

    fn doctest(&mut self, line: &[u8]) -> io::Result<()> {
        let Ok(line) = std::str::from_utf8(line) else { return Ok(()) };
        let line = unstyled(line);
        if let Some(count) = line.trim().strip_prefix("running ") {
            let count = count
                .strip_suffix(" tests")
                .or_else(|| count.strip_suffix(" test"))
                .ok_or_else(|| io::Error::other("malformed doctest suite start"))?
                .parse()
                .map_err(io::Error::other)?;
            self.doctest_started.push_back(count);
            add(&mut self.started, 1)?;
            return Ok(());
        }
        let Some(summary) = line.trim().strip_prefix("test result: ") else { return Ok(()) };
        let Some((status, summary)) = summary.split_once(". ") else {
            return Err(io::Error::other("malformed doctest summary"));
        };
        if !["ok", "FAILED"].contains(&status) {
            return Err(io::Error::other("unknown doctest summary status"));
        }
        let fields = summary.split(';').take(5).map(str::trim).collect::<Vec<_>>();
        let mut totals = [0; 5];
        for ((field, label), total) in fields
            .iter()
            .zip(["passed", "failed", "ignored", "measured", "filtered out"])
            .zip(totals.iter_mut())
        {
            let value = field
                .strip_suffix(label)
                .ok_or_else(|| io::Error::other("malformed doctest count"))?;
            *total = value.trim().parse().map_err(|_| io::Error::other("invalid doctest count"))?;
        }
        if fields.len() != 5 || totals[3] != 0 {
            return Err(io::Error::other("incomplete doctest count summary"));
        }
        let selected = totals[..4]
            .iter()
            .try_fold(0_u64, |sum, value| sum.checked_add(*value))
            .ok_or_else(|| io::Error::other("doctest selected count overflow"))?;
        if self.doctest_started.pop_front() != Some(selected) {
            return Err(io::Error::other("missing or inconsistent doctest suite start"));
        }
        add(&mut self.finished, 1)?;
        for value in totals {
            add(&mut self.discovered, value)?;
        }
        add(&mut self.executed, totals[0])?;
        add(&mut self.executed, totals[1])?;
        add(&mut self.ignored, totals[2])?;
        add(&mut self.skipped, totals[4])?;
        Ok(())
    }

    fn publish(&self) -> io::Result<()> {
        if if self.inventory {
            self.expected.is_empty() || self.expected != self.completed
        } else {
            self.started == 0 || self.started != self.finished
        } {
            return Err(io::Error::other("missing or incomplete test execution observations"));
        }
        Ok(())
    }
}

pub(crate) fn is_test(command: &Command) -> bool {
    let arguments = command.get_args().collect::<Vec<_>>();
    super::go_command::is_go_test(command)
        || arguments.windows(2).any(|pair| pair == ["nextest", "run"])
        || (arguments.iter().any(|argument| *argument == "test")
            && arguments.iter().any(|argument| *argument == "--doc"))
}

fn unstyled(line: &str) -> String {
    let mut output = String::with_capacity(line.len());
    let mut characters = line.chars().peekable();
    while let Some(character) = characters.next() {
        if character == '\u{1b}' && characters.peek() == Some(&'[') {
            characters.next();
            for character in characters.by_ref() {
                if ('@'..='~').contains(&character) {
                    break;
                }
            }
        } else {
            output.push(character);
        }
    }
    output
}

fn valid_id(id: &str) -> bool {
    !id.is_empty() && id.bytes().all(|byte| byte.is_ascii_digit() || byte == b'-')
}

fn copy_command(command: &Command) -> Command {
    let mut copy = Command::new(command.get_program());
    if let Some(directory) = command.get_current_dir() {
        copy.current_dir(directory);
    }
    for (key, value) in command.get_envs() {
        if let Some(value) = value {
            copy.env(key, value);
        } else {
            copy.env_remove(key);
        }
    }
    copy
}

fn inventory_command(command: &Command) -> io::Result<Command> {
    let mut inventory = copy_command(command);
    let mut arguments = command.get_args();
    let mut found = false;
    while let Some(argument) = arguments.next() {
        if !found && argument == "nextest" {
            inventory.args(["nextest", "list"]);
            if arguments.next().is_none_or(|argument| argument != "run") {
                return Err(io::Error::other("invalid nextest runner arguments"));
            }
            found = true;
        } else if found && argument == "--fail-fast" {
        } else if found
            && [
                "--status-level",
                "--final-status-level",
                "--success-output",
                "--failure-output",
                "--no-tests",
            ]
            .iter()
            .any(|flag| argument == *flag)
        {
            if arguments.next().is_none() {
                return Err(io::Error::other("missing nextest runner option value"));
            }
        } else {
            inventory.arg(argument);
        }
    }
    inventory.args(["--message-format", "json"]);
    Ok(inventory)
}

static CONTROLLER: OnceLock<bool> = OnceLock::new();

/// Only the configured native gate executable owns test-count interception.
/// Freeze verification before Cargo can replace this process's running executable.
/// Test binaries retain their rejected role even if a configured path changes later.
pub fn owns_controller() -> io::Result<bool> {
    if let Some(admitted) = CONTROLLER.get() {
        return Ok(*admitted);
    }
    let configured =
        ["UMPIRE_GATE_EXECUTABLE", "UMPIRE_GATE_CONTROLLER_EXECUTABLE"].map(std::env::var_os);
    let admitted = if configured.iter().all(Option::is_none) {
        false
    } else {
        let executable = std::env::current_exe()?;
        configured
            .into_iter()
            .flatten()
            .any(|path| fs::canonicalize(path).is_ok_and(|path| path == executable))
    };
    Ok(*CONTROLLER.get_or_init(|| admitted))
}

/// Capture actual completed test suites through the existing owned process controller.
/// Missing stage scope leaves ordinary callers unchanged.
pub fn observe(command: &Command) -> io::Result<Option<std::process::Output>> {
    observe_with_inventory(command, None)
}

/// Reuse the just-admitted scoped inventory for this exact test dispatch.
pub fn observe_with_inventory(
    command: &Command,
    admitted: Option<&[u8]>,
) -> io::Result<Option<std::process::Output>> {
    observe_streamed(command, admitted, false, false)
}

/// Preserve human streaming while privately retaining the runner's count stream.
pub fn observe_streamed(
    command: &Command,
    admitted: Option<&[u8]>,
    stream_stdout: bool,
    stream_stderr: bool,
) -> io::Result<Option<std::process::Output>> {
    if !is_test(command) || umpire_backend_process::work::directory()?.is_none() {
        return Ok(None);
    }
    if super::go_command::is_go_test(command) {
        return observe_go(command, stream_stdout, stream_stderr).map(Some);
    }
    let mut observed = copy_command(command);
    observed.args(command.get_args());
    let nextest =
        command.get_args().collect::<Vec<_>>().windows(2).any(|pair| pair == ["nextest", "run"]);
    if nextest {
        observed
            .args(["--message-format", "libtest-json", "--message-format-version", "0.1"])
            .env("NEXTEST_EXPERIMENTAL_LIBTEST_JSON", "1");
    }
    let id = umpire_backend_process::work::begin_tests()?;
    let mut inventory_errors = Vec::new();
    let inventory = if nextest && admitted.is_some() {
        Some(TestOutput::inventory(admitted.unwrap_or_default())?)
    } else if nextest {
        let output = super::scheduler::execute_inner(
            &inventory_command(command)?,
            false,
            stream_stderr,
            None,
        )?;
        if !output.status.success() {
            return Ok(Some(output));
        }
        inventory_errors = output.stderr;
        Some(TestOutput::inventory(&output.stdout)?)
    } else {
        None
    };
    let mut output =
        super::scheduler::execute_observed(&observed, stream_stdout && !nextest, stream_stderr)?;
    inventory_errors.extend(output.stderr);
    output.stderr = inventory_errors;
    let parse = (|| {
        let mut counts = inventory.unwrap_or_default();
        for line in output.stdout.split_inclusive(|byte| *byte == b'\n') {
            if line.len() > 1024 * 1024 {
                return Err(io::Error::other("test observation line exceeds bound"));
            }
            if nextest {
                counts.nextest(line)?;
            } else {
                counts.doctest(line)?;
            }
        }
        counts.publish()?;
        for (name, digest) in &counts.passed {
            umpire_backend_process::work::record(
                "tests-pass",
                &format!("{id}\t{}\t{digest}", encode_path(Path::new(name))),
            )?;
        }
        umpire_backend_process::work::record(
            "tests-finish",
            &format!(
                "{id}\t{}\t{}\t{}\t{}",
                counts.discovered, counts.executed, counts.ignored, counts.skipped
            ),
        )
    })();
    // A failed runner retains its exact status, including malformed or interrupted reports.
    if nextest || stream_stdout {
        output.stdout.clear();
    }
    if output.status.success()
        && let Err(error) = parse
    {
        io::stdout().write_all(&output.stdout)?;
        io::stderr().write_all(&output.stderr)?;
        return Err(error);
    }
    Ok(Some(output))
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct GoEnvironment {
    #[serde(rename = "GOFLAGS")]
    flags: String,
}

fn observe_go(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
) -> io::Result<std::process::Output> {
    let id = umpire_backend_process::work::begin_tests()?;
    let environment = super::scheduler::execute_inner(
        &super::go_command::environment(command)?,
        false,
        stream_stderr,
        None,
    )?;
    if !environment.status.success() {
        return display_go(environment, stream_stdout, false);
    }
    if environment.stdout.len() > 64 * 1024 {
        return Err(io::Error::other("Go environment observation exceeds bound"));
    }
    let GoEnvironment { flags } = serde_json::from_slice(&environment.stdout)?;
    let mut pinned = copy_command(command);
    pinned.args(command.get_args()).env("GOFLAGS", if flags.is_empty() { " " } else { &flags });
    let observed = super::go_command::execution(&pinned)?;
    let mut listing = super::scheduler::execute_inner(
        &super::go_command::inventory(&pinned)?,
        false,
        stream_stderr,
        None,
    )?;
    let mut diagnostics = environment.stderr;
    diagnostics.extend(listing.stderr);
    listing.stderr = diagnostics;
    if !listing.status.success() {
        return display_go(listing, stream_stdout, true);
    }
    let inventory = super::go_counts::Inventory::parse(&listing.stdout)?;
    let mut output =
        super::scheduler::execute_go_observed(&observed, stream_stdout, stream_stderr)?;
    listing.stderr.extend(output.stderr);
    output.stderr = listing.stderr;
    let parse = (|| {
        let observation = inventory.observe(&output.stdout)?;
        if output.status.success() && observation.failed {
            return Err(io::Error::other("successful Go process reported failed test metadata"));
        }
        let counts = observation.totals;
        umpire_backend_process::work::record(
            "tests-finish",
            &format!(
                "{id}\t{}\t{}\t{}\t{}",
                counts.discovered, counts.executed, counts.ignored, counts.skipped,
            ),
        )
    })();
    output = display_go(output, false, true)?;
    if stream_stdout {
        output.stdout.clear();
    }
    if output.status.success()
        && let Err(error) = parse
    {
        io::stdout().write_all(&output.stdout)?;
        io::stderr().write_all(&output.stderr)?;
        return Err(error);
    }
    Ok(output)
}

fn display_go(
    mut output: std::process::Output,
    stream_stdout: bool,
    json: bool,
) -> io::Result<std::process::Output> {
    if json && let Ok(decoded) = super::go_counts::output(&output.stdout) {
        output.stdout = decoded;
    }
    if stream_stdout {
        io::stdout().write_all(&output.stdout)?;
        io::stdout().flush()?;
        output.stdout.clear();
    }
    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn event(directory: &Path, slot: u64, record: &str) -> io::Result<()> {
        fs::write(directory.join(format!("{slot}.claim")), b"")?;
        fs::write(directory.join(format!("{slot}.event")), record)
    }

    #[test]
    fn missing_observations_remain_unknown_instead_of_zero() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        assert_eq!(collect(directory.path())?, WorkCounts::default());
        assert!(TestOutput::default().publish().is_err());
        Ok(())
    }

    #[test]
    fn observed_test_commands_and_real_launches_have_separate_counts() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        event(directory.path(), 0, "1\ttests-begin\t100-0\n")?;
        event(directory.path(), 1, "1\ttests-finish\t100-0\t7\t4\t1\t2\n")?;
        let configuration = "a".repeat(64);
        event(directory.path(), 2, &format!("1\ttlc-attempt\t100-0\t{configuration}\n"))?;
        event(directory.path(), 3, "1\ttlc-finish\t100-0\t1\n")?;
        event(directory.path(), 4, &format!("1\ttlc-attempt\t100-1\t{configuration}\n"))?;
        event(directory.path(), 5, "1\ttlc-finish\t100-1\t1\n")?;
        event(directory.path(), 6, &format!("1\ttlc-attempt\t101-0\t{configuration}\n"))?;
        event(directory.path(), 7, "1\ttlc-finish\t101-0\t1\n")?;
        event(directory.path(), 8, "1\ttlc-attempt\t102-0\tprobe\n")?;
        event(directory.path(), 9, "1\ttlc-finish\t102-0\t1\n")?;
        assert_eq!(
            collect(directory.path())?,
            WorkCounts {
                discovered: Some(7),
                executed: Some(4),
                ignored: Some(1),
                skipped: Some(2),
                tlc_launches: Some(4),
                tlc_configurations: Some(1),
                ..WorkCounts::default()
            }
        );
        Ok(())
    }

    #[test]
    fn supervisor_fork_and_unconfirmed_completion_do_not_claim_jvm_launches() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        event(directory.path(), 0, "1\ttlc-attempt\t100-0\t-\n")?;
        assert!(collect(directory.path()).is_err());
        event(directory.path(), 1, "1\ttlc-finish\t100-0\t0\n")?;
        assert_eq!(collect(directory.path())?.tlc_launches, None);
        Ok(())
    }

    #[test]
    fn partial_unknown_and_overflowed_events_refuse_counts() -> io::Result<()> {
        for record in [
            "truncated",
            "1\tunknown\t0\n",
            "1\ttests\tbad\t1\t0\t0\n",
            "1\ttlc-attempt\t100-0\tincorrect-hash\n",
        ] {
            let directory = tempfile::tempdir()?;
            event(directory.path(), 0, record)?;
            assert!(collect(directory.path()).is_err());
        }
        let directory = tempfile::tempdir()?;
        fs::write(directory.path().join("0.claim"), b"")?;
        assert!(collect(directory.path()).is_err());
        let directory = tempfile::tempdir()?;
        event(directory.path(), 0, &format!("1\ttests\t{}\t0\t0\t0\n", u64::MAX))?;
        event(directory.path(), 1, "1\ttests\t1\t0\t0\t0\n")?;
        assert!(collect(directory.path()).is_err());
        Ok(())
    }

    fn inventory() -> &'static [u8] {
        br#"{"test-count":3,"rust-suites":{"control":{"package-name":"control","binary-name":"binary","status":"listed","testcases":{"runs":{"ignored":false,"filter-match":{"status":"matches"}},"ignored":{"ignored":true,"filter-match":{"status":"mismatch","reason":"ignored"}},"filtered":{"ignored":false,"filter-match":{"status":"mismatch","reason":"expression"}}}}}}"#
    }

    #[test]
    fn structured_receipts_refuse_missing_partial_and_inconsistent_success_counts() -> io::Result<()>
    {
        let valid = serde_json::json!({"stage":"tests","state":"success","count_authority":"exclusive-stage","counts":{
            "discovered":3,"executed":1,"ignored":1,"skipped":1,"tlc_launches":null,"tlc_configurations":null},"count_error":null});
        validate_receipt(&valid)?;
        for counts in [
            serde_json::Value::Null,
            serde_json::json!({}),
            serde_json::json!({"executed":1}),
            serde_json::json!("unknown"),
        ] {
            let mut value = valid.clone();
            value["counts"] = counts;
            assert!(validate_receipt(&value).is_err());
        }
        let mut missing = valid.clone();
        missing.as_object_mut().ok_or_else(|| io::Error::other("receipt object"))?.remove("counts");
        assert!(validate_receipt(&missing).is_err());
        let mut inconsistent = valid.clone();
        inconsistent["counts"]["discovered"] = 99.into();
        assert!(validate_receipt(&inconsistent).is_err());
        let mut standalone = valid;
        standalone["stage"] = "standalone".into();
        assert!(validate_receipt(&standalone).is_err());
        standalone["child_count_receipts"] = serde_json::json!([]);
        assert!(validate_receipt(&standalone).is_err());
        Ok(())
    }

    #[test]
    fn stage_collector_uses_the_count_boundary_for_successful_receipts() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        fs::create_dir(directory.path().join("framework"))?;
        fs::write(directory.path().join("framework/tests.claim"), b"")?;
        let path = directory.path().join("framework/tests.json");
        let mut value = serde_json::json!({"stage":"tests","scope":"framework","state":"success","exit":0,"seconds":0.0,
            "count_authority":"exclusive-stage","count_error":null,"counts":WorkCounts::default()});
        for counts in [
            serde_json::Value::Null,
            serde_json::json!({}),
            serde_json::to_value(WorkCounts::default())?,
        ] {
            value["counts"] = counts;
            fs::write(&path, serde_json::to_vec(&value)?)?;
            assert!(
                super::super::scheduler::collect(
                    directory.path(),
                    &["framework/tests".into()],
                    true
                )
                .is_err()
            );
        }
        value["counts"] = serde_json::json!({"discovered":1,"executed":1,"ignored":0,"skipped":0,"tlc_launches":null,"tlc_configurations":null});
        fs::write(&path, serde_json::to_vec(&value)?)?;
        super::super::scheduler::collect(directory.path(), &["framework/tests".into()], true)?;
        value["counts"]["discovered"] = 2.into();
        fs::write(&path, serde_json::to_vec(&value)?)?;
        assert!(
            super::super::scheduler::collect(directory.path(), &["framework/tests".into()], true)
                .is_err()
        );
        Ok(())
    }

    #[test]
    fn inventory_requires_exact_cardinality_and_complete_selection_fields() -> io::Result<()> {
        for field in ["test-count", "rust-suites"] {
            let mut value: serde_json::Value = serde_json::from_slice(inventory())?;
            value
                .as_object_mut()
                .ok_or_else(|| io::Error::other("inventory object"))?
                .remove(field);
            assert!(TestOutput::inventory(&serde_json::to_vec(&value)?).is_err());
        }
        for count in [serde_json::json!("3"), serde_json::json!(4), serde_json::Value::Null] {
            let mut value: serde_json::Value = serde_json::from_slice(inventory())?;
            value["test-count"] = count;
            assert!(TestOutput::inventory(&serde_json::to_vec(&value)?).is_err());
        }
        let empty = TestOutput::inventory(br#"{"test-count":0,"rust-suites":{}}"#)?;
        assert!(empty.publish().is_err());
        Ok(())
    }

    #[test]
    fn only_observed_passes_link_the_admitted_test_binary_identity() -> io::Result<()> {
        let binary = tempfile::NamedTempFile::new()?;
        fs::write(binary.path(), b"admitted test binary")?;
        let digest = umpire_backend_process::bytes_sha256(b"admitted test binary");
        let mut value: serde_json::Value = serde_json::from_slice(inventory())?;
        value["rust-suites"]["control"]["binary-path"] = serde_json::json!(binary.path());
        value["rust-suites"]["control"]["package-name"] = "umpire-tlc".into();
        value["rust-suites"]["control"]["binary-name"] = "umpire_tlc".into();
        for event in ["ok", "failed"] {
            let mut counts = TestOutput::inventory(&serde_json::to_vec(&value)?)?;
            assert!(counts.passed.is_empty());
            counts.nextest(&serde_json::to_vec(&serde_json::json!({
                "type":"test", "event":event, "name":"umpire-tlc::umpire_tlc$runs",
            }))?)?;
            assert_eq!(counts.executed, 1);
            if event == "ok" {
                assert_eq!(counts.passed.get("umpire-tlc::umpire_tlc$runs"), Some(&digest));
            } else {
                assert!(counts.passed.is_empty());
            }
        }
        Ok(())
    }

    #[test]
    fn selected_tests_are_not_substituted_for_observed_execution() -> io::Result<()> {
        let mut counts = TestOutput::inventory(inventory())?;
        assert!(counts.publish().is_err());
        counts.nextest(
            br#"{"type":"suite","event":"ok","passed":100,"ignored":100,"filtered_out":100}"#,
        )?;
        assert!(counts.publish().is_err());
        counts.nextest(br#"{"type":"test","event":"ok","name":"control::binary$runs"}"#)?;
        counts.publish()?;
        assert_eq!(
            (counts.discovered, counts.executed, counts.ignored, counts.skipped),
            (3, 1, 1, 1)
        );
        assert!(
            counts
                .nextest(br#"{"type":"test","event":"ok","name":"control::binary$runs"}"#)
                .is_err()
        );
        Ok(())
    }

    #[test]
    fn doctest_summary_requires_every_actual_count() -> io::Result<()> {
        let mut counts = TestOutput::default();
        counts.doctest(b"running 3 tests\n")?;
        counts.doctest(b"test result: ok. 2 passed; 0 failed; 1 ignored; 0 measured; 3 filtered out; finished in 0.1s\n")?;
        assert_eq!(
            (counts.discovered, counts.executed, counts.ignored, counts.skipped),
            (6, 2, 1, 3)
        );
        assert!(counts.doctest(b"test result: ok. 2 passed; finished in 0.1s").is_err());
        assert!(
            counts
                .doctest(
                    b"test result: ok. 2 passed; 0 failed; 0 ignored; 1 measured; 0 filtered out"
                )
                .is_err()
        );
        Ok(())
    }

    #[test]
    fn duplicate_and_unfinished_test_producers_are_refused() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        event(directory.path(), 0, "1\ttests-begin\t100-0\n")?;
        assert!(collect(directory.path()).is_err());
        event(directory.path(), 1, "1\ttests-finish\t100-0\t7\t4\t1\t2\n")?;
        assert_eq!(collect(directory.path())?.discovered, Some(7));
        event(directory.path(), 2, "1\ttests-finish\t100-0\t7\t4\t1\t2\n")?;
        assert!(collect(directory.path()).is_err());
        Ok(())
    }
    #[test]
    fn observation_child() -> io::Result<()> {
        let Ok(mode) = std::env::var("WORK_OBSERVATION_CONTROL") else { return Ok(()) };
        if mode == "missing" {
            assert!(super::super::scheduler::scheduled("tests", || Ok(0)).is_err());
            return Ok(());
        }
        super::super::scheduler::scheduled("tests", || {
            let mut command = Command::new("/bin/sh");
            if let Some(manifest) = std::env::var_os("WORK_OBSERVATION_MANIFEST") {
                command = Command::new("mise");
                command.args(["exec", "--", "cargo", "nextest", "run", "--manifest-path"])
                    .arg(manifest).args(["-E", "test(=runs) or test(=ignored)"]);
            } else if mode == "failure" {
                command.args(["-c", "if [ \"$1\" = list ]; then printf '%s\n' \"$WORK_CONTROL_INVENTORY\"; else printf malformed; printf real-diagnostic >&2; exit 37; fi", "nextest", "run"])
                    .env("WORK_CONTROL_INVENTORY", std::str::from_utf8(inventory()).map_err(io::Error::other)?);
            } else {
                command.args(["-c", "if [ \"$1\" = list ]; then printf '%s\n' \"$WORK_CONTROL_INVENTORY\"; else printf '%s\n' '{\"type\":\"test\",\"event\":\"ok\",\"name\":\"control::binary$runs\"}'; fi", "nextest", "run"])
                    .env("WORK_CONTROL_INVENTORY", std::str::from_utf8(inventory()).map_err(io::Error::other)?);

            }
            let output = observe(&command)?.ok_or_else(|| io::Error::other("missing test observer"))?;
            assert!(output.stdout.is_empty(), "JSON escaped private capture");
            if !output.status.success() {
                assert_eq!(output.stderr, b"real-diagnostic");
                return Ok(output.status.code().unwrap_or(1));
            }
            let mut docs = Command::new("/bin/sh");
            if let Some(manifest) = std::env::var_os("WORK_OBSERVATION_MANIFEST") {
                docs = Command::new("mise");
                docs.args(["exec", "--", "cargo", "test", "--doc", "--manifest-path"]).arg(manifest);
            } else {
                docs.args(["-c", "printf '%s\n' 'running 2 tests' 'test result: ok. 1 passed; 0 failed; 1 ignored; 0 measured; 2 filtered out; finished in 0.1s'", "test", "--doc"]);
            }
            let output = observe(&docs)?.ok_or_else(|| io::Error::other("missing doctest observer"))?;
            assert!(output.status.success());
            Ok(0)
        }).map(|code| assert_eq!(code, if mode == "failure" { 37 } else { 0 }))
    }

    fn child(directory: &Path, scope: &str, mode: &str) -> io::Result<()> {
        let status = Command::new(std::env::current_exe()?)
            .args(["--exact", "gate::work_counts::tests::observation_child", "--nocapture"])
            .env("WORK_OBSERVATION_CONTROL", mode)
            .env("UMPIRE_GATE_RECEIPTS", directory)
            .env("UMPIRE_GATE_SCOPE", scope)
            .env("UMPIRE_GATE_STAGE", "tests")
            .env_remove("UMPIRE_GATE_TEST_CARRIER")
            .status()?;
        assert!(status.success());
        Ok(())
    }

    #[test]
    fn stage_receipt_combines_nextest_and_doctests_without_cross_scope_or_invocation_counts()
    -> io::Result<()> {
        let first = tempfile::tempdir()?;
        let second = tempfile::tempdir()?;
        child(first.path(), "framework", "success")?;
        child(first.path(), "extension", "success")?;
        child(second.path(), "framework", "success")?;
        for (directory, scope) in
            [(first.path(), "framework"), (first.path(), "extension"), (second.path(), "framework")]
        {
            let value: serde_json::Value =
                serde_json::from_slice(&fs::read(directory.join(scope).join("tests.json"))?)?;
            assert_eq!(value["state"], "success");
            assert_eq!(value["counts"]["discovered"], 7);
            assert_eq!(value["counts"]["executed"], 2);
            assert_eq!(value["counts"]["ignored"], 2);
            assert_eq!(value["counts"]["skipped"], 3);
            assert!(value["counts"]["tlc_launches"].is_null());
            assert!(value["count_error"].is_null());
        }
        Ok(())
    }

    #[test]
    fn malformed_counts_keep_the_real_failed_runner_status() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        child(directory.path(), "framework", "failure")?;
        let value: serde_json::Value =
            serde_json::from_slice(&fs::read(directory.path().join("framework/tests.json"))?)?;
        assert_eq!(value["exit"], 37);
        assert_eq!(value["state"], "failed");
        assert!(value["counts"].is_null());
        assert!(value["count_error"].as_str().is_some_and(|value| value.contains("completion")));
        Ok(())
    }

    #[test]
    fn rejected_controller_child() -> io::Result<()> {
        let Some(marker) = std::env::var_os("CONTROLLER_ROLE_MARKER") else { return Ok(()) };
        assert!(!owns_controller()?);
        fs::remove_file(&marker)?;
        std::os::unix::fs::symlink(std::env::current_exe()?, &marker)?;
        assert!(!owns_controller()?, "a rejected test carrier gained controller admission later");
        Ok(())
    }

    #[test]
    fn test_carrier_rejection_is_frozen_before_configured_path_can_change() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        let marker = directory.path().join("controller");
        std::os::unix::fs::symlink("/bin/true", &marker)?;
        let output = Command::new(std::env::current_exe()?)
            .args(["--exact", "gate::work_counts::tests::rejected_controller_child", "--nocapture"])
            .env("CONTROLLER_ROLE_MARKER", &marker)
            .env("UMPIRE_GATE_EXECUTABLE", &marker)
            .env_remove("UMPIRE_GATE_CONTROLLER_EXECUTABLE")
            .output()?;
        assert!(output.status.success(), "{output:?}");
        Ok(())
    }

    #[test]
    fn streaming_child() -> io::Result<()> {
        let Ok(kind) = std::env::var("WORK_STREAM_KIND") else { return Ok(()) };
        let mut command = Command::new("/bin/sh");
        let script = if kind == "nextest" {
            "if [ \"$1\" = list ]; then printf '%s\n' \"$WORK_CONTROL_INVENTORY\"; else printf early-observed >&2; while [ ! -f \"$WORK_STREAM_RELEASE\" ]; do sleep 0.01; done; printf '%s\n' '{\"type\":\"test\",\"event\":\"ok\",\"name\":\"control::binary$runs\"}'; fi"
        } else {
            "printf '%s\n' 'running 1 test' early-observed; while [ ! -f \"$WORK_STREAM_RELEASE\" ]; do sleep 0.01; done; printf '%s\n' 'test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.1s'"
        };
        command.args(["-c", script]).env(
            "WORK_CONTROL_INVENTORY",
            std::str::from_utf8(inventory()).map_err(io::Error::other)?,
        );
        if kind == "nextest" {
            command.args(["nextest", "run"]);
        } else {
            command.args(["test", "--doc"]);
        }
        let output = observe_streamed(&command, None, true, true)?
            .ok_or_else(|| io::Error::other("missing observation scope"))?;
        assert!(output.status.success());
        assert_eq!(output.stdout, b"");
        assert_eq!(output.stderr, b"");
        Ok(())
    }

    #[test]
    fn observed_tools_stream_human_channels_while_pending_without_returning_them_twice()
    -> io::Result<()> {
        for kind in ["nextest", "doctest"] {
            let directory = tempfile::tempdir()?;
            fs::create_dir_all(directory.path().join("work/framework/tests"))?;
            let stdout = directory.path().join("stdout");
            let stderr = directory.path().join("stderr");
            let release = directory.path().join("release");
            let mut child = Command::new(std::env::current_exe()?)
                .args(["--exact", "gate::work_counts::tests::streaming_child", "--nocapture"])
                .env("WORK_STREAM_KIND", kind)
                .env("WORK_STREAM_RELEASE", &release)
                .env("UMPIRE_GATE_RECEIPTS", directory.path())
                .env("UMPIRE_GATE_SCOPE", "framework")
                .env("UMPIRE_GATE_STAGE", "tests")
                .stdout(fs::File::create(&stdout)?)
                .stderr(fs::File::create(&stderr)?)
                .spawn()?;
            let channel = if kind == "nextest" { &stderr } else { &stdout };
            let mut early = false;
            for _ in 0..200 {
                if fs::read_to_string(channel)?.contains("early-observed") {
                    early = true;
                    break;
                }
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
            let pending = child.try_wait()?.is_none();
            fs::write(release, b"release")?;
            assert!(child.wait()?.success());
            assert!(early && pending, "observed {kind} diagnostics were buffered");
            assert_eq!(fs::read_to_string(channel)?.matches("early-observed").count(), 1);
            assert!(!fs::read_to_string(&stdout)?.contains("\"type\":\"test\""));
            let counts = collect(&directory.path().join("work/framework/tests"))?;
            assert_eq!(counts.executed, Some(1));
        }
        Ok(())
    }

    #[test]
    fn successful_test_stage_cannot_omit_its_producer() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        child(directory.path(), "framework", "missing")?;
        let value: serde_json::Value =
            serde_json::from_slice(&fs::read(directory.path().join("framework/tests.json"))?)?;
        assert_eq!(value["state"], "failed");
        assert_eq!(value["exit"], 1);
        assert!(
            value["count_error"]
                .as_str()
                .is_some_and(|value| value.contains("missing test producer"))
        );
        Ok(())
    }

    #[test]
    fn doctest_color_is_normalized_only_for_private_observation() -> io::Result<()> {
        let mut counts = TestOutput::default();
        counts.doctest(b"\x1b[1mrunning 1 test\x1b[0m\n")?;
        counts.doctest(b"test result: \x1b[32mok\x1b[0m. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out\n")?;
        counts.publish()?;
        assert_eq!(counts.executed, 1);
        Ok(())
    }

    #[test]
    fn doctests_refuse_missing_duplicate_and_mismatched_summaries() -> io::Result<()> {
        let mut missing = TestOutput::default();
        missing.doctest(b"running 1 test\n")?;
        assert!(missing.publish().is_err());
        assert!(
            missing
                .doctest(
                    b"test result: ok. 2 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out\n"
                )
                .is_err()
        );
        let mut duplicate = TestOutput::default();
        duplicate.doctest(b"running 0 tests\n")?;
        duplicate.doctest(
            b"test result: ok. 0 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out\n",
        )?;
        duplicate.publish()?;
        assert!(
            duplicate
                .doctest(
                    b"test result: ok. 0 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out\n"
                )
                .is_err()
        );
        Ok(())
    }

    fn child_records() -> &'static [u8] {
        b"{\"stage\":\"tests\",\"scope\":\"framework\",\"state\":\"success\",\"exit\":0,\"seconds\":0.0,\"count_authority\":\"exclusive-stage\",\"count_error\":null,\"counts\":{\"discovered\":3,\"executed\":3,\"ignored\":0,\"skipped\":0,\"tlc_launches\":null,\"tlc_configurations\":null}}\n"
    }

    #[test]
    fn child_authority_paths_preserve_tabs_and_non_unicode_bytes() -> io::Result<()> {
        for name in [b"child\tpath.json".as_slice(), b"child-\xff.json".as_slice()] {
            let directory = tempfile::tempdir()?;
            let file = directory.path().join(std::ffi::OsString::from_vec(name.to_vec()));
            fs::write(&file, child_records())?;
            let digest = umpire_backend_process::bytes_sha256(child_records());
            let observations = tempfile::tempdir()?;
            event(
                observations.path(),
                0,
                &format!("1\tchild-receipts\t{}\t{digest}\n", encode_path(&file)),
            )?;
            assert_eq!(decode_path(&encode_path(&file))?, file);
            let counts = collect(observations.path())?;
            assert_eq!(
                counts.child_receipts[0]["path_bytes"],
                serde_json::json!(file.as_os_str().as_bytes())
            );
            let parent = serde_json::json!({"stage":"standalone","state":"success","count_authority":"exclusive-stage",
                "counts":WorkCounts::default(),"count_error":null,"child_count_receipts":counts.child_receipts});
            validate_receipt(&parent)?;
        }
        for malformed in ["", "0", "zz", "00", "61"] {
            assert!(decode_path(malformed).is_err());
        }
        Ok(())
    }

    #[test]
    fn child_count_authority_is_embedded_without_inclusive_parent_summing() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        let child = tempfile::NamedTempFile::new()?;
        fs::write(child.path(), child_records())?;
        let digest = umpire_backend_process::bytes_sha256(&fs::read(child.path())?);
        event(
            directory.path(),
            0,
            &format!("1\tchild-receipts\t{}\t{digest}\n", encode_path(child.path())),
        )?;
        let observed = collect(directory.path())?;
        assert_eq!(observed.executed, None);
        assert_eq!(observed.child_receipts[0]["stages"][0]["counts"]["executed"], 3);
        let mut parent = serde_json::json!({"stage":"standalone","state":"success","count_authority":"exclusive-stage",
            "counts":WorkCounts::default(),"count_error":null,"child_count_receipts":observed.child_receipts});
        validate_receipt(&parent)?;
        drop(child);
        validate_receipt(&parent)?;
        parent["child_count_receipts"][0]["sha256"] = "bad-digest".into();
        assert!(validate_receipt(&parent).is_err());
        assert_eq!(observed.child_receipts[0]["stages"][0]["counts"]["executed"], 3);
        Ok(())
    }
}
