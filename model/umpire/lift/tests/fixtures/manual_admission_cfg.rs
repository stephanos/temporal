//! Compiled manual source used to prove refusal preserves native behavior.
use umpire_dsl::{Machine, Steps};

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

#[derive(Clone, Copy)]
pub(crate) enum Mode {
    Chosen,
    Other,
}
pub(crate) struct World(pub(crate) Mode);

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = ();
    type Fact = ();
    fn starts(&self) -> Vec<State> {
        match self.0 {
            #[r#cfg(any())]
            Mode::Chosen => vec![State { n: 9 }],
            Mode::Other => vec![State { n: 2 }],
            Mode::Chosen => vec![State { n: 0 }],
        }
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
            #[r#cfg_attr(all(), cfg(any()))]
            steps.push(((), vec![], State { n: 1 }));
            #[r#cfg(any())]
            steps.push(((), vec![], State { n: 2 }));
            steps.push(((), vec![], State { n: 3 }));
        }
        steps
    }
}

#[cfg(any())]
umpire_testing::fixture!(World(Mode::Chosen), 10);
