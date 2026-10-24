//! Replays the recorded Go broker orders
//! (`wasi/broker_test.go` `TestBrokerOrdersAreCurrent`) through the in-process
//! broker: scripted node call sequences on threads, as instance stores drive
//! it. Every call's output or error and the broker World's expanded wire must
//! equal the Go broker's; each node's network evidence is checked in for the
//! Go decoder (`TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments`).
use super::*;
use crate::cluster::network::{self as net, Endpoint, Fault, Link, ModelV2};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::Value;
use std::collections::BTreeMap;
use std::path::PathBuf;

#[path = "corpus.rs"]
mod corpus;

#[path = "readiness_probe_tests.rs"]
mod readiness_probe_tests;

fn directory() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/broker_orders")
}

fn broker_config() -> Config {
    Config {
        seed: 7,
        limits: WorldLimits {
            max_requests: 10000,
            max_events: 10000,
            max_queued_events: 1000,
            max_transitions: 40000,
            max_payload_bytes: 32 << 20,
            max_string_bytes: 1024,
        },
        pause_on_settle: false,
        transition_bytes: None,
    }
}

#[test]
fn changed_inspection_boundaries_never_materialize_world_history() {
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let mut previous = broker.inspection_world_commitment(None).unwrap();
    for ordinal in 0..1000u64 {
        broker.register(0, "idle", &ordinal.to_le_bytes()).unwrap();
        let current = broker.inspection_world_commitment(None).unwrap();
        assert_ne!(current, previous);
        previous = current;
        broker.ready(0, "idle").unwrap();
        broker.inspection_world_commitment(None).unwrap();
        lock(&broker.inner).world.quiesce().unwrap();
        broker.inspection_world_commitment(None).unwrap();
    }
    assert_eq!(
        broker
            .world_materializations
            .load(std::sync::atomic::Ordering::Relaxed),
        0,
        "inspection must bind the transition chain without scanning history"
    );
}

#[test]
fn finish_preserves_world_transition_capacity() {
    let mut config = broker_config();
    config.limits.max_transitions = 1;
    let broker = Broker::new(&config, &["a".into()], None, None).unwrap();
    broker.register(0, "idle", &[]).unwrap();
    assert!(matches!(
        broker.ready(0, "idle"),
        Err(Failure::Exhausted {
            dimension: "transitions",
            ..
        })
    ));
    broker.finish(0);
    let (termination, message) = broker.failure().unwrap();
    assert_eq!(termination, "capacity");
    assert!(
        message.contains("transitions limit=1 used=1 delta=1"),
        "{message}"
    );
}

#[test]
fn failed_peer_preserves_parked_incarnations_declared_boundary() {
    for failure in ["trap", "capacity"] {
        for boundary in ["crash", "stop"] {
            let broker = Arc::new(
                Broker::new(&broker_config(), &["a".into(), "b".into()], None, None).unwrap(),
            );
            broker.defer_starts();
            broker.activate(0).unwrap();
            broker.activate(1).unwrap();
            broker.run(1, 1).unwrap();
            let parked = broker.clone();
            let thread = std::thread::spawn(move || {
                assert!(parked.acquire(0));
                parked.idle(0, 10_000)
            });
            assert!(broker.acquire(1));
            broker
                .revoke_locked(&mut lock(&broker.inner), 0, boundary)
                .unwrap();
            broker.abort(failure, "failed peer".into());
            assert!(
                matches!(
                    thread.join().unwrap(),
                    Err(Failure::Boundary(actual, _)) if actual == boundary
                ),
                "{failure} must preserve {boundary}"
            );
        }
    }
}

#[test]
fn failed_cluster_drains_source_then_sorted_peers_under_thread_jitter() {
    let mut expected_wire = None;
    for failure in ["trap", "capacity"] {
        for launch in [[2, 1], [1, 2], [2, 1], [1, 2]] {
            let broker = Arc::new(
                Broker::new(
                    &broker_config(),
                    &["a".into(), "b".into(), "c".into()],
                    None,
                    None,
                )
                .unwrap(),
            );
            broker.start();
            for index in 0..3 {
                broker.register(index, "idle", &[]).unwrap();
            }
            broker.abort(failure, "failed source a".into());
            let mut threads = Vec::new();
            let (finished, completed) = std::sync::mpsc::channel();
            for index in launch {
                let broker = broker.clone();
                let finished = finished.clone();
                threads.push(std::thread::spawn(move || {
                    for _ in 0..index {
                        std::thread::yield_now();
                    }
                    broker.finish(index);
                    finished.send(index).unwrap();
                }));
                assert!(
                    matches!(
                        completed.recv_timeout(std::time::Duration::from_millis(50)),
                        Err(std::sync::mpsc::RecvTimeoutError::Timeout)
                    ),
                    "a peer finished before the failure source"
                );
            }
            broker.finish(0);
            for thread in threads {
                thread.join().unwrap();
            }
            let canceled: Vec<_> = lock(&broker.inner)
                .world
                .transitions()
                .iter()
                .filter_map(|transition| match &transition.body {
                    world::TransitionBody::Cancel { request_id, .. } => Some(*request_id),
                    _ => None,
                })
                .collect();
            assert_eq!(canceled, [1, 2, 3]);
            let wire = broker.world_wire().unwrap();
            if let Some(expected) = &expected_wire {
                assert_eq!(&wire, expected);
            } else {
                expected_wire = Some(wire);
            }
        }
    }
}

#[cfg(feature = "stress-jitter")]
#[test]
fn event_order_controller_commands_match_parser_fixture() {
    if crate::event_order::is_enabled() {
        let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
        broker
            .command(Command::Revoke {
                index: 0,
                incarnation: 1,
                kind: "crash",
            })
            .unwrap();
        crate::event_order::finish();
        return;
    }
    let output = std::process::Command::new(std::env::current_exe().unwrap())
        .args([
            "--exact",
            "cluster::handoff::broker::tests::event_order_controller_commands_match_parser_fixture",
            "--nocapture",
        ])
        .env("GOMAD_WASM_EVENT_ORDER", "1")
        .output()
        .unwrap();
    assert!(output.status.success(), "{output:?}");
    let stderr = String::from_utf8(output.stderr).unwrap();
    let mut frames = String::new();
    for line in stderr
        .lines()
        .filter(|line| line.starts_with("gomad-wasm-event-order-v1 "))
    {
        frames.push_str(line);
        frames.push('\n');
    }
    assert_eq!(
        frames,
        include_str!(
            "../../../../../../internal/backends/wasm/determinism/testdata/controller-event-order.stderr"
        )
    );
}

