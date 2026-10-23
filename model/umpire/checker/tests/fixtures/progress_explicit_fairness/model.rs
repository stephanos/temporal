use umpire_dsl::machine;

#[machine]
mod model {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub struct Model {
        done: bool,
        toggled: bool,
    }

    #[derive(Action, Copy)]
    pub enum User { Toggle }

    #[derive(Action, Copy)]
    #[internal]
    pub enum Clock { Finish }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Toggle if !self.done => self.toggled = !self.toggled,
                User::Toggle => {}
            }
        }

        #[on(Clock)]
        fn clock(&mut self, action: Clock) {
            match action {
                Clock::Finish => self.done = true,
            }
        }

        #[assumption]
        fn excluded_fair() -> Assumption {
            fair([Clock::Finish]).without([Clock::Finish])
        }

        #[progress]
        fn internal_is_not_fair(&self) -> LeadsTo {
            whenever(true).eventually(self.done)
        }

        #[progress]
        fn explicit_fair(&self) -> LeadsTo {
            whenever(true).eventually(self.done).assuming(fair([Clock::Finish]))
        }

        #[progress]
        fn fair_does_not_admit(&self) -> LeadsTo {
            whenever(true).eventually(self.done).assuming(Self::excluded_fair())
        }

        #[progress]
        fn only_finish(&self) -> LeadsTo {
            whenever(true).eventually(self.done).assuming(only([Clock::Finish]))
        }
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    umpire_testing::interpreted(model::world, None, 8, None)?;
        let result = umpire_checker::Checker::explore(&model::world(), 8)?;
    let text = (result.verify_all()?).to_string();
    Ok(std::collections::BTreeMap::from([("report.txt".into(), text)]))
}
