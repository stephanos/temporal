//! Compiled gate dispatch controls with synthetic leaf tools and real source scanners.
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::{
    fs,
    os::unix::fs::{PermissionsExt, symlink},
    path::{Path, PathBuf},
    process::{Command, Output},
};

pub(super) const STAGES: &[&str] = &[
    "proto-tools",
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
    "standalone",
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
    ("tests", "doctests"),
    ("wasm32-wasip1", "wasm32-wasip1"),
    ("core-wasm-clippy", "core-wasm-clippy"),
    ("wasm-guest", "wasm-guest"),
    ("go-host", "go-host"),
    ("kettle-go-host", "kettle-go-host"),
];

const MISE_SCRIPT: &str = r#"[ "$PWD" = "$CONTROL_ROOT" ]
[ "$CARGO_TARGET_DIR" = "$CONTROL_EXPECT_TARGET" ]
[ "$CARGO_HOME" = "$CONTROL_CALLER/real home" ]
[ "$UMPIRE_QUICK_CACHE" = 0 ] && [ "$UMPIRE_CACHE_DISABLE" = 1 ]
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
if [ "$*" = 'tasks ls --json' ]; then
 if [ -n "${CONTROL_REAL_MISE:-}" ]; then exec "$CONTROL_REAL_MISE" "$@"; fi
 cat "$CONTROL_ROOT/tasks.json"
 exit 0
fi
if [ "$1" = run ]; then
 if [ -n "${CONTROL_REAL_MISE:-}" ]; then
  exec "$CONTROL_REAL_MISE" "$@"
 fi
 [ "$2 $3 $4 $5 $6 $7 $8" = '--force --skip-tools --task-cache off --jobs 4 gate:all' ] ||
 [ "$2 $3 $4 $5 $6 $7 $8" = '--force --skip-tools --task-cache off --jobs 4 gate:child' ]
 aggregate=$8
 [ "$#" = 8 ]
 [ -d "$UMPIRE_GATE_RECEIPTS" ]
 printf '%s\n' "$UMPIRE_GATE_RECEIPTS" >> "$CONTROL_ROOT/invocations"
 case "${CONTROL_RECEIPT_FAULT:-}" in zero) exit 0 ;; esac
 case "${CONTROL_RECEIPT_FAULT:-}" in
  stream)
   printf 'streamed-early-diagnostic\n' >&2
   remaining=500
   while [ ! -f "$CONTROL_ROOT/stream-release" ]; do
    remaining=$((remaining - 1)); [ "$remaining" -gt 0 ] || exit 97; sleep 0.01
   done
   exit 37
   ;;
  simultaneous)
   CONTROL_FAIL=proto-tools "$UMPIRE_GATE_EXECUTABLE" gate --scheduled-stage proto-tools &
   first=$!
   CONTROL_FAIL=fmt "$UMPIRE_GATE_EXECUTABLE" gate --scheduled-stage fmt &
   second=$!
   first_status=0; second_status=0
   wait "$first" || first_status=$?
   wait "$second" || second_status=$?
   [ "$first_status:$second_status" = 37:37 ] || exit 96
   exit 37
   ;;
  orphan)
   /bin/bash -c 'echo $$ > "$CONTROL_ROOT/owned-child"; sleep 5; touch "$CONTROL_ROOT/child-escaped-cleanup"' &
   remaining=500
   while [ ! -s "$CONTROL_ROOT/owned-child" ]; do
    remaining=$((remaining - 1))
    [ "$remaining" -gt 0 ] || exit 97
    sleep 0.01
   done
   CONTROL_FAIL=fmt "$UMPIRE_GATE_EXECUTABLE" gate --scheduled-stage fmt
   exit 98
   ;;
 esac
 for stage in proto-tools retired-names timing-controls capability-controls source-boundary protobuf-controls protobuf-freshness kettle-protobuf-freshness fmt clippy docs tests wasm32-wasip1 core-wasm-clippy wasm-guest go-host kettle-go-host standalone; do
  if [ "$stage" = standalone ] && [ "$aggregate" = gate:child ]; then
   continue
  fi
  UMPIRE_GATE_STAGE="$stage" "$UMPIRE_GATE_EXECUTABLE" gate --scheduled-stage "$stage"
  if [ "${CONTROL_RECEIPT_FAULT:-}" = duplicate ]; then
   UMPIRE_GATE_STAGE="$stage" "$UMPIRE_GATE_EXECUTABLE" gate --scheduled-stage "$stage"
  fi
 done
 case "${CONTROL_RECEIPT_FAULT:-}" in
  partial) printf '{' > "$UMPIRE_GATE_RECEIPTS/fmt.json" ;;
  missing) rm "$UMPIRE_GATE_RECEIPTS/fmt.json" ;;
  extra) cp "$UMPIRE_GATE_RECEIPTS/fmt.json" "$UMPIRE_GATE_RECEIPTS/unexpected.json" ;;
 esac
 exit 0
