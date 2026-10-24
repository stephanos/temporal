use umpire::dsl::machine;

#[machine]
pub mod counter {
    use umpire::dsl::{Action, Outcome, Rejection, State, check, reject};

    #[derive(State)]
    pub struct Model {
        #[starts(0..=0)]
        pub value: u8,
    }

    #[derive(Action, Copy)]
    pub enum User { Advance, Disabled, Reject, Helper }

    fn reject_helper() {
        umpire::dsl::reject!(Rejection::Unavailable);
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => self.value = 1,
                User::Reject => reject!(Rejection::NotFound),
                User::Helper => reject_helper(),
                _ => {}
            }
        }

        #[postcondition]
        fn correct_outcome(&self, before: &Self, outcome: &Outcome) {
            check!(match outcome {
                Outcome::Accepted => self.value == 1 && before.value == 0,
                Outcome::Rejected(Rejection::NotFound | Rejection::Unavailable) => self == before,
                Outcome::Rejected(_) => false,
            });
        }

        #[postcondition(after = User::Reject)]
        fn rejected_not_found(&self, before: &Self, outcome: &Outcome) {
            check!(self == before && *outcome == Outcome::Rejected(Rejection::NotFound));
        }
    }
}
