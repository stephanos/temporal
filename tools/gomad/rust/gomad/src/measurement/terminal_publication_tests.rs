use super::*;
use crate::measurement::terminal_collection::Attempt;
use std::path::PathBuf;

pub(crate) fn fixtures(bytes: usize) -> (Vec<Record<Observation>>, build::Admission) {
    let identity = build::NativeIdentity {
        schema: "fixture".into(),
        role: "release".into(),
        features: vec![],
        host_implementation_sha256: "a".repeat(64),
        linked_implementation_sha256: "b".repeat(64),
        runtime_implementation_sha256: "c".repeat(64),
        configuration: "fixture".into(),
        source_inputs: BTreeMap::new(),
        executable_sha256: "d".repeat(64),
    };
    let admission = build::Admission {
        schema: "fixture".into(),
        build_receipt: "/fixture/build.json".into(),
        build_receipt_sha256: "e".repeat(64),
        cargo_artifact: "/fixture/artifact".into(),
        cargo_artifact_sha256: "d".repeat(64),
        executable: "/fixture/executable".into(),
        executable_sha256: "d".repeat(64),
        identity,
    };
    let records = (0..8)
        .map(|index| {
            let observation = if index < 6 {
                let root = NativeExecution {
                    outcome: json!({"z":1,"a":"雪"}),
                    streams: BTreeMap::from([(
                        "choice".into(),
                        (0..bytes).map(|i| i.to_le_bytes()[0]).collect(),
                    )]),
                    handle: Some(json!({"slot":index})),
                };
                Observation::Release {
                    observation: Box::new(NativeRun {
                        request: json!({"z":false,"a":[1,2]}),
                        report: json!({"exit":0}),
                        profile: json!({"cpu":7}),
                        incarnations: vec![NativeExecution {
                            outcome: json!({"exit":0}),
                            streams: BTreeMap::new(),
                            handle: None,
                        }],
                        root,
                        executed_steps: Some(json!([1, 2])),
                        cutoff: Some(json!({"why":"fixture"})),
                        modules: vec![json!({"z":2,"a":1})],
                    }),
                    process_exit: json!({"terminal_to_reap_nanos":123}),
                }
            } else {
                Observation::AdmissionRefusal {
                    observation: json!({"label":index,"refused":true}),
                }
            };
            Record {
                attempt: Attempt {
                    label: format!("attempt-{index}"),
                    native_root: PathBuf::from("/fixture/雪"),
                    role: "fixture".into(),
                },
                execution_failure: None,
                observation: Some(observation),
                retained_paths: vec!["/fixture/report.json".into()],
                verification_failure: None,
            }
        })
        .collect();
    (records, admission)
}

pub(crate) fn output_path(label: &str) -> PathBuf {
    std::env::temp_dir().join(format!(
        "gomad-terminal-publication-{}-{label}",
        crate::campaign::origin::identity().unwrap()
    ))
}

#[test]
fn streamed_terminal_ledger_preserves_canonical_bytes_and_failures() {
    let (records, admission) = fixtures(256);
    for failure in [
        None,
        Some("loaded-module-descriptor: actual retained failure"),
    ] {
        let path = output_path("bytes");
        publish(
            &path,
            &records,
            failure,
            &admission,
            Some(&admission),
            Path::new("/fixture/provenance.json"),
        )
        .unwrap();
        // Reproduce the former Value expansion and canonical encoder independently.
        let payloads: Vec<Value> = records
            .iter()
            .map(|record| match record.observation.as_ref().unwrap() {
                Observation::Release {
                    observation,
                    process_exit,
                } => {
                    json!({"kind":"release","observation":observation,"process_exit":process_exit})
                }
                Observation::AdmissionRefusal { observation } => {
                    json!({"kind":"admission-refusal","observation":observation})
                }
            })
            .collect();
        let mut expected_records = serde_json::to_value(&records).unwrap();
        for (record, payload) in expected_records
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .zip(&payloads)
        {
            record["observation"] = payload.clone();
        }
        let expected = json!({"accepted":failure.is_none(),"admission_refusals":payloads[6..].iter().map(|p| &p["observation"]).collect::<Vec<_>>(),
            "attempts":expected_records,"bound_nanos":1_000_000_000u64,
            "boundary":"actual classified native Outcome through completed linked Store/engine/watchdog cleanup; child supervisor wait retained separately",
            "diagnostic_control_admission":admission,"failure_reasons":failure,"helper_admission":admission,
            "module_provenance":"/fixture/provenance.json","observations":payloads[..6].iter().map(|p| &p["observation"]).collect::<Vec<_>>(),
            "original_control_mapping":{"success":"six admitted native release observations","post-load-authorization-mismatch":"separate native diagnostic module-digest and loaded-module-descriptor admission controls; historical helper wire applicability remains separate"},
            "process_exit_observations":payloads[..6].iter().map(|p| &p["process_exit"]).collect::<Vec<_>>(),"schema":"gomad.native-terminal-exit/v2"});
        assert_eq!(
            std::fs::read(&path).unwrap(),
            crate::evidence::canonical::encode(&expected).unwrap()
        );
        std::fs::remove_file(path).unwrap();
    }
}
