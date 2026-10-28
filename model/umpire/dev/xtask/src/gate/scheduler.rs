//! Invocation receipts around the mise-owned task graph.
use super::{budget_attempts, work_counts};
use std::{
    collections::BTreeSet,
    env, fs,
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    path::{Path, PathBuf},
    process::Command,
    time::{Instant, SystemTime, UNIX_EPOCH},
};

fn tasks(mut command: Command) -> io::Result<serde_json::Value> {
    let output = command.args(["tasks", "ls", "--json"]).output()?;
    if !output.status.success() {
        io::stderr().write_all(&output.stderr)?;
        return Err(io::Error::other("mise stage inventory failed"));
    }
    let tasks: serde_json::Value = serde_json::from_slice(&output.stdout)?;
    Ok(tasks)
}

/// Read stage order and prerequisite metadata from actual mise task definitions.
pub fn listing(command: Command, prefix: &str) -> io::Result<Vec<(String, Vec<String>)>> {
    listing_from_value(&tasks(command)?, prefix)
}

fn listing_from_value(
    tasks: &serde_json::Value,
    prefix: &str,
) -> io::Result<Vec<(String, Vec<String>)>> {
    let tasks = tasks.as_array().ok_or_else(|| io::Error::other("invalid mise task inventory"))?;
    let mut stages = Vec::new();
    let mut names = BTreeSet::new();
    let mut orders = BTreeSet::new();
    for task in tasks {
        if !task["name"].as_str().is_some_and(|name| name.starts_with(prefix)) {
            continue;
        }
        let Some((name, order)) = stage_metadata(task)? else { continue };
        if name.is_empty() || !name.bytes().all(|b| b.is_ascii_alphanumeric() || b == b'-') {
            return Err(io::Error::other("invalid mise stage name"));
        }
        let scope = task["env"]
            .as_array()
            .into_iter()
            .flatten()
            .filter_map(serde_json::Value::as_str)
            .find_map(|value| value.strip_prefix("UMPIRE_GATE_SCOPE="))
            .unwrap_or("");
        if !scope.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-') {
            return Err(io::Error::other("invalid mise stage scope"));
        }
        let identity = if scope.is_empty() { name.to_owned() } else { format!("{scope}/{name}") };
        if !names.insert(identity.clone()) || !orders.insert((scope.to_owned(), order)) {
            return Err(io::Error::other("duplicate mise stage inventory"));
        }
        let mut dependencies = task["depends"]
            .as_array()
            .ok_or_else(|| io::Error::other("invalid mise stage dependencies"))?
            .iter()
            .map(|value| {
                let dependency = value
                    .as_str()
                    .ok_or_else(|| io::Error::other("nonliteral mise stage dependency"))?;
                Ok(dependency.strip_prefix(prefix).unwrap_or(dependency).to_owned())
            })
            .collect::<io::Result<Vec<_>>>()?;
        if let Some(wait_for) = task["wait_for"].as_array() {
            for dependency in wait_for {
                let dependency = dependency
                    .as_str()
                    .ok_or_else(|| io::Error::other("nonliteral mise wait_for dependency"))?;
                if prefix.is_empty() || dependency.starts_with(prefix) {
                    dependencies.push(format!("wait_for:{dependency}"));
                }
            }
        }
        stages.push((scope.to_owned(), order, identity, dependencies));
    }
    if stages.is_empty() {
        return Err(io::Error::other("empty mise stage inventory"));
    }
    stages.sort();
    Ok(stages.into_iter().map(|(_, _, name, dependencies)| (name, dependencies)).collect())
}

fn stage_metadata(task: &serde_json::Value) -> io::Result<Option<(&str, usize)>> {
    let directives = task["env"]
        .as_array()
        .ok_or_else(|| io::Error::other("invalid mise task env directives"))?;
    let mut stage = None;
    let mut order = None;
    for directive in directives {
        let directive = directive
            .as_str()
            .ok_or_else(|| io::Error::other("invalid mise task env directive"))?;
        let Some((key, value)) = directive.split_once('=') else { continue };
        match key {
            "UMPIRE_GATE_STAGE" if stage.replace(value).is_some() => {
                return Err(io::Error::other("duplicate mise stage metadata"));
            }
            "UMPIRE_GATE_ORDER" if order.replace(value).is_some() => {
                return Err(io::Error::other("duplicate mise stage order"));
            }
            _ => (),
        }
    }
    match (stage, order) {
        (None, None) => Ok(None),
        (Some(stage), Some(order)) => {
            let order =
                order.parse::<usize>().map_err(|_| io::Error::other("invalid mise stage order"))?;
            Ok(Some((stage, order)))
        }
        _ => Err(io::Error::other("incomplete mise stage metadata")),
    }
}

