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
    let mut values = BTreeMap::new();
    for (name, value) in std::env::vars_os() {
        let name = name
            .into_string()
            .map_err(|_| failure("compiler environment name is not UTF-8"))?;
        if matches!(
            name.as_str(),
            "GOROOT"
                | "GOENV"
                | "GOFLAGS"
                | "GOTOOLCHAIN"
                | "GOWORK"
                | "GOOS"
                | "GOARCH"
                | "CGO_ENABLED"
                | "GOEXPERIMENT"
                | "GOMADSEED"
                | "GOMAD_CHILD_SEED"
                | "GOCACHE"
                | "GOCACHEPROG"
                | "GOWASM"
                | "GODEBUG"
        ) {
            continue;
        }
        #[cfg(feature = "determinism-fixture")]
        if name.starts_with("GOMAD_FIXTURE_") {
            continue;
        }
        values.insert(
            name,
            value
                .into_string()
                .map_err(|_| failure("compiler environment value is not UTF-8"))?,
        );
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

pub(crate) fn command<K: AsRef<OsStr>, V: AsRef<OsStr>>(
    compiler: &Path,
    args: &[String],
    directory: &Path,
    environment: &BTreeMap<K, V>,
    scratch: &Path,
    limit: u64,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
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
            let output = super::read(&stdout, limit);
            check_deadline(deadline)?;
            return output;
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
