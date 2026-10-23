use super::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{fs, path::Path};

fn output_wait_request(fd: u32, after: &str, nodes: &[&str], action: Action) -> Request {
    let module = format!(r#"(module
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "idle" (func $idle (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))
        (memory (export "memory") 1)
        (data (i32.const 128) "mark")
        (func (export "_start")
            i32.const 0 i32.const 128 i32.store
            i32.const 4 i32.const 4 i32.store
            i32.const {fd} i32.const 0 i32.const 1 i32.const 16 call $write drop
            {after}))"#).into_bytes();
    let incarnations = nodes.iter().map(|node| Incarnation {
        handle: Handle { node: (*node).into(), incarnation: 1 },
        network: None,
        request: linked::Request {
            module_sha256: format!("{:x}", Sha256::digest(&module)), module: module.clone(),
            limits: linked::Limits { memory_bytes: 65536, fuel: 1_000_000,
                output_bytes: 1024, evidence_bytes: 1 << 20, wall_timeout: Duration::from_secs(5) },
            model: Some(linked::ModelInput {
                configuration: serde_json::from_value(json!({
                    "entropy_key":vec![0u8;32], "working_directory_sha256":format!("{:x}", Sha256::digest(b"/workspace")),
                    "arguments":{"bytes":0,"count":0,"chunks":0},
                    "environment":{"bytes":0,"count":0,"chunks":0},
                    "seed":7,"mode":1,"limit":1_048_576,"diagnostics":false,
                    "epoch_nanos":946_684_800_000_000_000_u64,"read_step_nanos":0,"now":1,
                    "limits":{"calls":1_048_576,"transcript_bytes":1_048_576,"pending_events":1024}
                })).unwrap(),
                arguments: vec![], environment: vec!["PWD=/workspace".into()], replay_trace: None,
            }),
            namespace: Some(namespace::Image {
                entries: vec![namespace::ImageEntry {path:b"/workspace".to_vec(), directory:true, readonly:false, data:vec![]}],
                writable_directories: vec![b"/workspace".to_vec()], preopens: vec![b"/workspace".to_vec()],
                working_directory: b"/workspace".to_vec(), stdin: vec![], epoch_nanos: 946_684_800_000_000_000,
                limits: namespace::Limits { files: 100, descriptors: 16, filesystem_bytes: 4096, output_bytes: 1024 },
            }),
            storage: None,
        },
    }).collect::<Vec<_>>();
    let handles = incarnations
        .iter()
        .map(|item| item.handle.clone())
        .collect::<Vec<_>>();
    Request {
        nodes: nodes.iter().map(|node| (*node).into()).collect(),
        broker: BrokerConfig {
            seed: 7,
            pause_on_settle: false,
            transition_bytes: None,
            limits: WorldLimits {
                max_requests: 65536,
                max_events: 65536,
                max_queued_events: 1024,
                max_transitions: 1 << 20,
                max_payload_bytes: 1 << 20,
                max_string_bytes: 1024,
            },
        },
        network: None,
        incarnations,
        steps: [Action::Start, Action::Wait]
            .into_iter()
            .flat_map(|action| {
                handles
                    .iter()
                    .cloned()
                    .map(move |handle| Step { action, handle })
            })
            .collect(),
        boundaries: handles
            .into_iter()
            .map(|handle| Boundary {
                handle,
                action,
                marker: b"mark".to_vec(),
            })
            .collect(),
        crash_groups: vec![],
        wall_timeout: Duration::from_secs(5),
        evidence_bytes: 4 << 20,
    }
}

