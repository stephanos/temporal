//! Build the framework's five wasm controls and native parity reports.
use crate::Result;
use std::{env, fs, os::unix::process::ExitStatusExt, path::Path, process::Command};
use umpire_guest_driver::context::cargo_home;

const GUESTS: &[(&str, &str)] = &[
    ("umpire-stock-guest", "umpire_stock_guest"),
    ("umpire-kettle-guest", "umpire_kettle_guest"),
    ("umpire-control-guest", "stateful_control"),
    ("umpire-caught-panic-guest", "caught_panic_control"),
    ("umpire-overflow-guest", "overflow_control"),
];

/// Run native guest-driver directly, then produce the two native comparison reports.
pub fn run(root: &Path, driver: &Path) -> Result<i32> {
    let caller = env::current_dir()?;
    let home = cargo_home(&caller)?;
    let target = env::var_os("CARGO_TARGET_DIR").map_or_else(
        || root.join("target"),
        |value| {
            let path = std::path::PathBuf::from(value);
            if path.is_absolute() { path } else { caller.join(path) }
        },
    );
    fs::create_dir_all(&target)?;
    let target = target.canonicalize()?;
    run_with(root, driver, &target, &home, |command| {
        let status = crate::gate::scheduler::execute(command, true, true)?.status;
        Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
    })
}

fn run_with(
    root: &Path,
    driver: &Path,
    target: &Path,
    home: &Path,
    mut execute: impl FnMut(&mut Command) -> Result<i32>,
) -> Result<i32> {
    let configure = |command: &mut Command| {
        command
            .current_dir(root)
            .env("CARGO_TARGET_DIR", target)
            .env("CARGO_HOME", home)
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false");
    };
    for (package, artifact) in GUESTS {
        let mut command = Command::new(driver);
        configure(&mut command);
        command
            .args(["guest", package])
            .arg(target.join("wasm32-wasip1/guest").join(format!("{artifact}.wasm")));
        let status = execute(&mut command)?;
        if status != 0 {
            return Ok(status);
        }
    }
    for (package, binary, directory) in [
        ("umpire-caught-panic-guest", "panic-control-reports", "panic-native"),
        ("umpire-control-guest", "current-contract-reports", "current-contract-native"),
    ] {
        let mut command = Command::new("mise");
        configure(&mut command);
        command
            .args([
                "exec",
                "--",
                "cargo",
                "run",
                "--locked",
                "--profile",
                "guest",
                "-p",
                package,
                "--bin",
                binary,
                "--",
            ])
            .arg(target.join(directory));
        let status = execute(&mut command)?;
        if status != 0 {
            return Ok(status);
        }
    }
    Ok(0)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn five_artifacts_then_two_reports_keep_spaces_and_child_status() -> Result<()> {
        let root = Path::new("/workspace with spaces");
        let target = root.join("target with spaces");
        let home = root.join("cargo home");
        let driver = root.join("native driver");
        let mut calls = Vec::new();
        assert_eq!(
            run_with(root, &driver, &target, &home, |command| {
                assert_eq!(command.get_current_dir(), Some(root));
                assert!(
                    command.get_envs().any(
                        |(name, value)| name == "CARGO_HOME" && value == Some(home.as_os_str())
                    )
                );
                calls.push(
                    command.get_args().map(std::ffi::OsStr::to_os_string).collect::<Vec<_>>(),
                );
                Ok(0)
            })?,
            0
        );
        assert_eq!(calls.len(), 7);
        for (index, (package, artifact)) in GUESTS.iter().enumerate() {
            assert_eq!(
                calls[index],
                [
                    "guest".into(),
                    (*package).into(),
                    target
                        .join("wasm32-wasip1/guest")
                        .join(format!("{artifact}.wasm"))
                        .into_os_string()
                ]
            );
        }
        assert_eq!(calls[5].last(), Some(&target.join("panic-native").into_os_string()));
        assert_eq!(calls[6].last(), Some(&target.join("current-contract-native").into_os_string()));
        for fail in 1..=7 {
            let mut count = 0;
            assert_eq!(
                run_with(root, &driver, &target, &home, |_| {
                    count += 1;
                    Ok(if count == fail { 37 } else { 0 })
                })?,
                37
            );
            assert_eq!(count, fail);
        }
        Ok(())
    }
}

#[cfg(test)]
#[path = "build_guest_path_controls.rs"]
mod path_controls;
