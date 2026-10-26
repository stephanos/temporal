//! Native guest build and cold shell entrypoint controls.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{
    fs,
    os::unix::fs::PermissionsExt,
    path::{Path, PathBuf},
    process::{Command, Output},
    sync::atomic::{AtomicU64, Ordering},
};

pub(crate) struct Fixture {
    pub root: PathBuf,
    pub model: PathBuf,
}
impl Fixture {
    #[allow(clippy::too_many_lines)]
    pub(crate) fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "native guest controls {} {}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let model = root.join("checkout with spaces/model");
        fs::create_dir_all(model.join("scripts")).unwrap();
        let source = crate::framework_root();
        fs::copy(source.join("Cargo.lock"), model.join("Cargo.lock")).unwrap();
        fs::copy(source.join("Cargo.toml"), model.join("Cargo.toml")).unwrap();
        fs::copy(source.join("rust-toolchain.toml"), model.join("rust-toolchain.toml")).unwrap();
        fs::copy(source.join("mise.toml"), model.join("mise.toml")).unwrap();
        for name in ["guest.sh", "guest-freshness.sh", "xtask.sh"] {
            if source.join("scripts").join(name).exists() {
                fs::copy(source.join("scripts").join(name), model.join("scripts").join(name))
                    .unwrap();
            }
        }
        fs::write(root.join("built custom.wasm"), b"\0asm\x01\0\0\0").unwrap();
        fs::create_dir_all(root.join("bin")).unwrap();
        executable(
            &root.join("bin/mise"),
            r#"#!/bin/bash
set -eu
[[ "$MISE_AUTO_INSTALL" = false && "$MISE_EXEC_AUTO_INSTALL" = false ]] || { echo auto-install-enabled >&2; exit 98; }
printf '%s\n' "$*" >> "$FIXTURE/calls"
if [[ -n "${EXPECTED_CARGO_HOME:-}" ]]; then
  [[ "$CARGO_HOME" = "$EXPECTED_CARGO_HOME" ]] || { echo "wrong-cargo-home: $CARGO_HOME from $PWD" >&2; exit 96; }
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
if [[ " $* " = *' -p xtask '* || " $* " = *' -p umpire-guest-driver '* ]]; then
  [[ -z "${RUSTC_WRAPPER:-}${CARGO_ENCODED_RUSTFLAGS:-}${RUSTFLAGS:-}${RUSTC_BOOTSTRAP:-}" ]] || { echo contaminated-bootstrap >&2; exit 97; }
  [[ "${FAIL_BOOTSTRAP:-}" != 1 ]] || { echo bootstrap-compiler-failed >&2; exit 39; }
  bootstrap_target="$CARGO_TARGET_DIR"
  previous=""
  for argument in "$@"; do
    [[ "$previous" != --target-dir ]] || bootstrap_target="$argument"
    previous="$argument"
  done
  mkdir -p "$bootstrap_target/debug"
  cp "$NATIVE_DRIVER" "$bootstrap_target/debug/umpire-guest-driver"
  cp "$NATIVE_DRIVER" "$bootstrap_target/debug/xtask"
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
            fs::copy(crate::compiler_fixture(), root.join(name)).unwrap();
        }
        Self { root, model }
    }
    pub(crate) fn command(&self, freshness: bool, shell: bool) -> Command {
        let mut command = if shell {
            let mut command = Command::new("/bin/bash");
            command.arg(self.model.join("scripts").join(if freshness {
                "guest-freshness.sh"
            } else {
                "guest.sh"
            }));
            command
        } else {
            let mut command = Command::new(crate::driver_binary());
            command.arg(if freshness { "guest-freshness" } else { "guest" });
            command
        };
        command
            .current_dir(&self.model)
            .arg("example-guest")
            .arg(self.root.join("out.wasm"))
            .env_remove("BASH_ENV")
            .env("FIXTURE", &self.root)
            .env("NATIVE_DRIVER", crate::driver_binary())
            .env("UMPIRE_TOOL_ROOT", &self.model)
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
pub(crate) fn executable(path: &Path, contents: &str) {
    fs::write(path, contents).unwrap();
    fs::set_permissions(path, fs::Permissions::from_mode(0o755)).unwrap();
}
pub(crate) fn code(output: &Output, expected: i32) {
    assert_eq!(
        output.status.code(),
        Some(expected),
        "stdout={}\nstderr={}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
}
