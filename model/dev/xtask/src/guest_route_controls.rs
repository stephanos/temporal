use super::*;
use crate::script_control_support::Scratch;
use std::{
    fs, io,
    os::unix::{
        ffi::OsStringExt,
        fs::{PermissionsExt, symlink},
    },
};

#[test]
fn seven_packages_two_entries_preserve_caller_paths_and_exact_arguments() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let model = scratch.0.join("model with spaces");
    let caller = scratch.0.join("caller with spaces");
    fs::create_dir_all(model.join("umpire"))?;
    fs::create_dir_all(caller.join("real cargo"))?;
    fs::create_dir_all(caller.join("real target"))?;
    symlink("real cargo", caller.join("cargo home"))?;
    symlink("real target", caller.join("relative target"))?;
    let home = caller.join("cargo home").canonicalize()?;
    let tool = scratch.0.join("native driver");
    fs::write(&tool, b"#!/bin/bash\nset -euo pipefail\nprintf '%s\\0' \"$PWD\" \"$@\" \"$CARGO_TARGET_DIR\" \"$CARGO_HOME\"\nexit \"${ROUTE_STATUS:-0}\"\n")?;
    fs::set_permissions(&tool, fs::Permissions::from_mode(0o755))?;
    for (package, framework) in [
        ("umpire-stock-guest", true),
        ("umpire-kettle-guest", true),
        ("umpire-control-guest", true),
        ("umpire-caught-panic-guest", true),
        ("umpire-overflow-guest", true),
        ("umpire-temporal-guest", false),
        ("custom-extension", false),
    ] {
        for entry in ["guest", "guest-freshness"] {
            let mut args: Vec<OsString> =
                [entry, package, "relative output.wasm"].into_iter().map(OsString::from).collect();
            let workspace = prepare(&model, &caller, &mut args);
            assert_eq!(workspace, if framework { model.join("umpire") } else { model.clone() });
            let mut child =
                command(&tool, &workspace, &caller, &args, &home, Some("relative target".into()));
            child.env_remove("BASH_ENV");
            let output = xtask::gate::scheduler::execute(&child, false, false)?;
            assert!(output.status.success());
            let expected = [
                workspace.into_os_string(),
                entry.into(),
                package.into(),
                caller.join("relative output.wasm").into_os_string(),
                caller.join("relative target").into_os_string(),
                home.clone().into_os_string(),
            ];
            let expected: Vec<u8> = expected
                .into_iter()
                .flat_map(|value| {
                    let mut bytes = value.into_vec();
                    bytes.push(0);
                    bytes
                })
                .collect();
            assert_eq!(output.stdout, expected);
        }
    }
    let mut args = vec![
        "guest".into(),
        "custom-extension".into(),
        "/absolute/output.wasm".into(),
        "--check".into(),
    ];
    let workspace = prepare(&model, &caller, &mut args);
    assert_eq!(args[2], "/absolute/output.wasm");
    assert_eq!(args[3], "--check");
    let mut child = command(
        &tool,
        &workspace,
        &caller,
        &args,
        &home,
        Some(caller.join("real target").into_os_string()),
    );
    child.env_remove("BASH_ENV").env("ROUTE_STATUS", "37");
    let output = xtask::gate::scheduler::execute(&child, false, false)?;
    assert_eq!(output.status.code(), Some(37));
    Ok(())
}

#[test]
fn non_unicode_output_and_target_are_lossless() {
    let root = Path::new("/model");
    let caller = Path::new("/caller");
    let raw = OsString::from_vec(b"output-\xff.wasm".to_vec());
    let target = OsString::from_vec(b"target-\xfe".to_vec());
    let mut args = vec!["guest".into(), "custom-extension".into(), raw.clone()];
    let workspace = prepare(root, caller, &mut args);
    assert_eq!(args[2], caller.join(raw).into_os_string());
    let command = command(
        Path::new("/driver"),
        &workspace,
        caller,
        &args,
        Path::new("/cargo"),
        Some(target.clone()),
    );
    assert!(
        command.get_envs().any(|(key, value)| key == "CARGO_TARGET_DIR"
            && value == Some(caller.join(&target).as_os_str()))
    );
}

#[test]
fn raw_arguments_check_output_and_failure_status_survive_execution() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let caller = scratch.0.join("caller with spaces");
    fs::create_dir(&caller)?;
    let driver = scratch.0.join("driver");
    fs::write(&driver, b"#!/bin/sh\nprintf '%s\\0' \"$@\" \"$CARGO_TARGET_DIR\"\nexit 37\n")?;
    fs::set_permissions(&driver, fs::Permissions::from_mode(0o755))?;
    let raw = OsString::from_vec(b"output-\xff.wasm".to_vec());
    let target = OsString::from_vec(b"target-\xfe".to_vec());
    let output_path = caller.join(&raw);
    fs::write(&output_path, b"preserved output")?;
    for entry in ["guest", "guest-freshness"] {
        let mut args = vec![entry.into(), "custom-extension".into(), raw.clone(), "--check".into()];
        let workspace = prepare(&caller, &caller, &mut args);
        let child = command(&driver, &workspace, &caller, &args, &caller, Some(target.clone()));
        let output = xtask::gate::scheduler::execute(&child, false, false)?;
        assert_eq!(output.status.code(), Some(37));
        let expected: Vec<u8> = args
            .iter()
            .cloned()
            .chain([caller.join(&target).into_os_string()])
            .flat_map(|value| {
                let mut bytes = value.into_vec();
                bytes.push(0);
                bytes
            })
            .collect();
        assert_eq!(output.stdout, expected);
        assert_eq!(fs::read(&output_path)?, b"preserved output");
    }
    let child = command(&driver, &caller, &caller, &[], &caller, None);
    assert!(!child.get_envs().any(|(key, _)| key == "CARGO_TARGET_DIR"));
    Ok(())
}
