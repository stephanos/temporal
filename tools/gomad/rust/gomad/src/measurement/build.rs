use super::inputs::{self, CheckoutProof};
use crate::evidence::canonical;
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs::{self, OpenOptions},
    io::{Read, Write},
    os::unix::{
        fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
        process::CommandExt,
    },
    path::{Path, PathBuf},
    process::{Command, Stdio},
    sync::atomic::{AtomicU64, Ordering},
    time::{Duration, Instant},
};
const LIMIT: u64 = 64 << 20;
const ARTIFACT_LIMIT: u64 = 512 << 20;
const BUILD_SCHEMA: &str = "gomad.native-measurement-build/v1";
const ADMISSION_SCHEMA: &str = "gomad.native-measurement-admission/v1";

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "kind", content = "features", rename_all = "kebab-case")]
pub enum Role {
    Release,
    Warm,
    Stress,
    Diagnostic(Vec<String>),
}
impl Role {
    /// # Errors
    /// Refuses unsupported role features.
    pub fn features(&self) -> Result<Vec<String>, String> {
        let mut features = match self {
            Self::Release => vec![],
            Self::Stress => vec!["stress-jitter".into()],
            Self::Warm => vec!["diagnostic-profile".into()],
            Self::Diagnostic(features) => features.clone(),
        };
        features.sort();
        let admitted: Vec<Vec<String>> = [
            vec!["diagnostic-profile"],
            vec!["diagnostic-profile", "guest-profile"],
            vec!["diagnostic-fuel-off", "diagnostic-profile"],
        ]
        .into_iter()
        .map(|row| row.into_iter().map(str::to_owned).collect())
        .collect();
        if matches!(self, Self::Diagnostic(_)) && !admitted.contains(&features) {
            return Err(format!(
                "diagnostic helper features {features:?} are not an admitted build {admitted:?}"
            ));
        }
        Ok(features)
    }
    fn identity_role(&self) -> Result<&'static str, String> {
        let features = self.features()?;
        if matches!(self, Self::Stress) {
            return Ok("stress-jitter");
        }
        Ok(
            if features.iter().any(|feature| feature == "guest-profile") {
                "guest-profile"
            } else if features
                .iter()
                .any(|feature| feature == "diagnostic-fuel-off")
            {
                "diagnostic-fuel-off"
            } else if features.is_empty() {
                "ordinary"
            } else {
                "diagnostic-profile"
            },
        )
    }
}
fn cargo_features(role: &Role) -> Result<Vec<String>, String> {
    Ok(role
        .features()?
        .into_iter()
        .map(|feature| format!("helper-{feature}"))
        .collect())
}
/// # Errors
/// Refuses unsupported role features before constructing the pinned build command.
pub fn cargo_command(cargo: &Path, crate_root: &Path, role: &Role) -> Result<Vec<String>, String> {
    let mut command = vec![
        cargo.to_string_lossy().into_owned(),
        "build".into(),
        "--manifest-path".into(),
        crate_root.join("Cargo.toml").to_string_lossy().into_owned(),
        "--release".into(),
        "--locked".into(),
        "--jobs=1".into(),
        "--no-default-features".into(),
        "--message-format=json".into(),
        "--bin".into(),
        "gomad-measure".into(),
    ];
    let features = cargo_features(role)?;
    if !features.is_empty() {
        command.extend(["--features".into(), features.join(",")]);
    }
    Ok(command)
}
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeIdentity {
    pub schema: String,
    pub role: String,
    pub features: Vec<String>,
    pub host_implementation_sha256: String,
    pub linked_implementation_sha256: String,
    pub runtime_implementation_sha256: String,
    pub configuration: String,
    pub source_inputs: BTreeMap<String, String>,
    pub executable_sha256: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BuildReceipt {
    pub schema: String,
    pub crate_root: PathBuf,
    pub executable: PathBuf,
    pub executable_sha256: String,
    pub rustc: PathBuf,
    pub cargo: PathBuf,
    pub inputs: BTreeMap<PathBuf, String>,
    pub command: Vec<String>,
    #[serde(deserialize_with = "unique_environment")]
    pub environment: BTreeMap<String, String>,
    pub role: Role,
    pub rustc_version: String,
    pub cargo_version: String,
    pub identity: NativeIdentity,
}
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Admission {
    pub schema: String,
    pub build_receipt: PathBuf,
    pub build_receipt_sha256: String,
    pub cargo_artifact: PathBuf,
    pub cargo_artifact_sha256: String,
    pub executable: PathBuf,
    pub executable_sha256: String,
    pub identity: NativeIdentity,
}
fn unique_environment<'de, D: serde::Deserializer<'de>>(
    deserializer: D,
) -> Result<BTreeMap<String, String>, D::Error> {
    struct Unique;
    impl<'de> serde::de::Visitor<'de> for Unique {
        type Value = BTreeMap<String, String>;
        fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            formatter.write_str("environment with unique variable names")
        }
        fn visit_map<A: serde::de::MapAccess<'de>>(
            self,
            mut access: A,
        ) -> Result<Self::Value, A::Error> {
            let mut result = BTreeMap::new();
            while let Some((key, value)) = access.next_entry::<String, String>()? {
                if result.insert(key, value).is_some() {
                    return Err(serde::de::Error::custom(
                        "duplicate compiler environment variable",
                    ));
                }
            }
            Ok(result)
        }
    }
    deserializer.deserialize_map(Unique)
}
#[must_use]
pub fn compiler_environment(
    rustc: &Path,
    inputs: &BTreeMap<PathBuf, String>,
    entries: &[String],
) -> bool {
    if !rustc.is_absolute() || inputs.get(rustc).is_none_or(|digest| digest.len() != 64) {
        return false;
    }
    for (name, want) in [
        ("RUSTC", rustc.to_string_lossy().into_owned()),
        ("RUSTC_WRAPPER", String::new()),
        ("RUSTC_WORKSPACE_WRAPPER", String::new()),
    ] {
        let values: Vec<_> = entries
            .iter()
            .filter_map(|entry| entry.split_once('='))
            .filter(|(key, _)| *key == name)
            .map(|(_, value)| value)
            .collect();
        if values != vec![want.as_str()] {
            return false;
        }
    }
    true
}
#[must_use]
pub fn exact_cargo_command(receipt: &BuildReceipt, role: &Role) -> bool {
    receipt.cargo.is_absolute()
        && receipt.inputs.contains_key(&receipt.cargo)
        && cargo_command(&receipt.cargo, &receipt.crate_root, role)
            .is_ok_and(|command| command == receipt.command)
}

