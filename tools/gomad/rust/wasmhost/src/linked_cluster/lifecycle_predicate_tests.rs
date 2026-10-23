use super::*;

const FINISH: &str = "i32.const 512 i32.const 16 call $config drop
    i32.const 995 i32.const 1 i32.store8
    i32.const 992 i32.const 16 call $finish drop";

fn request(nodes: &[&str], body: &str) -> Request {
    let mut request = output_wait_request(1, body, nodes, Action::Crash);
    request.boundaries.clear();
    request
}

fn raw(root: &str, variant: &str, observed: Value) {
    println!(
        "{}",
        json!({
            "schema":"gomad.native-lifecycle-predicate/v1",
            "original_root":root,"variant":variant,"observed":observed
        })
    );
}

#[test]
fn failed_wait_preserves_first_terminal_and_never_starts_later_incarnations() {
    for (body, terminal, code) in [
        ("unreachable", "trap", None),
        ("i32.const 17 call $exit", "exit", Some(17)),
    ] {
        let mut input = request(&["a", "b"], body);
        input.steps.swap(1, 2);
        let declared = input.steps.clone();
        let prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        let execution = execute_prepared(prepared).unwrap();
        assert_eq!(execution.executed_steps, declared[..2]);
        assert_eq!(execution.incarnations.len(), 1);
        assert_eq!(execution.incarnations[0].handle.node, "a");
        assert_eq!(execution.outcome.termination, terminal);
        assert_eq!(execution.outcome.exit_code, code);
        assert_eq!(
            execution.incarnations[0].execution.outcome.termination,
            terminal
        );
        assert_eq!(execution.incarnations[0].execution.outcome.exit_code, code);
        assert_eq!(activity.snapshot().launched_stores, 1);
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        raw(
            "TestOperatorStopsActionsAfterFailedWait",
            terminal,
            json!({
                "steps":execution.executed_steps,"terminal":terminal,"exit_code":code,
                "incarnations":["a/1"],"activity":activity.snapshot()
            }),
        );
    }
}

#[test]
fn requested_stores_are_lazy_and_cancellation_before_execution_has_no_actions() {
    let mut input = request(&["a"], FINISH);
    input.nodes.push("b".into());
    let prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    assert_eq!(activity.snapshot().launched_stores, 0);
    let execution = execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "exit");
    assert_eq!(execution.outcome.exit_code, Some(0));
    assert_eq!(activity.snapshot().launched_stores, 1);
    assert_eq!(execution.incarnations.len(), 1);
    assert_eq!(execution.incarnations[0].handle.node, "a");
    raw(
        "TestProviderClusterStartsOnlyRequestedIncarnations",
        "native-lazy-store",
        json!({
            "declared_nodes":["a","b"],"incarnations":["a/1"],"activity":activity.snapshot()
        }),
    );

    let prepared = prepare(request(&["a"], FINISH)).unwrap();
    let activity = prepared.activity();
    prepared.cancellation.cancel();
    let execution = execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "cancelled");
    assert_eq!(execution.outcome.exit_code, None);
    assert!(execution.executed_steps.is_empty());
    assert!(execution.incarnations.is_empty());
    assert!(execution.world.is_empty());
    assert_eq!(activity.snapshot().launched_stores, 0);
    raw(
        "TestClusterHostTerminationStatus",
        "cancelled-before-start",
        json!({
            "terminal":execution.outcome.termination,"steps":execution.executed_steps,
            "activity":activity.snapshot()
        }),
    );
}

#[test]
fn mandatory_cleanup_keeps_only_authored_start_actions() {
    for mode in [
        "shutdown",
        "scenario-error",
        "context-cancel",
        "action-limit",
    ] {
        let mut input = request(
            if mode == "action-limit" {
                &["b", "a", "c"]
            } else {
                &["b", "a"]
            },
            FINISH,
        );
        input.steps.retain(|step| step.action == Action::Start);
        let declared = input.steps.clone();
        let broker = input.broker.clone();
        let mut prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        match mode {
            "shutdown" => prepared
                .cleanup_after_starts(CleanupMode::Shutdown)
                .unwrap(),
            "scenario-error" => prepared
                .cleanup_after_starts(CleanupMode::Error("scenario broke".into()))
                .unwrap(),
            "context-cancel" => prepared
                .cleanup_after_starts(CleanupMode::ContextCancellation)
                .unwrap(),
            "action-limit" => prepared.action_limit(2).unwrap(),
            _ => unreachable!(),
        }
        let execution = execute_prepared(prepared).unwrap();
        let expected = if mode == "action-limit" {
            &declared[..2]
        } else {
            &declared[..]
        };
        assert_eq!(execution.executed_steps, expected);
        assert!(
            execution
                .executed_steps
                .iter()
                .all(|step| step.action == Action::Start)
        );
        assert_eq!(execution.incarnations.len(), expected.len());
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        if mode == "context-cancel" {
            assert_eq!(execution.outcome.termination, "cancelled");
            assert_eq!(execution.outcome.exit_code, None);
            assert_eq!(execution.incarnations[0].handle.node, "a");
            assert_eq!(execution.incarnations[1].handle.node, "b");
            assert!(execution.world.is_empty());
            assert!(execution.network.is_empty());
        } else {
            let terminal = match mode {
                "scenario-error" => "scenario_failure",
                "action-limit" => "capacity",
                _ => "exit",
            };
            assert_eq!(execution.outcome.termination, terminal);
            for incarnation in &execution.incarnations {
                assert_eq!(incarnation.execution.outcome.termination, "stop");
                assert!(incarnation.interrupted);
            }
            if mode == "action-limit" {
                assert_eq!(
                    execution.outcome.message,
                    "scenario action capacity limit=2 used=2 delta=1"
                );
            }
            if mode == "shutdown" || mode == "action-limit" {
                validate_world(&execution.world, &broker).unwrap();
                assert!(!execution.world.is_empty());
            }
        }
        raw(
            "TestProviderMandatoryCleanupReapsWithoutLifecycleActions",
            mode,
            json!({
                "terminal":execution.outcome.termination,"message":execution.outcome.message,
                "steps":execution.executed_steps,"joined_incarnations":execution.incarnations.len(),
                "activity":activity.snapshot()
            }),
        );
    }
}

