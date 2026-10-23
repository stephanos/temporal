#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::linked_simulation::{Controller, Dimension, Plan};
use gomad_wasmhost::{linked, linked_cluster as cluster};
use serde_json::Value;
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::time::Duration;

fn prepared_cluster() -> cluster::Prepared {
    let module = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))".to_vec();
    prepared_cluster_with(&module, None)
}

fn prepared_cluster_with(
    module: &[u8],
    network: Option<cluster::NetworkConfig>,
) -> cluster::Prepared {
    prepared_cluster_fueled(module, network, 10000)
}

fn prepared_cluster_fueled(
    module: &[u8],
    network: Option<cluster::NetworkConfig>,
    fuel: u64,
) -> cluster::Prepared {
    let handles: Vec<_> = ["a", "b"]
        .into_iter()
        .map(|node| cluster::Handle {
            node: node.into(),
            incarnation: 1,
        })
        .collect();
    let incarnations = handles.iter().map(|handle| cluster::Incarnation {
        handle:handle.clone(), network:network.as_ref().map(|network| network.model.clone()),
        request:linked::Request {
            module:module.to_vec(),module_sha256:format!("{:x}", Sha256::digest(module)),
            limits:linked::Limits {fuel,memory_bytes:65536,output_bytes:1024,evidence_bytes:1<<20,wall_timeout:Duration::from_secs(5)},
            namespace:Some(linked::NamespaceImage {
                entries:Vec::new(),writable_directories:Vec::new(),preopens:vec![b"/".to_vec()],working_directory:b"/".to_vec(),stdin:Vec::new(),
                limits:linked::NamespaceLimits {files:16,descriptors:16,filesystem_bytes:4096,output_bytes:1024},epoch_nanos:0,
            }),
            storage:None,
            model:Some(linked::ModelInput {
                configuration:serde_json::from_value(json!({"entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":89,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,"limits":{"calls":if fuel > 10000 {1_000_000} else {100},"transcript_bytes":if fuel > 10000 {1 << 24} else {4096},"pending_events":16}})).unwrap(),
                arguments:Vec::new(),environment:vec!["PWD=/".into()],replay_trace:None,
            }),
        },
    }).collect();
    let steps = [cluster::Action::Start, cluster::Action::Wait]
        .into_iter()
        .flat_map(|action| {
            handles.iter().map(move |handle| cluster::Step {
                action,
                handle: handle.clone(),
            })
        })
        .collect();
    #[cfg_attr(not(feature = "determinism-fixture"), expect(unused_mut))]
    let mut prepared = cluster::prepare(cluster::Request {
        nodes: Vec::new(),
        broker: cluster::BrokerConfig {
            seed: 89,
            pause_on_settle: false,
            transition_bytes: None,
            limits: cluster::WorldLimits {
                max_requests: 100,
                max_events: 100,
                max_queued_events: 100,
                max_transitions: 1000,
                max_payload_bytes: 4096,
                max_string_bytes: 1024,
            },
        },
        network,
        incarnations,
        steps,
        boundaries: Vec::new(),
        crash_groups: Vec::new(),
        wall_timeout: Duration::from_secs(5),
        evidence_bytes: 1 << 20,
    })
    .unwrap();
    #[cfg(feature = "determinism-fixture")]
    prepared.conditions(linked::Conditions::DeterminismFixtureReplay);
    prepared
}

#[test]
fn linked_cluster_network_query_uses_the_broker_instead_of_standalone_fallback() {
    let network = serde_json::from_value(json!({
        "model":{"profile":"gomad.simulation-network-stream/v3","limits":{
            "listeners":8,"connections":8,"bytes":1_048_576,"connection_bytes":1_048_576,
            "stall_nanos":10_000_000_000_u64,"transitions":10000}},
        "addresses":{},"links":[],"strict_topology":false,"faults":[]
    }))
    .unwrap();
    for (network, expected) in [(Some(network), 0), (None, 2)] {
        let module = format!(r#"(module
            (import "gomad_wasm_v2" "net_stream" (func $network (param i32 i32) (result i32)))
            (memory (export "memory") 1)
            (data (i32.const 0) "\00\00\00\02")
            (func (export "_start")
                (if (call $network (i32.const 0) (i32.const 64)) (then unreachable))
                (if (i32.ne (i32.load8_u (i32.const 43)) (i32.const {expected})) (then unreachable))))"#).into_bytes();
        let observed = cluster::execute_prepared(prepared_cluster_with(&module, network)).unwrap();
        assert_eq!(observed.outcome.termination, "exit");
        assert!(
            observed
                .incarnations
                .iter()
                .all(|item| item.execution.outcome.exit_code == Some(0))
        );
    }
}

