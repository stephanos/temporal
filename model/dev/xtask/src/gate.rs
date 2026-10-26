//! Temporal stages of the combined Model gate.
use std::{
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    path::Path,
    process::Command,
};

pub(crate) fn stage(root: &Path, name: &str, mut command: Command) -> io::Result<i32> {
    let caller = std::env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    command.env("CARGO_HOME", home).env("UMPIRE_QUICK_CACHE", "0").env("UMPIRE_CACHE_DISABLE", "1");
    if let Some(target) = std::env::var_os("CARGO_TARGET_DIR") {
        let target = std::path::PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    let status = if std::env::var_os("UMPIRE_TIME_STEPS").is_some_and(|value| value == "1") {
        let mut timer = Command::new("/bin/bash");
        timer
            .arg(root.join("umpire/scripts/time-gate.sh"))
            .args(["--step", name])
            .arg(command.get_program())
            .args(command.get_args())
            .current_dir(root)
            .envs(command.get_envs().filter_map(|(key, value)| value.map(|value| (key, value))));
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

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    let guest = root.join("temporal/hosts/go/umpire_temporal_guest.wasm");
    let mut freshness = script(root, "guest-freshness.sh", &["umpire-temporal-guest"]);
    freshness.arg(&guest);
    let stages = [
        ("temporal-guest-freshness", freshness),
        ("retired-names", script(root, "retired-names.sh", &[])),
        ("timing-controls", script(root, "test-time-gate.sh", &[])),
        (
            "source-boundary",
            cargo(&["test", "-p", "umpire-temporal-tests", "--test", "boundary", "--quiet"]),
        ),
        ("protobuf-controls", script(root, "tests/proto-go.sh", &[])),
        ("temporal-protobuf-freshness", script(root, "proto-go.sh", &["temporal", "--check"])),
        ("fmt", cargo(&["fmt", "--all", "--check"])),
        (
            "clippy",
            cargo(&[
                "clippy",
                "--workspace",
                "--all-targets",
                "--all-features",
                "--",
                "-D",
                "warnings",
            ]),
        ),
        ("docs", {
            let mut command = cargo(&["doc", "--workspace", "--no-deps", "--quiet"]);
            command.env("RUSTDOCFLAGS", "-D warnings");
            command
        }),
    ];
    for (name, command) in stages {
        let status = stage(root, name, command)?;
        if status != 0 {
            return Ok(status);
        }
    }
    let status = crate::test_run::run(root)?;
    if status != 0 {
        return Ok(status);
    }
    let status = stage(root, "wasm-guest", script(root, "tests/build-guests.sh", &[]))?;
    if status != 0 {
        return Ok(status);
    }
    let mut host = Command::new("mise");
    host.args(["exec", "--", "go", "-C"])
        .arg(root.join("temporal/hosts/go"))
        .args(["test", "-tags", "test_dep", "-count=1", "./..."])
        .env("CGO_ENABLED", "0");
    stage(root, "temporal-go-host", host)
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
    fn spawn_failure_names_the_stage() {
        let error = stage(Path::new("/"), "missing-tool", Command::new("/missing/temporal-tool"))
            .unwrap_err();
        assert!(error.to_string().contains("missing-tool"));
    }

    #[test]
    fn run_control_child() {
        let Some(root) = std::env::var_os("UMPIRE_GATE_CONTROL_ROOT") else { return };
        assert_eq!(
            run(Path::new(&root)).unwrap(),
            std::env::var("UMPIRE_GATE_CONTROL_STATUS").unwrap().parse::<i32>().unwrap(),
        );
    }

    #[test]
    fn actual_temporal_gate_wiring_and_named_failures() {
        use std::{fs, os::unix::fs::PermissionsExt};
        let root = std::env::temp_dir().join(format!("temporal-gate-{}", std::process::id()));
        fs::create_dir_all(root.join("scripts/tests")).unwrap();
        fs::create_dir_all(root.join("bin")).unwrap();
        fs::create_dir_all(root.join("umpire/scripts")).unwrap();
        fs::copy(
            Path::new(env!("CARGO_MANIFEST_DIR")).join("../../umpire/scripts/time-gate.sh"),
            root.join("umpire/scripts/time-gate.sh"),
        )
        .unwrap();
        let labels = [
            ("guest-freshness.sh", "guest"),
            ("retired-names.sh", "retired"),
            ("test-time-gate.sh", "timing"),
            ("tests/proto-go.sh", "proto-controls"),
            ("proto-go.sh", "proto"),
            ("tests/build-guests.sh", "guest-build"),
        ];
        let body = r#"
[ "$UMPIRE_QUICK_CACHE:$UMPIRE_CACHE_DISABLE" = 0:1 ] || exit 9
[ "$CARGO_HOME" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-home" ] || exit 8
[ "$CARGO_TARGET_DIR" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-target" ] || exit 7
printf '%s\n' "$label" >> "$UMPIRE_GATE_CONTROL_LOG"
[ "$label" != "${UMPIRE_GATE_CONTROL_FAILURE:-}" ] || exit 37
"#;
        for (file, label) in labels {
            fs::write(
                root.join("scripts").join(file),
                format!("#!/bin/bash\nset -eu\nlabel={label}\n{body}"),
            )
            .unwrap();
        }
        let mise = root.join("bin/mise");
        fs::write(&mise, format!("#!/bin/bash\nset -eu\nlabel=\"$*\"\n{body}")).unwrap();
        fs::set_permissions(&mise, fs::Permissions::from_mode(0o755)).unwrap();
        let expected = [
            "guest",
            "retired",
            "timing",
            "exec -- cargo test -p umpire-temporal-tests --test boundary --quiet",
            "proto-controls",
            "proto",
            "exec -- cargo fmt --all --check",
            "exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings",
            "exec -- cargo doc --workspace --no-deps --quiet",
            "exec -- cargo test --workspace --locked --quiet",
            "guest-build",
            "exec -- go -C PLACEHOLDER/temporal/hosts/go test -tags test_dep -count=1 ./...",
        ]
        .map(|label| label.replace("PLACEHOLDER", &root.to_string_lossy()));
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
                .env(
                    "PATH",
                    format!("{}:{}", root.join("bin").display(), std::env::var("PATH").unwrap()),
                );
            let output = child.output().unwrap();
            assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stdout));
            if let Some(index) = failure {
                let names = [
                    "temporal-guest-freshness",
                    "retired-names",
                    "timing-controls",
                    "source-boundary",
                    "protobuf-controls",
                    "temporal-protobuf-freshness",
                    "fmt",
                    "clippy",
                    "docs",
                    "tests",
                    "wasm-guest",
                    "temporal-go-host",
                ];
                assert!(String::from_utf8_lossy(&output.stderr).contains(names[index]));
            }
            let actual = fs::read_to_string(&log).unwrap();
            let end = failure.map_or(expected.len(), |index| index + 1);
            assert_eq!(actual.lines().collect::<Vec<_>>(), expected[..end]);
        }
        fs::remove_dir_all(root).unwrap();
    }
}
