//! Native development command dispatcher.
#![forbid(unsafe_code)]
use std::{
    env,
    ffi::{OsStr, OsString},
    io::{self, Write},
    os::unix::process::{CommandExt, ExitStatusExt},
    path::{Path, PathBuf},
    process::Command,
};
use umpire_guest_driver::context::{ToolRoot, cargo_home};

fn guest_driver() -> xtask::Result<(PathBuf, std::process::ExitStatus)> {
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
    let status = xtask::gate::scheduler::execute(&build, true, true)?.status;
    Ok((directory.join("umpire-guest-driver"), status))
}

fn guest(arguments: Vec<OsString>) -> xtask::Result<i32> {
    let (driver, status) = guest_driver()?;
    if !status.success() {
        return Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)));
    }
    Err(Command::new(driver)
        .args(arguments)
        .env("CARGO_HOME", cargo_home(&env::current_dir()?)?)
        .exec()
        .into())
}

fn native_leaf(
    command: &OsStr,
    mut arguments: impl Iterator<Item = OsString>,
) -> xtask::Result<i32> {
    if arguments.next().is_some() {
        return Err(xtask::Error::Invalid(format!(
            "usage: cargo xtask {}",
            command.to_string_lossy()
        )));
    }
    match command.to_str() {
        Some("proto-tools") => xtask::proto::tools(ToolRoot::local()?.path()),
        Some("proto-controls") => xtask::proto::controls::run(),
        Some("capability-controls") => xtask::capability_controls::run(ToolRoot::local()?.path()),
        Some("timing-controls") => Ok(xtask::timing_controls::run(&env::current_exe()?)?),
        Some("build-guests") => {
            let (driver, status) = guest_driver()?;
            if !status.success() {
                return Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)));
            }
            xtask::build_guests::run(ToolRoot::local()?.path(), &driver)
        }
        _ => unreachable!(),
    }
}

fn time(arguments: Vec<OsString>) -> xtask::Result<i32> {
    if arguments.first().is_some_and(|argument| argument == "--step") {
        if arguments.len() < 3 {
            return Err(xtask::Error::Invalid(
                "usage: cargo xtask time --step NAME PROGRAM [ARG...]".into(),
            ));
        }
        let name = arguments[1]
            .to_str()
            .ok_or_else(|| xtask::Error::Invalid("timing step name must be UTF-8".into()))?;
        let mut child = Command::new(&arguments[2]);
        child.args(&arguments[3..]);
        Ok(xtask::gate::timing::execute(name, &child, true, None)?)
    } else {
        xtask::gate::run("time", arguments)
    }
}

fn run() -> xtask::Result<i32> {
    xtask::gate::work_counts::owns_controller()?;
    let mut supplied: Vec<_> = env::args_os().skip(1).collect();
    let _lease = if supplied.first().is_some_and(|arg| arg == "__owned-command") {
        supplied.remove(0);
        if env::var_os("UMPIRE_JVM_LEASE_FD").is_none() {
            return Err(xtask::Error::Invalid("missing owned-command descriptor".into()));
        }
        Some(xtask::gate::lease::JvmLease::inherited_or_acquire()?)
    } else if supplied.first().is_some_and(|arg| {
        ["gate", "validate", "time", "update", "quick", "test-run"].iter().any(|name| arg == name)
    }) && !supplied.iter().any(|arg| arg == "--list")
    {
        return xtask::gate::supervise(supplied);
    } else {
        None
    };
    let mut arguments = supplied.into_iter();
    match arguments.next().as_deref() {
        Some(command) if command == OsStr::new("quick") => xtask::quick::run(arguments.collect()),
        Some(command) if command == OsStr::new("time") => {
            time(arguments.collect())
        }
        Some(command)
            if ["gate", "fmt", "update", "validate"]
                .iter()
                .any(|name| command == OsStr::new(name)) =>
        {
            xtask::gate::run(
                command.to_str().ok_or_else(|| xtask::Error::Invalid("command must be UTF-8".into()))?,
                arguments.collect(),
            )
        }
        Some(command) if command == OsStr::new("proto-go") => {
            xtask::proto::go(&arguments.collect::<Vec<_>>())
        }
        Some(command) if command == OsStr::new("proto-go-spec") => {
            xtask::proto::go_spec(arguments.collect())
        }
        Some(command) if ["proto-tools", "proto-controls", "capability-controls", "timing-controls", "build-guests"].iter().any(|name| command == OsStr::new(name)) => {
            native_leaf(command, arguments)
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
        Some(command) if command == OsStr::new("fetch-apalache") || command == OsStr::new("fetch-tlc") => {
            let root = arguments.next();
            if arguments.next().is_some() {
                return Err(xtask::Error::Invalid(
                    format!("usage: cargo xtask {} [framework-root]", command.to_string_lossy()),
                ));
            }
            let tools = if let Some(root) = root {
                ToolRoot::admit(Path::new(&root))?
            } else {
                ToolRoot::local()?
            };
            let path = if command == OsStr::new("fetch-tlc") {
                xtask::fetch_apalache::install_tlc(tools.path())?
            } else {
                xtask::fetch_apalache::install(tools.path())?
            };
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
            "commands: gate, quick, fmt, update, time, validate, test-run, fetch-tlc, fetch-apalache, guest, guest-freshness, guest-lock, retired-names, proto-tools, proto-go, proto-go-spec, proto-controls, capability-controls, timing-controls, build-guests".into(),
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
