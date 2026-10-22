use gomad_wasmhost::linked;
use std::{fs::OpenOptions, io::Write, os::unix::fs::OpenOptionsExt as _};

pub(crate) fn before_execution(
    seed: u64,
    cancellation: &linked::Cancellation,
) -> Result<(), String> {
    if std::env::var("GOMAD_FIXTURE_BACKEND_CANCEL").as_deref() == Ok("observed-seed-8")
        && seed == 8
    {
        cancellation.cancel();
        return Err("cancelled: fixture held observed seed 8".into());
    }
    Ok(())
}

pub(crate) fn after_execution(
    ordinal: u64,
    cancellation: &linked::Cancellation,
) -> Result<(), String> {
    if let Some(path) = std::env::var_os("GOMAD_FIXTURE_BACKEND_EXECUTIONS") {
        let mut file = OpenOptions::new().append(true).create(true).mode(0o600)
            .open(path).map_err(|error| error.to_string())?;
        writeln!(file, "{ordinal}").map_err(|error| error.to_string())?;
    }
    if std::env::var("GOMAD_FIXTURE_BACKEND_CANCEL").as_deref() == Ok("choice-second")
        && ordinal == 1
    {
        cancellation.cancel();
        return Err("cancelled: fixture interrupted inside second round".into());
    }
    Ok(())
}

pub(crate) fn minimize_control(decisions: u64) -> Result<(), String> {
    if let Some(path) = std::env::var_os("GOMAD_FIXTURE_MINIMIZE_CONTROLS") {
        let mut file = OpenOptions::new()
            .append(true)
            .create(true)
            .mode(0o600)
            .open(path)
            .map_err(|error| error.to_string())?;
        writeln!(file, "{decisions}").map_err(|error| error.to_string())?;
    }
    Ok(())
}

pub(crate) fn minimize_checkpoint(phase: &str) -> Result<(), String> {
    if std::env::var("GOMAD_FIXTURE_MINIMIZE_INTERRUPT").as_deref() == Ok(phase) {
        return Err(format!("runner_failure: minimization fixture interrupted {phase}"));
    }
    Ok(())
}

pub(crate) fn progress_acknowledged() -> Result<(), String> {
    static ORDINAL: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
    let Some(directory) = std::env::var_os("GOMAD_FIXTURE_SOFT_REPORT_ACK") else {
        return Ok(());
    };
    let ordinal = ORDINAL.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    let path = std::path::Path::new(&directory).join(ordinal.to_string());
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(10);
    loop {
        if path.is_file() {
            return Ok(());
        }
        if std::time::Instant::now() >= deadline {
            return Err("runner_failure: soft assertion progress acknowledgment timed out".into());
        }
        std::thread::sleep(std::time::Duration::from_millis(5));
    }
}
