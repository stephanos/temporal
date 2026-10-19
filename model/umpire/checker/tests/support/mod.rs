use umpire_dsl::{Claim, Hole, Machine, StepClaim, Steps, Verdict};

#[derive(Clone, PartialEq, Eq, Hash)]
pub(crate) struct Value(u8);

impl std::fmt::Display for Value {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Value { 0: ")?;
        write!(out, "{:?}", self.0)?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub(crate) enum Action {
    Advance,
    Reset,
    Idle,
    Unused,
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Advance => "Action::Advance",
            Self::Reset => "Action::Reset",
            Self::Idle => "Action::Idle",
            Self::Unused => "Action::Unused",
        })?;
        Ok(())
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub(crate) enum Mode {
    SharedStarts,
    Edges,
    Fails,
    FailsOnEdge,
    EffectPanic,
    Overflow,
    InvariantPanic,
    PostconditionPanic,
    Holes,
    HoleThenFailure,
}

pub(crate) struct Tiny(pub(crate) Mode);

impl Machine for Tiny {
    type State = Value;
    type Action = Action;
    type Outcome = bool;
    type Fact = &'static str;

    fn starts(&self) -> Vec<Value> {
        if self.0 == Mode::SharedStarts {
            vec![Value(0), Value(1), Value(0)]
        } else {
            vec![Value(0)]
        }
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Advance, Action::Reset, Action::Idle, Action::Unused]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &Value, action: Action) -> Steps<Value, bool, &'static str> {
        let mut steps = Steps::new();
        if state.0 == 0 && action == Action::Advance {
            match self.0 {
                Mode::EffectPanic => panic!("broken heater"),
                Mode::Overflow => {
                    let limit = std::hint::black_box(u8::MAX);
                    steps.push((true, vec![], Value(limit + 1)));
                    return steps;
                }
                _ => {}
            }
        }
        let next = match (state.0, action) {
            (0 | 1, Action::Advance) => Some(Value(state.0 + 1)),
            (1, Action::Reset) => Some(Value(0)),
            (2, Action::Idle) => Some(state.clone()),
            _ => None,
        };
        if let Some(next) = next {
            steps.push((true, vec![], next));
        }
        steps
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("safe", |world, state| match world.0 {
            Mode::InvariantPanic if state.0 == 1 => panic!("broken invariant"),
            Mode::Holes => Verdict::Unknown(Hole { reason: "temperature unavailable" }),
            Mode::HoleThenFailure if state.0 == 0 => {
                Verdict::Unknown(Hole { reason: "initial gap" })
            }
            Mode::Fails | Mode::HoleThenFailure => Verdict::Fails,
            _ => Verdict::Holds,
        })]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        match self.0 {
            Mode::Edges => vec![
                StepClaim::<Self>::new("never_decreases", |_, before, after, _, _, _| {
                    (after.0 >= before.0).into()
                }),
                StepClaim::<Self>::after(
                    "idle_changes",
                    Action::Idle,
                    |_, before, after, _, _, _| (before != after).into(),
                ),
            ],
            Mode::FailsOnEdge => {
                vec![StepClaim::<Self>::new("unchanged", |_, before, after, _, _, _| {
                    (before == after).into()
                })]
            }
            Mode::PostconditionPanic => {
                vec![StepClaim::<Self>::new("panicking_edge", |_, _, _, _, _, _| {
                    panic!("broken postcondition")
                })]
            }
            Mode::Holes => vec![StepClaim::<Self>::new("edge_hole", |_, _, _, _, _, _| {
                Verdict::Unknown(Hole { reason: "edge unavailable" })
            })],
            _ => vec![],
        }
    }
}
