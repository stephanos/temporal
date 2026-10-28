#![forbid(unsafe_code)]
#![allow(missing_docs)]
use std::{
    fs::{self, File},
    io::{Read, Seek},
    os::unix::fs::{MetadataExt, PermissionsExt},
    path::{Path, PathBuf},
    process::{Command, ExitStatus},
    sync::atomic::{AtomicBool, Ordering},
    thread,
    time::{Duration, Instant},
};
/// Hard subprocess admission limits. A limit produces a refusal, never a verdict.
#[derive(Clone, Copy, Debug)]
pub struct Limits {
    /// Maximum wall time per Java process.
    pub timeout: Duration,
    /// Combined maximum stdout/stderr bytes per process and per check.
    pub output_bytes: u64,
    /// Maximum reachable dump size.
    pub dump_bytes: u64,
    /// Maximum Java heap in MiB.
    pub heap_mb: u32,
}

impl Default for Limits {
    fn default() -> Self {
        Self {
            timeout: Duration::from_secs(60),
            output_bytes: 8 * 1024 * 1024,
            dump_bytes: 128 * 1024 * 1024,
            heap_mb: 512,
        }
    }
}

/// A backend admission, execution or decoding failure; never a property verdict.
#[derive(Debug, thiserror::Error)]
#[error("TLC refusal: {0}")]
pub struct Refusal(pub String);

impl From<std::io::Error> for Refusal {
    fn from(error: std::io::Error) -> Self {
        Self(error.to_string())
    }
}

pub struct Scratch(tempfile::TempDir);

impl Scratch {
    pub fn new(root: &Path) -> Result<Self, Refusal> {
        Self::new_in(root, "build/tlc")
    }

    pub fn new_in(root: &Path, relative: &str) -> Result<Self, Refusal> {
        let root = root.canonicalize()?;
        let mut parent = root.clone();
        for component in Path::new(relative).components() {
            let std::path::Component::Normal(component) = component else {
                return Err(Refusal("scratch path must be workspace relative".into()));
            };
            parent.push(component);
            match fs::symlink_metadata(&parent) {
                Ok(meta) if !meta.is_dir() || meta.is_symlink() => {
                    return Err(Refusal(
                        "scratch directory contains a symlink or non-directory".into(),
                    ));
                }
                Ok(_) => {}
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                    match fs::create_dir(&parent) {
                        Ok(()) => {}
                        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
                        Err(error) => return Err(error.into()),
                    }
                }
                Err(error) => return Err(error.into()),
            }
            if parent.canonicalize()? != parent {
                return Err(Refusal("scratch directory escapes the repository".into()));
            }
        }
        let directory = tempfile::Builder::new()
            .prefix("run-")
            .permissions(fs::Permissions::from_mode(0o700))
            .tempdir_in(&parent)?;
        Ok(Self(directory))
    }

    pub fn path(&self) -> &Path {
        self.0.path()
    }

    pub fn finish<T, E: std::fmt::Display + From<Refusal>>(
        self,
        result: Result<T, E>,
    ) -> Result<T, E> {
        match self.0.close() {
            Ok(()) => result,
            Err(error) => Err(Refusal(format!(
                "scratch cleanup failed: {error}; result: {}",
                result.err().map_or_else(|| "completed".into(), |error| error.to_string())
            ))
            .into()),
        }
    }
}

pub fn java_command(java: &Path) -> Command {
    let mut command = Command::new(java);
    for name in
        ["JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "_JAVA_OPTIONS", "CLASSPATH", "TLA_LIBRARY"]
    {
        command.env_remove(name);
    }
    command
}

pub fn pinned_java(root: &Path, directory: &Path) -> Result<PathBuf, Refusal> {
    let mut resolve = Command::new("mise");
    resolve
        .args(["which", "java"])
        .current_dir(root)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    let (status, output) = run_process(&mut resolve, directory, "mise", Limits::default(), None)?;
    let java = PathBuf::from(output.trim());
    if !status.success() || !java.is_absolute() {
        return Err(Refusal(format!("mise did not resolve an absolute Java executable: {output}")));
    }
    Ok(java)
}

pub struct ProcessFailure {
    pub refusal: Refusal,
    pub output: String,
}
impl From<Refusal> for ProcessFailure {
    fn from(refusal: Refusal) -> Self {
        Self { refusal, output: String::new() }
    }
}
impl From<std::io::Error> for ProcessFailure {
    fn from(error: std::io::Error) -> Self {
        Refusal::from(error).into()
    }
}

/// Descendant ownership policy for the established native subprocess supervisor.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum DescendantPolicy {
    /// Preserve checker isolation, including Darwin descendant prevention.
    Checker,
    /// Allow trusted development carriers to spawn children. Native guardians
    /// share cancellation and retain hierarchical completion custody; carriers
    /// such as nextest must forward cancellation and wait for their own groups.
    /// Linux additionally retains subreaper cleanup.
    Orchestration,
}

