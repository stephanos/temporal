use umpire::dsl::machine;

/// `stay!()` keeps an unchanged step enabled as an `Accepted` self-loop; a rejection still wins.
#[machine]
pub mod counter {
    use umpire::dsl::{Action, Outcome, Rejection, State, check, reject, stay};

    #[derive(State)]
    pub struct Model {
        #[starts(0..=0)]
        pub value: u8,
    }

    #[derive(Action, Copy)]
    pub enum User { Advance, Ping, Refuse }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => {
                    stay!();
                    self.value = 1;
                }
                User::Ping => stay!(),
                User::Refuse => {
                    stay!();
                    reject!(Rejection::FailedPrecondition);
                }
                User::Advance => {}
            }
        }

        #[postcondition]
        fn changed_or_stayed(&self, before: &Self, outcome: &Outcome) {
            check!(match outcome {
                Outcome::Accepted => self.value == 1 || self == before,
                Outcome::Rejected(rejection) => {
                    *rejection == Rejection::FailedPrecondition && self == before
                }
            });
        }
    }
}
