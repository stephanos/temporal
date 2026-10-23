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
    let mut row = json!({"schema":"gomad.native-lifecycle-predicate/v1","original_root":root,"variant":variant});
    row.as_object_mut()
        .unwrap()
        .insert("observed".into(), observed);
    println!("{row}");
}

fn store_result_fact(item: &IncarnationResult) -> Value {
    json!({"handle":item.handle,"terminal":item.execution.outcome.termination,
        "exit_code":item.execution.outcome.exit_code,"message":item.execution.outcome.message,
        "authorization_to_terminal_nanos":item.execution.authorization_to_terminal_nanos,
        "interrupted":item.interrupted,"stdout":item.execution.evidence.get("stdout")})
}

fn store_facts(execution: &Execution) -> Vec<Value> {
    execution
        .incarnations
        .iter()
        .map(store_result_fact)
        .collect()
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
                "incarnations":["a/1"],"store_results":store_facts(&execution),"activity":activity.snapshot()
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
            "declared_nodes":["a","b"],"incarnations":["a/1"],"store_results":store_facts(&execution),"activity":activity.snapshot()
        }),
    );

    let prepared = prepare(request(&["a"], FINISH)).unwrap();
    let activity = prepared.activity();
    prepared.cancellation.cancel();
    let execution = execute_prepared(prepared).unwrap();
    assert_eq!(execution.outcome.termination, "cancelled");
    assert_eq!(execution.outcome.exit_code, None);
    assert_eq!(execution.executed_steps.len(), 0);
    assert!(execution.incarnations.is_empty());
    assert_eq!(execution.world, [] as [u8; 0]);
    assert_eq!(activity.snapshot().launched_stores, 0);
    raw(
        "TestClusterHostTerminationStatus",
        "cancelled-before-start",
        json!({
            "terminal":execution.outcome.termination,"steps":execution.executed_steps,
            "store_results":store_facts(&execution),"activity":activity.snapshot()
        }),
    );
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The original cleanup matrix keeps authored actions, joined ownership and raw mode observations together"
)]
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
            "i32.const 512 i32.const 16 call $config drop
            (loop
                i32.const 768 i64.const 0 i64.store
                i32.const 776 i64.const 0 i64.store
                i32.const 771 i32.const 1 i32.store8
                i32.const 768 i32.const 16 call $idle drop br 0)",
        );
        input.steps.retain(|step| step.action == Action::Start);
        input.nodes.push("dormant".into());
        for item in &mut input.incarnations {
            item.request.limits.fuel = u64::MAX;
        }
        let declared = input.steps.clone();
        let broker = input.broker.clone();
        let mut prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        let cleanup = match mode {
            "shutdown" => CleanupMode::Shutdown,
            "scenario-error" => CleanupMode::Error("scenario broke".into()),
            "context-cancel" => CleanupMode::ContextCancellation,
            "action-limit" => {
                prepared.action_limit(2).unwrap();
                CleanupMode::Shutdown
            }
            _ => unreachable!(),
        };
        prepared
            .cleanup_after_output(
                Handle {
                    node: "b".into(),
                    incarnation: 1,
                },
                b"mark".to_vec(),
                cleanup,
            )
            .unwrap();
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
        assert_eq!(activity.snapshot().launched_stores, expected.len() as u64);
        let acknowledged = execution
            .incarnations
            .iter()
            .find(|item| item.handle.node == "b")
            .unwrap();
        assert_eq!(acknowledged.execution.evidence["stdout"], b"mark");
        assert!(
            execution
                .incarnations
                .iter()
                .all(|item| item.execution.authorization_to_terminal_nanos.is_some())
        );
        assert!(
            execution
                .incarnations
                .iter()
                .all(|item| item.handle.node != "dormant" && item.handle.node != "c")
        );
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        if mode == "context-cancel" {
            assert_eq!(execution.outcome.termination, "cancelled");
            assert_eq!(execution.outcome.exit_code, None);
            assert_eq!(execution.incarnations[0].handle.node, "a");
            assert_eq!(execution.incarnations[1].handle.node, "b");
            assert_eq!(execution.world, [] as [u8; 0]);
            assert_eq!(execution.network, [] as [u8; 0]);
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
                assert_ne!(execution.world, [] as [u8; 0]);
            }
        }
        raw(
            "TestProviderMandatoryCleanupReapsWithoutLifecycleActions",
            mode,
            json!({
                "terminal":execution.outcome.termination,"message":execution.outcome.message,
                "steps":execution.executed_steps,"joined_incarnations":execution.incarnations.len(),
                "acknowledged_handle":acknowledged.handle,
                "acknowledged_stdout":acknowledged.execution.evidence["stdout"],
                "all_stores_dropped_before_return":execution.incarnations.iter().all(|item|
                    item.execution.authorization_to_terminal_nanos.is_some()),
                "dormant_started":false,"store_results":store_facts(&execution),"activity":activity.snapshot()
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
        let Err(error) = prepare(input) else {
            panic!("accepted {variant}")
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
                "message":execution.outcome.message,"store_results":store_facts(&execution),"activity":activity.snapshot()
            }),
        );
    }
}

