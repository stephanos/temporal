//! Shared bounded subprocess and workspace scratch support for lifted backends.
#![allow(missing_docs)]
use crate::{Limits, Refusal};
use std::{
    fs::{self, File},
    io::Read,
    path::{Path, PathBuf},
    process::{Child, Command, ExitStatus, Stdio},
    sync::atomic::{AtomicU64, Ordering},
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
        let parent = root.join(relative);
        let mut existing = parent.as_path();
        while !existing.exists() {
            existing =
                existing.parent().ok_or_else(|| Refusal("no existing scratch ancestor".into()))?;
        }
        if !existing.canonicalize()?.starts_with(root) {
            return Err(Refusal("TLC scratch directory escapes the repository".into()));
        }
        fs::create_dir_all(&parent)?;
        let parent = parent.canonicalize()?;
        if !parent.starts_with(root) {
            return Err(Refusal("TLC scratch directory escapes the repository".into()));
        }
        loop {
            let path = parent.join(format!(
                "run-{}-{}",
                std::process::id(),
                NEXT_RUN.fetch_add(1, Ordering::Relaxed)
            ));
            match fs::create_dir(&path) {
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

struct Running(Child);

impl Drop for Running {
    fn drop(&mut self) {
        let _ = self.0.kill();
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
    let stdout = directory.join(format!("{label}.stdout"));
    let stderr = directory.join(format!("{label}.stderr"));
    command.stdin(Stdio::null()).stdout(File::create(&stdout)?).stderr(File::create(&stderr)?);
    let mut child = Running(
        command.spawn().map_err(|error| Refusal(format!("cannot spawn {label}: {error}")))?,
    );
    let start = Instant::now();
    let status = loop {
        let bytes = fs::metadata(&stdout)?.len().saturating_add(fs::metadata(&stderr)?.len());
        if start.elapsed() >= limits.timeout
            || bytes > limits.output_bytes
            || dump.is_some_and(|path| {
                fs::metadata(path).is_ok_and(|meta| meta.len() > limits.dump_bytes)
            })
        {
            return Err(Refusal(format!(
                "subprocess limit exceeded ({label}); wall={:?}, output={bytes}",
                start.elapsed()
            )));
        }
        if let Some(status) = child.0.try_wait()? {
            break status;
        }
        thread::sleep(Duration::from_millis(5));
    };
    let output = format!(
        "{}{}",
        bounded_text(&stdout, limits.output_bytes)?,
        bounded_text(&stderr, limits.output_bytes)?
    );
    if u64::try_from(output.len()).unwrap_or(u64::MAX) > limits.output_bytes {
        return Err(Refusal(format!("subprocess output limit exceeded ({label})")));
    }
    if dump.is_some_and(|dump| fs::metadata(dump).is_ok_and(|meta| meta.len() > limits.dump_bytes))
    {
        return Err(Refusal(format!("subprocess dump limit exceeded ({label})")));
    }
    Ok((status, output))
}

pub(crate) fn bounded_text(path: &Path, limit: u64) -> Result<String, Refusal> {
    let mut text = String::new();
    File::open(path)?.take(limit.saturating_add(1)).read_to_string(&mut text)?;
    if u64::try_from(text.len()).unwrap_or(u64::MAX) > limit {
        return Err(Refusal(format!("file exceeds bounded read limit: {}", path.display())));
    }
    Ok(text)
}
