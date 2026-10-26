//! Framework tests across binaries, followed by Cargo's workspace doctests.
use std::{os::unix::process::ExitStatusExt, process::Command};

/// Run workspace tests with the caller's admitted gate environment and working directory.
pub fn run() -> crate::Result<i32> {
    run_with(|arguments| {
        let status = Command::new("mise").args(["exec", "--", "cargo"]).args(arguments).status()?;
        Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
    })
}

fn run_with(mut execute: impl FnMut(&[&str]) -> crate::Result<i32>) -> crate::Result<i32> {
    let status = execute(&[
        "nextest",
        "run",
        "--workspace",
        "--locked",
        "--fail-fast",
        "--run-ignored",
        "default",
        "--status-level",
        "slow",
        "--final-status-level",
        "slow",
        "--success-output",
        "never",
        "--failure-output",
        "immediate-final",
    ])?;
    if status != 0 {
        return Ok(status);
    }
    execute(&["test", "--doc", "--workspace", "--locked", "--quiet"])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn binaries_then_doctests_exactly_once() -> crate::Result<()> {
        let mut calls = Vec::new();
        assert_eq!(
            run_with(|args| {
                calls.push(args.join(" "));
                Ok(0)
            })?,
            0
        );
        assert_eq!(
            calls,
            [
                "nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final",
                "test --doc --workspace --locked --quiet",
            ]
        );
        Ok(())
    }

    #[test]
    fn binary_failure_stops_before_doctests() -> crate::Result<()> {
        let mut calls = 0;
        assert_eq!(
            run_with(|_| {
                calls += 1;
                Ok(42)
            })?,
            42
        );
        assert_eq!(calls, 1);
        Ok(())
    }

    #[test]
    fn doctest_failure_preserves_status() -> crate::Result<()> {
        let mut calls = 0;
        assert_eq!(
            run_with(|_| {
                calls += 1;
                Ok(if calls == 1 { 0 } else { 37 })
            })?,
            37
        );
        assert_eq!(calls, 2);
        Ok(())
    }

    #[test]
    fn missing_runner_fails_without_fallback() {
        let mut calls = 0;
        let result = run_with(|_| {
            calls += 1;
            Err(std::io::Error::new(std::io::ErrorKind::NotFound, "missing runner").into())
        });
        assert!(result.is_err());
        assert_eq!(calls, 1);
    }
}