#[test]
fn conditional_output_wait_retains_world_and_actual_terminal() {
    for (action, terminal) in [(Action::Crash, "crash"), (Action::Stop, "stop")] {
        for fd in [1, 2] {
            let request = output_wait_request(fd, "", &["a"], action);
            let world_config = crate::world::Config {
                seed: request.broker.seed,
                limits: crate::world::Limits {
                    max_requests: request.broker.limits.max_requests,
                    max_events: request.broker.limits.max_events,
                    max_queued_events: request.broker.limits.max_queued_events,
                    max_transitions: request.broker.limits.max_transitions,
                    max_payload_bytes: request.broker.limits.max_payload_bytes,
                    max_string_bytes: request.broker.limits.max_string_bytes,
                },
            };
            let execution = execute_prepared(prepare(request).unwrap()).unwrap();
            assert_eq!(execution.incarnations.len(), 1);
            let item = &execution.incarnations[0];
            assert_eq!(
                item.execution.outcome.termination,
                if fd == 1 { terminal } else { "exit" }
            );
            assert_eq!(item.interrupted, fd == 1);
            assert_eq!(execution.executed_steps[1].action, Action::Wait);
            assert_eq!(execution.outcome.termination, "exit", "{action:?} fd={fd}");
            assert_eq!(execution.outcome.exit_code, Some(0));
            assert!(!execution.world.is_empty(), "{action:?} fd={fd}");
            crate::world::retained::validate(&execution.world, &world_config).unwrap();
        }
    }
    let loop_after = "i32.const 512 i32.const 16 call $config drop
        (loop
            i32.const 768 i64.const 0 i64.store
            i32.const 776 i64.const 0 i64.store
            i32.const 771 i32.const 1 i32.store8
            i32.const 768 i32.const 16 call $idle drop br 0)";
    let mut request = output_wait_request(1, loop_after, &["a", "b"], Action::Crash);
    request.boundaries.clear();
    request.crash_groups.push(CrashGroup {
        nodes: vec!["a".into(), "b".into()],
        marker: b"mark".to_vec(),
    });
    let execution = execute_prepared(prepare(request).unwrap()).unwrap();
    assert_eq!(
        execution.outcome.termination, "exit",
        "{}",
        execution.outcome.message
    );
    assert_eq!(execution.outcome.exit_code, Some(0));
    assert_ne!(execution.world, [0_u8; 0]);
    assert_eq!(execution.incarnations.len(), 2);
    assert!(
        execution
            .incarnations
            .iter()
            .all(|item| item.interrupted && item.execution.outcome.termination == "crash")
    );
    for (after, terminal, code) in [
        ("unreachable", "trap", None),
        ("i32.const 17 call $exit", "exit", Some(17)),
    ] {
        let execution = execute_prepared(
            prepare(output_wait_request(2, after, &["a"], Action::Crash)).unwrap(),
        )
        .unwrap();
        assert_eq!(execution.outcome.termination, terminal);
        assert_eq!(execution.outcome.exit_code, code);
        assert_eq!(
            execution.incarnations[0].execution.outcome.termination,
            terminal
        );
    }
}

#[test]
fn unbound_same_kind_and_cancellation_remain_failures_with_armed_marker() {
    let loop_after = "(loop i32.const 1 i64.const 1 i32.const 64 call $clock drop br 0)";
    for terminal in ["crash", "stop"] {
        let action = if terminal == "stop" {
            Action::Stop
        } else {
            Action::Crash
        };
        let mut request = output_wait_request(1, loop_after, &["a"], action);
        request.boundaries[0].marker = b"never".to_vec();
        let prepared = prepare(request).unwrap();
        let activity = prepared.activity();
        let conditions = prepared.conditions.enter().unwrap();
        let session = conditions.session();
        let engine = Engine::new(&engine::configuration(true)).unwrap();
        let broker = Arc::new(Broker::new(&prepared.broker, &prepared.ids, None, None).unwrap());
        let cancellation = prepared.cancellation.clone();
        let _registration = cancellation.attach(&engine, Some(broker.clone())).unwrap();
        let execution = thread::scope(|scope| {
            let controller_broker = broker.clone();
            scope.spawn(move || {
                let started = std::time::Instant::now();
                while activity.snapshot().launched_stores == 0 {
                    assert!(started.elapsed() < Duration::from_secs(5));
                    thread::sleep(Duration::from_millis(1));
                }
                controller_broker
                    .command(Command::Revoke {
                        index: 0,
                        incarnation: 1,
                        kind: terminal,
                    })
                    .unwrap();
            });
            execute_inner(
                &engine,
                prepared,
                &broker,
                None,
                &AtomicBool::new(false),
                &session,
                None,
            )
        })
        .unwrap();
        assert_eq!(execution.outcome.termination, terminal);
        assert!(
            execution.world.is_empty(),
            "unexpected {terminal} must not retain World"
        );
    }
    let mut request = output_wait_request(1, loop_after, &["a"], Action::Crash);
    request.boundaries[0].marker = b"never".to_vec();
    let prepared = prepare(request).unwrap();
    let activity = prepared.activity();
    let cancellation = prepared.cancellation.clone();
    let execution = thread::scope(|scope| {
        scope.spawn(move || {
            let started = std::time::Instant::now();
            while activity.snapshot().launched_stores == 0 {
                assert!(started.elapsed() < Duration::from_secs(5));
                thread::sleep(Duration::from_millis(1));
            }
            cancellation.cancel();
        });
        execute_prepared(prepared)
    })
    .unwrap();
    assert_eq!(execution.outcome.termination, "cancelled");
    assert_eq!(execution.outcome.exit_code, None);
    assert_eq!(execution.outcome.message, "seed execution was cancelled");
}