#[cfg(feature = "diagnostic-profile")]
#[test]
fn diagnostic_world_calls_charge_nested_exclusive_owner_wall() {
    use crate::diagnostic::{self, AbiSpan, ImportBoundary, Owner, Phase, PhaseSpan, Session};
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let _session = Session::start();
    {
        let _execute = PhaseSpan::new(Phase::Execute);
        let _boundary = ImportBoundary::new();
        let _abi = AbiSpan::new(crate::abi::Operation::RuntimeIdle, true);
        diagnostic::serve(Owner::SchedulerHandoff);
        broker.register(0, "idle", &[]).unwrap();
        broker.ready(0, "idle").unwrap();
        broker.world_wire().unwrap();
    }
    let report = diagnostic::snapshot();
    let number = |value: &Value| value.as_u64().unwrap();
    let world = &report["owners"]["world"];
    let handoff = &report["owners"]["scheduler_handoff"];
    assert_eq!(number(&world["count"]), 3);
    assert!(number(&world["execute_exclusive_wall_nanos"]) > 0);
    assert_eq!(
        number(&world["wall_nanos"]) + number(&handoff["exclusive_wall_nanos"]),
        number(&handoff["wall_nanos"])
    );
    assert_eq!(report["reconciliation"]["difference_nanos"], 0);
    assert_eq!(report["span_errors"], 0);
}

fn assert_world_commitment(broker: &Broker, network: Option<&[u8]>) {
    use crate::model::inspection::{commitment, world_chain_commitment};
    let key = network.map(|bytes| commitment("network", bytes).try_into().unwrap());
    let digest = broker.inspection_world_commitment(key).unwrap();
    assert_eq!(
        digest.as_slice(),
        world_chain_commitment(&lock(&broker.inner).world.inspection_commitment(), key,)
    );
    let count = broker
        .world_materializations
        .load(std::sync::atomic::Ordering::Relaxed);
    assert_eq!(broker.inspection_world_commitment(key).unwrap(), digest);
    assert_eq!(
        broker
            .world_materializations
            .load(std::sync::atomic::Ordering::Relaxed),
        count
    );
}

#[test]
fn network_only_inspection_changes_bind_the_network_digest_without_world_wire() {
    use crate::model::inspection::{commitment, world_chain_commitment};
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let world = lock(&broker.inner).world.inspection_commitment();
    let before = broker
        .world_materializations
        .load(std::sync::atomic::Ordering::Relaxed);
    for network in [
        None,
        Some(b"".as_slice()),
        Some(b"a".as_slice()),
        Some(b"different length".as_slice()),
        None,
    ] {
        let identity: Option<[u8; 32]> =
            network.map(|bytes| commitment("network", bytes).try_into().unwrap());
        if let Some(identity) = identity {
            assert_eq!(
                broker.inspection_world_commitment(Some(identity)).unwrap(),
                <[u8; 32]>::try_from(world_chain_commitment(&world, Some(identity))).unwrap()
            );
        }
        assert_eq!(
            broker
                .inspection_world_commitment(identity)
                .unwrap()
                .as_slice(),
            world_chain_commitment(&world, identity)
        );
    }
    assert_eq!(
        broker
            .world_materializations
            .load(std::sync::atomic::Ordering::Relaxed),
        before
    );
}

/// The head plus retained transition encodings equal the generated wire,
/// and are refused exactly when the bounded generated write is, whether
/// the head, the retained transitions or the new ones cross the bound.
#[test]
fn inspection_wire_reuses_transitions_and_refuses_at_the_generated_bound() {
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let mut inner = lock(&broker.inner);
    let mut ids = Vec::new();
    for step in 0..60u64 {
        match step % 3 {
            0 => ids.push(
                inner
                    .world
                    .register(world::Request {
                        kind: "wait".into(),
                        resource: world::ResourceId {
                            adapter: "test".into(),
                            kind: "timer".into(),
                            key: format!("key-{step}"),
                        },
                        priority: 0,
                        payload: vec![step.to_le_bytes()[0]; (step % 7) as usize * 40],
                    })
                    .unwrap(),
            ),
            1 => {
                let id = ids[ids.len() / 2];
                let _ = inner.world.ready(world::Readiness {
                    request_id: id,
                    at: world::INITIAL_TIME + step.cast_signed(),
                    kind: "timer".into(),
                    payload: vec![1; 9],
                    equivalence_class: String::new(),
                });
            }
            _ => {
                inner.world.quiesce().unwrap();
            }
        }
        let mut uncached = Vec::new();
        world::wire_generated::write_snapshot(&inner.world, &mut uncached).unwrap();
        // Full retention first, so a smaller bound refuses on the retained
        // transitions; then the head, the last new transition and a reset.
        for maximum in [
            usize::MAX,
            uncached.len() / 2,
            uncached.len(),
            uncached.len() - 1,
            16,
            usize::MAX,
        ] {
            let mut bounded = Bounded {
                bytes: Vec::new(),
                limit: maximum,
            };
            let expected = world::wire_generated::write_snapshot(&inner.world, &mut bounded)
                .map_err(world_wire_failure);
            match broker.inspection_wire_locked(&mut inner, maximum) {
                Ok(wire) => {
                    assert!(expected.is_ok(), "{step} {maximum}");
                    let transitions = &inner.inspection_transitions.bytes[..wire.transitions];
                    assert_eq!([&wire.head[..], transitions].concat(), uncached);
                }
                Err(refused) => assert_eq!(Err(refused), expected, "{step} {maximum}"),
            }
        }
    }
}

#[test]
fn oversized_world_discards_stale_wire_and_never_publishes_partial_serialization() {
    let mut config = broker_config();
    config.limits.max_payload_bytes = 1 << 30;
    let broker = Broker::new(&config, &["a".into()], None, None).unwrap();
    broker.inspection_world_commitment(None).unwrap();
    lock(&broker.inner)
        .world
        .register(world::Request {
            kind: "wait".into(),
            resource: world::ResourceId {
                adapter: "test".into(),
                kind: "timer".into(),
                key: "large".into(),
            },
            priority: 0,
            payload: vec![7; 192 << 20],
        })
        .unwrap();
    let before = lock(&broker.inner).world.state_digest();
    assert!(matches!(
        broker.inspection_world_commitment(None),
        Err(Failure::Capacity("compressed-snapshot-expanded-bytes"))
    ));
    let inner = lock(&broker.inner);
    assert!(inner.inspection_world.is_none());
    assert_eq!(inner.world.state_digest(), before);
}