/// Read qualified stage identities in deterministic logical order.
pub fn inventory(command: Command, prefix: &str) -> io::Result<Vec<String>> {
    Ok(listing(command, prefix)?.into_iter().map(|(name, _)| name).collect())
}

/// Create an exclusively owned invocation receipt directory beneath the selected parent.
pub fn directory(parent: &Path) -> io::Result<PathBuf> {
    fs::create_dir_all(parent)?;
    let nonce = SystemTime::now().duration_since(UNIX_EPOCH).map_err(io::Error::other)?.as_nanos();
    let directory = parent.join(format!("gate-{}-{nonce}", std::process::id()));
    fs::create_dir(&directory)?;
    Ok(directory)
}

/// Claim and publish exactly one scheduled leaf execution.
pub fn scheduled(name: &str, run: impl FnMut() -> io::Result<i32>) -> io::Result<i32> {
    scheduled_with(name, run, None)
}

pub(super) fn skipped(name: &str, reason: &str) -> io::Result<i32> {
    scheduled_with(name, || Ok(0), Some(reason))
}

fn scheduled_with(
    name: &str,
    mut run: impl FnMut() -> io::Result<i32>,
    skipped: Option<&str>,
) -> io::Result<i32> {
    if name.is_empty() || !name.bytes().all(|b| b.is_ascii_alphanumeric() || b == b'-') {
        return Err(io::Error::other("invalid scheduled stage name"));
    }
    let directory = env::var_os("UMPIRE_GATE_RECEIPTS")
        .map(PathBuf::from)
        .filter(|path| path.is_absolute() && path.is_dir())
        .ok_or_else(|| io::Error::other("missing invocation-local stage receipt directory"))?;
    let scope = env::var("UMPIRE_GATE_SCOPE").unwrap_or_default();
    if !scope.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-') {
        return Err(io::Error::other("invalid stage receipt scope"));
    }
    let observations = directory.join("work").join(&scope).join(name);
    let directory = if scope.is_empty() { directory } else { directory.join(&scope) };
    fs::create_dir_all(&directory)?;
    let claim = directory.join(format!("{name}.claim"));
    if let Err(error) = fs::OpenOptions::new().write(true).create_new(true).open(claim) {
        if error.kind() == io::ErrorKind::AlreadyExists {
            fs::write(
                directory.join(format!("{name}.duplicate")),
                b"duplicate stage invocation\n",
            )?;
        }
        return Err(io::Error::other(format!("stage {name} receipt admission: {error}")));
    }
    fs::create_dir_all(
        observations.parent().ok_or_else(|| io::Error::other("missing work parent"))?,
    )?;
    fs::create_dir(&observations)?;
    let attempts_directory = directory
        .parent()
        .filter(|_| !scope.is_empty())
        .unwrap_or(&directory)
        .join("budget/attempts")
        .join(&scope)
        .join(name);
    fs::create_dir_all(&attempts_directory)?;
    let started = Instant::now();
    let boundary = budget_attempts::Boundary::begin(&observations)?;
    let result = run();
    let attempts =
        boundary.finish(&observations, 1, &attempts_directory.join("1.json")).and_then(|attempt| {
            let attempts = vec![attempt];
            budget_attempts::verify(&observations, &attempts)?;
            Ok(attempts)
        });
    let counts = attempts
        .as_ref()
        .map_err(|error| io::Error::other(error.to_string()))
        .and_then(|attempts| {
            work_counts::collect_selected(&observations, &attempts[0].event_names())
        })
        .and_then(|counts| {
            if name == "tests" && skipped.is_none() && counts.executed.is_none() {
                return Err(io::Error::other("missing test producer observations"));
            }
            Ok(counts)
        });
    let count_error = counts.as_ref().err().map(ToString::to_string);
    let mut result = match result {
        Ok(0) if count_error.is_some() => Err(io::Error::other(format!(
            "stage {name} work observations: {}",
            count_error.as_deref().unwrap_or_default()
        ))),
        other => other,
    };
    let mut receipt = serde_json::json!({
        "stage": name, "scope": scope,
        "state": if skipped.is_some() { "skipped" } else if result.as_ref().is_ok_and(|code| *code == 0) { "success" } else { "failed" },
        "reason": skipped,
        "exit": result.as_ref().copied().unwrap_or(1),
        "error": result.as_ref().err().map(ToString::to_string),
        "seconds": started.elapsed().as_secs_f64(),
        "attempt_event_manifests": attempts.ok(),
        "child_count_receipts": counts.as_ref().ok().map(|counts| &counts.child_receipts),
        "counts": counts.ok(), "count_error": count_error,
        "count_authority": "exclusive-stage",
    });
    if let Err(error) = work_counts::validate_receipt(&receipt) {
        receipt["count_error"] = error.to_string().into();
        if result.as_ref().is_ok_and(|code| *code == 0) {
            receipt["state"] = "failed".into();
            receipt["exit"] = 1.into();
            receipt["error"] = error.to_string().into();
            result = Err(error);
        }
    }
    let partial = directory.join(format!("{name}.partial"));
    let mut file = fs::OpenOptions::new().write(true).create_new(true).open(&partial)?;
    serde_json::to_writer(&mut file, &receipt)?;
    file.write_all(b"\n")?;
    file.sync_all()?;
    fs::hard_link(&partial, directory.join(format!("{name}.json")))?;
    fs::remove_file(partial)?;
    result
}

