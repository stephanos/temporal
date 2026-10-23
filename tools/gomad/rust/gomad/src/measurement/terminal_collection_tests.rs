use super::*;
use crate::measurement::build;
use serde_json::json;
use std::{
    collections::BTreeMap,
    os::unix::fs::{DirBuilderExt, PermissionsExt},
    path::Path,
    sync::atomic::{AtomicU64, Ordering},
    time::Duration,
};
static NEXT: AtomicU64 = AtomicU64::new(0);
struct Directory(PathBuf);
impl Directory {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-terminal-collection-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&path).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o700)).unwrap();
        Self(path)
    }
    fn attempts(&self) -> Vec<Attempt> {
        ["success", "loaded-module-descriptor"]
            .into_iter()
            .map(|label| Attempt {
                label: label.into(),
                role: "fixture".into(),
                native_root: self.0.join(label),
            })
            .collect()
    }
}
impl Drop for Directory {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn save(path: &Path, raw: &[u8]) {
    std::fs::write(path, raw).unwrap();
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600)).unwrap();
}
#[test]
fn terminal_failure_retains_each_partial_report_stderr_and_aggregate() {
    for publication_failure in [false, true] {
        let root = Directory::new();
        let attempts = root.attempts();
        let mut executions = 0;
        let mut verifications = 0;
        let aggregate = root.0.join("terminal-exit.json");
        let result = collect(
            &attempts,
            |attempt| {
                executions += 1;
                std::fs::DirBuilder::new()
                    .mode(0o700)
                    .create(&attempt.native_root)
                    .unwrap();
                save(&attempt.native_root.join("report.json"),br#"{"execution":{"Err":"helper transport failed"},"partial_terminal":"retained actual bytes"}"#);
                save(
                    &root.0.join(format!("{}-driver.stderr", attempt.label)),
                    b"helper diagnostics",
                );
                Err("helper transport failed".into())
            },
            |_| {
                verifications += 1;
                Ok(())
            },
            |records, failure| {
                assert!(failure.unwrap().contains("helper transport failed"));
                save(
                    &aggregate,
                    &serde_json::to_vec(
                        &json!({"accepted":false,"attempts":records,"failure_reasons":failure}),
                    )
                    .unwrap(),
                );
                if publication_failure {
                    Err("aggregate failure".into())
                } else {
                    Ok(())
                }
            },
        );
        let error = result.unwrap_err();
        assert!(error.contains("helper transport failed"));
        assert_eq!(error.contains("aggregate failure"), publication_failure);
        assert_eq!((executions, verifications), (2, 2));
        let retained: Value = serde_json::from_slice(&std::fs::read(&aggregate).unwrap()).unwrap();
        assert_eq!(retained["attempts"].as_array().unwrap().len(), 2);
        for (index, attempt) in attempts.iter().enumerate() {
            let record = &retained["attempts"][index];
            assert!(record["observation"].is_null());
            assert_eq!(record["execution_failure"], "helper transport failed");
            assert!(
                record["retained_paths"]
                    .as_array()
                    .unwrap()
                    .contains(&json!(attempt.native_root.join("report.json")))
            );
            assert!(
                record["retained_paths"]
                    .as_array()
                    .unwrap()
                    .contains(&json!(
                        root.0.join(format!("{}-driver.stderr", attempt.label))
                    ))
            );
            assert_eq!(
                std::fs::read(root.0.join(format!("{}-driver.stderr", attempt.label))).unwrap(),
                b"helper diagnostics"
            );
            let partial: Value = serde_json::from_slice(
                &std::fs::read(attempt.native_root.join("report.json")).unwrap(),
            )
            .unwrap();
            assert_eq!(partial["partial_terminal"], "retained actual bytes");
        }
    }
}
#[test]
fn failed_identity_verification_blocks_observation_acceptance_and_still_collects_next() {
    let root = Directory::new();
    let attempts = root.attempts();
    let mut executed = 0;
    let mut verified = 0;
    let result = collect(
        &attempts,
        |_| {
            executed += 1;
            Ok(json!({"actual_observation":"available"}))
        },
        |_| {
            verified += 1;
            Err("source changed".into())
        },
        |records, failure| {
            assert!(failure.unwrap().contains("source changed"));
            assert!(records.iter().all(|record| record.observation.is_none()
                && record.verification_failure.as_deref() == Some("source changed")));
            Ok(())
        },
    );
    assert!(result.unwrap_err().contains("source changed"));
    assert_eq!((executed, verified), (2, 2));
}
#[test]
fn actual_supervised_child_failures_still_publish_terminal_aggregate() {
    let root = Directory::new();
    let attempts = root.attempts();
    let aggregate = root.0.join("aggregate.json");
    let mut verifications = 0;
    let result = collect(
        &attempts,
        |attempt| {
            std::fs::DirBuilder::new()
                .mode(0o700)
                .create(&attempt.native_root)
                .unwrap();
            let observed=build::observe_command(&root.0,&format!("{}-driver",attempt.label),&root.0,
            &BTreeMap::from([("REPORT_OUTPUT".into(),attempt.native_root.join("report.json").to_string_lossy().into_owned())]),
            &["/bin/sh".into(),"-c".into(),r#"umask 077; printf '%s' '{"execution":{"Err":"controlled child failure"}}' > "$REPORT_OUTPUT"; printf 'actual child diagnostics' >&2; exit 7"#.into()],Duration::from_secs(5),1024);
            assert_eq!(observed.exit_code, Some(7));
            assert!(observed.pid.is_some());
            assert!(observed.reaped_monotonic_nanos.is_some());
            Err(observed.error.unwrap())
        },
        |attempt| {
            verifications += 1;
            let observed: build::CommandObservation = serde_json::from_slice(
                &std::fs::read(root.0.join(format!("{}-driver.json", attempt.label))).unwrap(),
            )
            .unwrap();
            assert_eq!(observed.termination, "exit_failure");
            assert_eq!(observed.cleanup_errors, Vec::<String>::new());
            Ok(())
        },
        |records, failure| {
            save(
                &aggregate,
                &serde_json::to_vec(&json!({"attempts":records,"failure_reasons":failure}))
                    .unwrap(),
            );
            Ok(())
        },
    );
    assert!(result.is_err());
    assert_eq!(verifications, 2);
    for attempt in attempts {
        assert_eq!(
            std::fs::read(root.0.join(format!("{}-driver.stderr", attempt.label))).unwrap(),
            b"actual child diagnostics"
        );
        let report: Value = serde_json::from_slice(
            &std::fs::read(attempt.native_root.join("report.json")).unwrap(),
        )
        .unwrap();
        assert_eq!(report["execution"]["Err"], "controlled child failure");
    }
    assert!(aggregate.is_file());
}

#[test]
fn successful_attempts_preserve_payloads_and_return_the_published_result() {
    let root = Directory::new();
    let attempts = root.attempts();
    let result = collect(
        &attempts,
        |attempt| Ok(json!({"label":attempt.label})),
        |_| Ok(()),
        |records, failure| {
            assert_eq!(failure, None);
            assert_eq!(records.len(), 2);
            assert_eq!(records[0].observation, Some(json!({"label":"success"})));
            assert_eq!(
                records[1].observation,
                Some(json!({"label":"loaded-module-descriptor"}))
            );
            assert!(
                records
                    .iter()
                    .all(|record| record.execution_failure.is_none()
                        && record.verification_failure.is_none()
                        && record.retained_paths.is_empty())
            );
            Ok("published")
        },
    );
    assert_eq!(result, Ok("published"));
}
