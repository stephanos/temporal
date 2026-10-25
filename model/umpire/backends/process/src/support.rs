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
    let mut child = crate::syscall::Running::spawn(command, directory, &out, &err)
        .map_err(|error| Refusal(format!("cannot spawn {label}: {error}")))?;
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
#[path = "tests.rs"]
mod tests;