#[test]
fn sealed_deadline_is_bounded_before_mutation_and_cannot_be_extended() {
    let mut input = request(&["a"], FINISH);
    input.wall_timeout = Duration::from_mins(11);
    let mut prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    assert_eq!(prepared.wall_timeout(), Duration::from_mins(11));
    prepared.bound_wall_timeout(Duration::from_mins(7)).unwrap();
    assert_eq!(prepared.wall_timeout(), Duration::from_mins(7));
    prepared
        .bound_wall_timeout(Duration::from_mins(11))
        .unwrap();
    assert_eq!(prepared.wall_timeout(), Duration::from_mins(7));
    assert!(prepared.bound_wall_timeout(Duration::ZERO).is_err());
    assert_eq!(prepared.wall_timeout(), Duration::from_mins(7));
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
    let mut input = request(&["b", "a"], FINISH);
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
    assert_eq!(activity.snapshot().launched_stores, 2);
    assert!(
        execution
            .incarnations
            .iter()
            .all(|item| item.execution.authorization_to_terminal_nanos.is_some())
    );
    assert_eq!(activity.snapshot().remote_callback_attempts, 0);
    assert_eq!(execution.world, [] as [u8; 0]);
    assert_eq!(execution.network, [] as [u8; 0]);
    raw(
        "TestProviderMandatoryCleanupReapsWithoutLifecycleActions",
        "context-cancel-after-actual-output",
        json!({
            "terminal":execution.outcome.termination,"steps":execution.executed_steps,
            "joined_incarnations":["a/1","b/1"],"acknowledged_stdout":"mark",
            "store_dropped_before_return":acknowledged.execution.authorization_to_terminal_nanos.is_some(),
            "store_results":store_facts(&execution),"activity":activity.snapshot()
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
        fn observe(&self, phase: &LifecyclePhase, before: bool) -> std::result::Result<(), String> {
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

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The bounded original refusal matrix keeps each failed admission and same-owner retry in one scope"
)]
fn refused_restart_preserves_owner_and_allows_retry_on_same_broker() {
    for variant in [
        "missing-storage-image",
        "stale-incarnation",
        "live-incarnation",
    ] {
        let mut input = request(&["a", "b"], FINISH);
        let model: crate::volume::StorageModel = serde_json::from_value(json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":4096}],
            "limits":{"PendingOperations":100,"Transitions":1000}
        }))
        .unwrap();
        input.incarnations[0].request.storage = Some(
            serde_json::from_value(json!({
                "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true}
            }))
            .unwrap(),
        );
        let mut prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        let mut sealed = prepared
            .incarnations
            .remove(&Handle {
                node: "a".into(),
                incarnation: 1,
            })
            .unwrap();
        let original_storage = serde_json::to_value(&sealed.execution.request.storage).unwrap();
        let image = sealed
            .execution
            .namespace
            .as_ref()
            .unwrap()
            .storage()
            .unwrap()
            .reference_image(1 << 20)
            .unwrap();
        let broker = Broker::new(&prepared.broker, &prepared.ids, None, None).unwrap();
        broker.defer_starts();
        if variant != "live-incarnation" {
            broker.finish(0);
        }
        let mut images = BTreeMap::new();
        if variant != "missing-storage-image" {
            images.insert("a".into(), image.clone());
        }
        let mut handle = Handle {
            node: "a".into(),
            incarnation: if variant == "stale-incarnation" { 3 } else { 2 },
        };
        let error = sealed
            .restart(&broker, 0, &handle, &images, None, false)
            .unwrap_err();
        assert_eq!(
            error.to_string(),
            if variant == "missing-storage-image" {
                "cluster restart lacks recovery image"
            } else {
                "WASI invalid: broker restart lifecycle"
            }
        );
        assert_eq!(broker.incarnation_slot(0), (1, 1));
        assert_eq!(broker.finished(0), variant != "live-incarnation");
        assert_eq!(broker.incarnation_slot(1), (1, 2));
        assert_eq!(
            serde_json::to_value(&sealed.execution.request.storage).unwrap(),
            original_storage
        );
        let refused_storage = serde_json::to_value(&sealed.execution.request.storage).unwrap();
        assert!(sealed.recovery_source.is_none());
        assert!(broker.failure().is_none());
        assert_eq!(activity.snapshot().launched_stores, 0);
        if variant == "live-incarnation" {
            broker.finish(0);
        }
        handle.incarnation = 2;
        images.insert("a".into(), image);
        let slot = sealed
            .restart(&broker, 0, &handle, &images, None, false)
            .unwrap();
        assert_eq!(slot, 3);
        assert_eq!(broker.incarnation_slot(0), (2, 3));
        assert!(!broker.finished(0));
        assert_eq!(broker.incarnation_slot(1), (1, 2));
        assert!(broker.failure().is_none());
        let recovered = sealed
            .execution
            .namespace
            .as_ref()
            .unwrap()
            .storage()
            .unwrap()
            .reference_image(1 << 20)
            .unwrap();
        assert_ne!(recovered.as_slice(), []);
        assert_eq!(
            serde_json::to_value(&sealed.execution.request.storage).unwrap()["image"],
            serde_json::to_value(&images["a"]).unwrap()
        );
        assert_eq!(activity.snapshot().launched_stores, 0);
        raw(
            "TestRejectedRestartLeavesNoUnownedBrokerSession",
            variant,
            json!({
                "refusal":error.to_string(),"before_incarnation":1,"before_slot":1,
                "after_refusal_incarnation":1,"after_refusal_slot":1,
                "refused_storage_unchanged":true,"retry_incarnation":2,"retry_slot":slot,
                "peer_incarnation":1,"peer_slot":2,"recovery_input_unchanged":true,
                "storage_before":original_storage,"storage_after_refusal":refused_storage,
                "retry_storage":serde_json::to_value(&sealed.execution.request.storage).unwrap(),
                "recovery_image":images["a"],"activity":activity.snapshot()
            }),
        );
        if variant == "missing-storage-image" {
            raw(
                "TestClusterRefusedRestartLeavesIncarnationRetryable",
                "recovery-image-refusal-and-retry",
                json!({
                    "refusal":error.to_string(),"refused_incarnation_owned":false,
                    "current_after_refusal":1,"retry_incarnation":2,"retry_slot":slot,
                    "recovery_input_unchanged":true,"storage_before":original_storage,
                    "storage_after_refusal":refused_storage,
                    "retry_storage":serde_json::to_value(&sealed.execution.request.storage).unwrap(),
                    "recovery_image":images["a"],"activity":activity.snapshot()
                }),
            );
        }
    }
}

#[test]
fn native_host_termination_preserves_watchdog_and_parent_cancellation_status() {
    for variant in [
        "watchdog",
        "cancelled",
        "parent deadline",
        "internal deadline",
    ] {
        let mut input = request(
            &["a"],
            "i32.const 512 i32.const 16 call $config drop (loop $forever br $forever)",
        );
        input.incarnations[0].request.limits.fuel = u64::MAX;
        let watchdog = matches!(variant, "watchdog" | "internal deadline");
        input.wall_timeout = if variant == "watchdog" {
            Duration::from_millis(20)
        } else {
            Duration::from_secs(1)
        };
        let mut prepared = prepare(input).unwrap();
        if variant == "internal deadline" {
            prepared
                .bound_wall_timeout(Duration::from_millis(20))
                .unwrap();
        }
        let activity = prepared.activity();
        let cancellation = prepared.cancellation.clone();
        let execution = thread::scope(|scope| {
            let (done, receive) = mpsc::sync_channel::<()>(0);
            if !watchdog {
                scope.spawn(move || {
                    if receive.recv_timeout(Duration::from_millis(20))
                        == Err(mpsc::RecvTimeoutError::Timeout)
                    {
                        cancellation.cancel();
                    }
                });
            }
            let execution = execute_prepared(prepared).unwrap();
            drop(done);
            execution
        });
        assert_eq!(
            execution.outcome.termination,
            if watchdog {
                "watchdog_timeout"
            } else {
                "cancelled"
            }
        );
        assert_eq!(execution.outcome.exit_code, None);
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        assert_eq!(
            activity.snapshot().launched_stores,
            execution.incarnations.len() as u64
        );
        assert!(
            execution
                .incarnations
                .iter()
                .all(|item| item.execution.authorization_to_terminal_nanos.is_some())
        );
        raw(
            if variant == "internal deadline" {
                "TestClusterInternalDeadlineIsWatchdog"
            } else {
                "TestClusterHostTerminationStatus"
            },
            variant,
            json!({
                "terminal":execution.outcome.termination,"exit_code":execution.outcome.exit_code,
                "watchdog_timeout":watchdog,"cancelled":!watchdog,
                "bound_millis":if watchdog { 20 } else { 1000 },
                "parent_cancel_after_millis":if watchdog { None } else { Some(20) },
                "joined_incarnations":execution.incarnations.len(),
                "all_stores_dropped_before_return":execution.incarnations.iter().all(|item|
                    item.execution.authorization_to_terminal_nanos.is_some()),
                "store_results":store_facts(&execution),"activity":activity.snapshot()
            }),
        );
    }
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The real five-second regression keeps store setup, timing, joined ownership and raw observations together"
)]
fn live_cleanup_reaps_a_noncooperative_store_at_the_original_five_second_grace() {
    assert_eq!(CLUSTER_STOP_GRACE, Duration::from_secs(5));
    let mut input = request(
        &["a"],
        "i32.const 512 i32.const 16 call $config drop (loop $busy br $busy)",
    );
    input.steps.retain(|step| step.action == Action::Start);
    input.wall_timeout = Duration::from_mins(15);
    input.incarnations[0].request.limits.fuel = u64::MAX;
    let mut prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    let conditions = prepared.conditions.enter().unwrap();
    let session = conditions.session();
    let engine = Engine::new(&engine::configuration(true)).unwrap();
    let modules = compile(&engine, &prepared, &mut Metrics::default()).unwrap();
    let broker = Arc::new(Broker::new(&prepared.broker, &prepared.ids, None, None).unwrap());
    broker.defer_starts();
    broker.activate(0).unwrap();
    broker.run(0, 1).unwrap();
    let handle = Handle {
        node: "a".into(),
        incarnation: 1,
    };
    let sealed = prepared.incarnations.remove(&handle).unwrap();
    let pre = modules[&sealed.execution.request.module_sha256].clone();
    let worker_engine = engine.clone();
    let worker_broker = broker.clone();
    let worker_activity = activity.clone();
    let worker_handle = handle.clone();
    let worker_conditions = prepared.conditions;
    let thread = thread::spawn(move || {
        let _conditions = worker_conditions.enter_shared(&session)?;
        let result = run_instance(
            &worker_engine,
            &pre,
            sealed,
            &worker_broker,
            None,
            worker_handle,
            0,
            1,
            &worker_activity,
        );
        worker_broker.finish(0);
        result
    });
    let mut running = BTreeMap::from([(handle.clone(), Running { thread })]);
    let mut completed = BTreeMap::new();
    let started = std::time::Instant::now();
    loop {
        let observed = broker
            .inspection
            .lock()
            .unwrap()
            .evidence
            .incarnations
            .iter()
            .any(|item| item.handle == handle && item.streams.stdout == b"mark");
        if observed {
            break;
        }
        assert!(
            started.elapsed() < Duration::from_secs(5),
            "guest did not produce actual mark"
        );
        thread::yield_now();
    }
    assert_eq!(activity.snapshot().launched_stores, 1);
    assert!(!running[&handle].thread.is_finished());
    let cleanup_started = std::time::Instant::now();
    let error = stop_live(
        &engine,
        &prepared.ids,
        &broker,
        &mut running,
        &mut completed,
        &Error::msg("actual scenario failure"),
    )
    .unwrap_err();
    let elapsed = cleanup_started.elapsed();
    assert!(elapsed >= CLUSTER_STOP_GRACE);
    assert!(elapsed < Duration::from_secs(10));
    assert_eq!(
        error.to_string(),
        "actual scenario failure; cluster cleanup deadline exceeded"
    );
    assert!(running.is_empty());
    assert_eq!(completed.len(), 1);
    let result = &completed[&handle];
    assert_eq!(result.execution.evidence["stdout"], b"mark");
    assert!(result.execution.authorization_to_terminal_nanos.is_some());
    assert!(broker.finished(0));
    assert_eq!(activity.snapshot().remote_callback_attempts, 0);
    raw(
        "TestClusterShutdownBoundsFinishedClusterByExecutionDeadline",
        "live-noncooperative-actual-five-second-grace",
        json!({
            "execution_deadline_millis":15*60*1000,"cleanup_grace_millis":5000,
            "elapsed_nanos":elapsed.as_nanos(),"cleanup_error":error.to_string(),
            "acknowledged_stdout":result.execution.evidence["stdout"],
            "authored_steps":prepared.steps,"joined_incarnations":1,
            "all_stores_dropped_before_return":true,"store_results":[store_result_fact(result)],
            "activity":activity.snapshot()
        }),
    );
}