#[test]
fn conditional_output_boundary_preserves_wait_lifecycle() {
    let handle = Handle {
        node: "a".into(),
        incarnation: 1,
    };
    let handles = BTreeSet::from([handle.clone()]);
    let mut request = Request {
        nodes: vec!["a".into()],
        broker: BrokerConfig {
            seed: 7,
            limits: WorldLimits {
                max_requests: 100,
                max_events: 100,
                max_queued_events: 100,
                max_transitions: 1000,
                max_payload_bytes: 4096,
                max_string_bytes: 1024,
            },
            pause_on_settle: false,
            transition_bytes: None,
        },
        network: None,
        incarnations: vec![],
        steps: vec![
            Step {
                action: Action::Start,
                handle: handle.clone(),
            },
            Step {
                action: Action::Wait,
                handle: handle.clone(),
            },
        ],
        boundaries: vec![Boundary {
            handle,
            action: Action::Crash,
            marker: b"committed durable-a".to_vec(),
        }],
        crash_groups: vec![],
        wall_timeout: Duration::from_secs(60),
        evidence_bytes: 64 << 20,
    };
    for boundary_action in [Action::Crash, Action::Stop] {
        request.boundaries[0].action = boundary_action;
        for terminal_action in [boundary_action, Action::Wait] {
            request.steps[1].action = terminal_action;
            assert_eq!(lifecycle(&request, &handles).unwrap(), ["a"]);
            assert_eq!(request.steps[1].action, terminal_action);
        }
        request.boundaries[0].marker.clear();
        assert!(lifecycle(&request, &handles).is_err());
        request.boundaries[0].marker = vec![1; 4097];
        assert!(lifecycle(&request, &handles).is_err());
        request.boundaries[0].marker = b"committed durable-a".to_vec();
        request.boundaries.push(request.boundaries[0].clone());
        assert!(lifecycle(&request, &handles).is_err());
        request.boundaries.pop();
        request.steps[1].action = if boundary_action == Action::Crash {
            Action::Stop
        } else {
            Action::Crash
        };
        assert!(lifecycle(&request, &handles).is_err());
    }
    request.steps[1].action = Action::Wait;
    request.boundaries[0].action = Action::Wait;
    assert!(lifecycle(&request, &handles).is_err());
    request.boundaries[0].action = Action::Crash;
    request.boundaries[0].handle.node = "unknown".into();
    assert!(lifecycle(&request, &handles).is_err());
}

fn number(value: &Value, field: &str) -> u64 {
    value[field]
        .as_u64()
        .unwrap_or_else(|| panic!("missing actual request {field}"))
}

