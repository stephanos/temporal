//! Compiled manual source used to prove refusal preserves native behavior.
use umpire_dsl::{Machine, StepError, Steps};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct State {
    pub(crate) n: u8,
}
impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Action {
    Go,
    Idle,
    Unused,
}
impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Go => "Action::Go",
            Self::Idle => "Action::Idle",
            Self::Unused => "Action::Unused",
        })
    }
}

pub(crate) struct World;

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = ();
    type Fact = ();
    fn starts(&self) -> Vec<State> {
        vec![State { n: 0 }]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Go, Action::Idle, Action::Unused]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &State, _: Action) -> Steps<State, (), ()> {
        let mut steps = Steps::new();
        if state.n == 0 {
            steps.push(((), vec![], State { n: 3 }));
        }
        steps
    }
    fn r#checked_step(&self, _: &State, _: Action) -> Result<Steps<State, (), ()>, StepError> {
        Err(StepError::RejectedAfterMutation)
    }
}
