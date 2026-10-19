use umpire_dsl::machine;

#[machine]
pub mod changed_state {
    use umpire_dsl::{Action, State, check};

    #[derive(State)]
    pub struct Model {
        pub value: bool,
    }

    #[derive(Action, Copy)]
    pub enum User {
        NoChange,
        SameValue,
        MutateRestore,
        ActualChange,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::SameValue if !self.value => {
                    self.value = false;
                }
                User::MutateRestore => {
                    let before = self.value;
                    self.value = !self.value;
                    self.value = before;
                }
                User::ActualChange if !self.value => {
                    self.value = true;
                }
                _ => {}
            }
        }

        #[postcondition(after = User::NoChange)]
        fn no_change_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[postcondition(after = User::SameValue)]
        fn same_value_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[postcondition(after = User::MutateRestore)]
        fn mutate_restore_checked(&self, before: &Self) {
            check!(self.value != before.value);
        }

        #[postcondition(after = User::ActualChange)]
        fn actual_change_checked(&self, before: &Self) {
            check!(self.value == before.value);
        }
    }
}