fi
[ "$1" = exec ] && [ "$2" = -- ]
shift 2
case "$1 $2" in
 'cargo metadata') case " $* " in *' --no-deps '*) label=inner-metadata ;; *) label=source-boundary ;; esac ;;
 'cargo fmt') label=fmt; [ "$*" = "${CONTROL_EXPECT_FMT:-cargo fmt --all --check}" ] ;;
 'cargo doc') label=docs; [ "$*" = 'cargo doc --workspace --no-deps --quiet' ]; [ "$RUSTDOCFLAGS" = '-D warnings' ] ;;
 'cargo nextest') if [ "$3" = list ]; then case " $* " in *' --workspace '*) label=test-inventory ;; *) label=inner-inventory ;; esac; else label=tests; fi ;;
 'cargo test') case " $* " in *' --doc '*) label=doctests ;; *) label=tests ;; esac ;;
 'cargo build') label=wasm32-wasip1 ;;
 'cargo clippy') case " $* " in *' --target '*) label=core-wasm-clippy ;; *) label=clippy; [ "$*" = 'cargo clippy --workspace --all-targets --all-features -- -D warnings' ] ;; esac ;;
 'go -C') [ "$CGO_ENABLED" = 0 ]; case "$3" in "$CONTROL_ROOT/hosts/go") label=go-host ;; "$CONTROL_ROOT/reference/kettle") label=kettle-go-host ;; *) exit 91 ;; esac
 [ "$UMPIRE_WASM" = "$CONTROL_EXPECT_TARGET/wasm32-wasip1/guest/umpire_stock_guest.wasm" ]
 [ "$UMPIRE_KETTLE_WASM" = "$CONTROL_EXPECT_TARGET/wasm32-wasip1/guest/umpire_kettle_guest.wasm" ]
 [ "$UMPIRE_STATEFUL_WASM" = "$CONTROL_EXPECT_TARGET/wasm32-wasip1/guest/stateful_control.wasm" ]
 [ "$UMPIRE_CAUGHT_PANIC_WASM" = "$CONTROL_EXPECT_TARGET/wasm32-wasip1/guest/caught_panic_control.wasm" ]
 [ "$UMPIRE_OVERFLOW_WASM" = "$CONTROL_EXPECT_TARGET/wasm32-wasip1/guest/overflow_control.wasm" ]
 [ "$UMPIRE_PANIC_NATIVE" = "$CONTROL_EXPECT_TARGET/panic-native" ]
 [ "$UMPIRE_CURRENT_CONTRACT_NATIVE" = "$CONTROL_EXPECT_TARGET/current-contract-native" ] ;;
 *) exit 92 ;;
esac
"#;

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
        write(&fixture.root.join("mise.toml"), include_str!("../../../mise.toml"));
        write(
            &fixture.root.join("dev/xtask/src/gate/tasks.toml"),
            include_str!("../src/gate/tasks.toml"),
        );
        let tasks = STAGES.iter().enumerate().map(|(index, name)| {
            let depends = match *name {
                "protobuf-controls" | "protobuf-freshness" | "kettle-protobuf-freshness" => vec!["gate:proto-tools"],
                "go-host" => vec!["gate:wasm-guest"],
                "kettle-go-host" => vec!["gate:wasm-guest", "gate:kettle-protobuf-freshness"],
                _ => vec![],
            };
            serde_json::json!({
                "name": format!("gate:{name}"),
                "env": [format!("UMPIRE_GATE_STAGE={name}"), format!("UMPIRE_GATE_ORDER={}", index + 1)],
                "depends": depends,
            })
        }).collect::<Vec<_>>();
        write(&fixture.root.join("tasks.json"), serde_json::to_vec(&tasks).unwrap());
        write(&fixture.root.join("rust-toolchain.toml"), "[toolchain]\nchannel = 'stable'\n");
        write(
            &fixture.root.join("dev/gate/policy/framework-boundary.txt"),
            "version = 1\nforbidden = ['forbidden_payload']\nretired_pattern = 'retired_payload'\norganization_prefix = 'github.com/acme/'\ncontrols = ['forbidden_payload']\naccepted_controls = ['github.com/acme/project']\n",
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
        Self::script(&fixture.bin.join("mise"), MISE_SCRIPT);
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
[ "$CARGO_TARGET_DIR" = "$CONTROL_EXPECT_TARGET" ]
[ "$CARGO_HOME" = "$CONTROL_CALLER/real home" ]
[ "$UMPIRE_QUICK_CACHE" = 0 ] && [ "$UMPIRE_CACHE_DISABLE" = 1 ]
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
if [ "${{CONTROL_OVERLAP:-0}}" = 1 ] && {{ [ "$label" = proto-tools ] || [ "$label" = fmt ]; }}; then
 case "$label" in
  proto-tools) peer=fmt ;;
  fmt) peer=proto-tools ;;
 esac
 mkdir "$CONTROL_ROOT/active-$label"
 remaining=500
 while [ ! -d "$CONTROL_ROOT/active-$peer" ]; do
  remaining=$((remaining - 1))
  [ "$remaining" -gt 0 ] || {{ printf 'independent stages never overlapped\n' >&2; exit 95; }}
  sleep 0.01
 done
 printf '%s\n' "$label" >> "$CONTROL_ROOT/overlap"
fi
if [ "${{CONTROL_MISE_DESCENDANT:-0}}" = 1 ]; then
 if [ "$label" = fmt ]; then
  /bin/bash -c 'echo $$ > "$CONTROL_ROOT/real-owned-child"; sleep 5; touch "$CONTROL_ROOT/real-child-escaped"' &
  while [ ! -f "$CONTROL_ROOT/failure-reached" ]; do sleep 0.01; done
  sleep 5
 elif [ "$label" = proto-tools ]; then
  remaining=500
  while [ ! -s "$CONTROL_ROOT/real-owned-child" ]; do
   remaining=$((remaining - 1)); [ "$remaining" -gt 0 ] || exit 97; sleep 0.01
  done
  touch "$CONTROL_ROOT/failure-reached"
 fi
fi
printf '%s|%s|%s|%s|%s|%s|%s|%s\n' "$label" "$PWD" "$CARGO_TARGET_DIR" "$CARGO_HOME" "$UMPIRE_QUICK_CACHE" "$UMPIRE_CACHE_DISABLE" "${{UMPIRE_BACKENDS:-unset}}" "$*" >> "$CONTROL_LOG"
if [ "$label" = tests ] && [ -n "${{CONTROL_EXPECT_PROFILE:-}}" ]; then
 [ "${{NEXTEST_PROFILE:-unset}}" = "$CONTROL_EXPECT_PROFILE" ]
 [ "${{NEXTEST_TEST_THREADS:-unset}}" = "$CONTROL_EXPECT_THREADS" ]