fn original_request(input: &Value) -> Request {
    let config = &input["configuration"];
    let broker: BrokerConfig = serde_json::from_value(config["broker"].clone()).unwrap();
    let incarnations = config["incarnations"]
        .as_array()
        .unwrap()
        .iter()
        .map(|item| {
            let module_file = item["module"]["file"].as_str().unwrap();
            let path = input["modules"][module_file].as_str().unwrap();
            let module = fs::read(path).unwrap();
            assert!(
                module.starts_with(b"\0asm"),
                "root35 requires actual unchanged compiled service guest"
            );
            assert!(
                module.len() <= 8 << 20,
                "original service module admission bound"
            );
            let limits = &item["limits"];
            let mut model = item["model"].clone();
            model["seed"] = json!(broker.seed);
            Incarnation {
                handle: serde_json::from_value(item["handle"].clone()).unwrap(),
                network: serde_json::from_value(item["network"].clone()).unwrap(),
                request: linked::Request {
                    module,
                    module_sha256: item["module"]["sha256"].as_str().unwrap().into(),
                    limits: linked::Limits {
                        memory_bytes: number(limits, "memory_bytes"),
                        fuel: number(limits, "fuel"),
                        output_bytes: number(limits, "output_bytes"),
                        evidence_bytes: number(limits, "evidence_bytes"),
                        wall_timeout: Duration::from_millis(number(limits, "wall_millis")),
                    },
                    model: Some(linked::ModelInput {
                        configuration: serde_json::from_value(model).unwrap(),
                        arguments: serde_json::from_value(item["arguments"].clone()).unwrap(),
                        environment: item["environment"]
                            .as_object()
                            .unwrap()
                            .iter()
                            .map(|(name, value)| format!("{name}={}", value.as_str().unwrap()))
                            .collect(),
                        replay_trace: None,
                    }),
                    namespace: Some(serde_json::from_value(item["namespace"].clone()).unwrap()),
                    storage: serde_json::from_value(item["storage"].clone()).unwrap(),
                },
            }
        })
        .collect();
    Request {
        nodes: serde_json::from_value(config["nodes"].clone()).unwrap(),
        broker,
        network: serde_json::from_value(config["network"].clone()).unwrap(),
        incarnations,
        steps: serde_json::from_value(config["steps"].clone()).unwrap(),
        boundaries: serde_json::from_value(config["boundaries"].clone()).unwrap(),
        crash_groups: serde_json::from_value(config["crash_groups"].clone()).unwrap(),
        wall_timeout: Duration::from_millis(number(&config["limits"], "wall_millis")),
        evidence_bytes: number(&config["limits"], "evidence_bytes"),
    }
}

fn prepared(input: &Value) -> Prepared {
    let mut prepared = prepare(original_request(input)).unwrap();
    assert_eq!(prepared.steps.len(), 2);
    assert_eq!(prepared.steps[0].handle.node, "a");
    assert_eq!(prepared.steps[1].handle.node, "b");
    assert!(
        prepared
            .steps
            .iter()
            .all(|step| step.action == Action::Start)
    );
    prepared
        .cleanup_after_starts(CleanupMode::Shutdown)
        .unwrap();
    prepared
}

fn assert_original_incarnations_match(observed: &Execution, reference: &Value) {
    use base64::{Engine as _, engine::general_purpose::STANDARD};

    for item in &observed.incarnations {
        let original = reference["states"]
            .as_array()
            .unwrap()
            .iter()
            .find(|state| {
                state["handle"]["ID"] == item.handle.node
                    && state["handle"]["Incarnation"] == item.handle.incarnation
            })
            .unwrap();
        assert_eq!(
            item.execution.outcome.termination,
            original["result"]["Termination"].as_str().unwrap()
        );
        assert_eq!(
            item.interrupted,
            original["result"]["RuntimeInterrupted"].as_bool().unwrap()
        );
        assert_eq!(
            item.execution.evidence["network"],
            STANDARD
                .decode(original["network_wire"].as_str().unwrap())
                .unwrap()
        );
    }
}