#[test]
fn network_owner_admission_refuses_original_per_node_and_initial_incarnation_cases() {
    let corpus: Value = serde_json::from_str(include_str!(
        "../../tests/testdata/network_cluster_cases/cross-node-ready-poll-and-stale-cancel-binary-fixture.json"
    )).unwrap();
    let network: NetworkConfig = serde_json::from_value(corpus["config"].clone()).unwrap();
    let make_request = || {
        let mut input = request(&["a", "b"], FINISH);
        input.network = Some(network.clone());
        for item in &mut input.incarnations {
            item.network = Some(network.model.clone());
        }
        input
    };
    let admitted = prepare(make_request()).unwrap();
    assert_eq!(admitted.activity().snapshot().launched_stores, 0);
    raw(
        "TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses",
        "valid",
        json!({
            "admitted":true,"profile":network.model.profile,"activity":admitted.activity().snapshot()
        }),
    );
    for variant in [
        "node without profile",
        "node local faults",
        "stock node",
        "no cluster network",
        "restarted incarnation",
    ] {
        let mut input = make_request();
        let expected = match variant {
            "node without profile" => {
                input.incarnations[1].network = None;
                "cluster boot network profile"
            }
            "node local faults" => {
                input.incarnations[1].network.as_mut().unwrap().faults.push(
                    serde_json::from_value(json!({"name":"x","action":"heal","at_nanos":1}))
                        .unwrap(),
                );
                "broker node local fault schedule"
            }
            "stock node" => {
                input.incarnations[0].request.model = None;
                "cluster requires helper-owned model"
            }
            "no cluster network" => {
                input.network = None;
                "cluster boot network profile"
            }
            "restarted incarnation" => {
                input.incarnations[1].handle.incarnation = 2;
                for step in &mut input.steps {
                    if step.handle.node == "b" {
                        step.handle.incarnation = 2;
                    }
                }
                "invalid cluster incarnation lifecycle"
            }
            _ => unreachable!(),
        };
        let error = prepare(input).err().unwrap();
        assert_eq!(error.to_string(), expected);
        raw(
            "TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses",
            variant,
            json!({
                "admitted":false,"admission_error":error.to_string(),"launched_stores":0
            }),
        );
    }
    let error = serde_json::from_value::<NetworkConfig>(json!({
        "faults":[{"name":"f","action":"partition","from":"a","to":"b","at_nanos":1}]
    }))
    .unwrap_err();
    assert_eq!(error.to_string(), "missing field `model`");
    raw(
        "TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses",
        "faults without network",
        json!({
            "admitted":false,"admission_error":error.to_string(),"launched_stores":0,
            "owner":"network-config-decoder"
        }),
    );
}