fi
if [ -n "${{CONTROL_EXPECT_CPU_JOBS:-}}" ]; then
 [ "$CARGO_BUILD_JOBS:$GOMAXPROCS" = "$CONTROL_EXPECT_CPU_JOBS:$CONTROL_EXPECT_CPU_JOBS" ]
 if [ "$CONTROL_EXPECT_CPU_JOBS" = 1 ]; then
  [ "$GOFLAGS" = '-tags=control -p=9 -p=1' ]
 else
  [ "$GOFLAGS" = '-tags=control -p=9' ]
 fi
fi
if [ "${{CONTROL_SIGNAL:-}}" = "$label" ]; then
 printf 'signal stdout diagnostic\n'
 printf 'signal stderr diagnostic\n' >&2
 kill -TERM $$
fi
[ "${{CONTROL_FAIL:-}}" != "$label" ] || {{ printf '%s child diagnostic\n' "$label" >&2; exit "${{CONTROL_EXIT:-37}}"; }}
if [ "${{CONTROL_EXPECT_UPDATE:-0}}" = 1 ]; then
 [ "$INSTA_UPDATE" = always ] && [ "$TRYBUILD" = overwrite ]
fi
if [ "${{CONTROL_FAIL_INNER:-0}}" = 1 ] && [ "$label" = tests ]; then
 case " $* " in *' -p umpire-checker '*) printf 'inner child diagnostic\n' >&2; exit 37 ;; esac
fi
if [ "$label" = inner-metadata ]; then
 printf '{{"workspace_members":["checker"],"packages":[{{"id":"checker","name":"umpire-checker","targets":[{{"kind":["lib"],"doctest":true}}]}}]}}\n'
fi
if [ "$label" = inner-inventory ] || [ "$label" = test-inventory ]; then
 printf '{{"test-count":1,"rust-suites":{{"checker":{{"package-name":"umpire-checker","binary-name":"umpire_checker","kind":"lib","status":"listed","testcases":{{"control":{{"ignored":false,"filter-match":{{"status":"matches"}}}}}}}}}}}}\n'
fi
if [ "$label" = tests ]; then
 case " $* " in *' --message-format '*) printf '%s\n' '{{"type":"test","event":"ok","name":"umpire-checker::umpire_checker$control"}}' ;; esac
fi
if [ "$label" = doctests ]; then
 printf '%s\n' 'running 1 test' 'test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.1s'
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
        self.command_target(timed, Some(Path::new("relative target")))
    }
    fn command_target(&self, timed: bool, target: Option<&Path>) -> Command {
        let expected = target.map_or_else(
            || self.root.join("target"),
            |path| {
                if path.is_absolute() { path.to_owned() } else { self.caller.join(path) }
            },
        );
        let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
        command
            .current_dir(&self.caller)
            .env_remove("BASH_ENV")
            .env_remove("UMPIRE_STANDALONE_CHILD")
            .env_remove("UMPIRE_STANDALONE_OUTPUT")
            .env_remove("UMPIRE_BACKENDS")
            .env_remove("UMPIRE_APALACHE")
            .env("PATH", format!("{}:/usr/bin:/bin", self.bin.display()))
            .env("UMPIRE_TOOL_ROOT", &self.root)
            .env("CONTROL_ROOT", &self.root)
            .env("CONTROL_CALLER", &self.caller)
            .env("CONTROL_LOG", &self.log)
            .env_remove("UMPIRE_GATE_RECEIPTS")
            .env_remove("UMPIRE_GATE_EXECUTABLE")
            .env_remove("UMPIRE_GATE_SCOPE")
            .env_remove("UMPIRE_GATE_STAGE")
            .env_remove("UMPIRE_GATE_ORDER")
            .env_remove("CONTROL_RECEIPT_FAULT")
            .env_remove("CONTROL_REAL_MISE")
            .env("CONTROL_EXPECT_TARGET", expected)
            .env("CARGO_HOME", "cargo home")
            .env("UMPIRE_TIME_STEPS", if timed { "1" } else { "0" })
            .env("UMPIRE_QUICK_CACHE", "1")
            .env("UMPIRE_CACHE_DISABLE", "0");
        if let Some(target) = target {
            command.env("CARGO_TARGET_DIR", target);
        } else {
            command.env_remove("CARGO_TARGET_DIR");
        }
        command
    }
    fn labels(&self) -> Vec<String> {
        fs::read_to_string(&self.log)
            .unwrap_or_default()
            .lines()
            .map(|line| line.split('|').next().unwrap().into())
            .filter(|label: &String| label != "test-inventory")
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
    let mut seen = std::collections::HashSet::new();
    for line in text.lines() {
        let (name, deps) = line.split_once('\t').unwrap();
        assert!(seen.insert(name), "duplicate stage {name}");
        for dependency in deps.split(',').filter(|dependency| !dependency.is_empty()) {
            assert!(seen.contains(dependency), "missing or unordered dependency {dependency}");
        }
    }
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
            } else if *leaf == "doctests" {
                vec!["tests", "doctests"]
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
        assert!(log.contains("cargo nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail"));
        assert!(log.contains("cargo test --doc --workspace --locked --quiet"));
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
        let mut expected = LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>();
        if entry == "validate" {
            assert_status(&output, 1);
            assert!(stderr(&output).contains("framework stage standalone failed: missing or escaping standalone configuration: Cargo.lock"), "{output:?}");
        } else {
            assert_status(&output, 0);
            expected.extend(["inner-metadata", "inner-inventory", "tests", "doctests"]);
        }
        assert_eq!(fixture.labels(), expected);
        let log = fs::read_to_string(&fixture.log).unwrap();
        if entry == "time" {
            assert!(
                log.lines()
                    .last()
                    .unwrap()
                    .ends_with("cargo test --doc --locked -p umpire-checker -- --nocapture")
            );
        } else {
            assert!(!log.contains("cargo test -p"));
        }
        if entry == "validate" {
            assert!(log.lines().all(|line| line.contains("|0|1|require|")), "{log}");
        }
    }
}