pub(super) fn collect(directory: &Path, expected: &[String], success: bool) -> io::Result<Vec<u8>> {
    let mut paths = Vec::new();
    for entry in fs::read_dir(directory)? {
        let path = entry?.path();
        if path.file_name().is_some_and(|name| name == "work" || name == "budget") {
            continue;
        }
        if path.is_dir() {
            let scope = path.file_name().and_then(|n| n.to_str()).unwrap_or_default();
            if !expected.iter().any(|name| name.starts_with(&format!("{scope}/"))) {
                return Err(io::Error::other("unexpected receipt directory"));
            }
            for entry in fs::read_dir(path)? {
                paths.push(entry?.path());
            }
        } else {
            paths.push(path);
        }
    }
    for path in paths {
        let name = path
            .strip_prefix(directory)
            .ok()
            .and_then(|p| p.to_str())
            .ok_or_else(|| io::Error::other("invalid stage receipt path"))?;
        if ["stages.jsonl", "stdout.log", "stderr.log", "admission.json"].contains(&name) {
            continue;
        }
        let stage = [".json", ".claim", ".duplicate", ".partial"]
            .into_iter()
            .find_map(|suffix| name.strip_suffix(suffix))
            .ok_or_else(|| io::Error::other("unexpected stage receipt path"))?;
        if !expected.iter().any(|name| name == stage) {
            return Err(io::Error::other(format!("unexpected stage receipt: {stage}")));
        }
        if name.ends_with(".duplicate") || name.ends_with(".partial") {
            return Err(io::Error::other(format!("duplicate or partial stage receipt: {stage}")));
        }
    }
    let mut records = Vec::new();
    for name in expected {
        let path = directory.join(format!("{name}.json"));
        let claimed = directory.join(format!("{name}.claim")).is_file();
        let value = if path.exists() {
            let value: serde_json::Value = serde_json::from_slice(&fs::read(path)?)?;
            if let Some(manifests) =
                value.get("attempt_event_manifests").filter(|value| !value.is_null())
            {
                let attempts: Vec<budget_attempts::Attempt> =
                    serde_json::from_value(manifests.clone())?;
                budget_attempts::verify(&directory.join("work").join(name), &attempts)?;
                for (index, attempt) in attempts.iter().enumerate() {
                    let path = directory
                        .join("budget/attempts")
                        .join(name)
                        .join(format!("{}.json", index + 1));
                    if &budget_attempts::Attempt::read(&path)? != attempt {
                        return Err(io::Error::other(
                            "attempt manifest differs from its stage receipt",
                        ));
                    }
                }
                match work_counts::collect(&directory.join("work").join(name)) {
                    Ok(observed) if value["counts"] != serde_json::to_value(&observed)? => {
                        return Err(io::Error::other(
                            "stage counts differ from attempt observations",
                        ));
                    }
                    Err(error) if !value["counts"].is_null() => return Err(error),
                    _ => (),
                }
            }
            if value["stage"] != name.rsplit('/').next().unwrap_or(name)
                || !value["seconds"].as_f64().is_some_and(|n| n.is_finite() && n >= 0.0)
                || value["scope"].as_str().unwrap_or("")
                    != name.split_once('/').map_or("", |(scope, _)| scope)
                || !value["exit"].is_i64()
                || work_counts::validate_receipt(&value).is_err()
                || !claimed
            {
                return Err(io::Error::other(format!("invalid stage receipt: {name}")));
            }
            let completed = match (value["state"].as_str(), value["exit"].as_i64()) {
                (Some("success"), Some(0)) => true,
                (Some("skipped"), Some(0)) => {
                    value["reason"].as_str().is_some_and(|reason| !reason.is_empty())
                }
                (Some("failed"), Some(code)) => code != 0,
                _ => false,
            };
            if !completed {
                return Err(io::Error::other(format!("partial stage receipt: {name}")));
            }
            if success && value["exit"] != 0 {
                return Err(io::Error::other(format!("failed stage receipt: {name}")));
            }
            value
        } else {
            if success {
                return Err(io::Error::other(format!("missing stage receipt: {name}")));
            }
            serde_json::json!({"stage": name.rsplit('/').next().unwrap_or(name), "scope": name.split_once('/').map_or("", |(scope, _)| scope),
                "state": if claimed { "cancelled" } else { "not-started" },
                "exit": null, "seconds": null, "counts": null, "count_error": "stage did not complete",
                "count_authority": "exclusive-stage", "child_count_receipts": []})
        };
        serde_json::to_writer(&mut records, &value)?;
        records.push(b'\n');
    }
    Ok(records)
}