/// An owned subprocess supervised through a parent-liveness pipe.
/// Dropping closes that pipe and waits for supervisor cleanup. Abrupt owner death
/// also closes the pipe. Input is `/dev/null`; supplied files receive raw output.
/// Stop signals retain their shell status (`128 + signal`). Orchestration inherits
/// completion/cancellation pipes through trusted carriers. Nested native guardians
/// independently observe root cancellation and retain custody through their own
/// direct waits; arbitrary uncooperative detached children are outside that contract.
pub struct OwnedProcess(crate::syscall::Running);

impl OwnedProcess {
    /// Start a process using the explicit descendant policy and caller-owned output files.
    /// Explicit command arguments, environment additions/removals and cwd are used.
    /// The inherited environment remains the process environment (`env_clear` is not
    /// supported). Command stdio settings are replaced by the descriptors above.
    pub fn spawn(
        command: &Command,
        directory: &Path,
        stdout: &File,
        stderr: &File,
        policy: DescendantPolicy,
    ) -> std::io::Result<Self> {
        match policy {
            DescendantPolicy::Checker => {
                crate::syscall::Running::spawn(command, directory, stdout, stderr)
            }
            DescendantPolicy::Orchestration => crate::syscall::Running::spawn_with_policy(
                command, directory, stdout, stderr, policy,
            ),
        }
        .map(Self)
    }

    /// Start with the explicit policy and only supplied environment values.
    /// Inherited lease and lifecycle custody are still verified; Checker targets
    /// receive neither capability nor locator.
    pub fn spawn_clean(
        command: &Command,
        directory: &Path,
        stdout: &File,
        stderr: &File,
        policy: DescendantPolicy,
    ) -> std::io::Result<Self> {
        let lease = JvmLease::inherited_file()?;
        crate::syscall::Running::spawn_with_custody(
            command,
            directory,
            (stdout, stderr),
            policy,
            lease.as_ref(),
            false,
        )
        .map(Self)
    }

    /// Start orchestration retaining shared lease custody through descendant cleanup.
    pub fn spawn_leased(
        command: &Command,
        directory: &Path,
        stdout: &File,
        stderr: &File,
        lease: &JvmLease,
    ) -> std::io::Result<Self> {
        Self::spawn_with_lease(command, directory, stdout, stderr, lease, true)
    }

    /// As `spawn_leased`, using only explicitly supplied command environment values.
    /// Use this for a command prepared with `env_clear`, which cannot be introspected.
    /// Verified inherited lifecycle capabilities are propagated into that clean
    /// environment along with the lease.
    pub fn spawn_leased_clean(
        command: &Command,
        directory: &Path,
        stdout: &File,
        stderr: &File,
        lease: &JvmLease,
    ) -> std::io::Result<Self> {
        Self::spawn_with_lease(command, directory, stdout, stderr, lease, false)
    }

    fn spawn_with_lease(
        command: &Command,
        directory: &Path,
        stdout: &File,
        stderr: &File,
        lease: &JvmLease,
        inherit_environment: bool,
    ) -> std::io::Result<Self> {
        lease.verify()?;
        crate::syscall::Running::spawn_with_custody(
            command,
            directory,
            (stdout, stderr),
            DescendantPolicy::Orchestration,
            Some(&lease.file),
            inherit_environment,
        )
        .map(Self)
    }

    /// Poll the supervisor; dropping a pending process cancels and waits for cleanup.
    pub fn try_wait(&mut self) -> std::io::Result<Option<ExitStatus>> {
        self.0.try_wait()
    }
}

/// One same-user host allocation shared across repositories and isolated copies.
/// The kernel releases it after the last inherited/duplicated descriptor closes.
/// No pathname is removed and no PID or environment marker grants authority.
pub struct JvmLease {
    file: File,
    root: PathBuf,
    directory: (u64, u64),
}