#[test]
fn scenario_forcing_runs_fresh_stores_and_retains_repeatable_actual_world() {
    use gomad_wasmhost::linked_simulation::{self as simulation, Override};
    let plan = |overrides| {
        Plan::new(
            format!("sha256:{}", "a".repeat(64)),
            format!("sha256:{}", "b".repeat(64)),
            89,
            overrides,
        )
        .unwrap()
    };
    let baseline = simulation::execute(prepared_cluster(), plan(Vec::new()), 16).unwrap();
    assert_eq!(baseline.cluster.outcome.termination, "exit");
    assert_eq!(baseline.cluster.incarnations.len(), 2);
    assert_ne!(baseline.cluster.world, [] as [u8; 0]);
    let observed = baseline
        .decisions
        .iter()
        .find(|decision| decision.dimension == Dimension::Scenario)
        .unwrap();
    let forced = Override::from_decision(observed, 1).unwrap();
    let selected = simulation::execute(prepared_cluster(), plan(vec![forced.clone()]), 16).unwrap();
    let repeated = simulation::execute(prepared_cluster(), plan(vec![forced]), 16).unwrap();
    assert_eq!(selected.cluster.outcome.termination, "exit");
    assert_eq!(selected.cluster.incarnations.len(), 2);
    assert_eq!(selected.decisions[0].selected, 1);
    assert_eq!(selected.cluster.world, repeated.cluster.world);
    assert_eq!(
        serde_json::to_vec(&selected.decisions).unwrap(),
        serde_json::to_vec(&repeated.decisions).unwrap()
    );
    assert!(selected.divergence.is_none());

    let mut changed_site = observed.clone();
    changed_site.site_sha256 = format!("sha256:{}", "c".repeat(64));
    let stale = Override::from_decision(&changed_site, 1).unwrap();
    let divergent = simulation::execute(prepared_cluster(), plan(vec![stale]), 16).unwrap();
    assert_eq!(
        divergent.cluster.outcome.termination,
        "instrumentation_divergence"
    );
    assert!(divergent.cluster.incarnations.is_empty());
    assert!(divergent.decisions.is_empty());
    assert!(divergent.divergence.unwrap().actual_exploration.is_some());
}

#[test]
fn go_issued_simulation_plans_force_the_observed_scenario_rank() {
    let vector: Value = serde_json::from_str(include_str!(
        "../../../internal/model/cluster/testdata/simulation_exploration_plan_vector.json"
    ))
    .unwrap();
    for (plan_key, selected) in [("root_plan", 0), ("forced_plan", 1)] {
        let raw = vector[plan_key].as_str().unwrap().as_bytes();
        let plan = Plan::decode(raw).unwrap();
        assert_eq!(plan.encode().unwrap(), raw);
        let mut controller = Controller::new(plan, 16).unwrap();
        let actual = controller
            .choose(
                Dimension::Scenario,
                vector["decision"]["site_sha256"].as_str().unwrap(),
                serde_json::from_value(vector["decision"]["alternatives"].clone()).unwrap(),
                0,
            )
            .unwrap();
        assert_eq!(actual, selected);
        assert_eq!(controller.decisions()[0].selected, selected);
        if selected == 0 {
            assert_eq!(
                serde_json::to_value(&controller.decisions()[0]).unwrap(),
                vector["decision"]
            );
        }
        controller.finish().unwrap();
    }
}

#[test]
fn unconsumed_or_changed_live_alternatives_preserve_typed_divergence() {
    let vector: Value = serde_json::from_str(include_str!(
        "../../../internal/model/cluster/testdata/simulation_exploration_plan_vector.json"
    ))
    .unwrap();
    let raw = vector["forced_plan"].as_str().unwrap().as_bytes();
    let mut missing = Controller::new(Plan::decode(raw).unwrap(), 16).unwrap();
    assert!(missing.finish().is_err());
    assert_eq!(missing.divergence().unwrap().ordinal, 0);
    assert!(missing.divergence().unwrap().actual_exploration.is_none());

    let mut changed = Controller::new(Plan::decode(raw).unwrap(), 16).unwrap();
    let mut alternatives: Vec<String> =
        serde_json::from_value(vector["decision"]["alternatives"].clone()).unwrap();
    alternatives.reverse();
    assert!(
        changed
            .choose(
                Dimension::Scenario,
                vector["decision"]["site_sha256"].as_str().unwrap(),
                alternatives,
                0,
            )
            .is_err()
    );
    let divergence = changed.divergence().unwrap();
    assert!(divergence.actual_exploration.is_some());
    assert!(changed.decisions().is_empty());
}