fn error(error: impl std::fmt::Display) -> String {
    error.to_string()
}
fn bounded(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let info = fs::symlink_metadata(path).map_err(error)?;
    if !info.is_file() || info.len() > limit {
        return Err(format!("bounded regular file required: {}", path.display()));
    }
    let mut bytes = Vec::new();
    fs::File::open(path)
        .map_err(error)?
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(error)?;
    if bytes.len() as u64 > limit {
        return Err("file exceeds bound".into());
    }
    Ok(bytes)
}
fn hash(path: &Path) -> Result<String, String> {
    Ok(canonical::digest(&bounded(path, ARTIFACT_LIMIT)?))
}
fn write_json(path: &Path, value: &impl Serialize) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(error)?;
    file.write_all(&serde_json::to_vec(value).map_err(error)?)
        .map_err(error)?;
    file.sync_all().map_err(error)
}
fn directory(root: &Path, prefix: &str) -> Result<PathBuf, String> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    if !root.is_absolute() {
        return Err("helper build/stage root must be absolute".into());
    }
    fs::create_dir_all(root).map_err(error)?;
    for _ in 0..100 {
        let path = root.join(format!(
            "{prefix}-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        match fs::DirBuilder::new().mode(0o700).create(&path) {
            Ok(()) => return Ok(path),
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(e) => return Err(error(e)),
        }
    }
    Err("cannot allocate private build/stage directory".into())
}
fn inherited(environ: &[String]) -> BTreeMap<String, String> {
    environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(name, _)| {
            matches!(
                *name,
                "PATH" | "HOME" | "CARGO_HOME" | "RUSTUP_HOME" | "TMPDIR"
            )
        })
        .map(|(key, value)| (key.into(), value.into()))
        .collect()
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CommandCapture {
    pub path: PathBuf,
    pub bytes: Option<u64>,
    pub captured_bytes: Option<u64>,
    pub sha256: Option<String>,
    pub truncated: bool,
    pub error: Option<String>,
}
#[derive(Debug, Serialize, Deserialize)]
pub struct CommandObservation {
    pub schema: String,
    pub command: Vec<String>,
    pub directory: PathBuf,
    pub environment: BTreeMap<String, String>,
    pub pid: Option<u32>,
    #[serde(default)]
    pub parent_pid: Option<u32>,
    #[serde(default)]
    pub clock: Option<String>,
    #[serde(default)]
    pub started_monotonic_nanos: Option<u64>,
    #[serde(default)]
    pub reaped_monotonic_nanos: Option<u64>,
    pub exit_code: Option<i32>,
    pub signal: Option<i32>,
    pub duration_nanos: u64,
    pub termination: String,
    pub stdout: CommandCapture,
    pub stderr: CommandCapture,
    pub stdout_sha256: Option<String>,
    pub stderr_sha256: Option<String>,
    pub cleanup_errors: Vec<String>,
    pub error: Option<String>,
    pub resource_usage: Option<serde_json::Value>,
}
fn capture_file(path: &Path, limit: u64) -> CommandCapture {
    let result = (|| -> Result<(u64, Vec<u8>), String> {
        let info = fs::symlink_metadata(path).map_err(error)?;
        if !info.is_file() {
            return Err("command capture is not a regular file".into());
        }
        let mut bytes = Vec::new();
        fs::File::open(path)
            .map_err(error)?
            .take(limit)
            .read_to_end(&mut bytes)
            .map_err(error)?;
        if info.len() > limit {
            OpenOptions::new()
                .write(true)
                .open(path)
                .map_err(error)?
                .set_len(limit)
                .map_err(error)?;
        }
        Ok((info.len(), bytes))
    })();
    match result {
        Ok((total, bytes)) => CommandCapture {
            path: path.into(),
            bytes: Some(total),
            captured_bytes: Some(bytes.len() as u64),
            sha256: Some(canonical::digest(&bytes)),
            truncated: total > limit,
            error: None,
        },
        Err(error) => CommandCapture {
            path: path.into(),
            bytes: None,
            captured_bytes: None,
            sha256: None,
            truncated: false,
            error: Some(error),
        },
    }
}
fn command(
    root: &Path,
    label: &str,
    cwd: &Path,
    environment: &BTreeMap<String, String>,
    argv: &[String],
) -> Result<Vec<u8>, String> {
    command_bounded(
        root,
        label,
        cwd,
        environment,
        argv,
        Duration::from_secs(7200),
        LIMIT,
    )
}
pub(super) fn command_bounded(
    root: &Path,
    label: &str,
    cwd: &Path,
    environment: &BTreeMap<String, String>,
    argv: &[String],
    timeout: Duration,
    limit: u64,
) -> Result<Vec<u8>, String> {
    let observation = observe_command(root, label, cwd, environment, argv, timeout, limit);
    if let Some(error) = observation.error {
        return Err(error);
    }
    bounded(&observation.stdout.path, limit)
}
#[expect(
    clippy::too_many_lines,
    reason = "Keep the bounded command or admission transaction and its refusal order together."
)]
pub(super) fn observe_command(
    root: &Path,
    label: &str,
    cwd: &Path,
    environment: &BTreeMap<String, String>,
    argv: &[String],
    timeout: Duration,
    limit: u64,
) -> CommandObservation {
    use std::os::unix::process::ExitStatusExt;
    let started = Instant::now();
    let started_monotonic_nanos = gomad_wasmhost::linked::monotonic_nanos().ok();
    let parent_pid = Some(std::process::id());
    let stdout = root.join(format!("{label}.stdout"));
    let stderr = root.join(format!("{label}.stderr"));
    let mut child = None;
    let mut pid = None;
    let mut status = None;
    let mut reaped_monotonic_nanos = None;
    let mut termination = "infrastructure_failure".to_string();
    let attempted = (|| -> Result<(), String> {
        if super::cancellation::cancelled() {
            termination = "cancelled".into();
            return super::cancellation::check();
        }
        let open = |path: &Path| {
            OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .open(path)
                .map_err(error)
        };
        let executable = argv.first().ok_or("empty native process command")?;
        child = Some(
            Command::new(executable)
                .args(&argv[1..])
                .current_dir(cwd)
                .env_clear()
                .envs(environment)
                .stdin(Stdio::null())
                .stdout(Stdio::from(open(&stdout)?))
                .stderr(Stdio::from(open(&stderr)?))
                .process_group(0)
                .spawn()
                .map_err(error)?,
        );
        pid = child.as_ref().map(std::process::Child::id);
        let deadline = started
            .checked_add(timeout)
            .ok_or("native process deadline exceeds clock range")?;
        loop {
            if super::cancellation::cancelled() {
                termination = "cancelled".into();
                return super::cancellation::check();
            }
            if let Some(exit) = child
                .as_mut()
                .ok_or("native process owner absent")?
                .try_wait()
                .map_err(error)?
            {
                reaped_monotonic_nanos = gomad_wasmhost::linked::monotonic_nanos().ok();
                status = Some(exit);
                termination = if exit.success() {
                    "exit"
                } else if exit.signal().is_some() {
                    "signal"
                } else {
                    "exit_failure"
                }
                .into();
                break;
            }
            if fs::metadata(&stdout).map_err(error)?.len() > limit
                || fs::metadata(&stderr).map_err(error)?.len() > limit
            {
                termination = "output_capacity".into();
                return Err("capacity: native process output exceeds bound".into());
            }
            if Instant::now() >= deadline {
                termination = "timeout".into();
                return Err("native process timeout".into());
            }
            std::thread::sleep(
                Duration::from_millis(10).min(deadline.saturating_duration_since(Instant::now())),
            );
        }
        Ok(())
    })();
    let mut cleanup_errors = Vec::new();
    if let Some(mut process) = child {
        let group = i32::try_from(process.id())
            .ok()
            .and_then(rustix::process::Pid::from_raw);
        if let Some(group) = group {
            match rustix::process::kill_process_group(group, rustix::process::Signal::KILL) {
                Ok(()) | Err(rustix::io::Errno::SRCH) => {}
                Err(error) => cleanup_errors.push(format!("kill owned process group: {error}")),
            }
        } else {
            cleanup_errors.push("native process group identity exceeds signed PID range".into());
        }
        if status.is_none()
            && let Err(error) = process.kill()
            && error.kind() != std::io::ErrorKind::InvalidInput
        {
            cleanup_errors.push(format!("kill native child: {error}"));
        }
        match process.wait() {
            Ok(exit) => {
                reaped_monotonic_nanos = gomad_wasmhost::linked::monotonic_nanos().ok();
                status = Some(exit);
            }
            Err(error) => cleanup_errors.push(format!("wait native child: {error}")),
        }
        if let Some(group) = group {
            let deadline = Instant::now() + Duration::from_secs(1);
            loop {
                match rustix::process::test_kill_process_group(group) {
                    Err(rustix::io::Errno::SRCH) => break,
                    Err(error) => {
                        cleanup_errors.push(format!("inspect owned process group: {error}"));
                        break;
                    }
                    Ok(()) if Instant::now() >= deadline => {
                        cleanup_errors
                            .push("owned process group remains after kill and wait".into());
                        break;
                    }
                    Ok(()) => std::thread::sleep(Duration::from_millis(10)),
                }
            }
        }
    }
    let out = capture_file(&stdout, limit);
    let err = capture_file(&stderr, limit);
    let mut failures = Vec::new();
    if let Err(error) = attempted {
        failures.push(error);
    }
    if out.truncated || err.truncated {
        termination = "output_capacity".into();
        failures.push("capacity: native process output exceeds bound".into());
    }
    for capture in [&out, &err] {
        if let Some(error) = &capture.error {
            failures.push(format!("capture {}: {error}", capture.path.display()));
        }
    }
    if let Some(exit) = status
        && !exit.success()
    {
        failures.push(format!(
            "native process exit={:?} signal={:?}",
            exit.code(),
            exit.signal()
        ));
    }
    failures.extend(cleanup_errors.iter().cloned());
    let mut observation = CommandObservation {
        schema: "gomad.native-process-observation/v1".into(),
        command: argv.to_vec(),
        directory: cwd.into(),
        environment: environment.clone(),
        pid,
        parent_pid,
        clock: Some("CLOCK_MONOTONIC".into()),
        started_monotonic_nanos,
        reaped_monotonic_nanos,
        exit_code: status.and_then(|status| status.code()),
        signal: status.and_then(|status| status.signal()),
        duration_nanos: u64::try_from(started.elapsed().as_nanos()).unwrap_or(u64::MAX),
        termination,
        stdout_sha256: out.sha256.clone(),
        stderr_sha256: err.sha256.clone(),
        stdout: out,
        stderr: err,
        cleanup_errors,
        error: (!failures.is_empty()).then(|| failures.join("; ")),
        resource_usage: None,
    };
    if let Err(error) = write_json(&root.join(format!("{label}.json")), &observation) {
        failures.push(format!("retain native process observation: {error}"));
        observation.error = Some(failures.join("; "));
    }
    observation
}

