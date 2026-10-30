//! Consumer selection and insta artifacts are exercised through the real runner.

#![forbid(unsafe_code)]

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
    fs::write(expected.join("artifact.tla.snap"), "---\nsource: control.rs\n---\nseed\n")?;
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
            let payload = expected.join(format!("artifact.{extension}.snap"));
            assert_eq!(
                fs::read_to_string(&payload)
                    .map_err(|e| format!(
                        "{}: {e}; runner: {}",
                        payload.display(),
                        String::from_utf8_lossy(&updated.stdout)
                    ))?
                    .split_once("\n---\n")
                    .ok_or("missing text snapshot header")?
                    .1,
                "first\n"
            );
            assert!(!expected.join(format!("artifact.{extension}")).exists());
        }
        let checked = child(&root.join("fixtures"), "no", "first\n", false)?;
        assert!(checked.status.success(), "{}", String::from_utf8_lossy(&checked.stdout));
        fs::write(expected.join("report.txt.snap"), "---\nsource: control.rs\n---\nseed\n")?;
        let mixed = child(&root.join("fixtures"), "always", "first\n", true)?;
        assert!(mixed.status.success(), "{}", String::from_utf8_lossy(&mixed.stdout));
        assert_eq!(
            fs::read_to_string(expected.join("report.txt.snap"))?
                .split_once("\n---\n")
                .ok_or("missing header")?
                .1,
            "checker selected\n"
        );
        assert!(!expected.join("report.txt").exists());
        let mixed = child(&root.join("fixtures"), "no", "first\n", true)?;
        assert!(mixed.status.success(), "{}", String::from_utf8_lossy(&mixed.stdout));
        Ok(())
    })
}

