//! A present stored optional value still evaluates its authored fallback.
use umpire_dsl::machine;

#[machine]
pub(crate) mod optional_eager {
    use umpire_dsl::{Action, Finite, State, World, check};

    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub(crate) struct Remaining(#[inputs(0..=3)] pub u8);

    #[derive(World)]
    pub(crate) struct World {
        pub site: u8,
        pub divisor: u8,
    }

    #[derive(State)]
    pub(crate) struct Model {
        pub value: Option<Remaining>,
        pub done: bool,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Read,
    }

    impl Model {
        #[starts]
        fn starts(_: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: Some(Remaining(1)), done: false }]
        }

        #[on(User)]
        fn read(&mut self, action: User, selected: &World) {
            match action {
                User::Read
                    if !self.done
                        && (selected.site != 0
                            || self.value.unwrap_or(Remaining(1 / selected.divisor)).0 > 0) =>
                {
                    if selected.site == 1 {
                        self.value = Some(self.value.unwrap_or(Remaining(1 / selected.divisor)));
                    }
                    self.done = true;
                }
                User::Read => {}
            }
        }

        #[invariant]
        fn eager(&self, selected: &World) {
            if selected.site == 2 {
                check!(self.value.unwrap_or(Remaining(1 / selected.divisor)).0 == 1);
            }
        }
    }
}
