//! Extension-owned guest entry point controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
mod retired_names;
#[path = "../../../umpire/dev/guest-driver/tests/pipeline/support.rs"]
#[allow(dead_code)]
mod support;
use std::{
    fs,
    path::{Path, PathBuf},
    process::Command,
};
use support::Fixture;
fn framework_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../../umpire")
}
fn xtask_binary() -> &'static PathBuf {
    static TOOL: std::sync::OnceLock<PathBuf> = std::sync::OnceLock::new();
    TOOL.get_or_init(|| {
        use std::os::unix::ffi::OsStringExt;
        let output = Command::new("/bin/bash")
            .arg(framework_root().join("scripts/xtask.sh"))
            .current_dir(framework_root())
            .env_remove("BASH_ENV")
            .output()
            .unwrap();
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        let path = output.stdout.strip_suffix(b"\n").unwrap_or(&output.stdout).to_vec();
        PathBuf::from(std::ffi::OsString::from_vec(path))
    })
}
fn driver_binary() -> PathBuf {
    xtask_binary().with_file_name("umpire-guest-driver")
}
fn compiler_fixture() -> &'static str {
    env!("CARGO_BIN_EXE_identity-compiler-fixture")
}
#[test]
fn actual_make_freshness_recipe_propagates_stale_failure() {
    let f = Fixture::new();
    fs::copy(f.model.join("Cargo.lock"), f.model.parent().unwrap().join("Cargo.lock")).unwrap();
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let makefile = fs::read_to_string(source.join("Makefile")).unwrap();
    let recipe = makefile
        .lines()
        .find(|line| line.starts_with('\t') && line.contains("temporal-guest-freshness"))
        .unwrap();
    fs::write(
        f.model.join("../freshness-control.mk"),
        format!(".PHONY: freshness-control\nfreshness-control:\n{recipe}\n"),
    )
    .unwrap();
    let output = f.model.join("temporal/hosts/go/umpire_temporal_guest.wasm");
    fs::create_dir_all(output.parent().unwrap()).unwrap();
    fs::write(&output, b"keep").unwrap();
    let environment = f.command(true, true);
    let mut command = Command::new("make");
    command.current_dir(f.model.parent().unwrap()).args([
        "--no-print-directory",
        "-f",
        "freshness-control.mk",
        "freshness-control",
    ]);
    for (key, value) in environment.get_envs() {
        if let Some(value) = value {
            command.env(key, value);
        } else {
            command.env_remove(key);
        }
    }
    let result = command.output().unwrap();
    assert!(!result.status.success());
    assert!(
        String::from_utf8_lossy(&result.stderr).contains("stale"),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert_eq!(fs::read(&output).unwrap(), b"keep");
}

#[test]
fn absent_embedded_guest_fails_go_compilation() {
    let f = Fixture::new();
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let go = Command::new("mise").args(["which", "go"]).current_dir(&source).output().unwrap();
    assert!(go.status.success(), "{}", String::from_utf8_lossy(&go.stderr));
    let module = f.root.join("isolated host");
    fs::create_dir_all(&module).unwrap();
    fs::copy(source.join("temporal/hosts/go/guest.go"), module.join("guest.go")).unwrap();
    fs::write(module.join("go.mod"), "module embedded-guest-control\n\ngo 1.27.0\n").unwrap();
    let output = Command::new(String::from_utf8(go.stdout).unwrap().trim())
        .args(["test", "-tags", "test_dep", "./..."])
        .current_dir(module)
        .env("GOTOOLCHAIN", "local")
        .env("CGO_ENABLED", "0")
        .env("GOCACHE", f.root.join("go-cache"))
        .output()
        .unwrap();
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr)
            .contains("umpire_temporal_guest.wasm: no matching files found"),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
}
