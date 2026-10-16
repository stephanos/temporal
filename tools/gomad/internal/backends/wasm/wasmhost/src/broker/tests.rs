//! Replays the recorded Go broker orders
//! (`wasi/broker_test.go` `TestBrokerOrdersAreCurrent`) through the in-process
//! broker: scripted node call sequences on threads, as instance stores drive
//! it. Every call's output or error and the broker World's expanded wire must
//! equal the Go broker's; each node's network evidence is checked in for the
//! Go decoder (`TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments`).
use super::*;
use crate::network::cluster::{self as net, Endpoint, Fault, Link, ModelV2};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::Value;
use std::collections::BTreeMap;
use std::path::PathBuf;

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
    }
}

fn assert_world_commitment(broker: &Broker, network: Option<&[u8]>) {
    use crate::model::inspection::commitment;
    let mut wire = broker.world_wire().unwrap();
    if let Some(bytes) = network {
        wire.extend_from_slice(&(bytes.len() as u64).to_le_bytes());
        wire.extend_from_slice(bytes);
    }
    let key = network.map(|bytes| (bytes, commitment("network", bytes).try_into().unwrap()));
    let digest = broker.inspection_world_commitment(key).unwrap();
    assert_eq!(digest.as_slice(), commitment("world", &wire));
    {
        let inner = lock(&broker.inner);
        let (revision, cached) = inner.inspection_world_wire.as_ref().unwrap();
        assert_eq!(*revision, inner.world.next_ids().2);
        let mut uncached = Bounded {
            bytes: Vec::new(),
            limit: MAXIMUM_EXPANDED_WORLD_BYTES,
        };
        world::wire_generated::write_snapshot(&inner.world, &mut uncached).unwrap();
        assert_eq!(cached.as_ref(), uncached.bytes);
    }
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
fn network_only_inspection_changes_reuse_the_exact_world_wire() {
    use crate::model::inspection::commitment;
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    let world = broker.world_wire().unwrap();
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
        let mut literal = world.clone();
        if let Some(bytes) = network {
            literal.extend_from_slice(&(bytes.len() as u64).to_le_bytes());
            literal.extend_from_slice(bytes);
        }
        let key = network.map(|bytes| (bytes, commitment("network", bytes).try_into().unwrap()));
        assert_eq!(
            broker.inspection_world_commitment(key).unwrap().as_slice(),
            commitment("world", &literal)
        );
    }
    assert_eq!(
        broker
            .world_materializations
            .load(std::sync::atomic::Ordering::Relaxed),
        before + 1
    );
}

