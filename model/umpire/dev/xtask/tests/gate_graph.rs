//! Source contracts for the single shipping stage graph; execution proofs remain separate.
#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::collections::{BTreeMap, BTreeSet};

fn tasks() -> BTreeMap<String, toml::Value> {
    let config: toml::Value = toml::from_str(include_str!("../src/gate/tasks.toml")).unwrap();
    config
        .as_table()
        .unwrap()
        .iter()
        .filter(|(name, _)| name.starts_with("gate:"))
        .map(|(name, task)| (name.clone(), task.clone()))
        .collect()
}

fn predecessors(tasks: &BTreeMap<String, toml::Value>, name: &str) -> BTreeSet<String> {
    let mut result = BTreeSet::new();
    let mut pending = vec![name.to_owned()];
    while let Some(name) = pending.pop() {
        let task = tasks.get(&name).unwrap_or_else(|| panic!("missing graph task {name}"));
        if let Some(dependencies) = task.get("depends") {
            for dependency in dependencies.as_array().unwrap() {
                let dependency = dependency.as_str().unwrap();
                assert!(dependency.starts_with("gate:"), "external graph edge {dependency}");
                if result.insert(dependency.to_owned()) {
                    pending.push(dependency.to_owned());
                }
            }
        }
    }
    result
}

#[test]
fn shipping_graph_has_unique_complete_stage_metadata_and_no_cycle() {
    let tasks = tasks();
    let mut stages = BTreeSet::new();
    let mut orders = BTreeSet::new();
    for (name, task) in &tasks {
        assert!(!predecessors(&tasks, name).contains(name), "cycle at {name}");
        if let Some(env) = task.get("env") {
            if let Some(stage) = env.get("UMPIRE_GATE_STAGE") {
                let stage = stage.as_str().unwrap();
                assert_eq!(name, &format!("gate:{stage}"));
                assert!(stages.insert(stage), "duplicate stage {stage}");
                let order = env["UMPIRE_GATE_ORDER"].as_str().unwrap().parse::<usize>().unwrap();
                assert!(orders.insert(order), "duplicate stage order {order}");
                let run = task["run"].as_str().unwrap();
                assert!(run.contains("UMPIRE_GATE_EXECUTABLE"), "{name}: {run}");
                assert!(run.contains(&format!("gate --scheduled-stage {stage}")), "{name}: {run}");
            }
        }
    }
    assert_eq!(stages, super::gate_wiring::STAGES.iter().copied().collect());
    assert_eq!(orders, (0..stages.len()).collect());
    let child = predecessors(&tasks, "gate:child");
    for stage in stages.iter().copied().filter(|stage| *stage != "standalone") {
        assert!(child.contains(&format!("gate:{stage}")), "child omitted {stage}");
    }
    assert!(!child.contains("gate:standalone"), "child recursion");
    assert!(predecessors(&tasks, "gate:all").contains("gate:standalone"));
}

#[test]
fn shared_cargo_targets_and_generated_outputs_have_predecessor_barriers() {
    let tasks = tasks();
    let shared_target = [
        "clippy",
        "docs",
        "tests",
        "wasm32-wasip1",
        "core-wasm-clippy",
        "wasm-guest",
        "go-host",
        "kettle-go-host",
    ];
    for (index, left) in shared_target.iter().enumerate() {
        for right in &shared_target[index + 1..] {
            let left = format!("gate:{left}");
            let right = format!("gate:{right}");
            assert!(
                predecessors(&tasks, &left).contains(&right)
                    || predecessors(&tasks, &right).contains(&left),
                "shared-target stages may overlap: {left}, {right}"
            );
        }
    }
    for (consumer, producer) in [
        ("protobuf-controls", "proto-tools"),
        ("protobuf-freshness", "proto-tools"),
        ("kettle-protobuf-freshness", "proto-tools"),
        ("clippy", "kettle-protobuf-freshness"),
        ("tests", "kettle-protobuf-freshness"),
        ("go-host", "wasm-guest"),
        ("kettle-go-host", "wasm-guest"),
        ("kettle-go-host", "kettle-protobuf-freshness"),
    ] {
        assert!(
            predecessors(&tasks, &format!("gate:{consumer}")).contains(&format!("gate:{producer}")),
            "{consumer} can precede {producer}"
        );
    }
}

#[test]
fn graph_keeps_independent_read_only_work_schedulable_together() {
    let tasks = tasks();
    for (left, right) in [("fmt", "proto-tools"), ("retired-names", "fmt")] {
        let left = format!("gate:{left}");
        let right = format!("gate:{right}");
        assert!(!predecessors(&tasks, &left).contains(&right));
        assert!(!predecessors(&tasks, &right).contains(&left));
    }
}
