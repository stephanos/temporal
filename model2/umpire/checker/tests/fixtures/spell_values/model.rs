use umpire_dsl::{Claim, Machine, Spell, Steps};

struct World;

#[derive(Clone, PartialEq, Eq, Hash)]
struct State {
    value: Option<(u8, bool)>,
}

impl Spell for State {
    fn spell(&self, out: &mut String) {
        out.push_str("State { value: ");
        self.value.spell(out);
        out.push_str(" }");
    }
}

#[derive(Clone, PartialEq, Eq, Hash)]
enum Action {
    Advance { amount: u8 },
}

impl Spell for Action {
    fn spell(&self, out: &mut String) {
        let Self::Advance { amount } = self;
        out.push_str("Action::Advance { amount: ");
        amount.spell(out);
        out.push_str(" }");
    }
}

enum Decision {
    Accepted,
}

impl Spell for Decision {
    fn spell(&self, out: &mut String) {
        match self {
            Self::Accepted => out.push_str("Decision::Accepted"),
        }
    }
}

enum Fact {
    Changed { amount: u8 },
}

impl Spell for Fact {
    fn spell(&self, out: &mut String) {
        let Self::Changed { amount } = self;
        out.push_str("Fact::Changed { amount: ");
        amount.spell(out);
        out.push_str(" }");
    }
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
