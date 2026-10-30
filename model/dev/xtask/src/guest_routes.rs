//! Caller-preserving routing from the system workspace to the native guest driver.
use std::{
    ffi::OsString,
    path::{Path, PathBuf},
    process::Command,
};

pub(crate) fn prepare(root: &Path, caller: &Path, arguments: &mut [OsString]) -> PathBuf {
    let mut workspace = root.to_owned();
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
    workspace
}

pub(crate) fn command(
    tool: &Path,
    workspace: &Path,
    caller: &Path,
    arguments: &[OsString],
    home: &Path,
    target: Option<OsString>,
) -> Command {
    let mut command = Command::new(tool);
    command.args(arguments).current_dir(workspace).env("CARGO_HOME", home);
    if let Some(target) = target {
        let target = PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    command
}

#[cfg(test)]
#[path = "guest_route_controls.rs"]
mod tests;
