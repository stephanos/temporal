//! Temporal stages of the combined Model gate.
use std::os::unix::ffi::OsStringExt;
use std::{
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    path::Path,
    process::Command,
};

fn admit_command(name: &str, command: &mut Command) -> io::Result<()> {
    let caller = std::env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    command
        .env("CARGO_HOME", home)
        .env("UMPIRE_QUICK_CACHE", "0")
        .env("UMPIRE_CACHE_DISABLE", "1")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    if name == "tests" || name == "source-boundary" {
        crate::test_run::configure(
            command,
            std::env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require"),
            std::env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
        );
    }

    if let Some(target) = std::env::var_os("CARGO_TARGET_DIR") {
        let target = std::path::PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    xtask::gate::admission::configure(
        command,
        std::env::var_os("UMPIRE_BACKENDS").is_some_and(|v| v == "require")
            || std::env::var_os("UMPIRE_APALACHE").is_some_and(|v| v == "require"),
    );
    Ok(())
}

pub(crate) fn stage(root: &Path, name: &str, mut command: Command) -> io::Result<i32> {
    admit_command(name, &mut command)?;
    let status = if std::env::var_os("UMPIRE_TIME_STEPS").is_some_and(|value| value == "1") {
        let mut timer = Command::new("/bin/bash");
        timer
            .arg(root.join("umpire/scripts/time-gate.sh"))
            .args(["--step", name])
            .arg(command.get_program())
            .args(command.get_args())
            .current_dir(root);
        for (key, value) in command.get_envs() {
            if let Some(value) = value {
                timer.env(key, value);
            } else {
                timer.env_remove(key);
            }
        }
        timer.status()
    } else {
        command.current_dir(root).status()
    }
    .map_err(|error| io::Error::other(format!("Temporal stage {name}: {error}")))?;
    let code = status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
    if code != 0 {
        let _ = writeln!(io::stderr(), "Temporal stage {name} failed (exit {code})");
    }
    Ok(code)
}

pub(crate) fn cargo(arguments: &[&str]) -> Command {
    let mut command = Command::new("mise");
    command.args(["exec", "--", "cargo"]).args(arguments);
    command
}

fn script(root: &Path, script: &str, arguments: &[&str]) -> Command {
    let mut command = Command::new("/bin/bash");
    command.arg(root.join("scripts").join(script)).args(arguments);
    command
}

use xtask::gate::scheduler;

fn mise(root: &Path) -> io::Result<Command> {
    let caller = std::env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let mut command = Command::new("mise");
    command
        .current_dir(root.parent().ok_or_else(|| io::Error::other("missing repository root"))?)
        .env("CARGO_HOME", home)
        .env("UMPIRE_FRAMEWORK_ROOT", root.join("umpire"))
        .env("UMPIRE_TEMPORAL_ROOT", root)
        .env("UMPIRE_QUICK_CACHE", "0")
        .env("UMPIRE_CACHE_DISABLE", "1")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    if let Some(target) = std::env::var_os("CARGO_TARGET_DIR") {
        let target = std::path::PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    admit_command("graph", &mut command)?;
    Ok(command)
}

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    let bootstrap = framework_executable(root)?;
    if !bootstrap.status.success() {
        writeln!(
            io::stderr(),
            "framework bootstrap failed (exit {})",
            bootstrap.status.code().unwrap_or(1)
        )?;
        return Ok(bootstrap.status.code().unwrap_or(1));
    }
    let framework = std::ffi::OsString::from_vec(
        bootstrap.stdout.strip_suffix(b"\n").unwrap_or(&bootstrap.stdout).to_owned(),
    );
    let expected = scheduler::inventory(mise(root)?, "")?;
    let caller = std::env::current_dir()?;
    let target = std::env::var_os("CARGO_TARGET_DIR").map_or_else(
        || root.join("target"),
        |path| {
            let path = std::path::PathBuf::from(path);
            if path.is_absolute() { path } else { caller.join(path) }
        },
    );
    let records = scheduler::directory(&target.join("gate-receipts"))?;
    let mut command = mise(root)?;
    command
        .env("UMPIRE_TEMPORAL_GATE_EXECUTABLE", std::env::current_exe()?)
        .env("UMPIRE_GATE_EXECUTABLE", framework);
    scheduler::run(command, "model:all", &records, &expected)
}

fn framework_executable(root: &Path) -> io::Result<std::process::Output> {
    let mut command = Command::new("/bin/bash");
    command
        .arg(root.join("scripts/xtask.sh"))
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    admit_command("bootstrap", &mut command)?;
    let output = command.output()?;
    io::stderr().write_all(&output.stderr)?;
    Ok(output)
}

pub(crate) fn list(root: &Path) -> io::Result<i32> {
    for (name, deps) in scheduler::listing(mise(root)?, "")? {
        writeln!(io::stdout(), "{name}\t{}", deps.join(","))?;
    }
    Ok(0)
}

pub(crate) fn scheduled_stage(root: &Path, name: &str) -> io::Result<i32> {
    scheduler::scheduled(name, || run_stage(root, name))
}

pub(crate) fn run_stage(root: &Path, name: &str) -> io::Result<i32> {
    let command = match name {
        "temporal-prerequisites" => {
            let mut command = Command::new("mise");
            command
                .args(["exec", "--", "make", "-C"])
                .arg(root.parent().ok_or_else(|| io::Error::other("missing repository root"))?)
                .args(["-j1", "model-prerequisites"])
                .env_remove("MAKEFLAGS")
                .env_remove("MFLAGS");
            command
        }
        "temporal-guest-freshness" => {
            let mut command = script(root, "guest-freshness.sh", &["umpire-temporal-guest"]);
            command.arg(root.join("temporal/hosts/go/umpire_temporal_guest.wasm"));
            command
        }
        "retired-names" => script(root, "retired-names.sh", &[]),
        "timing-controls" => script(root, "test-time-gate.sh", &[]),
        "source-boundary" => {
            use std::collections::BTreeSet;
            let selection = xtask::test_run::Selection {
                package: "umpire-temporal-tests",
                target: Some("boundary"),
                filter: None,
                doctests: false,
            };
            return xtask::test_run::run_selected_checked_with(
                &[selection],
                &BTreeSet::new(),
                |args| {
                    let mut command = cargo(&[]);
                    command.args(args).current_dir(root);
                    admit_command("source-boundary", &mut command)?;
                    let output = command.output()?;
                    io::stderr().write_all(&output.stderr)?;
                    let code = output.status.code().unwrap_or(1);
                    if code != 0 {
                        writeln!(
                            io::stderr(),
                            "Temporal stage source-boundary failed (exit {code})"
                        )?;
                    }
                    Ok((code, output.stdout))
                },
                |_, args| {
                    let mut command = cargo(&[]);
                    command.args(args);
                    stage(root, "source-boundary", command).map_err(Into::into)
                },
            )
            .map_err(io::Error::other);
        }
        "protobuf-controls" => script(root, "tests/proto-go.sh", &[]),
        "temporal-protobuf-freshness" => script(root, "proto-go.sh", &["temporal", "--check"]),
        "fmt" => cargo(&["fmt", "--all", "--check"]),
        "clippy" => cargo(&[
            "clippy",
            "--workspace",
            "--all-targets",
            "--all-features",
            "--",
            "-D",
            "warnings",
        ]),
        "docs" => {
            let mut command = cargo(&["doc", "--workspace", "--no-deps", "--quiet"]);
            command.env("RUSTDOCFLAGS", "-D warnings");
            command
        }
        "tests" => return crate::test_run::run(root),
        "wasm-guest" => script(root, "tests/build-guests.sh", &[]),
        "temporal-go-host" => {
            let mut command = Command::new("mise");
            command
                .args(["exec", "--", "go", "-C"])
                .arg(root.join("temporal/hosts/go"))
                .args(["test", "-tags", "test_dep", "-count=1", "./..."])
                .env("CGO_ENABLED", "0");
            command
        }
        _ => return Err(io::Error::other(format!("unknown Temporal stage: {name}"))),
    };
    stage(root, name, command)
}

#[cfg(test)]
#[allow(clippy::unwrap_used, clippy::expect_used)]
mod tests {
    use super::*;

    #[test]
    fn full_gate_overrides_cache_and_preserves_failure() {
        let mut command = Command::new("/bin/bash");
        command.args(["-c", "test \"$UMPIRE_QUICK_CACHE\" = 0 && test \"$UMPIRE_CACHE_DISABLE\" = 1 || exit 9; exit 37"])
            .env("UMPIRE_QUICK_CACHE", "1").env("UMPIRE_CACHE_DISABLE", "0");
        assert_eq!(
            stage(&Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."), "control", command)
                .unwrap(),
            37
        );
    }

    #[test]
    fn test_policy_control_child() {
        let Ok(expected) = std::env::var("TEMPORAL_EXPECT_TEST_POLICY") else { return };
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "test \"${NEXTEST_PROFILE:-unset}:${NEXTEST_TEST_THREADS:-unset}\" = \"$TEMPORAL_EXPECT_TEST_POLICY\""]);
        assert_eq!(stage(Path::new("/"), "tests", command).unwrap(), 0, "{expected}");
    }

    #[test]
    fn test_stage_propagates_required_policy_and_preserves_skipped_defaults() {
        for (backend, apalache, inherited, expected) in [
            ("skip", "skip", None, "unset:unset"),
            ("skip", "skip", Some("2"), "unset:2"),
            ("require", "skip", None, "required:4"),
            ("require", "skip", Some("99"), "required:4"),
            ("require", "skip", Some("2"), "required:2"),
            ("require", "skip", Some("1"), "required:1"),
            ("require", "skip", Some("malformed"), "required:4"),
            ("require", "skip", Some("0"), "required:4"),
            ("skip", "require", Some("2"), "required:1"),
            ("require", "require", Some("99"), "required:1"),
        ] {
            let mut child = Command::new(std::env::current_exe().unwrap());
            child
                .args(["--exact", "gate::tests::test_policy_control_child", "--nocapture"])
                .env("TEMPORAL_EXPECT_TEST_POLICY", expected)
                .env("UMPIRE_BACKENDS", backend)
                .env("UMPIRE_APALACHE", apalache)
                .env_remove("NEXTEST_PROFILE")
                .env_remove("NEXTEST_TEST_THREADS");
            if backend == "require" || apalache == "require" {
                child.env("NEXTEST_PROFILE", "default");
            }
            if let Some(inherited) = inherited {
                child.env("NEXTEST_TEST_THREADS", inherited);
            }
            let output = child.output().unwrap();
            assert!(output.status.success(), "{output:?}");
        }
    }

    #[test]
    fn spawn_failure_names_the_stage() {
        let error = stage(Path::new("/"), "missing-tool", Command::new("/missing/temporal-tool"))
            .unwrap_err();
        assert!(error.to_string().contains("missing-tool"));
    }

    #[test]
    fn run_control_child() {
        let Some(root) = std::env::var_os("UMPIRE_GATE_CONTROL_ROOT") else { return };
        if let Some(name) = std::env::var_os("UMPIRE_GATE_CONTROL_STAGE") {
            let name = name.to_str().unwrap();
            let status = scheduled_stage(Path::new(&root), name).unwrap();
            std::process::exit(status);
        }
        assert_eq!(
            run(Path::new(&root)).unwrap(),
            std::env::var("UMPIRE_GATE_CONTROL_STATUS").unwrap().parse::<i32>().unwrap(),
        );
    }

    const CONTROL_BODY: &str = r#"
[ "$UMPIRE_QUICK_CACHE:$UMPIRE_CACHE_DISABLE" = 0:1 ] || exit 9
[ "$CARGO_HOME" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-home" ] || exit 8
[ "$CARGO_TARGET_DIR" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-target" ] || exit 7
printf '%s\n' "$label" >> "$UMPIRE_GATE_CONTROL_LOG"
[ "$label" != "${UMPIRE_GATE_CONTROL_FAILURE:-}" ] || exit 37
"#;

    const CONTROL_STAGES: [(&str, &str); 15] = [
        ("temporal-prerequisites", "prerequisites"),
        ("temporal-guest-freshness", "guest"),
        ("retired-names", "retired"),
        ("timing-controls", "timing"),
        ("source-boundary", "boundary-inventory"),
        ("source-boundary", "boundary-tests"),
        ("protobuf-controls", "proto-controls"),
        ("temporal-protobuf-freshness", "proto"),
        ("fmt", "exec -- cargo fmt --all --check"),
        ("clippy", "exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings"),
        ("docs", "exec -- cargo doc --workspace --no-deps --quiet"),
        (
            "tests",
            "exec -- cargo nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail",
        ),
        ("tests", "exec -- cargo test --doc --workspace --locked --quiet"),
        ("wasm-guest", "guest-build"),
        (
            "temporal-go-host",
            "exec -- go -C PLACEHOLDER/temporal/hosts/go test -tags test_dep -count=1 ./...",
        ),
    ];

    fn control_inventory(stage_names: &[&str]) -> String {
        serde_json::to_string(
            &stage_names
                .iter()
                .enumerate()
                .map(|(order, name)| {
                    serde_json::json!({"name": format!("model:{name}"), "env": [
                        format!("UMPIRE_GATE_STAGE={name}"), format!("UMPIRE_GATE_ORDER={order}")
                    ], "depends": []})
                })
                .collect::<Vec<_>>(),
        )
        .unwrap()
    }

    fn control_directory() -> std::path::PathBuf {
        use std::fs;
        let root = std::env::temp_dir().join(format!("temporal-gate-{}", std::process::id()));
        fs::create_dir_all(root.join("scripts/tests")).unwrap();
        fs::create_dir_all(root.join("bin")).unwrap();
        fs::create_dir_all(root.join("umpire/scripts")).unwrap();
        fs::copy(
            Path::new(env!("CARGO_MANIFEST_DIR")).join("../../umpire/scripts/time-gate.sh"),
            root.join("umpire/scripts/time-gate.sh"),
        )
        .unwrap();
        fs::write(root.join("scripts/xtask.sh"), "#!/bin/bash\nprintf '/bin/true\\n'\n").unwrap();
        root
    }

    fn control_mise_script(scheduler: &str) -> String {
        let body = CONTROL_BODY;
        let nextest = r#"
case "$*" in
 'exec -- make -C '*' -j1 model-prerequisites')
   [ "$5" = "$(dirname "$UMPIRE_GATE_CONTROL_ROOT")" ] || exit 92
   [ -z "${MAKEFLAGS+x}${MFLAGS+x}" ] || exit 93
   label=prerequisites ;;
 'exec -- cargo nextest list'*)
   label=boundary-inventory
   BODY
   printf '%s\n' '{"rust-suites":{"boundary":{"package-name":"umpire-temporal-tests","binary-name":"boundary","kind":"test","status":"listed","testcases":{"control":{"ignored":false,"filter-match":{"status":"matches"}}}}}}'
   exit 0 ;;
 'exec -- cargo nextest run'*' -p umpire-temporal-tests '*) label=boundary-tests ;;
 *) label="$*" ;;