#[test]
fn typed_owner_admission_refuses_missing_model_namespace_and_unattributable_handles() {
    for variant in [
        "missing-model",
        "missing-namespace",
        "unknown-step",
        "zero-incarnation",
        "duplicate-handle",
        "model-seed",
    ] {
        let mut input = request(&["a"], FINISH);
        let expected = match variant {
            "missing-model" => {
                input.incarnations[0].request.model = None;
                "cluster requires helper-owned model"
            }
            "missing-namespace" => {
                input.incarnations[0].request.namespace = None;
                "cluster requires sealed namespace"
            }
            "unknown-step" => {
                input.steps[1].handle.node = "unknown".into();
                "unknown cluster lifecycle handle"
            }
            "zero-incarnation" => {
                input.incarnations[0].handle.incarnation = 0;
                "invalid or duplicate cluster handle"
            }
            "duplicate-handle" => {
                let mut duplicate = request(&["a"], FINISH);
                input.incarnations.push(duplicate.incarnations.remove(0));
                "invalid or duplicate cluster handle"
            }
            "model-seed" => {
                input.broker.seed += 1;
                "cluster model seed mismatch"
            }
            _ => unreachable!(),
        };
        let error = match prepare(input) {
            Ok(_) => panic!("accepted {variant}"),
            Err(error) => error,
        };
        assert_eq!(error.to_string(), expected);
        raw(
            if variant == "missing-model" {
                "TestClusterRejectsResultWithoutModelEvidence"
            } else {
                "TestClusterRejectsUnattributableFrames"
            },
            variant,
            json!({"admission_error":error.to_string()}),
        );
    }
}

#[test]
fn retired_framed_guest_callbacks_are_refused_by_native_owner_sink() {
    for callback in [b"call".as_slice(), b"sync".as_slice()] {
        let activity = Activity::default();
        let mut sink = NoRemote(activity.clone());
        let error = sink.write(callback).unwrap_err();
        assert_eq!(error.kind(), io::ErrorKind::Unsupported);
        assert_eq!(
            error.to_string(),
            "linked cluster forbids transport callbacks"
        );
        assert_eq!(activity.snapshot().remote_callback_attempts, 1);
        assert_eq!(activity.snapshot().launched_stores, 0);
        raw(
            "TestClusterRejectsGuestCallbacks",
            std::str::from_utf8(callback).unwrap(),
            json!({
                "error_kind":"unsupported","message":error.to_string(),"activity":activity.snapshot()
            }),
        );
    }
}

#[test]
fn observed_failure_drains_prestarted_peers_without_executing_their_waits() {
    for body in ["unreachable", "i32.const 17 call $exit"] {
        let input = request(&["a", "b"], body);
        let declared = input.steps.clone();
        let prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        let execution = execute_prepared(prepared).unwrap();
        assert_eq!(execution.executed_steps, declared[..3]);
        assert_eq!(execution.incarnations.len(), 2);
        assert_eq!(execution.incarnations[0].handle.node, "a");
        assert_eq!(execution.incarnations[1].handle.node, "b");
        let failed = &execution.incarnations[0].execution.outcome;
        assert_eq!(execution.outcome.termination, failed.termination);
        assert_eq!(execution.outcome.exit_code, failed.exit_code);
        assert_eq!(execution.outcome.message, failed.message);
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        raw(
            "TestClusterShutdownWaitsForObservedFailureDrain",
            failed.termination,
            json!({
                "steps":execution.executed_steps,"joined_incarnations":["a/1","b/1"],
                "terminal":execution.outcome.termination,"exit_code":execution.outcome.exit_code,
                "message":execution.outcome.message,"activity":activity.snapshot()
            }),
        );
    }
}

