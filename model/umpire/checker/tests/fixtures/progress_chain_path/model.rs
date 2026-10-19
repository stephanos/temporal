use umpire_dsl::machine;

// A chain built outside a literal `path!` passes `#[machine]`; the checker reports it as a
// panicked claim when the conversion refuses it.
#[machine]
mod model {
    use umpire_dsl::{Action, Gap, State, Waypoint};

    #[derive(State)]
    pub struct Model {
        done: bool,
    }

    #[derive(Action, Copy)]
    pub enum User { Finish }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Finish => self.done = true,
            }
        }

        #[progress]
        fn chain(&self) -> LeadsTo {
            let chain = Waypoint::<Self>::Holds(!self.done)
                .then(Gap::Any, Waypoint::Holds(self.done))
                .then(Gap::Any, Waypoint::Holds(true));
            LeadsTo::from(chain)
        }
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let result = umpire_checker::Checker::explore(&model::world(), 8)?;
    let text = (result.verify_all()?).to_string();
    Ok(std::collections::BTreeMap::from([("report.txt".into(), text)]))
}