fn finish_before_deadline(mut child: std::process::Child) -> (bool, Output) {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(5);
    loop {
        if child.try_wait().unwrap().is_some() {
            return (true, child.wait_with_output().unwrap());
        }
        if std::time::Instant::now() >= deadline {
            child.kill().unwrap();
            return (false, child.wait_with_output().unwrap());
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
}

#[test]
fn supervision_public_fmt_finishes_while_host_lease_is_held() {
    let fixture = Fixture::new();
    let _lease = umpire_backend_process::JvmLease::inherited_or_acquire().unwrap();
    for expected in [0, 37] {
        fixture.clear();
        let mut command = fixture.command(false);
        command
            .arg("fmt")
            .env_remove("UMPIRE_JVM_LEASE_FD")
            .env("CONTROL_EXPECT_FMT", "cargo fmt --all")
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped());
        if expected != 0 {
            command.env("CONTROL_FAIL", "fmt");
        }
        let (completed, output) = finish_before_deadline(command.spawn().unwrap());
        assert!(completed, "public fmt waited for the unrelated host lease: {output:?}");
        assert_status(&output, expected);
        assert_eq!(fixture.labels(), ["fmt"]);
        assert!(fs::read_to_string(&fixture.log).unwrap().ends_with("cargo fmt --all\n"));
    }
}

#[test]
#[cfg(target_os = "linux")]
fn supervision_public_fmt_reaps_background_children_before_returning_failure() {
    let fixture = Fixture::new();
    Fixture::script(
        &fixture.bin.join("mise"),
        &format!("{MISE_SCRIPT}\nsleep 30 &\nprintf '%s' $! > \"$CONTROL_ROOT/fmt-child\"\n"),
    );
    let output = fixture
        .command(false)
        .arg("fmt")
        .env("CONTROL_EXPECT_FMT", "cargo fmt --all")
        .env("CONTROL_FAIL", "fmt")
        .output()
        .unwrap();
    assert_status(&output, 37);
    let pid = fs::read_to_string(fixture.root.join("fmt-child")).unwrap();
    let alive = Command::new("/bin/kill")
        .args(["-0", &pid])
        .stderr(std::process::Stdio::null())
        .status()
        .unwrap()
        .success();
    if alive {
        Command::new("/bin/kill").args(["-KILL", &pid]).status().unwrap();
    }
    assert!(!alive, "public fmt returned while its background child was alive");
}

#[test]
fn supervision_scratch_is_private_and_removed_after_success_and_failure() {
    let fixture = Fixture::new();
    let temporary = fixture.caller.join("scratch");
    fs::create_dir(&temporary).unwrap();
    Fixture::script(
        &fixture.bin.join("mise"),
        &format!(
            "{MISE_SCRIPT}\ntouch \"$CONTROL_ROOT/entered\"\nremaining=500\nwhile [ ! -f \"$CONTROL_ROOT/release\" ]; do\n remaining=$((remaining - 1)); [ \"$remaining\" -gt 0 ] || exit 97; sleep 0.01\ndone\n"
        ),
    );
    for expected in [0, 37] {
        let mut command = fixture.command(false);
        command.args(["gate", "--stage", "fmt"]).env("TMPDIR", &temporary);
        if expected != 0 {
            command.env("CONTROL_FAIL", "fmt");
        }
        let mut child = command.spawn().unwrap();
        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(5);
        while !fixture.root.join("entered").exists() {
            if child.try_wait().unwrap().is_some() || std::time::Instant::now() >= deadline {
                child.kill().ok();
                panic!("supervised leaf did not start: {:?}", child.wait().unwrap());
            }
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
        let mut directories = fs::read_dir(&temporary)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .collect::<Vec<_>>();
        if temporary.join("umpire-command").is_dir() {
            directories = fs::read_dir(temporary.join("umpire-command"))
                .unwrap()
                .map(|entry| entry.unwrap().path())
                .collect();
        }
        let permissions = directories
            .iter()
            .filter(|path| path.join("admission.json").is_file())
            .map(|path| (path.clone(), fs::metadata(path).unwrap().permissions().mode() & 0o777))
            .collect::<Vec<_>>();
        write(&fixture.root.join("release"), "release");
        let (completed, output) = finish_before_deadline(child);
        assert!(completed, "supervisor did not finish: {output:?}");
        assert_status(&output, expected);
        assert_eq!(permissions.len(), 1, "missing live supervision scratch: {directories:?}");
        assert_eq!(permissions[0].1, 0o700, "supervisor scratch exposes logs: {permissions:?}");
        assert!(!permissions[0].0.exists(), "supervisor scratch was retained");
        assert_eq!(fs::read_dir(&temporary).unwrap().count(), 0);
        fs::remove_file(fixture.root.join("entered")).unwrap();
        fs::remove_file(fixture.root.join("release")).unwrap();
    }
}

#[test]
fn supervision_ignores_obstructed_legacy_shared_scratch_parent() {
    let fixture = Fixture::new();
    let temporary = fixture.caller.join("scratch");
    fs::create_dir(&temporary).unwrap();
    write(&temporary.join("umpire-command"), "belongs to another user");
    let output = fixture
        .command(false)
        .args(["gate", "--stage", "fmt"])
        .env("TMPDIR", &temporary)
        .output()
        .unwrap();
    assert_status(&output, 0);
    assert_eq!(fixture.labels(), ["fmt"]);
    assert_eq!(
        fs::read_to_string(temporary.join("umpire-command")).unwrap(),
        "belongs to another user"
    );
    assert_eq!(fs::read_dir(&temporary).unwrap().count(), 1);
}

#[test]
fn convenience_entries_keep_exact_arguments_and_propagate_failed_inner_status() {
    let fixture = Fixture::new();
    for (entry, expected) in
        [("fmt", "cargo fmt --all"), ("update", "cargo test --doc --workspace --locked --quiet")]
    {
        fixture.clear();
        let mut command = fixture.command(false);
        command.arg(entry);
        if entry == "update" {
            command.env("CONTROL_EXPECT_UPDATE", "1");
        } else {
            command.env("CONTROL_EXPECT_FMT", expected);
        }
        let output = command.output().unwrap();
        assert_status(&output, 0);
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert_eq!(log.lines().count(), if entry == "update" { 2 } else { 1 });
        assert!(log.lines().last().unwrap().ends_with(expected), "{log}");
        if entry == "update" {
            assert!(
                log.lines().next().unwrap().contains("cargo nextest run --workspace --locked"),
                "{log}"
            );
        }
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
        if entry == "time" {
            assert_status(&output, 37);
            assert!(
                stderr(&output).contains("framework stage inner loop (edit+test) failed (exit 37)"),
                "{output:?}"
            );
            assert!(stderr(&output).contains("inner child diagnostic"), "{output:?}");
            assert_eq!(fixture.labels().len(), LEAVES.len() + 3);
        } else {
            assert_status(&output, 1);
            assert!(stderr(&output).contains("framework stage standalone failed: missing or escaping standalone configuration: Cargo.lock"), "{output:?}");
            assert_eq!(fixture.labels(), LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>());
        }
        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(!stdout.contains("gate total"), "{output:?}");
        assert!(!stdout.contains("over soft target"), "{output:?}");
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

#[test]
fn guest_and_go_stage_target_matrix_preserves_unset_relative_and_absolute_paths() {
    let fixture = Fixture::new();
    let absolute = fixture.caller.join("absolute target");
    for target in [None, Some(Path::new("relative target")), Some(absolute.as_path())] {
        let expected = target.map_or_else(
            || fixture.root.join("target"),
            |path| {
                if path.is_absolute() { path.to_owned() } else { fixture.caller.join(path) }
            },
        );
        for timed in [false, true] {
            for stage in ["wasm-guest", "go-host", "kettle-go-host"] {
                fixture.clear();
                let output = fixture
                    .command_target(timed, target)
                    .args(["gate", "--stage", stage])
                    .output()
                    .unwrap();
                assert_status(&output, 0);
                assert_eq!(fixture.labels(), [stage]);
                let log = fs::read_to_string(&fixture.log).unwrap();
                let fields = log.trim_end().split('|').collect::<Vec<_>>();
                assert_eq!(fields[1], fixture.root.to_str().unwrap());
                assert_eq!(fields[2], expected.to_str().unwrap());
                assert_eq!(fields[3], fixture.caller.join("real home").to_str().unwrap());
                if stage != "wasm-guest" {
                    assert!(fields[7].ends_with("test -tags test_dep -count=1 ./..."));
                }
            }
        }
    }
}

#[test]
fn test_runner_route_preserves_one_workspace_call_and_failures_in_both_modes() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for entry in [vec!["gate", "--stage", "tests"], vec!["update"]] {
            for code in [0, 1, 4, 37, 143] {
                fixture.clear();
                let mut command = fixture.command(timed);
                command.args(&entry);
                if entry == ["update"] {
                    command.env("CONTROL_EXPECT_UPDATE", "1");
                }
                if code == 143 {
                    command.env("CONTROL_SIGNAL", "tests");
                } else if code != 0 {
                    command.env("CONTROL_FAIL", "tests").env("CONTROL_EXIT", code.to_string());
                }
                let output = command.output().unwrap();
                assert_status(&output, code);
                let expected = if code == 0 { vec!["tests", "doctests"] } else { vec!["tests"] };
                assert_eq!(fixture.labels(), expected);
                let log = fs::read_to_string(&fixture.log).unwrap();
                assert_eq!(log.lines().count(), expected.len());
                assert!(log.lines().next().unwrap().ends_with("cargo nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail"), "{log}");
                if code == 0 {
                    assert!(
                        log.lines()
                            .last()
                            .unwrap()
                            .ends_with("cargo test --doc --workspace --locked --quiet"),
                        "{log}"
                    );
                }
                if code == 143 {
                    let diagnostics =
                        format!("{}{}", String::from_utf8_lossy(&output.stdout), stderr(&output));
                    assert!(diagnostics.contains("signal stdout diagnostic"));
                    assert!(diagnostics.contains("signal stderr diagnostic"));
                }
            }
        }
    }
    fixture.clear();
    let output = fixture.command(false).args(["test-run", "extra"]).output().unwrap();
    assert_status(&output, 1);
    assert!(stderr(&output).contains("usage: cargo xtask test-run"));
    assert_eq!(fixture.labels(), Vec::<String>::new());
}

#[test]
fn root_required_backend_environment_selects_standalone_on_gate_and_time_routes() {
    let fixture = Fixture::new();
    write(&fixture.root.join("backends/native/src/lib.rs"), "// benchmark input\n");
    for entry in ["gate", "time"] {
        fixture.clear();
        let output =
            fixture.command(false).arg(entry).env("UMPIRE_BACKENDS", "require").output().unwrap();
        assert_status(&output, 1);
        assert!(stderr(&output).contains("framework stage standalone failed: missing or escaping standalone configuration: Cargo.lock"), "{output:?}");
        assert_eq!(fixture.labels(), LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>());
        let log = fs::read_to_string(&fixture.log).unwrap();
        assert!(log.lines().all(|line| line.contains("|0|1|require|")), "{log}");
        assert!(!log.contains("cargo test -p"));
    }
}

#[test]
fn required_profile_reaches_gate_update_and_direct_test_children() {
    let fixture = Fixture::new();
    for entry in [vec!["gate", "--stage", "tests"], vec!["update"], vec!["test-run"]] {
        for (backend, apalache, inherited, profile, threads) in [
            ("skip", "skip", None, "unset", "unset"),
            ("skip", "skip", Some("2"), "unset", "2"),
            ("require", "skip", Some("99"), "required", "4"),
            ("require", "skip", None, "required", "4"),
            ("require", "skip", Some("2"), "required", "2"),
            ("require", "skip", Some("1"), "required", "1"),
            ("require", "skip", Some("malformed"), "required", "4"),
            ("require", "skip", Some("0"), "required", "4"),
            ("skip", "require", Some("2"), "required", "1"),
            ("require", "require", Some("99"), "required", "1"),
        ] {
            fixture.clear();
            let mut command = fixture.command(false);
            if entry == ["test-run"] {
                command
                    .current_dir(&fixture.root)
                    .env("CARGO_TARGET_DIR", fixture.caller.join("relative target"))
                    .env("CARGO_HOME", fixture.caller.join("real home"))
                    .env("UMPIRE_QUICK_CACHE", "0")
                    .env("UMPIRE_CACHE_DISABLE", "1");
            }
            command
                .args(&entry)
                .env("UMPIRE_BACKENDS", backend)
                .env("UMPIRE_APALACHE", apalache)
                .env("CONTROL_EXPECT_PROFILE", profile)
                .env("CONTROL_EXPECT_THREADS", threads)
                .env("CARGO_BUILD_JOBS", "99")
                .env("GOMAXPROCS", "99")
                .env("GOFLAGS", "-tags=control -p=9")
                .env(
                    "CONTROL_EXPECT_CPU_JOBS",
                    if backend == "require" || apalache == "require" { "1" } else { "99" },
                )
                .env_remove("NEXTEST_PROFILE")
                .env_remove("NEXTEST_TEST_THREADS");
            if backend == "require" || apalache == "require" {
                command.env("NEXTEST_PROFILE", "default");
            }
            if let Some(inherited) = inherited {
                command.env("NEXTEST_TEST_THREADS", inherited);
            }
            let output = command.output().unwrap();
            assert_eq!(
                output.status.code(),
                Some(0),
                "entry={entry:?}, backend={backend}, apalache={apalache}: {output:?}"
            );
            assert_eq!(fixture.labels(), ["tests", "doctests"]);
        }
    }
}

#[test]
fn scheduled_stage_requires_invocation_directory_before_starting_leaf() {
    let fixture = Fixture::new();
    let output =
        fixture.command(false).args(["gate", "--scheduled-stage", "fmt"]).output().unwrap();
    assert_status(&output, 1);
    assert!(stderr(&output).contains("invocation-local stage receipt directory"), "{output:?}");
    assert_eq!(fixture.labels(), Vec::<String>::new());
}

#[test]
fn scheduled_stage_atomically_commits_success_failure_and_scanner_error_receipts() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for (stage, failure, expected) in
            [("fmt", false, Some(0)), ("fmt", true, Some(37)), ("source-boundary", false, None)]
        {
            fixture.clear();
            let receipts = tempfile::tempdir_in(&fixture.caller).unwrap();
            if expected.is_none() {
                write(&fixture.root.join("planted"), "forbidden_payload");
            }
            let mut command = fixture.command(timed);
            command
                .args(["gate", "--scheduled-stage", stage])
                .env("UMPIRE_GATE_RECEIPTS", receipts.path());
            if failure {
                command.env("CONTROL_FAIL", "fmt");
            }
            let output = command.output().unwrap();
            assert_status(&output, expected.unwrap_or(1));
            let receipt: serde_json::Value = serde_json::from_slice(
                &fs::read(receipts.path().join(format!("{stage}.json"))).unwrap(),
            )
            .unwrap();
            assert_eq!(receipt["stage"], stage);
            assert_eq!(receipt["exit"].as_i64(), Some(i64::from(expected.unwrap_or(1))));
            if expected.is_none() {
                assert_eq!(receipt["state"], "failed");
                assert!(receipt["error"].is_string());
            }
            assert!(receipt["seconds"].as_f64().unwrap().is_finite());
            assert!(receipt["seconds"].as_f64().unwrap() >= 0.0);
            let published = fs::read_dir(receipts.path())
                .unwrap()
                .map(|entry| entry.unwrap().file_name())
                .collect::<Vec<_>>();
            assert!(
                published.iter().all(|name| {
                    name == "work"
                        || name == &std::ffi::OsString::from(format!("{stage}.json"))
                        || name == &std::ffi::OsString::from(format!("{stage}.claim"))
                }),
                "partial receipt files: {published:?}"
            );
            if expected.is_none() {
                fs::remove_file(fixture.root.join("planted")).unwrap();
            }
        }
    }
}