#[test]
fn inspection_world_cache_matches_every_world_mutation_and_failed_operations() {
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let request = || world::Request {
        kind: "wait".into(),
        resource: world::ResourceId {
            adapter: "test".into(),
            kind: "timer".into(),
            key: "a/1".into(),
        },
        priority: 0,
        payload: vec![7],
    };
    let check = || {
        for network in [
            None,
            Some(b"".as_slice()),
            Some(b"fresh network evidence".as_slice()),
            None,
        ] {
            assert_world_commitment(&broker, network);
        }
    };
    check();
    let first = lock(&broker.inner).world.register(request()).unwrap();
    check();
    lock(&broker.inner)
        .world
        .ready(world::Readiness {
            request_id: first,
            at: world::INITIAL_TIME + 17,
            kind: "timer".into(),
            payload: vec![8],
            equivalence_class: "same".into(),
        })
        .unwrap();
    check();
    lock(&broker.inner).world.quiesce().unwrap();
    check();
    lock(&broker.inner).world.cancel(first).unwrap();
    check();
    let second = lock(&broker.inner).world.register(request()).unwrap();
    check();
    lock(&broker.inner).world.quiesce().unwrap();
    check();
    lock(&broker.inner).world.cancel(second).unwrap();
    check();
    lock(&broker.inner).world.cancel(second).unwrap();
    check();
    lock(&broker.inner).world.quiesce().unwrap();
    check();
    let third = lock(&broker.inner).world.register(request()).unwrap();
    check();
    assert_world_commitment(&broker, None);
    let before = broker.world_wire().unwrap();
    let count = broker
        .world_materializations
        .load(std::sync::atomic::Ordering::Relaxed);
    {
        let mut inner = lock(&broker.inner);
        assert!(inner.world.cancel(0).is_err());
        assert!(
            inner
                .world
                .ready(world::Readiness {
                    request_id: third,
                    at: 0,
                    kind: "invalid".into(),
                    payload: vec![],
                    equivalence_class: String::new()
                })
                .is_err()
        );
        let mut invalid = request();
        invalid.kind.clear();
        assert!(inner.world.register(invalid).is_err());
    }
    assert_eq!(broker.world_wire().unwrap(), before);
    broker.inspection_world_commitment(None).unwrap();
    assert_eq!(
        broker
            .world_materializations
            .load(std::sync::atomic::Ordering::Relaxed),
        count + 1
    );
}

type Outputs = Arc<Mutex<BTreeMap<String, Vec<Value>>>>;
/// Outputs per node, the broker World's expanded wire and network evidence.
type Replayed = (
    BTreeMap<String, Vec<Value>>,
    Vec<u8>,
    BTreeMap<String, Vec<u8>>,
);

fn step_data(step: &Value, outputs: &Outputs) -> Vec<u8> {
    let field = |key: &str| step.get(key).and_then(Value::as_u64).unwrap_or(0);
    if step["op"] == "idle" {
        let mut data = vec![0u8; 16];
        data[..4].copy_from_slice(&1u32.to_be_bytes());
        data[8..].copy_from_slice(&field("Deadline").max(field("deadline")).to_be_bytes());
        return data;
    }
    let outputs = lock(outputs);
    let resolve = |key: &str, at: usize, width: usize| -> u64 {
        let Some(reference) = step.get(key).filter(|value| !value.is_null()) else {
            return 0;
        };
        let node = reference["node"].as_str().unwrap();
        let index = usize::try_from(reference["step"].as_u64().unwrap()).unwrap();
        let data = STANDARD
            .decode(outputs[node][index]["data_base64"].as_str().unwrap())
            .unwrap();
        if width == 8 {
            u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
        } else {
            u64::from(u32::from_be_bytes(data[at..at + 4].try_into().unwrap()))
        }
    };
    let code = u32::try_from(field("code")).unwrap();
    let count = u32::try_from(field("count")).unwrap();
    let mut payload = step["payload"].as_str().unwrap_or("").as_bytes().to_vec();
    if code == 4 || code == 12 {
        payload = vec![0; count as usize];
    }
    let mut data = vec![0u8; 64];
    data[..4].copy_from_slice(&2u32.to_be_bytes());
    data[4..8].copy_from_slice(&code.to_be_bytes());
    data[8..16].copy_from_slice(&field("id").to_be_bytes());
    data[16..24].copy_from_slice(&resolve("handle", 16, 8).to_be_bytes());
    data[32..36].copy_from_slice(&(u32::try_from(resolve("port", 48, 4)).unwrap()).to_be_bytes());
    let header_count = if code == 12 { 0 } else { count };
    data[36..40].copy_from_slice(&header_count.to_be_bytes());
    for (index, byte) in step["address"].as_array().unwrap().iter().enumerate() {
        data[56 + index] = u8::try_from(byte.as_u64().unwrap()).unwrap();
    }
    data.extend(payload);
    data
}

fn network_config(scenario: &Value) -> Option<(net::Config, ModelV2)> {
    let model: ModelV2 = serde_json::from_value(scenario.get("network")?.clone()).ok()?;
    let text = |value: &Value, key: &str| value[key].as_str().unwrap_or("").to_owned();
    let number = |value: &Value, key: &str| value[key].as_u64().unwrap_or(0);
    let faults = scenario["faults"]
        .as_array()
        .map(|faults| {
            faults
                .iter()
                .map(|fault| Fault {
                    name: text(fault, "Name"),
                    action: text(fault, "Action"),
                    from: text(fault, "From"),
                    to: text(fault, "To"),
                    at_nanos: number(fault, "AtNanos"),
                    delay_nanos: number(fault, "DelayNanos"),
                    symmetric: fault["Symmetric"].as_bool().unwrap_or(false),
                })
                .collect()
        })
        .unwrap_or_default();
    let links = scenario["links"]
        .as_array()
        .map(|links| {
            links
                .iter()
                .map(|link| Link {
                    from: text(link, "From"),
                    to: text(link, "To"),
                    enabled: link["Enabled"].as_bool().unwrap_or(false),
                    delay_nanos: number(link, "DelayNanos"),
                })
                .collect()
        })
        .unwrap_or_default();
    Some((
        net::Config {
            model: model.clone(),
            addresses: BTreeMap::new(),
            links,
            strict_topology: scenario["strict_topology"].as_bool().unwrap_or(false),
            faults,
        },
        model,
    ))
}

