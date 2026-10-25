//! Native guest build and cold shell entrypoint controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{
    fs,
    io::Write,
    os::unix::fs::PermissionsExt,
    path::{Path, PathBuf},
    process::{Command, Output},
    sync::atomic::{AtomicU64, Ordering},
};

struct Fixture {
    root: PathBuf,
    model: PathBuf,
}
impl Fixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "native guest controls {} {}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let model = root.join("checkout with spaces/model");
        fs::create_dir_all(model.join("scripts")).unwrap();
        let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
        fs::copy(source.join("Cargo.lock"), model.join("Cargo.lock")).unwrap();
        fs::copy(source.join("rust-toolchain.toml"), model.join("rust-toolchain.toml")).unwrap();
        fs::copy(source.join("../mise.toml"), model.join("../mise.toml")).unwrap();
        for name in ["guest.sh", "guest-freshness.sh", "xtask.sh"] {
            if source.join("scripts").join(name).exists() {
                fs::copy(source.join("scripts").join(name), model.join("scripts").join(name))
                    .unwrap();
            }
        }
        fs::write(root.join("built custom.wasm"), b"\0asm\x01\0\0\0").unwrap();
        let pins: toml::Value =
            toml::from_str(&fs::read_to_string(model.join("../mise.toml")).unwrap()).unwrap();
        for (tool, names) in
            [("rust", vec!["rustc", "cargo"]), ("go", vec!["go"]), ("protoc", vec!["protoc"])]
        {
            let directory = root.join(tool).join("bin");
            fs::create_dir_all(&directory).unwrap();
            for name in names {
                executable(&directory.join(name), "#!/bin/sh\nexit 0\n");
            }
            fs::write(root.join(format!("{tool}.json")), serde_json::to_vec(&serde_json::json!([{"version": pins["tools"][tool].as_str().unwrap(), "install_path": root.join(tool), "installed":true,"active":true}])).unwrap()).unwrap();
        }
        let lib = root.join("rust/lib/rustlib");
        fs::create_dir_all(lib.join("src/rust/library")).unwrap();
        fs::write(lib.join("src/rust/library/Cargo.toml"), "[workspace]\n").unwrap();
        fs::create_dir_all(lib.join("wasm32-wasip1/lib")).unwrap();
        fs::write(lib.join("wasm32-wasip1/lib/libstd-control.rlib"), "").unwrap();
        fs::create_dir_all(root.join("bin")).unwrap();
        executable(
            &root.join("bin/mise"),
            r#"#!/bin/bash
set -eu
[[ "$MISE_AUTO_INSTALL" = false && "$MISE_EXEC_AUTO_INSTALL" = false ]] || { echo auto-install-enabled >&2; exit 98; }
printf '%s\n' "$*" >> "$FIXTURE/calls"
if [[ "$1" = ls ]]; then
  [[ "${FAIL_PROBE:-}" != 1 ]] || { echo probe-failed >&2; exit 23; }
  [[ "${BAD_METADATA:-}" != 1 ]] || { echo '[{}]'; exit; }
  [[ "${MISSING:-}" != "$4" ]] || { echo '[]'; exit; }
  cat "$FIXTURE/$4.json"; exit
fi
[[ "$1 $2" = 'exec --' ]]
shift 2
if [[ "$1" = rustc ]]; then
  [[ "$2" != --print ]] || { echo "$FIXTURE/rust"; exit; }
  echo 'rustc 1.99.1 (58cf8f9c3 2026-10-13)'; exit
fi
[[ "$1" = cargo ]]
if [[ "$2" = metadata ]]; then
  [[ "${FAIL_METADATA:-}" != 1 ]] || { echo metadata-failed >&2; exit 41; }
  echo '{"packages":[],"resolve":{"nodes":[]}}'; exit
fi
if [[ " $* " = *' -p xtask '* ]]; then
  [[ -z "${RUSTC_WRAPPER:-}${CARGO_ENCODED_RUSTFLAGS:-}${RUSTFLAGS:-}${RUSTC_BOOTSTRAP:-}" ]] || { echo contaminated-bootstrap >&2; exit 97; }
  [[ "${FAIL_BOOTSTRAP:-}" != 1 ]] || { echo bootstrap-compiler-failed >&2; exit 39; }
  bootstrap_target="$CARGO_TARGET_DIR"
  previous=""
  for argument in "$@"; do
    [[ "$previous" != --target-dir ]] || bootstrap_target="$argument"
    previous="$argument"
  done
  mkdir -p "$bootstrap_target/debug"
  cp "$NATIVE_XTASK" "$bootstrap_target/debug/xtask"
  printf '{"reason":"compiler-artifact","target":{"name":"xtask","kind":["bin"],"crate_types":["bin"]},"executable":"%s/debug/xtask","filenames":[]}\n' "$CARGO_TARGET_DIR"
  exit
fi
: > "$FIXTURE/guest-built"
printf '%s\n' "$CARGO_TARGET_DIR" > "$FIXTURE/guest-cache"
[[ -f "$RUSTC_WRAPPER" && "$RUSTC_WRAPPER" = /* ]]
[[ "$RUSTC_WRAPPER" = */guest-rustc ]]
[[ "$UMPIRE_GUEST_PREVIOUS_WRAPPER" = "$FIXTURE/previous wrapper" ]]
[[ "$CARGO_MAKEFLAGS" = 'jobserver-marker' ]]
[[ "$CARGO_ENCODED_RUSTFLAGS" = *'-Cpanic=unwind'* ]]
[[ "$CARGO_ENCODED_RUSTFLAGS" = *'--remap-path-prefix='* ]]
DYLD_FALLBACK_LIBRARY_PATH=loader-marker FIXTURE_EXPECT_LOADER=loader-marker \
FIXTURE_PREVIOUS="$FIXTURE/previous-called" FIXTURE_OUTPUT="$FIXTURE/loader.json" \
"$RUSTC_WRAPPER" "$FIXTURE/compiler probe" -V
[[ "${FAIL_BUILD:-}" != 1 ]] || { echo guest-compiler-failed >&2; exit 37; }
case "${ARTIFACT_MODE:-one}" in
 malformed) echo '{';;
 missing) echo '{"reason":"build-finished","success":true}';;
 multiple) printf '{"reason":"compiler-artifact","target":{"crate_types":["cdylib"]},"filenames":["%s/built custom.wasm","%s/other.wasm"]}\n' "$FIXTURE" "$FIXTURE";;
 *) printf '{"reason":"compiler-artifact","target":{"crate_types":["cdylib"]},"filenames":["%s/built custom.wasm"]}\n' "$FIXTURE";;
