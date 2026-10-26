//! Temporal tests across binaries and doctests, owned by the gate.
use std::{env, io, path::Path, process::Command};

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    run_with(|arguments| {
        let mut command = crate::gate::cargo(arguments);
        configure(
            &mut command,
            env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require"),
            env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
        );
        crate::gate::stage(root, "tests", command)
    })
}

pub(crate) fn test_threads(apalache: bool) -> usize {
    // Optional mixed-backend allocation still needs its real resource proof.
    if apalache { 1 } else { 4 }
}

pub(crate) fn configure(command: &mut Command, required: bool, apalache: bool) {
    if required || apalache {
        command
            .env("NEXTEST_PROFILE", "required")
            .env("NEXTEST_TEST_THREADS", test_threads(apalache).to_string());
    }
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
    #[allow(clippy::unwrap_used)]
    fn required_backend_policy_caps_inherited_threads_and_preserves_skipped_defaults() {
        for (required, apalache, threads) in
            [(true, false, "4"), (false, true, "1"), (true, true, "1")]
        {
            let mut command = Command::new("/bin/sh");
            command.env("NEXTEST_PROFILE", "default").env("NEXTEST_TEST_THREADS", "99");
            configure(&mut command, required, apalache);
            let output = command
                .args(["-c", "printf '%s:%s' \"$NEXTEST_PROFILE\" \"$NEXTEST_TEST_THREADS\""])
                .output()
                .unwrap();
            assert!(output.status.success());
            assert_eq!(String::from_utf8(output.stdout).unwrap(), format!("required:{threads}"));
        }
        let mut command = Command::new("/bin/sh");
        configure(&mut command, false, false);
        assert!(
            !command
                .get_envs()
                .any(|(key, _)| key == "NEXTEST_PROFILE" || key == "NEXTEST_TEST_THREADS")
        );
    }

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