pub(super) fn spawn_owned(
    command: &Command,
    caller: &Path,
    lease: &umpire_backend_process::JvmLease,
) -> (umpire_backend_process::OwnedProcess, tempfile::TempDir) {
    let directory = tempfile::tempdir_in(caller).unwrap();
    let out = fs::File::create(directory.path().join("stdout")).unwrap();
    let err = fs::File::create(directory.path().join("stderr")).unwrap();
    let process = umpire_backend_process::OwnedProcess::spawn_leased(
        command,
        directory.path(),
        &out,
        &err,
        lease,
    )
    .unwrap();
    (process, directory)
}

pub(super) fn finish_owned(
    (mut process, directory): (umpire_backend_process::OwnedProcess, tempfile::TempDir),
) -> Output {
    let status = loop {
        if let Some(status) = process.try_wait().unwrap() {
            break status;
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    };
    drop(process);
    Output {
        status,
        stdout: fs::read(directory.path().join("stdout")).unwrap(),
        stderr: fs::read(directory.path().join("stderr")).unwrap(),
    }
}

#[test]
fn concurrent_duplicate_stage_claim_runs_leaf_exactly_once_and_preserves_receipt() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let receipts = tempfile::tempdir_in(&fixture.caller).unwrap();
        let lease = umpire_backend_process::JvmLease::inherited_or_acquire().unwrap();
        let mut first_command = fixture.command(timed);
        first_command
            .args(["gate", "--scheduled-stage", "fmt"])
            .env("UMPIRE_GATE_RECEIPTS", receipts.path());
        let first = spawn_owned(&first_command, &fixture.caller, &lease);
        let second = spawn_owned(&first_command, &fixture.caller, &lease);
        let first = finish_owned(first);
        let second = finish_owned(second);
        let statuses = [first.status.code().unwrap(), second.status.code().unwrap()];
        assert!(statuses == [0, 1] || statuses == [1, 0], "{statuses:?}");
        assert_eq!(fixture.labels(), ["fmt"]);
        let bytes = fs::read(receipts.path().join("fmt.json")).unwrap();
        let receipt: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
        assert_eq!(receipt["stage"], "fmt");
        assert_eq!(receipt["exit"], 0);
        let third = finish_owned(spawn_owned(&first_command, &fixture.caller, &lease));
        assert_status(&third, 1);
        assert_eq!(fs::read(receipts.path().join("fmt.json")).unwrap(), bytes);
        assert_eq!(fixture.labels(), ["fmt"]);
    }
}