#[test]
fn storage_volume_order_boot_admission_keeps_the_same_first_refusal() {
    for kind in ["selection", "context"] {
        for iteration in 0..64 {
            let model: crate::volume::StorageModel = serde_json::from_value(json!({
                "volumes":[{"ID":"a","Path":"/a","CapacityBytes":512},
                    {"ID":"b","Path":"/b","CapacityBytes":512}],
                "limits":{"PendingOperations":1000,"Transitions":1000}
            }))
            .unwrap();
            let mut descriptor = json!({"model":model,"model_identity":model.identity(),
                "return_image":true,"crash_limits":{"States":16,"Operations":8,"Depth":8,
                    "Bytes":1 << 20,"Work":256}});
            let ids = if iteration % 2 == 0 {
                ["a", "b"]
            } else {
                ["b", "a"]
            };
            let mut context = serde_json::Map::new();
            let mut selection = serde_json::Map::new();
            for id in ids {
                if kind == "selection" {
                    selection.insert(
                        id.into(),
                        if id == "a" { json!([0]) } else { json!([1, 1]) },
                    );
                } else {
                    context.insert(id.into(), json!({
                        "PendingSHA256":if id == "a" { "invalid".into() } else { format!("sha256:{}", "a".repeat(64)) },
                        "Operations":[{"ID":0,"Kind":"write","Dependencies":null,"Inode":0,
                            "Offset":0,"Bytes":0,"PayloadSHA256":"","EffectSHA256":"","Path":"","Destination":""}]
                    }));
                }
            }
            let context_bytes = if kind == "selection" {
                descriptor["crash_selection"] = Value::Object(selection);
                None
            } else {
                let bytes = serde_json::to_vec(&context).unwrap();
                descriptor["crash_context"] = json!({"bytes":bytes.len(),
                    "chunks":(bytes.len() as u64).div_ceil(64*1024),
                    "sha256":format!("{:x}",Sha256::digest(&bytes))});
                Some(bytes)
            };
            let mut input = request(&["a"], FINISH);
            input.incarnations[0].request.storage = Some(
                serde_json::from_value(json!({
                    "descriptor":descriptor,"context":context_bytes
                }))
                .unwrap(),
            );
            let error = prepare(input).err().unwrap();
            assert_eq!(
                error.to_string(),
                if kind == "selection" {
                    "WASI invalid: storage crash selection zero operation"
                } else {
                    "WASI invalid: storage crash context inventory"
                }
            );
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({
                    "schema":"gomad.core-predicate-observation/v1","case":"storage-validation-volume-boot-order",
                    "inputs":{"kind":kind,"iteration":iteration,"insertion_ids":ids,
                        "descriptor":descriptor,"context":context},"observed":{"refusal":error.to_string()}
                })
            );
        }
    }
}