#[derive(Debug)]
pub struct NativeSourceProof {
    repo: PathBuf,
    host_manifest: String,
    helper_manifest: String,
    current: bool,
}
impl NativeSourceProof {
    fn current(proof: &CheckoutProof) -> Self {
        Self {
            repo: proof.repo().into(),
            host_manifest: inputs::SOURCE_INPUTS.into(),
            helper_manifest: gomad_wasmhost::linked::SOURCE_INPUTS.into(),
            current: true,
        }
    }
    #[must_use]
    pub fn repo(&self) -> &Path {
        &self.repo
    }
    /// # Errors
    /// Refuses changed admitted source bindings.
    pub fn verify(&self) -> Result<(), String> {
        source_semantic_inputs(self, &BTreeMap::new()).map(|_| ())
    }
}
#[derive(Debug)]
pub struct HistoricalAdmission {
    pub admission: Admission,
    pub source: NativeSourceProof,
    receipt: PathBuf,
    role: Role,
}
impl HistoricalAdmission {
    /// # Errors
    /// Refuses a historical admission whose source, receipt or executable changed.
    pub fn revalidate(&self) -> Result<Admission, String> {
        validate_admission_source(&self.source, &self.receipt, &self.role)
    }
}

fn semantic_inputs(
    proof: &CheckoutProof,
    environment: &BTreeMap<String, String>,
) -> Result<BTreeMap<PathBuf, String>, String> {
    source_semantic_inputs(&NativeSourceProof::current(proof), environment)
}
fn source_semantic_inputs(
    source: &NativeSourceProof,
    environment: &BTreeMap<String, String>,
) -> Result<BTreeMap<PathBuf, String>, String> {
    let repo = source.repo();
    let mut result = BTreeMap::new();
    for (base, manifest) in [
        (
            repo.join("tools/gomad/rust/gomad"),
            source.host_manifest.as_str(),
        ),
        (
            repo.join("tools/gomad/rust/wasmhost"),
            source.helper_manifest.as_str(),
        ),
    ] {
        inputs::authenticate_source_closure(&base, manifest)?;
        for line in manifest.lines() {
            let (digest, name) = line.split_once('\t').ok_or("malformed source manifest")?;
            let path = base.join(name);
            if !source.current
                && (!Path::new(name).is_relative()
                    || !fs::canonicalize(&path)
                        .map_err(error)?
                        .starts_with(fs::canonicalize(repo).map_err(error)?))
            {
                return Err("historical native source manifest escapes explicit checkout".into());
            }
            result.insert(path, digest.into());
        }
    }
    for ancestor in repo.join("tools/gomad/rust/gomad").ancestors() {
        for suffix in [".cargo/config", ".cargo/config.toml"] {
            let path = ancestor.join(suffix);
            if path.try_exists().map_err(error)? {
                result.insert(path.clone(), hash(&path)?);
            }
        }
    }
    let home = environment
        .get("CARGO_HOME")
        .map(PathBuf::from)
        .or_else(|| {
            environment
                .get("HOME")
                .map(|home| Path::new(home).join(".cargo"))
        });
    if let Some(home) = home {
        for suffix in ["config", "config.toml"] {
            let path = home.join(suffix);
            if path.try_exists().map_err(error)? {
                result.insert(path.clone(), hash(&path)?);
            }
        }
    }
    Ok(result)
}
fn verify_inputs(inputs: &BTreeMap<PathBuf, String>) -> Result<(), String> {
    for (path, want) in inputs {
        if hash(path)? != *want {
            return Err(format!("native build input changed: {}", path.display()));
        }
    }
    Ok(())
}
fn tool_path(bytes: &[u8]) -> Result<PathBuf, String> {
    let path = PathBuf::from(std::str::from_utf8(bytes).map_err(error)?.trim());
    if !path.is_absolute() {
        return Err("absolute pinned Rust tool path required".into());
    }
    Ok(path)
}
/// # Errors
/// Refuses identities with incompatible schemas, features, source owners or executable digests.
pub fn validate_identity(
    identity: &NativeIdentity,
    role: &Role,
    digest: &str,
) -> Result<(), String> {
    let source = NativeSourceProof {
        repo: PathBuf::new(),
        host_manifest: inputs::SOURCE_INPUTS.into(),
        helper_manifest: gomad_wasmhost::linked::SOURCE_INPUTS.into(),
        current: true,
    };
    validate_identity_source(identity, role, digest, &source)
}
fn validate_identity_source(
    identity: &NativeIdentity,
    role: &Role,
    digest: &str,
    source: &NativeSourceProof,
) -> Result<(), String> {
    if identity.schema != "gomad.native-measurement-identity/v1"
        || (source.current
            && identity.runtime_implementation_sha256
                != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"))
        || identity.runtime_implementation_sha256.len() != 64
        || !identity
            .runtime_implementation_sha256
            .bytes()
            .all(|byte| byte.is_ascii_hexdigit())
        || identity.configuration
            != format!(
                "{}{}",
                gomad_wasmhost::linked::CONFIGURATION,
                if role.identity_role()? == "ordinary" {
                    String::new()
                } else {
                    format!("-{}", role.identity_role()?)
                }
            )
        || identity.role != role.identity_role()?
        || identity.features != role.features()?
        || identity.executable_sha256 != digest
        || identity.source_inputs
            != BTreeMap::from([
                ("host".into(), source.host_manifest.clone()),
                ("helper".into(), source.helper_manifest.clone()),
            ])
        || [
            identity.host_implementation_sha256.as_str(),
            identity.linked_implementation_sha256.as_str(),
        ]
        .iter()
        .any(|digest| digest.len() != 64 || !digest.bytes().all(|byte| byte.is_ascii_hexdigit()))
    {
        return Err(
            "native executable role, features, source closure or implementation identity mismatch"
                .into(),
        );
    }
    Ok(())
}
fn executable_identity(
    root: &Path,
    label: &str,
    executable: &Path,
    environment: &BTreeMap<String, String>,
    role: &Role,
) -> Result<NativeIdentity, String> {
    let before = hash(executable)?;
    let bytes = command(
        root,
        label,
        root,
        environment,
        &[
            executable.to_string_lossy().into_owned(),
            "--identity".into(),
        ],
    )?;
    let identity: NativeIdentity = serde_json::from_slice(&bytes).map_err(error)?;
    validate_identity(&identity, role, &before)?;
    if hash(executable)? != before {
        return Err("native executable changed during identity request".into());
    }
    Ok(identity)
}

/// # Panics
/// Panics if a validated Cargo executable field is not a string.
///
/// # Errors
/// Refuses malformed Cargo output, unsuccessful builds or ambiguous release artifacts.
pub fn cargo_artifact(data: &[u8], role: &Role) -> Result<PathBuf, String> {
    let mut found = None;
    let mut finished = false;
    for frame in serde_json::Deserializer::from_slice(data).into_iter::<serde_json::Value>() {
        let frame = frame.map_err(error)?;
        if frame["reason"] == "compiler-artifact"
            && frame["target"]["name"] == "gomad-measure"
            && frame
                .get("executable")
                .is_some_and(serde_json::Value::is_string)
        {
            let mut features: Vec<String> =
                serde_json::from_value(frame["features"].clone()).map_err(error)?;
            features.sort();
            if found.is_some()
                || features != cargo_features(role)?
                || frame["profile"]["opt_level"] != "3"
                || frame["profile"]["debug_assertions"] != false
                || frame["target"]["kind"] != serde_json::json!(["bin"])
            {
                return Err(
                    "duplicate Cargo artifact or unexpected native release role/features".into(),
                );
            }
            found = Some(PathBuf::from(frame["executable"].as_str().unwrap()));
        }
        if frame["reason"] == "build-finished" {
            finished = frame["success"] == true;
        }
    }
    if !finished {
        return Err("successful native release artifact absent from Cargo output".into());
    }
    found.ok_or_else(|| "successful native release artifact absent from Cargo output".into())
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep the bounded command or admission transaction and its refusal order together."
)]
/// # Errors
/// Returns errors from tool identity checks, isolated compilation, receipt validation or staging.
pub fn build_role(
    proof: &CheckoutProof,
    root: &Path,
    role: &Role,
    environ: &[String],
) -> Result<PathBuf, String> {
    role.features()?;
    let root = directory(root, "release-native")?;
    let crate_root = proof.repo().join("tools/gomad/rust/gomad");
    let mut environment = inherited(environ);
    environment.insert(
        "CARGO_TARGET_DIR".into(),
        root.join("target").to_string_lossy().into_owned(),
    );
    let mut inputs = semantic_inputs(proof, &environment)?;
    let rustc = tool_path(&command(
        &root,
        "rustc-path",
        &crate_root,
        &environment,
        &[
            "rustup".into(),
            "which".into(),
            "--toolchain".into(),
            "1.99.1".into(),
            "rustc".into(),
        ],
    )?)?;
    let cargo = tool_path(&command(
        &root,
        "cargo-path",
        &crate_root,
        &environment,
        &[
            "rustup".into(),
            "which".into(),
            "--toolchain".into(),
            "1.99.1".into(),
            "cargo".into(),
        ],
    )?)?;
    inputs.insert(rustc.clone(), hash(&rustc)?);
    inputs.insert(cargo.clone(), hash(&cargo)?);
    let rustc_version = String::from_utf8(command(
        &root,
        "rustc-version",
        &crate_root,
        &environment,
        &[rustc.to_string_lossy().into_owned(), "-vV".into()],
    )?)
    .map_err(error)?;
    let cargo_version = String::from_utf8(command(
        &root,
        "cargo-version",
        &crate_root,
        &environment,
        &[cargo.to_string_lossy().into_owned(), "--version".into()],
    )?)
    .map_err(error)?;
    if !rustc_version.starts_with("rustc 1.99.1 ") || !cargo_version.starts_with("cargo 1.99.1 ") {
        return Err("native measurement build requires pinned Rust/Cargo 1.99.1".into());
    }
    environment.extend([
        ("RUSTC".into(), rustc.to_string_lossy().into_owned()),
        ("RUSTC_WRAPPER".into(), String::new()),
        ("RUSTC_WORKSPACE_WRAPPER".into(), String::new()),
    ]);
    let argv = cargo_command(&cargo, &crate_root, role)?;
    let output = command(&root, "cargo-build", &crate_root, &environment, &argv)?;
    let executable = cargo_artifact(&output, role)?;
    if executable != root.join("target/release/gomad-measure") {
        return Err("Cargo did not produce the fresh isolated native release executable".into());
    }
    let mut current = semantic_inputs(proof, &environment)?;
    current.insert(rustc.clone(), hash(&rustc)?);
    current.insert(cargo.clone(), hash(&cargo)?);
    if current != inputs {
        return Err("native Rust semantic inputs changed during build".into());
    }
    let executable_sha256 = hash(&executable)?;
    let identity =
        executable_identity(&root, "artifact-identity", &executable, &environment, role)?;
    for label in [
        "rustc-path",
        "cargo-path",
        "rustc-version",
        "cargo-version",
        "cargo-build",
        "artifact-identity",
    ] {
        for suffix in ["stdout", "stderr", "json"] {
            let path = root.join(format!("{label}.{suffix}"));
            inputs.insert(path.clone(), hash(&path)?);
        }
    }
    let receipt = BuildReceipt {
        schema: BUILD_SCHEMA.into(),
        crate_root,
        executable,
        executable_sha256,
        rustc,
        cargo,
        inputs,
        command: argv,
        environment,
        role: role.clone(),
        rustc_version,
        cargo_version,
        identity,
    };
    let receipt_path = root.join("helper-build.json");
    write_json(&receipt_path, &receipt)?;
    stage_role(proof, &root, &receipt_path, role)?;
    Ok(receipt_path)
}

