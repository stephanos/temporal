//! Temporal wasm build leaf, invoked after the framework driver bootstrap.
use std::{
    env, fs, io,
    os::unix::process::ExitStatusExt,
    path::{Path, PathBuf},
    process::Command,
};

/// Preserve caller-relative Cargo locations before entering the Model workspace.
pub(crate) fn run(root: &Path, driver: &Path) -> io::Result<i32> {
    let caller = env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let target = env::var_os("CARGO_TARGET_DIR").filter(|value| !value.is_empty()).map_or_else(
        || root.join("target"),
        |value| {
            let path = PathBuf::from(value);
            if path.is_absolute() { path } else { caller.join(path) }
        },
    );
    fs::create_dir_all(&target)?;
    let target = target.canonicalize()?;
    let mut command = Command::new(driver);
    command
        .current_dir(root)
        .env("CARGO_HOME", home)
        .env("CARGO_TARGET_DIR", &target)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .args(["guest", "umpire-temporal-guest"])
        .arg(target.join("wasm32-wasip1/guest/umpire_temporal_guest.wasm"));
    let status = xtask::gate::scheduler::execute(&command, true, true)?.status;
    Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
}

#[cfg(test)]
#[path = "temporal_guest_path_controls.rs"]
mod path_controls;
