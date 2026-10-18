use umpire_dsl::{Action, Claim, Progress, State, invariant, machine, on, postcondition, progress};

#[derive(State)]
pub struct Counter {
    #[starts(0..=0)]
    value: u8,
}

#[derive(Action)]
pub enum User {
    Advance,
}

#[machine(actions = (User,))]
impl Counter {
    #[postcondition]
    fn advances(&self, before: &Self) -> bool {
        self.value == before.value + 1
    }

    #[progress]
    fn reaches_one() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("reaches_one")
            .whenever(|_, _| true.into())
            .eventually(|_, state| (state.value == 1).into())
    }

    #[invariant]
    fn bounded(&self) -> bool {
        self.value <= 1
    }

    #[postcondition]
    fn stays_zero(&self, _before: &Self) -> bool {
        self.value == 0
    }

    #[invariant]
    fn starts_zero(&self) -> bool {
        self.value == 0
    }

    #[progress]
    fn already_fulfilled() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("already_fulfilled")
            .whenever(|_, _| true.into())
            .eventually(|_, _| true.into())
    }

    #[on(User::Advance)]
    fn advance(&mut self) -> Option<()> {
        (self.value == 0).then(|| self.value += 1)
    }
}

umpire_testing::fixture!(CounterWorld, 2, Claim::<CounterWorld>::new("supplemental", |_, _| true.into()));
