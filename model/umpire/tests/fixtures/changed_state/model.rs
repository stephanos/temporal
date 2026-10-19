use umpire_dsl::machine;

#[machine]
pub mod changed_state {
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub struct Model {
        pub value: bool,
    }

    #[derive(Action)]
    pub enum User {
        NoChange,
        SameValue,
        MutateRestore,
        ActualChange,
    }

    #[transitions]
    impl Model {
        #[on(User::NoChange)]
        #[expect(clippy::unused_self, reason = "the no-op control requires a transition receiver")]
        fn no_change(&mut self) {}

        #[on(User::SameValue)]
        fn same_value(&mut self) {
            if !self.value {
                self.value = false;
            }
        }

        #[on(User::MutateRestore)]
        fn mutate_restore(&mut self) {
            let before = self.value;
            self.value = !self.value;
            self.value = before;
        }

        #[on(User::ActualChange)]
        fn actual_change(&mut self) {
            if !self.value {
                self.value = true;
            }
        }
    }

    #[postconditions]
    impl Model {
        #[after(User::NoChange)]
        fn no_change_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[after(User::SameValue)]
        fn same_value_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[after(User::MutateRestore)]
        fn mutate_restore_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[after(User::ActualChange)]
        fn actual_change_checked(&self, before: &Self) {
            check!(self.value == before.value);
        }
    }
}

umpire_testing::fixture!(changed_state::world(), 2);