/// # Errors
/// Refuses receipts with changed source, build commands, role features or artifact bindings.
pub fn validate_build_receipt(
    proof: &CheckoutProof,
    receipt: &BuildReceipt,
    expected: &Role,
) -> Result<(), String> {
    validate_build_receipt_source(&NativeSourceProof::current(proof), receipt, expected)
}
#[expect(
    clippy::too_many_lines,
    reason = "Keep the bounded command or admission transaction and its refusal order together."
)]
fn validate_build_receipt_source(
    source: &NativeSourceProof,
    receipt: &BuildReceipt,
    expected: &Role,
) -> Result<(), String> {
    let crate_root = source.repo().join("tools/gomad/rust/gomad");
    if receipt.schema != BUILD_SCHEMA
        || receipt.crate_root != crate_root
        || receipt.role.features()? != expected.features()?
        || !exact_cargo_command(receipt, expected)
        || !receipt.rustc.is_absolute()
        || !receipt.cargo.is_absolute()
        || !receipt.rustc_version.starts_with("rustc 1.99.1 ")
        || !receipt.cargo_version.starts_with("cargo 1.99.1 ")
        || receipt.environment.get("RUSTC") != Some(&receipt.rustc.to_string_lossy().into_owned())
        || receipt
            .environment
            .get("RUSTC_WRAPPER")
            .is_none_or(|value| !value.is_empty())
        || receipt
            .environment
            .get("RUSTC_WORKSPACE_WRAPPER")
            .is_none_or(|value| !value.is_empty())
        || receipt.environment.keys().any(|name| {
            !matches!(
                name.as_str(),
                "PATH"
                    | "HOME"
                    | "CARGO_HOME"
                    | "RUSTUP_HOME"
                    | "TMPDIR"
                    | "CARGO_TARGET_DIR"
                    | "RUSTC"
                    | "RUSTC_WRAPPER"
                    | "RUSTC_WORKSPACE_WRAPPER"
            )
        })
    {
        return Err("native helper receipt does not bind pinned release command, role and compiler environment".into());
    }
    if !compiler_environment(
        &receipt.rustc,
        &receipt.inputs,
        &receipt
            .environment
            .iter()
            .map(|(key, value)| format!("{key}={value}"))
            .collect::<Vec<_>>(),
    ) {
        return Err("native compiler environment controls mismatch".into());
    }
    let target = receipt
        .environment
        .get("CARGO_TARGET_DIR")
        .ok_or("isolated target directory absent")?;
    if !Path::new(target).is_absolute()
        || receipt.executable != Path::new(target).join("release/gomad-measure")
    {
        return Err("native receipt does not bind isolated release executable".into());
    }
    let mut semantic = source_semantic_inputs(source, &receipt.environment)?;
    semantic.insert(receipt.rustc.clone(), hash(&receipt.rustc)?);
    semantic.insert(receipt.cargo.clone(), hash(&receipt.cargo)?);
    if semantic
        .iter()
        .any(|(path, digest)| receipt.inputs.get(path) != Some(digest))
    {
        return Err("native receipt omits or changes semantic build input".into());
    }
    verify_inputs(&receipt.inputs)?;
    if hash(&receipt.executable)? != receipt.executable_sha256 {
        return Err("native Cargo artifact identity changed".into());
    }
    let logs = Path::new(target)
        .parent()
        .ok_or("build evidence directory absent")?;
    for label in [
        "rustc-path",
        "cargo-path",
        "rustc-version",
        "cargo-version",
        "cargo-build",
        "artifact-identity",
    ] {
        for suffix in ["stdout", "stderr", "json"] {
            let path = logs.join(format!("{label}.{suffix}"));
            if receipt.inputs.get(&path) != Some(&hash(&path)?) {
                return Err("native receipt omits command evidence".into());
            }
        }
    }
    if cargo_artifact(&bounded(&logs.join("cargo-build.stdout"), LIMIT)?, expected)?
        != receipt.executable
    {
        return Err("Cargo output does not bind retained native executable".into());
    }
    let build: serde_json::Value =
        serde_json::from_slice(&bounded(&logs.join("cargo-build.json"), LIMIT)?).map_err(error)?;
    if build["command"] != serde_json::to_value(&receipt.command).map_err(error)?
        || build["environment"] != serde_json::to_value(&receipt.environment).map_err(error)?
        || build["directory"] != serde_json::to_value(&receipt.crate_root).map_err(error)?
        || build["exit_code"] != 0
    {
        return Err("retained native Cargo invocation evidence mismatch".into());
    }
    if bounded(&logs.join("rustc-version.stdout"), LIMIT)? != receipt.rustc_version.as_bytes()
        || bounded(&logs.join("cargo-version.stdout"), LIMIT)? != receipt.cargo_version.as_bytes()
    {
        return Err("retained compiler version evidence mismatch".into());
    }
    let identity: NativeIdentity =
        serde_json::from_slice(&bounded(&logs.join("artifact-identity.stdout"), LIMIT)?)
            .map_err(error)?;
    if identity != receipt.identity {
        return Err("retained native identity evidence mismatch".into());
    }
    validate_identity_source(
        &receipt.identity,
        expected,
        &receipt.executable_sha256,
        source,
    )?;
    Ok(())
}

