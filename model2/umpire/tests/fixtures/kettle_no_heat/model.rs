use crate::fixture_kettle::{Act, Kettle, KettleWorld};
use umpire_dsl::{Machine, Progress, Scope, Steps};

pub struct NoHeat;

impl Machine for NoHeat {
    type State = Kettle;
    type Action = Act;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Kettle> {
        vec![Kettle { temperature: 20, heating_on: false, beeping: false }]
    }

    fn actions(&self) -> Vec<Act> {
        KettleWorld.actions()
    }

    fn internal(&self, action: &Act) -> bool {
        KettleWorld.internal(action)
    }

    fn step(&self, state: &Kettle, action: Act) -> Steps<Kettle, (), ()> {
        if action == Act::HeaterHeat { Steps::new() } else { KettleWorld.step(state, action) }
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        vec![
            Progress::<Self>::new("boils")
                .whenever(|_, state| state.heating_on.into())
                .eventually(|_, state| state.beeping.into())
                .within(Scope::without(&[Act::UserDepressButton])),
        ]
    }
}

#[cfg(test)]
umpire_testing::fixture!(NoHeat, 16);
