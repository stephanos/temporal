use crate::support::{Mode, Tiny};
use std::{collections::BTreeMap, panic};
use umpire_checker::Checker;

pub(crate) fn run() -> umpire_testing::RunResult {
    let error = panic::catch_unwind(|| Checker::explore(&Tiny(Mode::Edges), 0))
        .err().ok_or("zero budget was accepted")?;
    let message = error.downcast_ref::<&str>().copied()
        .or_else(|| error.downcast_ref::<String>().map(String::as_str))
        .ok_or("budget refusal has no message")?;
    Ok(BTreeMap::from([("report.txt".into(), format!("refused: {message}\n"))]))
}