fn failed(directory: &Path, expected: &[String]) -> Option<i32> {
    expected.iter().find_map(|name| {
        let data = fs::read(directory.join(format!("{name}.json"))).ok()?;
        let value: serde_json::Value = serde_json::from_slice(&data).ok()?;
        let code = i32::try_from(value["exit"].as_i64()?).ok()?;
        (code != 0).then_some(code)
    })
}

pub(super) fn supervised(
    command: &Command,
    directory: &Path,
    expected: &[String],
) -> io::Result<i32> {
    supervised_inner(command, directory, expected, false, true)
}

pub(super) fn supervised_clean(
    command: &Command,
    directory: &Path,
    stream: bool,
) -> io::Result<i32> {
    supervised_inner(command, directory, &[], true, stream)
}

fn supervised_inner(
    command: &Command,
    directory: &Path,
    expected: &[String],
    clean: bool,
    stream: bool,
) -> io::Result<i32> {
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let lease = JvmLease::inherited_or_acquire()?;
    fs::write(
        directory.join("admission.json"),
        serde_json::to_vec_pretty(&super::admission::receipt(command))?,
    )?;
    let stdout = fs::File::create(directory.join("stdout.log"))?;
    let stderr = fs::File::create(directory.join("stderr.log"))?;
    let mut process = if clean {
        OwnedProcess::spawn_leased_clean(command, directory, &stdout, &stderr, &lease)?
    } else {
        OwnedProcess::spawn_leased(command, directory, &stdout, &stderr, &lease)?
    };
    let mut output = fs::File::open(directory.join("stdout.log"))?;
    let mut errors = fs::File::open(directory.join("stderr.log"))?;
    let code = loop {
        if stream {
            relay(&mut output, &mut io::stdout())?;
            relay(&mut errors, &mut io::stderr())?;
        }
        if let Some(code) = failed(directory, expected) {
            std::thread::sleep(std::time::Duration::from_millis(30));
            break failed(directory, expected).unwrap_or(code);
        }
        if let Some(status) = process.try_wait()? {
            break status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    };
    drop(process);
    let code = failed(directory, expected).unwrap_or(code);
    if stream || code != 0 {
        relay(&mut output, &mut io::stdout())?;
        relay(&mut errors, &mut io::stderr())?;
    }
    Ok(code)
}

/// Execute a leaf controller with owned cleanup and explicit output routing.
/// Captured channels are returned only after all owned descendants complete.
pub fn execute(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
) -> io::Result<std::process::Output> {
    execute_with_inventory(command, stream_stdout, stream_stderr, None)
}

/// Execute tests with the inventory just admitted by their scoped caller.
pub fn execute_with_inventory(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
    inventory: Option<&[u8]>,
) -> io::Result<std::process::Output> {
    let observed = if work_counts::owns_controller()? {
        work_counts::observe_streamed(command, inventory, stream_stdout, stream_stderr)?
    } else {
        None
    };
    if let Some(output) = observed {
        return Ok(output);
    }
    execute_inner(command, stream_stdout, stream_stderr, None)
}

pub(super) fn capture_clean(
    command: &Command,
    policy: umpire_backend_process::DescendantPolicy,
) -> io::Result<std::process::Output> {
    execute_inner(command, false, false, Some(policy))
}

pub(super) fn execute_inner(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
    clean_policy: Option<umpire_backend_process::DescendantPolicy>,
) -> io::Result<std::process::Output> {
    execute_captured(command, stream_stdout, stream_stderr, clean_policy, StdoutCapture::Ordinary)
}

pub(super) fn execute_observed(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
) -> io::Result<std::process::Output> {
    execute_captured(command, stream_stdout, stream_stderr, None, StdoutCapture::Retained)
}

pub(super) fn execute_go_observed(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
) -> io::Result<std::process::Output> {
    execute_captured(command, stream_stdout, stream_stderr, None, StdoutCapture::GoJson)
}

#[derive(Clone, Copy)]
enum StdoutCapture {
    Ordinary,
    Retained,
    GoJson,
}

fn execute_captured(
    command: &Command,
    stream_stdout: bool,
    stream_stderr: bool,
    clean_policy: Option<umpire_backend_process::DescendantPolicy>,
    capture: StdoutCapture,
) -> io::Result<std::process::Output> {
    use umpire_backend_process::{DescendantPolicy, OwnedProcess, run_owned};
    let retain_stdout = !matches!(capture, StdoutCapture::Ordinary);
    let go_json = matches!(capture, StdoutCapture::GoJson);
    let caller = env::current_dir()?;
    if stream_stdout && stream_stderr && !retain_stdout {
        return Ok(std::process::Output {
            status: run_owned(command, &caller)?,
            stdout: Vec::new(),
            stderr: Vec::new(),
        });
    }
    let directory = tempfile::tempdir()?;
    let stdout_path = directory.path().join("stdout.log");
    let stderr_path = directory.path().join("stderr.log");
    let stdout = fs::File::create(&stdout_path)?;
    let stderr = fs::File::create(&stderr_path)?;
    let mut process = if let Some(policy) = clean_policy {
        OwnedProcess::spawn_clean(command, &caller, &stdout, &stderr, policy)?
    } else {
        OwnedProcess::spawn(command, &caller, &stdout, &stderr, DescendantPolicy::Orchestration)?
    };
    let mut output = fs::File::open(&stdout_path)?;
    let mut errors = fs::File::open(&stderr_path)?;
    let mut go_output = go_json.then(super::go_counts::OutputRelay::default);
    let status = loop {
        if stream_stdout {
            relay_observed(&mut output, &mut io::stdout(), go_output.as_mut())?;
        }
        if stream_stderr {
            relay(&mut errors, &mut io::stderr())?;
        }
        if let Some(status) = process.try_wait()? {
            break status;
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    };
    drop(process);
    if stream_stdout {
        relay_observed(&mut output, &mut io::stdout(), go_output.as_mut())?;
        if let Some(go_output) = &mut go_output {
            go_output.finish(&mut io::stdout())?;
        }
    }
    if stream_stderr {
        relay(&mut errors, &mut io::stderr())?;
    }
    Ok(std::process::Output {
        status,
        stdout: if stream_stdout && !retain_stdout { Vec::new() } else { fs::read(stdout_path)? },
        stderr: if stream_stderr { Vec::new() } else { fs::read(stderr_path)? },
    })
}

fn relay_observed(
    input: &mut fs::File,
    output: &mut dyn Write,
    go_output: Option<&mut super::go_counts::OutputRelay>,
) -> io::Result<()> {
    use std::io::Read;
    let Some(go_output) = go_output else { return relay(input, output) };
    let mut buffer = [0; 16 * 1024];
    loop {
        let size = input.read(&mut buffer)?;
        if size == 0 {
            return Ok(());
        }
        go_output.feed(&buffer[..size], output)?;
    }
}

fn relay(input: &mut fs::File, output: &mut dyn Write) -> io::Result<()> {
    io::copy(input, output)?;
    output.flush()
}

/// Run one mise graph under lease custody and publish complete ordered stage records.
pub fn run(
    mut command: Command,
    task: &str,
    directory: &Path,
    expected: &[String],
) -> io::Result<i32> {
    command
        .args(["run", "--force", "--skip-tools", "--task-cache", "off", "--jobs", "4", task])
        .env("UMPIRE_GATE_RECEIPTS", directory)
        .env("MISE_TASK_PGID_MANAGED", "1");
    let code = supervised(&command, directory, expected)?;
    match collect(directory, expected, code == 0) {
        Ok(records) => fs::write(directory.join("stages.jsonl"), records)?,
        Err(error) => {
            // Preserve the real failed leaf status while retaining complete invalid-stage evidence.
            let mut records = Vec::new();
            for name in expected {
                let mut value = fs::read(directory.join(format!("{name}.json")))
                    .ok()
                    .and_then(|bytes| serde_json::from_slice::<serde_json::Value>(&bytes).ok())
                    .filter(|value| {
                        value["stage"] == name.rsplit('/').next().unwrap_or(name)
                            && value["exit"].is_i64()
                    })
                    .unwrap_or_else(|| {
                        serde_json::json!({
                            "stage": name.rsplit('/').next().unwrap_or(name),
                            "scope": name.split_once('/').map_or("", |(scope, _)| scope),
                            "state": "partial", "exit": null, "seconds": null,
                            "counts": null, "count_error": "partial stage receipt", "count_authority": "exclusive-stage", "child_count_receipts": [],
                        })
                    });
                value["receipt_error"] = error.to_string().into();
                serde_json::to_writer(&mut records, &value)?;
                records.push(b'\n');
            }
            fs::write(directory.join("stages.jsonl"), records)?;
            writeln!(io::stderr(), "invalid gate receipts: {error}")?;
            if code == 0 {
                return Err(error);
            }
        }
    }
    Ok(code)
}

#[cfg(test)]
mod inventory_controls {
    use super::*;
    use serde_json::json;

    fn task(name: &str, stage: &str, order: &str) -> serde_json::Value {
        json!({"name": name, "env": [format!("UMPIRE_GATE_STAGE={stage}"),
            format!("UMPIRE_GATE_ORDER={order}")], "depends": []})
    }

    #[test]
    fn upstream_env_directive_array_preserves_logical_order_and_dependencies() -> io::Result<()> {
        let value = json!([
            {"name": "gate:all", "env": [], "depends": ["gate:tests"]},
            {"name": "gate:tests", "env": ["UMPIRE_GATE_ORDER=1", "UMPIRE_GATE_STAGE=tests"],
                "depends": ["gate:proto-tools"]},
            task("gate:proto-tools", "proto-tools", "0"),
            {"name": "model:unrelated", "env": [], "depends": []}
        ]);
        assert_eq!(
            listing_from_value(&value, "gate:")?,
            vec![("proto-tools".into(), vec![]), ("tests".into(), vec!["proto-tools".into()])]
        );
        Ok(())
    }

    #[test]
    fn assumed_env_object_and_incomplete_metadata_are_refused() {
        for env in [
            json!({"UMPIRE_GATE_STAGE": "tests", "UMPIRE_GATE_ORDER": "0"}),
            json!(["UMPIRE_GATE_STAGE=tests"]),
            json!(["UMPIRE_GATE_ORDER=0"]),
            json!(["UMPIRE_GATE_STAGE=tests", "UMPIRE_GATE_ORDER=not-a-number"]),
        ] {
            let mut task = task("gate:tests", "tests", "0");
            task["env"] = env;
            assert!(listing_from_value(&json!([task]), "gate:").is_err());
        }
    }

    #[test]
    fn duplicate_metadata_and_nonliteral_dependency_are_refused() {
        for env in [
            json!(["UMPIRE_GATE_STAGE=tests", "UMPIRE_GATE_STAGE=docs", "UMPIRE_GATE_ORDER=0"]),
            json!(["UMPIRE_GATE_STAGE=tests", "UMPIRE_GATE_ORDER=0", "UMPIRE_GATE_ORDER=1"]),
        ] {
            let mut task = task("gate:tests", "tests", "0");
            task["env"] = env;
            assert!(listing_from_value(&json!([task]), "gate:").is_err());
        }
        let mut task = task("gate:tests", "tests", "0");
        task["depends"] = json!([{"task": "gate:docs", "optional": true}]);
        assert!(listing_from_value(&json!([task]), "gate:").is_err());
    }
}

#[cfg(test)]
mod execution_controls {
    use super::*;

    #[test]
    fn captured_leaf_preserves_cwd_environment_bytes_and_failed_status() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        let mut command = Command::new("/bin/sh");
        command
            .args(["-c", "test \"$PWD\" = \"$CONTROL_DIRECTORY\" || exit 9; printf '%s' \"$CONTROL_BYTES\"; printf error >&2; exit 37"])
            .current_dir(directory.path())
            .env("CONTROL_DIRECTORY", directory.path().canonicalize()?)
            .env("CONTROL_BYTES", "raw\nbytes");
        let output = execute(&command, false, false)?;
        assert_eq!(output.status.code(), Some(37));
        assert_eq!(output.stdout, b"raw\nbytes");
        assert_eq!(output.stderr, b"error");
        Ok(())
    }

    #[test]
    fn unconfigured_leaf_uses_the_callers_working_directory() -> io::Result<()> {
        let output = execute(&Command::new("/bin/pwd"), false, false)?;
        assert!(output.status.success());
        let mut expected =
            env::current_dir()?.canonicalize()?.into_os_string().into_encoded_bytes();
        expected.push(b'\n');
        assert_eq!(output.stdout, expected);
        Ok(())
    }

    #[test]
    fn captured_leaf_preserves_signal_status() -> io::Result<()> {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "kill -TERM $$"]);
        let output = execute(&command, false, false)?;
        assert_eq!(
            output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)),
            143
        );
        Ok(())
    }

    #[test]
    fn streamed_channels_are_not_returned_as_captured_output() -> io::Result<()> {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "printf output; printf error >&2; exit 37"]);
        let output = execute(&command, true, true)?;
        assert_eq!(output.status.code(), Some(37));
        assert_eq!(output.stdout, b"");
        assert_eq!(output.stderr, b"");
        Ok(())
    }
}

