//! Shared bounded subprocess and workspace scratch support for lifted backends.
#![allow(missing_docs)]
use crate::{Limits, Refusal};
use std::{
    fs::{self, File},
    io::Read,
    os::unix::fs::DirBuilderExt,
    path::{Path, PathBuf},
    process::{Child, Command, ExitStatus, Stdio},
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    thread,
    time::{Duration, Instant},
};
static NEXT_RUN: AtomicU64 = AtomicU64::new(0);
pub struct Scratch(pub PathBuf);

impl Scratch {
    pub fn new(root: &Path) -> Result<Self, Refusal> {
        Self::new_in(root, "model/build/tlc")
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
        loop {
            let path = parent.join(format!(
                "run-{}-{}",
                std::process::id(),
                NEXT_RUN.fetch_add(1, Ordering::Relaxed)
            ));
            match fs::DirBuilder::new().mode(0o700).create(&path) {
                Ok(()) => return Ok(Self(path)),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
                Err(error) => return Err(error.into()),
            }
        }
    }

    pub fn finish<T>(self, result: Result<T, Refusal>) -> Result<T, Refusal> {
        match fs::remove_dir_all(&self.0) {
            Ok(()) => result,
            Err(error) => Err(Refusal(format!(
                "TLC cleanup failed: {error}; result: {}",
                result.err().map_or_else(|| "completed".into(), |error| error.to_string())
            ))),
        }
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

const JVM_POLICY: &str = "UMPIRE_INTERNAL_JVM_PROCESS";

pub fn java_command(java: &Path) -> Command {
    let mut command = Command::new(java);
    command.env(JVM_POLICY, "1");
    for name in
        ["JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "_JAVA_OPTIONS", "CLASSPATH", "TLA_LIBRARY"]
    {
        command.env_remove(name);
    }
    command
}

pub fn pinned_java(root: &Path, directory: &Path) -> Result<PathBuf, Refusal> {
    if !fs::read_to_string(root.join("mise.toml"))?
        .lines()
        .any(|line| line.trim() == "java = \"temurin-27\"")
    {
        return Err(Refusal("expected repository mise Java pin temurin-27".into()));
    }
    let mut resolve = Command::new("mise");
    resolve.args(["which", "java"]).current_dir(root);
    let (status, output) = run_process(&mut resolve, directory, "mise", Limits::default(), None)?;
    let java = PathBuf::from(output.trim());
    if !status.success() || !java.is_absolute() || !java.is_file() {
        return Err(Refusal(format!("mise did not resolve an absolute Java executable: {output}")));
    }
    let mut version = java_command(&java);
    version.arg("-version").current_dir(directory);
    let (status, output) = run_process(&mut version, directory, "java", Limits::default(), None)?;
    validate_java(status.success(), &output)?;
    Ok(java)
}

fn validate_java(success: bool, output: &str) -> Result<(), Refusal> {
    if !success
        || !output.lines().any(|line| line.starts_with("openjdk version \"27\""))
        || !output.contains("Temurin-27+")
    {
        return Err(Refusal(format!("mise Java is not pinned Temurin 27: {output}")));
    }
    Ok(())
}

struct Running(Child);

impl Drop for Running {
    fn drop(&mut self) {
        drop(self.0.stdin.take());
        let _ = self.0.wait();
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
}

pub fn run_process_cancelled(
    command: &mut Command,
    directory: &Path,
    label: &str,
    limits: Limits,
    dump: Option<&Path>,
    cancelled: &AtomicBool,
) -> Result<(ExitStatus, String), Refusal> {
    let stdout = directory.join(format!("{label}.stdout"));
    let stderr = directory.join(format!("{label}.stderr"));
    File::create(&stdout)?;
    File::create(&stderr)?;
    if cancelled.load(Ordering::Relaxed) {
        return Err(Refusal(format!("subprocess cancelled ({label})")));
    }
    let mut supervised = Command::new("python3");
    let jvm = command
        .get_envs()
        .any(|(name, value)| name == JVM_POLICY && value == Some(std::ffi::OsStr::new("1")));
    supervised
        .arg("-c")
        .arg(include_str!("process.py"))
        .arg(if jvm { "jvm" } else { "process" })
        .arg(command.get_program());
    supervised.args(command.get_args());
    supervised.current_dir(command.get_current_dir().unwrap_or(directory));
    for (key, value) in command.get_envs() {
        if key == JVM_POLICY {
            continue;
        }
        if let Some(value) = value {
            supervised.env(key, value);
        } else {
            supervised.env_remove(key);
        }
    }
    supervised.stdin(Stdio::piped()).stdout(Stdio::piped()).stderr(Stdio::piped());
    let mut child = Running(
        supervised.spawn().map_err(|error| Refusal(format!("cannot spawn {label}: {error}")))?,
    );
    let bytes = Arc::new(AtomicU64::new(0));
    let out = drain(child.0.stdout.take(), bytes.clone(), limits.output_bytes);
    let err = drain(child.0.stderr.take(), bytes.clone(), limits.output_bytes);
    let start = Instant::now();
    let result = (|| loop {
        if cancelled.load(Ordering::Relaxed) {
            return Err(Refusal(format!("subprocess cancelled ({label})")));
        }
        if start.elapsed() >= limits.timeout
            || bytes.load(Ordering::Relaxed) > limits.output_bytes
            || dump.is_some_and(|path| !within_budget(path, limits.dump_bytes).unwrap_or(false))
        {
            return Err(Refusal(format!("subprocess limit exceeded ({label})")));
        }
        if let Some(status) = child.0.try_wait()? {
            return Ok(status);
        }
        thread::sleep(Duration::from_millis(5));
    })();
    drop(child);
    let output = out.join().map_err(|_| Refusal("stdout reader panicked".into()))??;
    let errors = err.join().map_err(|_| Refusal("stderr reader panicked".into()))??;
    fs::write(stdout, &output)?;
    fs::write(stderr, &errors)?;
    let status = result?;
    if bytes.load(Ordering::Relaxed) > limits.output_bytes
        || dump.is_some_and(|path| !within_budget(path, limits.dump_bytes).unwrap_or(false))
    {
        return Err(Refusal(format!("subprocess output/dump limit exceeded ({label})")));
    }
    let output = String::from_utf8([output, errors].concat())
        .map_err(|error| Refusal(format!("non-UTF8 subprocess output: {error}")))?;
    Ok((status, output))
}

fn drain<R: Read + Send + 'static>(
    reader: Option<R>,
    bytes: Arc<AtomicU64>,
    limit: u64,
) -> thread::JoinHandle<Result<Vec<u8>, Refusal>> {
    thread::spawn(move || {
        let mut reader = reader.ok_or_else(|| Refusal("missing subprocess pipe".into()))?;
        let mut output = Vec::new();
        let mut buffer = [0; 8192];
        loop {
            let count = reader.read(&mut buffer)?;
            if count == 0 {
                break;
            }
            let before = bytes.fetch_add(count as u64, Ordering::Relaxed);
            let keep =
                usize::try_from(limit.saturating_sub(before)).unwrap_or(usize::MAX).min(count);
            output.extend_from_slice(&buffer[..keep]);
        }
        Ok(output)
    })
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

pub(crate) fn bounded_text(path: &Path, limit: u64) -> Result<String, Refusal> {
    let mut text = String::new();
    File::open(path)?.take(limit.saturating_add(1)).read_to_string(&mut text)?;
    if u64::try_from(text.len()).unwrap_or(u64::MAX) > limit {
        return Err(Refusal(format!("file exceeds bounded read limit: {}", path.display())));
    }
    Ok(text)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn timeout_cleans_descendants_before_return() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let heartbeat = scratch.0.join("heartbeat");
        let mut command = Command::new("python3");
        command.args(["-c", "import subprocess,sys,time; subprocess.Popen([sys.executable,'-c', 'import time,sys; time.sleep(.2); open(sys.argv[1],\"w\").write(\"leaked\")',sys.argv[1]]); time.sleep(10)"]).arg(&heartbeat);
        let limits = Limits { timeout: Duration::from_millis(100), ..Limits::default() };
        assert!(run_process(&mut command, &scratch.0, "tree", limits, None).is_err());
        thread::sleep(Duration::from_millis(300));
        assert!(!heartbeat.exists(), "descendant survived timeout");
        scratch.finish(Ok(()))
    }
    #[test]
    #[cfg(target_os = "linux")]
    fn cancellation_reaps_descendants_in_separate_sessions_and_closes_stdin() -> Result<(), Refusal>
    {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let pids = scratch.0.join("pids");
        let mut command = Command::new("python3");
        command.args(["-c", "import os,subprocess,sys,time; assert sys.stdin.read() == ''; child=subprocess.Popen([sys.executable,'-c','import time; time.sleep(60)'],start_new_session=True); open(sys.argv[1],'w').write(str(os.getpid())+' '+str(child.pid)); time.sleep(60)"]).arg(&pids);
        let cancelled = Arc::new(AtomicBool::new(false));
        let flag = cancelled.clone();
        let ready = pids.clone();
        let cancel = thread::spawn(move || {
            let start = Instant::now();
            while !ready.exists() && start.elapsed() < Duration::from_secs(5) {
                thread::sleep(Duration::from_millis(5));
            }
            flag.store(true, Ordering::Relaxed);
        });
        let result = run_process_cancelled(
            &mut command,
            &scratch.0,
            "cancel",
            Limits::default(),
            None,
            &cancelled,
        );
        cancel.join().map_err(|_| Refusal("cancel thread panicked".into()))?;
        assert!(result.is_err_and(|e| e.0.contains("cancelled")));
        let pids = fs::read_to_string(pids)?;
        for pid in pids.split_whitespace() {
            assert!(!Path::new("/proc").join(pid).exists(), "owned pid {pid} was not reaped");
        }
        let path = scratch.0.clone();
        scratch.finish(Ok(()))?;
        assert!(!path.exists());
        Ok(())
    }
    #[test]
    fn wrong_java_version_vendor_and_failed_launch_are_refused() {
        for (success, output) in [
            (false, "openjdk version \"27\"\nTemurin-27+"),
            (true, "openjdk version \"21\"\nTemurin-21+"),
            (true, "openjdk version \"27\"\nOtherVendor-27+"),
        ] {
            assert!(validate_java(success, output).is_err());
        }
        assert!(
            validate_java(
                true,
                "openjdk version \"27\" 2026-09-15\nOpenJDK Runtime Environment Temurin-27+18"
            )
            .is_ok()
        );
    }
    #[test]
    fn java_policy_is_internal_and_environment_injection_is_removed() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let mut command = java_command(Path::new("python3"));
        command.args(["-c", "import os; print(os.getenv('UMPIRE_INTERNAL_JVM_PROCESS')); print(os.getenv('JAVA_TOOL_OPTIONS'))"]);
        let (status, output) =
            run_process(&mut command, &scratch.0, "policy", Limits::default(), None)?;
        assert!(status.success());
        assert_eq!(output, "None\nNone\n");
        scratch.finish(Ok(()))
    }
}
