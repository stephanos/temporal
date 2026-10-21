#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::linked_simulation::{Controller, Dimension, Plan};
use gomad_wasmhost::{linked, linked_cluster as cluster};
use serde_json::Value;
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::time::Duration;

fn prepared_cluster() -> cluster::Prepared {
    let module = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))".to_vec();
    prepared_cluster_with(module, None)
}

fn prepared_cluster_with(
    module: Vec<u8>,
    network: Option<cluster::NetworkConfig>,
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
            module:module.clone(),module_sha256:format!("{:x}", Sha256::digest(&module)),
            limits:linked::Limits {fuel:10000,memory_bytes:65536,output_bytes:1024,evidence_bytes:1<<20,wall_timeout:Duration::from_secs(5)},
            namespace:Some(linked::NamespaceImage {
                entries:Vec::new(),writable_directories:Vec::new(),preopens:vec![b"/".to_vec()],working_directory:b"/".to_vec(),stdin:Vec::new(),
                limits:linked::NamespaceLimits {files:16,descriptors:16,filesystem_bytes:4096,output_bytes:1024},epoch_nanos:0,
            }),
            storage:None,
            model:Some(linked::ModelInput {
                configuration:serde_json::from_value(json!({"entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":89,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}})).unwrap(),
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
    cluster::prepare(cluster::Request {
        nodes: Vec::new(),
        broker: cluster::BrokerConfig {
            seed: 89,
            pause_on_settle: false,
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
    .unwrap()
}

#[test]
fn linked_cluster_network_query_uses_the_broker_instead_of_standalone_fallback() {
    let module = br#"(module
        (import "gomad_wasm_v2" "net_stream" (func $network (param i32 i32) (result i32)))
        (memory (export "memory") 1)
        (data (i32.const 0) "\00\00\00\02")
        (func (export "_start")
            (if (call $network (i32.const 0) (i32.const 64)) (then unreachable))
            (if (i32.load (i32.const 40)) (then unreachable))))"#
        .to_vec();
    let network = serde_json::from_value(json!({
        "model":{"profile":"gomad.simulation-network-stream/v3","limits":{
            "listeners":8,"connections":8,"bytes":1048576,"connection_bytes":1048576,
            "stall_nanos":10000000000_u64,"transitions":10000}},
        "addresses":{},"links":[],"strict_topology":false,"faults":[]
    }))
    .unwrap();
    let observed = cluster::execute_prepared(prepared_cluster_with(module, Some(network))).unwrap();
    assert_eq!(observed.outcome.termination, "exit");
    assert!(
        observed
            .incarnations
            .iter()
            .all(|item| item.execution.outcome.exit_code == Some(0))
    );
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
    assert!(!baseline.cluster.world.is_empty());
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
        "../../../../runner/testdata/simulation_exploration_plan_vector.json"
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
        "../../../../runner/testdata/simulation_exploration_plan_vector.json"
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
