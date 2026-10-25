//! Compiled gate dispatch controls with synthetic leaf tools and real source scanners.
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{
    fs,
    os::unix::fs::{PermissionsExt, symlink},
    path::{Path, PathBuf},
    process::{Command, Output},
};

const STAGES: &[&str] = &[
    "retired-names",
    "timing-controls",
    "capability-controls",
    "source-boundary",
    "protobuf-controls",
    "protobuf-freshness",
    "kettle-protobuf-freshness",
    "fmt",
    "clippy",
    "docs",
    "tests",
    "wasm32-wasip1",
    "core-wasm-clippy",
    "wasm-guest",
    "go-host",
    "kettle-go-host",
];
const LEAVES: &[(&str, &str)] = &[
    ("proto-tools", "proto-tools"),
    ("timing-controls", "timing-controls"),
    ("timing-controls", "guest-controls"),
    ("capability-controls", "capability-controls"),
    ("source-boundary", "source-boundary"),
    ("protobuf-controls", "protobuf-controls"),
    ("protobuf-freshness", "protobuf-freshness"),
    ("kettle-protobuf-freshness", "kettle-protobuf-freshness"),
    ("fmt", "fmt"),
    ("clippy", "clippy"),
    ("docs", "docs"),
    ("tests", "tests"),
    ("wasm32-wasip1", "wasm32-wasip1"),
    ("core-wasm-clippy", "core-wasm-clippy"),
    ("wasm-guest", "wasm-guest"),
    ("go-host", "go-host"),
    ("kettle-go-host", "kettle-go-host"),
];

struct Fixture {
    _directory: tempfile::TempDir,
    root: PathBuf,
    caller: PathBuf,
    bin: PathBuf,
    log: PathBuf,
}
fn write(path: &Path, content: impl AsRef<[u8]>) {
    fs::create_dir_all(path.parent().unwrap()).unwrap();
    fs::write(path, content).unwrap();
}
impl Fixture {
    fn new() -> Self {
        let directory = tempfile::tempdir().unwrap();
        let base = fs::canonicalize(directory.path()).unwrap();
        let fixture = Self {
            root: base.join("framework"),
            caller: base.join("caller"),
            bin: base.join("bin"),
            log: base.join("calls"),
            _directory: directory,
        };
        write(&fixture.root.join("Cargo.toml"), "[workspace]\nmembers = []\n");
        write(&fixture.root.join("mise.toml"), "[tools]\n");
        write(&fixture.root.join("rust-toolchain.toml"), "[toolchain]\nchannel = 'stable'\n");
        write(
            &fixture.root.join("dev/gate/policy/framework-boundary.txt"),
            "version = 1\nforbidden = ['forbidden_payload']\nretired_pattern = 'retired_payload'\norganization_prefix = 'github.com/acme/'\ncontrols = ['forbidden_payload']\n",
        );
        fs::create_dir_all(fixture.caller.join("real home")).unwrap();
        symlink("real home", fixture.caller.join("cargo home")).unwrap();
        for (script, label) in [
            ("proto-tools.sh", "proto-tools"),
            ("test-time-gate.sh", "timing-controls"),
            ("tests/build-guests-controls.sh", "guest-controls"),
            ("test-capabilities.sh", "capability-controls"),
            ("tests/proto-go.sh", "protobuf-controls"),
            ("tests/build-guests.sh", "wasm-guest"),
        ] {
            Self::script(&fixture.root.join("scripts").join(script), &format!("label={label}\n"));
        }
        Self::script(
            &fixture.root.join("scripts/proto-go.sh"),
            "case $1 in core) label=protobuf-freshness ;; kettle) label=kettle-protobuf-freshness ;; *) exit 90 ;; esac\n[ \"$2\" = --check ]\n",
        );
        Self::script(
            &fixture.bin.join("mise"),
            r#"[ "$1" = exec ] && [ "$2" = -- ]
shift 2
case "$1 $2" in
 'cargo metadata') label=source-boundary ;;
 'cargo fmt') label=fmt ;;
 'cargo doc') label=docs; [ "$RUSTDOCFLAGS" = '-D warnings' ] ;;
 'cargo test') label=tests ;;
 'cargo build') label=wasm32-wasip1 ;;
 'cargo clippy') case " $* " in *' --target '*) label=core-wasm-clippy ;; *) label=clippy ;; esac ;;
 'go -C') [ "$CGO_ENABLED" = 0 ]; case "$3" in "$CONTROL_ROOT/hosts/go") label=go-host ;; "$CONTROL_ROOT/reference/kettle") label=kettle-go-host ;; *) exit 91 ;; esac
 [ "$UMPIRE_WASM" = "$CONTROL_CALLER/relative target/wasm32-wasip1/guest/umpire_stock_guest.wasm" ]
 [ "$UMPIRE_CURRENT_CONTRACT_NATIVE" = "$CONTROL_CALLER/relative target/current-contract-native" ] ;;
 *) exit 92 ;;