/// fn-227 R1 guard (received by fn-232.8): a watchdog-ended session with two
/// admitted live incarnations (one running guest code, one awaiting its turn)
/// drains them with repeatable outcome and network evidence across runs.
#[test]
fn live_incarnation_watchdog_shutdown_evidence_is_repeatable() {
    let network: cluster::NetworkConfig = serde_json::from_value(json!({
        "model":{"profile":"gomad.simulation-network-stream/v3","limits":{
            "listeners":8,"connections":8,"bytes":1_048_576,"connection_bytes":1_048_576,
            "stall_nanos":10_000_000_000_u64,"transitions":10000}},
        "addresses":{},"links":[],"strict_topology":false,"faults":[]
    }))
    .unwrap();
    let spin = br#"(module
        (import "wasi_snapshot_preview1" "sched_yield" (func $yield (result i32)))
        (memory (export "memory") 1)
        (func (export "_start") (drop (call $yield)) (drop (call $yield)) (loop (br 0))))"#;
    let mut first = None;
    for _ in 0..16 {
        let mut prepared = prepared_cluster_fueled(spin, Some(network.clone()), u64::MAX >> 2);
        prepared
            .bound_wall_timeout(Duration::from_millis(200))
            .unwrap();
        let observed = cluster::execute_prepared(prepared).unwrap();
        let view = (observed.outcome.termination, observed.network.clone());
        if let Some(first) = &first {
            assert_eq!(&view, first, "live-node shutdown evidence changed");
        } else {
            first = Some(view);
        }
    }
    let (termination, network) = first.unwrap();
    assert_eq!(termination, "watchdog_timeout");
    assert_ne!(network.len(), 0);
}

/// A live incarnation: it sleeps one virtual hour through a monotonic
/// `poll_oneoff` subscription, then exits.
const SLEEPER: &str = r#"(module
    (import "wasi_snapshot_preview1" "poll_oneoff" (func $poll (param i32 i32 i32 i32) (result i32)))
    (memory (export "memory") 1)
    (func (export "_start")
        (i64.store (i32.const 0) (i64.const 7))
        (i32.store8 (i32.const 8) (i32.const 0))
        (i32.store (i32.const 16) (i32.const 1))
        (i64.store (i32.const 24) (i64.const 3600000000000))
        (i64.store (i32.const 32) (i64.const 0))
        (i32.store16 (i32.const 40) (i32.const 0))
        (drop (call $poll (i32.const 0) (i32.const 64) (i32.const 1) (i32.const 128)))))"#;
const EXIT: &str = r#"(module (memory (export "memory") 1) (func (export "_start")))"#;

fn live_peer_cluster() -> cluster::Prepared {
    use cluster::Action::{Start, Wait};
    let nodes = [("a", SLEEPER), ("b", SLEEPER), ("c", EXIT), ("d", EXIT)];
    let incarnations = nodes.iter().map(|(node, wat)| {
        let module = wat.as_bytes().to_vec();
        cluster::Incarnation {
            handle: cluster::Handle { node: (*node).into(), incarnation: 1 },
            network: None,
            request: linked::Request {
                module_sha256: format!("{:x}", Sha256::digest(&module)),
                module,
                limits: linked::Limits { fuel: 100_000, memory_bytes: 65536, output_bytes: 1024, evidence_bytes: 1 << 20, wall_timeout: Duration::from_secs(5) },
                namespace: Some(linked::NamespaceImage {
                    entries: Vec::new(), writable_directories: Vec::new(), preopens: vec![b"/".to_vec()], working_directory: b"/".to_vec(), stdin: Vec::new(),
                    limits: linked::NamespaceLimits { files: 16, descriptors: 16, filesystem_bytes: 4096, output_bytes: 1024 }, epoch_nanos: 0,
                }),
                storage: None,
                model: Some(linked::ModelInput {
                    configuration: serde_json::from_value(json!({"entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":89,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}})).unwrap(),
                    arguments: Vec::new(), environment: vec!["PWD=/".into()], replay_trace: None,
                }),
            },
        }
    }).collect();
    let step = |action, node: &str| cluster::Step {
        action,
        handle: cluster::Handle {
            node: node.into(),
            incarnation: 1,
        },
    };
    let steps = vec![
        step(Start, "a"),
        step(Start, "b"),
        step(Start, "c"),
        step(Start, "d"),
        step(Wait, "c"),
        step(Wait, "d"),
        step(Wait, "a"),
        step(Wait, "b"),
    ];
    #[cfg_attr(not(feature = "determinism-fixture"), expect(unused_mut))]
    let mut prepared = cluster::prepare(cluster::Request {
        nodes: Vec::new(),
        broker: cluster::BrokerConfig {
            seed: 89,
            pause_on_settle: false,
            transition_bytes: None,
            limits: cluster::WorldLimits {
                max_requests: 100,
                max_events: 100,
                max_queued_events: 100,
                max_transitions: 1000,
                max_payload_bytes: 4096,
                max_string_bytes: 1024,
            },
        },
        network: None,
        incarnations,
        steps,
        boundaries: Vec::new(),
        crash_groups: Vec::new(),
        wall_timeout: Duration::from_secs(5),
        evidence_bytes: 1 << 20,
    })
    .unwrap();
    #[cfg(feature = "determinism-fixture")]
    prepared.conditions(linked::Conditions::DeterminismFixtureReplay);
    prepared
}

