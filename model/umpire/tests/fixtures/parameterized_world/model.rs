//! One private two-parameter World supplies correlated complete starts.

use umpire_dsl::machine;

#[machine]
pub mod counter {
    use umpire_dsl::{Action, State, World, check, path};

    #[derive(World)]
    pub struct World {
        value: u8,
        limit: u8,
    }

    #[derive(State)]
    pub struct Model {
        pub value: u8,
        origin: u8,
    }

    #[derive(Action, Copy)]
    pub enum User {
        Advance,
    }

    impl Model {
        #[starts]
        fn starts(selected: &World) -> impl IntoIterator<Item = Self> {
            if selected.value >= selected.limit {
                Vec::new()
            } else {
                vec![
                    Self { value: selected.value, origin: selected.value },
                    Self { value: selected.value + 1, origin: selected.value + 1 },
                    Self { value: selected.value, origin: selected.value },
                ]
            }
        }

        #[on(User)]
        fn user(&mut self, requested: User, selected: &World) {
            match requested {
                User::Advance if self.value < selected.limit => {
                    self.value += 1;
                }
                User::Advance => {}
            }
        }

        #[invariant]
        fn bounded(&self, constants: &World) {
            check!(self.value <= constants.limit);
        }

        #[postcondition(after = User::Advance)]
        fn advances(&self, previous: &Self, constants: &World) {
            check!(self.value == previous.value + 1);
            check!(self.origin == previous.origin);
            check!(self.value <= constants.limit);
        }

        #[scope]
        fn admitted(_selected: &World) -> Assumption {
            only([User::Advance])
        }

        #[assumption]
        fn fair_user(_selected: &World) -> Assumption {
            fair([User::Advance])
        }

        #[progress]
        fn finishes(&self, selected: &World) -> LeadsTo {
            path![self.value == selected.value, .., self.value == selected.limit]
                .assuming(Self::fair_user(selected))
        }
    }
}
