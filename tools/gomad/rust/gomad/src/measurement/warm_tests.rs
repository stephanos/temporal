use super::*;
use gomad_wasmhost::linked::Outcome;
use std::time::Duration;

fn binding() -> Binding {
    Binding {
        source_commit: "a".repeat(40),
        source_root: "/checkout".into(),
        source_inputs: BTreeMap::from([("driver.rs".into(), "b".repeat(64))]),
        driver_sha256: "c".repeat(64),
        host_sha256: "d".repeat(64),
        module_sha256: "e".repeat(64),
        config_sha256: "f".repeat(64),
        model_sha256: "1".repeat(64),
        runtime_identity: "runtime".into(),
        host_boot_id: "boot".into(),
        host_cpu: "cpu".into(),
        host_os: "linux".into(),
        host_arch: "x86_64".into(),
        engine_version: "47.0.3".into(),
        engine_configuration: "configuration".into(),
        module_producer_sha256: "2".repeat(64),
        seed: 0,
        artifact_bytes: 10,
        envelope_bytes: 20,
    }
}

fn observation(index: usize) -> Observation {
    let mut outcome = Outcome::new("exit", "");
    outcome.exit_code = Some(0);
    outcome.metrics = Some(Box::default());
    let metrics = outcome.metrics.as_mut().unwrap();
    metrics.prepare_nanos = 100;
    metrics.compile_nanos = if index == 0 { 50 } else { 0 };
    metrics.cache_hit = index != 0;
    let terminal = Instant::now();
    Observation {
        execution: Execution {
            timing: Some(gomad_wasmhost::linked::ExecutionTiming {
                authorization_started: terminal - Duration::from_nanos(10),
                terminal_classified: terminal,
                returned: terminal + Duration::from_nanos(100),
            }),
            #[cfg(feature = "helper-guest-profile")]
            guest_profile_error: None,
            #[cfg(feature = "helper-diagnostic-profile")]
            external_owner_attempts: 0,
            authorization_to_terminal_nanos: Some(10),
            outcome,
            evidence: BTreeMap::from([("stdout.bin".into(), b"result".to_vec())]),
            event_order: None,
        },
        phases: [
            "cache_read",
            "envelope_verify",
            "deserialize",
            "cache_identity",
        ]
        .into_iter()
        .map(|name| {
            (
                name.into(),
                Phase {
                    count: 1,
                    wall_nanos: 10,
                },
            )
        })
        .collect(),
        wall_nanos: 200,
        teardown: TeardownInterval {
            terminal,
            returned: terminal + Duration::from_nanos(100),
            recorded_nanos: 100,
        },
        fresh_store_id: (index + 1) as u64,
    }
}

fn series(role: Role) -> Series {
    collect(binding(), role, 5, |index| Ok(observation(index))).unwrap()
}

#[test]
fn collection_primes_cache_then_collects_five_distinct_native_stores() {
    let mut visited = Vec::new();
    let series = collect(binding(), Role::Candidate, 5, |index| {
        visited.push(index);
        Ok(observation(index))
    })
    .unwrap();
    assert_eq!(visited, [0, 1, 2, 3, 4, 5]);
    assert_eq!(series.prepare_timing().median, 100);
    assert!(!series.warmup.cache_hit);
    assert!(series.runs.iter().all(|run| run.cache_hit));
    compare(&self::series(Role::Baseline), &series).unwrap();
}

#[test]
fn collection_requires_five_repetitions_and_propagates_execution_failures() {
    assert!(
        collect(binding(), Role::Candidate, 4, |_| panic!(
            "must reject before execution"
        ))
        .is_err()
    );
    let mut visited = Vec::new();
    let error = collect(binding(), Role::Candidate, 5, |index| {
        visited.push(index);
        if index == 2 {
            Err("native cache refused".into())
        } else {
            Ok(observation(index))
        }
    })
    .err()
    .unwrap();
    assert_eq!(error, "native cache refused");
    assert_eq!(visited, [0, 1, 2]);
}

#[test]
fn actual_native_outcome_metrics_and_teardown_are_required() {
    for mutation in 0..9 {
        let result = collect(binding(), Role::Candidate, 5, |index| {
            let mut observed = observation(index);
            if index == 1 {
                match mutation {
                    0 => observed.execution.outcome.metrics = None,
                    1 => observed.execution.outcome.exit_code = Some(1),
                    2 => observed.execution.outcome.termination = "trap",
                    3 => {
                        observed
                            .execution
                            .outcome
                            .metrics
                            .as_mut()
                            .unwrap()
                            .cache_hit = false;
                    }
                    4 => {
                        observed
                            .execution
                            .outcome
                            .metrics
                            .as_mut()
                            .unwrap()
                            .compile_nanos = 1;
                    }
                    5 => {
                        observed
                            .execution
                            .outcome
                            .metrics
                            .as_mut()
                            .unwrap()
                            .prepare_nanos = 0;
                    }
                    6 => observed.teardown.returned = observed.teardown.terminal,
                    7 => observed.teardown.recorded_nanos = 0,
                    _ => observed.fresh_store_id = 1,
                }
            }
            Ok(observed)
        });
        assert!(result.is_err(), "mutation {mutation}");
    }
}