#[cfg(test)]
mod receipt_controls {
    use super::*;

    #[test]
    fn attempt_manifest_child() -> io::Result<()> {
        let Some(mode) = env::var_os("BUDGET_ATTEMPT_CONTROL") else { return Ok(()) };
        let code = scheduled("tests", || {
            let id = umpire_backend_process::work::begin_tests()?;
            umpire_backend_process::work::record("tests-finish", &format!("{id}\t1\t1\t0\t0"))?;
            Ok(if mode == "failure" { 37 } else { 0 })
        })?;
        assert_eq!(code, if mode == "failure" { 37 } else { 0 });
        Ok(())
    }

    #[test]
    fn scheduled_payload_publishes_and_rechecks_immutable_attempt_manifest() -> io::Result<()> {
        for mode in ["success", "failure"] {
            let root = tempfile::tempdir()?;
            let output = Command::new(env::current_exe()?)
                .args([
                    "--exact",
                    "gate::scheduler::receipt_controls::attempt_manifest_child",
                    "--nocapture",
                ])
                .env("BUDGET_ATTEMPT_CONTROL", mode)
                .env("UMPIRE_GATE_RECEIPTS", root.path())
                .env("UMPIRE_GATE_SCOPE", "framework")
                .env("UMPIRE_GATE_STAGE", "tests")
                .output()?;
            assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
            let receipt: serde_json::Value =
                serde_json::from_slice(&fs::read(root.path().join("framework/tests.json"))?)?;
            assert_eq!(receipt["exit"], if mode == "failure" { 37 } else { 0 });
            assert_eq!(receipt["attempt_event_manifests"].as_array().map(Vec::len), Some(1));
            let manifest = root.path().join("budget/attempts/framework/tests/1.json");
            assert!(manifest.is_file());
            collect(root.path(), &["framework/tests".into()], mode == "success")?;
            let receipt_path = root.path().join("framework/tests.json");
            let original = fs::read(&receipt_path)?;
            let mut changed = receipt;
            changed["counts"]["discovered"] = 2.into();
            changed["counts"]["executed"] = 2.into();
            fs::write(&receipt_path, serde_json::to_vec(&changed)?)?;
            assert!(collect(root.path(), &["framework/tests".into()], mode == "success").is_err());
            fs::write(&receipt_path, original)?;
            fs::write(&manifest, b"{}")?;
            assert!(collect(root.path(), &["framework/tests".into()], mode == "success").is_err());
        }
        Ok(())
    }

