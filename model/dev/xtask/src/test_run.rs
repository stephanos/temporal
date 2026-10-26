//! Temporal tests across binaries and doctests, owned by the gate.
use std::{io, path::Path};

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    run_with(|arguments| crate::gate::stage(root, "tests", crate::gate::cargo(arguments)))
}

fn run_with(mut execute: impl FnMut(&[&str]) -> io::Result<i32>) -> io::Result<i32> {
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
    fn binaries_then_doctests_exactly_once() -> io::Result<()> {
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
    fn binary_failure_stops_before_doctests() -> io::Result<()> {
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
    fn doctest_failure_preserves_status() -> io::Result<()> {
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
            Err(io::Error::new(io::ErrorKind::NotFound, "missing runner"))
        });
        assert!(result.is_err());
        assert_eq!(calls, 1);
    }
}
