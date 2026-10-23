use super::collection::FailureReasons;
use serde::Serialize;
use serde_json::Value;
use std::path::PathBuf;

#[derive(Clone, Serialize)]
pub(crate) struct Attempt {
    pub(crate) label: String,
    pub(crate) role: String,
    pub(crate) native_root: PathBuf,
}
#[derive(Serialize)]
pub(crate) struct Record {
    pub(crate) attempt: Attempt,
    pub(crate) retained_paths: Vec<PathBuf>,
    pub(crate) observation: Option<Value>,
    pub(crate) execution_failure: Option<String>,
    pub(crate) verification_failure: Option<String>,
}
fn retained_paths(attempt: &Attempt) -> Result<Vec<PathBuf>, String> {
    let mut paths = Vec::new();
    for name in [
        "request.json",
        "report.json",
        "outcome.json",
        "profile.json",
        "rootmodules.json",
        "streams.json",
        "stdout",
        "stderr",
        "transcript",
        "choice",
        "network",
    ] {
        paths.push(attempt.native_root.join(name));
    }
    if let Some(parent) = attempt.native_root.parent() {
        for suffix in ["json", "stdout", "stderr"] {
            paths.push(parent.join(format!("{}-driver.{suffix}", attempt.label)));
        }
    }
    let mut retained = Vec::new();
    for path in paths {
        match std::fs::symlink_metadata(&path) {
            Ok(info) if info.is_file() => retained.push(path),
            Ok(_) => return Err("terminal raw observation path is not a regular file".into()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => return Err(error.to_string()),
        }
    }
    Ok(retained)
}

/// # Errors
/// Collects every admitted attempt, then joins execution, verification and publication failures.
pub(crate) fn collect<T>(
    attempts: &[Attempt],
    mut execute: impl FnMut(&Attempt) -> Result<Value, String>,
    mut verify: impl FnMut(&Attempt) -> Result<(), String>,
    publish: impl FnOnce(&[Record], Option<&str>) -> Result<T, String>,
) -> Result<T, String> {
    let mut failures = FailureReasons::default();
    let mut observations = Vec::new();
    for attempt in attempts {
        let executed = execute(attempt);
        let verified = verify(attempt);
        if let Err(error) = &executed {
            failures.push(format!("{}: {error}", attempt.label));
        }
        if let Err(error) = &verified {
            failures.push(format!(
                "{} identity/source verification: {error}",
                attempt.label
            ));
        }
        let paths = match retained_paths(attempt) {
            Ok(paths) => paths,
            Err(error) => {
                failures.push(format!("{} raw retention: {error}", attempt.label));
                Vec::new()
            }
        };
        let execution_failure = executed.as_ref().err().cloned();
        let verification_failure = verified.as_ref().err().cloned();
        observations.push(Record {
            attempt: attempt.clone(),
            retained_paths: paths,
            observation: executed.ok().filter(|_| verified.is_ok()),
            execution_failure,
            verification_failure,
        });
    }
    let result = failures.publish(|failure| publish(&observations, failure))?;
    if let Some(failure) = failures.message() {
        Err(failure)
    } else {
        Ok(result)
    }
}

#[cfg(test)]
#[path = "terminal_collection_tests.rs"]
mod tests;
