//! Native parity and refusal checks for generated dispatch.

#[path = "../../../dev/testing/src/snapshot_payload.rs"]
#[allow(unreachable_pub, reason = "the shared reader is public when reexported by umpire-testing")]
mod snapshot_payload;

use prost::Message;
use umpire_runtime::{
    Model,
    protocol::{self, response::Outcome},
};

#[path = "../src/refusal.rs"]
mod refusal_mapping;

#[test]
fn progress_query_refusals_survive_generated_protocol() {
    for (error, message) in [
        (
            umpire_checker::Refusal::ProgressRegistrationMissing { claim: "missing".into() },
            "unknown registered progress claim: missing",
        ),
        (
            umpire_checker::Refusal::ProgressRegistrationAmbiguous { claim: "duplicate".into() },
            "ambiguous registered progress claim: duplicate",
        ),
        (
            umpire_checker::Refusal::ProgressCatalogChanged,
            "progress query action catalog differs from exploration",
        ),
    ] {
        let response =
            protocol::Response { outcome: Some(refusal_mapping::checker_refusal(&error)) };
        let decoded = protocol::Response::decode(response.encode_to_vec().as_slice())
            .unwrap_or_else(|error| panic!("generated response must decode: {error}"));
        let Some(Outcome::Refusal(refusal)) = decoded.outcome else {
            panic!("expected progress query refusal")
        };
        assert_eq!(refusal.code, 2);
        assert_eq!(refusal.code(), protocol::RefusalCode::MalformedRequest);
        assert_eq!(refusal.message, message);
    }
}

#[test]
fn progress_bound_refusal_survives_generated_protocol() {
    let error = umpire_checker::Refusal::ProgressBound { claim: "pending".into(), given: 0 };
    let response = protocol::Response { outcome: Some(refusal_mapping::checker_refusal(&error)) };
    let decoded = protocol::Response::decode(response.encode_to_vec().as_slice())
        .unwrap_or_else(|error| panic!("generated response must decode: {error}"));
    let Some(Outcome::Refusal(refusal)) = decoded.outcome else {
        panic!("expected progress bound refusal")
    };
    assert_eq!(refusal.code, 2);
    assert_eq!(refusal.code(), protocol::RefusalCode::MalformedRequest);
    assert_eq!(refusal.message, "progress claim pending bound must be at least one (given: 0)");
}

#[test]
fn scope_action_refusal_survives_generated_protocol() {
    let error = umpire_checker::Refusal::ScopeAction { action: "User::Missing".into() };
    let response = protocol::Response { outcome: Some(refusal_mapping::checker_refusal(&error)) };
    let decoded = protocol::Response::decode(response.encode_to_vec().as_slice())
        .unwrap_or_else(|error| panic!("generated response must decode: {error}"));
    let Some(Outcome::Refusal(refusal)) = decoded.outcome else {
        panic!("expected scope action refusal")
    };
    assert_eq!(refusal.code, 7);
    assert_eq!(refusal.code(), protocol::RefusalCode::ScopeAction);
    assert_eq!(refusal.message, "unknown scope action: User::Missing");
}

#[test]
fn empty_assumption_refusal_survives_generated_protocol() {
    let response = protocol::Response {
        outcome: Some(refusal_mapping::checker_refusal(&umpire_checker::Refusal::EmptyAssumption)),
    };
    let decoded = protocol::Response::decode(response.encode_to_vec().as_slice())
        .unwrap_or_else(|error| panic!("generated response must decode: {error}"));
    let Some(Outcome::Refusal(refusal)) = decoded.outcome else {
        panic!("expected empty assumption refusal")
    };
    assert_eq!(refusal.code, 7);
    assert_eq!(refusal.code(), protocol::RefusalCode::ScopeAction);
    assert_eq!(refusal.message, "assumption admits no actions");
}

fn explore(model: &mut Model, budget: u32) -> protocol::Response {
    model.dispatch(1, &protocol::ExploreRequest { budget }.encode_to_vec())
}

fn rows(model: &mut Model, state_id: u32) -> protocol::Response {
    model.dispatch(2, &protocol::RowsRequest { state_id }.encode_to_vec())
}

fn refusal(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(value)) = response.outcome else { panic!("expected refusal") };
    assert_eq!(value.code, code as i32);
    assert_ne!(value.message, "");
}

#[test]
fn generated_dispatch_matches_trait_kettle_goldens() {
    let mut model = Model::default();
    let Some(Outcome::Explore(result)) = explore(&mut model, 1 << 20).outcome else {
        panic!("expected exploration");
    };
    assert_eq!(
        result.report,
        snapshot_payload::snapshot_payload(include_str!(
            "../../../backends/native/tests/fixtures/sample_kettle_trait/expected/report.txt.snap"
        ))
        .unwrap_or_else(|error| panic!("{error}"))
    );
    assert_eq!(
        (
            result.starts,
            result.reached,
            result.emitted,
            result.evaluated,
            result.refused,
            result.depth,
            result.cut
        ),
        (81, 283, 424, 1132, 708, 62, false)
    );
    let mut text = String::new();
    for state_id in 0..result.reached {
        let Some(Outcome::Rows(result)) = rows(&mut model, state_id).outcome else {
            panic!("expected rows")
        };
        for row in result.rows {
            text.push_str(&row);
        }
    }
    assert_eq!(
        text,
        snapshot_payload::snapshot_payload(include_str!(
            "../../../backends/native/tests/fixtures/sample_kettle_trait/expected/rows.txt.snap"
        ))
        .unwrap_or_else(|error| panic!("{error}"))
    );
}

#[test]
fn invalid_requests_have_typed_refusals() {
    let mut model = Model::default();
    refusal(rows(&mut model, 0), protocol::RefusalCode::NotExplored);
    refusal(model.dispatch(123, &[]), protocol::RefusalCode::UnknownMethod);
    for method in [1, 2] {
        refusal(model.dispatch(method, &[0xff]), protocol::RefusalCode::MalformedRequest);
    }
    refusal(explore(&mut model, 0), protocol::RefusalCode::Budget);
    explore(&mut model, 1 << 20);
    refusal(rows(&mut model, u32::MAX), protocol::RefusalCode::State);
}

#[test]
fn successful_explore_replaces_graph_and_refusal_preserves_it() {
    let mut model = Model::default();
    explore(&mut model, 1 << 20);
    refusal(explore(&mut model, 0), protocol::RefusalCode::Budget);
    assert!(matches!(rows(&mut model, 282).outcome, Some(Outcome::Rows(_))));
    let Some(Outcome::Explore(result)) = explore(&mut model, 1).outcome else {
        panic!("expected exploration")
    };
    assert_eq!(result.reached, 1);
    assert!(result.cut);
    refusal(rows(&mut model, 282), protocol::RefusalCode::State);
}