#[test]
#[ignore = "requires authenticated original open-streams service input and Go final network wire"]
fn final_network_uses_finished_shared_owner_instead_of_cached_incarnation() {
    use base64::{Engine as _, engine::general_purpose::STANDARD};

    let input_path =
        std::env::var_os("GOMAD_LIVE_CLUSTER_FINISH_INPUT").expect("original root34 input path");
    let reference_path = std::env::var_os("GOMAD_LIVE_CLUSTER_FINISH_REFERENCE")
        .expect("original root34 Go observation path");
    let input: Value = serde_json::from_slice(&fs::read(input_path).unwrap()).unwrap();
    let reference: Value = serde_json::from_slice(&fs::read(reference_path).unwrap()).unwrap();
    assert_eq!(input["schema"], "gomad.live-cluster-observation/v1");
    assert_eq!(
        reference["test"],
        "TestClusterBrokerDurableRestartParity/open-streams/seed-7"
    );
    assert_eq!(reference["ordinal"], 73);
    assert_eq!(input["forcing"], json!({}));
    let request = original_request(&input);
    assert_eq!(request.nodes, ["a", "b"]);
    assert_eq!(request.incarnations.len(), 3);
    assert!(
        request
            .incarnations
            .iter()
            .all(|item| item.request.module_sha256
                == "58af6ef04c8ea078ea338cc5db1caf0ea4b727c5e6fabbe4cc47265f2f81b426")
    );
    assert_eq!(
        request
            .steps
            .iter()
            .map(|step| (
                step.action,
                step.handle.node.as_str(),
                step.handle.incarnation
            ))
            .collect::<Vec<_>>(),
        vec![
            (Action::Start, "a", 1),
            (Action::Start, "b", 1),
            (Action::Wait, "a", 1),
            (Action::Wait, "b", 1),
            (Action::Restart, "a", 2),
            (Action::Wait, "a", 2)
        ]
    );
    let expected = STANDARD
        .decode(reference["network_wire"].as_str().unwrap())
        .unwrap();
    assert_ne!(expected, Vec::<u8>::new());
    let observed = execute_prepared(prepare(request).unwrap()).unwrap();
    assert_eq!(observed.outcome.termination, "exit");
    assert_eq!(observed.outcome.exit_code, Some(0));
    assert_eq!(
        observed.world,
        crate::world::retained::expand(
            &STANDARD
                .decode(reference["result"]["World"].as_str().unwrap())
                .unwrap()
        )
        .unwrap()
    );
    assert_eq!(observed.incarnations.len(), 3);
    assert_original_incarnations_match(&observed, &reference);
    let recovered = observed
        .incarnations
        .iter()
        .find(|item| item.handle.node == "a" && item.handle.incarnation == 2)
        .unwrap();
    assert_eq!(
        recovered.execution.outcome.local_model.as_ref().unwrap()["finished"],
        true
    );
    assert_eq!(
        STANDARD
            .decode(
                recovered.execution.outcome.local_model.as_ref().unwrap()["terminal_base64"]
                    .as_str()
                    .unwrap()
            )
            .unwrap()
            .len(),
        16
    );
    let cached = observed.incarnations.last().unwrap();
    assert_eq!(cached.handle.node, "b");
    assert_eq!(cached.handle.incarnation, 1);
    assert_ne!(cached.execution.evidence["network"], expected);
    assert_eq!(
        observed.network, expected,
        "final network must reflect the finished shared owner, without an incarnation's operations log"
    );
}

