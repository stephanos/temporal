//! Compile negative controls against the framework capability policies.
use crate::{Error, Result};
use std::{
    fs,
    io::{self, Write},
    path::Path,
    process::{Command, Output},
};

const CAPABILITIES: &str = r#"#![allow(dead_code)]
use std::fs::read as aliased_read;
pub fn forbidden() {
    let _ = aliased_read("unused");
    let _ = std::time::Instant::now();
    let _ = std::thread::spawn(|| ());
    let _ = std::path::Path::new("unused").exists();
    std::thread::yield_now();
    let _ = std::fs::DirBuilder::new();
    let _: Option<std::fs::File> = None;
    let _: Option<std::time::SystemTime> = None;
    let _: Option<std::thread::JoinHandle<()>> = None;
}
"#;
const SYMBOLS: &[&str] = &[
    "std::fs::read",
    "std::time::Instant::now",
    "std::thread::spawn",
    "std::fs::File",
    "std::time::SystemTime",
    "std::thread::JoinHandle",
    "std::path::Path::exists",
    "std::thread::yield_now",
    "std::fs::DirBuilder::new",
];
const SAFE_ROOTS: &[&str] = &[
    "author/dsl/src/lib.rs",
    "dev/xtask/src/main.rs",
    "dev/guest-driver/src/main.rs",
    "backends/native/build.rs",
    "author/dsl/tests/main.rs",
    "author/dsl/tests/ui.rs",
];

fn refusal(output: &Output, symbols: &[&str], label: &str) -> Result<()> {
    let diagnostics = String::from_utf8_lossy(&output.stderr);
    let stdout = String::from_utf8_lossy(&output.stdout);
    if output.status.success()
        || symbols.iter().any(|symbol| !diagnostics.contains(symbol) && !stdout.contains(symbol))
    {
        io::stderr().write_all(&output.stdout)?;
        io::stderr().write_all(&output.stderr)?;
        return Err(Error::Invalid(format!("missing capability refusal: {label}")));
    }
    Ok(())
}

/// Reject forbidden APIs and unsafe overrides using the installed pinned compilers.
pub fn run(root: &Path) -> Result<i32> {
    run_with(root, |command| Ok(crate::gate::scheduler::execute(command, false, false)?))
}

fn run_with(root: &Path, mut execute: impl FnMut(&mut Command) -> Result<Output>) -> Result<i32> {
    let scratch = tempfile::tempdir()?;
    let source = scratch.path().join("control.rs");
    fs::write(&source, CAPABILITIES)?;
    let compiler = |tool: &str, input: &Path| {
        let mut command = Command::new("mise");
        command
            .current_dir(root)
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false")
            .args([
                "exec",
                "--",
                tool,
                "--edition=2024",
                "--crate-type=lib",
                "--emit=metadata",
                "--out-dir",
            ])
            .arg(scratch.path())
            .arg(input);
        command
    };
    for directory in ["author/dsl", "backends/native"] {
        let mut command = compiler("clippy-driver", &source);
        command.env("CLIPPY_CONF_DIR", root.join(directory)).args([
            "-D",
            "clippy::disallowed_methods",
            "-D",
            "clippy::disallowed_types",
        ]);
        refusal(&execute(&mut command)?, SYMBOLS, directory)?;
    }
    let source = scratch.path().join("unsafe.rs");
    for name in SAFE_ROOTS {
        let contents = fs::read_to_string(root.join(name))?;
        if !contents.lines().any(|line| line == "#![forbid(unsafe_code)]") {
            return Err(Error::Invalid(format!("missing forbid(unsafe_code): {name}")));
        }
        fs::write(
            &source,
            "#![forbid(unsafe_code)]\n#[allow(unsafe_code)]\npub unsafe fn forbidden() {}\n",
        )?;
        refusal(&execute(&mut compiler("rustc", &source))?, &["E0453"], name)?;
    }
    fs::write(
        &source,
        "#![deny(unsafe_code)]\n#[allow(unsafe_code)]\npub unsafe fn audited() {}\n",
    )?;
    let allowed = execute(&mut compiler("rustc", &source))?;
    if !allowed.status.success() {
        io::stderr().write_all(&allowed.stdout)?;
        io::stderr().write_all(&allowed.stderr)?;
        return Err(Error::Invalid("audited unsafe allowance was refused".into()));
    }
    fs::write(&source, "#![deny(unsafe_code)]\npub unsafe fn forbidden() {}\n")?;
    refusal(
        &execute(&mut compiler("rustc", &source))?,
        &["declaration of an `unsafe` function"],
        "unaudited unsafe",
    )?;
    Ok(0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::process::ExitStatusExt;
    use std::process::ExitStatus;

    fn output(success: bool, diagnostics: &str) -> Output {
        Output {
            status: ExitStatus::from_raw(if success { 0 } else { 256 }),
            stdout: Vec::new(),
            stderr: diagnostics.as_bytes().to_vec(),
        }
    }

    #[test]
    fn negative_control_requires_failure_and_every_symbol() {
        assert!(refusal(&output(true, "api"), &["api"], "policy").is_err());
        assert!(refusal(&output(false, "api"), &["api", "type"], "policy").is_err());
        assert!(refusal(&output(false, "api type"), &["api", "type"], "policy").is_ok());
    }

    #[test]
    fn both_policies_six_safe_roots_and_audited_exception() -> Result<()> {
        let root = tempfile::tempdir()?;
        for name in SAFE_ROOTS {
            let path = root.path().join(name);
            fs::create_dir_all(
                path.parent().ok_or_else(|| Error::Invalid("missing parent".into()))?,
            )?;
            fs::write(path, "#![forbid(unsafe_code)]\n")?;
        }
        let mut calls = 0;
        let mut scratch = None;
        assert_eq!(
            run_with(root.path(), |command| {
                assert_eq!(command.get_current_dir(), Some(root.path()));
                let arguments: Vec<_> = command.get_args().collect();
                let out = arguments
                    .iter()
                    .position(|value| *value == "--out-dir")
                    .ok_or_else(|| Error::Invalid("missing output".into()))?;
                scratch = Some(std::path::PathBuf::from(arguments[out + 1]));
                calls += 1;
                Ok(match calls {
                    1 | 2 => output(false, &SYMBOLS.join("\n")),
                    3..=8 => output(false, "E0453"),
                    9 => output(true, ""),
                    _ => output(false, "declaration of an `unsafe` function"),
                })
            })?,
            0
        );
        assert_eq!(calls, 10);
        assert!(!scratch.ok_or_else(|| Error::Invalid("missing scratch".into()))?.exists());
        Ok(())
    }
}