/// # Errors
/// Refuses changed or invalid artifacts and returns filesystem errors while publishing the private executable.
pub fn stage_executable(
    artifact: &Path,
    destination: &Path,
    expected_sha256: &str,
) -> Result<(), String> {
    struct Staged(PathBuf);
    impl Drop for Staged {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }
    let metadata = fs::symlink_metadata(artifact).map_err(error)?;
    if !metadata.is_file()
        || metadata.len() == 0
        || metadata.len() > ARTIFACT_LIMIT
        || metadata.permissions().mode() & 0o111 == 0
    {
        return Err("Cargo artifact is not a bounded regular executable".into());
    }
    let bytes = bounded(artifact, ARTIFACT_LIMIT)?;
    if canonical::digest(&bytes) != expected_sha256 {
        return Err("Cargo artifact changed before staging".into());
    }
    let parent = destination.parent().ok_or("staging parent absent")?;
    let temporary = directory(parent, ".native-stage")?;
    let staged = Staged(temporary);
    let path = staged.0.join("executable");
    let mut file = OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o755)
        .open(&path)
        .map_err(error)?;
    file.write_all(&bytes).map_err(error)?;
    file.set_permissions(fs::Permissions::from_mode(0o755))
        .map_err(error)?;
    file.sync_all().map_err(error)?;
    drop(file);
    if hash(&path)? != expected_sha256 {
        return Err("staged executable differs from Cargo artifact".into());
    }
    fs::hard_link(&path, destination).map_err(|error| {
        format!("publish admitted native executable without replacement: {error}")
    })?;
    fs::File::open(parent)
        .map_err(error)?
        .sync_all()
        .map_err(error)?;
    if hash(artifact)? != expected_sha256 || hash(destination)? != expected_sha256 {
        return Err("Cargo artifact or admitted executable changed during publication".into());
    }
    Ok(())
}