impl JvmLease {
    fn host_root() -> std::io::Result<PathBuf> {
        Ok(fs::canonicalize("/tmp")?
            .join(format!("umpire-validation-{}", rustix::process::getuid().as_raw())))
    }

    /// Wait for the exclusive host allocation; targets, Cargo homes and TMPDIR do not alter it.
    pub fn acquire() -> std::io::Result<Self> {
        Self::acquire_in(Self::host_root()?)
    }

    /// Adopt an actually inherited descriptor, or acquire a new outer allocation.
    /// A present malformed locator, foreign descriptor or contended description refuses.
    pub fn inherited_or_acquire() -> std::io::Result<Self> {
        Self::inherited()?.map_or_else(Self::acquire, Ok)
    }

    /// Verify optional inherited custody for a checker guardian, without waiting.
    /// Absence does not open the host directory/file or acquire an allocation.
    /// Release custody by closing descriptions, never by explicitly unlocking them.
    pub fn inherited_file() -> std::io::Result<Option<File>> {
        Ok(Self::inherited()?.map(|lease| lease.file))
    }

    fn inherited() -> std::io::Result<Option<Self>> {
        match std::env::var_os(crate::syscall::LEASE_ENV) {
            None => Ok(None),
            Some(value) if value == crate::syscall::LEASE_FD.to_string().as_str() => {
                Self::admit_inherited(crate::syscall::inherited_lease()?, Self::host_root()?)
                    .map(Some)
            }
            Some(_) => Err(std::io::Error::other("invalid inherited JVM lease descriptor locator")),
        }
    }

    fn admit_inherited(file: File, root: PathBuf) -> std::io::Result<Self> {
        let directory = Self::directory(&root)?;
        let lease = Self { file, root, directory };
        lease.verify()?;
        let probe = Self::open_file(&lease.root, false)?;
        let actual = lease.file.metadata()?;
        let other = probe.metadata()?;
        if (actual.dev(), actual.ino()) != (other.dev(), other.ino()) {
            return Err(std::io::Error::other("inherited JVM lease changed during admission"));
        }
        match probe.try_lock() {
            Ok(()) => return Err(std::io::Error::other("unheld inherited JVM lease refused")),
            Err(fs::TryLockError::WouldBlock) => (),
            Err(fs::TryLockError::Error(error)) => return Err(error),
        }
        lease.verify()?;
        // Another holder can exit between the independent probe and this check.
        // Acceptance requires actual exclusive custody, not a race-free ownership history.
        lease.file.try_lock().map_err(|error| {
            std::io::Error::other(format!("inherited JVM lease is not exclusive: {error}"))
        })?;
        lease.verify()?;
        Ok(lease)
    }

    fn open_file(root: &Path, create: bool) -> std::io::Result<File> {
        use std::os::unix::fs::OpenOptionsExt;
        fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create(create)
            .truncate(false)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
            .open(root.join("jvm.lock"))
    }

    fn acquire_in(root: PathBuf) -> std::io::Result<Self> {
        let directory = Self::directory(&root)?;
        let file = Self::open_file(&root, true)?;
        let lease = Self { file, root, directory };
        lease.verify()?;
        lease.file.lock()?;
        lease.verify()?;
        Ok(lease)
    }

    fn directory(root: &Path) -> std::io::Result<(u64, u64)> {
        use std::os::unix::fs::DirBuilderExt;
        match fs::DirBuilder::new().mode(0o700).create(root) {
            Ok(()) => (),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
            Err(error) => return Err(error),
        }
        let metadata = fs::symlink_metadata(root)?;
        if !metadata.is_dir()
            || metadata.uid() != rustix::process::getuid().as_raw()
            || metadata.mode() & 0o777 != 0o700
            || fs::canonicalize(root)? != root
        {
            return Err(std::io::Error::other(
                "JVM lease directory is not a private stable owned directory",
            ));
        }
        Ok((metadata.dev(), metadata.ino()))
    }

