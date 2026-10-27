//! Invocation receipts around the mise-owned task graph.
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

pub(super) fn listing(command: Command, prefix: &str) -> io::Result<Vec<(String, Vec<String>)>> {
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
        if !["", "framework", "temporal"].contains(&scope) {
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
            "UMPIRE_GATE_STAGE" => {
                if stage.replace(value).is_some() {
                    return Err(io::Error::other("duplicate mise stage metadata"));
                }
            }
            "UMPIRE_GATE_ORDER" => {
                if order.replace(value).is_some() {
                    return Err(io::Error::other("duplicate mise stage order"));
                }
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

pub(super) fn inventory(command: Command, prefix: &str) -> io::Result<Vec<String>> {
    Ok(listing(command, prefix)?.into_iter().map(|(name, _)| name).collect())
}

pub(super) fn directory(parent: &Path) -> io::Result<PathBuf> {
    fs::create_dir_all(parent)?;
    let nonce = SystemTime::now().duration_since(UNIX_EPOCH).map_err(io::Error::other)?.as_nanos();
    let directory = parent.join(format!("gate-{}-{nonce}", std::process::id()));
    fs::create_dir(&directory)?;
    Ok(directory)
}

pub(super) fn scheduled(name: &str, run: impl FnOnce() -> io::Result<i32>) -> io::Result<i32> {
    scheduled_with(name, run, None)
}

pub(super) fn skipped(name: &str, reason: &str) -> io::Result<i32> {
    scheduled_with(name, || Ok(0), Some(reason))
}

fn scheduled_with(
    name: &str,
    run: impl FnOnce() -> io::Result<i32>,
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
    if !["", "framework", "temporal"].contains(&scope.as_str()) {
        return Err(io::Error::other("invalid stage receipt scope"));
    }
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
    let started = Instant::now();
    let result = run();
    let receipt = serde_json::json!({
        "stage": name, "scope": scope,
        "state": if skipped.is_some() { "skipped" } else if result.as_ref().is_ok_and(|code| *code == 0) { "success" } else { "failed" },
        "reason": skipped,
        "exit": result.as_ref().copied().unwrap_or(1),
        "error": result.as_ref().err().map(ToString::to_string),
        "seconds": started.elapsed().as_secs_f64(),
    });
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
        if path.is_dir() {
            let scope = path.file_name().and_then(|n| n.to_str()).unwrap_or_default();
            if !["framework", "temporal"].contains(&scope) {
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
        if ["stages.jsonl", "stdout.log", "stderr.log"].contains(&name) {
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
            if value["stage"] != name.rsplit('/').next().unwrap_or(name)
                || !value["seconds"].as_f64().is_some_and(|n| n.is_finite() && n >= 0.0)
                || value["scope"].as_str().unwrap_or("")
                    != name.split_once('/').map(|(scope, _)| scope).unwrap_or("")
                || !value["exit"].is_i64()
                || !claimed
            {
                return Err(io::Error::other(format!("invalid stage receipt: {name}")));
            }
            if success && value["exit"] != 0 {
                return Err(io::Error::other(format!("failed stage receipt: {name}")));
            }
            value
        } else {
            if success {
                return Err(io::Error::other(format!("missing stage receipt: {name}")));
            }
            serde_json::json!({"stage": name.rsplit('/').next().unwrap_or(name), "scope": name.split_once('/').map(|(scope, _)| scope).unwrap_or(""),
                "state": if claimed { "cancelled" } else { "not-started" },
                "exit": null, "seconds": null})
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
    supervised_inner(command, directory, expected, false)
}

pub(super) fn supervised_clean(command: &Command, directory: &Path) -> io::Result<i32> {
    supervised_inner(command, directory, &[], true)
}

fn supervised_inner(
    command: &Command,
    directory: &Path,
    expected: &[String],
    clean: bool,
) -> io::Result<i32> {
    use umpire_backend_process::{JvmLease, OwnedProcess};
    let lease = JvmLease::inherited_or_acquire()?;
    let stdout = fs::File::create(directory.join("stdout.log"))?;
    let stderr = fs::File::create(directory.join("stderr.log"))?;
    let mut process = if clean {
        OwnedProcess::spawn_leased_clean(command, directory, &stdout, &stderr, &lease)?
    } else {
        OwnedProcess::spawn_leased(command, directory, &stdout, &stderr, &lease)?
    };
    let code = loop {
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
    io::stdout().write_all(&fs::read(directory.join("stdout.log"))?)?;
    io::stderr().write_all(&fs::read(directory.join("stderr.log"))?)?;
    Ok(code)
}

pub(super) fn run(
    mut command: Command,
    task: &str,
    directory: &Path,
    expected: &[String],
) -> io::Result<i32> {
    command
        .args(["run", "--force", "--skip-tools", "--task-cache", "off", "--jobs", "4", task])
        .env("UMPIRE_GATE_RECEIPTS", directory);
    let code = supervised(&command, directory, expected)?;
    match collect(directory, expected, code == 0) {
        Ok(records) => fs::write(directory.join("stages.jsonl"), records)?,
        Err(error) => {
            // Preserve the real failed leaf status while retaining complete invalid-stage evidence.
            let records =
                expected
                    .iter()
                    .map(|name| {
                        let mut value = fs::read(directory.join(format!("{name}.json"))).ok()
                    .and_then(|bytes| serde_json::from_slice::<serde_json::Value>(&bytes).ok())
                    .filter(|value| value["stage"] == name.rsplit('/').next().unwrap_or(name)
                        && value["exit"].is_i64())
                    .unwrap_or_else(|| serde_json::json!({
                        "stage": name.rsplit('/').next().unwrap_or(name),
                        "scope": name.split_once('/').map(|(scope, _)| scope).unwrap_or(""),
                        "state": "partial", "exit": null, "seconds": null,
                    }));
                        value["receipt_error"] = error.to_string().into();
                        format!("{value}\n")
                    })
                    .collect::<String>();
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
mod receipt_controls {
    use super::*;

    fn write(directory: &Path, name: &str, exit: i32) -> io::Result<()> {
        let (scope, stage) = name.split_once('/').unwrap_or(("", name));
        fs::create_dir_all(directory.join(scope))?;
        fs::write(directory.join(format!("{name}.claim")), b"")?;
        fs::write(
            directory.join(format!("{name}.json")),
            serde_json::to_vec(&serde_json::json!({
                "scope": scope, "stage": stage, "state": if exit == 0 { "success" } else { "failed" },
                "exit": exit, "seconds": 0.0,
            }))?,
        )
    }

    #[test]
    fn scoped_receipts_keep_duplicate_public_names_and_logical_order() -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        write(directory.path(), "temporal/fmt", 0)?;
        write(directory.path(), "framework/fmt", 0)?;
        let names = vec!["framework/fmt".into(), "temporal/fmt".into()];
        let bytes = collect(directory.path(), &names, true)?;
        let rows = String::from_utf8(bytes).map_err(io::Error::other)?;
        let rows = rows
            .lines()
            .map(serde_json::from_str::<serde_json::Value>)
            .collect::<Result<Vec<_>, _>>()?;
        assert_eq!(rows[0]["scope"], "framework");
        assert_eq!(rows[1]["scope"], "temporal");
        assert!(rows.iter().all(|row| row["stage"] == "fmt"));
        Ok(())
    }

    #[test]
    fn failed_invocation_distinguishes_cancelled_and_not_started_without_accepting_missing_success()
    -> io::Result<()> {
        let directory = tempfile::tempdir()?;
        write(directory.path(), "framework/fmt", 37)?;
        fs::write(directory.path().join("framework/tests.claim"), b"")?;
        let names = vec!["framework/fmt".into(), "framework/tests".into(), "temporal/tests".into()];
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
