#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn fixed_inputs_preserve_public_limits_and_selected_method() {
        let flags = fixed_explore_flags();
        for flag in [
            "--parallel=1",
            "--choice-bytes=64MiB",
            "--output-limit=256MiB",
            "--success-bytes=4GiB",
            "--terminate-grace=2s",
        ] {
            assert!(flags.contains(&flag.to_owned()));
        }
        assert_eq!(
            test_pattern("TestSuite/.*/TestOne"),
            "^TestSuite$/.* /^TestOne$".replace(".* /", ".*/")
        );
        assert_eq!(test_pattern("TestSuite/TestA+B"), "^TestSuite$/^TestA\\+B$");
        assert_eq!(
            fixed_test_flags(),
            ["-test.v", "-test.count=1", "-test.timeout=72h"]
        );
    }
    #[test]
    fn package_caches_are_shared_between_units_and_module_cwd_is_retained() {
        let dirs = Dirs::new(Path::new("/scratch"), "./tests/nested");
        assert!(dirs.preparation.starts_with(&dirs.root));
        assert!(dirs.engine.starts_with(&dirs.root));
        assert_ne!(dirs.preparation, dirs.runs);
        let package = Package {
            import_path: "fixture".into(),
            pattern: "./tests/nested".into(),
            module_directory: "/checkout/tests/nested".into(),
            target: "./.".into(),
            tests: vec!["TestA".into(), "TestB".into()],
            methods: vec![],
            registrations: vec![],
            units: BTreeMap::from([
                ("TestA".into(), vec!["TestA".into()]),
                ("TestB".into(), vec!["TestB".into()]),
            ]),
            top_level_tests: vec!["TestA".into(), "TestB".into()],
            guest_excluded: BTreeMap::new(),
            list_count: 2,
            parsed_count: 2,
        };
        assert_eq!(package.target, "./.");
        assert_eq!(package.module_directory, "/checkout/tests/nested");
        assert_eq!(
            Dirs::new(Path::new("/scratch"), &package.pattern).engine,
            dirs.engine
        );
    }
    #[test]
    fn retention_budget_deduplicates_reasons_without_overflow() {
        let mut kept = Retention::new(10, &BTreeMap::new());
        assert!(kept.admit(Class::GuestFailure, "panic", 11).is_err());
        kept.used = 9;
        assert!(kept.admit(Class::GuestFailure, "panic", u64::MAX).is_err());
        kept.used = 0;
        kept.reasons.insert((Class::GuestFailure, "panic".into()));
        assert!(kept.admit(Class::GuestFailure, "panic", 1).is_err());
        assert!(kept.admit(Class::Pass, "", 1).is_err());
    }
    #[test]
    fn model_identity_is_package_independent_and_configuration_bound() {
        let first = json!({"working_directory_sha256":"package-a","epoch_nanos":946_684_800_000_000_000_u64,"seed":0});
        let mut second = first.clone();
        second["working_directory_sha256"] = json!("package-b");
        assert_eq!(
            model_digest(&first).unwrap(),
            model_digest(&second).unwrap()
        );
        second["epoch_nanos"] = json!(1);
        assert_ne!(
            model_digest(&first).unwrap(),
            model_digest(&second).unwrap()
        );
    }
    #[test]
    fn package_independent_configuration_only_removes_package_fields() {
        let value = json!({"WorkingDirectory":"/workspace/tests/a","Environment":["PWD=/workspace/tests/a","TMPDIR=/tmp"],"WritableDirectories":["/workspace","/tmp","/workspace/tests/a"],"Limits":{"OutputBytes":256<<20}});
        let mut other = value.clone();
        other["WorkingDirectory"] = json!("/workspace/tests/b");
        other["Environment"][0] = json!("PWD=/workspace/tests/b");
        other["WritableDirectories"][2] = json!("/workspace/tests/b");
        assert_eq!(
            configuration_digest(value.clone()).unwrap(),
            configuration_digest(other.clone()).unwrap()
        );
        other["Limits"]["OutputBytes"] = json!(1);
        assert_ne!(
            configuration_digest(value).unwrap(),
            configuration_digest(other).unwrap()
        );
    }
}

use super::{
    classify::{self, ArtifactFacts, Class, Event, Observation},
    enumerate::{self, Package},
    options::Options,
    process::{Command, Context, Executor},
    record::{self, Candidate, Header, Journal, TestEntry},
};
use crate::{
    evidence::{canonical, store::Opened},
    install::{config, manifest::Installation},
};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    io::Read,
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

