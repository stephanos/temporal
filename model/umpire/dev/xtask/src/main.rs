//! Native development command dispatcher.
#![forbid(unsafe_code)]
use std::{
    env,
    ffi::{OsStr, OsString},
    io::{self, Write},
    os::unix::process::{CommandExt, ExitStatusExt},
    path::Path,
    process::Command,
};
use umpire_guest_driver::context::{ToolRoot, cargo_home};

fn guest(arguments: Vec<OsString>) -> xtask::Result<i32> {
    let tools = ToolRoot::local()?;
    let cargo_home = cargo_home(&env::current_dir()?)?;
    let executable = env::current_exe()?;
    let directory = executable
        .parent()
        .ok_or_else(|| xtask::Error::Invalid("xtask executable needs parent".into()))?;
    let target = directory
        .parent()
        .ok_or_else(|| xtask::Error::Invalid("xtask executable needs target directory".into()))?;
    let mut build = Command::new("mise");
    build
        .args(["exec", "--", "cargo", "build", "--locked", "--manifest-path"])
        .arg(tools.path().join("Cargo.toml"))
        .args([
            "-p",
            "umpire-guest-driver",
            "--bin",
            "umpire-guest-driver",
            "--profile",
            "dev",
            "--target-dir",
        ])
        .arg(target)
        .current_dir(tools.path())
        .env("CARGO_HOME", &cargo_home)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    for key in [
        "RUSTC_WRAPPER",
        "RUSTC_WORKSPACE_WRAPPER",
        "RUSTFLAGS",
        "CARGO_ENCODED_RUSTFLAGS",
        "RUSTC_BOOTSTRAP",
        "CARGO_BUILD_TARGET",
    ] {
        build.env_remove(key);
    }
    let status = build.status()?;
    if !status.success() {
        return Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)));
    }
    Err(Command::new(directory.join("umpire-guest-driver"))
        .args(arguments)
        .env("CARGO_HOME", cargo_home)
        .exec()
        .into())
}

fn run() -> xtask::Result<i32> {
    let mut supplied: Vec<_> = env::args_os().skip(1).collect();
    let _lease = if supplied.first().is_some_and(|arg| arg == "__owned-command") {
        supplied.remove(0);
        if env::var_os("UMPIRE_JVM_LEASE_FD").is_none() {
            return Err(xtask::Error::Invalid("missing owned-command descriptor".into()));
        }
        Some(xtask::gate::lease::JvmLease::inherited_or_acquire()?)
    } else if supplied.first().is_some_and(|arg| {
        ["gate", "validate", "time", "update", "fmt", "quick", "test-run"]
            .iter()
            .any(|name| arg == name)
    }) && !supplied.iter().any(|arg| arg == "--list")
    {
        return xtask::gate::supervise(supplied);
    } else {
        None
    };
    let mut arguments = supplied.into_iter();
    match arguments.next().as_deref() {
        Some(command) if command == OsStr::new("quick") => xtask::quick::run(arguments.collect()),
        Some(command)
            if ["gate", "fmt", "update", "time", "validate"]
                .iter()
                .any(|name| command == OsStr::new(name)) =>
        {
            xtask::gate::run(
                command.to_str().ok_or_else(|| xtask::Error::Invalid("command must be UTF-8".into()))?,
                arguments.collect(),
            )
        }
        Some(command) if command == OsStr::new("test-run") => {
            let args: Vec<_> = arguments.collect();
            match args.as_slice() {
                [] => xtask::test_run::run(),
                [flag, package] if flag == "--package" => xtask::test_run::run_package(
                    package.to_str().ok_or_else(|| xtask::Error::Invalid("package must be UTF-8".into()))?),
                _ => Err(xtask::Error::Invalid("usage: cargo xtask test-run [--package NAME]".into())),
            }
        }
        Some(command) if command == OsStr::new("fetch-apalache") => {
            let root = arguments.next();
            if arguments.next().is_some() {
                return Err(xtask::Error::Invalid(
                    "usage: cargo xtask fetch-apalache [framework-root]".into(),
                ));
            }
            let tools = if let Some(root) = root {
                ToolRoot::admit(Path::new(&root))?
            } else {
                ToolRoot::local()?
            };
            let path = xtask::fetch_apalache::install(tools.path())?;
            io::stdout().write_all(path.as_os_str().as_encoded_bytes())?;
            io::stdout().write_all(b"\n")?;
            Ok(0)
        }
        Some(command)
            if ["guest", "guest-freshness", "guest-lock"]
                .iter()
                .any(|name| command == OsStr::new(name)) =>
        {
            guest(std::iter::once(command.to_owned()).chain(arguments).collect())
        }
        Some(command) if command == OsStr::new("retired-names") => {
            let pattern = arguments.next();
            if pattern.is_none() || arguments.next().is_some() {
                return Ok(2);
            }
            Ok(xtask::retired_names::run(
                pattern.as_deref().unwrap_or_default(),
                io::stdin().lock(),
                io::stdout().lock(),
            ))
        }
        _ => Err(xtask::Error::Invalid(
            "commands: gate, quick, fmt, update, time, validate, test-run, fetch-apalache, guest, guest-freshness, guest-lock, retired-names".into(),
        )),
    }
}
fn main() {
    let status = match run() {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "xtask: {error}");
            1
        }
    };
    std::process::exit(status);
}