#[test]
fn pending_text_snapshots_require_explicit_acceptance() -> Result<(), Box<dyn Error>> {
    with_control("accept-consumers", |root, expected| {
        assert!(child(&root.join("fixtures"), "always", "first\n", false)?.status.success());
        for extension in ["cfg", "tla"] {
            let pending = child(&root.join("fixtures"), "new", "accepted\n", false)?;
            assert!(!pending.status.success(), "mismatching snapshot silently passed");
            let path = expected.join(format!("artifact.{extension}.snap.new"));
            let snapshot = insta::Snapshot::from_file(&path)?;
            assert!(
                !child(&root.join("fixtures"), "always", "first\n", false)?.status.success(),
                "pending snapshot passed during update"
            );
            snapshot.save(&expected.join(format!("artifact.{extension}.snap")))?;
            fs::remove_file(path)?;
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
            "report.txt.snap",
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
        for (file, content, diagnostic) in [
            ("artifact.cfg.snap.new", Some("pending\n"), "pending golden"),
            ("artifact.cfg.snap", Some("malformed\n"), "incomplete golden"),
            ("artifact.cfg.snap", None, "incomplete golden"),
        ] {
            let path = expected.join(file);
            let original = fs::read(&path).ok();
            if let Some(content) = content {
                fs::write(&path, content)?;
            } else {
                fs::remove_file(&path)?;
            }
            let error = umpire_testing::run_at_skipping(
                &root.join("fixtures"),
                vec![("control", || panic!("skipped producer executed"))],
                &[Consumer {
                    name: "external",
                    expected: &["artifact.tla", "artifact.cfg"],
                    render: |_, _| panic!("skipped backend executed"),
                }],
                &["external"],
            )
            .err()
            .ok_or("incomplete or pending skipped snapshot passed")?;
            assert!(error.to_string().contains(diagnostic), "{error}");
            if let Some(original) = original {
                fs::write(path, original)?;
            } else {
                fs::remove_file(path)?;
            }
        }
        Ok(())
    })
}

fn with_selection_control(
    name: &str,
    check: impl FnOnce(&Path, Vec<umpire_testing::Fixture>) -> umpire_testing::TestResult,
) -> umpire_testing::TestResult {
    with_control(name, |root, _| {
        let fixtures = root.join("fixtures");
        let expected = fixtures.join("other/expected");
        fs::create_dir_all(&expected)?;
        fs::write(fixtures.join("other/model.rs"), "")?;
        fs::write(expected.join("artifact.tla.snap"), "---\nsource: control.rs\n---\nseed\n")?;
        check(
            &fixtures,
            vec![
                ("control", || panic!("consumer must use its renderer")),
                ("other", || panic!("unselected producer executed")),
            ],
        )
    })
}

#[test]
fn selected_fixture_renders_twice_without_running_its_siblings() -> umpire_testing::TestResult {
    use std::sync::atomic::{AtomicUsize, Ordering};
    static RENDERS: AtomicUsize = AtomicUsize::new(0);

    with_selection_control("selected-renders", |root, fixtures| {
        RENDERS.store(0, Ordering::SeqCst);
        umpire_testing::run_fixture_at_skipping(
            root,
            fixtures,
            &[Consumer {
                name: "control",
                expected: &["artifact.tla"],
                render: |_, folder| {
                    assert_eq!(folder.file_name().and_then(|name| name.to_str()), Some("control"));
                    RENDERS.fetch_add(1, Ordering::SeqCst);
                    Ok(Files::from([("artifact.tla".into(), "seed\n".into())]))
                },
            }],
            &[],
            "control",
        )?;
        assert_eq!(RENDERS.load(Ordering::SeqCst), 2);
        Ok(())
    })
}

#[test]
fn selection_admits_all_registry_and_snapshot_inputs_before_rendering() -> umpire_testing::TestResult
{
    with_selection_control("selected-admission", |root, fixtures| {
        let consumers = [Consumer {
            name: "control",
            expected: &["artifact.tla"],
            render: |_, _| panic!("render preceded full admission"),
        }];
        let expected = root.join("other/expected");
        for (file, content, diagnostic) in [
            ("artifact.tla.snap.new", "pending\n", "pending golden"),
            ("orphan.txt.snap", "---\nsource: control.rs\n---\nseed\n", "no consumer"),
            ("artifact.tla.snap", "malformed\n", "invalid golden"),
        ] {
            let path = expected.join(file);
            let original = fs::read(&path).ok();
            fs::write(&path, content)?;
            let error = umpire_testing::run_fixture_at_skipping(
                root,
                fixtures.clone(),
                &consumers,
                &[],
                "control",
            )
            .err()
            .ok_or("invalid unselected fixture passed admission")?;
            assert!(error.to_string().contains(diagnostic), "{error}");
            assert!(error.to_string().contains("other"), "{error}");
            if let Some(original) = original {
                fs::write(path, original)?;
            } else {
                fs::remove_file(path)?;
            }
        }
        for (registry, selected, diagnostic) in [
            (vec![fixtures[0]], "control", "registry"),
            (vec![fixtures[0], fixtures[0], fixtures[1]], "control", "registry"),
            (fixtures.clone(), "missing", "unknown selected fixture"),
        ] {
            let error =
                umpire_testing::run_fixture_at_skipping(root, registry, &consumers, &[], selected)
                    .err()
                    .ok_or("invalid fixture selection passed")?;
            assert!(error.to_string().contains(diagnostic), "{error}");
        }
        let overlapping = [
            Consumer { name: "one", expected: &["artifact.tla"], render: |_, _| panic!("render") },
            Consumer { name: "two", expected: &["*.tla"], render: |_, _| panic!("render") },
        ];
        let error =
            umpire_testing::run_fixture_at_skipping(root, fixtures, &overlapping, &[], "control")
                .err()
                .ok_or("ambiguous consumers passed")?;
        assert!(error.to_string().contains("multiple consumers"), "{error}");
        Ok(())
    })
}

#[test]
fn selected_fixture_preserves_dynamic_orphan_and_nondeterminism_refusals()
-> umpire_testing::TestResult {
    use std::sync::atomic::{AtomicUsize, Ordering};
    static RENDERS: AtomicUsize = AtomicUsize::new(0);

    with_selection_control("selected-refusals", |root, fixtures| {
        for (render, diagnostic) in [
            (
                (|_, _| Ok(Files::new())) as fn(fn() -> RunResult, &Path) -> RunResult,
                "golden has no fixture",
            ),
            (
                |_, _| Ok(Files::from([("undeclared.txt".into(), "seed\n".into())])),
                "undeclared output",
            ),
            (|_, _| Err("selected producer refused".into()), "selected producer refused"),
            (
                |_, _| {
                    Ok(Files::from([(
                        "artifact.tla".into(),
                        RENDERS.fetch_add(1, Ordering::SeqCst).to_string(),
                    )]))
                },
                "nondeterministic control producer",
            ),
        ] {
            let error = umpire_testing::run_fixture_at_skipping(
                root,
                fixtures.clone(),
                &[Consumer { name: "control", expected: &["artifact.tla"], render }],
                &[],
                "control",
            )
            .err()
            .ok_or("invalid selected output passed")?;
            assert!(error.to_string().contains(diagnostic), "{error}");
        }
        Ok(())
    })
}

#[test]
fn selected_fixture_admits_skipped_sibling_goldens() -> umpire_testing::TestResult {
    with_selection_control("selected-skipped", |root, fixtures| {
        let consumers = [Consumer {
            name: "external",
            expected: &["artifact.tla", "artifact.cfg"],
            render: |_, _| panic!("skipped consumer executed"),
        }];
        fs::write(
            root.join("control/expected/artifact.cfg.snap"),
            "---\nsource: control.rs\n---\nseed\n",
        )?;
        let error = umpire_testing::run_fixture_at_skipping(
            root,
            fixtures,
            &consumers,
            &["external"],
            "control",
        )
        .err()
        .ok_or("incomplete skipped sibling passed")?;
        assert!(error.to_string().contains("incomplete golden"), "{error}");
        assert!(error.to_string().contains("other"), "{error}");
        Ok(())
    })
}

#[test]
fn update_mode_repairs_corrupt_consumer_owned_snapshots() -> umpire_testing::TestResult {
    with_control("repair-consumer-snapshot", |root, expected| {
        let snapshot = expected.join("artifact.tla.snap");
        fs::write(&snapshot, "truncated snapshot\n")?;
        let refused = child(&root.join("fixtures"), "no", "repaired\n", false)?;
        assert!(!refused.status.success());
        let diagnostics = format!(
            "{}{}",
            String::from_utf8_lossy(&refused.stdout),
            String::from_utf8_lossy(&refused.stderr)
        );
        assert!(diagnostics.contains("invalid golden"), "{diagnostics}");
        let repaired = child(&root.join("fixtures"), "always", "repaired\n", false)?;
        assert!(repaired.status.success(), "{repaired:?}");
        assert_eq!(
            umpire_testing::snapshot_payload(&fs::read_to_string(&snapshot)?)?,
            "repaired\n"
        );
        assert!(child(&root.join("fixtures"), "no", "repaired\n", false)?.status.success());
        Ok(())
    })
}

#[test]
fn selected_update_child() -> umpire_testing::TestResult {
    let Some(root) = std::env::var_os("UMPIRE_SELECTED_UPDATE_ROOT") else { return Ok(()) };
    let mut consumers = vec![Consumer {
        name: "control",
        expected: &["artifact.tla"],
        render: |_, folder| {
            assert!(matches!(
                folder.file_name().and_then(|name| name.to_str()),
                Some("control" | "other")
            ));
            Ok(Files::from([("artifact.tla".into(), "repaired\n".into())]))
        },
    }];
    let skipped = std::env::var_os("UMPIRE_SELECTED_UPDATE_SKIP").is_some();
    if skipped {
        consumers.push(Consumer {
            name: "skipped",
            expected: &["artifact.cfg"],
            render: |_, _| panic!("skipped producer executed"),
        });
    }
    let selected =
        std::env::var("UMPIRE_SELECTED_UPDATE_FIXTURE").unwrap_or_else(|_| "control".into());
    umpire_testing::run_fixture_at_skipping(
        Path::new(&root),
        vec![
            ("control", || panic!("consumer must use its renderer")),
            ("other", || panic!("unselected producer executed")),
        ],
        &consumers,
        if skipped { &["skipped"] } else { &[] },
        &selected,
    )
}

#[test]
fn selected_updates_admit_repairable_siblings_and_preserve_skipped_goldens()
-> umpire_testing::TestResult {
    with_selection_control("selected-update-repair", |root, _| {
        let spawn = |update: &str,
                     skipped: bool,
                     selected: &str|
         -> Result<std::process::Output, Box<dyn Error>> {
            let mut command = Command::new(std::env::current_exe()?);
            command
                .args(["--exact", "selected_update_child", "--nocapture"])
                .env("UMPIRE_SELECTED_UPDATE_ROOT", root)
                .env("INSTA_UPDATE", update)
                .env("UMPIRE_SELECTED_UPDATE_FIXTURE", selected)
                .env_remove("UMPIRE_SELECTED_UPDATE_SKIP");
            if skipped {
                command.env("UMPIRE_SELECTED_UPDATE_SKIP", "1");
            }
            Ok(command.output()?)
        };
        let selected = root.join("control/expected/artifact.tla.snap");
        let sibling = root.join("other/expected/artifact.tla.snap");
        fs::write(&selected, "truncated selected snapshot\n")?;
        let refused = spawn("no", false, "control")?;
        assert!(!refused.status.success());
        assert_eq!(fs::read_to_string(&selected)?, "truncated selected snapshot\n");
        let repaired = spawn("always", false, "control")?;
        assert!(repaired.status.success(), "{repaired:?}");
        assert_eq!(
            umpire_testing::snapshot_payload(&fs::read_to_string(&selected)?)?,
            "repaired\n"
        );
        assert_eq!(umpire_testing::snapshot_payload(&fs::read_to_string(&sibling)?)?, "seed\n");
        fs::write(&sibling, "truncated unselected snapshot\n")?;
        let refused = spawn("no", false, "control")?;
        assert!(!refused.status.success());
        let diagnostics = format!(
            "{}{}",
            String::from_utf8_lossy(&refused.stdout),
            String::from_utf8_lossy(&refused.stderr)
        );
        assert!(
            diagnostics.contains("invalid golden") && diagnostics.contains("other"),
            "{diagnostics}"
        );
        let selected_update = spawn("always", false, "control")?;
        assert!(selected_update.status.success(), "{selected_update:?}");
        assert_eq!(fs::read_to_string(&sibling)?, "truncated unselected snapshot\n");
        let owning_update = spawn("always", false, "other")?;
        assert!(owning_update.status.success(), "{owning_update:?}");
        assert_eq!(umpire_testing::snapshot_payload(&fs::read_to_string(&sibling)?)?, "repaired\n");
        let skipped = root.join("control/expected/artifact.cfg.snap");
        fs::write(&skipped, "truncated skipped snapshot\n")?;
        let refused = spawn("always", true, "control")?;
        assert!(!refused.status.success());
        let diagnostics = format!(
            "{}{}",
            String::from_utf8_lossy(&refused.stdout),
            String::from_utf8_lossy(&refused.stderr)
        );
        assert!(diagnostics.contains("incomplete golden"), "{diagnostics}");
        assert_eq!(fs::read_to_string(&skipped)?, "truncated skipped snapshot\n");
        Ok(())
    })
}