/// Runs one recorded scenario: outputs per node, the World wire and each
/// node's network evidence.
fn replay(scenario: &Value) -> Replayed {
    let ids: Vec<String> = scenario["nodes"]
        .as_array()
        .unwrap()
        .iter()
        .map(|id| id.as_str().unwrap().to_owned())
        .collect();
    let network = network_config(scenario);
    let shared = network
        .as_ref()
        .map(|(config, _)| Arc::new(Mutex::new(Shared::new(config, &ids).unwrap())));
    let broker = Arc::new(Broker::new(&broker_config(), &ids, shared.clone(), None).unwrap());
    let outputs: Outputs = Arc::new(Mutex::new(BTreeMap::new()));
    let evidence = Arc::new(Mutex::new(BTreeMap::new()));
    let threads: Vec<_> = ids
        .iter()
        .enumerate()
        .map(|(index, id)| {
            let (broker, outputs, evidence, shared) = (
                broker.clone(),
                outputs.clone(),
                evidence.clone(),
                shared.clone(),
            );
            let id = id.clone();
            let steps = scenario["steps"][&id]
                .as_array()
                .cloned()
                .unwrap_or_default();
            let profile = network.as_ref().map(|(_, model)| model.clone());
            std::thread::spawn(move || {
                let endpoint = shared.as_ref().map(|shared| {
                    Endpoint::new(
                        shared.clone(),
                        &id,
                        1,
                        index as u64 + 1,
                        profile.as_ref().unwrap(),
                        946_684_800_000_000_000,
                        8 << 20,
                    )
                    .unwrap()
                });
                let mut member = Member {
                    broker: broker.clone(),
                    incarnation: 1,
                    index,
                    network: endpoint,
                };
                let record = |value: Value| {
                    lock(&outputs).entry(id.clone()).or_default().push(value);
                };
                assert!(broker.acquire(index));
                let (mut now, mut advances) = (1u64, 0u64);
                for step in steps {
                    if step["op"] == "advance_now" {
                        now += step["nanos"].as_u64().unwrap();
                        record(serde_json::json!({}));
                        continue;
                    }
                    let mut data = step_data(&step, &outputs);
                    let mut clock = Clockwork {
                        now: &mut now,
                        advances: &mut advances,
                        runtime_active: true,
                    };
                    let result = if step["op"] == "idle" {
                        member.idle(&mut data, &mut clock)
                    } else {
                        member.net_stream(&mut data, &mut clock)
                    };
                    match result {
                        Ok(()) => {
                            record(serde_json::json!({"data_base64": STANDARD.encode(&data)}));
                        }
                        Err(failure) => {
                            record(serde_json::json!({"error": failure.message()}));
                            break;
                        }
                    }
                }
                if let Some(endpoint) = &member.network {
                    lock(&evidence).insert(id.clone(), endpoint.evidence().unwrap());
                }
                broker.finish(index);
            })
        })
        .collect();
    broker.start();
    assert!(broker.wait());
    for thread in threads {
        thread.join().unwrap();
    }
    let outputs = std::mem::take(&mut *lock(&outputs));
    let evidence = std::mem::take(&mut *lock(&evidence));
    for bytes in evidence.values() {
        assert_world_commitment(&broker, Some(bytes));
    }
    (outputs, broker.world_wire().unwrap(), evidence)
}

#[test]
fn broker_replays_recorded_go_orders() {
    let regenerate = std::env::var("GOMAD_BROKER_ORDERS_REGENERATE").as_deref() == Ok("1");
    let mut files: Vec<_> = std::fs::read_dir(directory())
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| {
            path.extension()
                .is_some_and(|extension| extension == "json")
        })
        .collect();
    files.sort();
    assert!(files.len() >= 4);
    for path in files {
        let scenario: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        let name = scenario["name"].as_str().unwrap().to_owned();
        let (outputs, wire, evidence) = replay(&scenario);
        let expected: BTreeMap<String, Vec<Value>> =
            serde_json::from_value(scenario["outputs"].clone()).unwrap();
        for (node, want) in &expected {
            let got = outputs.get(node).cloned().unwrap_or_default();
            for (index, (got, want)) in got.iter().zip(want).enumerate() {
                let want = if want.as_object().is_some_and(serde_json::Map::is_empty) {
                    serde_json::json!({})
                } else {
                    want.clone()
                };
                assert_eq!(got, &want, "{name} {node} step {index}");
            }
            assert_eq!(got.len(), want.len(), "{name} {node} step count");
        }
        assert_eq!(
            STANDARD.encode(&wire),
            scenario["world_base64"].as_str().unwrap(),
            "{name}: World expanded wire"
        );
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"network.broker.replay","inputs":{"name":name},"observed":{"outputs":outputs,"world_base64":STANDARD.encode(&wire),"evidence":evidence.iter().map(|(node, bytes)| (node.clone(), STANDARD.encode(bytes))).collect::<BTreeMap<_, _>>()}})
        );
        for (node, bytes) in evidence {
            #[cfg(not(feature = "conformance"))]
            {
                let (_, profile) = network_config(&scenario).unwrap();
                crate::linked_cluster::validate_network_evidence(
                    &bytes,
                    8 << 20,
                    &profile,
                    &profile,
                )
                .unwrap();
            }
            let path = directory().join(format!("{name}.{node}.gmnetev1"));
            if regenerate {
                std::fs::write(&path, &bytes).unwrap();
            } else {
                assert_eq!(
                    std::fs::read(&path).unwrap(),
                    bytes,
                    "{name} {node}: stale network evidence"
                );
            }
        }
    }
}

#[test]
fn broker_world_wire_is_bounded_while_written() {
    use std::io::Write as _;
    let mut wire = Bounded {
        bytes: Vec::new(),
        limit: 8,
    };
    wire.write_all(b"GMWSNAP1").unwrap();
    let error = wire.write_all(b"x").unwrap_err();
    assert_eq!(error.kind(), std::io::ErrorKind::FileTooLarge);
    assert_eq!(wire.bytes, b"GMWSNAP1");
}

/// Incremental inspection recording against the original eager oracle: the
/// recorded Go broker orders drive full instance states (local model,
/// optional namespace and storage, broker member and network endpoint) on
/// their own threads, with boundaries, namespace mutations, crashes, a second
/// incarnation of every node and optional sealing or exhaustion interleaved.
mod inspection_differential {
    use super::*;
    use crate::engine::{Snapshot, State};
    use crate::model::inspection::{self, Handle, Witness};
    use crate::namespace::{Image, ImageEntry, Limits, Namespace, Open, Operation};

    #[derive(Clone, Copy, PartialEq, Eq, Debug)]
    enum Interrupt {
        None,
        Seal,
        Exhaust,
    }

