//! Consumer selection and insta artifacts are exercised through the real runner.

use std::{error::Error, fs, path::Path, process::Command};
use umpire_testing::{Consumer, Files, RunResult};

fn control_files() -> RunResult {
    let content = std::env::var("UMPIRE_CONTROL_CONTENT")?;
    Ok(Files::from([("artifact.tla".into(), content.clone()), ("artifact.cfg".into(), content)]))
}

#[test]
fn runner_child() -> Result<(), Box<dyn Error>> {
    let Ok(root) = std::env::var("UMPIRE_CONTROL_ROOT") else { return Ok(()) };
    umpire_testing::run_at(
        Path::new(&root),
        vec![("control", || {
            assert!(
                std::env::var_os("UMPIRE_CONTROL_CHECKER").is_some(),
                "unselected checker producer ran"
            );
            Ok(Files::from([("report.txt".into(), "checker selected\n".into())]))
        })],
        &[
            umpire_testing::CHECKER,
            Consumer {
                name: "control",
                expected: &["artifact.tla", "artifact.cfg"],
                render: |_, _| control_files(),
            },
        ],
    )
}

fn child(
    root: &Path,
    update: &str,
    content: &str,
    checker: bool,
) -> Result<std::process::Output, Box<dyn Error>> {
    let mut command = Command::new(std::env::current_exe()?);
    if checker {
        command.env("UMPIRE_CONTROL_CHECKER", "1");
    } else {
        command.env_remove("UMPIRE_CONTROL_CHECKER");
    }
    Ok(command
        .args(["--exact", "runner_child", "--nocapture"])
        .env("UMPIRE_CONTROL_ROOT", root)
        .env("UMPIRE_CONTROL_CONTENT", content)
        .env("INSTA_UPDATE", update)
        .output()?)
}