esac
"#,
        );
        for name in ["compiler probe", "previous wrapper"] {
            fs::copy(env!("CARGO_BIN_EXE_identity-compiler-fixture"), root.join(name)).unwrap();
        }
        Self { root, model }
    }
    fn command(&self, freshness: bool, shell: bool) -> Command {
        let mut command = if shell {
            let mut command = Command::new("/bin/bash");
            command.arg(self.model.join("scripts").join(if freshness {
                "guest-freshness.sh"
            } else {
                "guest.sh"
            }));
            command
        } else {
            let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
            command.arg(if freshness { "guest-freshness" } else { "guest" });
            command
        };
        command
            .current_dir(&self.model)
            .arg("example-guest")
            .arg(self.root.join("out.wasm"))
            .env_remove("BASH_ENV")
            .env("FIXTURE", &self.root)
            .env("NATIVE_XTASK", env!("CARGO_BIN_EXE_xtask"))
            .env("PATH", format!("{}:/usr/bin:/bin", self.root.join("bin").display()))
            .env("CARGO_TARGET_DIR", self.root.join("custom target"))
            .env("CARGO_HOME", self.root.join("cargo home"))
            .env("RUSTC_WRAPPER", self.root.join("previous wrapper"))
            .env("CARGO_MAKEFLAGS", "jobserver-marker")
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false")
            .env_remove("RUSTFLAGS")
            .env_remove("CARGO_ENCODED_RUSTFLAGS")
            .env_remove("RUSTC_BOOTSTRAP");
        command
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}
fn executable(path: &Path, contents: &str) {
    fs::write(path, contents).unwrap();
    fs::set_permissions(path, fs::Permissions::from_mode(0o755)).unwrap();
}
fn code(output: &Output, expected: i32) {
    assert_eq!(
        output.status.code(),
        Some(expected),
        "stdout={}\nstderr={}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
}

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
        fs::write(f.model.join("../mise.toml"), contents).unwrap();
        code(&f.command(true, true).output().unwrap(), 1);
        assert!(!f.root.join("guest-built").exists());
    }
}

#[test]
fn actual_make_freshness_recipe_propagates_stale_failure() {
    let f = Fixture::new();
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
    command.env("CARGO_TARGET_DIR", "relative target");
    code(&command.output().unwrap(), 0);
    assert_eq!(fs::read(f.root.join("relative output.wasm")).unwrap(), b"\0asm\x01\0\0\0");
    let first = fs::read(f.root.join("guest-cache")).unwrap();
    fs::OpenOptions::new()
        .append(true)
        .open(f.model.join("Cargo.lock"))
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