fn storage_engine_request(descriptor: &Value, context: Option<&Value>) -> linked::Request {
    let mut input = request(&["a"], FINISH).incarnations.remove(0).request;
    let module = br#"(module
        (import "wasi_snapshot_preview1" "path_create_directory" (func $mkdir (param i32 i32 i32) (result i32)))
        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
        (memory (export "memory") 1)
        (data (i32.const 0) "db")
        (func (export "_start")
            i32.const 512 i32.const 16 call $config drop
            i32.const 4 i32.const 0 i32.const 2 call $mkdir if unreachable end
            i32.const 995 i32.const 1 i32.store8
            i32.const 992 i32.const 16 call $finish drop))"#.to_vec();
    input.module_sha256 = format!("{:x}", Sha256::digest(&module));
    input.module = module;
    input.limits.evidence_bytes = 8 << 20;
    let namespace = input.namespace.as_mut().unwrap();
    namespace.preopens = vec![b"/".to_vec(), b"/workspace".to_vec()];
    namespace.limits.filesystem_bytes = 1 << 20;
    let mut descriptor = descriptor.clone();
    let context = context.map(|context| {
        let bytes = serde_json::to_vec(context).unwrap();
        descriptor["crash_context"] = json!({"bytes":bytes.len(),
            "chunks":(bytes.len() as u64).div_ceil(64*1024),
            "sha256":format!("{:x}", Sha256::digest(&bytes))});
        bytes
    });
    input.storage =
        Some(serde_json::from_value(json!({"descriptor":descriptor,"context":context})).unwrap());
    input
}

fn native_request_input_facts(input: &linked::Request) -> Value {
    let model = input.model.as_ref().unwrap();
    let configuration = &model.configuration;
    assert!(configuration.tape.is_none());
    assert!(configuration.replay.is_none());
    assert!(configuration.inspection.is_none());
    let configuration = json!({
        "entropy_key":configuration.entropy_key,
        "working_directory_sha256":configuration.working_directory_sha256,
        "arguments":{"bytes":0,"count":0,"chunks":0},
        "environment":{"bytes":0,"count":0,"chunks":0},
        "seed":configuration.seed,"mode":configuration.mode,"limit":configuration.limit,
        "diagnostics":configuration.diagnostics,"replay":null,
        "epoch_nanos":configuration.epoch_nanos,"read_step_nanos":configuration.read_step_nanos,
        "now":configuration.now,"limits":{"calls":configuration.limits.calls,
            "transcript_bytes":configuration.limits.transcript_bytes,
            "pending_events":configuration.limits.pending_events},
        "digest_only":configuration.digest_only,"split":configuration.split,"inspection":null
    });
    json!({
        "module_bytes":input.module,"wat_utf8":std::str::from_utf8(&input.module).unwrap(),
        "request":{"limits":{"memory_bytes":input.limits.memory_bytes,"fuel":input.limits.fuel,
            "output_bytes":input.limits.output_bytes,"evidence_bytes":input.limits.evidence_bytes,
            "wall_timeout_millis":input.limits.wall_timeout.as_millis()},
            "model_configuration":configuration,"arguments":model.arguments,
            "environment":model.environment,"replay_trace":model.replay_trace,
            "namespace":input.namespace,"storage":input.storage}
    })
}