    #[derive(Clone, Copy, Debug)]
    struct Variant {
        namespace: bool,
        storage: bool,
        interrupt: Interrupt,
    }

    fn model(handle: &Handle, recorder: &Arc<Mutex<inspection::Recorder>>) -> crate::model::Model {
        let config: crate::model::Config = serde_json::from_value(serde_json::json!({
            "seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],
            "working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
            "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},
            "epoch_nanos":1000,"read_step_nanos":3,"now":1,
            "limits":{"calls":1_000_000,"transcript_bytes":1 << 24,"pending_events":16}}))
        .unwrap();
        config.validate().unwrap();
        let mut model = crate::model::Model::new(config);
        model.runtime("runtime_config", vec![0; 16]).unwrap();
        model.inspection.recorder = recorder.clone();
        model.inspection.handle = handle.clone();
        model.inspection.started = true;
        model
    }

    /// The first debug exchange, once the store holds the continuation.
    fn observe(state: &State) {
        let model = state.model.as_ref().unwrap();
        model.inspection.lock().observe(&model.inspection.handle);
    }

    fn namespace(variant: Variant) -> Option<Namespace> {
        if !variant.namespace {
            return None;
        }
        let mut namespace = Namespace::new(Image {
            entries: vec![ImageEntry {
                path: b"/workspace".to_vec(),
                directory: true,
                readonly: false,
                data: vec![],
            }],
            writable_directories: vec![b"/workspace".to_vec()],
            preopens: vec![b"/workspace".to_vec()],
            working_directory: b"/workspace".to_vec(),
            stdin: Vec::new(),
            limits: Limits {
                files: 64,
                descriptors: 64,
                filesystem_bytes: 1 << 20,
                output_bytes: 1 << 20,
            },
            epoch_nanos: 1000,
        })
        .unwrap();
        if variant.storage {
            let model: crate::volume::StorageModel =
                serde_json::from_value(storage_model()).unwrap();
            namespace
                .attach_storage(crate::volume::Storage::new(model, 1 << 24).unwrap())
                .unwrap();
        }
        Some(namespace)
    }

