//! One private two-parameter World supplies correlated complete starts.

use umpire_dsl::machine;

std::thread_local! {
    static ENUMERATIONS: std::cell::Cell<usize> = const { std::cell::Cell::new(0) };
}

pub(crate) fn enumerations() -> usize {
    ENUMERATIONS.get()
}

pub(crate) fn reset_enumerations() {
    ENUMERATIONS.set(0);
}

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

    #[starts]
    impl Model {
        fn starts(selected: &World) -> impl IntoIterator<Item = Self> {
            super::ENUMERATIONS.set(super::ENUMERATIONS.get() + 1);
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
    }

    impl Model {
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
