//! Suite goldens use the shared runner's snapshot and determinism checks.

use std::{
    error::Error,
    fs,
    path::Path,
    process::Command,
    sync::atomic::{AtomicUsize, Ordering},
};
use umpire_testing::{Files, RunResult, SUITE};

fn suite_files() -> RunResult {
    static CALLS: AtomicUsize = AtomicUsize::new(0);
    let content = if std::env::var_os("UMPIRE_SUITE_NONDETERMINISTIC").is_some() {
        CALLS.fetch_add(1, Ordering::Relaxed).to_string()
    } else {
        std::env::var("UMPIRE_SUITE_CONTENT")?
    };
    let mut files = Files::from([
        ("expansion.txt".into(), content.clone()),
        ("expansion/sample.txt".into(), content.clone()),
        ("coverage.txt".into(), content),
    ]);
    if std::env::var_os("UMPIRE_SUITE_MIXED").is_some() {
        files.insert("report.txt".into(), "checker selected\n".into());
    }
    Ok(files)
}

#[test]
fn runner_child() -> Result<(), Box<dyn Error>> {
    let Ok(root) = std::env::var("UMPIRE_SUITE_ROOT") else { return Ok(()) };
    umpire_testing::run_at(
        Path::new(&root),
        vec![("control", suite_files)],
        &[umpire_testing::CHECKER, SUITE],
    )
}

fn child(
    root: &Path,
    update: &str,
    content: &str,
    deterministic: bool,
    mixed: bool,
) -> Result<std::process::Output, Box<dyn Error>> {
    let mut command = Command::new(std::env::current_exe()?);
    if deterministic {
        command.env_remove("UMPIRE_SUITE_NONDETERMINISTIC");
    } else {
        command.env("UMPIRE_SUITE_NONDETERMINISTIC", "1");
    }
    if mixed {
        command.env("UMPIRE_SUITE_MIXED", "1");
    } else {
        command.env_remove("UMPIRE_SUITE_MIXED");
    }
    Ok(command
        .args(["--exact", "runner_child", "--nocapture"])
        .env("UMPIRE_SUITE_ROOT", root)
        .env("UMPIRE_SUITE_CONTENT", content)
        .env("INSTA_UPDATE", update)
        .output()?)
}

fn with_control(
    name: &str,
    check: impl FnOnce(&Path, &Path) -> umpire_testing::TestResult,
) -> umpire_testing::TestResult {
    let root = std::env::temp_dir().join(format!("umpire-suite-{name}-{}", std::process::id()));
    let expected = root.join("control/expected");
    fs::create_dir_all(&expected)?;
    fs::write(root.join("control/model.rs"), "")?;
    fs::write(expected.join("coverage.snap.txt"), "seed\n")?;
    let result = check(&root, &expected);
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn suite_snapshots_update_compare_and_reject_orphans() -> Result<(), Box<dyn Error>> {
    with_control("snapshots", |root, expected| {
        let updated = child(root, "always", "first\n", true, false)?;
        assert!(updated.status.success(), "{}", String::from_utf8_lossy(&updated.stdout));
        for file in ["expansion.snap.txt", "expansion/sample.snap.txt", "coverage.snap.txt"] {
            assert_eq!(fs::read_to_string(expected.join(file))?, "first\n");
        }
        for _ in 0..2 {
            let checked = child(root, "no", "first\n", true, false)?;
            assert!(checked.status.success(), "{}", String::from_utf8_lossy(&checked.stdout));
        }
        let mismatch = child(root, "no", "changed\n", true, false)?;
        assert!(!mismatch.status.success(), "mismatching suite goldens passed");
        assert!(
            String::from_utf8_lossy(&mismatch.stderr).contains("snapshot assertion"),
            "{}",
            String::from_utf8_lossy(&mismatch.stderr)
        );
        fs::write(expected.join("expansion/orphan.snap.txt"), "orphan\n")?;
        let orphan = child(root, "always", "first\n", true, false)?;
        assert!(!orphan.status.success(), "orphan suite golden passed during update");
        assert!(String::from_utf8_lossy(&orphan.stderr).contains("golden has no fixture"));
        Ok(())
    })
}

#[test]
fn suite_snapshots_reject_nondeterministic_producers() -> Result<(), Box<dyn Error>> {
    with_control("nondeterminism", |root, _| {
        let result = child(root, "always", "", false, false)?;
        assert!(!result.status.success(), "nondeterministic suite producer passed");
        assert!(
            String::from_utf8_lossy(&result.stderr).contains("nondeterministic suite producer")
        );
        Ok(())
    })
}

#[test]
fn checker_and_suite_share_one_fixture_producer() -> Result<(), Box<dyn Error>> {
    with_control("mixed", |root, expected| {
        let unselected = child(root, "always", "suite selected\n", true, true)?;
        assert!(!unselected.status.success(), "unselected checker output silently passed");
        assert!(
            String::from_utf8_lossy(&unselected.stderr).contains("undeclared output report.txt")
        );
        fs::write(expected.join("report.snap.txt"), "seed\n")?;
        let updated = child(root, "always", "suite selected\n", true, true)?;
        assert!(updated.status.success(), "{}", String::from_utf8_lossy(&updated.stderr));
        assert_eq!(fs::read_to_string(expected.join("report.snap.txt"))?, "checker selected\n");
        assert_eq!(fs::read_to_string(expected.join("coverage.snap.txt"))?, "suite selected\n");
        let checked = child(root, "no", "suite selected\n", true, true)?;
        assert!(checked.status.success(), "{}", String::from_utf8_lossy(&checked.stderr));
        Ok(())
    })
}

#[test]
fn suite_producer_rejects_unknown_and_ambiguous_outputs() -> Result<(), Box<dyn Error>> {
    let cases: [(fn() -> RunResult, &str); 2] = [
        (
            || Ok(Files::from([("unknown.txt".into(), "unknown\n".into())])),
            "undeclared output unknown.txt",
        ),
        (
            || Ok(Files::from([("expansion.txt".into(), "ambiguous\n".into())])),
            "produced output has multiple consumers: expansion.txt",
        ),
    ];
    with_control("invalid-owners", |root, _| {
        for (produce, message) in cases {
            let result = umpire_testing::run_at(
                root,
                vec![("control", produce)],
                &[
                    SUITE,
                    umpire_testing::Consumer {
                        name: "duplicate",
                        expected: &["expansion.txt"],
                        render: |_, _| panic!("unselected consumer executed"),
                    },
                ],
            );
            let error = result.err().ok_or("invalid output ownership silently passed")?;
            assert!(error.to_string().contains(message), "{error}");
        }
        Ok(())
    })
}