fn storage_state_sections(bytes: &[u8]) -> BTreeMap<u32, Vec<u8>> {
    assert_eq!(&bytes[..8], crate::storage_transport::STATE_MAGIC);
    let count = u32::from_le_bytes(bytes[12..16].try_into().unwrap());
    let mut cursor = 16;
    let mut sections = BTreeMap::new();
    for _ in 0..count {
        let tag = u32::from_le_bytes(bytes[cursor..cursor + 4].try_into().unwrap());
        let length = usize::try_from(u64::from_le_bytes(
            bytes[cursor + 4..cursor + 12].try_into().unwrap(),
        ))
        .unwrap();
        cursor += 12;
        assert!(
            sections
                .insert(tag, bytes[cursor..cursor + length].to_vec())
                .is_none()
        );
        cursor += length;
    }
    assert_eq!(cursor + 40, bytes.len());
    assert_eq!(
        u64::from_le_bytes(bytes[cursor..cursor + 8].try_into().unwrap()),
        cursor as u64
    );
    assert_eq!(
        Sha256::digest(&bytes[..cursor]).as_slice(),
        &bytes[cursor + 8..]
    );
    sections
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The bounded original baseline, selected-state and divergence matrix shares actual native enumerations"
)]
fn actual_storage_engine_selection_and_divergence_retain_peak_memory_and_enumeration() {
    let model: crate::volume::StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1 << 20}],
        "limits":{"PendingOperations":1000,"Transitions":10000}
    }))
    .unwrap();
    let baseline = json!({"model":model,"model_identity":model.identity(),"return_image":true,
        "crash_limits":{"States":16,"Operations":8,"Depth":8,"Bytes":1 << 20,"Work":256}});
    let mut observations = Vec::new();
    for fresh_execution in 0..2 {
        let input = storage_engine_request(&baseline, None);
        let module_sha256 = input.module_sha256.clone();
        let input_facts = native_request_input_facts(&input);
        let result = linked::execute_prepared(linked::prepare(input).unwrap()).unwrap();
        assert_eq!(result.outcome.termination, "exit");
        assert_eq!(result.outcome.exit_code, Some(0));
        assert!(result.outcome.peak_memory_bytes > 0);
        let sections = storage_state_sections(&result.evidence["storage_state"]);
        let context: Value = serde_json::from_slice(&sections[&6]).unwrap();
        assert_ne!(
            context["data"]["Operations"].as_array().unwrap().as_slice(),
            []
        );
        let enumerations: Value = serde_json::from_slice(&sections[&3]).unwrap();
        assert_eq!(enumerations["data"]["Complete"], true);
        assert!(enumerations["data"]["States"].as_array().unwrap().len() >= 2);
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({
                "schema":"gomad.core-predicate-observation/v1","case":"storage-engine-baseline",
                "inputs":{"descriptor":baseline,"fresh_execution":fresh_execution,"module_sha256":module_sha256,"module_bytes":input_facts["module_bytes"],
                "wat_utf8":input_facts["wat_utf8"],"request":input_facts["request"]},
                "observed":{"termination":result.outcome.termination,"message":result.outcome.message,
                    "exit_code":result.outcome.exit_code,"peak_memory_bytes":result.outcome.peak_memory_bytes,
                    "state_bytes":result.evidence["storage_state"],"log_bytes":result.evidence["storage"],
                    "image_bytes":result.evidence.get("storage_image"),"context":context,"enumerations":enumerations}
            })
        );
        observations.push((context, enumerations, result));
    }
    for stream in ["storage_state", "storage", "storage_image"] {
        assert_eq!(
            observations[0].2.evidence[stream],
            observations[1].2.evidence[stream]
        );
    }
    let states = observations[0].1["data"]["States"].as_array().unwrap();
    for (state_index, state) in states.iter().enumerate() {
        let mut previous = None;
        for (fresh_execution, (context, _, _)) in observations.iter().enumerate() {
            let mut descriptor = baseline.clone();
            descriptor["crash_selection"] = json!({"data":state["SelectedOperations"]});
            let input = storage_engine_request(&descriptor, Some(context));
            let module_sha256 = input.module_sha256.clone();
            let input_facts = native_request_input_facts(&input);
            let result = linked::execute_prepared(linked::prepare(input).unwrap()).unwrap();
            assert_eq!(result.outcome.termination, "exit");
            assert_eq!(result.outcome.exit_code, Some(0));
            assert!(result.outcome.peak_memory_bytes > 0);
            let evidence = (
                result.evidence["storage_state"].clone(),
                result.evidence["storage"].clone(),
                result.evidence["storage_image"].clone(),
            );
            if let Some(previous) = &previous {
                assert_eq!(&evidence, previous);
            }
            previous = Some(evidence);
            let image = &result.evidence["storage_image"];
            let recovered = crate::volume::Storage::recover(
                model.clone(),
                8 << 20,
                image,
                &format!("{:x}", Sha256::digest(image)),
                2 << 20,
            )
            .unwrap();
            let present = recovered.fs.lookup("/workspace/db").is_ok();
            assert_eq!(
                present,
                state["Entries"]
                    .as_array()
                    .unwrap()
                    .iter()
                    .any(|entry| entry["Path"] == "/db")
            );
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({
                    "schema":"gomad.core-predicate-observation/v1","case":"storage-engine-selected",
                    "inputs":{"descriptor":descriptor,"context":context,"state":state,"state_index":state_index,
                        "fresh_execution":fresh_execution,"module_sha256":module_sha256,"module_bytes":input_facts["module_bytes"],
                "wat_utf8":input_facts["wat_utf8"],"request":input_facts["request"]},
                    "observed":{"termination":result.outcome.termination,"message":result.outcome.message,
                        "exit_code":result.outcome.exit_code,"peak_memory_bytes":result.outcome.peak_memory_bytes,
                        "state_bytes":result.evidence["storage_state"],"log_bytes":result.evidence["storage"],
                        "image_bytes":image,"db_present":present}
                })
            );
        }
    }
    for (fresh_execution, (context, enumerations, _)) in observations.iter().enumerate() {
        let mut context = context.clone();
        context["data"]["PendingSHA256"] = json!(format!("sha256:{}", "0".repeat(64)));
        let mut descriptor = baseline.clone();
        descriptor["crash_selection"] =
            json!({"data":states.last().unwrap()["SelectedOperations"]});
        let input = storage_engine_request(&descriptor, Some(&context));
        let module_sha256 = input.module_sha256.clone();
        let input_facts = native_request_input_facts(&input);
        let result = linked::execute_prepared(linked::prepare(input).unwrap()).unwrap();
        assert_eq!(result.outcome.termination, "invalid");
        assert_eq!(
            result.outcome.message,
            "storage crash context diverged from prior observation"
        );
        assert_eq!(result.outcome.exit_code, None);
        assert!(result.outcome.peak_memory_bytes > 0);
        assert!(!result.evidence.contains_key("storage_image"));
        let sections = storage_state_sections(&result.evidence["storage_state"]);
        let retained: Value = serde_json::from_slice(&sections[&3]).unwrap();
        assert_eq!(&retained, enumerations);
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({
                "schema":"gomad.core-predicate-observation/v1","case":"storage-engine-divergence",
                "inputs":{"descriptor":descriptor,"context":context,"fresh_execution":fresh_execution,
                    "module_sha256":module_sha256,"module_bytes":input_facts["module_bytes"],
                "wat_utf8":input_facts["wat_utf8"],"request":input_facts["request"]},
                "observed":{"termination":result.outcome.termination,"message":result.outcome.message,
                    "exit_code":result.outcome.exit_code,"peak_memory_bytes":result.outcome.peak_memory_bytes,
                    "state_bytes":result.evidence["storage_state"],"log_bytes":result.evidence["storage"],
                    "image_bytes":result.evidence.get("storage_image"),"enumerations":retained,
                    "original_enumerations":enumerations}
            })
        );
    }
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The original held-peer drain case keeps actual store ownership, the 50ms release gate and native control observations together"
)]
fn observed_failure_drain_waits_for_held_native_peer_without_post_failure_controls() {
    let mut input = request(&["a", "b"], FINISH);
    input.incarnations[0] = request(&["a"], "unreachable").incarnations.remove(0);
    let mut prepared = prepare(input).unwrap();
    let activity = prepared.activity();
    let conditions = prepared.conditions.enter().unwrap();
    let session = conditions.session();
    let engine = Engine::new(&engine::configuration(true)).unwrap();
    let modules = compile(&engine, &prepared, &mut Metrics::default()).unwrap();
    let broker = Arc::new(Broker::new(&prepared.broker, &prepared.ids, None, None).unwrap());
    broker.defer_starts();
    let peer_handle = Handle {
        node: "b".into(),
        incarnation: 1,
    };
    let peer = prepared.incarnations.remove(&peer_handle).unwrap();
    let peer_module = peer.execution.request.module.clone();
    let peer_module_sha256 = peer.execution.request.module_sha256.clone();
    let peer_pre = modules[&peer_module_sha256].clone();
    broker.activate(1).unwrap();
    broker.run(1, 1).unwrap();
    let (ready, peer_ready) = mpsc::sync_channel(0);
    let (release, peer_release) = mpsc::sync_channel(0);
    let peer_engine = engine.clone();
    let peer_broker = broker.clone();
    let peer_activity = activity.clone();
    let peer_conditions = prepared.conditions;
    let peer_thread = thread::spawn(move || -> Result<Completed> {
        let _conditions = peer_conditions.enter_shared(&session)?;
        if !peer_broker.acquire(1) {
            return Err(Error::msg("held peer was not admitted"));
        }
        let completed = run_instance(
            &peer_engine,
            &peer_pre,
            peer,
            &peer_broker,
            None,
            peer_handle,
            1,
            2,
            &peer_activity,
        )?;
        peer_broker.finish(1);
        ready
            .send(store_result_fact(&completed.result))
            .map_err(|_| Error::msg("held peer observation abandoned"))?;
        peer_release
            .recv()
            .map_err(|_| Error::msg("held peer release abandoned"))?;
        Ok(completed)
    });
    let held_peer = peer_ready.recv_timeout(Duration::from_secs(5)).unwrap();
    assert_eq!(held_peer["terminal"], "exit");
    assert_eq!(held_peer["exit_code"], 0);
    assert_eq!(held_peer["stdout"], json!(b"mark"));
    assert!(
        held_peer["authorization_to_terminal_nanos"]
            .as_u64()
            .unwrap()
            > 0
    );
    assert!(!peer_thread.is_finished());

    let failed_handle = Handle {
        node: "a".into(),
        incarnation: 1,
    };
    let failed = prepared.incarnations.remove(&failed_handle).unwrap();
    let failed_module = failed.execution.request.module.clone();
    let failed_module_sha256 = failed.execution.request.module_sha256.clone();
    broker.activate(0).unwrap();
    broker.run(0, 1).unwrap();
    assert!(broker.acquire(0));
    let failed = run_instance(
        &engine,
        &modules[&failed_module_sha256],
        failed,
        &broker,
        None,
        failed_handle.clone(),
        0,
        1,
        &activity,
    )
    .unwrap();
    assert_eq!(failed.result.execution.outcome.termination, "trap");
    let failed_message = failed.result.execution.outcome.message.clone();
    broker.abort_terminal(0, "trap", failed_message.clone());
    broker.finish(0);
    let before_controls = broker.native_control_attempts();
    assert_eq!(before_controls, 4);
    let before_lifecycle = broker.inspection.lock().unwrap().evidence.lifecycle.clone();
    let running = BTreeMap::from([(
        Handle {
            node: "b".into(),
            incarnation: 1,
        },
        Running {
            thread: peer_thread,
        },
    )]);
    let completed = BTreeMap::from([(failed_handle, failed.result)]);
    let (entered, enter_drain) = mpsc::sync_channel(0);
    let (returned, receive) = mpsc::sync_channel(0);
    let drain_broker = broker.clone();
    let drainer = thread::spawn(move || {
        let mut completed = completed;
        entered.send(()).unwrap();
        let error = drain_running(running, &mut completed, &drain_broker, true);
        returned
            .send((error.map(|error| error.to_string()), completed))
            .unwrap();
    });
    enter_drain.recv_timeout(Duration::from_secs(5)).unwrap();
    let probe_started = std::time::Instant::now();
    assert!(matches!(
        receive.recv_timeout(Duration::from_millis(50)),
        Err(mpsc::RecvTimeoutError::Timeout)
    ));
    let probe_elapsed = probe_started.elapsed();
    let held_controls = broker.native_control_attempts();
    let held_lifecycle = broker.inspection.lock().unwrap().evidence.lifecycle.clone();
    assert_eq!(held_controls, before_controls);
    assert_eq!(held_lifecycle, before_lifecycle);
    let release_started = std::time::Instant::now();
    release.send(()).unwrap();
    let (join_error, completed) = receive.recv_timeout(Duration::from_secs(5)).unwrap();
    let release_to_return = release_started.elapsed();
    drainer.join().unwrap();
    assert!(join_error.is_none());
    assert_eq!(completed.len(), 2);
    let execution = completed
        .values()
        .map(store_result_fact)
        .collect::<Vec<_>>();
    assert_eq!(execution[1], held_peer);
    assert_eq!(execution[0]["terminal"], "trap");
    assert_eq!(execution[0]["message"], failed_message);
    let after_controls = broker.native_control_attempts();
    let after_lifecycle = broker.inspection.lock().unwrap().evidence.lifecycle.clone();
    assert_eq!(after_controls, before_controls);
    assert_eq!(after_lifecycle, before_lifecycle);
    assert_eq!(activity.snapshot().launched_stores, 2);
    assert_eq!(activity.snapshot().remote_callback_attempts, 0);
    raw(
        "TestClusterShutdownWaitsForObservedFailureDrain",
        "held-peer-release",
        json!({
            "terminal":"trap","exit_code":null,"message":failed_message,
            "failed_module_bytes":failed_module,"failed_module_sha256":failed_module_sha256,
            "peer_module_bytes":peer_module,"peer_module_sha256":peer_module_sha256,
            "held_peer_result":held_peer,"store_results":execution,"activity":activity.snapshot(),
            "probe_millis":50,"probe_outcome":"timeout","probe_elapsed_nanos":probe_elapsed.as_nanos(),
            "release_to_return_nanos":release_to_return.as_nanos(),"join_error":join_error,
            "native_control_entry_points":["activate","run","command"],
            "before_control_attempts":before_controls,"held_control_attempts":held_controls,
            "after_control_attempts":after_controls,"before_lifecycle_bytes":before_lifecycle,
            "held_lifecycle_bytes":held_lifecycle,"after_lifecycle_bytes":after_lifecycle
        }),
    );
}

