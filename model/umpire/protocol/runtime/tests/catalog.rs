//! Catalog admission refusals reach every runtime consumer with existing protocol codes.
use umpire_runtime::testing::ResponseExt;

mod support;

use prost::Message;
use support::any;

use umpire_dsl::{ActionCatalog, CatalogError, Machine, Steps};
use umpire_runtime::{
    AnswerPlan, Projection, Realize, Runtime,
    prost_types::Any,
    protocol::{self, response::Outcome},
};

#[path = "../src/refusal.rs"]
mod refusal_mapping;

/// The support world whose catalog admission refuses, as an over-limit declaration would.
struct Limited(CatalogError);

impl Machine for Limited {
    type State = support::State;
    type Action = support::Action;
    type Outcome = support::Outcome;
    type Fact = support::Fact;

    fn starts(&self) -> Vec<Self::State> {
        support::World { starts: 1 }.starts()
    }
    fn actions(&self) -> Vec<Self::Action> {
        support::World { starts: 1 }.actions()
    }
    fn checked_catalog(&self, _: usize) -> Result<ActionCatalog<Self::Action>, CatalogError> {
        Err(self.0)
    }
    fn internal(&self, action: &Self::Action) -> bool {
        support::World { starts: 1 }.internal(action)
    }
    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact> {
        support::World { starts: 1 }.step(state, action)
    }
}

impl Realize<Limited> for support::Realization {
    fn observe(
        &self,
        observation: &Any,
        state: &support::State,
    ) -> Result<Projection<support::Action, support::Outcome>, protocol::Refusal> {
        <Self as Realize<support::World>>::observe(self, observation, state)
    }
    fn answer(
        &self,
        callback: &Any,
        state: &support::State,
    ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
        <Self as Realize<support::World>>::answer(self, callback, state)
    }
}

#[test]
fn catalog_refusals_map_to_the_budget_code() {
    for (error, message) in [
        (
            umpire_checker::Refusal::CatalogLimit { count: 4097, limit: 4096 },
            "action catalog has 4097 actions, over limit 4096",
        ),
        (
            umpire_checker::Refusal::CatalogOverflow { limit: 4096 },
            "action catalog count overflows usize (limit 4096)",
        ),
    ] {
        let response =
            protocol::Response { outcome: Some(refusal_mapping::checker_refusal(&error)) };
        let decoded = protocol::Response::decode(response.encode_to_vec().as_slice())
            .unwrap_or_else(|error| panic!("generated response must decode: {error}"));
        assert_eq!(decoded.refusal(protocol::RefusalCode::Budget).message, message);
    }
}

#[test]
fn explore_judge_answer_and_replay_refuse_without_publishing() {
    let budget = protocol::RefusalCode::Budget;
    let error = CatalogError::TooLarge { count: 4097, limit: 4096 };
    let message = "action catalog has 4097 actions, over limit 4096".to_owned();
    let mut runtime = Runtime::new(Limited(error), support::Realization);
    let explore = runtime.dispatch(1, &protocol::ExploreRequest { budget: 8 }.encode_to_vec());
    assert_eq!(explore.refusal(budget).message, message);
    let rows = runtime.dispatch(2, &protocol::RowsRequest { state_id: 0 }.encode_to_vec());
    rows.refusal(protocol::RefusalCode::NotExplored);
    for value in [1, 10, 200] {
        let judged = runtime
            .dispatch(4, &protocol::JudgeRequest { observation: Some(any(value)) }.encode_to_vec());
        assert_eq!(judged.refusal(budget).message, message, "observation {value}");
    }
    let answered =
        runtime.dispatch(5, &protocol::AnswerRequest { callback: Some(any(1)) }.encode_to_vec());
    assert_eq!(answered.refusal(budget).message, message);
    let current =
        runtime.dispatch(4, &protocol::JudgeRequest { observation: Some(any(0)) }.encode_to_vec());
    let Some(Outcome::Judge(protocol::JudgeResponse {
        judgment: Some(protocol::judge_response::Judgment::Explained(explanation)),
        ..
    })) = current.outcome
    else {
        panic!("refusals must leave the run current and unfrozen: {current:?}")
    };
    assert_eq!(explanation.steps, [] as [String; 0]);
    let admitted =
        support::runtime().dispatch(1, &protocol::ExploreRequest { budget: 8 }.encode_to_vec());
    assert!(matches!(admitted.outcome, Some(Outcome::Explore(_))));
    let overflow =
        Runtime::new(Limited(CatalogError::Overflow { limit: 4096 }), support::Realization)
            .dispatch(1, &protocol::ExploreRequest { budget: 8 }.encode_to_vec());
    overflow.refusal(budget);
}