#[test]
#[ignore = "requires authenticated original storage-crash service input and Go World/network"]
fn joined_storage_crash_preserves_successful_session_world_and_network() {
    use base64::{Engine as _, engine::general_purpose::STANDARD};

    let input_path = std::env::var_os("GOMAD_LIVE_CLUSTER_STORAGE_WAIT_INPUT")
        .expect("original root34 storage-crash input path");
    let reference_path = std::env::var_os("GOMAD_LIVE_CLUSTER_STORAGE_WAIT_REFERENCE")
        .expect("original root34 storage-crash Go observation path");
    let input: Value = serde_json::from_slice(&fs::read(input_path).unwrap()).unwrap();
    let reference: Value = serde_json::from_slice(&fs::read(reference_path).unwrap()).unwrap();
    assert_eq!(
        reference["test"],
        "TestClusterBrokerDurableRestartParity/storage-crash/seed-7"
    );
    assert_eq!(reference["ordinal"], 91);
    assert_eq!(input["forcing"], json!({}));
    let request = original_request(&input);
    assert_eq!(request.nodes, ["a", "b"]);
    assert_eq!(request.incarnations.len(), 3);
    assert!(
        request
            .incarnations
            .iter()
            .all(|item| item.request.module_sha256
                == "58af6ef04c8ea078ea338cc5db1caf0ea4b727c5e6fabbe4cc47265f2f81b426")
    );
    assert_eq!(
        request
            .steps
            .iter()
            .map(|step| (
                step.action,
                step.handle.node.as_str(),
                step.handle.incarnation
            ))
            .collect::<Vec<_>>(),
        vec![
            (Action::Start, "a", 1),
            (Action::Start, "b", 1),
            (Action::Wait, "a", 1),
            (Action::Wait, "b", 1),
            (Action::Restart, "a", 2),
            (Action::Wait, "a", 2)
        ]
    );
    let observed = execute_prepared(prepare(request).unwrap()).unwrap();
    assert_eq!(observed.incarnations.len(), 3);
    assert_original_incarnations_match(&observed, &reference);
    for item in &observed.incarnations {
        let original = reference["states"]
            .as_array()
            .unwrap()
            .iter()
            .find(|state| {
                state["handle"]["ID"] == item.handle.node
                    && state["handle"]["Incarnation"] == item.handle.incarnation
            })
            .unwrap();
        let storage = item
            .execution
            .evidence
            .get("storage")
            .map_or(&[][..], Vec::as_slice);
        assert_eq!(
            storage,
            STANDARD
                .decode(
                    original["result"]["StorageTranscript"]
                        .as_str()
                        .unwrap_or("")
                )
                .unwrap()
        );
        if item.handle.node == "a" && item.handle.incarnation == 1 {
            assert_eq!(item.execution.outcome.termination, "storage_crash");
            assert!(item.interrupted);
        } else {
            assert_eq!(item.execution.outcome.termination, "exit");
            assert_eq!(item.execution.outcome.exit_code, Some(0));
            assert!(!item.interrupted);
        }
    }
    assert_eq!(
        observed.outcome.termination, "exit",
        "joined modeled storage crash is an incarnation result, not a failed session"
    );
    assert_eq!(observed.outcome.exit_code, Some(0));
    let original_world = STANDARD
        .decode(reference["result"]["World"].as_str().unwrap())
        .unwrap();
    assert_eq!(
        observed.world,
        crate::world::retained::expand(&original_world).unwrap()
    );
    assert_eq!(
        observed.network,
        STANDARD
            .decode(reference["network_wire"].as_str().unwrap())
            .unwrap()
    );
}

#[test]
#[ignore = "requires original root35 compiled service input; mandatory live corpus invokes explicitly"]
fn shutdown_reinsertion_preserves_actual_live_guest_world() {
    let input_path = std::env::var_os("GOMAD_LIVE_CLUSTER_SHUTDOWN_INPUT")
        .expect("mandatory original root35 request path");
    let output_path = std::env::var_os("GOMAD_LIVE_CLUSTER_SHUTDOWN_OUTPUT")
        .expect("mandatory new root35 evidence directory");
    let input: Value = serde_json::from_slice(&fs::read(input_path).unwrap()).unwrap();
    assert_eq!(input["schema"], "gomad.live-cluster-observation/v1");
    let output = Path::new(&output_path);
    fs::create_dir(output).unwrap();
    let mut baseline = None;
    let mut receipts = Vec::new();
    for iteration in 0..16 {
        let mut prepared = prepared(&input);
        let live = Arc::new(AtomicBool::new(false));
        prepared.shutdown_reinsertion = Some((iteration % 2 != 0, live.clone()));
        let observed = execute_prepared(prepared).unwrap();
        assert!(
            live.load(Ordering::Acquire),
            "both original started handles must be live at reinsertion"
        );
        assert_eq!(observed.incarnations.len(), 2);
        assert_eq!(
            observed.executed_steps.len(),
            2,
            "shutdown invented lifecycle actions"
        );
        assert_eq!(observed.outcome.termination, "exit");
        assert_eq!(observed.outcome.exit_code, Some(0));
        assert_ne!(observed.world, [0_u8; 0]);
        let actual = (observed.world, observed.network);
        if let Some(expected) = &baseline {
            assert_eq!(&actual, expected, "iteration {iteration}");
        } else {
            baseline = Some(actual.clone());
        }
        fs::write(output.join(format!("world-{iteration}.bin")), &actual.0).unwrap();
        fs::write(output.join(format!("network-{iteration}.bin")), &actual.1).unwrap();
        receipts.push(
            json!({"iteration":iteration,"reverse_insertion":iteration % 2 != 0,
            "live_verified":live.load(Ordering::Acquire),"completed":observed.incarnations.len(),
            "world_bytes":actual.0.len(),"network_bytes":actual.1.len()}),
        );
    }
    fs::write(
        output.join("result.json"),
        serde_json::to_vec(&json!({
            "schema":"gomad.live-cluster-shutdown-reinsertion/v1","iterations":receipts,
            "conditions":"ordinary","unchanged_guest":true,
        }))
        .unwrap(),
    )
    .unwrap();
    println!("root35: 16 actual post-Start live-map reconstruction and shutdown comparisons");
}

