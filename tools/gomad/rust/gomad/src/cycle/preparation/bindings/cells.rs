use crate::cycle::{custody, preparation::inputs::workloads};
use serde_json::{Value, json};
use std::path::Path;

pub(crate) fn cell_spec(
    label: &str,
    evidence: &Path,
    manifest: &Path,
    source_digest: &str,
    bindings: &Value,
) -> Result<Value, String> {
    let (workload, profile, cell, identity) = cell(label)?;
    let digest = source_digest
        .strip_prefix("sha256:")
        .unwrap_or(source_digest);
    if !custody::valid_digest(digest) {
        return Err("invalid cell source digest".into());
    }
    custody::clean_absolute(evidence)?;
    custody::clean_absolute(manifest)?;
    let repo = Path::new(
        bindings["repo"]
            .as_str()
            .ok_or("cell candidate repo missing")?,
    );
    if matches!(workload, "sqlite" | "multi-instance") {
        let replication = workload == "multi-instance";
        let source = if replication {
            let base = repo.join("tools/gomad/internal/backends/wasm/testdata");
            let mut bytes =
                custody::read_file(&base.join("temporal_storage/recovery_test.go"), 16 << 20)?;
            bytes.extend(custody::read_file(
                &base.join("temporal_broker/replication_test.go"),
                16 << 20,
            )?);
            Some(format!(
                "./.tmp/fn232-temporal-replication-{}",
                &custody::digest(&bytes)[..16]
            ))
        } else {
            None
        };
        let working = replication.then(|| {
            Path::new(bindings["root"].as_str().unwrap_or(""))
                .join("stages")
                .join(label)
                .join("source-module")
        });
        if let Some(path) = &working {
            custody::clean_absolute(path)?;
        }
        let mut spec = workloads::qualification_spec(
            repo,
            evidence,
            if replication { "replication" } else { "sqlite" },
            &workloads::QualificationOptions {
                canary: cell == "canary",
                source: source.as_deref(),
                source_manifest: Some(manifest),
                guest_profile: bindings["environment"]["GOMAD_WASM_QUALIFICATION_GUEST_PROFILE"]
                    .as_str()
                    .unwrap_or(""),
                source_working_dir: working.as_deref(),
                original_clusters: false,
            },
        )?;
        spec["Manifest"]["Profile"] = json!(profile);
        if spec["Manifest"]["IntegratedSourceSHA256"] != format!("sha256:{digest}") {
            return Err("cell source manifest differs from frozen source".into());
        }
        return Ok(spec);
    }
    let onebox = workload == "onebox";
    let mut target = json!({"Backend":"wasm","Kind":if onebox{"go-test"}else{"go-run"},"Source":if onebox{"./tests"}else{"."},"WorkingDir":if onebox{repo.to_owned()}else{repo.join("tools/gomad/internal/backends/wasm/qualification/testdata/fixture")},"BuildTags":if onebox{vec!["disable_grpc_modules","gomad_wasm_slim","sqlite3_dotlk","test_dep"]}else{vec!["test_dep"]},"PreparationRoot":evidence.join("preparation")});
    target["Args"] = if onebox {
        json!([
            "-test.v",
            "-test.count=1",
            "-test.timeout=72h",
            format!(
                "-test.run={}",
                if cell == "canary" {
                    "["
                } else {
                    "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
                }
            ),
            "-persistenceType=sql",
            "-persistenceDriver=sqlite"
        ])
    } else if cell == "canary" {
        json!(["canary"])
    } else {
        json!([])
    };
    let mut campaign = json!({"Target":target,"ExecutionTimeout":if onebox{900_000_000_000_u64}else{60_000_000_000},"OverallTimeout":if onebox{1_800_000_000_000_u64}else{180_000_000_000},"TerminateGrace":if onebox{1_000_000_000}else{100_000_000},"OutputLimit":if onebox{256<<20}else{1<<20},"WorldTransitionLimit":if onebox{32<<20}else{1<<20},"OnFailure":"all","FailureBudget":1,"Artifacts":evidence.join("artifacts")});
    if onebox {
        campaign["ChoiceTraceLimit"] = json!(64 << 20);
        campaign["SuccessBytesLimit"] = json!(2_u64 << 30);
        campaign["Environment"] = json!([
            "TEMPORAL_ROOT=/workspace",
            "TMPDIR=/tmp",
            "TEMPORAL_TEST_LOG_LEVEL=warn"
        ]);
    }
    Ok(
        json!({"Manifest":{"Provisional":false,"IntegratedSourceManifestPath":manifest,"IntegratedSourceSHA256":format!("sha256:{digest}"),"ID":identity,"Workload":workload,"Profile":profile,"Cell":cell,"Seed":7,"Repeat":2},"Campaign":campaign,"ArtifactRoot":evidence}),
    )
}

fn cell(label: &str) -> Result<(&str, &str, &str, &str), String> {
    for (workload, profile, identity) in [
        ("fixture", "controlled", "ready-select-timer"),
        ("fixture", "cooperative", "ready-select-timer"),
        ("sqlite", "cooperative", "sqlite-durable-recovery"),
        ("onebox", "cooperative", "unchanged-workflow-type-smoke"),
        (
            "multi-instance",
            "cooperative",
            "independent-temporal-replication-durable-restart",
        ),
    ] {
        for cell in ["success", "canary"] {
            if label == format!("workload-{workload}-{profile}-{cell}") {
                return Ok((workload, profile, cell, identity));
            }
        }
    }
    Err("unknown fixed workload cell".into())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn fixed_fixture_and_onebox_cells_preserve_limits_and_canaries() {
        let config = json!({"repo":"/candidate"});
        for workload in [
            "fixture-controlled",
            "fixture-cooperative",
            "onebox-cooperative",
        ] {
            for cell in ["success", "canary"] {
                let spec = cell_spec(
                    &format!("workload-{workload}-{cell}"),
                    Path::new("/evidence"),
                    Path::new("/source.manifest"),
                    &"a".repeat(64),
                    &config,
                )
                .unwrap();
                assert_eq!(spec["Manifest"]["Cell"], cell);
                assert_eq!(spec["Manifest"]["Repeat"], 2);
                assert_eq!(spec["Manifest"]["Provisional"], false);
                assert_eq!(spec["Campaign"]["FailureBudget"], 1);
                if workload.starts_with("onebox") {
                    assert_eq!(spec["Campaign"]["ExecutionTimeout"], 900_000_000_000_u64);
                    assert_eq!(spec["Campaign"]["SuccessBytesLimit"], 2_u64 << 30);
                    assert_eq!(
                        spec["Campaign"]["Target"]["Args"][3],
                        if cell == "canary" {
                            "-test.run=["
                        } else {
                            "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
                        }
                    );
                } else {
                    assert_eq!(
                        spec["Campaign"]["Target"]["Args"],
                        if cell == "canary" {
                            json!(["canary"])
                        } else {
                            json!([])
                        }
                    );
                }
            }
        }
        assert!(
            cell_spec(
                "workload-fixture-controlled-other",
                Path::new("/evidence"),
                Path::new("/manifest"),
                &"a".repeat(64),
                &config
            )
            .unwrap_err()
            .contains("unknown fixed workload")
        );
    }
}