    fn verify(&self) -> std::io::Result<()> {
        if Self::directory(&self.root)? != self.directory {
            return Err(std::io::Error::other("JVM lease directory changed"));
        }
        let opened = self.file.metadata()?;
        let named = fs::symlink_metadata(self.root.join("jvm.lock"))?;
        let uid = rustix::process::getuid().as_raw();
        if !opened.is_file()
            || !named.is_file()
            || opened.nlink() != 1
            || named.nlink() != 1
            || opened.uid() != uid
            || named.uid() != uid
            || opened.mode() & 0o777 != 0o600
            || named.mode() & 0o777 != 0o600
            || (opened.dev(), opened.ino()) != (named.dev(), named.ino())
        {
            return Err(std::io::Error::other(
                "JVM lease is not a stable private owned regular file",
            ));
        }
        Ok(())
    }
}

/// Run development orchestration with inherited raw stdout/stderr and owned cleanup.
/// Uses the orchestration policy above and `/dev/null` input. No wall/output limit
/// is imposed; callers needing budgets should poll `OwnedProcess` and drop on refusal.
pub fn run_owned(command: &Command, directory: &Path) -> std::io::Result<ExitStatus> {
    use std::os::fd::AsFd;
    let stdout = File::from(std::io::stdout().as_fd().try_clone_to_owned()?);
    let stderr = File::from(std::io::stderr().as_fd().try_clone_to_owned()?);
    let mut process =
        OwnedProcess::spawn(command, directory, &stdout, &stderr, DescendantPolicy::Orchestration)?;
    loop {
        if let Some(status) = process.try_wait()? {
            return Ok(status);
        }
        thread::sleep(Duration::from_millis(5));
    }
}

pub fn run_process(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
) -> Result<(ExitStatus, String), Refusal> {
    run_process_cancelled(command, directory, label, limits, dump, &AtomicBool::new(false))
        .map_err(|error| error.refusal)
}

pub fn run_process_cancelled(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
    cancelled: &AtomicBool,
) -> Result<(ExitStatus, String), ProcessFailure> {
    run_process_cancelled_with_policy(
        command,
        directory,
        label,
        limits,
        dump,
        cancelled,
        DescendantPolicy::Checker,
    )
}

/// Capture bounded output using the established supervisor and explicit descendant policy.
#[allow(clippy::too_many_arguments)]
pub fn run_process_cancelled_with_policy(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
    cancelled: &AtomicBool,
    policy: DescendantPolicy,
) -> Result<(ExitStatus, String), ProcessFailure> {
    run_process_inner(command, directory, label, limits, dump, cancelled, policy, None)
}

/// Execute a backend-admitted invocation with explicit accounting provenance.
pub fn run_process_observed(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
    invocation: &crate::work::TlcInvocation,
) -> Result<(ExitStatus, String), ProcessFailure> {
    run_process_inner(
        command,
        directory,
        label,
        limits,
        dump,
        &AtomicBool::new(false),
        DescendantPolicy::Checker,
        Some(invocation),
    )
}