    fn storage_model() -> Value {
        serde_json::json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1 << 20}],
            "limits":{"PendingOperations":1000,"Transitions":100_000}
        })
    }

    fn state(member: Member, namespace: Option<Namespace>, model: crate::model::Model) -> State {
        State {
            memory: None,
            input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
            output: Box::new(Vec::<u8>::new()),
            limits: crate::engine::Limits {
                max: 65536,
                peak: 65536,
                failure: None,
            },
            calls: 0,
            completed_calls: 0,
            host_nanos: None,
            stdout_progress: false,
            output_limit: 1 << 20,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: Some(model),
            dormant_debug: None,
            namespace,
            request_scratch: Vec::new(),
            write_scratch: crate::protocol::WriteBuffers::default(),
            broker: Some(member),
            network: None,
            helper_owned: false,
        }
    }

    fn witness(ordinal: u64) -> Witness {
        let mut raw = vec![0; 128];
        raw[..8].copy_from_slice(&(2 * ordinal + 1).to_be_bytes());
        raw[16..24].copy_from_slice(&(ordinal * 3).to_be_bytes());
        raw[104..112].copy_from_slice(&ordinal.to_be_bytes());
        raw[112..120].copy_from_slice(&1u64.to_be_bytes());
        Witness::decode(&raw).unwrap()
    }

    /// A debug decision: the boundary snapshot, then the boundary itself.
    fn decide(state: &mut State, ordinal: &mut u64, log: &mut Vec<String>) {
        state.inspection_snapshot(Snapshot::Boundary);
        let model = state.model.as_mut().unwrap();
        let handle = model.inspection.handle.clone();
        model.inspection.witness = witness(*ordinal);
        let result = model.inspection.lock().boundary(
            &handle,
            witness(*ordinal),
            model.now,
            vec![ordinal.to_le_bytes()[0]; (*ordinal % 3) as usize],
            &model.inspection.world_commitment,
        );
        log.push(match result {
            Ok(boundary) => format!(
                "boundary {} {}",
                boundary.order,
                STANDARD.encode(serde_json::to_vec(&boundary.cuts).unwrap())
            ),
            Err(error) => format!("boundary error {error}"),
        });
        *ordinal += 1;
    }

    fn execute(state: &mut State, operation: Operation) -> Option<u32> {
        let reply = state
            .namespace
            .as_mut()
            .unwrap()
            .execute(operation, 7)
            .unwrap();
        match reply.output {
            Some(crate::namespace::Output::Fd(fd)) => Some(fd),
            _ => None,
        }
    }

    /// Namespace mutations, including same-length rewrites, revision-only
    /// touches and drained output.
    fn mutate(state: &mut State, step: usize) {
        if state.namespace.is_none() {
            return;
        }
        let open = |name: &str| {
            Operation::Open(Open {
                fd: 3,
                path: name.as_bytes().to_vec(),
                dirflags: 0,
                oflags: 1,
                base: crate::namespace::RIGHTS_ALL,
                inheriting: 0,
                flags: 0,
            })
        };
        match step % 5 {
            0 | 1 => {
                let fd = execute(state, open(&format!("f{}", step % 2))).unwrap();
                execute(
                    state,
                    Operation::Write {
                        fd,
                        data: vec![step.to_le_bytes()[0]; 5 + step % 7],
                        offset: None,
                    },
                );
                execute(state, Operation::Close(fd));
            }
            2 => {
                // Same length, different content.
                let fd = execute(state, open("f0")).unwrap();
                execute(
                    state,
                    Operation::Write {
                        fd,
                        data: vec![!(step.to_le_bytes()[0]); 5],
                        offset: Some(0),
                    },
                );
                execute(state, Operation::Close(fd));
            }
            3 => {
                execute(state, Operation::FilestatGet(3));
            }
            _ => {
                execute(
                    state,
                    Operation::Write {
                        fd: 1,
                        data: format!("out{step}").into_bytes(),
                        offset: None,
                    },
                );
                if step.is_multiple_of(2) {
                    state.namespace.as_mut().unwrap().take_output();
                }
            }
        }
    }

    /// The instance's terminal as `cluster::run_instance` records it.
    fn terminate(state: &mut State, failed: bool) {
        record_terminal(state, failed).unwrap();
    }

    fn record_terminal(
        state: &mut State,
        failed: bool,
    ) -> Result<Option<Vec<u8>>, (&'static str, String)> {
        let storage = state
            .namespace
            .as_ref()
            .is_some_and(|namespace| namespace.storage().is_some())
            .then(|| {
                serde_json::from_value::<crate::storage_transport::StorageDescriptor>(
                    serde_json::json!({
                        "model": storage_model(), "model_identity": "differential",
                        "return_image": false,
                    }),
                )
                .unwrap()
            });
        let mut outcome = if failed {
            crate::protocol::Outcome::new("crash", "differential crash")
        } else {
            let mut outcome = crate::protocol::Outcome::new("exit", "");
            outcome.exit_code = Some(0);
            outcome
        };
        crate::cluster::execution::record_terminal(state, storage.as_ref(), &mut outcome)
    }

    fn interrupt(state: &State, variant: Variant) {
        let mut recorder = state.model.as_ref().unwrap().inspection.lock();
        match variant.interrupt {
            Interrupt::None => {}
            Interrupt::Seal => recorder.sealed = true,
            Interrupt::Exhaust => recorder.exhausted = true,
        }
    }

    /// Runs one store thread; a panic aborts the broker so that parked peers
    /// fail instead of waiting forever for the continuation.
    fn guarded(broker: &Arc<Broker>, body: impl FnOnce()) {
        if let Err(panic) = std::panic::catch_unwind(std::panic::AssertUnwindSafe(body)) {
            broker.abort("infrastructure", "differential store panicked".into());
            std::panic::resume_unwind(panic);
        }
    }

    /// Everything one run recorded: the final inspection evidence, every
    /// boundary result and every broker call's output, per node.
    type Run = (Vec<u8>, BTreeMap<String, Vec<String>>);

    #[expect(
        clippy::too_many_lines,
        reason = "The scenario retains the complete ordered Go conformance sequence"
    )]
    fn run(scenario: &Value, variant: Variant, eager: bool) -> Run {
        let ids: Vec<String> = scenario["nodes"]
            .as_array()
            .unwrap()
            .iter()
            .map(|id| id.as_str().unwrap().to_owned())
            .collect();
        let network = network_config(scenario);
        let shared = network
            .as_ref()
            .map(|(config, _)| Arc::new(Mutex::new(Shared::new(config, &ids).unwrap())));
        let broker = Arc::new(Broker::new(&broker_config(), &ids, shared.clone(), None).unwrap());
        let outputs: Outputs = Arc::new(Mutex::new(BTreeMap::new()));
        let logs = Arc::new(Mutex::new(BTreeMap::<String, Vec<String>>::new()));
        let profile = network.as_ref().map(|(_, model)| model.clone());
        let endpoint = |id: &str, incarnation, slot| {
            shared.as_ref().map(|shared| {
                Endpoint::new(
                    shared.clone(),
                    id,
                    incarnation,
                    slot,
                    profile.as_ref().unwrap(),
                    946_684_800_000_000_000,
                    8 << 20,
                )
                .unwrap()
            })
        };
        // First incarnations: the recorded orders.
        let threads: Vec<_> = ids
            .iter()
            .enumerate()
            .map(|(index, id)| {
                let (broker, outputs, logs) = (broker.clone(), outputs.clone(), logs.clone());
                let id = id.clone();
                let steps = scenario["steps"][&id]
                    .as_array()
                    .cloned()
                    .unwrap_or_default();
                let endpoint = endpoint(&id, 1, index as u64 + 1);
                std::thread::spawn(move || {
                    guarded(&broker.clone(), || {
                        inspection::set_eager_oracle(eager);
                        let handle = Handle {
                            node: id.clone(),
                            incarnation: 1,
                        };
                        let model = model(&handle, &broker.inspection);
                        let member = Member {
                            broker: broker.clone(),
                            incarnation: 1,
                            index,
                            network: endpoint,
                        };
                        let mut state = state(member, namespace(variant), model);
                        let mut log = Vec::new();
                        let mut ordinal = 0;
                        assert!(broker.acquire(index));
                        observe(&state);
                        decide(&mut state, &mut ordinal, &mut log);
                        let mut failed = false;
                        for (step_index, step) in steps.iter().enumerate() {
                            if step["op"] == "advance_now" {
                                state.model.as_mut().unwrap().now +=
                                    step["nanos"].as_u64().unwrap();
                                lock(&outputs)
                                    .entry(id.clone())
                                    .or_default()
                                    .push(serde_json::json!({}));
                                continue;
                            }
                            if step_index % 3 == 1 {
                                mutate(&mut state, step_index);
                            }
                            if step_index % 2 == 0 {
                                decide(&mut state, &mut ordinal, &mut log);
                            }
                            if index == 0 && step_index == steps.len() / 2 {
                                interrupt(&state, variant);
                            }
                            let data = step_data(step, &outputs);
                            let op = if step["op"] == "idle" {
                                crate::abi::Operation::RuntimeIdle
                            } else {
                                crate::abi::Operation::RuntimeNetStream
                            };
                            let output = match state.broker_call(op, data) {
                                Ok(data) => {
                                    serde_json::json!({"data_base64": STANDARD.encode(&data)})
                                }
                                Err(error) => serde_json::json!({"error": error.to_string()}),
                            };
                            let error = output.get("error").is_some();
                            log.push(output.to_string());
                            lock(&outputs).entry(id.clone()).or_default().push(output);
                            if error {
                                failed = true;
                                break;
                            }
                        }
                        if !failed {
                            mutate(&mut state, steps.len());
                            decide(&mut state, &mut ordinal, &mut log);
                        }
                        terminate(&mut state, failed);
                        lock(&logs).insert(handle.node + "/1", log);
                        broker.finish(index);
                    });
                })
            })
            .collect();
        broker.start();
        for thread in threads {
            thread.join().unwrap();
        }
        // Second incarnations: fresh stores that idle on the shared clock.
        // Both restarts apply before either store runs: their controller
        // effects precede every second-incarnation event.
        let mut slots = Vec::new();
        for index in 0..ids.len() {
            broker
                .command(Command::Restart {
                    index,
                    incarnation: 2,
                })
                .unwrap();
            slots.push(lock(&broker.inner).sessions[index].slot);
        }
        let threads: Vec<_> = ids
            .iter()
            .enumerate()
            .map(|(index, id)| {
                let (broker, logs) = (broker.clone(), logs.clone());
                let id = id.clone();
                let endpoint = endpoint(&id, 2, slots[index]);
                std::thread::spawn(move || {
                    guarded(&broker.clone(), || {
                        inspection::set_eager_oracle(eager);
                        let handle = Handle {
                            node: id.clone(),
                            incarnation: 2,
                        };
                        let model = model(&handle, &broker.inspection);
                        let member = Member {
                            broker: broker.clone(),
                            incarnation: 2,
                            index,
                            network: endpoint,
                        };
                        let mut state = state(member, namespace(variant), model);
                        let mut log = Vec::new();
                        let mut ordinal = 100;
                        assert!(broker.acquire(index));
                        observe(&state);
                        for round in 0..4u64 {
                            mutate(&mut state, usize::try_from(round).unwrap() + index);
                            decide(&mut state, &mut ordinal, &mut log);
                            let mut data = vec![0u8; 16];
                            data[..4].copy_from_slice(&1u32.to_be_bytes());
                            data[8..]
                                .copy_from_slice(&((round + 1) * 10_000_000_000).to_be_bytes());
                            let output =
                                match state.broker_call(crate::abi::Operation::RuntimeIdle, data) {
                                    Ok(data) => STANDARD.encode(&data),
                                    Err(error) => format!("error {error}"),
                                };
                            log.push(output);
                            decide(&mut state, &mut ordinal, &mut log);
                        }
                        terminate(&mut state, false);
                        lock(&logs).insert(handle.node + "/2", log);
                        broker.finish(index);
                    });
                })
            })
            .collect();
        broker.start();
        for thread in threads {
            thread.join().unwrap();
        }
        assert!(broker.wait());
        let evidence = lock(&broker.inspection).bytes(true).unwrap();
        let logs = std::mem::take(&mut *lock(&logs));
        (evidence, logs)
    }

    fn scenarios() -> Vec<Value> {
        let mut files: Vec<_> = std::fs::read_dir(directory())
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| {
                path.extension()
                    .is_some_and(|extension| extension == "json")
            })
            .collect();
        files.sort();
        files
            .into_iter()
            .map(|path| serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap())
            .collect()
    }

    fn assert_identical(variant: Variant) {
        for scenario in scenarios() {
            let name = scenario["name"].as_str().unwrap().to_owned();
            let (eager, eager_logs) = run(&scenario, variant, true);
            let (incremental, incremental_logs) = run(&scenario, variant, false);
            assert_eq!(
                eager_logs, incremental_logs,
                "{name} {variant:?}: boundaries or calls"
            );
            if eager != incremental {
                let directory = std::env::temp_dir().join("gomad-inspection-differential");
                std::fs::create_dir_all(&directory).unwrap();
                std::fs::write(directory.join(format!("{name}.eager.json")), &eager).unwrap();
                std::fs::write(
                    directory.join(format!("{name}.incremental.json")),
                    &incremental,
                )
                .unwrap();
            }
            assert!(
                eager == incremental,
                "{name} {variant:?}: inspection evidence differs"
            );
            let evidence: Value = serde_json::from_slice(&incremental).unwrap();
            let boundaries = eager_logs
                .values()
                .flatten()
                .filter(|line| line.starts_with("boundary ") && !line.contains("error"))
                .count();
            let served = eager_logs
                .values()
                .flatten()
                .filter(|line| !line.starts_with("boundary") && !line.contains("error"))
                .count();
            assert!(served > 0, "{name}: no broker call was served");
            if variant.interrupt == Interrupt::None {
                assert_eq!(evidence["complete"], true, "{name}");
                assert!(boundaries > 0, "{name}");
            }
        }
    }

    #[test]
    fn incremental_recording_matches_eager_oracle_without_namespace() {
        assert_identical(Variant {
            namespace: false,
            storage: false,
            interrupt: Interrupt::None,
        });
    }

    #[test]
    fn incremental_recording_matches_eager_oracle_with_namespace() {
        assert_identical(Variant {
            namespace: true,
            storage: false,
            interrupt: Interrupt::None,
        });
    }

    #[test]
    fn incremental_recording_matches_eager_oracle_with_storage() {
        assert_identical(Variant {
            namespace: true,
            storage: true,
            interrupt: Interrupt::None,
        });
    }

    /// Each node of the first recorded scenario observes, records a boundary
    /// cut and a deferred network commitment, then idles and terminates. With
    /// `missed`, eager writes leave the commitment pending (a broken
    /// supersede): node 0's idle handoff and every node's finish must refuse
    /// with an infrastructure failure. Returns each node's (idle, terminal).
    /// A panicking node aborts the broker, so a regression fails instead of
    /// leaving a peer parked.
    #[expect(
        clippy::needless_collect,
        reason = "All worker threads must spawn before the broker starts and before any join"
    )]
    fn deferred_handoffs(missed: bool) -> Vec<(Result<(), Failure>, Result<(), String>)> {
        let scenario = scenarios().into_iter().next().unwrap();
        let ids: Vec<String> = scenario["nodes"]
            .as_array()
            .unwrap()
            .iter()
            .map(|id| id.as_str().unwrap().to_owned())
            .collect();
        let (config, profile) = network_config(&scenario).unwrap();
        let shared = Arc::new(Mutex::new(Shared::new(&config, &ids).unwrap()));
        let broker =
            Arc::new(Broker::new(&broker_config(), &ids, Some(shared.clone()), None).unwrap());
        let threads: Vec<_> = ids
            .iter()
            .enumerate()
            .map(|(index, id)| {
                let (broker, shared, profile, id) =
                    (broker.clone(), shared.clone(), profile.clone(), id.clone());
                std::thread::spawn(move || {
                    let mut result = None;
                    guarded(&broker.clone(), || {
                        let handle = Handle {
                            node: id.clone(),
                            incarnation: 1,
                        };
                        let endpoint = Endpoint::new(
                            shared,
                            &id,
                            1,
                            index as u64 + 1,
                            &profile,
                            946_684_800_000_000_000,
                            8 << 20,
                        )
                        .unwrap();
                        let member = Member {
                            broker: broker.clone(),
                            incarnation: 1,
                            index,
                            network: Some(endpoint),
                        };
                        let mut state = state(member, None, model(&handle, &broker.inspection));
                        assert!(broker.acquire(index));
                        observe(&state);
                        decide(&mut state, &mut 0, &mut Vec::new());
                        let member = state.broker.as_mut().unwrap();
                        member.inspection_network_snapshot(NetworkWrite::Deferred);
                        assert!(lock(&broker.inspection).deferred_network_pending(&handle));
                        inspection::set_missed_supersede(missed);
                        let mut data = vec![0u8; 16];
                        data[..4].copy_from_slice(&1u32.to_be_bytes());
                        data[8..].copy_from_slice(&10_000_000_000u64.to_be_bytes());
                        let mut now = state.model.as_ref().unwrap().now;
                        let mut advances = 0;
                        let idle = member.idle(
                            &mut data,
                            &mut Clockwork {
                                now: &mut now,
                                advances: &mut advances,
                                runtime_active: true,
                            },
                        );
                        let terminal = record_terminal(&mut state, false)
                            .map(|_| ())
                            .map_err(|(kind, message)| format!("{kind}: {message}"));
                        inspection::set_missed_supersede(false);
                        // Supersede the commitment left behind, so the next node's
                        // boundary does not observe it (a debug assertion).
                        lock(&broker.inspection).network_digest(&handle, [0; 32]);
                        broker.finish(index);
                        result = Some((idle, terminal));
                    });
                    result.unwrap()
                })
            })
            .collect();
        broker.start();
        threads
            .into_iter()
            .map(|thread| thread.join().unwrap())
            .collect()
    }

    #[test]
    fn eager_writes_supersede_deferred_network_commitments_at_handoffs() {
        for (idle, terminal) in deferred_handoffs(false) {
            assert!(idle.is_ok(), "{idle:?}");
            assert_eq!(terminal, Ok(()));
        }
    }

    #[test]
    fn handoffs_refuse_a_deferred_network_commitment_left_pending() {
        let results = deferred_handoffs(true);
        let (idle, _) = &results[0];
        assert!(
            matches!(idle, Err(Failure::Infrastructure(message))
                if message == "deferred network inspection commitment pending at idle handoff"),
            "{idle:?}"
        );
        for (_, terminal) in &results {
            assert_eq!(
                terminal,
                &Err(
                    "infrastructure: deferred network inspection commitment pending at instance finish"
                        .to_owned()
                )
            );
        }
    }

    #[test]
    fn incremental_recording_matches_eager_oracle_when_sealed_or_exhausted() {
        for interrupt in [Interrupt::Seal, Interrupt::Exhaust] {
            assert_identical(Variant {
                namespace: true,
                storage: true,
                interrupt,
            });
        }
    }
}

