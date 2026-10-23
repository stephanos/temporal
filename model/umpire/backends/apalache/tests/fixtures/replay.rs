//! Inputs, multiple starts, rejection and an unrelated after-filtered postcondition.
use umpire_dsl::machine;
#[machine]
pub(crate) mod inputs {
    use umpire_dsl::{Action, Outcome, Rejection, State, World, check, reject};
    #[derive(World)]
    pub(crate) struct World {
        pub limit: u8,
    }
    #[derive(State)]
    pub(crate) struct Model {
        pub value: u8,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Set(#[inputs([1, 1, 2])] u8),
        Reset,
    }
    impl Model {
        #[starts]
        fn initial(_: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: 0 }, Self { value: 1 }]
        }
        #[on(User)]
        fn step(&mut self, action: User, world: &World) {
            match action {
                User::Set(1) => {
                    reject!(Rejection::NotFound);
                }
                User::Set(value) if value <= world.limit => self.value = value,
                User::Reset if self.value > 0 => self.value = 0,
                _ => {}
            }
        }
        #[invariant]
        fn bounded(&self, world: &World) {
            check!(self.value <= world.limit);
        }
        #[postcondition(after=User::Reset)]
        fn no_reset(&self, before: &Self) {
            check!(self.value == before.value);
        }
        #[postcondition]
        fn no_rejection(&self, before: &Self, result: &Outcome) {
            if *result == Outcome::Rejected(Rejection::NotFound) {
                check!(self.value != before.value);
            }
        }
    }
}
