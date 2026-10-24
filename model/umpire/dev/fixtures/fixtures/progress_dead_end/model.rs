pub use crate::sample_kettle::kettle::{self, Heater, Model, User};
use umpire::dsl::{Machine, Progress, Scope, Steps};

pub struct ProgressDeadEnd<M>(pub M);

impl<M> Machine for ProgressDeadEnd<M>
where
    M: Machine<State = Model, Outcome = (), Fact = ()>,
    M::Action: From<User> + From<Heater>,
{
    type State = Model;
    type Action = M::Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Model> {
        vec![Model { temperature: 20, heating_on: false, beeping: false }]
    }

    fn actions(&self) -> Vec<M::Action> {
        self.0.actions()
    }

    fn internal(&self, action: &M::Action) -> bool {
        self.0.internal(action)
    }

    fn step(&self, state: &Model, action: M::Action) -> Steps<Model, (), ()> {
        if action == M::Action::from(Heater::Heat) {
            Steps::new()
        } else {
            self.0.step(state, action)
        }
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        vec![
            Progress::<Self>::new("boils")
                .whenever(|_, state| state.heating_on.into())
                .eventually(|_, state| state.beeping.into())
                .within(Scope::without(&[M::Action::from(User::DepressButton)])),
        ]
    }
}