#[test]
fn every_warm_cache_phase_requires_an_observed_count_and_consistent_time() {
    for phase in [
        "cache_read",
        "envelope_verify",
        "deserialize",
        "cache_identity",
    ] {
        for mutation in 0..4 {
            let result = collect(binding(), Role::Candidate, 5, |index| {
                let mut observed = observation(index);
                if index == 2 {
                    match mutation {
                        0 => {
                            observed.phases.remove(phase);
                        }
                        1 => observed.phases.get_mut(phase).unwrap().count = 0,
                        2 => observed.phases.get_mut(phase).unwrap().wall_nanos = 101,
                        _ => observed.phases.get_mut(phase).unwrap().wall_nanos = 0,
                    }
                }
                Ok(observed)
            });
            assert!(result.is_err(), "{phase} mutation {mutation}");
        }
    }
}

#[test]
fn candidate_prepare_median_and_each_teardown_are_bounded_by_one_second() {
    for role in [Role::Baseline, Role::Candidate] {
        let result = collect(binding(), role, 5, |index| {
            let mut observed = observation(index);
            observed
                .execution
                .outcome
                .metrics
                .as_mut()
                .unwrap()
                .prepare_nanos = 1_000_000_001;
            Ok(observed)
        });
        assert_eq!(result.is_ok(), role == Role::Baseline);
    }
    let mut after = series(Role::Candidate);
    after.runs[0].teardown_nanos = 1_000_000_001;
    after.runs[0].observed_teardown_nanos = 1_000_000_001;
    assert!(after.validate().is_err());
}

#[test]
fn raw_native_behavior_changes_or_projection_tampering_are_refused() {
    for mutation in 0..6 {
        let mut changed = series(Role::Candidate);
        match mutation {
            0 => changed.runs[0]
                .raw_evidence
                .get_mut("stdout.bin")
                .unwrap()
                .push(1),
            1 => changed.runs[0].raw_outcome["metrics"]["compile_nanos"] = 1.into(),
            2 => changed.runs[0].prepare_nanos += 1,
            3 => changed.runs[0].raw_outcome["exit_code"] = 1.into(),
            4 => changed.runs[0].event_order = Some(vec![1]),
            _ => changed.runs[0].fresh_store_id = changed.runs[1].fresh_store_id,
        }
        assert!(changed.validate().is_err(), "mutation {mutation}");
    }
    assert!(
        collect(binding(), Role::Candidate, 5, |index| {
            let mut observed = observation(index);
            if index == 3 {
                observed
                    .execution
                    .evidence
                    .insert("stderr.bin".into(), b"different".to_vec());
            }
            Ok(observed)
        })
        .is_err()
    );
}

#[test]
fn comparisons_require_same_real_host_module_runtime_model_and_current_producer() {
    let before = series(Role::Baseline);
    for mutation in 0..13 {
        let mut after = series(Role::Candidate);
        match mutation {
            0 => after.binding.host_boot_id.push('x'),
            1 => after.binding.host_cpu.push('x'),
            2 => after.binding.host_os.push('x'),
            3 => after.binding.host_arch.push('x'),
            4 => after.binding.engine_configuration.push('x'),
            5 => after.binding.module_sha256 = "9".repeat(64),
            6 => after.binding.config_sha256 = "9".repeat(64),
            7 => after.binding.runtime_identity.push('x'),
            8 => after.binding.model_sha256 = "9".repeat(64),
            9 => after.binding.module_producer_sha256 = "9".repeat(64),
            10 => after.binding.seed += 1,
            11 => after.role = Role::Baseline,
            _ => after.binding.source_inputs.clear(),
        }
        assert!(compare(&before, &after).is_err(), "mutation {mutation}");
    }
    let mut different_source = series(Role::Candidate);
    different_source.binding.source_commit = "9".repeat(40);
    different_source.binding.driver_sha256 = "9".repeat(64);
    different_source.binding.host_sha256 = "9".repeat(64);
    compare(&before, &different_source).unwrap();
}

#[cfg(feature = "helper-guest-profile")]
#[test]
fn sampler_finalization_failure_does_not_admit_successful_terminal_as_warm_sample() {
    let error = collect(binding(), Role::Candidate, 5, |index| {
        let mut observed = observation(index);
        observed.execution.guest_profile_error =
            Some("native sampler file publication failed".into());
        Ok(observed)
    })
    .err()
    .unwrap();
    assert!(error.contains("sampler file publication failed"));
}

