use umpire_dsl::{Assumption, Machine, Progress, Steps};

#[derive(Clone, PartialEq, Eq, Hash)]
pub struct State(pub u8);

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("State { 0: ")?;
        write!(out, "{:?}", self.0)?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Clone, Copy, PartialEq, Eq, Hash)]
pub enum Action {
    Enter,
    Left,
    Right,
    Escape,
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Enter => "Action::Enter",
            Self::Left => "Action::Left",
            Self::Right => "Action::Right",
            Self::Escape => "Action::Escape",
        })?;
        Ok(())
    }
}

/// The progress graphs the attribute surface cannot state: nondeterministic choice and starts
/// outside a field range.
pub enum Mode {
    Choices,
    MultipleStarts,
}

pub struct Graph(pub Mode);

impl Machine for Graph {
    type State = State;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        match self.0 {
            Mode::MultipleStarts => vec![State(9), State(0)],
            Mode::Choices => vec![State(0)],
        }
    }

    fn actions(&self) -> Vec<Action> {
        vec![Action::Enter, Action::Left, Action::Right, Action::Escape]
    }

    fn internal(&self, action: &Action) -> bool {
        matches!(self.0, Mode::Choices) && *action == Action::Escape
    }

    fn step(&self, state: &State, action: Action) -> Steps<State, (), ()> {
        use Action::{Escape, Left};
        let edges: &[(u8, Action, u8)] = match self.0 {
            Mode::Choices => &[(0, Escape, 0), (0, Escape, 9)],
            Mode::MultipleStarts => &[(0, Left, 0)],
        };
        let mut steps = Steps::new();
        for &(from, edge_action, to) in edges {
            if state.0 == from && action == edge_action {
                steps.push(((), vec![], State(to)));
            }
        }
        steps
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        let claim = Progress::<Self>::new("reaches_target")
            .whenever(|_, _| true.into())
            .eventually(|_, state| (state.0 == 9).into());
        vec![match self.0 {
            Mode::Choices => claim.within(Assumption::fair(&[Action::Escape])),
            Mode::MultipleStarts => claim,
        }]
    }
}

umpire_testing::fixture!(Graph(Mode::Choices), 16);
