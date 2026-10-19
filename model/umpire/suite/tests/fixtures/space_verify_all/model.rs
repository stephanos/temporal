use std::fmt::Write;

use umpire_suite::{Refusal, Space, verify_all};

use crate::sample_kettle::kettle::Model;

fn room() -> Model {
    Model { temperature: 20, heating_on: false, beeping: false }
}

fn outcome<M: umpire_dsl::Machine>(report: Result<umpire_checker::Report<M>, Refusal>) -> String {
    report.map_or_else(|refusal| format!("refused: {refusal}\n"), |report| report.to_string())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    write!(out, "from room temperature:\n{}", outcome(verify_all(&Space::from([room()]))))?;
    let cut = verify_all(&Space::from([room()]).budget(10));
    write!(out, "from room temperature, budget 10:\n{}", outcome(cut))?;
    let empty = verify_all(&Space::from([room()]).budget(0));
    write!(out, "budget 0:\n{}", outcome(empty))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