esac
"#.replace("BODY", body);
        format!("#!/bin/bash\nset -eu\n{scheduler}\n{nextest}\n{body}")
    }

    #[test]
    fn actual_temporal_gate_wiring_and_named_failures() {
        use std::{fs, os::unix::fs::PermissionsExt};
        let root = control_directory();
        let labels = [
            ("guest-freshness.sh", "guest"),
            ("retired-names.sh", "retired"),
            ("test-time-gate.sh", "timing"),
            ("tests/proto-go.sh", "proto-controls"),
            ("proto-go.sh", "proto"),
            ("tests/build-guests.sh", "guest-build"),
        ];
        let body = CONTROL_BODY;
        for (file, label) in labels {
            fs::write(
                root.join("scripts").join(file),
                format!("#!/bin/bash\nset -eu\nlabel={label}\n{body}"),
            )
            .unwrap();
        }
        let mise = root.join("bin/mise");
        let stages = CONTROL_STAGES;
        let mut stage_names = Vec::new();
        for (name, _) in &stages {
            if !stage_names.contains(name) {
                stage_names.push(*name);
            }
        }
        let inventory = control_inventory(&stage_names);
        let scheduler = format!(
            r#"
if [ "$*" = "tasks ls --json" ]; then
  printf '%s\n' '{inventory}'
  exit 0
fi
if [ "$*" = "run --force --skip-tools --task-cache off --jobs 4 model:all" ]; then
  for stage_name in {names}; do
    UMPIRE_GATE_CONTROL_STAGE=$stage_name "$UMPIRE_TEMPORAL_GATE_EXECUTABLE" --exact gate::tests::run_control_child --nocapture || exit $?
  done
  exit 0
fi
"#,
            names = stage_names.join(" ")
        );
        fs::write(&mise, control_mise_script(&scheduler)).unwrap();
        fs::set_permissions(&mise, fs::Permissions::from_mode(0o755)).unwrap();
        let expected =
            stages.map(|(_, label)| label.replace("PLACEHOLDER", &root.to_string_lossy()));
        let log = root.join("calls");
        for (timed, failure) in std::iter::once(None)
            .chain((0..expected.len()).map(Some))
            .flat_map(|failure| [("0", failure), ("1", failure)])
        {
            fs::write(&log, "").unwrap();
            let mut child = Command::new(std::env::current_exe().unwrap());
            child
                .args(["--exact", "gate::tests::run_control_child", "--nocapture"])
                .env("UMPIRE_GATE_CONTROL_ROOT", &root)
                .env("UMPIRE_GATE_CONTROL_LOG", &log)
                .env("UMPIRE_GATE_CONTROL_STATUS", if failure.is_some() { "37" } else { "0" })
                .env(
                    "UMPIRE_GATE_CONTROL_FAILURE",
                    failure.map_or("", |index| expected[index].as_str()),
                )
                .env("UMPIRE_QUICK_CACHE", "1")
                .env("UMPIRE_CACHE_DISABLE", "0")
                .env("CARGO_HOME", "control-relative-home")
                .env("CARGO_TARGET_DIR", "control-relative-target")
                .env("UMPIRE_GATE_CONTROL_CALLER", std::env::current_dir().unwrap())
                .env("UMPIRE_TIME_STEPS", timed)
                .env("MAKEFLAGS", "-n -j99")
                .env("MFLAGS", "-n -j99")
                .env(
                    "PATH",
                    format!("{}:{}", root.join("bin").display(), std::env::var("PATH").unwrap()),
                );
            let output = child.output().unwrap();
            assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stdout));
            if let Some(index) = failure {
                assert!(
                    String::from_utf8_lossy(&output.stderr).contains(stages[index].0),
                    "failure={index}, stage={:?}: {output:?}",
                    stages[index]
                );
            }
            let actual = fs::read_to_string(&log).unwrap();
            let end = failure.map_or(expected.len(), |index| index + 1);
            assert_eq!(actual.lines().collect::<Vec<_>>(), expected[..end]);
        }
        fs::remove_dir_all(root).unwrap();
    }
}
