use umpire_dsl::{Assumption, AssumptionExt, Machine, Progress, Scope, Steps};

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

/// The progress graphs the attribute surface cannot state: self-loops, nondeterministic choice
/// and starts outside a field range.
pub enum Mode {
    ExcludedEscape,
    Choices,
    Prefix,
    MultipleStarts,
    TargetBarrier,
    SelfLoop,
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
            _ => vec![State(0)],
        }
    }

    fn actions(&self) -> Vec<Action> {
        vec![Action::Enter, Action::Left, Action::Right, Action::Escape]
    }

    fn internal(&self, action: &Action) -> bool {
        match self.0 {
            Mode::ExcludedEscape | Mode::Choices => *action == Action::Escape,
            _ => false,
        }
    }

    fn step(&self, state: &State, action: Action) -> Steps<State, (), ()> {
        use Action::{Enter, Escape, Left, Right};
        let edges: &[(u8, Action, u8)] = match self.0 {
            Mode::ExcludedEscape => &[(0, Left, 0), (0, Escape, 9)],
            Mode::Choices => &[(0, Escape, 0), (0, Escape, 9)],
            Mode::Prefix => &[(0, Enter, 1), (1, Left, 1)],
            Mode::MultipleStarts | Mode::SelfLoop => &[(0, Left, 0)],
            Mode::TargetBarrier => &[(0, Enter, 9), (9, Left, 1), (1, Right, 1)],
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
        let mut claim = Progress::<Self>::new("reaches_target")
            .whenever(|world, state| match world.0 {
                Mode::Prefix => (state.0 == 1).into(),
                Mode::TargetBarrier => (state.0 == 0).into(),
                _ => true.into(),
            })
            .eventually(|_, state| (state.0 == 9).into());
        claim = match self.0 {
            Mode::Choices => claim.within(Assumption::fair(&[Action::Escape])),
            Mode::Prefix => claim.within(Scope::without(&[Action::Enter])),
            Mode::ExcludedEscape => {
                claim.within(Assumption::fair(&[Action::Escape]).without([Action::Escape]))
            }
            _ => claim,
        };
        vec![claim]
    }
}

umpire_testing::fixture!(Graph(Mode::SelfLoop), 16);