esac
"#,
        );
        Self::script(&fixture.bin.join("curl"), "label=apalache-download\n");
        fixture
    }
    fn script(path: &Path, body: &str) {
        write(
            path,
            format!(
                r#"#!/bin/bash
set -euo pipefail
{body}
[ "$PWD" = "$CONTROL_ROOT" ]
[ "$CARGO_TARGET_DIR" = "$CONTROL_CALLER/relative target" ]
[ "$CARGO_HOME" = "$CONTROL_CALLER/real home" ]
[ "$UMPIRE_QUICK_CACHE" = 0 ] && [ "$UMPIRE_CACHE_DISABLE" = 1 ]
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
printf '%s|%s|%s|%s|%s|%s|%s|%s\n' "$label" "$PWD" "$CARGO_TARGET_DIR" "$CARGO_HOME" "$UMPIRE_QUICK_CACHE" "$UMPIRE_CACHE_DISABLE" "${{UMPIRE_BACKENDS:-unset}}" "$*" >> "$CONTROL_LOG"
if [ "${{CONTROL_SIGNAL:-}}" = "$label" ]; then
 printf 'signal stdout diagnostic\n'
 printf 'signal stderr diagnostic\n' >&2
 kill -TERM $$
fi
[ "${{CONTROL_FAIL:-}}" != "$label" ] || {{ printf '%s child diagnostic\n' "$label" >&2; exit 37; }}
if [ "${{CONTROL_EXPECT_UPDATE:-0}}" = 1 ]; then
 [ "$INSTA_UPDATE" = always ] && [ "$TRYBUILD" = overwrite ]
fi
if [ "${{CONTROL_FAIL_INNER:-0}}" = 1 ] && [ "$*" = 'cargo test -p umpire-checker --quiet' ]; then
 printf 'inner child diagnostic\n' >&2
 exit 37
fi
if [ "$label" = source-boundary ]; then
 printf '{{"packages":[{{"source":null,"manifest_path":"%s/Cargo.toml"}}]}}\n' "$CONTROL_ROOT"
fi
"#
            ),
        );
        fs::set_permissions(path, fs::Permissions::from_mode(0o755)).unwrap();
    }
    fn command(&self, timed: bool) -> Command {
        let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
        command
            .current_dir(&self.caller)
            .env_remove("BASH_ENV")
            .env_remove("UMPIRE_BACKENDS")
            .env_remove("UMPIRE_APALACHE")
            .env("PATH", format!("{}:/usr/bin:/bin", self.bin.display()))
            .env("UMPIRE_TOOL_ROOT", &self.root)
            .env("CONTROL_ROOT", &self.root)
            .env("CONTROL_CALLER", &self.caller)
            .env("CONTROL_LOG", &self.log)
            .env("CARGO_TARGET_DIR", "relative target")
            .env("CARGO_HOME", "cargo home")
            .env("UMPIRE_TIME_STEPS", if timed { "1" } else { "0" })
            .env("UMPIRE_QUICK_CACHE", "1")
            .env("UMPIRE_CACHE_DISABLE", "0");
        command
    }
    fn labels(&self) -> Vec<String> {
        fs::read_to_string(&self.log)
            .unwrap_or_default()
            .lines()
            .map(|line| line.split('|').next().unwrap().into())
            .collect()
    }
    fn clear(&self) {
        write(&self.log, "");
    }
}
fn stderr(output: &Output) -> String {
    String::from_utf8_lossy(&output.stderr).into_owned()
}
fn assert_status(output: &Output, code: i32) {
    assert_eq!(output.status.code(), Some(code), "{output:?}");
}

#[test]
fn compiled_stage_inventory_matches_preserved_framework_stages() {
    let output = Fixture::new().command(false).args(["gate", "--list"]).output().unwrap();
    assert_status(&output, 0);
    let text = String::from_utf8(output.stdout).unwrap();
    let names: Vec<_> = text.lines().map(|line| line.split('\t').next().unwrap()).collect();
    assert_eq!(names, STAGES);
    assert!(text.contains("protobuf-controls\tproto-tools\n"));
    assert!(text.contains("go-host\twasm-guest\n"));
    assert!(text.contains("kettle-go-host\twasm-guest,kettle-protobuf-freshness\n"));
}

