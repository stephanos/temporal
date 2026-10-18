//! Native parity and refusal checks for generated dispatch.

use prost::Message;
use umpire_wasm::{
    Oracle,
    protocol::{self, response::Outcome},
};

fn explore(oracle: &mut Oracle, budget: u32) -> protocol::Response {
    oracle.dispatch(1, &protocol::ExploreRequest { budget }.encode_to_vec())
}

fn rows(oracle: &mut Oracle, state_id: u32) -> protocol::Response {
    oracle.dispatch(2, &protocol::RowsRequest { state_id }.encode_to_vec())
}

fn refusal(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(value)) = response.outcome else { panic!("expected refusal") };
    assert_eq!(value.code, code as i32);
    assert_ne!(value.message, "");
}

#[test]
fn generated_dispatch_matches_trait_kettle_goldens() {
    let mut oracle = Oracle::default();
    let Some(Outcome::Explore(result)) = explore(&mut oracle, 1 << 20).outcome else {
        panic!("expected exploration");
    };
    assert_eq!(
        result.report,
        include_str!("../../tests/fixtures/kettle_trait/expected/report.txt")
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
        let Some(Outcome::Rows(result)) = rows(&mut oracle, state_id).outcome else {
            panic!("expected rows")
        };
        for row in result.rows {
            text.push_str(&row);
        }
    }
    assert_eq!(text, include_str!("../../tests/fixtures/kettle_trait/expected/rows.txt"));
}

#[test]
fn invalid_requests_have_typed_refusals() {
    let mut oracle = Oracle::default();
    refusal(rows(&mut oracle, 0), protocol::RefusalCode::NotExplored);
    refusal(oracle.dispatch(123, &[]), protocol::RefusalCode::UnknownMethod);
    for method in [1, 2] {
        refusal(oracle.dispatch(method, &[0xff]), protocol::RefusalCode::MalformedRequest);
    }
    refusal(explore(&mut oracle, 0), protocol::RefusalCode::Budget);
    explore(&mut oracle, 1 << 20);
    refusal(rows(&mut oracle, u32::MAX), protocol::RefusalCode::State);
}

#[test]
fn successful_explore_replaces_graph_and_refusal_preserves_it() {
    let mut oracle = Oracle::default();
    explore(&mut oracle, 1 << 20);
    refusal(explore(&mut oracle, 0), protocol::RefusalCode::Budget);
    assert!(matches!(rows(&mut oracle, 282).outcome, Some(Outcome::Rows(_))));
    let Some(Outcome::Explore(result)) = explore(&mut oracle, 1).outcome else {
        panic!("expected exploration")
    };
    assert_eq!(result.reached, 1);
    assert!(result.cut);
    refusal(rows(&mut oracle, 282), protocol::RefusalCode::State);
}