pub fn file_sha256(path: &Path) -> Result<String, String> {
    use sha2::{Digest, Sha256};
    let mut file = fs::File::open(path).map_err(|e| e.to_string())?;
    let mut hash = Sha256::new();
    let mut buffer = [0; 16 << 10];
    loop {
        let count = file.read(&mut buffer).map_err(|e| e.to_string())?;
        if count == 0 {
            break;
        }
        hash.update(&buffer[..count]);
    }
    Ok(format!("sha256:{:x}", hash.finalize()))
}
pub fn fixed_explore_flags() -> Vec<String> {
    let mut flags = [
        "--json",
        "--parallel=1",
        "--terminate-grace=2s",
        "--choices",
        "--choice-bytes=64MiB",
        "--output-limit=256MiB",
        "--keep-successes=all",
        "--success-limit=1",
        "--success-bytes=4GiB",
    ]
    .map(str::to_owned)
    .to_vec();
    flags.extend(
        enumerate::guest_tags()
            .into_iter()
            .map(|tag| format!("--build-tag={tag}")),
    );
    flags.extend(
        [
            "TEMPORAL_ROOT=/workspace",
            "TMPDIR=/tmp",
            "TEMPORAL_TEST_LOG_LEVEL=warn",
        ]
        .map(|env| format!("--env={env}")),
    );
    flags
}
pub fn fixed_test_flags() -> Vec<String> {
    ["-test.v", "-test.count=1", "-test.timeout=72h"]
        .map(str::to_owned)
        .to_vec()
}
fn test_pattern(test: &str) -> String {
    test.split('/')
        .map(|part| {
            if part == ".*" {
                ".*".into()
            } else {
                let mut value = "^".to_owned();
                for c in part.chars() {
                    if "\\.+*?()|[]{}^$".contains(c) {
                        value.push('\\');
                    }
                    value.push(c);
                }
                value.push('$');
                value
            }
        })
        .collect::<Vec<_>>()
        .join("/")
}
fn duration(value: Duration) -> String {
    format!("{}ns", value.as_nanos())
}
fn explore_args(
    options: &Options,
    pkg: &Package,
    dirs: &Dirs,
    artifacts: &Path,
    test: &str,
    timeout: Duration,
) -> Vec<String> {
    let mut args = vec![
        "--installation".into(),
        dirs.manifest.display().to_string(),
        "explore".into(),
        format!("--seeds={}", options.seed),
        format!("--execution-timeout={}", duration(timeout)),
        format!(
            "--overall-timeout={}",
            duration(timeout + options.prepare_timeout)
        ),
        "--artifacts".into(),
        artifacts.display().to_string(),
        "--working-dir".into(),
        pkg.module_directory.clone(),
    ];
    args.extend(fixed_explore_flags());
    args.extend([
        "go-test".into(),
        pkg.target.clone(),
        "--".into(),
        format!("-test.run={test}"),
    ]);
    args.extend(fixed_test_flags());
    args
}
struct Dirs {
    root: PathBuf,
    install: PathBuf,
    manifest: PathBuf,
    config: PathBuf,
    preparation: PathBuf,
    engine: PathBuf,
    runs: PathBuf,
}
impl Dirs {
    fn new(work: &Path, pattern: &str) -> Self {
        let digest = canonical::digest(pattern.as_bytes());
        let slug = format!(
            "{}-{}",
            pattern.trim_start_matches("./").replace('/', "-"),
            &digest[..8]
        );
        let root = work.join("packages").join(slug);
        Self {
            install: root.join("install"),
            manifest: root.join("install/gomad-wasm.json"),
            config: root.join("install/guest-config.json"),
            preparation: root.join("preparation"),
            engine: root.join("engine"),
            runs: root.join("runs"),
            root,
        }
    }
}
fn directory(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::DirBuilderExt as _;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(path)
        .map_err(|e| e.to_string())
}
fn reset(path: &Path) -> Result<(), String> {
    match fs::remove_dir_all(path) {
        Ok(()) => {}
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
        Err(e) => return Err(e.to_string()),
    }
    directory(path)
}
fn remove(path: &Path) -> Result<(), String> {
    match fs::remove_dir_all(path) {
        Ok(()) => Ok(()),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(e) => Err(e.to_string()),
    }
}
fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    use std::{io::Write, os::unix::fs::OpenOptionsExt as _};
    let mut file = fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| e.to_string())?;
    file.write_all(bytes).map_err(|e| e.to_string())
}
fn tree_size(root: &Path) -> Result<(u64, usize), String> {
    if !root.exists() {
        return Ok((0, 0));
    }
    let mut pending = vec![root.to_owned()];
    let mut bytes = 0_u64;
    let mut count = 0;
    while let Some(path) = pending.pop() {
        for entry in fs::read_dir(path).map_err(|e| e.to_string())? {
            let entry = entry.map_err(|e| e.to_string())?;
            let meta = entry.metadata().map_err(|e| e.to_string())?;
            if entry.file_type().map_err(|e| e.to_string())?.is_symlink() {
                return Err("sweep scratch contains a symlink".into());
            }
            if meta.is_dir() {
                pending.push(entry.path());
            } else if meta.is_file() {
                bytes = bytes
                    .checked_add(meta.len())
                    .ok_or("scratch size overflow")?;
                count += 1;
            }
        }
    }
    Ok((bytes, count))
}
fn execute(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    path: &Path,
    args: Vec<String>,
    dir: &Path,
    timeout: Duration,
) -> Result<Vec<u8>, String> {
    let completed = executor.exec(
        context,
        &Command {
            path: path.to_owned(),
            args,
            dir: dir.to_owned(),
            env: options.environ.clone(),
            timeout,
        },
    );
    context.check()?;
    if completed.exit != 0 || completed.error.is_some() {
        return Err(format!(
            "{}: exit {} {:?}: {}",
            path.display(),
            completed.exit,
            completed.error,
            String::from_utf8_lossy(&completed.stderr)
        ));
    }
    Ok(completed.stdout)
}
fn source_identity(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
) -> Result<(String, String), String> {
    let git = Path::new("git");
    let mut status = vec![
        "-C".into(),
        options.checkout.display().to_string(),
        "status".into(),
        "--porcelain".into(),
        "--untracked-files=all".into(),
    ];
    if options.out.starts_with(&options.checkout) {
        status.extend(["--".into(), ".".into()]);
        for name in ["sweep.jsonl", "sweep.json", "sweep.json.tmp"] {
            let path = options.out.join(name);
            let rel = path
                .strip_prefix(&options.checkout)
                .map_err(|e| e.to_string())?;
            status.push(format!(":(exclude,literal){}", rel.display()));
        }
    }
    let status = execute(
        executor,
        context,
        options,
        git,
        status,
        &options.work,
        Duration::from_secs(60),
    )?;
    if !status.iter().all(u8::is_ascii_whitespace) {
        return Err("refused: checkout has tracked or untracked changes outside generated sweep outputs; use a clean frozen checkout".into());
    }
    let head = execute(
        executor,
        context,
        options,
        git,
        vec![
            "-C".into(),
            options.checkout.display().to_string(),
            "rev-parse".into(),
            "HEAD".into(),
        ],
        &options.work,
        Duration::from_secs(60),
    )?;
    let inventory = execute(
        executor,
        context,
        options,
        git,
        vec![
            "-C".into(),
            options.checkout.display().to_string(),
            "ls-files".into(),
            "--stage".into(),
        ],
        &options.work,
        Duration::from_secs(60),
    )?;
    Ok((
        String::from_utf8(head)
            .map_err(|e| e.to_string())?
            .trim()
            .to_owned(),
        format!("sha256:{}", canonical::digest(&inventory)),
    ))
}
fn configuration_digest(mut config: Value) -> Result<String, String> {
    config["WorkingDirectory"] = json!("");
    for (key, prefix) in [
        ("Environment", "PWD="),
        ("WritableDirectories", "/workspace/"),
    ] {
        if let Some(values) = config[key].as_array_mut() {
            values.retain(|v| !v.as_str().is_some_and(|v| v.starts_with(prefix)));
        }
    }
    Ok(format!(
        "sha256:{}",
        canonical::digest(&canonical::encode(&config)?)
    ))
}
fn install_package(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    pkg: &Package,
    dirs: &Dirs,
) -> Result<Installation, String> {
    reset(&dirs.root)?;
    for path in [&dirs.install, &dirs.preparation, &dirs.engine, &dirs.runs] {
        directory(path)?;
    }
    let mut args = vec![
        "install".into(),
        "--compiler".into(),
        options.compiler.display().to_string(),
        "--helper".into(),
        options.helper.display().to_string(),
        "--runtime-root".into(),
        options.runtime_root.display().to_string(),
        "--cache".into(),
        dirs.preparation.display().to_string(),
        "--compile-cache".into(),
        dirs.engine.display().to_string(),
        "--output".into(),
        dirs.manifest.display().to_string(),
    ];
    if options.config_source == "temporal-install" {
        args.extend([
            "--temporal-checkout".into(),
            options.checkout.display().to_string(),
            "--temporal-package".into(),
            pkg.pattern.clone(),
        ]);
    } else {
        let mut config = config::build(
            "gomad.wasi-cooperative/v1",
            None,
            Some((&options.checkout, &pkg.pattern)),
        )?
        .original;
        config
            .as_object_mut()
            .ok_or("guest config is not an object")?
            .remove("network_v2");
        config["Args"] = json!(["temporal.test"]);
        write(
            &dirs.config,
            &serde_json::to_vec(&config).map_err(|e| e.to_string())?,
        )?;
        args.extend([
            "--profile".into(),
            "gomad.wasi-cooperative/v1".into(),
            "--config".into(),
            dirs.config.display().to_string(),
            "--memory-bytes".into(),
            (4_u64 << 30).to_string(),
            "--fuel".into(),
            (1_u64 << 50).to_string(),
        ]);
    }
    execute(
        executor,
        context,
        options,
        &options.gomad,
        args,
        &dirs.root,
        Duration::from_secs(600),
    )
    .map_err(|e| format!("refused: gomad install ({}): {e}", options.config_source))?;
    let bytes = crate::install::manifest::read(&dirs.manifest, 512 << 20)?;
    let installed: Installation = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
    if installed.schema != crate::install::manifest::SCHEMA
        || installed.helper_sha256 != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || installed.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
    {
        return Err("refused: installation implementation mismatch".into());
    }
    if installed.go_version != "go1.27.1"
        || installed.compiler_path != options.compiler
        || installed.compiler_sha256
            != file_sha256(&options.compiler)?.trim_start_matches("sha256:")
        || installed.runtime_root != options.runtime_root
        || installed.runtime_implementation_sha256.as_deref()
            != Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"))
    {
        return Err("refused: installation compiler or runtime identity mismatch".into());
    }
    let admission = installed
        .admission
        .as_ref()
        .ok_or("refused: native installation admission missing")?;
    if admission.host_executable_sha256
        != file_sha256(&options.gomad)?.trim_start_matches("sha256:")
        || admission.explicit_helper.as_ref().is_none_or(|h| {
            h.path != options.helper || h.sha256 != admission.host_executable_sha256
        })
    {
        return Err(
            "refused: installation helper identity does not match explicit linked host".into(),
        );
    }
    context.check()?;
    Ok(installed)
}
fn artifact_facts(opened: &Opened) -> Result<ArtifactFacts, String> {
    opened.authenticate_all()?;
    let manifest = opened.manifest();
    let record = &manifest.record;
    if record.identity.implementation != crate::replay::preflight::implementation()?
        || record.identity.platform != crate::replay::preflight::platform()
        || record.profile != "gomad.wasi-cooperative/v1"
    {
        return Err("native artifact implementation, host or profile mismatch".into());
    }
    let stream = |name: &str| -> Result<Vec<u8>, String> {
        if let Some(stream) = record.streams.get(name) {
            opened.payload(name, stream.bytes.min(256 << 20))
        } else {
            Ok(Vec::new())
        }
    };
    Ok(ArtifactFacts {
        kind: if record.outcome.class == "watchdog" {
            "watchdog-timeout"
        } else {
            "execution"
        }
        .into(),
        module_sha256: record.identity.target.clone(),
        domain: record.outcome.class.clone(),
        reason: record.outcome.reason.clone(),
        termination: record.outcome.termination.clone(),
        exit_code: record.outcome.exit_code.map(u64::from),
        stdout: stream("output/stdout")?,
        stderr: stream("output/stderr")?,
        stdout_truncated: false,
        stored_bytes: opened.stored_bytes(),
        read_error: String::new(),
    })
}
#[expect(
    clippy::too_many_lines,
    reason = "The receiver keeps campaign, execution and artifact checks in one admission boundary."
)]
fn authenticated_observation(
    test: &str,
    seed: u64,
    run_root: &Path,
    completed: super::process::Completed,
) -> Result<(Observation, Vec<Event>), String> {
    use crate::campaign::ledger::{Event as CampaignEvent, Execution, Journal as Campaign};
    let mut observation = Observation {
        test: test.into(),
        exit: completed.exit,
        exec_error: completed.error.unwrap_or_default(),
        stdout: completed.stdout,
        stderr: completed.stderr,
        artifacts: BTreeMap::new(),
    };
    let mut events = classify::parse_events(&observation.stdout)?;
    let progress = events
        .iter()
        .find(|event| event.r#type == "soft_assertion")
        .cloned();
    if !observation.exec_error.is_empty() || events.iter().any(|event| event.r#type == "error") {
        return Ok((observation, events));
    }
    let event = events
        .iter_mut()
        .find(|event| event.r#type == "result")
        .ok_or("native result event missing")?;
    let path = Path::new(&event.campaign_path);
    if !path.is_absolute() || !path.starts_with(run_root) {
        return Err("native campaign escaped this execution root".into());
    }
    let campaign = Campaign::open(path, false)?;
    let mut executions: BTreeMap<u64, Execution> = BTreeMap::new();
    let mut complete = None;
    for row in campaign.events() {
        match row {
            CampaignEvent::Receipt { execution } => {
                executions.insert(execution.ordinal, execution.clone());
            }
            CampaignEvent::Outcome {
                executions: rows, ..
            } => {
                for execution in rows {
                    executions.insert(execution.ordinal, execution.clone());
                }
            }
            CampaignEvent::Complete { stop_reason } => complete = Some(stop_reason.as_str()),
            CampaignEvent::Admit { .. } => {}
        }
    }
    if complete.is_none() || executions.len() != 1 || event.attempted != 1 {
        return Err("native campaign is not one complete isolated execution".into());
    }
    let execution = executions
        .get(&0)
        .ok_or("native campaign ordinal zero missing")?;
    if execution.seed != seed
        || execution.record.seed != seed
        || !execution
            .record
            .arguments
            .iter()
            .any(|arg| arg == &format!("-test.run={}", test_pattern(test)))
    {
        return Err("native execution seed or selected test identity mismatch".into());
    }
    validate_soft_progress(progress.as_ref(), execution, path)?;
    let success = execution.record.outcome.class == "success";
    let watchdog = execution.record.outcome.class == "watchdog";
    if event.succeeded != u64::from(success)
        || event.failures != u64::from(!success)
        || event.watchdogs != u64::from(watchdog)
    {
        return Err("native result counters disagree with authenticated execution".into());
    }
    let artifact = execution
        .artifact
        .as_ref()
        .ok_or("native execution artifact reference missing")?;
    let opened = campaign.open_artifact(artifact)?;
    if opened.manifest().record != execution.record
        || opened.stored_bytes() != execution.stored_bytes
    {
        return Err("native artifact does not match authenticated campaign execution".into());
    }
    let full = path.join(artifact).display().to_string();
    let report = observation
        .stdout
        .split(|b| *b == b'\n')
        .find_map(|line| serde_json::from_slice::<classify::NativeResult>(line).ok())
        .ok_or("native result report missing")?;
    validate_report(&report, execution, &full, complete, seed)?;
    let facts = artifact_facts(&opened)?;
    event.classification = if execution.divergence.is_some() {
        "replay_divergence"
    } else if success {
        "success"
    } else if watchdog {
        "watchdog_observation"
    } else {
        "target_failure"
    }
    .into();
    let artifact_event = Event {
        r#type: "artifact".into(),
        classification: if success { "success" } else { "failure" }.into(),
        path: full.clone(),
        ..Event::default()
    };
    observation.artifacts.insert(full, facts);
    events.push(artifact_event);
    Ok((observation, events))
}
fn validate_soft_progress(
    progress: Option<&Event>,
    execution: &crate::campaign::ledger::Execution,
    campaign: &Path,
) -> Result<(), String> {
    let Some(progress) = progress else {
        return Ok(());
    };
    let retained = execution
        .soft_assertions
        .as_ref()
        .ok_or("soft assertion progress has no authenticated report")?;
    let retained: crate::campaign::soft_assertions::Report =
        serde_json::from_value(retained.clone())
            .map_err(|error| format!("invalid authenticated soft assertion report: {error}"))?;
    if Path::new(&progress.campaign_path) != campaign
        || progress.seed != Some(execution.seed)
        || progress.soft_assertions.as_ref() != Some(&retained)
    {
        return Err("soft assertion progress differs from authenticated campaign report".into());
    }
    Ok(())
}

fn validate_report(
    report: &classify::NativeResult,
    execution: &crate::campaign::ledger::Execution,
    full: &str,
    complete: Option<&str>,
    seed: u64,
) -> Result<(), String> {
    let success = execution.record.outcome.class == "success";
    let watchdog = execution.record.outcome.class == "watchdog";
    let expected_class = if success {
        "success"
    } else if watchdog {
        "watchdog_observation"
    } else {
        "target_failure"
    };
    if complete != Some(report.stop_reason.as_str())
        || report.selection_count != 1
        || report.cancelled != 0
        || report.distinct_failures != u64::from(!success)
        || report.retained_evidence != 1
        || report.retained_bytes != execution.stored_bytes
        || report.classification != expected_class
        || if success {
            report.success_artifacts.len() != 1
                || report.success_artifacts[0].seed != seed
                || report.success_artifacts[0].path != full
                || report.success_artifacts[0].record_hash != execution.record.hash()?
        } else {
            !report.success_artifacts.is_empty()
        }
    {
        return Err(
            "native result references or totals disagree with authenticated campaign".into(),
        );
    }
    if let Some(presentation) = &report.artifacts
        && (presentation.len() != 1
            || presentation[0].path != full
            || presentation[0].record_hash != execution.record.hash()?
            || presentation[0].replay_argv
                != ["gomad".to_owned(), "replay".into(), full.to_owned()]
            || presentation[0].replay_command.is_empty()
            || presentation[0].replay_command.len() > 16384)
    {
        return Err(
            "native artifact replay presentation disagrees with authenticated reference".into(),
        );
    }
    Ok(())
}

struct Retention {
    budget: u64,
    used: u64,
    reasons: BTreeSet<(Class, String)>,
}
impl Retention {
    fn new(budget: u64, latest: &BTreeMap<(String, String), TestEntry>) -> Self {
        let mut kept = Self {
            budget,
            used: 0,
            reasons: BTreeSet::new(),
        };
        for entry in latest.values().filter(|entry| entry.artifact_retained) {
            kept.used = kept.used.saturating_add(entry.artifact_bytes);
            kept.reasons.insert((entry.class, entry.reason.clone()));
        }
        kept
    }
    fn admit(&self, class: Class, reason: &str, bytes: u64) -> Result<(), String> {
        if class == Class::Pass || bytes == 0 {
            return Err(String::new());
        }
        if self.reasons.contains(&(class, reason.into())) {
            return Err("exemplar for this reason already retained".into());
        }
        if self
            .used
            .checked_add(bytes)
            .is_none_or(|total| total > self.budget)
        {
            return Err("retention budget exhausted".into());
        }
        Ok(())
    }
    fn keep(
        &mut self,
        entry: &mut TestEntry,
        source: &Path,
        destination: &Path,
        artifact: &str,
    ) -> Result<(), String> {
        if let Err(note) = self.admit(entry.class, &entry.reason, entry.artifact_bytes) {
            entry.retention_note = note;
            return remove(source);
        }
        directory(
            destination
                .parent()
                .ok_or("retention destination has no parent")?,
        )?;
        if destination.exists() {
            return Err("retention destination already exists".into());
        }
        match fs::rename(source, destination) {
            Ok(()) => {
                self.used += entry.artifact_bytes;
                self.reasons.insert((entry.class, entry.reason.clone()));
                entry.artifact_retained = true;
                if !artifact.is_empty() {
                    entry.artifact_path = destination
                        .join(
                            Path::new(artifact)
                                .strip_prefix(source)
                                .map_err(|_| "artifact escaped execution root")?,
                        )
                        .display()
                        .to_string();
                }
                Ok(())
            }
            Err(error) => {
                entry.retention_note = format!("retain failed: {error}");
                remove(source)
            }
        }
    }
}
fn measured_entry(
    entry: &mut TestEntry,
    dirs: &Dirs,
    root: &Path,
    before: usize,
) -> Result<(), String> {
    entry.preparation_bytes = tree_size(&dirs.preparation)?.0;
    entry.artifact_bytes = tree_size(root)?.0;
    let (bytes, after) = tree_size(&dirs.engine)?;
    entry.engine_cache_bytes = bytes;
    entry.engine_entries_added = after.saturating_sub(before);
    entry.engine_cache_hit = before > 0 && before == after;
    Ok(())
}
fn warm(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    pkg: &Package,
    dirs: &Dirs,
) -> Result<TestEntry, String> {
    let root = dirs.runs.join("warm-up");
    reset(&root)?;
    let before = tree_size(&dirs.engine)?.1;
    let output = executor.exec(
        context,
        &Command {
            path: options.gomad.clone(),
            args: explore_args(
                options,
                pkg,
                dirs,
                &root.join("artifacts"),
                "^$",
                options.prepare_timeout,
            ),
            dir: root.clone(),
            env: options.environ.clone(),
            timeout: 2 * options.prepare_timeout + Duration::from_secs(300),
        },
    );
    context.check().inspect_err(|_| {
        let _ = remove(&root);
    })?;
    let mut entry = TestEntry::new(
        &pkg.pattern,
        "(warm-up)",
        Class::NotRun,
        "warm-up only; not a test row",
    );
    entry.config_source.clone_from(&options.config_source);
    entry.attempts = 1;
    entry.gomad_exit = output.exit;
    entry.wall_seconds = output.elapsed.as_secs_f64();
    if let Some(error) = output.error {
        entry.reason.push_str(": ");
        entry.reason.push_str(&classify::normalize(&error));
    }
    measured_entry(&mut entry, dirs, &root, before)?;
    remove(&root)?;
    Ok(entry)
}
fn run_test(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    pkg: &Package,
    name: &str,
    dirs: &Dirs,
    retention: &mut Retention,
) -> Result<TestEntry, String> {
    let mut entry = TestEntry::new(&pkg.pattern, name, Class::NotRun, "");
    entry.config_source.clone_from(&options.config_source);
    let mut root = PathBuf::new();
    let mut artifact = String::new();
    for attempt in 1..=2 {
        root = dirs
            .runs
            .join(format!("{}-{attempt}", canonical::digest(name.as_bytes())));
        reset(&root)?;
        let before = tree_size(&dirs.engine)?.1;
        let output = executor.exec(
            context,
            &Command {
                path: options.gomad.clone(),
                args: explore_args(
                    options,
                    pkg,
                    dirs,
                    &root.join("artifacts"),
                    &test_pattern(name),
                    options.per_test_timeout,
                ),
                dir: root.clone(),
                env: options.environ.clone(),
                timeout: options.per_test_timeout
                    + options.prepare_timeout
                    + Duration::from_secs(300),
            },
        );
        entry.attempts = attempt;
        entry.gomad_exit = output.exit;
        entry.wall_seconds = output.elapsed.as_secs_f64();
        if let Err(error) = context.check() {
            remove(&root)?;
            return Err(error);
        }
        let outcome = match authenticated_observation(name, options.seed, &root, output) {
            Ok((observation, events)) => classify::classify_native(&observation, &events),
            Err(error) => classify::Outcome {
                class: Class::Infrastructure,
                reason: format!("native evidence: {}", classify::normalize(&error)),
                ..classify::Outcome::default()
            },
        };
        entry.class = outcome.class;
        entry.reason = outcome.reason;
        entry.skipped = outcome.skipped;
        entry.subtests = outcome.subtests;
        entry.module_sha256 = outcome.module_sha256;
        artifact = outcome.artifact_path;
        measured_entry(&mut entry, dirs, &root, before)?;
        if entry.class != Class::Infrastructure
            || attempt == 2
            || !context.fits(options.per_test_timeout + options.prepare_timeout)
        {
            break;
        }
        entry.first_attempt_reason.clone_from(&entry.reason);
        remove(&root)?;
    }
    let destination = options
        .artifacts
        .join(
            dirs.root
                .file_name()
                .ok_or("package scratch name missing")?,
        )
        .join(canonical::digest(name.as_bytes()));
    retention.keep(&mut entry, &root, &destination, &artifact)?;
    Ok(entry)
}

fn admit_output(options: &Options) -> Result<(), String> {
    // A fresh native root must never modify the frozen Go-lineage journal or sidecar.
    for name in ["sweep.jsonl", "sweep.json"] {
        let path = options.out.join(name);
        if path.exists() {
            let bytes = crate::install::manifest::read(&path, 256 << 20)?;
            let first = bytes
                .split(|b| *b == b'\n')
                .find(|line| !line.is_empty())
                .ok_or("refused: existing sweep file is empty")?;
            let value: Value = serde_json::from_slice(first)
                .or_else(|_| serde_json::from_slice(&bytes))
                .map_err(|e| format!("refused: existing sweep file: {e}"))?;
            if value["schema"] != record::SCHEMA {
                return Err(
                    "refused: old Go or unsupported sweep lineage; use a fresh native output root"
                        .into(),
                );
            }
        }
    }
    for dir in [&options.out, &options.work, &options.artifacts] {
        directory(dir)?;
    }
    Ok(())
}

fn base_candidate(
    options: &Options,
    git_sha: &str,
    source_sha256: &str,
) -> Result<Candidate, String> {
    let candidate = Candidate::from([
        ("git_sha".into(), json!(git_sha)),
        ("source_sha256".into(), json!(source_sha256)),
        ("gomad_sha256".into(), json!(file_sha256(&options.gomad)?)),
        ("helper_sha256".into(), json!(file_sha256(&options.helper)?)),
        (
            "compiler_sha256".into(),
            json!(file_sha256(&options.compiler)?),
        ),
        ("config_source".into(), json!(options.config_source)),
        ("seed".into(), json!(options.seed)),
        (
            "per_test_timeout_nanos".into(),
            json!(
                u64::try_from(options.per_test_timeout.as_nanos())
                    .map_err(|_| "per-test duration overflow")?
            ),
        ),
        (
            "prepare_timeout_nanos".into(),
            json!(
                u64::try_from(options.prepare_timeout.as_nanos())
                    .map_err(|_| "preparation duration overflow")?
            ),
        ),
        ("build_tags".into(), json!(enumerate::guest_tags())),
        ("explore_flags".into(), json!(fixed_explore_flags())),
        ("test_flags".into(), json!(fixed_test_flags())),
    ]);
    Ok(candidate)
}

fn model_digest(model: &Value) -> Result<String, String> {
    let mut model = model.clone();
    model
        .as_object_mut()
        .ok_or("installed model configuration is not an object")?
        .remove("working_directory_sha256");
    let identity =
        json!({"implementation":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,"model":model});
    Ok(format!(
        "sha256:{}",
        canonical::digest(&canonical::encode(&identity)?)
    ))
}

fn installed_identity(installed: &Installation) -> Result<Candidate, String> {
    let admission = installed
        .admission
        .as_ref()
        .ok_or("native admission missing")?;
    let identity = Candidate::from([
        ("go_version".into(), json!(installed.go_version)),
        (
            "runtime_implementation_sha256".into(),
            json!(
                installed
                    .runtime_implementation_sha256
                    .as_ref()
                    .map(|value| format!("sha256:{value}"))
            ),
        ),
        (
            "model_sha256".into(),
            json!(model_digest(&installed.model)?),
        ),
        (
            "guest_config_sha256".into(),
            json!(configuration_digest(admission.guest_configuration.clone())?),
        ),
        (
            "engine".into(),
            json!(format!(
                "{}/{}/{}",
                gomad_wasmhost::linked::Conditions::compiled().configuration(),
                std::env::consts::OS,
                std::env::consts::ARCH
            )),
        ),
        ("profile".into(), json!(installed.profile)),
        ("memory_bytes".into(), json!(installed.memory_bytes)),
        ("fuel".into(), json!(installed.fuel)),
    ]);
    Ok(identity)
}

fn admit_candidate(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
) -> Result<(Vec<Package>, Candidate), String> {
    let packages = enumerate::enumerate(executor, context, options)?;
    let (git_sha, source_sha256) = source_identity(executor, context, options)?;
    let mut candidate = base_candidate(options, &git_sha, &source_sha256)?;
    if candidate["gomad_sha256"] != candidate["helper_sha256"] {
        return Err(
            "refused: pinned helper identity does not match the linked Rust implementation".into(),
        );
    }
    let mut installation_identity = None;
    for pkg in &packages {
        let installed = install_package(
            executor,
            context,
            options,
            pkg,
            &Dirs::new(&options.work, &pkg.pattern),
        )?;
        let identity = installed_identity(&installed)?;
        if installation_identity
            .as_ref()
            .is_some_and(|previous| previous != &identity)
        {
            return Err(
                "refused: package installations have different candidate identities".into(),
            );
        }
        installation_identity = Some(identity);
    }
    candidate.extend(installation_identity.ok_or("no package installations")?);
    let after = source_identity(executor, context, options)?;
    if candidate["git_sha"] != json!(after.0) || candidate["source_sha256"] != json!(after.1) {
        return Err("refused: source changed during admission".into());
    }
    context.check()?;
    Ok((packages, candidate))
}

pub fn run_with(
    executor: &mut impl Executor,
    context: &Context,
    options: &Options,
    harness_sha256: &str,
) -> Result<Vec<String>, String> {
    options.validate()?;
    options.require_files()?;
    admit_output(options)?;
    context.check()?;
    let (packages, candidate) = admit_candidate(executor, context, options)?;
    let header = Header {
        statement: record::OBSERVATION_STATEMENT.into(),
        candidate,
        host: json!({"os":std::env::consts::OS,"arch":std::env::consts::ARCH}),
        packages: packages.clone(),
        options: json!({"max_tests":options.max_tests,"retain_bytes":options.retain_bytes,"overall_budget_nanos":u64::try_from(options.overall_budget.as_nanos()).map_err(|_|"overall duration overflow")?,"config_source":options.config_source}),
        harness_sha256: harness_sha256.into(),
    };
    let mut journal = Journal::open(options.out.join("sweep.jsonl"), &header)?;
    let latest = journal.latest();
    let mut retention = Retention::new(options.retain_bytes, &latest);
    for pkg in &packages {
        let dirs = Dirs::new(&options.work, &pkg.pattern);
        let mut warmed = false;
        for name in pkg.tests.iter().take(if options.max_tests == 0 {
            usize::MAX
        } else {
            options.max_tests
        }) {
            if context.interrupted() {
                return Err("sweep cancelled".into());
            }
            if latest
                .get(&(pkg.pattern.clone(), name.clone()))
                .is_some_and(|entry| entry.class != Class::NotRun)
            {
                continue;
            }
            let mut entry = if let Some(file) = pkg.guest_excluded.get(name) {
                TestEntry::new(
                    &pkg.pattern,
                    name,
                    Class::BuildFailure,
                    format!(
                        "guest-file-set: {file} is excluded from the wasip1 guest build under the slim tags"
                    ),
                )
            } else if !context.fits(options.per_test_timeout + options.prepare_timeout) {
                TestEntry::new(
                    &pkg.pattern,
                    name,
                    Class::NotRun,
                    "overall budget exhausted before start",
                )
            } else {
                if !warmed && context.fits(options.per_test_timeout + 3 * options.prepare_timeout) {
                    match warm(executor, context, options, pkg, &dirs) {
                        Ok(entry) => journal.append_warmup(entry)?,
                        Err(_) if !context.interrupted() && context.check().is_err() => {
                            let mut entry = TestEntry::new(
                                &pkg.pattern,
                                name,
                                Class::NotRun,
                                "overall budget exhausted during warm-up",
                            );
                            entry.config_source.clone_from(&options.config_source);
                            journal.append_test(entry)?;
                            warmed = true;
                            continue;
                        }
                        Err(error) => return Err(error),
                    }
                }
                warmed = true;
                if context.fits(options.per_test_timeout + options.prepare_timeout) {
                    match run_test(executor, context, options, pkg, name, &dirs, &mut retention) {
                        Ok(entry) => entry,
                        Err(_) if !context.interrupted() && context.check().is_err() => {
                            TestEntry::new(
                                &pkg.pattern,
                                name,
                                Class::NotRun,
                                "overall budget exhausted during test",
                            )
                        }
                        Err(error) => return Err(error),
                    }
                } else {
                    TestEntry::new(
                        &pkg.pattern,
                        name,
                        Class::NotRun,
                        "overall budget exhausted after warm-up",
                    )
                }
            };
            entry.config_source.clone_from(&options.config_source);
            journal.append_test(entry)?;
        }
        remove(&dirs.root)?;
    }
    let sidecar = record::build_sidecar(journal.rows())?;
    record::write_sidecar(options.out.join("sweep.json"), &sidecar)?;
    Ok(record::verify(&sidecar))
}
pub fn run(options: &Options, harness_sha256: &str) -> Result<Vec<String>, String> {
    let context = Context::new(Instant::now() + options.overall_budget);
    let _guard = crate::backstop::arm_until(context.deadline);
    run_with(&mut super::process::Host, &context, options, harness_sha256)
}

#[cfg(test)]
#[path = "lifecycle_tests.rs"]
mod lifecycle_tests;

#[cfg(test)]
#[path = "source_identity_tests.rs"]
mod source_identity_tests;
