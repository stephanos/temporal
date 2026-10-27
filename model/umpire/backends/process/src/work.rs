//! Invocation-scoped observations from native producers.
use std::{
    env, fs,
    io::{self, Write},
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};

static NEXT: AtomicU64 = AtomicU64::new(0);
/// Maximum completed producer events in one stage.
pub const MAX_EVENTS: u64 = 100_000;
/// Maximum bytes in one producer observation.
pub const MAX_EVENT_BYTES: usize = 4096;

/// Resolve only the current stage's already-created observation directory.
pub fn directory() -> io::Result<Option<PathBuf>> {
    let Some(root) = env::var_os("UMPIRE_GATE_RECEIPTS") else { return Ok(None) };
    let Some(stage) = env::var_os("UMPIRE_GATE_STAGE") else {
        return Err(io::Error::other("work observation scope lacks stage"));
    };
    let stage = stage
        .to_str()
        .filter(|stage| {
            !stage.is_empty()
                && stage.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
        })
        .ok_or_else(|| io::Error::other("invalid work observation stage"))?;
    let root = PathBuf::from(root);
    if !root.is_absolute() {
        return Err(io::Error::other("work observations need an absolute invocation directory"));
    }
    let scope = env::var("UMPIRE_GATE_SCOPE").unwrap_or_default();
    if !scope.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-') {
        return Err(io::Error::other("invalid work observation scope"));
    }
    let directory = root.join("work").join(scope).join(stage);
    let metadata = fs::symlink_metadata(&directory)?;
    if !metadata.is_dir() || metadata.is_symlink() {
        return Err(io::Error::other("invalid work observation directory"));
    }
    Ok(Some(directory))
}

/// Publish one bounded event without appending to another process's file.
/// Missing scope disables observations; malformed active scope refuses execution.
pub fn record(kind: &str, values: &str) -> io::Result<()> {
    let Some(directory) = directory()? else { return Ok(()) };
    if kind.is_empty()
        || !kind.bytes().all(|byte| byte.is_ascii_lowercase() || byte == b'-')
        || values.bytes().any(|byte| byte == b'\n' || byte == b'\r')
    {
        return Err(io::Error::other("invalid work observation record"));
    }
    let record = format!("1\t{kind}\t{values}\n");
    if record.len() > MAX_EVENT_BYTES {
        return Err(io::Error::other("work observation exceeds byte limit"));
    }
    // Slot names bound all concurrent producers together; PID reuse cannot overwrite evidence.
    for slot in NEXT.fetch_add(1, Ordering::Relaxed)..MAX_EVENTS {
        let claim = directory.join(format!("{slot}.claim"));
        match fs::OpenOptions::new().write(true).create_new(true).open(claim) {
            Ok(_) => {
                NEXT.fetch_max(slot + 1, Ordering::Relaxed);
                let partial = directory.join(format!("{slot}.partial"));
                let mut file =
                    fs::OpenOptions::new().write(true).create_new(true).open(&partial)?;
                file.write_all(record.as_bytes())?;
                file.sync_all()?;
                fs::hard_link(&partial, directory.join(format!("{slot}.event")))?;
                fs::remove_file(partial)?;
                return Ok(());
            }
            Err(error) if error.kind() == io::ErrorKind::AlreadyExists => (),
            Err(error) => return Err(error),
        }
    }
    Err(io::Error::other("work observation event limit exceeded"))
}

/// A supervisor fork is an attempt, not proof that the checker executable started.
pub(crate) fn spawned(command: &std::process::Command) -> io::Result<Option<String>> {
    if !command.get_args().any(|argument| argument == "tlc2.TLC") || directory()?.is_none() {
        return Ok(None);
    }
    let configuration = command
        .get_envs()
        .find_map(|(name, value)| (name == "UMPIRE_TLC_CONFIGURATION").then_some(value).flatten());
    let configuration = configuration.and_then(std::ffi::OsStr::to_str).unwrap_or_else(|| {
        if command.get_args().any(|argument| argument == "-h") { "probe" } else { "-" }
    });
    if !["-", "probe"].contains(&configuration)
        && (configuration.len() != 64
            || !configuration.bytes().all(|byte| byte.is_ascii_hexdigit()))
    {
        return Err(io::Error::other("invalid TLC configuration observation"));
    }
    let id = format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
    record("tlc-attempt", &format!("{id}\t{configuration}"))?;
    Ok(Some(id))
}

/// Confirm execution from the admitted checker's own startup output.
pub(crate) fn finished(id: Option<&str>, output: &str) -> io::Result<()> {
    if let Some(id) = id {
        let started = output.contains("TLC2 Version ")
            || output.contains(
                "TLC - provides model checking and simulation of TLA+ specifications - Version ",
            );
        record("tlc-finish", &format!("{id}\t{}", u8::from(started)))?;
    }
    Ok(())
}

/// Begin one actual test-run command; completion must use the same producer identity.
pub fn begin_tests() -> io::Result<String> {
    let id = format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
    record("tests-begin", &id)?;
    Ok(id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scope_probe_child() -> io::Result<()> {
        let Ok(mode) = env::var("WORK_SCOPE_PROBE") else { return Ok(()) };
        let mut command = std::process::Command::new("java");
        if mode == "tlc" {
            command.arg("tlc2.TLC");
        }
        let result = spawned(&command);
        if mode == "tlc" {
            assert!(result.is_err());
        } else {
            assert_eq!(result?, None);
        }
        Ok(())
    }

    #[test]
    fn unrelated_processes_do_not_validate_tlc_observation_scope() -> io::Result<()> {
        for mode in ["other", "tlc"] {
            let status = std::process::Command::new(env::current_exe()?)
                .args(["--exact", "work::tests::scope_probe_child", "--nocapture"])
                .env("WORK_SCOPE_PROBE", mode)
                .env("UMPIRE_GATE_RECEIPTS", "relative-invalid-scope")
                .env("UMPIRE_GATE_STAGE", "tests")
                .status()?;
            assert!(status.success());
        }
        Ok(())
    }
}