/// # Errors
/// Refuses invalid build receipts or changed executable identities and returns staging errors.
pub fn stage_role(
    proof: &CheckoutProof,
    root: &Path,
    receipt_path: &Path,
    expected: &Role,
) -> Result<PathBuf, String> {
    if !receipt_path.is_absolute() {
        return Err("absolute retained native build receipt required".into());
    }
    let bytes = bounded(receipt_path, LIMIT)?;
    let receipt: BuildReceipt = serde_json::from_slice(&bytes).map_err(error)?;
    validate_build_receipt(proof, &receipt, expected)?;
    let root = directory(root, "admitted-native")?;
    let executable = root.join("gomad-measure");
    let identity = executable_identity(
        &root,
        "artifact-identity",
        &receipt.executable,
        &receipt.environment,
        expected,
    )?;
    if identity != receipt.identity {
        return Err("retained Cargo artifact identity does not match build receipt".into());
    }
    stage_executable(&receipt.executable, &executable, &receipt.executable_sha256)?;
    validate_build_receipt(proof, &receipt, expected)?;
    let staged_identity = executable_identity(
        &root,
        "staged-identity",
        &executable,
        &receipt.environment,
        expected,
    )?;
    if staged_identity != identity {
        return Err(
            "admitted native executable role/source identity changed during publication".into(),
        );
    }
    if hash(receipt_path)? != canonical::digest(&bytes) {
        return Err("native build receipt changed during staging".into());
    }
    let admission = Admission {
        schema: ADMISSION_SCHEMA.into(),
        build_receipt: receipt_path.into(),
        build_receipt_sha256: canonical::digest(&bytes),
        cargo_artifact: receipt.executable,
        cargo_artifact_sha256: receipt.executable_sha256.clone(),
        executable,
        executable_sha256: receipt.executable_sha256,
        identity: staged_identity,
    };
    let path = root.join("helper-admission.json");
    write_json(&path, &admission)?;
    Ok(path)
}