#[allow(clippy::too_many_arguments)]
fn run_process_inner(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
    cancelled: &AtomicBool,
    policy: DescendantPolicy,
    invocation: Option<&crate::work::TlcInvocation>,
) -> Result<(ExitStatus, String), ProcessFailure> {
    let typed = invocation
        .map(|invocation| crate::work::TlcAttempt::begin(invocation, command))
        .transpose()?
        .flatten();
    let stdout = directory.join(format!("{label}.stdout"));
    let stderr = directory.join(format!("{label}.stderr"));
    let open =
        |path: &Path| fs::OpenOptions::new().create_new(true).read(true).write(true).open(path);
    let mut out = open(&stdout)?;
    let mut err = open(&stderr)?;
    if cancelled.load(Ordering::Relaxed) {
        if let Some(attempt) = typed {
            attempt.finish(command, b"", "refused", None)?;
        }
        return Err(Refusal(format!("subprocess cancelled ({label})")).into());
    }
    let clean = invocation.and_then(|invocation| invocation.clean_command(command));
    let spawned = if let Some(clean) = &clean {
        OwnedProcess::spawn_clean(clean, directory, &out, &err, policy)
    } else {
        OwnedProcess::spawn(command, directory, &out, &err, policy)
    };
    let mut child = match spawned {
        Ok(child) => child,
        Err(error) => {
            if let Some(attempt) = typed {
                attempt.finish(command, b"", "spawn-error", None)?;
            }
            return Err(Refusal(format!("cannot spawn {label}: {error}")).into());
        }
    };
    let start = Instant::now();
    let observation = if invocation.is_none() { crate::work::spawned(command)? } else { None };
    let result = (|| loop {
        if cancelled.load(Ordering::Relaxed) {
            return Err(Refusal(format!("subprocess cancelled ({label})")));
        }
        if start.elapsed() >= limits.timeout
            || out.metadata()?.len().saturating_add(err.metadata()?.len()) > limits.output_bytes
            || !same_file(&stdout, &out)?
            || !same_file(&stderr, &err)?
            || dump.is_some_and(|path| !within_budget(path, limits.dump_bytes).unwrap_or(false))
        {
            return Err(Refusal(format!("subprocess limit or unsafe output ({label})")));
        }
        if let Some(status) = child.try_wait()? {
            return Ok(status);
        }
        thread::sleep(Duration::from_millis(5));
    })();
    drop(child);
    out.rewind()?;
    err.rewind()?;
    let mut bytes = Vec::new();
    out.by_ref().take(limits.output_bytes).read_to_end(&mut bytes)?;
    let remaining = limits.output_bytes.saturating_sub(bytes.len() as u64);
    err.by_ref().take(remaining).read_to_end(&mut bytes)?;
    let result = result.and_then(|status| {
        if out.metadata()?.len().saturating_add(err.metadata()?.len()) > limits.output_bytes
            || !same_file(&stdout, &out)?
            || !same_file(&stderr, &err)?
            || dump.is_some_and(|path| !within_budget(path, limits.dump_bytes).unwrap_or(false))
        {
            return Err(Refusal(format!(
                "subprocess output/dump limit or unsafe output ({label})"
            )));
        }
        Ok(status)
    });
    let cause = if std::str::from_utf8(&bytes).is_err() {
        "nonutf8"
    } else if result.is_err() {
        "refused"
    } else {
        "completed"
    };
    let accounting = typed
        .map(|attempt| {
            attempt.finish(command, &bytes, cause, result.as_ref().ok().and_then(ExitStatus::code))
        })
        .transpose();
    let output = match String::from_utf8(bytes) {
        Ok(output) => output,
        Err(error) => {
            let mut output = String::from_utf8_lossy(error.as_bytes()).into_owned();
            let limit = usize::try_from(limits.output_bytes).unwrap_or(usize::MAX);
            output.truncate(output.floor_char_boundary(limit));
            let refusal = match accounting {
                Ok(_) => format!("non-UTF8 subprocess output: {error}"),
                Err(accounting) => format!(
                    "non-UTF8 subprocess output: {error}; accounting observation failed: {accounting}"
                ),
            };
            crate::work::finished(observation.as_deref(), "")?;
            return Err(ProcessFailure { refusal: Refusal(refusal), output });
        }
    };
    if let Err(error) = accounting {
        return Err(ProcessFailure {
            refusal: Refusal(format!(
                "accounting observation failed: {error}; process outcome: {result:?}"
            )),
            output,
        });
    }
    let observation = crate::work::finished(observation.as_deref(), &output);
    if let Err(error) = observation {
        return Err(ProcessFailure {
            refusal: Refusal(format!(
                "accounting observation failed: {error}; process outcome: {result:?}"
            )),
            output,
        });
    }
    match result {
        Ok(status) if status.success() => Ok((status, output)),
        Ok(status) => Ok((status, output)),
        Err(refusal) => Err(ProcessFailure { refusal, output }),
    }
}

fn same_file(path: &Path, file: &File) -> Result<bool, Refusal> {
    let current = match fs::symlink_metadata(path) {
        Ok(meta) => meta,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(error) => return Err(error.into()),
    };
    let original = file.metadata()?;
    Ok(current.is_file() && current.dev() == original.dev() && current.ino() == original.ino())
}

fn within_budget(path: &Path, limit: u64) -> Result<bool, Refusal> {
    fn size(
        path: &Path,
        remaining: &mut u64,
        entries: &mut usize,
        depth: usize,
    ) -> Result<bool, Refusal> {
        if depth > 64 || *entries == 0 {
            return Ok(false);
        }
        *entries -= 1;
        let meta = match fs::symlink_metadata(path) {
            Ok(meta) => meta,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(true),
            Err(error) => return Err(error.into()),
        };
        if meta.is_symlink() {
            return Ok(false);
        }
        if meta.is_dir() {
            for entry in fs::read_dir(path)? {
                if !size(&entry?.path(), remaining, entries, depth + 1)? {
                    return Ok(false);
                }
            }
        } else if meta.is_file() && meta.len() <= *remaining {
            *remaining -= meta.len();
        } else {
            return Ok(false);
        }
        Ok(true)
    }
    size(path, &mut { limit }, &mut 4096, 0)
}

