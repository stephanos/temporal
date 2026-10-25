//! Native guest pipeline controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
#[path = "pipeline/support.rs"]
mod support;
use std::{fs, io::Write, process::Command};
use support::{Fixture, code, executable};
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
fn cold_shell_bootstrap_and_cached_native_validation() {
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
        command.env("MISSING", "go");
        if let Some(ci) = ci {
            command.env("CI", ci);
        } else {
            command.env_remove("CI");
        }
        let result = command.output().unwrap();
        assert!(!result.status.success());
        assert!(!f.root.join("guest-built").exists());
    }
}

#[test]
fn cold_bootstrap_failures_never_build_or_publish() {
    for failure in ["missing-mise", "FAIL_BOOTSTRAP"] {
        let f = Fixture::new();
        let mut command = f.command(true, true);
        if failure == "missing-mise" {
            command.env("PATH", "/usr/bin:/bin");
        } else {
            command.env(failure, "1");
        }
        let output = command.output().unwrap();
        assert!(
            String::from_utf8_lossy(&output.stderr).contains(if failure == "missing-mise" {
                "requires mise"
            } else {
                "bootstrap-compiler-failed"
            }),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
        code(&output, 1);
        assert!(!f.root.join("guest-built").exists());
        assert!(!f.root.join("out.wasm").exists());
    }
}

#[test]
fn native_prerequisite_failures_stop_before_guest_build() {
    for (key, value) in [
        ("MISSING", "rust"),
        ("MISSING", "go"),
        ("MISSING", "protoc"),
        ("FAIL_PROBE", "1"),
        ("BAD_METADATA", "1"),
    ] {
        let f = Fixture::new();
        code(&f.command(true, true).env(key, value).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
        assert!(!f.root.join("out.wasm").exists());
    }
    for remove in [
        "rust/bin/cargo",
        "rust/lib/rustlib/src/rust/library/Cargo.toml",
        "rust/lib/rustlib/wasm32-wasip1/lib/libstd-control.rlib",
    ] {
        let f = Fixture::new();
        fs::remove_file(f.root.join(remove)).unwrap();
        code(&f.command(true, true).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
    }
    for contents in ["invalid [", "[tools]\nrust='0.0.0'\ngo='0'\nprotoc='0'\n"] {
        let f = Fixture::new();
        fs::write(f.model.join("mise.toml"), contents).unwrap();
        code(&f.command(true, true).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
    }
}

#[test]
fn shell_caller_relative_paths_and_changed_inputs_select_new_cache() {
    let f = Fixture::new();
    let original = f.command(false, true);
    let mut command = Command::new("/bin/bash");
    command
        .arg(f.model.join("scripts/guest.sh"))
        .args(["example-guest", "relative output.wasm"])
        .current_dir(&f.root);
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
fn rust_toolchain_metadata_and_rustup_failures_refuse() {
    for content in ["invalid [", "[toolchain]\nchannel='0.0.0'\n"] {
        let f = Fixture::new();
        fs::write(f.model.join("rust-toolchain.toml"), content).unwrap();
        code(&f.command(true, true).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
    }
    for script in [
        "#!/bin/sh\necho no-installed-toolchain\n",
        "#!/bin/sh\necho rustup-probe-failed >&2\nexit 27\n",
    ] {
        let f = Fixture::new();
        executable(&f.root.join("rust/bin/rustc"), script);
        fs::hard_link(f.root.join("rust/bin/rustc"), f.root.join("rust/bin/rustup")).unwrap();
        code(&f.command(true, true).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
    }
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
