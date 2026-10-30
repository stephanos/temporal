//! Native guest pipeline controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
#[path = "pipeline/support.rs"]
mod support;
use std::{fs, io::Write, process::Command};
use support::{Fixture, code};
#[test]
fn guest_build_check_and_artifact_refusals() {
    let f = Fixture::new();
    code(&f.command(false, false).output().unwrap(), 0);
    assert_eq!(fs::read(f.root.join("out.wasm")).unwrap(), b"\0asm\x01\0\0\0");
    code(&f.command(false, false).arg("--check").output().unwrap(), 0);
    for contents in [b"keep".as_slice(), b"\0asm", b"\0asm\x01\0\0\0X"] {
        fs::write(f.root.join("out.wasm"), contents).unwrap();
        code(&f.command(false, false).arg("--check").output().unwrap(), 2);
        assert_eq!(fs::read(f.root.join("out.wasm")).unwrap(), contents);
    }
    fs::remove_file(f.root.join("out.wasm")).unwrap();
    code(&f.command(false, false).arg("--check").output().unwrap(), 2);
    assert!(!f.root.join("out.wasm").exists());
    for (key, value) in [
        ("ARTIFACT_MODE", "malformed"),
        ("ARTIFACT_MODE", "missing"),
        ("ARTIFACT_MODE", "multiple"),
        ("FAIL_BUILD", "1"),
        ("FAIL_METADATA", "1"),
    ] {
        fs::write(f.root.join("out.wasm"), "keep").unwrap();
        let result = f.command(false, false).env(key, value).output().unwrap();
        if key == "FAIL_BUILD" {
            assert!(String::from_utf8_lossy(&result.stderr).contains("guest-compiler-failed"));
        }
        code(&result, if key == "FAIL_BUILD" { 37 } else { 1 });
        assert_eq!(fs::read(f.root.join("out.wasm")).unwrap(), b"keep");
    }
}

#[test]
fn cold_retained_bootstrap_and_cached_native_validation() {
    let f = Fixture::new();
    fs::write(f.root.join("out.wasm"), b"\0asm\x01\0\0\0").unwrap();
    code(
        &f.command(true, true)
            .env("MISE_AUTO_INSTALL", "true")
            .env("MISE_EXEC_AUTO_INSTALL", "true")
            .output()
            .unwrap(),
        0,
    );
    assert!(f.root.join("guest-built").exists());
    fs::remove_file(f.root.join("guest-built")).unwrap();
    for ci in [None, Some(""), Some("true")] {
        let mut command = f.command(true, true);
        if let Some(ci) = ci {
            command.env("CI", ci);
        } else {
            command.env_remove("CI");
        }
        let result = command.output().unwrap();
        code(&result, 0);
        assert!(f.root.join("guest-built").exists());
        fs::remove_file(f.root.join("guest-built")).unwrap();
    }
}

#[test]
fn cold_bootstrap_failure_never_builds_or_publishes() {
    let f = Fixture::new();
    let output = f.command(true, true).env("FAIL_BOOTSTRAP", "1").output().unwrap();
    assert!(String::from_utf8_lossy(&output.stderr).contains("bootstrap-compiler-failed"));
    code(&output, 1);
    assert!(!f.root.join("guest-built").exists());
    assert!(!f.root.join("out.wasm").exists());
}

#[test]
fn native_caller_relative_paths_and_changed_inputs_select_new_cache() {
    let f = Fixture::new();
    let original = f.command(false, true);
    let mut command = Command::new(crate::driver_binary());
    command.args(["guest", "example-guest", "relative output.wasm"]).current_dir(&f.root);
    for (key, value) in original.get_envs() {
        if let Some(value) = value {
            command.env(key, value);
        } else {
            command.env_remove(key);
        }
    }
    fs::copy(f.model.join("Cargo.lock"), f.root.join("Cargo.lock")).unwrap();
    command.env("CARGO_TARGET_DIR", "relative target");
    code(&command.output().unwrap(), 0);
    assert_eq!(fs::read(f.root.join("relative output.wasm")).unwrap(), b"\0asm\x01\0\0\0");
    let first = fs::read(f.root.join("guest-cache")).unwrap();
    fs::OpenOptions::new()
        .append(true)
        .open(f.root.join("Cargo.lock"))
        .unwrap()
        .write_all(b"\n# cache invalidation control\n")
        .unwrap();
    code(&command.output().unwrap(), 0);
    assert_ne!(fs::read(f.root.join("guest-cache")).unwrap(), first);
    assert!(String::from_utf8(first).unwrap().contains("relative target/guest-cache/"));
}