#[test]
fn every_command_boundary_preserves_failure_status_and_environment_in_both_modes() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for (stage, leaf) in LEAVES {
            fixture.clear();
            let output = fixture
                .command(timed)
                .args(["gate", "--stage", stage])
                .env("CONTROL_FAIL", leaf)
                .output()
                .unwrap();
            assert_status(&output, 37);
            assert!(
                stderr(&output).contains(&format!("framework stage {stage} failed (exit 37)")),
                "{output:?}"
            );
            assert!(stderr(&output).contains(&format!("{leaf} child diagnostic")), "{output:?}");
            let expected = if *leaf == "guest-controls" {
                vec!["timing-controls", *leaf]
            } else {
                vec![*leaf]
            };
            assert_eq!(fixture.labels(), expected, "stage {stage}, timed={timed}");
        }
    }
}

#[test]
fn full_dispatch_runs_each_leaf_once_in_order_and_bypasses_cache_from_first_command() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let output = fixture.command(timed).arg("gate").output().unwrap();
        assert_status(&output, 0);
        assert_eq!(fixture.labels(), LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>());
        assert!(String::from_utf8_lossy(&output.stdout).contains("UMPIRE_APALACHE=skip (default)"));
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert!(log.lines().next().unwrap().contains("|0|1|unset|"));
        assert!(log.contains("cargo test --workspace --locked --quiet"));
        assert!(log.contains(
            "cargo build --target wasm32-wasip1 -p umpire-dsl -p umpire-checker --quiet"
        ));
        assert!(log.contains("cargo clippy --target wasm32-wasip1 -p umpire-dsl -p umpire-checker --lib --locked -- -D warnings"));
        assert!(log.contains("test -tags test_dep -count=1 ./..."));
    }
}

#[test]
fn full_dispatch_stops_at_every_failed_leaf_in_both_modes() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for (index, (stage, leaf)) in LEAVES.iter().enumerate() {
            fixture.clear();
            let output =
                fixture.command(timed).arg("gate").env("CONTROL_FAIL", leaf).output().unwrap();
            assert_status(&output, 37);
            assert!(
                stderr(&output).contains(&format!("framework stage {stage} failed (exit 37)")),
                "{output:?}"
            );
            assert_eq!(
                fixture.labels(),
                LEAVES[..=index].iter().map(|(_, name)| *name).collect::<Vec<_>>()
            );
        }
    }
}

#[test]
fn scanner_stages_reject_real_planted_inputs_and_full_gate_stops_before_later_commands() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for (stage, payload, reason, expected_labels) in [
            (
                "source-boundary",
                "forbidden_payload",
                "framework forbidden name: planted:1",
                vec!["proto-tools", "timing-controls", "guest-controls", "capability-controls"],
            ),
            ("retired-names", "retired_payload", "retired", vec!["proto-tools"]),
        ] {
            write(&fixture.root.join("planted"), payload);
            fixture.clear();
            let output = fixture.command(timed).args(["gate", "--stage", stage]).output().unwrap();
            assert_status(&output, 1);
            assert!(stderr(&output).contains(reason), "{output:?}");
            assert_eq!(fixture.labels(), Vec::<String>::new());
            let output = fixture.command(timed).arg("gate").output().unwrap();
            assert_status(&output, 1);
            assert!(stderr(&output).contains(reason), "{output:?}");
            assert_eq!(fixture.labels(), expected_labels);
            fs::remove_file(fixture.root.join("planted")).unwrap();
        }
    }
    write(
        &fixture.root.join("dev/gate/policy/framework-boundary.txt"),
        "version = 1\nunknown = true\n",
    );
    for stage in ["source-boundary", "retired-names"] {
        let output = fixture.command(false).args(["gate", "--stage", stage]).output().unwrap();
        assert_status(&output, 1);
        assert!(stderr(&output).contains(stage), "{output:?}");
        assert!(stderr(&output).contains("unknown field"), "{output:?}");
    }
}

#[test]
fn required_optional_backend_fetch_failure_is_named_and_prevents_first_stage() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let output = fixture
            .command(timed)
            .arg("gate")
            .env("UMPIRE_APALACHE", "require")
            .env("CONTROL_FAIL", "apalache-download")
            .output()
            .unwrap();
        assert_status(&output, 1);
        assert!(stderr(&output).contains("Apalache download failed"), "{output:?}");
        assert!(stderr(&output).contains("37"), "{output:?}");
        assert!(stderr(&output).contains("framework stage apalache failed (exit 1)"), "{output:?}");
        assert_eq!(fixture.labels(), ["apalache-download"]);
    }
}

