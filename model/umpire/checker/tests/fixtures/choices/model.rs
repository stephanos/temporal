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
        vec![Value(0)]
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
            steps.push((false, vec!["rejected"], Value(2)));
            steps.push((true, vec!["accepted"], Value(1)));
            steps.push((false, vec!["same_target"], Value(1)));
        }
        steps
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("safe", |_, _| Verdict::Holds)]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![StepClaim::<Self>::new("choice_facts", |_, _, after, _, outcome, facts| {
            ((*outcome && after.0 == 1 && facts == ["accepted"])
                || (!outcome
                    && ((after.0 == 2 && facts == ["rejected"])
                        || (after.0 == 1 && facts == ["same_target"]))))
            .into()
        })]
    }
}

umpire_testing::fixture!(Tiny, 3);
