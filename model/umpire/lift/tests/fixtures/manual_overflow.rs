//! An overwritten next-state field still evaluates its overflowing increment.
use umpire_dsl::{Machine, Steps};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct State {
    n: u8,
}

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Action {
    Tick,
    Idle,
    Unused,
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Tick => "Action::Tick",
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
        vec![State { n: 255 }]
    }

    fn actions(&self) -> Vec<Action> {
        vec![Action::Tick, Action::Idle, Action::Unused]
    }

    fn internal(&self, _: &Action) -> bool {
        false
    }

    fn step(&self, state: &State, action: Action) -> Steps<State, (), ()> {
        let mut next = state.clone();
        match action {
            Action::Tick => {
                next.n += 1;
                next.n = 0;
            }
            _ => return Steps::new(),
        }
        let mut steps = Steps::new();
        steps.push(((), Vec::new(), next));
        steps
    }
}