#[test]
fn sealed_deadline_is_bounded_before_mutation_and_cannot_be_extended() {
    let mut input = request(&["a"], FINISH);
    input.wall_timeout = Duration::from_secs(11 * 60);
    let mut prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    assert_eq!(prepared.wall_timeout(), Duration::from_secs(11 * 60));
    prepared
        .bound_wall_timeout(Duration::from_secs(7 * 60))
        .unwrap();
    assert_eq!(prepared.wall_timeout(), Duration::from_secs(7 * 60));
    prepared
        .bound_wall_timeout(Duration::from_secs(11 * 60))
        .unwrap();
    assert_eq!(prepared.wall_timeout(), Duration::from_secs(7 * 60));
    assert!(prepared.bound_wall_timeout(Duration::ZERO).is_err());
    assert_eq!(prepared.wall_timeout(), Duration::from_secs(7 * 60));
    assert_eq!(activity.snapshot().launched_stores, 0);
    raw(
        "TestOpenClusterRecordsHelperExecutionDeadline",
        "native-sealed-deadline",
        json!({
            "sealed_millis":11*60*1000,"bounded_millis":7*60*1000,
            "zero_refused":true,"activity":activity.snapshot()
        }),
    );
}

#[test]
fn cancellation_after_actual_output_keeps_joined_incarnations_and_start_prefix() {
    let mut input = request(&["b", "a"], "(loop br 0)");
    input.steps.retain(|step| step.action == Action::Start);
    let declared = input.steps.clone();
    for incarnation in &mut input.incarnations {
        incarnation.request.limits.fuel = 4_000_000_000;
    }
    let mut prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    prepared
        .cleanup_after_output(
            Handle {
                node: "b".into(),
                incarnation: 1,
            },
            b"mark".to_vec(),
            CleanupMode::ContextCancellation,
        )
        .unwrap();
    let execution = execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "cancelled");
    assert_eq!(execution.outcome.exit_code, None);
    assert_eq!(execution.executed_steps, declared);
    assert_eq!(execution.incarnations.len(), 2);
    assert_eq!(execution.incarnations[0].handle.node, "a");
    let acknowledged = &execution.incarnations[1];
    assert_eq!(acknowledged.handle.node, "b");
    assert_eq!(acknowledged.execution.evidence["stdout"], b"mark");
    assert!(
        acknowledged
            .execution
            .authorization_to_terminal_nanos
            .is_some()
    );
    assert!(activity.snapshot().launched_stores >= 1);
    assert_eq!(activity.snapshot().remote_callback_attempts, 0);
    assert!(execution.world.is_empty());
    assert!(execution.network.is_empty());
    raw(
        "TestProviderMandatoryCleanupReapsWithoutLifecycleActions",
        "context-cancel-after-actual-output",
        json!({
            "terminal":execution.outcome.termination,"steps":execution.executed_steps,
            "joined_incarnations":["a/1","b/1"],"acknowledged_stdout":"mark",
            "store_dropped_before_return":acknowledged.execution.authorization_to_terminal_nanos.is_some(),
            "activity":activity.snapshot()
        }),
    );
}

#[test]
fn synchronous_observer_cancellation_preserves_before_and_after_refusals() {
    struct Observer {
        cancellation: linked::Cancellation,
        activity: Activity,
        calls: Mutex<Vec<(u64, bool)>>,
        refuse: (u64, bool),
    }
    impl Observer {
        fn observe(
            &self,
            phase: &LifecyclePhase,
            before: bool,
        ) -> std::result::Result<(), String> {
            assert_eq!(phase.handles.len(), 2);
            self.calls.lock().unwrap().push((phase.incarnation, before));
            if self.refuse == (phase.incarnation, before) {
                assert_eq!(
                    self.activity.snapshot().launched_stores,
                    if self.refuse == (1, true) { 0 } else { 2 }
                );
                self.cancellation.cancel();
                return Err("admission refused".into());
            }
            Ok(())
        }
    }
    impl LifecycleObserver for Observer {
        fn before(&self, phase: &LifecyclePhase) -> std::result::Result<(), String> {
            self.observe(phase, true)
        }
        fn after(&self, phase: &LifecyclePhase) -> std::result::Result<(), String> {
            self.observe(phase, false)
        }
    }
    for slow_first in [false, true] {
        for refuse in [(1, true), (1, false), (2, true)] {
            let prepared = prepare(phase_schedule_request(slow_first)).unwrap();
            let observer = Observer {
                cancellation: prepared.cancellation.clone(),
                activity: prepared.activity(),
                calls: Mutex::new(Vec::new()),
                refuse,
            };
            let error = execute_prepared_observed(prepared, Some(&observer))
                .err()
                .unwrap();
            assert_eq!(error.to_string(), "admission refused");
            assert!(observer.cancellation.cancelled());
            let expected = [(1, true), (1, false), (2, true)];
            let through = expected.iter().position(|value| *value == refuse).unwrap() + 1;
            assert_eq!(*observer.calls.lock().unwrap(), expected[..through]);
            assert_eq!(observer.activity.snapshot().remote_callback_attempts, 0);
        }
    }
}
