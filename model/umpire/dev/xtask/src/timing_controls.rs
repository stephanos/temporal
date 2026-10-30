//! Behavioral controls for the native timing step using disposable child commands.
use std::{
    fs,
    io::{self, Write},
    path::Path,
    process::{Command, Output},
};

fn require(condition: bool, message: &str) -> io::Result<()> {
    if condition { Ok(()) } else { Err(io::Error::other(message)) }
}

fn status(output: &Output, expected: i32, label: &str) -> io::Result<()> {
    if output.status.code() == Some(expected) {
        return Ok(());
    }
    io::stderr().write_all(&output.stdout)?;
    io::stderr().write_all(&output.stderr)?;
    Err(io::Error::other(format!(
        "timing control {label}: expected exit {expected}, got {}",
        output.status
    )))
}

fn step(
    tool: &Path,
    scratch: &Path,
    label: &str,
    body: &str,
    arguments: &[&str],
) -> io::Result<Output> {
    let mut command = Command::new(tool);
    command
        .args(["time", "--step", label, "/bin/sh", "-c", body, "--"])
        .args(arguments)
        .current_dir(scratch)
        .env("TMPDIR", scratch)
        .env_remove("BASH_ENV");
    crate::gate::scheduler::execute(&command, false, false)
}

fn no_temporary_output(scratch: &Path) -> io::Result<()> {
    require(fs::read_dir(scratch)?.next().is_none(), "timing step retained temporary output")
}

/// Exercise the shipping timing front door, including its child process and output boundaries.
pub fn run(tool: &Path) -> io::Result<i32> {
    let directory = tempfile::tempdir()?;
    let scratch = directory.path().canonicalize()?;
    for (body, expected) in [("true", 0), ("false", 1), ("exit 37", 37)] {
        let output = step(tool, &scratch, "intentional_control", body, &[])?;
        status(&output, expected, body)?;
        no_temporary_output(&scratch)?;
    }
    for expected in [0, 37] {
        let output = step(
            tool,
            &scratch,
            "diagnostic_control",
            "printf 'stdout failure diagnostic\n'; printf 'stderr failure diagnostic\n' >&2; exit \"$1\"",
            &[&expected.to_string()],
        )?;
        status(&output, expected, "diagnostic_control")?;
        no_temporary_output(&scratch)?;
        let stdout = String::from_utf8_lossy(&output.stdout);
        let stderr = String::from_utf8_lossy(&output.stderr);
        if expected == 37 {
            let first = stderr.find("stdout failure diagnostic");
            let second = stderr.find("stderr failure diagnostic");
            require(
                first.zip(second).is_some_and(|(first, second)| first < second),
                "failed timing step lost or reordered child diagnostics",
            )?;
        } else {
            require(
                ![stdout.as_ref(), stderr.as_ref()].iter().any(|stream| {
                    stream.contains("stdout failure diagnostic")
                        || stream.contains("stderr failure diagnostic")
                }),
                "successful timing step exposed child output",
            )?;
            require(
                stdout.contains("diagnostic_control"),
                "successful timing step omitted its timing line",
            )?;
        }
    }
    let caller =
        scratch.to_str().ok_or_else(|| io::Error::other("non-UTF8 timing control directory"))?;
    let output = step(tool, &scratch, "caller-directory", "[ \"$PWD\" = \"$1\" ]", &[caller])?;
    status(&output, 0, "caller-directory")?;
    no_temporary_output(&scratch)?;
    let output = step(tool, &scratch, "child-status", "exit 37", &[])?;
    status(&output, 37, "child-status")?;
    no_temporary_output(&scratch)?;
    let output = step(tool, &scratch, "signal-status", "kill -TERM $$", &[])?;
    status(&output, 143, "signal-status")?;
    no_temporary_output(&scratch)?;
    relative_command(tool, &scratch)?;
    #[cfg(target_os = "linux")]
    descendants(tool, &scratch)?;
    Ok(0)
}

fn relative_command(tool: &Path, scratch: &Path) -> io::Result<()> {
    use std::os::unix::fs::PermissionsExt;
    let child = scratch.join("relative-child");
    fs::write(&child, "#!/bin/sh\n[ \"$PWD\" = \"$1\" ]\n")?;
    fs::set_permissions(&child, fs::Permissions::from_mode(0o700))?;
    let mut command = Command::new(tool);
    command
        .args(["time", "--step", "relative-child", "./relative-child"])
        .arg(scratch)
        .current_dir(scratch)
        .env("TMPDIR", scratch)
        .env_remove("BASH_ENV");
    let output = crate::gate::scheduler::execute(&command, false, false)?;
    fs::remove_file(child)?;
    status(&output, 0, "relative-child")?;
    no_temporary_output(scratch)
}

#[cfg(target_os = "linux")]
fn descendants(tool: &Path, scratch: &Path) -> io::Result<()> {
    let output = step(
        tool,
        scratch,
        "owned-descendant",
        "sleep 30 & printf '%s' $! > child.pid; exit 37",
        &[],
    )?;
    status(&output, 37, "owned-descendant")?;
    let pid = fs::read_to_string(scratch.join("child.pid"))?;
    fs::remove_file(scratch.join("child.pid"))?;
    let mut command = Command::new("/bin/kill");
    command.args(["-0", &pid]);
    let alive = crate::gate::scheduler::execute(&command, false, false)?.status.success();
    if alive {
        let mut command = Command::new("/bin/kill");
        command.args(["-KILL", &pid]);
        crate::gate::scheduler::execute(&command, false, false)?;
    }
    require(!alive, "timing step returned before owned descendant cleanup")?;
    no_temporary_output(scratch)
}