#[test]
fn finished_broker_world_uses_the_shared_binary_owner() {
    use sha2::Digest;
    let config = broker_config();
    let broker = Broker::new(&config, &["a".into()], None, None).unwrap();
    broker.finish(0);
    let wire = broker.world_wire().unwrap();
    let owner_config = world::Config {
        seed: config.seed,
        limits: world::Limits {
            max_requests: config.limits.max_requests,
            max_events: config.limits.max_events,
            max_queued_events: config.limits.max_queued_events,
            max_transitions: config.limits.max_transitions,
            max_payload_bytes: config.limits.max_payload_bytes,
            max_string_bytes: config.limits.max_string_bytes,
        },
    };
    world::retained::validate(&wire, &owner_config).unwrap();
    let envelope = world::retained::compress(&wire).unwrap();
    assert_eq!(world::retained::expand(&envelope).unwrap(), wire);
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        serde_json::json!({
            "schema":"gomad.core-predicate-observation/v1",
            "case":"finished_broker_binary_world_owner",
            "inputs":{"nodes":["a"],"seed":config.seed},
            "observed":{"expanded_sha256":format!("{:x}",sha2::Sha256::digest(&wire)),"compressed_sha256":format!("{:x}",sha2::Sha256::digest(&envelope))}
        })
    );
}

