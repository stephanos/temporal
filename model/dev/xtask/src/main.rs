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

fn run() -> io::Result<i32> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let caller = env::current_dir()?;
    let cargo_home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let mut arguments: Vec<_> = env::args_os().skip(1).collect();
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
