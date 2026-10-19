use umpire_dsl::{Action, Claim, Machine, Progress, State, StepClaim, Steps};

#[derive(State)]
pub struct Counter {
    #[starts(0..=0)]
    pub value: u8,
}

#[derive(Action)]
pub enum User {
    Advance,
}

pub struct CounterWorld;

impl CounterWorld {
    fn advances() -> StepClaim<Self> {
        StepClaim::<Self>::new("advances", |_, before, after, _, (), _| {
            (after.value == before.value + 1).into()
        })
    }

    fn reaches_one() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("reaches_one")
            .whenever(|_, _| true.into())
            .eventually(|_, state| (state.value == 1).into())
    }

    fn bounded() -> Claim<Self> {
        Claim::<Self>::new("bounded", |_, state| (state.value <= 1).into())
    }

    fn stays_zero() -> StepClaim<Self> {
        StepClaim::<Self>::new("stays_zero", |_, _, after, _, (), _| (after.value == 0).into())
    }

    fn starts_zero() -> Claim<Self> {
        Claim::<Self>::new("starts_zero", |_, state| (state.value == 0).into())
    }

    fn already_fulfilled() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("already_fulfilled")
            .whenever(|_, _| true.into())
            .eventually(|_, _| true.into())
    }

}

impl Machine for CounterWorld {
    type State = Counter;
    type Action = User;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Counter> {
        Counter::starts()
    }

    fn actions(&self) -> Vec<User> {
        User::actions()
    }

    fn internal(&self, _: &User) -> bool {
        false
    }

    fn step(&self, state: &Counter, _: User) -> Steps<Counter, (), ()> {
        let mut steps = Steps::new();
        if state.value == 0 {
            steps.push(((), Vec::new(), Counter { value: 1 }));
        }
        steps
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Self::bounded(), Self::starts_zero()]
    }

    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![Self::advances(), Self::stays_zero()]
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        vec![Self::reaches_one(), Self::already_fulfilled()]
    }

    fn claim_order(&self) -> Vec<&'static str> {
        vec![
            "advances",
            "reaches_one",
            "bounded",
            "stays_zero",
            "starts_zero",
            "already_fulfilled",
        ]
    }
}

umpire_testing::fixture!(CounterWorld, 2, Claim::<CounterWorld>::new("supplemental", |_, _| true.into()));
