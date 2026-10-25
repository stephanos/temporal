//! Explicit initial-state selection retains the complete exploration and transactional run.
use umpire_runtime::protocol::{Method, RefusalCode};
use umpire_runtime::testing::{GuestExt, ResponseExt};

#[path = "../support/mod.rs"]
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

struct Selected;

impl umpire_runtime::capability::Identity for Selected {
    type Id = ();

    fn identify(&self, _: &prost_types::Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<World as Machine>::Action> for Selected {
    type World = World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<World> for Selected {
    type Target = umpire_runtime::NoTarget;

    fn project(
        &self,
        _world: &World,
        observation: &prost_types::Any,
        state: &State,
    ) -> Result<
        Projection<support::Action, support::Outcome, umpire_runtime::NoTarget>,
        protocol::Refusal,
    > {
        <support::Realization as Realize<support::World>>::observe(
            &support::Realization,
            &support::World { starts: 1 },
            observation,
            &support::State(state.value),
        )
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Selected {
    type World = World;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<World> for Selected {
    fn answer(
        &self,
        _world: &World,
        _: &prost_types::Any,
        _: &State,
    ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
        Ok(AnswerPlan::new(
            vec![support::Action::Respond.into()],
            support::Action::Finish,
            support::any(42),
        ))
    }
}

impl Realize<World> for Selected {}

fn selected() -> Runtime<World, Selected> {
    Runtime::with_initial(World, Selected, State { value: 1, marker: false })
        .unwrap_or_else(|refusal| panic!("valid selected start refused: {refusal:?}"))
}

fn observation(value: u8) -> Vec<u8> {
    protocol::JudgeRequest { observation: Some(support::any(value)) }.encode_to_vec()
}

#[test]
fn selected_initial_requires_exact_membership_including_other_fields() {
    for state in [State { value: 2, marker: false }, State { value: 1, marker: true }] {
        let Err(refusal) = Runtime::with_initial(World, Selected, state) else {
            panic!("invalid initial state accepted")
        };
        assert_eq!(refusal.code, RefusalCode::State as i32);
    }
    let mut legacy = support::runtime();
    assert_eq!(
        legacy.dispatch(Method::Judge as u32, &observation(0)).explanation().steps,
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
    assert_eq!(guest.judge(support::any(1)).explanation().steps, [] as [String; 0]);
    assert_eq!(guest.judge(support::any(2)).explanation().steps, ["Respond"]);
    assert_eq!(guest.judge(support::any(0)).unexplained(), ["Finish"]);
    assert!(matches!(guest.reset(None).outcome, Some(Outcome::Reset(_))));
    assert_eq!(guest.judge(support::any(1)).explanation().steps, [] as [String; 0]);
    assert_eq!(guest.judge(support::any(2)).explanation().steps, ["Respond"]);
    let Some(Outcome::Explore(exploration)) =
        GuestExt::call(&mut guest, Method::Explore, &protocol::ExploreRequest { budget: 16 })
            .outcome
    else {
        panic!("expected Explore")
    };
    assert_eq!(exploration.starts, 2);
    let Some(Outcome::Rows(rows)) =
        GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest { state_id: 0 }).outcome
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
            (Method::Judge as u32, observation(2)),
            (Method::Judge as u32, observation(0)),
            (
                Method::Answer as u32,
                protocol::AnswerRequest { callback: Some(support::any(4)) }.encode_to_vec(),
            ),
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
            assert_eq!(guest.judge(support::any(1)).explanation().steps, [] as [String; 0]);
        }
        assert_eq!(guest.judge(support::any(2)).explanation().steps, ["Respond"]);
        assert_eq!(guest.judge(support::any(0)).unexplained(), ["Finish"]);
        let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            guest.dispatch_to(Method::Reset as u32, &[], |_| {
                assert!(!panics, "publication failed");
                false
            })
        }));
        if panics {
            assert!(result.is_err());
        } else {
            assert!(!result.unwrap_or(true));
        }
        assert_eq!(guest.judge(support::any(1)).unexplained(), ["Finish"]);
        assert!(matches!(guest.reset(None).outcome, Some(Outcome::Reset(_))));
        assert_eq!(guest.judge(support::any(1)).explanation().steps, [] as [String; 0]);
    }
}
