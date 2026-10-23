//! One valid transition precedes the evaluated checked-integer refusal.
use umpire_dsl::machine;
#[machine]
pub(crate) mod prefix {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    pub(crate) struct World {}
    #[derive(State)]
    pub(crate) struct Model {
        #[starts(254..=254)]
        pub value: u8,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Advance,
    }
    impl Model {
        #[on(User)]
        fn advance(&mut self, action: User) {
            match action {
                User::Advance => self.value += 1,
            }
        }
    }
}
