use crate::support::{Mode, Tiny};
use umpire_checker::{Checker, Refusal, StateId};

pub(crate) fn run() -> umpire_testing::RunResult {
    let result = Checker::explore(&Tiny(Mode::Edges), 3)?;
    let Err(error @ Refusal::State { given: 3 }) = result.rows(StateId(3)) else {
        return Err("unknown state did not return Refusal::State { given: 3 }".into());
    };
    Ok(umpire_testing::Files::from([("report.txt".into(), format!("refused: {error}\n"))]))
}