/// fn-227 R1 (received by fn-232.8): Go `ClusterSession.Shutdown` stops live
/// incarnations in (ID, Incarnation) order when a session ends early without a
/// cluster failure. A simulation divergence right after both sleepers started
/// must leave them with retained `stop` terminals and repeatable World evidence,
/// whichever order they started in, rather than an infrastructure abort.
#[test]
fn divergence_stops_live_incarnations_in_sorted_order() {
    use gomad_wasmhost::linked_simulation::{self as simulation, Override};
    let plan = |overrides| {
        Plan::new(
            format!("sha256:{}", "a".repeat(64)),
            format!("sha256:{}", "b".repeat(64)),
            89,
            overrides,
        )
        .unwrap()
    };
    let scenario = |overrides: &Vec<Override>| -> Vec<gomad_wasmhost::linked_simulation::Decision> {
        let observed =
            simulation::execute(live_peer_cluster(), plan(overrides.clone()), 16).unwrap();
        assert_eq!(observed.cluster.outcome.termination, "exit");
        observed
            .decisions
            .into_iter()
            .filter(|decision| decision.dimension == Dimension::Scenario)
            .collect()
    };
    for ranks in [[0_u32, 0], [1, 0]] {
        // Force the first two ready-start choices to the sleepers (a then b, or b then a).
        let mut overrides = Vec::new();
        for rank in ranks {
            let decisions = scenario(&overrides);
            overrides.push(Override::from_decision(&decisions[overrides.len()], rank).unwrap());
        }
        let decisions = scenario(&overrides);
        let mut stale = decisions[overrides.len()].clone();
        stale.site_sha256 = format!("sha256:{}", "c".repeat(64));
        overrides.push(Override::from_decision(&stale, 1).unwrap());
        let mut worlds = Vec::new();
        for _ in 0..4 {
            let divergent =
                simulation::execute(live_peer_cluster(), plan(overrides.clone()), 16).unwrap();
            #[cfg(feature = "determinism-fixture")]
            {
                // The recorded event order proves the stop terminals' sequence.
                let events =
                    String::from_utf8(divergent.cluster.event_order.clone().unwrap()).unwrap();
                let stops: Vec<String> = events
                    .lines()
                    .filter_map(|line| line.split_once(' ').map(|(_, json)| json))
                    .map(|json| serde_json::from_str::<Value>(json).unwrap())
                    .filter(|event| {
                        event["value"]["event"] == "terminal"
                            && event["value"]["termination"] == "stop"
                    })
                    .map(|event| event["instance"].as_str().unwrap().to_owned())
                    .collect();
                assert_eq!(stops, ["a", "b"], "ranks {ranks:?}: stop order");
            }
            assert_eq!(
                divergent.cluster.outcome.termination,
                "instrumentation_divergence"
            );
            assert!(divergent.divergence.is_some());
            let terminals: Vec<_> = divergent
                .cluster
                .incarnations
                .iter()
                .map(|item| (item.handle.node.clone(), item.execution.outcome.termination))
                .collect();
            assert_eq!(
                terminals,
                [("a".to_string(), "stop"), ("b".to_string(), "stop")],
                "ranks {ranks:?}: live peers were not stopped"
            );
            assert_ne!(divergent.cluster.world.len(), 0);
            worlds.push(divergent.cluster.world);
        }
        assert!(
            worlds.windows(2).all(|pair| pair[0] == pair[1]),
            "ranks {ranks:?}: World evidence changed across repetitions"
        );
    }
}