#[test]
fn successful_scheduler_status_cannot_replace_complete_unique_atomic_stage_records() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for fault in ["zero", "duplicate", "partial", "missing", "extra"] {
            fixture.clear();
            let output = fixture
                .command(timed)
                .arg("gate")
                .env("CONTROL_RECEIPT_FAULT", fault)
                .output()
                .unwrap();
            assert!(!output.status.success(), "accepted {fault} receipts: {output:?}");
            assert!(!String::from_utf8_lossy(&output.stdout).contains("gate total"));
            if fault == "zero" {
                assert_eq!(fixture.labels(), Vec::<String>::new());
            } else if fault == "duplicate" {
                assert_eq!(fixture.labels(), ["proto-tools"]);
            } else {
                assert_eq!(
                    fixture.labels(),
                    LEAVES.iter().map(|(_, leaf)| *leaf).collect::<Vec<_>>()
                );
            }
        }
    }
}

#[test]
fn independent_native_stages_reach_each_other_while_both_are_active() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let receipts = tempfile::tempdir_in(&fixture.caller).unwrap();
        let lease = umpire_backend_process::JvmLease::inherited_or_acquire().unwrap();
        let mut command = fixture.command(timed);
        command
            .args(["gate", "--scheduled-stage", "proto-tools"])
            .env("UMPIRE_GATE_RECEIPTS", receipts.path())
            .env("CONTROL_OVERLAP", "1");
        let first = spawn_owned(&command, &fixture.caller, &lease);
        let mut command = fixture.command(timed);
        command
            .args(["gate", "--scheduled-stage", "fmt"])
            .env("UMPIRE_GATE_RECEIPTS", receipts.path())
            .env("CONTROL_OVERLAP", "1");
        let second = spawn_owned(&command, &fixture.caller, &lease);
        assert_status(&finish_owned(first), 0);
        assert_status(&finish_owned(second), 0);
        let mut overlap = fs::read_to_string(fixture.root.join("overlap"))
            .unwrap()
            .lines()
            .map(str::to_owned)
            .collect::<Vec<_>>();
        overlap.sort();
        assert_eq!(overlap, ["fmt", "proto-tools"]);
        for stage in ["proto-tools", "fmt"] {
            fs::remove_dir(fixture.root.join(format!("active-{stage}"))).unwrap();
            let receipt: serde_json::Value = serde_json::from_slice(
                &fs::read(receipts.path().join(format!("{stage}.json"))).unwrap(),
            )
            .unwrap();
            assert_eq!(receipt["exit"], 0);
        }
        fs::remove_file(fixture.root.join("overlap")).unwrap();
    }
}

