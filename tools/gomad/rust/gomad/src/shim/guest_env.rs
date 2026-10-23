use std::{collections::BTreeMap, ffi::OsString, path::Path, process::Command};
pub(super) fn validate() -> Result<(), String> {
    for (key, admitted) in [("GOOS", "wasip1"), ("GOARCH", "wasm"), ("CGO_ENABLED", "0")] {
        if std::env::var_os(key).is_some_and(|value| !value.is_empty() && value != admitted) {
            return Err(format!("unsupported guest option: {key}"));
        }
    }
    Ok(())
}
pub(super) fn command(
    compiler: &Path,
    arguments: &[OsString],
    environment: &BTreeMap<String, String>,
) -> Command {
    let mut command = Command::new(compiler);
    command.args(arguments).env_clear().envs(environment);
    command
}
