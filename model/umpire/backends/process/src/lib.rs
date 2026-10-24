//! Shared bounded subprocess and workspace scratch support for lifted backends.
#![allow(missing_docs)]
use std::{
    fs::{self, File},
    io::{Read, Seek},
    os::unix::{
        fs::{DirBuilderExt, MetadataExt},
        process::CommandExt,
    },
    path::{Path, PathBuf},
    process::{Child, Command, ExitStatus, Stdio},
    sync::atomic::{AtomicBool, AtomicU64, Ordering},
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

    pub fn finish<T, E: std::fmt::Display + From<Refusal>>(
        self,
        result: Result<T, E>,
    ) -> Result<T, E> {
        match fs::remove_dir_all(&self.0) {
            Ok(()) => result,
            Err(error) => Err(Refusal(format!(
                "TLC cleanup failed: {error}; result: {}",
                result.err().map_or_else(|| "completed".into(), |error| error.to_string())
            ))
            .into()),
        }
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
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
    let stdout = directory.join(format!("{label}.stdout"));
    let stderr = directory.join(format!("{label}.stderr"));
    let open =
        |path: &Path| fs::OpenOptions::new().create_new(true).read(true).write(true).open(path);
    let mut out = open(&stdout)?;
    let mut err = open(&stderr)?;
    if cancelled.load(Ordering::Relaxed) {
        return Err(Refusal(format!("subprocess cancelled ({label})")).into());
    }
    let mut supervised = Command::new("python3");
    supervised.arg("-c").arg(include_str!("process.py")).arg(command.get_program());
    supervised.args(command.get_args());
    supervised.current_dir(command.get_current_dir().unwrap_or(directory));
    for (key, value) in command.get_envs() {
        if let Some(value) = value {
            supervised.env(key, value);
        } else {
            supervised.env_remove(key);
        }
    }
    supervised
        .process_group(0)
        .stdin(Stdio::piped())
        .stdout(Stdio::from(out.try_clone()?))
        .stderr(Stdio::from(err.try_clone()?));
    let mut child =
        Running(supervised.spawn().map_err(|e| Refusal(format!("cannot spawn {label}: {e}")))?);
    let start = Instant::now();
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
        if let Some(status) = child.0.try_wait()? {
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
    let output = match String::from_utf8(bytes) {
        Ok(output) => output,
        Err(error) => {
            let mut output = String::from_utf8_lossy(error.as_bytes()).into_owned();
            let limit = usize::try_from(limits.output_bytes).unwrap_or(usize::MAX);
            output.truncate(output.floor_char_boundary(limit));
            return Err(ProcessFailure {
                refusal: Refusal(format!("non-UTF8 subprocess output: {error}")),
                output,
            });
        }
    };
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
    match result {
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
mod tests {
    use super::*;

    #[test]
    fn direct_helper_timeout_retains_output_and_cleans_scratch() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let mut command = Command::new("python3");
        command.args(["-c", "import sys,time; print('started', flush=True); time.sleep(10)"]);
        let failure = run_process_cancelled(
            &mut command,
            &scratch.0,
            "direct",
            Limits { timeout: Duration::from_secs(1), ..Limits::default() },
            None,
            &AtomicBool::new(false),
        )
        .err()
        .ok_or_else(|| Refusal("direct helper exceeded timeout".into()))?;
        assert!(failure.refusal.0.contains("limit"));
        assert_eq!(failure.output, "started\n");
        let path = scratch.0.clone();
        scratch.finish::<_, Refusal>(Ok(()))?;
        assert!(!path.exists());
        Ok(())
    }

    #[test]
    #[cfg(target_os = "linux")]
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
        use std::sync::Arc;
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
        assert!(result.is_err_and(|e| e.refusal.0.contains("cancelled")));
        let pids = fs::read_to_string(pids)?;
        for pid in pids.split_whitespace() {
            assert!(!Path::new("/proc").join(pid).exists(), "owned pid {pid} was not reaped");
        }
        let path = scratch.0.clone();
        scratch.finish::<_, Refusal>(Ok(()))?;
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
    fn java_environment_is_clean_and_supervisor_group_is_private() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let mut command = java_command(Path::new("python3"));
        command.args(["-c", "import os; print(os.getenv('JAVA_TOOL_OPTIONS')); print(os.getpgid(os.getppid()) == os.getppid())"]);
        let (status, output) =
            run_process(&mut command, &scratch.0, "policy", Limits::default(), None)?;
        assert!(status.success());
        assert_eq!(output, "None\nTrue\n");
        scratch.finish(Ok(()))
    }
    #[test]
    fn substituted_log_symlink_cannot_modify_an_external_file() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let work = Scratch::new(&scratch.0)?;
        let protected = scratch.0.join("protected");
        fs::write(&protected, "unchanged")?;
        let mut command = Command::new("python3");
        command.args(["-c", "import os,sys; os.unlink('capture.stdout'); os.symlink(sys.argv[1], 'capture.stdout'); print('captured')"]).arg(&protected);
        let result = run_process(&mut command, &work.0, "capture", Limits::default(), None);
        assert_eq!(fs::read_to_string(&protected)?, "unchanged");
        assert!(result.is_err());
        work.finish::<_, Refusal>(Ok(()))?;
        scratch.finish(Ok(()))
    }

    #[test]
    fn substituted_log_fifo_returns_without_waiting_for_a_reader() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        let work = scratch.0.clone();
        let (send, receive) = std::sync::mpsc::channel();
        let worker = thread::spawn(move || {
            let mut command = Command::new("python3");
            command.args(["-c", "import os; os.unlink('capture.stdout'); os.mkfifo('capture.stdout'); print('captured')"]);
            let result = run_process(&mut command, &work, "capture", Limits::default(), None);
            let _ = send.send(result);
        });
        let result = receive.recv_timeout(Duration::from_secs(2));
        let blocked = result.is_err();
        let unblock = if blocked {
            Some(
                fs::OpenOptions::new()
                    .read(true)
                    .write(true)
                    .open(scratch.0.join("capture.stdout"))?,
            )
        } else {
            None
        };
        let result = result
            .or_else(|_| receive.recv_timeout(Duration::from_secs(2)))
            .map_err(|e| Refusal(e.to_string()))?;
        worker.join().map_err(|_| Refusal("capture thread panicked".into()))?;
        drop(unblock);
        assert!(!blocked, "log publication waited for a FIFO reader");
        assert!(result.is_err());
        scratch.finish(Ok(()))
    }

    #[test]
    fn failed_captures_keep_one_combined_budget_and_refuse_invalid_utf8() -> Result<(), Refusal> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
        let scratch = Scratch::new(&root)?;
        for (label, script, limit, expected) in [
            (
                "combined",
                "import sys; sys.stdout.write('abcd'); sys.stderr.write('efgh')",
                6,
                "abcdef",
            ),
            ("utf8", "import sys; sys.stdout.buffer.write(bytes([255]))", 1, ""),
        ] {
            let mut command = Command::new("python3");
            command.args(["-c", script]);
            let limits = Limits { output_bytes: limit, ..Limits::default() };
            let failure = run_process_cancelled(
                &mut command,
                &scratch.0,
                label,
                limits,
                None,
                &AtomicBool::new(false),
            )
            .err()
            .ok_or_else(|| Refusal("malformed or oversized output admitted".into()))?;
            assert!(failure.output.len() as u64 <= limit);
            assert_eq!(failure.output, expected);
            if label == "utf8" {
                assert!(failure.refusal.0.contains("non-UTF8"));
                assert_eq!(fs::read(scratch.0.join("utf8.stdout"))?, [255]);
            }
        }
        scratch.finish(Ok(()))
    }
}
