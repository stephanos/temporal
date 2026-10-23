use super::failure;
use std::{
    collections::BTreeMap,
    ffi::OsStr,
    fs::{self, OpenOptions},
    path::{Path, PathBuf},
    process::{Command, Stdio},
    sync::atomic::{AtomicU64, Ordering},
    time::{Duration, Instant},
};

pub(crate) struct Temporary(pub PathBuf);
impl Temporary {
    pub(crate) fn new(root: &Path) -> Result<Self, String> {
        use std::os::unix::fs::DirBuilderExt as _;
        static NEXT: AtomicU64 = AtomicU64::new(0);
        fs::create_dir_all(root).map_err(failure)?;
        for _ in 0..100 {
            let path = root.join(format!(
                ".rust-preparation-{}-{}",
                std::process::id(),
                NEXT.fetch_add(1, Ordering::Relaxed)
            ));
            match fs::DirBuilder::new().mode(0o700).create(&path) {
                Ok(()) => return Ok(Self(path)),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
                Err(error) => return Err(failure(error)),
            }
        }
        Err(failure("cannot allocate private preparation directory"))
    }
}
impl Drop for Temporary {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

pub(crate) fn environment() -> Result<BTreeMap<String, String>, String> {
    controlled_environment(std::env::vars_os())
}

fn controlled_environment(
    entries: impl IntoIterator<Item = (std::ffi::OsString, std::ffi::OsString)>,
) -> Result<BTreeMap<String, String>, String> {
    let mut values = BTreeMap::new();
    for (name, value) in entries {
        let name = name
            .into_string()
            .map_err(|_| failure("compiler environment name is not UTF-8"))?;
        let value = value
            .into_string()
            .map_err(|_| failure("compiler environment value is not UTF-8"))?;
        // Only explicitly controlled compiler inputs reach the child or its observations.
        if !matches!(
            name.as_str(),
            "PATH"
                | "TMPDIR"
                | "HOME"
                | "GOPATH"
                | "GOMODCACHE"
                | "XDG_CACHE_HOME"
                | "GOMAXPROCS"
                | "GOTMPDIR"
        ) {
            continue;
        }
        values.insert(name, value);
    }
    for (name, value) in [
        ("GOENV", "off"),
        ("GOPROXY", "off"),
        ("GOSUMDB", "off"),
        ("GOPRIVATE", ""),
        ("GONOPROXY", "none"),
        ("GOVCS", "*:off"),
        ("GOFLAGS", ""),
        ("GOTOOLCHAIN", "local"),
        ("GOWORK", "off"),
        ("GOOS", "wasip1"),
        ("GOARCH", "wasm"),
        ("CGO_ENABLED", "0"),
        ("GOEXPERIMENT", "nogreenteagc"),
        ("GOCACHEPROG", ""),
        ("GOWASM", ""),
        ("GODEBUG", ""),
    ] {
        values.insert(name.into(), value.into());
    }
    Ok(values)
}

pub(crate) fn check_deadline(deadline: Instant) -> Result<(), String> {
    if Instant::now() >= deadline {
        return Err("overall_timeout: preparation budget exhausted".into());
    }
    Ok(())
}

pub(crate) struct ObservedCommand {
    pub argv: Vec<String>,
    pub cwd: PathBuf,
    pub environment: BTreeMap<String, String>,
    pub owner_pid: u32,
    pub pid: u32,
    pub start_ns: u64,
    pub end_ns: u64,
    pub exit: i32,
    pub output_limit: u64,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}

fn observed_time() -> Result<u64, String> {
    u64::try_from(
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(failure)?
            .as_nanos(),
    )
    .map_err(failure)
}

pub(crate) fn command<K: AsRef<OsStr>, V: AsRef<OsStr>>(
    compiler: &Path,
    args: &[String],
    directory: &Path,
    environment: &BTreeMap<K, V>,
    scratch: &Path,
    limit: u64,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    execute_command(
        compiler,
        args,
        directory,
        environment,
        scratch,
        limit,
        deadline,
    )
    .map(|command| command.stdout)
}

pub(crate) fn observed_command(
    compiler: &Path,
    args: &[String],
    directory: &Path,
    environment: &BTreeMap<String, String>,
    scratch: &Path,
    limit: u64,
    deadline: Instant,
) -> Result<ObservedCommand, String> {
    let executed = execute_command(
        compiler,
        args,
        directory,
        environment,
        scratch,
        limit,
        deadline,
    )?;
    Ok(ObservedCommand {
        argv: std::iter::once(compiler.to_string_lossy().into_owned())
            .chain(args.iter().cloned())
            .collect(),
        cwd: directory.into(),
        environment: environment.clone(),
        owner_pid: std::process::id(),
        pid: executed.pid,
        start_ns: executed.start_ns,
        end_ns: executed.end_ns,
        exit: executed.exit,
        output_limit: limit,
        stdout: executed.stdout,
        stderr: executed.stderr,
    })
}

struct ExecutedCommand {
    pid: u32,
    start_ns: u64,
    end_ns: u64,
    exit: i32,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
}

fn execute_command<K: AsRef<OsStr>, V: AsRef<OsStr>>(
    compiler: &Path,
    args: &[String],
    directory: &Path,
    environment: &BTreeMap<K, V>,
    scratch: &Path,
    limit: u64,
    deadline: Instant,
) -> Result<ExecutedCommand, String> {
    use std::os::unix::process::CommandExt as _;
    check_deadline(deadline)?;
    let stdout = scratch.join("command.stdout");
    let stderr = scratch.join("command.stderr");
    let open = |path: &Path| {
        OpenOptions::new()
            .write(true)
            .create(true)
            .truncate(true)
            .open(path)
            .map_err(failure)
    };
    let start_ns = observed_time()?;
    let child = Command::new(compiler)
        .args(args)
        .current_dir(directory)
        .env_clear()
        .envs(environment)
        .stdin(Stdio::null())
        .stdout(Stdio::from(open(&stdout)?))
        .stderr(Stdio::from(open(&stderr)?))
        .process_group(0)
        .spawn()
        .map_err(failure)?;
    let mut owner = Child(child);
    let child = &mut owner.0;
    let command_deadline = Instant::now() + Duration::from_secs(600);
    loop {
        check_deadline(deadline)?;
        if let Some(status) = child.try_wait().map_err(failure)? {
            if !status.success() {
                return Err(failure(format!(
                    "compiler command failed: {}",
                    String::from_utf8_lossy(&super::read(&stderr, limit)?)
                )));
            }
            let output = super::read(&stdout, limit)?;
            let errors = super::read(&stderr, limit)?;
            let end_ns = observed_time()?;
            check_deadline(deadline)?;
            return Ok(ExecutedCommand {
                pid: child.id(),
                start_ns,
                end_ns,
                exit: status
                    .code()
                    .ok_or_else(|| failure("successful compiler command lacks exit code"))?,
                stdout: output,
                stderr: errors,
            });
        }
        let overflow = fs::metadata(&stdout).map_err(failure)?.len() > limit
            || fs::metadata(&stderr).map_err(failure)?.len() > limit;
        if overflow || Instant::now() >= command_deadline {
            return Err(if overflow {
                "capacity: compiler output exceeds bound".into()
            } else {
                failure("compiler command timeout")
            });
        }
        std::thread::sleep(
            Duration::from_millis(10).min(
                deadline
                    .min(command_deadline)
                    .saturating_duration_since(Instant::now()),
            ),
        );
    }
}

struct Child(std::process::Child);
impl Drop for Child {
    fn drop(&mut self) {
        if let Ok(raw) = i32::try_from(self.0.id())
            && let Some(pid) = rustix::process::Pid::from_raw(raw)
        {
            let _ = rustix::process::kill_process_group(pid, rustix::process::Signal::KILL);
        }
        let _ = self.0.kill();
        let _ = self.0.wait();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn receiving_compiler_environment_is_exact_controlled_map_without_unrelated_host_values() {
        let raw = [
            ("PATH", "/compiler/bin"),
            ("HOME", "/compiler/home"),
            ("GOMODCACHE", "/frozen/modules"),
            ("TMPDIR", "/private/tmp"),
            ("GOFLAGS", "-race"),
            ("GOWASM", "satconv,signext"),
            ("GODEBUG", "foreign"),
            ("SECRET_TOKEN", "must-never-reach-compiler-or-receipt"),
        ];
        let actual =
            controlled_environment(raw.map(|(key, value)| (key.into(), value.into()))).unwrap();
        assert_eq!(actual["PATH"], "/compiler/bin");
        assert_eq!(actual["HOME"], "/compiler/home");
        assert_eq!(actual["GOMODCACHE"], "/frozen/modules");
        assert_eq!(actual["TMPDIR"], "/private/tmp");
        for key in ["GOFLAGS", "GOWASM", "GODEBUG", "GOCACHEPROG"] {
            assert_eq!(actual[key], "");
        }
        assert_eq!(actual["GOOS"], "wasip1");
        assert_eq!(actual["GOARCH"], "wasm");
        assert!(!actual.contains_key("SECRET_TOKEN"));
        assert_eq!(actual.len(), 20);
    }
    #[test]
    fn receiving_compiler_environment_preserves_inherited_utf8_refusal_without_logging_value() {
        use std::os::unix::ffi::OsStringExt as _;
        let error = controlled_environment([(
            std::ffi::OsString::from("SECRET_TOKEN"),
            std::ffi::OsString::from_vec(vec![255]),
        )])
        .unwrap_err();
        assert!(error.contains("not UTF-8"));
        assert!(!error.contains("SECRET_TOKEN"));
    }
}
