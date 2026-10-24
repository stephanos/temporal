use umpire_dsl::{Claim, Machine, Steps};

struct World;

#[derive(Clone, PartialEq, Eq, Hash)]
struct State {
    value: Option<(u8, bool)>,
}

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("State { value: ")?;
        write!(out, "{:?}", self.value)?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Clone, PartialEq, Eq, Hash)]
enum Action {
    Advance { amount: u8 },
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let Self::Advance { amount } = self;
        out.write_str("Action::Advance { amount: ")?;
        write!(out, "{amount:?}")?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Debug)]
enum Decision {
    Accepted,
}

#[derive(Debug)]
#[expect(dead_code, reason = "the fact payload is consumed only by derived Debug formatting")]
enum Fact {
    Changed { amount: u8 },
}

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = Decision;
    type Fact = Fact;

    fn starts(&self) -> Vec<State> {
        vec![State { value: None }]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Advance { amount: 1 }]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &State, action: Action) -> Steps<State, Decision, Fact> {
        let Action::Advance { amount } = action;
        let mut steps = Steps::new();
        if state.value.is_none() {
            steps.push((
                Decision::Accepted,
                vec![Fact::Changed { amount }],
                State { value: Some((amount, true)) },
            ));
        }
        steps
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("unchanged", |_, state| state.value.is_none().into())]
    }
}

umpire_testing::fixture!(World, 2);