#[test]
fn actual_linked_world_transition_byte_bound_changes_capacity_and_replays_the_retained_prefix() {
    for limit in [1, 1 << 20, 32 << 20] {
        let make = || {
            let mut request = output_wait_request(
                1,
                "i32.const 512 i32.const 16 call $config drop
                i32.const 771 i32.const 1 i32.store8
                i32.const 783 i32.const 10 i32.store8
                i32.const 768 i32.const 16 call $idle drop
                i32.const 995 i32.const 1 i32.store8
                i32.const 992 i32.const 16 call $finish drop",
                &["a"],
                Action::Crash,
            );
            request.boundaries.clear();
            request.broker.transition_bytes = Some(limit);
            request
        };
        let broker = make().broker;
        let observed = execute_prepared(prepare(make()).unwrap()).unwrap();
        if limit == 1 {
            assert_eq!(
                observed.outcome.termination, "capacity",
                "{}",
                observed.outcome.message
            );
            assert!(observed.outcome.message.contains("transition-bytes"));
        } else {
            assert_eq!(
                observed.outcome.termination, "exit",
                "{}",
                observed.outcome.message
            );
            assert_eq!(observed.outcome.exit_code, Some(0));
            assert_eq!(
                observed.incarnations[0].execution.evidence["stdout"],
                b"mark"
            );
        }
        validate_world(&observed.world, &broker).unwrap();
        let replayed = execute_prepared(prepare(make()).unwrap()).unwrap();
        assert_eq!(replayed.world, observed.world);
        assert_eq!(replayed.outcome.termination, observed.outcome.termination);
        assert_eq!(replayed.outcome.message, observed.outcome.message);
        if limit != 1 {
            let mut foreign = broker.clone();
            foreign.transition_bytes = Some(1);
            assert!(validate_world(&observed.world, &foreign).is_err());
        }
    }
}

#[test]
fn one_and_thirty_two_mib_world_transition_bounds_change_the_same_actual_guest() {
    let body = "i32.const 512 i32.const 16 call $config drop
        i32.const 2048 i32.const 4096 i32.store
        (loop $again
            i32.const 768 i64.const 0 i64.store
            i32.const 776 i64.const 0 i64.store
            i32.const 771 i32.const 1 i32.store8
            i32.const 2052 i32.const 2052 i32.load i32.const 10 i32.add i32.store
            i32.const 782 i32.const 2052 i32.load i32.const 8 i32.shr_u i32.store8
            i32.const 783 i32.const 2052 i32.load i32.store8
            i32.const 768 i32.const 16 call $idle drop
            i32.const 2048 i32.const 2048 i32.load i32.const 1 i32.sub i32.store
            i32.const 2048 i32.load br_if $again)
        i32.const 995 i32.const 1 i32.store8
        i32.const 992 i32.const 16 call $finish drop";
    let mut module_digest = None;
    for limit in [1 << 20, 32 << 20] {
        let mut request = output_wait_request(1, body, &["a"], Action::Crash);
        request.boundaries.clear();
        request.broker.transition_bytes = Some(limit);
        request.wall_timeout = Duration::from_secs(30);
        request.evidence_bytes = 32 << 20;
        request.incarnations[0].request.limits.wall_timeout = Duration::from_secs(30);
        request.incarnations[0].request.limits.fuel = 10_000_000;
        let digest = request.incarnations[0].request.module_sha256.clone();
        if let Some(expected) = &module_digest {
            assert_eq!(&digest, expected);
        } else {
            module_digest = Some(digest);
        }
        let broker = request.broker.clone();
        let observed = execute_prepared(prepare(request).unwrap()).unwrap();
        assert_eq!(
            observed.outcome.termination,
            if limit == 1 << 20 { "capacity" } else { "exit" },
            "{}",
            observed.outcome.message
        );
        if limit == 1 << 20 {
            assert!(
                observed
                    .outcome
                    .message
                    .contains("transition-bytes limit=1048576")
            );
        }
        validate_world(&observed.world, &broker).unwrap();
        assert_ne!(observed.world, [] as [u8; 0]);
    }
}

