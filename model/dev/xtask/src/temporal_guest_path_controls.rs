use super::*;
use crate::script_control_support::Scratch;
use std::{
    ffi::OsStr,
    os::unix::{
        ffi::OsStringExt,
        fs::{PermissionsExt, symlink},
    },
};

#[test]
#[ignore = "isolated subprocess entry"]
fn isolated_build_child() -> io::Result<()> {
    let root = env::var_os("UMPIRE_TEMPORAL_PATH_ROOT")
        .ok_or_else(|| io::Error::other("missing child root"))?;
    let driver = env::var_os("UMPIRE_TEMPORAL_PATH_DRIVER")
        .ok_or_else(|| io::Error::other("missing child driver"))?;
    std::process::exit(run(Path::new(&root), Path::new(&driver))?)
}

#[test]
fn relative_absolute_default_and_symlink_paths_preserve_temporal_guest() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let model = scratch.0.join("model with spaces");
    let caller = scratch.0.join("caller with spaces");
    fs::create_dir_all(&model)?;
    fs::create_dir_all(caller.join("physical cargo"))?;
    fs::create_dir_all(caller.join("physical target"))?;
    symlink("physical cargo", caller.join("cargo home"))?;
    symlink("physical target", caller.join("target link"))?;
    let driver = scratch.0.join("native guest driver");
    fs::write(&driver, br#"#!/bin/bash
set -euo pipefail
[[ "$PWD" = "$EXPECTED_ROOT" && "$CARGO_HOME" = "$EXPECTED_HOME" && "$CARGO_TARGET_DIR" = "$EXPECTED_TARGET" ]]
[[ "$MISE_AUTO_INSTALL" = false && "$MISE_EXEC_AUTO_INSTALL" = false ]]
printf '%s\0' "$@" > "$CONTROL_LOG"
exit "${CONTROL_STATUS:-0}"
"#)?;
    fs::set_permissions(&driver, fs::Permissions::from_mode(0o755))?;
    let absolute_home = caller.join("cargo home");
    let physical_home = caller.join("physical cargo");
    let missing_home = caller.join("missing cargo");
    let absolute_target = caller.join("target link");
    let physical_target = caller.join("physical target");
    let default_target = model.join("target");
    let raw_home = std::ffi::OsString::from_vec(b"cargo-\xff".to_vec());
    let raw_target = std::ffi::OsString::from_vec(b"target-\xfe".to_vec());
    let raw_home_path = caller.join(&raw_home);
    let raw_target_path = caller.join(&raw_target);
    fs::create_dir(&raw_home_path)?;
    for (home, expected_home) in [
        (OsStr::new("cargo home"), &physical_home),
        (absolute_home.as_os_str(), &physical_home),
        (OsStr::new("missing cargo"), &missing_home),
        (raw_home.as_os_str(), &raw_home_path),
    ] {
        for (target, expected_target) in [
            (Some(OsStr::new("target link")), &physical_target),
            (Some(absolute_target.as_os_str()), &physical_target),
            (None, &default_target),
            (Some(raw_target.as_os_str()), &raw_target_path),
        ] {
            for status in [0, 37] {
                let mut command = Command::new(env::current_exe()?);
                command
                    .args([
                        "--exact",
                        "build_guests::path_controls::isolated_build_child",
                        "--nocapture",
                        "--ignored",
                    ])
                    .current_dir(&caller)
                    .env("UMPIRE_TEMPORAL_PATH_ROOT", &model)
                    .env("UMPIRE_TEMPORAL_PATH_DRIVER", &driver)
                    .env("CARGO_HOME", home)
                    .env("EXPECTED_HOME", expected_home)
                    .env("EXPECTED_ROOT", &model)
                    .env("EXPECTED_TARGET", expected_target)
                    .env("CONTROL_LOG", scratch.0.join("calls"))
                    .env("CONTROL_STATUS", status.to_string())
                    .env("MISE_AUTO_INSTALL", "true")
                    .env("MISE_EXEC_AUTO_INSTALL", "true")
                    .env_remove("BASH_ENV");
                if let Some(target) = target {
                    command.env("CARGO_TARGET_DIR", target);
                } else {
                    command.env_remove("CARGO_TARGET_DIR");
                }
                let output = xtask::gate::scheduler::execute(&command, false, false)?;
                assert_eq!(
                    output.status.code(),
                    Some(status),
                    "{}",
                    String::from_utf8_lossy(&output.stderr)
                );
                let mut expected = b"guest\0umpire-temporal-guest\0".to_vec();
                expected.extend(
                    expected_target
                        .join("wasm32-wasip1/guest/umpire_temporal_guest.wasm")
                        .as_os_str()
                        .as_encoded_bytes(),
                );
                expected.push(0);
                assert_eq!(fs::read(scratch.0.join("calls"))?, expected);
            }
        }
    }
    let path = scratch.0.clone();
    drop(scratch);
    assert!(!path.exists());
    Ok(())
}