pub fn bounded_text(path: &Path, limit: u64) -> Result<String, Refusal> {
    if !fs::symlink_metadata(path)?.is_file() {
        return Err(Refusal("bounded read requires a regular file".into()));
    }
    let file = File::open(path)?;
    if !same_file(path, &file)? {
        return Err(Refusal("bounded read file changed".into()));
    }
    let mut text = String::new();
    file.take(limit.saturating_add(1)).read_to_string(&mut text)?;
    if u64::try_from(text.len()).unwrap_or(u64::MAX) > limit {
        return Err(Refusal(format!("file exceeds bounded read limit: {}", path.display())));
    }
    Ok(text)
}

#[cfg(test)]
#[allow(clippy::unwrap_used)]
mod lease_tests {
    use super::*;
    use std::os::unix::fs::{OpenOptionsExt, symlink};

    #[test]
    fn admission_refuses_symlink_directory_and_foreign_file_without_writing() {
        let temporary = tempfile::tempdir().unwrap();
        let foreign = temporary.path().join("foreign");
        fs::create_dir(&foreign).unwrap();
        let root = temporary.path().join("host");
        symlink(&foreign, &root).unwrap();
        assert!(JvmLease::acquire_in(root.clone()).is_err());
        fs::remove_file(&root).unwrap();
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let victim = temporary.path().join("victim");
        fs::write(&victim, "unchanged").unwrap();
        symlink(&victim, root.join("jvm.lock")).unwrap();
        assert!(JvmLease::acquire_in(root).is_err());
        assert_eq!(fs::read_to_string(victim).unwrap(), "unchanged");
    }

