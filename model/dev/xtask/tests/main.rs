//! Extension-owned guest entry point controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
mod make_paths;
#[path = "../../../../../umpire/tools/guest-driver/tests/pipeline/support.rs"]
#[allow(dead_code)]
mod support;
use std::{
    fs,
    path::{Path, PathBuf},
    process::Command,
};
use support::Fixture;
fn framework_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../../umpire")
}
fn xtask_binary() -> &'static PathBuf {
    static TOOL: std::sync::OnceLock<PathBuf> = std::sync::OnceLock::new();
    TOOL.get_or_init(|| {
        use std::os::unix::ffi::OsStringExt;
        let output = Command::new("/bin/bash")
            .arg(framework_root().join("tools/scripts/xtask.sh"))
            .current_dir(framework_root())
            .env_remove("BASH_ENV")
            .env_remove("BUILD_CACHE_MANAGED")
            .env_remove("BUILD_CACHE_TOOL")
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
fn public_timer_uses_consumer_supervision_before_the_full_gate() {
    let fixture = Fixture::new();
    support::executable(
        &fixture.root.join("bin/make"),
        r#"#!/bin/sh
set -eu
[ "$*" = "-C $CONTROL_REPOSITORY model-check" ]
[ "$UMPIRE_TIME_STEPS" = 1 ]
printf 'consumer timer reached\n' > "$CONTROL_MARKER"
exit 37
"#,
    );
    let model = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize().unwrap();
    let marker = fixture.root.join("timer-reached");
    let mut paths = vec![fixture.root.join("bin")];
    paths.extend(std::env::split_paths(&std::env::var_os("PATH").unwrap_or_default()));
    let output = Command::new(env!("CARGO_BIN_EXE_temporal-xtask"))
        .arg("time")
        .current_dir(&model)
        .env("PATH", std::env::join_paths(paths).unwrap())
        .env("CONTROL_REPOSITORY", model.parent().unwrap())
        .env("CONTROL_MARKER", &marker)
        .env_remove("UMPIRE_GATE_RECEIPTS")
        .env_remove("UMPIRE_GATE_SCOPE")
        .env_remove("UMPIRE_GATE_STAGE")
        .env_remove("BASH_ENV")
        .output()
        .unwrap();
    assert_eq!(output.status.code(), Some(37), "{output:?}");
    assert_eq!(fs::read_to_string(marker).unwrap(), "consumer timer reached\n");
    assert!(String::from_utf8_lossy(&output.stdout).contains("gate command:"));
    assert!(!String::from_utf8_lossy(&output.stderr).contains("managed gate needs"));
}

#[test]
fn public_fmt_finishes_while_host_lease_is_held() {
    let native = std::env::var_os("UMPIRE_CONTROL_FRAMEWORK_XTASK")
        .map_or_else(|| xtask_binary().clone(), PathBuf::from);
    let f = Fixture::new();
    support::executable(
        &f.root.join("bin/mise"),
        r#"#!/bin/bash
set -euo pipefail
if [[ " $* " == *' cargo build '* ]]; then
  printf '%s\n' bootstrap-entered >> "$CONTROL_PHASE_LOG"
  previous=""
  for argument in "$@"; do
    [[ "$previous" != --target-dir ]] || bootstrap="$argument"
    previous="$argument"
  done
  mkdir -p "$bootstrap/debug"
  ln -sf "$CONTROL_NATIVE" "$bootstrap/debug/xtask"
  cp "$(type -P true)" "$bootstrap/debug/umpire-guest-driver"
  printf '%s\n' bootstrap-returning >> "$CONTROL_PHASE_LOG"
  exit 0
fi
[[ "$PWD" = "$CONTROL_FRAMEWORK" ]]
[[ "$*" = 'exec -- cargo fmt --all' ]]
printf '%s\n' fmt-entered >> "$CONTROL_PHASE_LOG"
printf '%s\n' "$*" >> "$CONTROL_LOG"
printf '%s\n' fmt-returning >> "$CONTROL_PHASE_LOG"
exit "$CONTROL_STATUS"
"#,
    );
    // Returns whether the child exited before `deadline`, its output, phases and elapsed time.
    let run = |label: &str, expected: i32, log: &Path, deadline: std::time::Duration| {
        let phases = f.root.join(format!("fmt-phases-{label}"));
        fs::write(&phases, b"").unwrap();
        let started = std::time::Instant::now();
        let mut child = Command::new(env!("CARGO_BIN_EXE_temporal-xtask"))
            .arg("fmt")
            .current_dir(&f.model)
            .env_remove("BASH_ENV")
            .env_remove("BUILD_CACHE_MANAGED")
            .env_remove("BUILD_CACHE_TOOL")
            .env_remove("UMPIRE_JVM_LEASE_FD")
            .env("PATH", format!("{}:/usr/bin:/bin", f.root.join("bin").display()))
            .env("CARGO_HOME", f.root.join("cargo home"))
            .env("CARGO_TARGET_DIR", f.root.join("target"))
            .env("CONTROL_FRAMEWORK", framework_root().canonicalize().unwrap())
            .env("CONTROL_NATIVE", &native)
            .env("CONTROL_LOG", log)
            .env("CONTROL_PHASE_LOG", &phases)
            .env("CONTROL_STATUS", expected.to_string())
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped())
            .spawn()
            .unwrap();
        let completed = loop {
            if child.try_wait().unwrap().is_some() {
                break true;
            }
            if started.elapsed() >= deadline {
                child.kill().unwrap();
                break false;
            }
            std::thread::sleep(std::time::Duration::from_millis(10));
        };
        let elapsed = started.elapsed();
        let output = child.wait_with_output().unwrap();
        (completed, output, fs::read_to_string(phases).unwrap(), elapsed)
    };
    // A lease wait never ends while this test holds the lease, so any bound separates it from
    // completion. Scale the bound from the same command run without the held lease, so host
    // load slows both alike; a healthy run returns as soon as it exits.
    let unbounded = std::time::Duration::from_secs(300);
    let (completed, output, phases, reference) =
        run("reference", 0, &f.root.join("reference-calls"), unbounded);
    assert!(completed && output.status.code() == Some(0), "{output:?}; phases={phases:?}");
    let deadline = std::time::Duration::from_secs(30) + reference * 4;
    let _lease = xtask::gate::lease::JvmLease::inherited_or_acquire().unwrap();
    for expected in [0, 37] {
        let (completed, output, phases, elapsed) =
            run(&expected.to_string(), expected, &f.root.join("fmt-calls"), deadline);
        assert!(
            completed,
            "public bridge fmt waited for the unrelated host lease: {output:?}; expected={expected}; public_exit_before_deadline={completed}; elapsed={elapsed:?}; unleased_reference={reference:?}; phases={phases:?}"
        );
        assert_eq!(
            output.status.code(),
            Some(expected),
            "{output:?}; expected={expected}; public_exit_before_deadline={completed}; phases={phases:?}"
        );
    }
    assert_eq!(
        fs::read_to_string(f.root.join("fmt-calls")).unwrap(),
        "exec -- cargo fmt --all\nexec -- cargo fmt --all\n"
    );
}

#[test]
fn actual_make_freshness_recipe_propagates_stale_failure() {
    let f = Fixture::new();
    fs::copy(f.model.join("Cargo.lock"), f.model.parent().unwrap().join("Cargo.lock")).unwrap();
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let makefile = fs::read_to_string(source.join("Makefile")).unwrap();
    let recipe = makefile
        .lines()
        .find(|line| {
            line.starts_with('\t')
                && line.contains("--bin temporal-xtask")
                && line.ends_with("-- gate")
        })
        .unwrap();
    fs::write(
        f.model.join("../freshness-control.mk"),
        format!("RUST_MODEL_CARGO := mise exec -- cargo\n.PHONY: freshness-control\nfreshness-control:\n{recipe}\n"),
    )
    .unwrap();
    let output = f.model.join("hosts/go/umpire_temporal_guest.wasm");
    fs::create_dir_all(output.parent().unwrap()).unwrap();
    fs::write(&output, b"keep").unwrap();
    fs::rename(f.root.join("bin/mise"), f.root.join("bin/real-mise")).unwrap();
    support::executable(
        &f.root.join("bin/mise"),
        r#"#!/bin/bash
set -euo pipefail
if [[ "$*" == *'--bin temporal-xtask --profile dev -- gate' ]]; then
  printf 'temporal gate freshness delegation\n' >> "$FIXTURE/gate-called"
  if "$NATIVE_DRIVER" guest-freshness umpire-temporal-guest "$PWD/model/hosts/go/umpire_temporal_guest.wasm"; then
    exit 0
  fi
  exit 37
fi
exec "$FIXTURE/bin/real-mise" "$@"
"#,
    );
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
    assert_eq!(result.status.code(), Some(2));
    assert_eq!(
        fs::read_to_string(f.root.join("gate-called")).unwrap(),
        "temporal gate freshness delegation\n"
    );
    assert!(
        String::from_utf8_lossy(&result.stderr).contains("stale"),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert!(String::from_utf8_lossy(&result.stderr).contains("Error 37"));
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
    fs::copy(source.join("hosts/go/guest.go"), module.join("guest.go")).unwrap();
    fs::write(module.join("go.mod"), "module embedded-guest-control\n\ngo 1.27.0\n").unwrap();
    let output = Command::new(String::from_utf8(go.stdout).unwrap().trim())
        .args(["test", "-tags", "test_dep", "./..."])
        .current_dir(module)
        .env("GOTOOLCHAIN", "local")
        .env("GOWORK", "off")
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