/// # Errors
/// Refuses an admission with changed source, receipt, identity or private executable bindings.
pub fn validate_admission(
    proof: &CheckoutProof,
    path: &Path,
    expected: &Role,
) -> Result<Admission, String> {
    validate_admission_source(&NativeSourceProof::current(proof), path, expected)
}
fn validate_admission_source(
    source: &NativeSourceProof,
    path: &Path,
    expected: &Role,
) -> Result<Admission, String> {
    if !path.is_absolute() {
        return Err("absolute native admission receipt required".into());
    }
    let admission: Admission = serde_json::from_slice(&bounded(path, LIMIT)?).map_err(error)?;
    if admission.schema != ADMISSION_SCHEMA
        || !admission.build_receipt.is_absolute()
        || !admission.cargo_artifact.is_absolute()
        || !admission.executable.is_absolute()
        || admission.cargo_artifact == admission.executable
        || admission.cargo_artifact_sha256 != admission.executable_sha256
        || hash(&admission.build_receipt)? != admission.build_receipt_sha256
        || hash(&admission.executable)? != admission.executable_sha256
    {
        return Err("native admission does not bind immutable build receipt, Cargo artifact and measured executable".into());
    }
    let receipt: BuildReceipt =
        serde_json::from_slice(&bounded(&admission.build_receipt, LIMIT)?).map_err(error)?;
    validate_build_receipt_source(source, &receipt, expected)?;
    if admission.cargo_artifact != receipt.executable
        || admission.cargo_artifact_sha256 != receipt.executable_sha256
        || admission.identity != receipt.identity
    {
        return Err("native admission changed raw Cargo build bindings".into());
    }
    let metadata = fs::symlink_metadata(&admission.executable).map_err(error)?;
    if !metadata.is_file()
        || metadata.nlink() != 1
        || metadata.permissions().mode() & 0o777 != 0o755
    {
        return Err("admitted native executable must be a private regular executable".into());
    }
    Ok(admission)
}