#[test]
fn loader_inheritance_survives_orchestration_environment_purge() {
    let f = Fixture::new();
    let mise = f.root.join("bin/mise");
    let script = fs::read_to_string(&mise).unwrap();
    fs::write(&mise, script.replace("set -eu\n", "set -eu\nunset DYLD_FALLBACK_LIBRARY_PATH\n"))
        .unwrap();
    code(&f.command(false, false).output().unwrap(), 0);
    assert_eq!(fs::read(f.root.join("previous-called")).unwrap(), b"called");
    let observed: serde_json::Value =
        serde_json::from_slice(&fs::read(f.root.join("loader.json")).unwrap()).unwrap();
    assert_eq!(observed["loader"], serde_json::json!(b"loader-marker".as_slice()));
}

#[test]
fn relative_cargo_home_stays_with_distinct_caller_for_probes_metadata_and_build() {
    let f = Fixture::new();
    fs::write(f.root.join("out.wasm"), b"\0asm\x01\0\0\0").unwrap();
    let caller = f.root.join("caller");
    fs::create_dir_all(caller.join("cargo home")).unwrap();
    fs::create_dir_all(f.model.join("cargo home")).unwrap();
    fs::copy(f.model.join("Cargo.lock"), caller.join("Cargo.lock")).unwrap();
    std::os::unix::fs::symlink(caller.join("cargo home"), caller.join("cargo alias")).unwrap();
    for shell in [false, true] {
        for home in ["cargo home", "cargo alias"] {
            let result = f
                .command(true, shell)
                .current_dir(&caller)
                .env("CARGO_HOME", home)
                .env("EXPECTED_CARGO_HOME", caller.join("cargo home"))
                .output()
                .unwrap();
            code(&result, 0);
        }
    }
}

#[test]
fn retained_bootstrap_stdout_selects_executable_and_native_route_creates_output() {
    let f = Fixture::new();
    let output = f.bootstrap_command().output().unwrap();
    code(&output, 0);
    let tool =
        f.root.join("custom target").canonicalize().unwrap().join("xtask-bootstrap/debug/xtask");
    let mut expected = tool.as_os_str().as_encoded_bytes().to_vec();
    expected.push(b'\n');
    assert_eq!(output.stdout, expected);
    assert!(tool.is_file());
    assert!(tool.with_file_name("umpire-guest-driver").is_file());
    assert!(!f.root.join("out.wasm").exists());
    code(&f.command(false, true).output().unwrap(), 0);
    assert_eq!(fs::read(f.root.join("out.wasm")).unwrap(), b"\0asm\x01\0\0\0");
    assert_eq!(fs::read(f.root.join("bootstrap-path")).unwrap(), expected);
}

#[test]
fn missing_mise_refuses_bootstrap_without_fallback_cargo() {
    let f = Fixture::new();
    let bin = f.root.join("bootstrap-only bin");
    fs::create_dir(&bin).unwrap();
    for (name, executable) in [("dirname", "/usr/bin/dirname"), ("mkdir", "/bin/mkdir")] {
        std::os::unix::fs::symlink(executable, bin.join(name)).unwrap();
    }
    support::executable(
        &bin.join("cargo"),
        "#!/bin/bash\n: > \"$FIXTURE/fallback-cargo\"\nexit 0\n",
    );
    let output = f.bootstrap_command().env("PATH", &bin).output().unwrap();
    code(&output, 1);
    assert_eq!(output.stdout, b"");
    assert!(String::from_utf8_lossy(&output.stderr).contains("mise"));
    assert!(String::from_utf8_lossy(&output.stderr).contains("xtask bootstrap failed"));
    assert!(!f.root.join("fallback-cargo").exists());
    assert!(!f.root.join("guest-built").exists());
    assert!(!f.root.join("out.wasm").exists());
}