#[test]
fn full_gate_publishes_complete_ordered_invocation_local_receipts_on_every_run() {
    let fixture = Fixture::new();
    let expected = STAGES.to_vec();
    let mut directories = std::collections::HashSet::new();
    for timed in [false, true] {
        fixture.clear();
        let output = fixture.command(timed).arg("gate").output().unwrap();
        assert_status(&output, 0);
        let invocations = fs::read_to_string(fixture.root.join("invocations")).unwrap();
        let directory = PathBuf::from(invocations.lines().last().unwrap());
        assert!(directory.is_absolute());
        assert!(directory.starts_with(fixture.caller.join("relative target/gate-receipts")));
        assert!(directories.insert(directory.clone()), "reused invocation {directory:?}");
        let rows = fs::read_to_string(directory.join("stages.jsonl"))
            .unwrap()
            .lines()
            .map(|line| serde_json::from_str::<serde_json::Value>(line).unwrap())
            .collect::<Vec<_>>();
        assert_eq!(rows.len(), expected.len());
        let actual = rows.iter().map(|row| row["stage"].as_str().unwrap()).collect::<Vec<_>>();
        assert_eq!(actual, expected);
        for row in rows {
            let name = row["stage"].as_str().unwrap();
            assert_eq!(row["exit"], 0);
            assert!(row["seconds"].as_f64().unwrap() >= 0.0);
            assert!(directory.join(format!("{name}.claim")).is_file());
            let atomic: serde_json::Value =
                serde_json::from_slice(&fs::read(directory.join(format!("{name}.json"))).unwrap())
                    .unwrap();
            assert_eq!(atomic, row);
        }
    }
}

#[test]
fn simultaneous_failed_native_leaves_both_retain_names_receipts_and_status() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let output = fixture
            .command(timed)
            .arg("gate")
            .env("CONTROL_RECEIPT_FAULT", "simultaneous")
            .output()
            .unwrap();
        assert_status(&output, 37);
        for stage in ["fmt", "proto-tools"] {
            assert!(
                stderr(&output).contains(&format!("framework stage {stage} failed (exit 37)")),
                "{output:?}"
            );
            assert!(stderr(&output).contains(&format!("{stage} child diagnostic")), "{output:?}");
        }
        let mut labels = fixture.labels();
        labels.sort();
        assert_eq!(labels, ["fmt", "proto-tools"]);
        let invocations = fs::read_to_string(fixture.root.join("invocations")).unwrap();
        let directory = PathBuf::from(invocations.lines().last().unwrap());
        let records = fs::read_to_string(directory.join("stages.jsonl"))
            .unwrap()
            .lines()
            .map(|line| serde_json::from_str::<serde_json::Value>(line).unwrap())
            .collect::<Vec<_>>();
        assert_eq!(records.len(), STAGES.len());
        for record in records {
            if record["stage"] == "proto-tools" || record["stage"] == "fmt" {
                assert_eq!(record["exit"], 37);
            } else {
                assert_eq!(record["state"], "not-started");
            }
        }
    }
}