#[test]
fn native_world_configuration_requires_every_explicit_capacity() {
    let full = serde_json::json!({"seed":7,"limits":{"max_requests":100,"max_events":100,
        "max_queued_events":10,"max_transitions":1000,"max_payload_bytes":1 << 20,"max_string_bytes":1024}});
    let admitted: Config = serde_json::from_value(full.clone()).unwrap();
    admitted.validate().unwrap();
    for field in [
        "limits",
        "max_requests",
        "max_events",
        "max_queued_events",
        "max_transitions",
        "max_payload_bytes",
        "max_string_bytes",
    ] {
        let mut input = full.clone();
        if field == "limits" {
            input.as_object_mut().unwrap().remove(field);
        } else {
            input["limits"].as_object_mut().unwrap().remove(field);
        }
        let error = serde_json::from_value::<Config>(input.clone())
            .unwrap_err()
            .to_string();
        assert!(error.contains("missing field"), "{field}: {error}");
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"world.explicit-capacities","inputs":{"missing":field,"configuration":input},"observed":{"error":error}})
        );
    }
}

/// Original `TestTemporalStorageBrokerPayloadAccounting`: the request bound is
/// checked before the payload bound, each registration accounts its 32-byte
/// payload digest once per World copy (64 bytes), never the raw callback
/// input, and a refused registration leaves the World unchanged.
#[test]
fn registration_checks_requests_before_payload_and_refusal_keeps_world() {
    let payload = b"storage".repeat(128);
    for (name, payload_bytes, accepted, dimension) in [
        ("request-first", 192, 3, "requests"),
        ("payload-double-accounting", 191, 2, "payload-bytes"),
    ] {
        let mut config = broker_config();
        config.limits.max_requests = 3;
        config.limits.max_payload_bytes = payload_bytes;
        let broker = Broker::new(&config, &["a".into()], None, None).unwrap();
        for _ in 0..accepted {
            broker.register(0, "storage", &payload).unwrap();
            broker.ready(0, "storage").unwrap();
        }
        let before = lock(&broker.inner).world.cardinality();
        assert_eq!(
            (before.requests, before.events, before.payload_bytes),
            (accepted, accepted, accepted * 64),
            "{name}: broker payload accounting"
        );
        let refused = broker.register(0, "storage", &payload);
        assert!(
            matches!(refused, Err(Failure::Exhausted { dimension: actual, .. }) if actual == dimension),
            "{name}: broker boundary, want {dimension} capacity"
        );
        assert_eq!(
            lock(&broker.inner).world.cardinality(),
            before,
            "{name}: rejected request changed World"
        );
    }
}

/// Original `TestHelperPreservesInspectionFailuresWithoutFullTapeFinish`: an
/// inspection failure the broker raises keeps its own termination, stops the
/// run as an inspection stop and is not replaced by a later abort.
#[test]
fn inspection_failures_keep_their_termination_and_stop_inspection() {
    for termination in [
        "instrumentation_divergence",
        "debug_error",
        "probe_fuel_limit",
    ] {
        let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
        broker.abort(termination, "inspection".into());
        broker.abort("trap", "later".into());
        assert_eq!(broker.failure().map(|(kind, _)| kind), Some(termination));
        assert!(broker.inspection_stopped(), "{termination}");
    }
}