#[test]
fn measurement_phase_callbacks_guard_actual_store_dispatch_and_join_for_both_schedules() {
    use std::sync::Mutex;
    struct Observer {
        calls: Mutex<Vec<(u64, bool)>>,
        refuse: Option<(u64, bool)>,
    }
    impl LifecycleObserver for Observer {
        fn before(&self, phase: &LifecyclePhase) -> std::result::Result<(), String> {
            assert_eq!(phase.handles.len(), 2);
            self.calls.lock().unwrap().push((phase.incarnation, true));
            if self.refuse == Some((phase.incarnation, true)) {
                return Err("before input changed".into());
            }
            Ok(())
        }
        fn after(&self, phase: &LifecyclePhase) -> std::result::Result<(), String> {
            assert_eq!(phase.handles.len(), 2);
            self.calls.lock().unwrap().push((phase.incarnation, false));
            if self.refuse == Some((phase.incarnation, false)) {
                return Err("after input changed".into());
            }
            Ok(())
        }
    }
    for slow_first in [false, true] {
        for refuse in [
            None,
            Some((1, true)),
            Some((1, false)),
            Some((2, true)),
            Some((2, false)),
        ] {
            let mut request = output_wait_request(1, "", &["a", "b"], Action::Stop);
            request.boundaries.clear();
            let second = output_wait_request(1, "", &["a", "b"], Action::Stop)
                .incarnations
                .into_iter()
                .map(|mut item| {
                    item.handle.incarnation = 2;
                    item
                })
                .collect::<Vec<_>>();
            request.incarnations.extend(second);
            for (index, incarnation) in request.incarnations.iter_mut().enumerate() {
                let module = if (index % 2 == 0) == slow_first {
                    br#"(module (memory (export "memory") 1) (func (export "_start") (local $i i32)
                    (loop $loop local.get $i i32.const 1 i32.add local.tee $i i32.const 10000 i32.lt_u br_if $loop)))"#.to_vec()
                } else {
                    br#"(module (memory (export "memory") 1) (func (export "_start")))"#.to_vec()
                };
                incarnation.request.module_sha256 = format!("{:x}", Sha256::digest(&module));
                incarnation.request.module = module;
                incarnation.request.model = None;
            }
            request.steps = [(1, Action::Start), (2, Action::Restart)]
                .into_iter()
                .flat_map(|(incarnation, start)| {
                    let order = if slow_first {
                        [
                            ("a", start),
                            ("b", start),
                            ("a", Action::Wait),
                            ("b", Action::Wait),
                        ]
                    } else {
                        [
                            ("a", start),
                            ("a", Action::Wait),
                            ("b", start),
                            ("b", Action::Wait),
                        ]
                    };
                    order.map(move |(node, action)| Step {
                        action,
                        handle: Handle {
                            node: node.into(),
                            incarnation,
                        },
                    })
                })
                .collect();
            let observer = Observer {
                calls: Mutex::new(Vec::new()),
                refuse,
            };
            let result = execute_prepared_observed(prepare(request).unwrap(), Some(&observer));
            let expected = [(1, true), (1, false), (2, true), (2, false)];
            let through = refuse.map_or(4, |refuse| {
                expected.iter().position(|value| *value == refuse).unwrap() + 1
            });
            assert_eq!(*observer.calls.lock().unwrap(), expected[..through]);
            if refuse.is_some() {
                assert!(result.err().unwrap().to_string().contains("input changed"));
            } else {
                let run = result.unwrap();
                assert_eq!(run.incarnations.len(), 4);
                assert!(
                    run.incarnations
                        .iter()
                        .all(|item| item.execution.outcome.termination == "exit"
                            && item.execution.outcome.exit_code == Some(0))
                );
                assert_eq!(run.executed_steps.len(), 8);
            }
        }
    }
}
