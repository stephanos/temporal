//! Explicit provisioning command controls; no release asset is downloaded here.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{fs, os::unix::fs::{PermissionsExt, symlink}, process::Command};

#[test]
fn dispatch_usage_failure_is_nonzero_and_structured() {
    for name in ["fetch-tlc", "fetch-apalache"] {
        let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
            .args([name, ".", "extra"]).output().unwrap();
        assert_eq!(output.status.code(), Some(1));
        assert!(output.stdout.is_empty());
        assert!(String::from_utf8(output.stderr).unwrap()
            .starts_with(&format!("xtask: usage: cargo xtask {name}")));
    }
}

#[test]
fn provider_uses_explicit_opt_in_and_surfaces_install_failure() {
    for name in ["fetch-tlc", "fetch-apalache"] {
        let root = tempfile::tempdir().unwrap();
        let bin = root.path().join("bin");
        fs::create_dir(&bin).unwrap();
        let mise = bin.join("mise");
        fs::write(&mise, r#"#!/bin/sh
printf '%s\n' "$@" > "$FIXTURE_LOG"
exit 42
"#).unwrap();
        fs::set_permissions(&mise, fs::Permissions::from_mode(0o700)).unwrap();
        for file in ["Cargo.toml", "mise.toml", "rust-toolchain.toml"] {
            fs::write(root.path().join(file), "fixture").unwrap();
        }
        fs::create_dir(root.path().join("mise-tools")).unwrap();
        for file in ["mise.toml", "mise.apalache.toml", "empty.toml"] {
            fs::write(root.path().join("mise-tools").join(file), "").unwrap();
        }
        let log = root.path().join("args");
        let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
            .arg(name).arg(root.path()).env("PATH", &bin).env("FIXTURE_LOG", &log)
            .output().unwrap();
        assert_eq!(output.status.code(), Some(1));
        assert!(String::from_utf8(output.stderr).unwrap().contains("mise jar installation failed"));
        let expected = if name == "fetch-tlc" { "install\nhttp:tlc@1.7.4\n" }
            else { "--env\napalache\ninstall\nhttp:apalache@0.62.4\n" };
        assert_eq!(fs::read_to_string(log).unwrap(), expected);
        let outside = tempfile::tempdir().unwrap();
        fs::remove_dir(root.path().join("build/mise-jars/cache")).unwrap();
        symlink(outside.path(), root.path().join("build/mise-jars/cache")).unwrap();
        let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
            .arg(name).arg(root.path()).env("PATH", &bin).output().unwrap();
        assert_eq!(output.status.code(), Some(1));
        assert!(String::from_utf8(output.stderr).unwrap().contains("symlink"));
        assert_eq!(fs::read_dir(outside.path()).unwrap().count(), 0);
    }
}

#[test]
fn concurrent_failed_provisioning_preserves_partial_data_and_cleans_capture() {
    let root = tempfile::tempdir().unwrap();
    let bin = root.path().join("bin");
    fs::create_dir(&bin).unwrap();
    let mise = bin.join("mise");
    fs::write(&mise, r#"#!/bin/sh
printf partial > "$MISE_DATA_DIR/install-partial"
printf failed-provider >&2
exit 42
"#).unwrap();
    fs::set_permissions(&mise, fs::Permissions::from_mode(0o700)).unwrap();
    for file in ["Cargo.toml", "mise.toml", "rust-toolchain.toml"] {
        fs::write(root.path().join(file), "fixture").unwrap();
    }
    fs::create_dir(root.path().join("mise-tools")).unwrap();
    for file in ["mise.toml", "mise.apalache.toml", "empty.toml"] {
        fs::write(root.path().join("mise-tools").join(file), "").unwrap();
    }
    std::thread::scope(|scope| {
        let workers: Vec<_> = (0..2).map(|_| {
            let root = root.path();
            let bin = &bin;
            scope.spawn(move || Command::new(env!("CARGO_BIN_EXE_xtask"))
                .arg("fetch-tlc").arg(root).env("PATH", bin).output().unwrap())
        }).collect();
        for worker in workers {
            let output = worker.join().unwrap();
            assert_eq!(output.status.code(), Some(1));
            assert!(output.stdout.is_empty());
            let error = String::from_utf8(output.stderr).unwrap();
            assert!(error.contains("failed-provider"));
            assert!(error.contains("mise jar installation failed"));
        }
    });
    assert!(root.path().join("build/mise-jars/data/install-partial").is_file());
    assert_eq!(fs::read_dir(root.path().join("build/mise-jars/runs")).unwrap().count(), 0);
    assert!(!root.path().join("build/mise-jars/data/installs/http-tlc/1.7.4/tla2tools.jar").exists());
}