#[test]
fn memory_capacity_keeps_owned_model_metadata_before_and_after_runtime_bootstrap() {
    for (variant, body, configured) in [
        (
            "memory-capacity-before-bootstrap",
            "i32.const 1 memory.grow drop",
            false,
        ),
        (
            "memory-capacity-after-bootstrap",
            "i32.const 512 i32.const 16 call $config drop i32.const 1 memory.grow drop",
            true,
        ),
    ] {
        let input = request(&["a"], body);
        let request_facts = native_request_input_facts(&input.incarnations[0].request);
        let module = input.incarnations[0].request.module.clone();
        let module_sha256 = input.incarnations[0].request.module_sha256.clone();
        let offered_fuel = input.incarnations[0].request.limits.fuel;
        let memory_bytes = input.incarnations[0].request.limits.memory_bytes;
        let prepared = prepare(input).unwrap();
        let activity = prepared.activity();
        let execution = execute_prepared(prepared).unwrap();
        assert_eq!(execution.executed_steps.len(), 2);
        assert_eq!(execution.incarnations.len(), 1);
        assert_eq!(execution.outcome.termination, "capacity");
        assert_eq!(execution.outcome.exit_code, None);
        let result = &execution.incarnations[0].execution;
        assert_eq!(result.outcome.termination, "capacity");
        assert_eq!(result.outcome.exit_code, None);
        assert!(
            result
                .outcome
                .message
                .contains("guest linear memory capacity")
        );
        assert_eq!(result.outcome.peak_memory_bytes, 65536);
        assert_eq!(result.outcome.output_bytes, 4);
        let model = result.outcome.local_model.as_ref().unwrap();
        assert_eq!(model["configured"], configured);
        assert_eq!(model["finished"], false);
        let choice = result.evidence.get("choice").cloned().unwrap_or_default();
        let transcript = &result.evidence["transcript"];
        assert_eq!(
            model["trace_sha256"],
            format!("{:x}", Sha256::digest(&choice))
        );
        assert_eq!(model["transcript_bytes"], transcript.len());
        assert_eq!(
            model["transcript_sha256"],
            format!("{:x}", Sha256::digest(transcript))
        );
        assert!(result.authorization_to_terminal_nanos.is_some());
        assert_eq!(activity.snapshot().launched_stores, 1);
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        raw(
            "TestClusterRejectsResultWithoutModelEvidence",
            variant,
            json!({
                "module_bytes":module,"module_sha256":module_sha256,"request":request_facts["request"],
                "terminal":result.outcome.termination,"exit_code":result.outcome.exit_code,
                "message":result.outcome.message,"fuel_remaining":result.outcome.fuel_remaining,
                "offered_fuel":offered_fuel,"memory_bytes":memory_bytes,
                "peak_memory_bytes":result.outcome.peak_memory_bytes,"output_bytes":result.outcome.output_bytes,
                "local_model":model,"choice_bytes":choice,"transcript_bytes":transcript,
                "executed_steps":execution.executed_steps,"store_results":store_facts(&execution),
                "activity":activity.snapshot()
            }),
        );
    }
}