    fn write(directory: &Path, name: &str, exit: i32) -> io::Result<()> {
        let (scope, stage) = name.split_once('/').unwrap_or(("", name));
        fs::create_dir_all(directory.join(scope))?;
        fs::write(directory.join(format!("{name}.claim")), b"")?;
        fs::write(
            directory.join(format!("{name}.json")),
            serde_json::to_vec(&serde_json::json!({
                "scope": scope, "stage": stage, "state": if exit == 0 { "success" } else { "failed" },
                "exit": exit, "seconds": 0.0, "counts": work_counts::WorkCounts::default(), "count_error": null, "count_authority": "exclusive-stage",
            }))?,
        )
    }

    #[test]
    fn scoped_receipts_keep_duplicate_public_names_and_logical_order() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        write(directory.path(), "extension/fmt", 0)?;
        write(directory.path(), "framework/fmt", 0)?;
        let names = vec!["framework/fmt".into(), "extension/fmt".into()];
        let bytes = collect(directory.path(), &names, true)?;
        let rows = String::from_utf8(bytes).map_err(io::Error::other)?;
        let rows = rows
            .lines()
            .map(serde_json::from_str::<serde_json::Value>)
            .collect::<Result<Vec<_>, _>>()?;
        assert_eq!(rows[0]["scope"], "framework");
        assert_eq!(rows[1]["scope"], "extension");
        assert!(rows.iter().all(|row| row["stage"] == "fmt"));
        let file = directory.path().join("framework/fmt.json");
        let mut partial: serde_json::Value = serde_json::from_slice(&fs::read(&file)?)?;
        partial["state"] = "partial".into();
        fs::write(file, serde_json::to_vec(&partial)?)?;
        assert!(collect(directory.path(), &names, true).is_err());
        Ok(())
    }

    #[test]
    fn failed_invocation_distinguishes_cancelled_and_not_started_without_accepting_missing_success()
    -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        write(directory.path(), "framework/fmt", 37)?;
        fs::write(directory.path().join("framework/tests.claim"), b"")?;
        let names =
            vec!["framework/fmt".into(), "framework/tests".into(), "extension/tests".into()];
        assert!(collect(directory.path(), &names, true).is_err());
        assert_eq!(failed(directory.path(), &names), Some(37));
        let bytes = collect(directory.path(), &names, false)?;
        let rows = String::from_utf8(bytes).map_err(io::Error::other)?;
        let rows = rows
            .lines()
            .map(serde_json::from_str::<serde_json::Value>)
            .collect::<Result<Vec<_>, _>>()?;
        assert_eq!(rows[0]["exit"], 37);
        assert_eq!(rows[1]["state"], "cancelled");
        assert_eq!(rows[2]["state"], "not-started");
        fs::write(directory.path().join("framework/fmt.duplicate"), b"duplicate")?;
        assert!(collect(directory.path(), &names, false).is_err());
        Ok(())
    }
}
