use umpire_dsl::{Claim, Machine, StepClaim, Steps, Verdict};

#[derive(Clone, PartialEq, Eq, Hash)]
pub struct Value(pub u8);

impl std::fmt::Display for Value {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Value { 0: ")?;
        write!(out, "{:?}", self.0)?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Clone, PartialEq, Eq, Hash)]
pub enum Action {
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

pub struct Tiny;

impl Machine for Tiny {
    type State = Value;
    type Action = Action;
    type Outcome = bool;
    type Fact = &'static str;

    fn starts(&self) -> Vec<Value> {
        vec![Value(0), Value(1), Value(0)]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Advance, Action::Reset, Action::Idle, Action::Unused]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &Value, action: Action) -> Steps<Value, bool, &'static str> {
        let mut steps = Steps::new();
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
        vec![Claim::<Self>::new("safe", |_, _| Verdict::Holds)]
    }
    
}

umpire_testing::fixture!(Tiny, 3);