struct RetainedCache(std::path::PathBuf);
impl RetainedCache {
    fn new() -> Self {
        use std::os::unix::fs::DirBuilderExt;
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "gomad-warm-retained-unit-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&path)
            .unwrap();
        Self(std::fs::canonicalize(path).unwrap())
    }
}
impl Drop for RetainedCache {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn retained(index: usize, cache: &RetainedCache) -> NativeRun {
    let observed = observation(index);
    let outcome = serde_json::to_value(&observed.execution.outcome).unwrap();
    let store = index as u64 + 1;
    let implementation = "f".repeat(64);
    NativeRun {
        request: serde_json::json!({"kind":"standalone","compile_cache":cache.0,"guest":{"module":"/retained/module.wasm","module_sha256":"e".repeat(64),"model":null,"limits":{"fuel":1000}}}),
        report: serde_json::json!({"diagnostic_only":true,"features":["diagnostic-profile"],"host_sha256":"d".repeat(64),"host_implementation_sha256":"c".repeat(64),"linked_implementation_sha256":implementation,"host_os":"linux","host_arch":"x86_64","configuration":format!("{}-diagnostic-profile",gomad_wasmhost::linked::CONFIGURATION),"fresh_store_id":store,"terminal_to_return_nanos":100,"timing":{"schema":NATIVE_TIMING_SCHEMA,"scope":NATIVE_TIMING_SCOPE,"clock":"monotonic","implementation_sha256":implementation,"fresh_store_id":store,"terminal_to_return_nanos":100}}),
        profile: serde_json::json!({"elapsed_wall_nanos":1000,"phases":observed.phases}),
        root: evidence::NativeExecution {
            outcome,
            streams: observed.execution.evidence,
            handle: None,
        },
        incarnations: Vec::new(),
        executed_steps: None,
        cutoff: None,
        modules: Vec::new(),
    }
}
fn retained_binding(cache: &RetainedCache) -> Binding {
    let mut binding = binding();
    binding.config_sha256 = retained_configuration_sha256(&retained(0, cache).request).unwrap();
    binding.engine_configuration = format!(
        "{}-diagnostic-profile",
        gomad_wasmhost::linked::CONFIGURATION
    );
    binding
}

#[test]
fn retained_collector_uses_bound_native_monotonic_interval_and_five_fresh_cache_hits() {
    let cache = RetainedCache::new();
    let series = collect_retained(retained_binding(&cache), Role::Candidate, 5, |index| {
        Ok(retained(index, &cache))
    })
    .unwrap();
    assert_eq!(series.runs.len(), 5);
    assert_eq!(series.prepare_timing().median, 100);
    for row in &series.runs {
        assert!(row.native.is_some());
        assert_eq!(row.teardown_nanos, 100);
        assert_eq!(
            row.observed_teardown_nanos, 0,
            "no independent clock measurement is fabricated"
        );
    }
    let baseline = collect_retained(retained_binding(&cache), Role::Baseline, 5, |index| {
        Ok(retained(index, &cache))
    })
    .unwrap();
    compare(&baseline, &series).unwrap();
}

#[test]
fn retained_collector_refuses_missing_timing_source_and_altered_cache_or_closure() {
    let cache = RetainedCache::new();
    for mutation in 0..10 {
        let error = collect_retained(retained_binding(&cache), Role::Candidate, 5, |index| {
            let mut run = retained(index, &cache);
            if index == 1 {
                match mutation {
                    0 => {
                        run.report.as_object_mut().unwrap().remove("timing");
                    }
                    1 => run.report["timing"]["clock"] = "wall-clock".into(),
                    2 => run.report["timing"]["scope"] = "whole-helper-wall".into(),
                    3 => run.report["timing"]["implementation_sha256"] = "a".repeat(64).into(),
                    4 => run.report["timing"]["fresh_store_id"] = 999.into(),
                    5 => run.root.outcome["metrics"]["cache_hit"] = false.into(),
                    6 => run.root.outcome["metrics"]["compile_nanos"] = 1.into(),
                    7 => run.request["guest"]["limits"]["fuel"] = 1001.into(),
                    8 => run.report["host_sha256"] = "a".repeat(64).into(),
                    _ => run.profile["phases"]["deserialize"]["count"] = 0.into(),
                }
            }
            Ok(run)
        })
        .err()
        .unwrap();
        assert!(!error.is_empty(), "mutation {mutation}");
    }
}

#[test]
fn direct_native_timing_cannot_understate_the_one_second_boundary() {
    let mut observed = observation(1);
    let terminal = observed.teardown.terminal;
    let returned = terminal + Duration::from_nanos(1_000_500_000);
    observed.execution.timing.as_mut().unwrap().returned = returned;
    observed.teardown.returned = returned;
    observed.teardown.recorded_nanos = 1_000_000_000;
    assert!(Run::observed(observed).is_err());
    let mut run = Run::observed(observation(1)).unwrap();
    run.observed_teardown_nanos = 1_000_500_000;
    run.teardown_nanos = 1_000_000_000;
    assert!(run.validate(true, false).is_err());
}