#[test]
fn oversized_world_discards_stale_wire_and_never_publishes_partial_serialization() {
    let mut config = broker_config();
    config.limits.max_payload_bytes = 1 << 30;
    let broker = Broker::new(&config, &["a".into()], None, None).unwrap();
    broker.inspection_world_commitment(None).unwrap();
    assert!(lock(&broker.inner).inspection_world_wire.is_some());
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
    assert!(inner.inspection_world_wire.is_none());
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

#[test]
fn inspection_abort_keeps_its_terminal_and_releases_dormant_and_held_members() {
    let broker = Broker::new(
        &broker_config(),
        &["a".into(), "b".into(), "c".into()],
        None,
        None,
    )
    .unwrap();
    broker.defer_starts();
    broker.activate(0).unwrap();
    broker.activate(1).unwrap();
    broker.run(0, 1).unwrap();
    broker.abort("inspected_prefix", "verified cut".into());
    assert!(!broker.acquire(1));
    assert!(!broker.acquire(2));
    assert!(!broker.wait());
    broker.abort("infrastructure", "neighbor aborted".into());
    assert_eq!(
        broker.failure(),
        Some(("inspected_prefix", "verified cut".into()))
    );
}

#[test]
fn manual_driver_owns_turn_until_target_finishes_and_reaps_revoked_peer() {
    let broker = Broker::new(&broker_config(), &["a".into(), "b".into()], None, None).unwrap();
    broker.defer_starts();
    broker.start();
    broker.activate(0).unwrap();
    broker.activate(1).unwrap();
    assert_eq!(lock(&broker.inner).active, None);
    broker
        .command(Command::Revoke {
            index: 0,
            incarnation: 1,
            kind: "stop",
        })
        .unwrap();
    assert_eq!(lock(&broker.inner).active, None);
    assert!(broker.run(0, 2).is_err());
    broker.run(0, 1).unwrap();
    assert!(broker.acquire(0));
    assert_eq!(broker.check(0, 1).unwrap_err().termination(), "stop");
    assert!(broker.run(1, 1).is_err());
    broker.finish(0);
    assert_eq!(lock(&broker.inner).active, None);
    assert!(lock(&broker.inner).sessions[0].hold.is_none());
    broker.run(1, 1).unwrap();
    assert!(broker.acquire(1));
    // A terminal target is idempotent even while another target owns the turn.
    broker.run(0, 1).unwrap();
    assert_eq!(lock(&broker.inner).active, Some(1));
    broker.finish(1);
    broker.restart(0, 2).unwrap();
    assert_eq!(lock(&broker.inner).active, None);
    broker.run(0, 2).unwrap();
    assert!(broker.acquire(0));
    broker.finish(0);
    assert!(broker.wait());
}

#[test]
fn manual_live_revoke_resumes_idle_target_under_its_driver() {
    let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
    broker.defer_starts();
    broker.activate(0).unwrap();
    broker.run(0, 1).unwrap();
    {
        let mut inner = lock(&broker.inner);
        inner.active = None;
        inner.sessions[0].state = State::Idle;
    }
    broker
        .command(Command::Revoke {
            index: 0,
            incarnation: 1,
            kind: "stop",
        })
        .unwrap();
    assert_eq!(lock(&broker.inner).active, Some(0));
    assert!(broker.acquire(0));
    broker.finish(0);
    assert!(broker.wait());
}

#[test]
fn output_revocation_is_cumulative_and_releases_hold_on_finish() {
    for kind in ["crash", "stop"] {
        let broker = Broker::new(&broker_config(), &["a".into(), "b".into()], None, None).unwrap();
        broker.boundary(0, 1, kind, b"committed".to_vec()).unwrap();
        broker.start();
        assert!(broker.acquire(0));
        broker.output(0, b"com").unwrap();
        let error = broker.output(0, b"mitted").unwrap_err();
        assert_eq!(error.termination(), kind);
        assert_eq!(broker.current(), 1);
        assert_eq!(broker.check(0, 1).unwrap_err().termination(), kind);
        broker.finish(0);
        assert!(broker.acquire(1));
        assert_eq!(broker.idle(1, 1000).unwrap(), 1000);
        broker.finish(1);
        assert!(broker.wait());
        assert!(broker.check(0, 1).is_err());
        let (incarnation, slot) = broker.restart(0, 2).unwrap();
        assert_eq!((incarnation, slot), (2, 3));
        assert!(broker.check(0, 1).is_err());
        assert!(broker.check(0, 2).is_ok());
        broker.finish(0);
    }
}

#[test]
fn output_barrier_revokes_idle_peer_before_another_effect() {
    let broker =
        Arc::new(Broker::new(&broker_config(), &["a".into(), "b".into()], None, None).unwrap());
    broker
        .barrier(&["a".into(), "b".into()], b"ready".to_vec())
        .unwrap();
    broker.start();
    let peer = broker.clone();
    let a = std::thread::spawn(move || {
        assert!(peer.acquire(0));
        peer.output(0, b"ready").unwrap();
        let error = peer.idle(0, 1000).unwrap_err();
        assert_eq!(error.termination(), "crash");
        peer.finish(0);
    });
    assert!(broker.acquire(1));
    assert_eq!(
        broker.output(1, b"ready").unwrap_err().termination(),
        "crash"
    );
    broker.finish(1);
    a.join().unwrap();
    assert!(broker.wait());
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
        let index = reference["step"].as_u64().unwrap() as usize;
        let data = STANDARD
            .decode(outputs[node][index]["data_base64"].as_str().unwrap())
            .unwrap();
        if width == 8 {
            u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
        } else {
            u64::from(u32::from_be_bytes(data[at..at + 4].try_into().unwrap()))
        }
    };
    let code = field("code") as u32;
    let count = field("count") as u32;
    let mut payload = step["payload"].as_str().unwrap_or("").as_bytes().to_vec();
    if code == 4 || code == 12 {
        payload = vec![0; count as usize];
    }
    let mut data = vec![0u8; 64];
    data[..4].copy_from_slice(&2u32.to_be_bytes());
    data[4..8].copy_from_slice(&code.to_be_bytes());
    data[8..16].copy_from_slice(&field("id").to_be_bytes());
    data[16..24].copy_from_slice(&resolve("handle", 16, 8).to_be_bytes());
    data[32..36].copy_from_slice(&(resolve("port", 48, 4) as u32).to_be_bytes());
    let header_count = if code == 12 { 0 } else { count };
    data[36..40].copy_from_slice(&header_count.to_be_bytes());
    for (index, byte) in step["address"].as_array().unwrap().iter().enumerate() {
        data[56 + index] = byte.as_u64().unwrap() as u8;
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
                            record(serde_json::json!({"data_base64": STANDARD.encode(&data)}))
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
fn restart_cannot_mutate_network_during_an_active_incarnation() {
    let ids = vec!["a".into(), "b".into()];
    let (config, _) = network_config(&serde_json::json!({
        "network": {"profile": net::PROFILE, "limits": {"listeners":8,"connections":8,"bytes":1024,
            "connection_bytes":1024,"stall_nanos":1000,"transitions":64}}
    }))
    .unwrap();
    let shared = Arc::new(Mutex::new(Shared::new(&config, &ids).unwrap()));
    let broker = Arc::new(Broker::new(&broker_config(), &ids, Some(shared.clone()), None).unwrap());
    broker.start();
    assert!(broker.acquire(0));
    broker.finish(0);
    assert!(broker.acquire(1));
    broker.finish(1);
    broker.restart(0, 2).unwrap();
    assert!(broker.acquire(0));
    let peer = broker.clone();
    let (entered, waiting) = mpsc::sync_channel(1);
    let (completed, outcome) = mpsc::sync_channel(1);
    let restart = std::thread::spawn(move || {
        entered.send(()).unwrap();
        completed.send(peer.restart(1, 2)).unwrap();
    });
    waiting.recv().unwrap();
    let early = outcome
        .recv_timeout(std::time::Duration::from_millis(100))
        .ok();
    broker.check(0, 2).unwrap();
    broker.start();
    let snapshot = serde_json::to_value(lock(&shared).record().unwrap()).unwrap();
    broker.finish(0);
    let result = early.clone().unwrap_or_else(|| outcome.recv().unwrap());
    restart.join().unwrap();
    assert!(broker.acquire(1));
    broker.finish(1);
    assert!(
        early.is_none(),
        "restart crossed an active store's evidence flush"
    );
    assert_eq!(result.unwrap(), (2, 4));
    assert!(
        snapshot["snapshot"]["nodes"]
            .as_array()
            .unwrap()
            .iter()
            .filter(|node| node["node"].as_str().unwrap().starts_with("b."))
            .all(|node| node["last_incarnation"] == 1)
    );
}

#[test]
fn commands_keep_live_revocation_and_link_polling_during_active_execution() {
    for kind in ["crash", "stop"] {
        let broker = Broker::new(&broker_config(), &["a".into()], None, None).unwrap();
        broker.start();
        assert!(broker.acquire(0));
        let (send, reply) = mpsc::sync_channel(1);
        lock(&broker.inner).commands.push_back((
            Command::Revoke {
                index: 0,
                incarnation: 1,
                kind,
            },
            send,
        ));
        assert_eq!(broker.check(0, 1).unwrap_err().termination(), kind);
        reply.recv().unwrap().unwrap();
        broker.finish(0);
    }
    let ids = vec!["a".into(), "b".into()];
    let (config, _) = network_config(&serde_json::json!({
        "network": {"profile": net::PROFILE, "limits": {"listeners":8,"connections":8,"bytes":1024,
            "connection_bytes":1024,"stall_nanos":1000,"transitions":64}}
    }))
    .unwrap();
    let shared = Arc::new(Mutex::new(Shared::new(&config, &ids).unwrap()));
    let broker = Broker::new(&broker_config(), &ids, Some(shared.clone()), None).unwrap();
    broker.start();
    assert!(broker.acquire(0));
    let (send, reply) = mpsc::sync_channel(1);
    lock(&broker.inner).commands.push_back((
        Command::Link {
            action: "partition".into(),
            from: "a".into(),
            to: "b".into(),
            delay: 0,
            symmetric: true,
        },
        send,
    ));
    broker.check(0, 1).unwrap();
    reply.recv().unwrap().unwrap();
    assert!(
        lock(&shared)
            .record()
            .unwrap()
            .snapshot
            .links
            .unwrap()
            .iter()
            .filter(|link| link.from[..1] != link.to[..1])
            .all(|link| !link.enabled)
    );
}

#[test]
fn queued_restart_applies_when_active_store_idles() {
    let broker =
        Arc::new(Broker::new(&broker_config(), &["a".into(), "b".into()], None, None).unwrap());
    broker.start();
    assert!(broker.acquire(0));
    broker.finish(1);
    let (send, reply) = mpsc::sync_channel(1);
    lock(&broker.inner).commands.push_back((
        Command::Restart {
            index: 1,
            incarnation: 2,
        },
        send,
    ));
    let peer = broker.clone();
    let store = std::thread::spawn(move || {
        assert!(peer.acquire(1));
        assert!(peer.check(1, 2).is_ok());
        peer.finish(1);
    });
    assert_eq!(broker.idle(0, 1000).unwrap(), 1000);
    reply.recv().unwrap().unwrap();
    broker.finish(0);
    store.join().unwrap();
    assert!(broker.wait());
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
                let want = if want.as_object().is_some_and(|object| object.is_empty()) {
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
        for (node, bytes) in evidence {
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
fn broker_pauses_at_clock_advances_for_live_link_changes() {
    use std::io::Cursor;
    let ids = vec!["a".to_owned(), "b".to_owned()];
    let (config, _) = network_config(&serde_json::json!({
        "network": {"profile": net::PROFILE, "limits": {"listeners":8,"connections":8,"bytes":1024,
            "connection_bytes":1024,"stall_nanos":1000,"transitions":64}}
    }))
    .unwrap();
    let shared = Arc::new(Mutex::new(Shared::new(&config, &ids).unwrap()));
    let control = b"{\"type\":\"control\",\"command\":\"link\",\"action\":\"partition\",\"from\":\"a\",\"to\":\"b\",\"symmetric\":true}\n{\"type\":\"resume\"}\n{\"type\":\"resume\"}\n".to_vec();
    let input: SharedInput = Arc::new(Mutex::new(Box::new(Cursor::new(control))));
    #[derive(Clone, Default)]
    struct Capture(Arc<Mutex<Vec<u8>>>);
    impl std::io::Write for Capture {
        fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
            lock(&self.0).extend_from_slice(data);
            Ok(data.len())
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }
    let capture = Capture::default();
    let output: SharedOutput = Arc::new(Mutex::new(Box::new(capture.clone())));
    let mut paused = broker_config();
    paused.pause_on_settle = true;
    let broker =
        Arc::new(Broker::new(&paused, &ids, Some(shared.clone()), Some((input, output))).unwrap());
    let threads: Vec<_> = (0..2)
        .map(|index| {
            let broker = broker.clone();
            std::thread::spawn(move || {
                let mut member = Member {
                    broker: broker.clone(),
                    incarnation: 1,
                    index,
                    network: None,
                };
                assert!(broker.acquire(index));
                let (mut now, mut advances) = (1, 0);
                for deadline in [100u64, 200] {
                    let mut data = vec![0u8; 16];
                    data[..4].copy_from_slice(&1u32.to_be_bytes());
                    data[8..].copy_from_slice(&deadline.to_be_bytes());
                    let mut clock = Clockwork {
                        now: &mut now,
                        advances: &mut advances,
                        runtime_active: true,
                    };
                    member.idle(&mut data, &mut clock).unwrap();
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
    assert_eq!(
        String::from_utf8(lock(&capture.0).clone()).unwrap(),
        "{\"at\":100,\"reason\":\"settle\",\"type\":\"pause\"}\n{\"at\":200,\"reason\":\"settle\",\"type\":\"pause\"}\n"
    );
    let links = lock(&shared).record().unwrap().snapshot.links.unwrap();
    assert!(
        links
            .iter()
            .filter(|link| link.from[..1] != link.to[..1])
            .all(|link| !link.enabled),
        "the live partition applied at the pause"
    );
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
