use umpire_checker::{Checker, Refusal};
use umpire_dsl::{Machine, StepError, machine};

#[machine]
pub mod invalid {
    use umpire_dsl::{Action, Rejection, State};

    #[derive(State)]
    pub struct Model {
        #[starts(0..=0)]
        pub value: u8,
    }

    #[derive(Action, Copy)]
    pub enum User { RejectThenMutate, MutateThenReject }

    fn reject_helper() { umpire_dsl::reject!(Rejection::NotFound); }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::MutateThenReject => {
                    self.value = 1;
                    umpire_dsl::reject!(Rejection::NotFound);
                }
                User::RejectThenMutate => {
                    reject_helper();
                    self.value = 1;
                }
            }
        }
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let world = invalid::world();
    let before = invalid::Model { value: 0 };
    for action in world.actions() {
        assert_eq!(world.checked_step(&before, action), Err(StepError::RejectedAfterMutation));
    }
    let Err(error @ Refusal::Step { .. }) = Checker::explore(&world, 2) else {
        return Err("rejecting mutation was not refused".into());
    };
    Ok(umpire_testing::Files::from([("report.txt".into(), format!("refused: {error}\n"))]))
}
