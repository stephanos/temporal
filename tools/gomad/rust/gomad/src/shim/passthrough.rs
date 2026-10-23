use std::{
    ffi::OsString,
    os::unix::process::{CommandExt as _, ExitStatusExt as _},
    path::Path,
    process::{Command, ExitStatus},
};
pub(super) fn run(compiler: &Path, arguments: &[OsString]) -> Result<i32, String> {
    // No environment normalization or argument decoding belongs on the host route.
    Command::new(compiler)
        .arg0(
            std::env::args_os()
                .next()
                .unwrap_or_else(|| compiler.as_os_str().to_owned()),
        )
        .args(arguments)
        .status()
        .map(status)
        .map_err(|e| e.to_string())
}
pub(super) fn status(status: ExitStatus) -> i32 {
    status
        .code()
        .unwrap_or_else(|| 128 + status.signal().unwrap_or(1))
}
