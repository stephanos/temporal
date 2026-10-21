#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::linked_simulation::{Controller, Dimension, Plan};
use serde_json::Value;

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
            assert_eq!(serde_json::to_value(&controller.decisions()[0]).unwrap(), vector["decision"]);
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
    assert!(changed
        .choose(
            Dimension::Scenario,
            vector["decision"]["site_sha256"].as_str().unwrap(),
            alternatives,
            0,
        )
        .is_err());
    let divergence = changed.divergence().unwrap();
    assert!(divergence.actual_exploration.is_some());
    assert!(changed.decisions().is_empty());
}
