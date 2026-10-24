use crate::support::{Mode, Tiny};
use std::collections::BTreeMap;
use umpire_checker::{Checker, Refusal};

pub(crate) fn run() -> umpire_testing::RunResult {
    let Err(error @ Refusal::Budget { given: 0 }) = Checker::explore(&Tiny(Mode::Edges), 0) else {
        return Err("zero budget did not return Refusal::Budget { given: 0 }".into());
    };
    Ok(BTreeMap::from([("report.txt".into(), format!("refused: {error}\n"))]))
}