    #[test]
    fn admission_refuses_hardlinks_fifo_and_public_permissions() {
        let temporary = tempfile::tempdir().unwrap();
        let root = temporary.path().join("host");
        let lease = JvmLease::acquire_in(root.clone()).unwrap();
        drop(lease);
        fs::hard_link(root.join("jvm.lock"), temporary.path().join("alias")).unwrap();
        assert!(JvmLease::acquire_in(root.clone()).is_err());
        fs::remove_file(temporary.path().join("alias")).unwrap();
        fs::set_permissions(root.join("jvm.lock"), fs::Permissions::from_mode(0o644)).unwrap();
        assert!(JvmLease::acquire_in(root.clone()).is_err());
        fs::remove_file(root.join("jvm.lock")).unwrap();
        assert!(Command::new("mkfifo").arg(root.join("jvm.lock")).status().unwrap().success());
        assert!(JvmLease::acquire_in(root.clone()).is_err());
        fs::remove_file(root.join("jvm.lock")).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o755)).unwrap();
        assert!(JvmLease::acquire_in(root).is_err());
    }

    #[test]
    fn file_replacement_during_wait_refuses_after_original_owner_releases() {
        let temporary = tempfile::tempdir().unwrap();
        let root = temporary.path().join("host");
        let owner = JvmLease::acquire_in(root.clone()).unwrap();
        let path = root.join("jvm.lock");
        let waiting_file =
            fs::OpenOptions::new().read(true).write(true).mode(0o600).open(&path).unwrap();
        assert!(matches!(waiting_file.try_lock(), Err(fs::TryLockError::WouldBlock)));
        let waiting =
            JvmLease { file: waiting_file, root: root.clone(), directory: owner.directory };
        waiting.verify().unwrap();
        fs::rename(&path, root.join("old.lock")).unwrap();
        fs::OpenOptions::new().write(true).create_new(true).mode(0o600).open(&path).unwrap();
        drop(owner);
        waiting.file.lock().unwrap();
        assert!(waiting.verify().is_err());
    }

    #[test]
    fn kernel_lock_releases_without_unlinking_or_stale_pid_recovery() {
        let temporary = tempfile::tempdir().unwrap();
        let root = temporary.path().join("host");
        let lease = JvmLease::acquire_in(root.clone()).unwrap();
        let inode = lease.file.metadata().unwrap().ino();
        drop(lease);
        let next = JvmLease::acquire_in(root).unwrap();
        assert_eq!(next.file.metadata().unwrap().ino(), inode);
    }
    #[cfg(target_os = "linux")]
    fn control(role: &str, directory: &Path) -> Command {
        let mut command = Command::new(std::env::current_exe().unwrap());
        command
            .args(["--exact", "support::lease_tests::lifecycle_child", "--nocapture"])
            .env("UMPIRE_LEASE_CONTROL_ROLE", role)
            .env("UMPIRE_LEASE_CONTROL_DIR", directory);
        command
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn leased_owner_death_excludes_contender_until_escaped_child_is_reaped() {
        let directory = tempfile::tempdir().unwrap();
        let status = run_owned(&control("harness", directory.path()), directory.path()).unwrap();
        assert!(status.success());
        assert_eq!(fs::read_to_string(directory.path().join("acquired")).unwrap(), "verified");
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn lifecycle_child() -> std::io::Result<()> {
        let Ok(role) = std::env::var("UMPIRE_LEASE_CONTROL_ROLE") else { return Ok(()) };
        let directory = PathBuf::from(std::env::var_os("UMPIRE_LEASE_CONTROL_DIR").unwrap());
        let path = directory.join("pids");
        match role.as_str() {
            "owner" => {
                let lease = JvmLease::acquire_in(directory.join("host"))?;
                let out = File::create(directory.join("out"))?;
                let err = File::create(directory.join("err"))?;
                let mut child = OwnedProcess::spawn_leased(
                    &control("tree", &directory),
                    &directory,
                    &out,
                    &err,
                    &lease,
                )?;
                loop {
                    if child.try_wait()?.is_some() {
                        break;
                    }
                    thread::sleep(Duration::from_millis(5));
                }
            }
            "tree" => {
                let mut escaped = Command::new("setsid").args(["/bin/sleep", "60"]).spawn()?;
                fs::write(&path, format!("{} {}", std::process::id(), escaped.id()))?;
                let parent = fs::read_to_string("/proc/self/status")?;
                let parent = parent.lines().find_map(|l| l.strip_prefix("PPid:")).unwrap().trim();
                fs::write(directory.join("supervisor"), parent)?;
                thread::sleep(Duration::from_secs(60));
                escaped.wait()?;
            }
            "contender" => {
                let _lease = JvmLease::acquire_in(directory.join("host"))?;
                for pid in fs::read_to_string(&path)?.split_whitespace() {
                    assert!(
                        !Path::new("/proc").join(pid).exists(),
                        "lease released before owned PID{pid} was reaped"
                    );
                }
                fs::write(directory.join("acquired"), "verified")?;
            }
            "harness" => {
                crate::syscall::set_subreaper()?;
                let mut owner = control("owner", &directory).spawn()?;
                let started = Instant::now();
                while !path.exists() || !directory.join("supervisor").exists() {
                    assert!(
                        started.elapsed() < Duration::from_secs(5),
                        "owner did not become ready"
                    );
                    thread::sleep(Duration::from_millis(5));
                }
                let supervisor = fs::read_to_string(directory.join("supervisor"))?;
                let file =
                    fs::metadata(format!("/proc/{}/fd/9", supervisor.trim())).map_err(|error| {
                        std::io::Error::other(format!("supervisor lost lease custody: {error}"))
                    })?;
                let named = fs::metadata(directory.join("host/jvm.lock"))?;
                assert_eq!(
                    (file.dev(), file.ino()),
                    (named.dev(), named.ino()),
                    "supervisor lost custody"
                );
                let mut contender = control("contender", &directory).spawn()?;
                thread::sleep(Duration::from_millis(25));
                assert!(contender.try_wait()?.is_none());
                assert!(!directory.join("acquired").exists());
                owner.kill()?;
                owner.wait()?;
                let started = Instant::now();
                loop {
                    if let Some(status) = contender.try_wait()? {
                        assert!(status.success());
                        break;
                    }
                    assert!(
                        started.elapsed() < Duration::from_secs(5),
                        "lease did not release after abrupt owner death"
                    );
                    thread::sleep(Duration::from_millis(5));
                }
                crate::syscall::reap_owned_children()?;
            }
            _ => panic!("unknown lease lifecycle role"),
        }
        Ok(())
    }
    #[test]
    fn inherited_admission_rejects_unheld_canonical_and_held_wrong_description() {
        let temporary = tempfile::tempdir().unwrap();
        let root = temporary.path().join("host");
        let initial = JvmLease::acquire_in(root.clone()).unwrap();
        drop(initial);
        let unheld =
            fs::OpenOptions::new().read(true).write(true).open(root.join("jvm.lock")).unwrap();
        let error = JvmLease::admit_inherited(unheld, root.clone()).err().unwrap();
        assert!(error.to_string().contains("unheld"), "{error}");
        let owner = JvmLease::acquire_in(root.clone()).unwrap();
        let other =
            fs::OpenOptions::new().read(true).write(true).open(root.join("jvm.lock")).unwrap();
        assert!(JvmLease::admit_inherited(other, root.clone()).is_err());
        let admitted =
            JvmLease::admit_inherited(owner.file.try_clone().unwrap(), root.clone()).unwrap();
        drop(owner);
        let probe =
            fs::OpenOptions::new().read(true).write(true).open(root.join("jvm.lock")).unwrap();
        assert!(matches!(probe.try_lock(), Err(fs::TryLockError::WouldBlock)));
        drop(admitted);
        probe.try_lock().unwrap();
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn checker_guardian_holds_last_private_description_while_target_has_no_capability() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("host");
        let lease = JvmLease::acquire_in(root.clone()).unwrap();
        let out = File::create(directory.path().join("out")).unwrap();
        let err = File::create(directory.path().join("err")).unwrap();
        let mut command = Command::new(std::env::current_exe().unwrap());
        command
            .args(["--exact", "support::lease_tests::checker_target_control", "--nocapture"])
            .env("UMPIRE_CHECKER_PRIVATE_DIRECTORY", directory.path());
        let mut guardian = crate::syscall::Running::spawn_with_custody(
            &command,
            directory.path(),
            (&out, &err),
            DescendantPolicy::Checker,
            Some(&lease.file),
            true,
        )
        .unwrap();
        drop(lease);
        let ready = directory.path().join("checker-ready");
        let started = Instant::now();
        while !ready.exists() {
            if let Some(status) = guardian.try_wait().unwrap() {
                panic!(
                    "Checker target failed before readiness: {status}: {}",
                    fs::read_to_string(directory.path().join("err")).unwrap()
                );
            }
            assert!(
                started.elapsed() < Duration::from_secs(5),
                "Checker target failed: {}",
                fs::read_to_string(directory.path().join("err")).unwrap()
            );
            thread::sleep(Duration::from_millis(5));
        }
        let pid = fs::read_to_string(ready).unwrap();
        let descriptor = fs::metadata(format!("/proc/{}/fd/9", pid.trim())).unwrap();
        let named = fs::metadata(root.join("jvm.lock")).unwrap();
        assert_eq!((descriptor.dev(), descriptor.ino()), (named.dev(), named.ino()));
        let probe =
            fs::OpenOptions::new().read(true).write(true).open(root.join("jvm.lock")).unwrap();
        assert!(
            matches!(probe.try_lock(), Err(fs::TryLockError::WouldBlock)),
            "Checker guardian lost last custody"
        );
        fs::write(directory.path().join("checker-release"), "release").unwrap();
        let started = Instant::now();
        loop {
            if let Some(status) = guardian.try_wait().unwrap() {
                assert!(status.success());
                break;
            }
            assert!(started.elapsed() < Duration::from_secs(5));
            thread::sleep(Duration::from_millis(5));
        }
        probe.try_lock().unwrap();
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn checker_target_control() -> std::io::Result<()> {
        let Some(directory) = std::env::var_os("UMPIRE_CHECKER_PRIVATE_DIRECTORY") else {
            return Ok(());
        };
        assert!(
            std::env::var_os(crate::syscall::LEASE_ENV).is_none(),
            "Checker target received lease locator"
        );
        assert!(crate::syscall::inherited_lease().is_err(), "Checker target received FD9");
        let directory = PathBuf::from(directory);
        let status = fs::read_to_string("/proc/self/status")?;
        let parent = status.lines().find_map(|line| line.strip_prefix("PPid:")).unwrap().trim();
        fs::write(directory.join("checker-ready"), parent)?;
        let started = Instant::now();
        while !directory.join("checker-release").exists() {
            assert!(started.elapsed() < Duration::from_secs(5), "checker target release timeout");
            thread::sleep(Duration::from_millis(5));
        }
        Ok(())
    }
}