/// # Errors
/// Refuses historical receipts whose explicit source closure or staged executable does not match.
pub fn validate_historical_admission(
    source_root: &Path,
    receipt_path: &Path,
    expected: &Role,
) -> Result<Admission, String> {
    if !source_root.is_absolute() {
        return Err("historical native source root must be explicitly absolute".into());
    }
    let claimed: Admission =
        serde_json::from_slice(&bounded(receipt_path, LIMIT)?).map_err(error)?;
    let source = NativeSourceProof {
        repo: source_root.into(),
        host_manifest: claimed
            .identity
            .source_inputs
            .get("host")
            .ok_or("historical host source manifest absent")?
            .clone(),
        helper_manifest: claimed
            .identity
            .source_inputs
            .get("helper")
            .ok_or("historical helper source manifest absent")?
            .clone(),
        current: false,
    };
    validate_identity_source(
        &claimed.identity,
        expected,
        &claimed.executable_sha256,
        &source,
    )?;
    validate_admission_source(&source, receipt_path, expected)
}

/// # Errors
/// Refuses historical bindings or executable identity observations that fail authentication.
pub fn admit_historical(
    source_root: &Path,
    receipt_path: &Path,
    expected: &Role,
    observation_root: &Path,
    environ: &[String],
) -> Result<HistoricalAdmission, String> {
    if !source_root.is_absolute() || !observation_root.is_absolute() {
        return Err(
            "historical native source and observation roots must be explicit absolute paths".into(),
        );
    }
    let bytes = bounded(receipt_path, LIMIT)?;
    let claimed: Admission = serde_json::from_slice(&bytes).map_err(error)?;
    let source = NativeSourceProof {
        repo: source_root.into(),
        host_manifest: claimed
            .identity
            .source_inputs
            .get("host")
            .ok_or("historical host source manifest absent")?
            .clone(),
        helper_manifest: claimed
            .identity
            .source_inputs
            .get("helper")
            .ok_or("historical helper source manifest absent")?
            .clone(),
        current: false,
    };
    validate_identity_source(
        &claimed.identity,
        expected,
        &claimed.executable_sha256,
        &source,
    )?;
    let admitted = validate_admission_source(&source, receipt_path, expected)?;
    fs::create_dir_all(observation_root).map_err(error)?;
    let environment = environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(name, _)| !name.starts_with("GOMAD_"))
        .map(|(name, value)| (name.into(), value.into()))
        .collect();
    let observed = command_bounded(
        observation_root,
        "historical-native-identity",
        observation_root,
        &environment,
        &[
            admitted.executable.to_string_lossy().into_owned(),
            "--identity".into(),
        ],
        Duration::from_secs(10),
        4 << 20,
    )?;
    let identity: NativeIdentity = serde_json::from_slice(&observed).map_err(error)?;
    validate_identity_source(&identity, expected, &admitted.executable_sha256, &source)?;
    if identity != admitted.identity || hash(receipt_path)? != canonical::digest(&bytes) {
        return Err(
            "historical actual native identity or immutable admission receipt changed".into(),
        );
    }
    validate_admission_source(&source, receipt_path, expected)?;
    Ok(HistoricalAdmission {
        admission: admitted,
        source,
        receipt: receipt_path.into(),
        role: expected.clone(),
    })
}

/// # Errors
/// Returns filesystem or serialization errors while retaining compiled source ownership.
pub fn compiled_owners(root: &Path) -> Result<PathBuf, String> {
    let owners = BTreeMap::from([
        ("host", env!("GOMAD_HOST_IMPLEMENTATION_SHA256")),
        ("linked", gomad_wasmhost::linked::IMPLEMENTATION_SHA256),
        ("runtime", env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")),
    ]);
    for (name, digest) in &owners {
        if digest.len() != 64 || !digest.bytes().all(|byte| byte.is_ascii_hexdigit()) {
            return Err(format!(
                "missing compiled owner identity: {name}={digest:?}"
            ));
        }
    }
    if !root.is_absolute() {
        return Err("identity capture requires absolute root".into());
    }
    fs::create_dir_all(root).map_err(error)?;
    let path = root.join("compiled-owner-identities.json");
    write_json(
        &path,
        &serde_json::json!({"schema":"gomad.native-compiled-owner-identities/v1","owners":owners,"source_inputs":{"host":inputs::SOURCE_INPUTS,"linked":gomad_wasmhost::linked::SOURCE_INPUTS}}),
    )?;
    Ok(path)
}

#[cfg(test)]
#[path = "build_tests.rs"]
mod tests;
