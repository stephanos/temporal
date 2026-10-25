//! External artifact ownership preserves native golden validation.

use std::{fs, path::Path, process::Command};
use umpire_testing::{ExternalConsumer, Files, TestResult, run_at_skipping_with_external};

const LIVE: ExternalConsumer =
    ExternalConsumer { name: "TestLiveArtifact", fixture: "control", expected: "live.txt.snap" };

#[test]
fn external_runner_child() -> TestResult {
    let Ok(root) = std::env::var("UMPIRE_EXTERNAL_ROOT") else { return Ok(()) };
    run_at_skipping_with_external(
        Path::new(&root),
        vec![("control", || Ok(Files::from([("report.txt".into(), "native\n".into())])))],
        &[umpire_testing::CHECKER],
        &[],
        &[LIVE],
    )
}

fn with_fixture(name: &str, check: impl FnOnce(&Path, &Path) -> TestResult) -> TestResult {
    let root = std::env::temp_dir().join(format!("umpire-external-{name}-{}", std::process::id()));
    let expected = root.join("control/expected");
    fs::create_dir_all(&expected)?;
    fs::write(root.join("control/model.rs"), "")?;
    fs::write(expected.join("report.txt.snap"), "---\nsource: control.rs\n---\nnative\n")?;
    fs::write(expected.join("live.txt.snap"), "---\nsource: control.rs\n---\nlive proof\n")?;
    let result = check(&root, &expected);
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn external_artifact_is_unchanged_during_native_update_and_compare() -> TestResult {
    with_fixture("preserve", |root, expected| {
        for update in ["always", "no"] {
            let result = Command::new(std::env::current_exe()?)
                .args(["--exact", "external_runner_child", "--nocapture"])
                .env("UMPIRE_EXTERNAL_ROOT", root)
                .env("INSTA_UPDATE", update)
                .output()?;
            assert!(result.status.success(), "{}", String::from_utf8_lossy(&result.stderr));
            let diagnostic = String::from_utf8_lossy(&result.stderr);
            assert!(diagnostic.contains("TestLiveArtifact"), "{diagnostic}");
            assert!(diagnostic.contains("no execution or comparison evidence"), "{diagnostic}");
            assert_eq!(
                umpire_testing::snapshot_payload(&fs::read_to_string(
                    expected.join("live.txt.snap")
                )?)?,
                "live proof\n"
            );
            assert!(!expected.join("live.snap").exists());
        }
        Ok(())
    })
}

#[test]
fn external_declarations_refuse_invalid_ownership_and_paths() -> TestResult {
    with_fixture("invalid", |root, expected| {
        for (fixture, artifact, name, diagnostic) in [
            ("absent", "live.txt.snap", "TestLiveArtifact", "unknown external fixture"),
            ("control", "absent.txt.snap", "TestLiveArtifact", "missing external artifact"),
            ("control", "live.txt.snap", "", "invalid external consumer"),
            ("control", "../live.txt", "TestLiveArtifact", "invalid external artifact"),
            ("control", "./live.txt", "TestLiveArtifact", "invalid external artifact"),
            ("control", "nested/live.txt", "TestLiveArtifact", "invalid external artifact"),
            ("control", "live.snap.txt", "TestLiveArtifact", "invalid external artifact"),
            (
                "control",
                "report.txt.snap",
                "TestLiveArtifact",
                "external artifact has native consumer",
            ),
        ] {
            let error = run_at_skipping_with_external(
                root,
                vec![("control", || Ok(Files::new()))],
                &[umpire_testing::CHECKER],
                &[],
                &[ExternalConsumer { name, fixture, expected: artifact }],
            )
            .err()
            .ok_or("invalid ownership passed")?;
            assert!(error.to_string().contains(diagnostic), "{fixture}/{artifact}: {error}");
        }
        let error = run_at_skipping_with_external(
            root,
            vec![("control", || Ok(Files::new()))],
            &[umpire_testing::CHECKER],
            &[],
            &[LIVE, LIVE],
        )
        .err()
        .ok_or("duplicate external ownership passed")?;
        assert!(error.to_string().contains("duplicate external artifact"), "{error}");
        fs::write(expected.join("unknown.txt"), "unknown\n")?;
        let error = run_at_skipping_with_external(
            root,
            vec![("control", || Ok(Files::new()))],
            &[umpire_testing::CHECKER],
            &[],
            &[LIVE],
        )
        .err()
        .ok_or("unknown golden passed")?;
        assert!(error.to_string().contains("expected file has no consumer"), "{error}");
        fs::remove_file(expected.join("unknown.txt"))?;
        let error = run_at_skipping_with_external(
            root,
            vec![("control", || Ok(Files::from([("live.txt".into(), "forged\n".into())])))],
            &[umpire_testing::CHECKER],
            &[],
            &[LIVE],
        )
        .err()
        .ok_or("native producer supplied external evidence")?;
        assert!(error.to_string().contains("native producer emitted external artifact"), "{error}");
        assert_eq!(
            umpire_testing::snapshot_payload(&fs::read_to_string(expected.join("live.txt.snap"))?)?,
            "live proof\n"
        );
        fs::write(expected.join("live.txt.snap.new"), "pending")?;
        let error = run_at_skipping_with_external(
            root,
            vec![("control", || Ok(Files::new()))],
            &[umpire_testing::CHECKER],
            &[],
            &[LIVE],
        )
        .err()
        .ok_or("pending external snapshot passed")?;
        assert!(error.to_string().contains("pending golden"), "{error}");
        fs::remove_file(expected.join("live.txt.snap.new"))?;
        fs::remove_file(expected.join("report.txt.snap"))?;
        let error = run_at_skipping_with_external(
            root,
            vec![("control", || panic!("external-only fixture ran native producer"))],
            &[umpire_testing::CHECKER],
            &[],
            &[LIVE],
        )
        .err()
        .ok_or("external-only fixture bypassed native golden requirement")?;
        assert!(
            error.to_string().contains("fixture has no consumer-owned expected files"),
            "{error}"
        );
        Ok(())
    })
}