#[test]
fn failed_scheduler_is_waited_and_its_owned_descendant_is_terminated_before_return() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        fixture.clear();
        let output = fixture
            .command(timed)
            .arg("gate")
            .env("CONTROL_RECEIPT_FAULT", "orphan")
            .output()
            .unwrap();
        assert_status(&output, 37);
        assert!(stderr(&output).contains("framework stage fmt failed (exit 37)"), "{output:?}");
        assert_eq!(fixture.labels(), ["fmt"]);
        assert!(
            !fixture.root.join("child-escaped-cleanup").exists(),
            "scheduler left descendant running to completion"
        );
        let pid = fs::read_to_string(fixture.root.join("owned-child")).unwrap();
        let mut gone = false;
        for _ in 0..100 {
            let alive = Command::new("/bin/kill")
                .args(["-0", pid.trim()])
                .stdout(std::process::Stdio::null())
                .stderr(std::process::Stdio::null())
                .status()
                .unwrap();
            if !alive.success() {
                gone = true;
                break;
            }
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
        assert!(gone, "owned descendant {pid} survived failed scheduler");
        fs::remove_file(fixture.root.join("owned-child")).unwrap();
    }
}

#[test]
fn shipping_mise_graph_really_overlaps_independent_native_leaves() {
    let fixture = Fixture::new();
    let mise = std::env::split_paths(&std::env::var_os("PATH").unwrap())
        .map(|path| path.join("mise"))
        .find(|path| path.is_file())
        .expect("shipping scheduler mise is required for its integration control");
    let output = fixture
        .command(false)
        .arg("gate")
        .env("CONTROL_REAL_MISE", mise)
        .env("MISE_TRUSTED_CONFIG_PATHS", &fixture.root)
        .env("CONTROL_OVERLAP", "1")
        .output()
        .unwrap();
    assert_status(&output, 0);
    let mut overlap = fs::read_to_string(fixture.root.join("overlap"))
        .unwrap()
        .lines()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    overlap.sort();
    assert_eq!(overlap, ["fmt", "proto-tools"]);
    let mut actual = fixture.labels();
    actual.sort();
    let mut expected = LEAVES.iter().map(|(_, leaf)| (*leaf).to_owned()).collect::<Vec<_>>();
    expected.sort();
    assert_eq!(actual, expected);
}

#[test]
fn required_test_stage_selects_bounded_profile_and_empty_selection_status_is_preserved() {
    let fixture = Fixture::new();
    for timed in [false, true] {
        for code in [0, 4] {
            fixture.clear();
            let mut command = fixture.command(timed);
            command.args(["gate", "--stage", "tests"]).env("UMPIRE_BACKENDS", "require");
            if code != 0 {
                command.env("CONTROL_FAIL", "tests").env("CONTROL_EXIT", code.to_string());
            }
            let output = command.output().unwrap();
            assert_status(&output, code);
            let log = fs::read_to_string(&fixture.log).unwrap();
            assert!(log.lines().next().unwrap().ends_with("cargo nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail"), "{log}");
            assert_eq!(
                fixture.labels(),
                if code == 0 { vec!["tests", "doctests"] } else { vec!["tests"] }
            );
            assert!(log.lines().all(|line| line.contains("|0|1|require|")), "{log}");
        }
    }
}

#[test]
fn actual_mise_failed_leaf_reaps_its_active_sibling_descendant_before_return() {
    let fixture = Fixture::new();
    let mise = std::env::split_paths(&std::env::var_os("PATH").unwrap())
        .map(|path| path.join("mise"))
        .find(|path| path.is_file())
        .unwrap();
    let output = fixture
        .command(false)
        .arg("gate")
        .env("CONTROL_REAL_MISE", mise)
        .env("MISE_TRUSTED_CONFIG_PATHS", &fixture.root)
        .env("CONTROL_OVERLAP", "1")
        .env("CONTROL_MISE_DESCENDANT", "1")
        .env("CONTROL_FAIL", "proto-tools")
        .output()
        .unwrap();
    assert_status(&output, 37);
    assert!(stderr(&output).contains("proto-tools failed (exit 37)"), "{output:?}");
    let pid = fs::read_to_string(fixture.root.join("real-owned-child")).unwrap();
    assert!(
        !Command::new("/bin/kill")
            .args(["-0", pid.trim()])
            .stdout(std::process::Stdio::null())
            .stderr(std::process::Stdio::null())
            .status()
            .unwrap()
            .success(),
        "real mise sibling descendant {pid} survived"
    );
    assert!(!fixture.root.join("real-child-escaped").exists());
}

#[test]
fn outer_supervision_streams_separate_diagnostics_before_scheduler_completion() {
    let fixture = Fixture::new();
    let lease = umpire_backend_process::JvmLease::inherited_or_acquire().unwrap();
    let mut command = fixture.command(false);
    command.arg("gate").env("CONTROL_RECEIPT_FAULT", "stream");
    let (mut child, directory) = spawn_owned(&command, &fixture.caller, &lease);
    let stderr = directory.path().join("stderr");
    let mut observed = false;
    for _ in 0..200 {
        if fs::read_to_string(&stderr).unwrap().contains("streamed-early-diagnostic") {
            observed = true;
            break;
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    let pending = child.try_wait().unwrap().is_none();
    write(&fixture.root.join("stream-release"), "release");
    let output = finish_owned((child, directory));
    assert_status(&output, 37);
    assert!(observed && pending, "diagnostic was buffered until scheduler completion");
    assert_eq!(
        String::from_utf8_lossy(&output.stderr).matches("streamed-early-diagnostic").count(),
        1
    );
}
