//! Explicit initial-state selection retains the complete exploration and transactional run.

mod support;

use prost::Message;
use protocol::response::Outcome;
use umpire_dsl::{Machine, Steps};
use umpire_runtime::{AnswerPlan, Guest, Projection, Realize, Runtime, protocol};

#[derive(Clone, PartialEq, Eq, Hash)]
struct State {
    value: u8,
    marker: bool,
}

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{}:{}", self.value, self.marker)
    }
}

struct World;

impl Machine for World {
    type State = State;
    type Action = support::Action;
    type Outcome = support::Outcome;
    type Fact = support::Fact;

    fn starts(&self) -> Vec<State> {
        vec![State { value: 0, marker: false }, State { value: 1, marker: false }]
    }

    fn actions(&self) -> Vec<Self::Action> {
        support::World { starts: 1 }.actions()
    }

    fn internal(&self, _: &Self::Action) -> bool {
        false
    }

    fn step(&self, state: &State, action: Self::Action) -> Steps<State, Self::Outcome, Self::Fact> {
        support::World { starts: 1 }
            .step(&support::State(state.value), action)
            .into_iter()
            .map(|(outcome, facts, next)| {
                (outcome, facts, State { value: next.0, marker: state.marker })
            })
            .collect()
    }
}

impl Realize<World> for support::Realization {
    fn observe(
        &self,
        observation: &umpire_runtime::google::protobuf::Any,
        state: &State,
    ) -> Result<Projection<support::Action, support::Outcome>, protocol::Refusal> {
        <Self as Realize<support::World>>::observe(self, observation, &support::State(state.value))
    }

    fn answer(
        &self,
        _: &umpire_runtime::google::protobuf::Any,
        _: &State,
    ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
        Ok(AnswerPlan::new(
            vec![support::Action::Respond.into()],
            support::Action::Finish,
            support::any(42),
        ))
    }
}

fn selected() -> Runtime<World, support::Realization> {
    Runtime::with_initial(World, support::Realization, State { value: 1, marker: false })
        .unwrap_or_else(|refusal| panic!("valid selected start refused: {refusal:?}"))
}

fn observation(value: u8) -> Vec<u8> {
    protocol::JudgeRequest { observation: Some(support::any(value)) }.encode_to_vec()
}

fn publish(
    guest: &mut Guest<Runtime<World, support::Realization>>,
    method: u32,
    request: &[u8],
) -> Outcome {
    let mut result = None;
    assert!(guest.dispatch_to(method, request, |bytes| {
        result = protocol::Response::decode(bytes).ok().and_then(|response| response.outcome);
        true
    }));
    result.unwrap_or_else(|| panic!("missing response"))
}

fn explained(outcome: Outcome) -> Vec<String> {
    let Outcome::Judge(response) = outcome else { panic!("expected Judge") };
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = response.judgment else {
        panic!("expected explanation")
    };
    explanation.steps
}

fn unexplained(outcome: Outcome) -> Vec<String> {
    let Outcome::Judge(response) = outcome else { panic!("expected Judge") };
    let Some(protocol::judge_response::Judgment::Unexplained(evidence)) = response.judgment else {
        panic!("expected unexplained")
    };
    evidence.expected_actions
}

#[test]
fn selected_initial_requires_exact_membership_including_other_fields() {
    for state in [State { value: 2, marker: false }, State { value: 1, marker: true }] {
        let Err(refusal) = Runtime::with_initial(World, support::Realization, state) else {
            panic!("invalid initial state accepted")
        };
        assert_eq!(refusal.code, protocol::RefusalCode::State as i32);
    }
    let mut legacy = support::runtime();
    assert_eq!(
        explained(
            legacy.dispatch(4, &observation(0)).outcome.unwrap_or_else(|| panic!("missing Judge"))
        ),
        [] as [String; 0],
    );
    for starts in [0, 1] {
        let result = Runtime::with_initial(
            support::World { starts },
            support::Realization,
            support::State(0),
        );
        assert_eq!(result.is_ok(), starts == 1);
    }
}

#[test]
fn selected_reset_after_freeze_restores_initial_without_restricting_exploration() {
    let mut guest = Guest::new(selected());
    assert_eq!(explained(publish(&mut guest, 4, &observation(1))), [] as [String; 0]);
    assert_eq!(explained(publish(&mut guest, 4, &observation(2))), ["Respond"]);
    assert_eq!(unexplained(publish(&mut guest, 4, &observation(0))), ["Finish"]);
    assert!(matches!(publish(&mut guest, 3, &[]), Outcome::Reset(_)));
    assert_eq!(explained(publish(&mut guest, 4, &observation(1))), [] as [String; 0]);
    assert_eq!(explained(publish(&mut guest, 4, &observation(2))), ["Respond"]);
    let Outcome::Explore(exploration) =
        publish(&mut guest, 1, &protocol::ExploreRequest { budget: 16 }.encode_to_vec())
    else {
        panic!("expected Explore")
    };
    assert_eq!(exploration.starts, 2);
    let Outcome::Rows(rows) =
        publish(&mut guest, 2, &protocol::RowsRequest { state_id: 0 }.encode_to_vec())
    else {
        panic!("expected Rows")
    };
    assert!(rows.rows.iter().any(|row| row.contains("Deliver")));
}

#[test]
fn selected_publication_failures_preserve_advancement_freeze_and_reset() {
    for panics in [false, true] {
        let mut guest = Guest::new(selected());
        for (method, request) in [
            (4, observation(2)),
            (4, observation(0)),
            (5, protocol::AnswerRequest { callback: Some(support::any(4)) }.encode_to_vec()),
        ] {
            let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                guest.dispatch_to(method, &request, |bytes| {
                    let response = protocol::Response::decode(bytes)
                        .unwrap_or_else(|error| panic!("invalid response: {error}"));
                    assert!(!matches!(response.outcome, Some(Outcome::Refusal(_))));
                    assert!(!panics, "publication failed");
                    false
                })
            }));
            if panics {
                assert!(result.is_err());
            } else {
                assert!(!result.unwrap_or(true));
            }
            assert_eq!(explained(publish(&mut guest, 4, &observation(1))), [] as [String; 0]);
        }
        assert_eq!(explained(publish(&mut guest, 4, &observation(2))), ["Respond"]);
        assert_eq!(unexplained(publish(&mut guest, 4, &observation(0))), ["Finish"]);
        let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            guest.dispatch_to(3, &[], |_| {
                assert!(!panics, "publication failed");
                false
            })
        }));
        if panics {
            assert!(result.is_err());
        } else {
            assert!(!result.unwrap_or(true));
        }
        assert_eq!(unexplained(publish(&mut guest, 4, &observation(1))), ["Finish"]);
        assert!(matches!(publish(&mut guest, 3, &[]), Outcome::Reset(_)));
        assert_eq!(explained(publish(&mut guest, 4, &observation(1))), [] as [String; 0]);
    }
}