#[test]
fn validate_forces_backend_mode_and_time_entries_stop_before_benchmark_after_failed_gate() {
    let fixture = Fixture::new();
    write(&fixture.root.join("backends/native/src/lib.rs"), "// benchmark input\n");
    for entry in ["time", "validate"] {
        fixture.clear();
        let before = fs::metadata(fixture.root.join("backends/native/src/lib.rs"))
            .unwrap()
            .modified()
            .unwrap();
        let output = fixture
            .command(false)
            .arg(entry)
            .env("UMPIRE_BACKENDS", "skip")
            .env("CONTROL_FAIL", "tests")
            .output()
            .unwrap();
        assert_status(&output, 37);
        assert!(stderr(&output).contains("framework stage tests failed (exit 37)"), "{output:?}");
        assert_eq!(fixture.labels().last().unwrap(), "tests");
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert!(!log.contains("cargo test -p"));
        assert_eq!(
            fs::metadata(fixture.root.join("backends/native/src/lib.rs"))
                .unwrap()
                .modified()
                .unwrap(),
            before
        );
        if entry == "validate" {
            assert!(log.lines().all(|line| line.contains("|0|1|require|")), "{log}");
        }
        fixture.clear();
        let output = fixture.command(false).arg(entry).output().unwrap();
        assert_status(&output, 0);
        let mut expected = LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>();
        expected.push("tests");
        assert_eq!(fixture.labels(), expected);
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert!(log.lines().last().unwrap().ends_with("cargo test -p umpire-checker --quiet"));
        if entry == "validate" {
            assert!(log.lines().all(|line| line.contains("|0|1|require|")), "{log}");
        }
    }
}

#[test]
fn convenience_entries_keep_exact_arguments_and_propagate_failed_inner_status() {
    let fixture = Fixture::new();
    for (entry, expected) in
        [("fmt", "cargo fmt --all"), ("update", "cargo test --workspace --locked --quiet")]
    {
        fixture.clear();
        let mut command = fixture.command(false);
        command.arg(entry);
        if entry == "update" {
            command.env("CONTROL_EXPECT_UPDATE", "1");
        }
        let output = command.output().unwrap();
        assert_status(&output, 0);
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert_eq!(log.lines().count(), 1);
        assert!(log.lines().next().unwrap().ends_with(expected), "{log}");
    }
    write(&fixture.root.join("backends/native/src/lib.rs"), "// benchmark input\n");
    for entry in ["time", "validate"] {
        fixture.clear();
        let output = fixture
            .command(false)
            .arg(entry)
            .env("CONTROL_FAIL_INNER", "1")
            .env("UMPIRE_GATE_BUDGET", "-1")
            .env("UMPIRE_INNER_BUDGET", "-1")
            .output()
            .unwrap();
        assert_status(&output, 37);
        assert!(
            stderr(&output).contains("framework stage inner loop (edit+test) failed (exit 37)"),
            "{output:?}"
        );
        assert!(stderr(&output).contains("inner child diagnostic"), "{output:?}");
        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(!stdout.contains("gate total"), "{output:?}");
        assert!(!stdout.contains("over soft target"), "{output:?}");
        assert_eq!(fixture.labels().len(), LEAVES.len() + 1);
    }
}

#[test]
fn signaled_stage_retains_shell_status_diagnostics_and_stops_dispatch() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for full in [false, true] {
            fixture.clear();
            let mut command = fixture.command(timed);
            command.arg("gate").env("CONTROL_SIGNAL", "capability-controls");
            if !full {
                command.args(["--stage", "capability-controls"]);
            }
            let output = command.output().unwrap();
            assert_status(&output, 143);
            assert!(
                stderr(&output).contains("framework stage capability-controls failed (exit 143)"),
                "{output:?}"
            );
            let diagnostics =
                format!("{}{}", String::from_utf8_lossy(&output.stdout), stderr(&output));
            assert!(diagnostics.contains("signal stdout diagnostic"), "{output:?}");
            assert!(diagnostics.contains("signal stderr diagnostic"), "{output:?}");
            let expected = if full {
                vec!["proto-tools", "timing-controls", "guest-controls", "capability-controls"]
            } else {
                vec!["capability-controls"]
            };
            assert_eq!(fixture.labels(), expected);
        }
    }
}
