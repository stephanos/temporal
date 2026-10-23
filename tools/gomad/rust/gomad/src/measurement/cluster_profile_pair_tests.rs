use super::*;
use std::collections::BTreeMap;

fn fixture() -> Value {
    let mut profile = json!({"schema":"gomad.wasm-profile/v2","diagnostic_only":true,"backend":"cranelift-jit","host_os":"linux","host_arch":"aarch64",
        "phases":{},"abi":{},"nested":{},"residual":{"instantiate_wall_nanos":0,"execute_wall_nanos":70},"abi_overlap_errors":0,
        "abi_excluded":{"count":0,"wall_nanos":0,"instantiate_wall_nanos":0,"execute_wall_nanos":0},"abi_ops":{},"owners":{},
        "reconciliation":{"execute_wall_nanos":100,"attributed_host_wall_nanos":30,"guest_residual_wall_nanos":70,"outer_host_wall_nanos":30,"difference_nanos":0,"tolerance_nanos":50000,"distribution_relative_error_ppm":62500},
        "threads":[],"threads_omitted":0,"unattributed_imports":0,"unframed_imports":0,"span_errors":0,"elapsed_wall_nanos":100,"wall_basis":"single-store elapsed","report_rows_dropped":0});
    for name in "module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute".split_whitespace() {
        profile["phases"][name] = json!({"count":0,"wall_nanos":0,"user_nanos":0,"system_nanos":0,"cpu_samples":0,"peak_rss_bytes_high_water":0});
    }
    profile["phases"]["execute"] = json!({"count":1,"wall_nanos":100,"user_nanos":70,"system_nanos":10,"cpu_samples":1,"peak_rss_bytes_high_water":1024});
    for name in
        "clock scheduler select observation idle other_local remote_callback".split_whitespace()
    {
        profile["abi"][name] = profile["abi_excluded"].clone();
    }
    profile["abi"]["clock"] =
        json!({"count":1,"wall_nanos":30,"instantiate_wall_nanos":0,"execute_wall_nanos":30});
    for name in "alternative_hash encoding transcript_hash flush remote_write remote_read_wait"
        .split_whitespace()
    {
        profile["nested"][name] = json!({"count":0,"wall_nanos":0});
    }
    profile["nested"]["encoding"] = json!({"count":1,"wall_nanos":5});
    let series = json!({"count":1,"wall_nanos":30,"exclusive_wall_nanos":30,"instantiate_wall_nanos":0,"execute_wall_nanos":30,"execute_exclusive_wall_nanos":30,"min_nanos":30,"median_nanos":30,"p99_nanos":30,"max_nanos":30});
    profile["owners"]["clock"] = series.clone();
    profile["abi_ops"]["wasi_snapshot_preview1.clock_time_get"] = series;
    profile["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["served_by"] = json!({"clock":1});
    crate::profile::validate(&profile).unwrap();
    profile
}

#[test]
fn original_five_pair_cases_and_zero_measured_do_not_fabricate_counters() {
    let raw = canonical::encode(&fixture()).unwrap();
    for (second, crashed, complete, measured, fails) in [
        (raw.as_slice(), false, true, 2, false),
        (b"".as_slice(), true, false, 1, false),
        (b"{".as_slice(), true, false, 1, false),
        (b"".as_slice(), false, false, 1, true),
        (b"invalid".as_slice(), false, false, 1, true),
    ] {
        let result = decode(&[
            Capture {
                raw: &raw,
                crashed: false,
                reaped: true,
            },
            Capture {
                raw: second,
                crashed,
                reaped: true,
            },
        ]);
        if fails {
            let partial = result.err().unwrap().summary;
            assert_eq!(partial["measured_captures"], 1);
            assert_eq!(partial["complete"], false);
            assert_eq!(
                partial["measured_sum"]["phases"]["execute"]["wall_nanos"],
                100
            );
            assert!(!partial["instances"][0]["profile"].is_null());
            assert!(partial["instances"][1]["profile"].is_null());
            continue;
        }
        let result = result.unwrap();
        assert_eq!(result["complete"], complete);
        assert_eq!(result["measured_captures"], measured);
        for (category, bucket, field, value) in [
            ("phases", "execute", "user_nanos", 70),
            ("phases", "execute", "system_nanos", 10),
            ("abi", "clock", "execute_wall_nanos", 30),
            ("nested", "encoding", "wall_nanos", 5),
        ] {
            assert_eq!(
                result["measured_sum"][category][bucket][field],
                value * measured
            );
        }
        assert_eq!(
            result["measured_sum"]["residual"]["execute_wall_nanos"],
            70 * measured
        );
        assert_eq!(
            result["measured_sum"]["phases"]["execute"]["peak_rss_bytes_high_water"],
            1024
        );
    }
    let result = decode(&[
        Capture {
            raw: b"",
            crashed: true,
            reaped: true,
        },
        Capture {
            raw: b"{",
            crashed: true,
            reaped: true,
        },
    ])
    .unwrap();
    assert_eq!(result["measured_captures"], 0);
    assert!(result["measured_sum"].is_null());
    assert_eq!(result["complete"], false);
    for capture in [
        Capture {
            raw: b"",
            crashed: true,
            reaped: false,
        },
        Capture {
            raw: &vec![b' '; (16 << 10) + 1],
            crashed: true,
            reaped: true,
        },
    ] {
        assert!(decode(&[capture]).is_err());
    }
}

#[test]
fn original_pair_overflow_host_identity_and_negative_counter_refuse() {
    let raw = canonical::encode(&fixture()).unwrap();
    for kind in 0..3 {
        let mut changed = fixture();
        match kind {
            0 => changed["phases"]["compile"]["count"] = json!(u64::MAX),
            1 => changed["host_arch"] = json!("x86_64"),
            _ => changed["phases"]["execute"]["user_nanos"] = json!(-1),
        }
        let changed = canonical::encode(&changed).unwrap();
        let first = if kind == 0 {
            changed.as_slice()
        } else {
            raw.as_slice()
        };
        assert!(
            decode(&[
                Capture {
                    raw: first,
                    crashed: false,
                    reaped: true
                },
                Capture {
                    raw: &changed,
                    crashed: false,
                    reaped: true
                }
            ])
            .is_err()
        );
    }
}

#[test]
fn original_owner_work_and_wait_remain_distinct_and_checked() {
    let mut owners = json!({"scheduler_handoff":{"execute_exclusive_wall_nanos":5},"clock":{"execute_exclusive_wall_nanos":7},
        "handoff_wait":{"execute_exclusive_wall_nanos":100},"instrumentation":{"execute_exclusive_wall_nanos":3}});
    assert_eq!(
        owner_split(&owners).unwrap(),
        json!({"host_work_nanos":12,"waiting_nanos":103})
    );
    for name in ["handoff_wait", "instrumentation"] {
        assert!(gomad_wasmhost::profile_inventory::owner_names().contains(&name));
    }
    for (name, field) in [
        ("clock", "host_work_nanos"),
        ("handoff_wait", "waiting_nanos"),
    ] {
        owners[name]["execute_exclusive_wall_nanos"] = json!(8);
        assert_ne!(
            owner_split(&owners).unwrap()[field],
            if field == "host_work_nanos" {
                json!(12)
            } else {
                json!(103)
            }
        );
    }
    owners["clock"]["execute_exclusive_wall_nanos"] = json!(u64::MAX);
    assert!(owner_split(&owners).is_err());
}

#[test]
fn linked_store_identity_and_single_process_resource_scope_are_explicit() {
    use super::super::super::evidence::NativeExecution;
    let mut profile = fixture();
    profile["threads"]=json!([10,10,5,5].into_iter().enumerate().map(|(index,host)|json!({
        "slot":index+1,"incarnation":index/2+1,"execute_wall_nanos":25,"attributed_host_wall_nanos":host,
        "guest_residual_wall_nanos":25-host,"outer_host_wall_nanos":host,"difference_nanos":0,"tolerance_nanos":50000})).collect::<Vec<_>>());
    profile["wall_basis"] =
        json!("aggregate store thread-time; overlapping spans are not elapsed shares");
    crate::profile::validate(&profile).unwrap();
    let handles: Vec<_> = [1, 2]
        .into_iter()
        .flat_map(|incarnation| {
            ["active", "standby"].map(move |node| json!({"node":node,"incarnation":incarnation}))
        })
        .collect();
    let run = NativeRun {
        request: json!({"nodes":["active","standby"],"incarnations":handles.iter().map(|handle|json!({"handle":handle})).collect::<Vec<_>>()}),
        report: Value::Null,
        profile,
        root: NativeExecution {
            outcome: json!({"termination":"exit","exit_code":0}),
            streams: BTreeMap::new(),
            handle: None,
        },
        incarnations: handles
            .into_iter()
            .map(|handle| NativeExecution {
                outcome: json!({"termination":"exit","exit_code":0}),
                streams: BTreeMap::new(),
                handle: Some(handle),
            })
            .collect(),
        executed_steps: None,
        cutoff: None,
        modules: vec![],
    };
    let result = summarize(&run).unwrap();
    assert_eq!(result["measured_captures"], 1);
    assert_eq!(result["stores"].as_array().unwrap().len(), 4);
    assert_eq!(
        result["measured_sum"]["phases"]["execute"]["user_nanos"],
        70
    );
    assert_eq!(
        result["measured_sum"]["phases"]["execute"]["peak_rss_bytes_high_water"],
        1024
    );
    for kind in 0..3 {
        let mut changed = run.clone();
        match kind {
            0 => changed.profile["threads"][3]["slot"] = json!(99),
            1 => changed.profile["threads"][3]["incarnation"] = json!(3),
            _ => {
                changed.incarnations.pop();
            }
        }
        assert!(summarize(&changed).is_err());
    }
    let mut zero = fixture();
    zero["phases"]["execute"] = json!({"count":0,"wall_nanos":0,"user_nanos":0,"system_nanos":0,"cpu_samples":0,"peak_rss_bytes_high_water":0});
    zero["owners"] = json!({});
    zero["abi_ops"] = json!({});
    zero["abi"]["clock"] = zero["abi_excluded"].clone();
    zero["residual"]["execute_wall_nanos"] = json!(0);
    for field in [
        "execute_wall_nanos",
        "attributed_host_wall_nanos",
        "guest_residual_wall_nanos",
        "outer_host_wall_nanos",
    ] {
        zero["reconciliation"][field] = json!(0);
    }
    assert!(
        crate::profile::validate(&zero)
            .unwrap_err()
            .contains("no execute interval")
    );
}

#[test]
fn actual_reaped_capture_failure_retains_availability_without_fabricated_aggregate() {
    use std::os::unix::fs::DirBuilderExt;
    let root = std::env::temp_dir().join(format!(
        "gomad-profile-pair-reap-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    let evidence = root.join("evidence");
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&evidence)
        .unwrap();
    for (name, script, killed) in [
        ("killed", "kill -KILL $$", true),
        ("successful-missing", "exit 0", false),
    ] {
        let actual = super::super::super::build::command_bounded(
            &evidence,
            &format!("{name}-driver"),
            &root,
            &BTreeMap::new(),
            &["/bin/sh".into(), "-c".into(), script.into()],
            std::time::Duration::from_secs(5),
            1 << 20,
        );
        assert_eq!(actual.is_err(), killed);
        retain_failed_capture(&root, name, true, "controlled process capture failure").unwrap();
        let result: Value = serde_json::from_slice(
            &std::fs::read(root.join(format!("{name}-profile-pair-failure.json"))).unwrap(),
        )
        .unwrap();
        assert_eq!(result["passed"], false);
        assert_eq!(result["measured_captures"], 0);
        assert!(result["measured_sum"].is_null());
        assert_eq!(result["complete"], false);
        assert_eq!(!result["capture_failure"].is_null(), !killed);
        assert!(result["process_id"].as_u64().is_some_and(|pid| pid > 0));
    }
    std::fs::remove_dir_all(root).unwrap();
}

/// Original `TestDiagnosticReportRejectsMalformedOrUnattributedMeasurements`
/// on its literal phase-profile fixture.
#[test]
fn diagnostic_report_rejects_malformed_or_unattributed_measurements() {
    let valid = serde_json::to_string(&fixture()).unwrap();
    let report = crate::profile::decode(valid.as_bytes()).unwrap();
    assert_eq!(report["phases"]["execute"]["user_nanos"], 70);
    assert_eq!(report["abi"]["clock"]["execute_wall_nanos"], 30);
    assert_eq!(report["residual"]["execute_wall_nanos"], 70);
    let replace = |old: &str, new: &str| {
        let changed = valid.replacen(old, new, 1);
        assert_ne!(changed, valid, "{old} absent from fixture");
        changed
    };
    for (name, data) in [
        ("missing", String::new()),
        ("oversized", "x".repeat((48 << 10) + 1)),
        ("duplicate-report", format!("{valid}{valid}")),
        ("trailing", format!("{valid}unexpected\n")),
        ("null", "null".into()),
        (
            "wrong-schema",
            replace("gomad.wasm-profile/v2", "gomad.wasm-profile/v3"),
        ),
        (
            "unknown-top",
            replace("\"schema\":", "\"extra\":0,\"schema\":"),
        ),
        (
            "nested-duplicate",
            replace("\"user_nanos\":70", "\"user_nanos\":1,\"user_nanos\":70"),
        ),
        (
            "escaped-duplicate",
            replace(
                "\"user_nanos\":70",
                "\"\\u0075ser_nanos\":1,\"user_nanos\":70",
            ),
        ),
        ("missing-phase", replace("\"module_read\":", "\"absent\":")),
        ("missing-measurement", replace("\"user_nanos\":70,", "")),
        (
            "unknown-measurement",
            replace("\"user_nanos\":70", "\"user_nanos\":70,\"invented\":0"),
        ),
        (
            "not-diagnostic",
            replace("\"diagnostic_only\":true", "\"diagnostic_only\":false"),
        ),
        ("not-jit", replace("cranelift-jit", "interpreter")),
        (
            "negative",
            replace("\"user_nanos\":70", "\"user_nanos\":-1"),
        ),
        (
            "too-many-cpu-samples",
            replace("\"cpu_samples\":1", "\"cpu_samples\":2"),
        ),
        (
            "overlapping-abi",
            replace("\"abi_overlap_errors\":0", "\"abi_overlap_errors\":1"),
        ),
        (
            "wrong-residual",
            replace(
                "\"residual\":{\"execute_wall_nanos\":70",
                "\"residual\":{\"execute_wall_nanos\":71",
            ),
        ),
    ] {
        assert!(
            crate::profile::decode(data.as_bytes()).is_err(),
            "accepted {name} diagnostic"
        );
    }
}