fn with_control(
    name: &str,
    check: impl FnOnce(&Path, &Path) -> umpire_testing::TestResult,
) -> umpire_testing::TestResult {
    let root = std::env::temp_dir().join(format!("umpire-{name}-{}", std::process::id()));
    let expected = root.join("fixtures/control/expected");
    fs::create_dir_all(&expected)?;
    fs::write(root.join("fixtures/control/model.rs"), "")?;
    fs::write(expected.join("artifact@tla.snap.tla"), "seed\n")?;
    let result = check(&root, &expected);
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn consumers_select_generic_snapshot_files() -> Result<(), Box<dyn Error>> {
    with_control("select-consumers", |root, expected| {
        let updated = child(&root.join("fixtures"), "always", "first\n", false)?;
        assert!(updated.status.success(), "{}", String::from_utf8_lossy(&updated.stdout));
        for extension in ["tla", "cfg"] {
            let payload = expected.join(format!("artifact@{extension}.snap.{extension}"));
            assert_eq!(
                fs::read_to_string(&payload).map_err(|e| format!(
                    "{}: {e}; runner: {}",
                    payload.display(),
                    String::from_utf8_lossy(&updated.stdout)
                ))?,
                "first\n"
            );
            assert!(!expected.join(format!("artifact.{extension}")).exists());
        }
        let checked = child(&root.join("fixtures"), "no", "first\n", false)?;
        assert!(checked.status.success(), "{}", String::from_utf8_lossy(&checked.stdout));
        fs::write(expected.join("report.snap.txt"), "seed\n")?;
        let mixed = child(&root.join("fixtures"), "always", "first\n", true)?;
        assert!(mixed.status.success(), "{}", String::from_utf8_lossy(&mixed.stdout));
        assert_eq!(fs::read_to_string(expected.join("report.snap.txt"))?, "checker selected\n");
        assert!(!expected.join("report.txt").exists());
        let mixed = child(&root.join("fixtures"), "no", "first\n", true)?;
        assert!(mixed.status.success(), "{}", String::from_utf8_lossy(&mixed.stdout));
        Ok(())
    })
}

#[test]
#[ignore = "requires the optional cargo-insta CLI; run explicitly with --ignored"]
fn accepts_generic_pending_snapshots_with_cli() -> Result<(), Box<dyn Error>> {
    with_control("accept-consumers", |root, expected| {
        let updated = child(&root.join("fixtures"), "always", "first\n", false)?;
        assert!(updated.status.success(), "{}", String::from_utf8_lossy(&updated.stdout));
        let pending = child(&root.join("fixtures"), "new", "accepted\n", false)?;
        assert!(!pending.status.success(), "mismatching snapshot silently passed");
        assert!(expected.join("artifact@cfg.snap.new").is_file());
        assert!(expected.join("artifact@cfg.snap.new.cfg").is_file());
        fs::write(
            root.join("Cargo.toml"),
            "[package]\nname = \"snapshot-control\"\nversion = \"0.0.0\"\nedition = \"2024\"\n[workspace]\n[lib]\npath = \"fixtures/control/model.rs\"\n",
        )?;
        let accepted = Command::new("cargo")
            .args(["insta", "accept", "--manifest-path"])
            .arg(root.join("Cargo.toml"))
            .output()?;
        assert!(accepted.status.success(), "{}", String::from_utf8_lossy(&accepted.stderr));
        let checked = child(&root.join("fixtures"), "no", "accepted\n", false)?;
        // The first mismatch stops insta; accept the second output on its next run.
        if !checked.status.success() {
            let pending = child(&root.join("fixtures"), "new", "accepted\n", false)?;
            assert!(!pending.status.success());
            let accepted = Command::new("cargo")
                .args(["insta", "accept", "--manifest-path"])
                .arg(root.join("Cargo.toml"))
                .output()?;
            assert!(accepted.status.success());
        }
        let checked = child(&root.join("fixtures"), "no", "accepted\n", false)?;
        assert!(checked.status.success(), "{}", String::from_utf8_lossy(&checked.stdout));
        Ok(())
    })
}

#[test]
fn rejects_orphan_snapshot_artifacts() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-artifacts-{}", std::process::id()));
    let expected = root.join("control/expected");
    fs::create_dir_all(&expected)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        fs::write(root.join("control/model.rs"), "")?;
        for artifact in [
            "report.txt",
            "report.snap.txt",
            "report.snap",
            "report.snap.new.txt",
            "report.snap.new",
        ] {
            let content = if matches!(artifact, "report.snap" | "report.snap.new") {
                "---\nextension: txt\nsnapshot_kind: binary\n---\n"
            } else {
                "orphan\n"
            };
            fs::write(expected.join(artifact), content)?;
            let error = umpire_testing::run_at(
                &root,
                vec![("control", || Ok(Files::new()))],
                &[umpire_testing::CHECKER],
            )
            .err()
            .ok_or("orphan artifact silently accepted")?;
            assert!(error.to_string().contains(artifact), "{artifact}: {error}");
            assert!(error.to_string().contains(&*expected.to_string_lossy()), "{error}");
            fs::remove_file(expected.join(artifact))?;
        }
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn skipped_backend_requires_selected_complete_goldens() -> Result<(), Box<dyn Error>> {
    with_control("skipped-consumer", |root, expected| {
        let updated = child(&root.join("fixtures"), "always", "first\n", false)?;
        assert!(updated.status.success());
        let backend = Consumer {
            name: "external",
            expected: &["artifact.tla", "artifact.cfg"],
            render: |_, _| panic!("skipped backend executed"),
        };
        umpire_testing::run_at_skipping(
            &root.join("fixtures"),
            vec![("control", || panic!("unselected checker executed"))],
            &[backend],
            &["external"],
        )?;
        fs::write(expected.join("artifact@cfg.snap.new.cfg"), "orphan\n")?;
        let result = umpire_testing::run_at_skipping(
            &root.join("fixtures"),
            vec![("control", || panic!("unselected checker executed"))],
            &[Consumer {
                name: "external",
                expected: &["artifact.tla", "artifact.cfg"],
                render: |_, _| panic!("skipped backend executed"),
            }],
            &["external"],
        );
        assert!(result.is_err(), "an orphan pending snapshot passed while skipped");
        Ok(())
    })
}
