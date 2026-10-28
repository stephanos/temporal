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
    process::Command,
};

mod build_guests;
mod gate;
mod guest_routes;
#[cfg(test)]
mod model_controls;
mod proto;
mod quick;
#[cfg(test)]
#[path = "../tests/retired_names.rs"]
mod retired_name_policy_controls;
mod retired_names;
#[cfg(test)]
mod script_control_support;
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
        [
            "gate",
            "test-run",
            "quick",
            "update",
            "time",
            "validate",
            "timing-controls",
            "proto-controls",
        ]
        .iter()
        .any(|name| arg == name)
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
    if supplied.first().is_some_and(|command| command == "time" || command == "retired-names") {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask <time|retired-names>"));
        }
        return if supplied[0] == "time" {
            gate::timing::run(&root)
        } else {
            Ok(retired_names::run(
                root.parent().ok_or_else(|| io::Error::other("missing repository root"))?,
            ))
        };
    }
    if supplied
        .first()
        .is_some_and(|command| command == "timing-controls" || command == "proto-controls")
    {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask <timing-controls|proto-controls>"));
        }
        return gate::control_tests(
            &root,
            supplied[0].to_str().ok_or_else(|| io::Error::other("control name must be UTF-8"))?,
        );
    }
    let caller = env::current_dir()?;
    let cargo_home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let mut arguments = supplied;
    if arguments.first().is_some_and(|command| command == "quick") {
        arguments = quick::arguments(&root, arguments.into_iter().skip(1).collect())?;
    }
    let guest =
        arguments.first().is_some_and(|command| command == "guest" || command == "guest-freshness");
    let workspace =
        if guest { guest_routes::prepare(&root, &caller, &mut arguments) } else { caller.clone() };
    let output = gate::framework_executable(&root)?;
    if !output.status.success() {
        return Ok(output
            .status
            .code()
            .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)));
    }
    let path = output.stdout.strip_suffix(b"\n").unwrap_or(&output.stdout);
    let tool = PathBuf::from(OsString::from_vec(path.to_owned()));
    if arguments.first().is_some_and(|command| command == "proto-go") {
        return proto::run(&root, &tool, &arguments[1..]);
    }
    if arguments.first().is_some_and(|command| command == "build-guests") {
        if arguments.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask build-guests"));
        }
        return build_guests::run(&root, &tool.with_file_name("umpire-guest-driver"));
    }
    let mut command = if guest {
        guest_routes::command(
            &tool,
            &workspace,
            &caller,
            &arguments,
            &cargo_home,
            env::var_os("CARGO_TARGET_DIR"),
        )
    } else {
        let mut command = Command::new(tool);
        command.args(arguments).current_dir(workspace).env("CARGO_HOME", cargo_home);
        command
    };
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
