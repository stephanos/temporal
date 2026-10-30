//! Timing output over the gate's owned command executor.
use std::{
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    process::{Command, Output},
    time::Instant,
};

/// Execute a stage, suppressing successful diagnostics when timed.
/// Failed timed stages replay stdout followed by stderr after owned cleanup.
pub fn capture(command: &Command, timed: bool, inventory: Option<&[u8]>) -> io::Result<Output> {
    let output = super::scheduler::execute_with_inventory(command, !timed, !timed, inventory)?;
    if timed && !output.status.success() {
        io::stderr().write_all(&output.stdout)?;
        io::stderr().write_all(&output.stderr)?;
    }
    Ok(output)
}

/// Run one public timed step without changing its caller's working directory.
pub fn execute(
    name: &str,
    command: &Command,
    timed: bool,
    inventory: Option<&[u8]>,
) -> io::Result<i32> {
    let started = Instant::now();
    let output = capture(command, timed, inventory)?;
    let code = output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0));
    if code == 0 && timed {
        writeln!(io::stdout(), "{name:<32} {:6.3}s", started.elapsed().as_secs_f64())?;
    }
    Ok(code)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn capture_preserves_status_and_captures_both_failed_streams() -> io::Result<()> {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "printf output; printf error >&2; exit 37"]);
        let output = capture(&command, true, None)?;
        assert_eq!(output.status.code(), Some(37));
        assert_eq!(output.stdout, b"output");
        assert_eq!(output.stderr, b"error");
        Ok(())
    }

    #[test]
    fn step_preserves_signal_status() -> io::Result<()> {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "kill -TERM $$"]);
        assert_eq!(execute("signal", &command, true, None)?, 143);
        Ok(())
    }
}
