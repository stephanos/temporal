//! System workspace command bridge to the independent framework tools.
#![forbid(unsafe_code)]
use std::{
    env,
    ffi::OsString,
    io::{self, Write},
    os::unix::{
        ffi::OsStringExt,
        process::{CommandExt, ExitStatusExt},
    },
    path::{Path, PathBuf},
    process::{Command, Stdio},
};

mod gate;
mod quick;
mod test_run;

fn run() -> io::Result<i32> {
    xtask::gate::work_counts::owns_controller()?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let mut supplied: Vec<_> = env::args_os().skip(1).collect();
    let _lease = if supplied.first().is_some_and(|arg| arg == "__owned-command") {
        supplied.remove(0);
        if env::var_os("UMPIRE_JVM_LEASE_FD").is_none() {
            return Err(io::Error::other("missing owned-command descriptor"));
        }
        Some(xtask::gate::lease::JvmLease::inherited_or_acquire()?)
    } else if supplied.first().is_some_and(|arg| {
        ["gate", "test-run", "quick", "update", "time", "validate"].iter().any(|name| arg == name)
    }) && !supplied.iter().any(|arg| arg == "--list")
    {
        return xtask::gate::supervise(supplied).map_err(io::Error::other);
    } else {
        None
    };
    if supplied.first().is_some_and(|command| command == "test-run") {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask test-run"));
        }
        return test_run::run(&root);
    }
    if supplied.first().is_some_and(|command| command == "gate") {
        let arguments: Vec<_> = supplied.into_iter().skip(1).collect();
        return match arguments.as_slice() {
            [] => gate::run(&root),
            [flag] if flag == "--list" => gate::list(&root),
            [flag, name] if flag == "--stage" || flag == "--scheduled-stage" => {
                let name =
                    name.to_str().ok_or_else(|| io::Error::other("gate stage must be UTF-8"))?;
                if flag == "--stage" {
                    gate::run_stage(&root, name)
                } else {
                    gate::scheduled_stage(&root, name)
                }
            }
            _ => Err(io::Error::other(
                "usage: cargo xtask gate [--list | --stage NAME | --scheduled-stage NAME]",
            )),
        };
    }
    let caller = env::current_dir()?;
    let cargo_home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let mut arguments = supplied;
    if arguments.first().is_some_and(|command| command == "quick") {
        arguments = quick::arguments(&root, arguments.into_iter().skip(1).collect())?;
    }
    let guest =
        arguments.first().is_some_and(|command| command == "guest" || command == "guest-freshness");
    let mut workspace = caller.clone();
    if guest {
        workspace.clone_from(&root);
        if arguments.get(1).is_some_and(|package| {
            [
                "umpire-stock-guest",
                "umpire-kettle-guest",
                "umpire-control-guest",
                "umpire-caught-panic-guest",
                "umpire-overflow-guest",
            ]
            .iter()
            .any(|known| package == known)
        }) {
            workspace.push("umpire");
        }
        if let Some(output) = arguments.get_mut(2) {
            let path = Path::new(output);
            if path.is_relative() {
                *output = caller.join(path).into_os_string();
            }
        }
    }
    let output = Command::new("/bin/bash")
        .arg(root.join("scripts/xtask.sh"))
        .env("CARGO_HOME", &cargo_home)
        .stderr(Stdio::inherit())
        .output()?;
    if !output.status.success() {
        return Ok(output
            .status
            .code()
            .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)));
    }
    let path = output.stdout.strip_suffix(b"\n").unwrap_or(&output.stdout);
    let tool = PathBuf::from(OsString::from_vec(path.to_owned()));
    let mut command = Command::new(tool);
    command.args(arguments).current_dir(workspace).env("CARGO_HOME", cargo_home);
    if guest && let Some(target) = env::var_os("CARGO_TARGET_DIR") {
        let target = PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    Err(command.exec())
}

fn main() {
    let status = match run() {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "temporal-xtask: {error}");
            1
        }
    };
    std::process::exit(status);
}
