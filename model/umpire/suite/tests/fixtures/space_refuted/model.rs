use std::fmt::Write;

use umpire_suite::{Space, refuted};

use crate::kettle::{at, kettle::Model};

fn outcome<T: std::fmt::Display, E: std::fmt::Display>(answer: Result<T, E>) -> String {
    answer.map_or_else(|refusal| format!("refused: {refusal}\n"), |path| path.to_string())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let near = Space::from([at(97, false, false)]);
    write!(out, "never beeps, from 97:\n{}", outcome(refuted(&near, |k: &Model| !k.beeping)))?;
    let both = Space::from([at(20, false, false), at(99, true, false)]);
    write!(out, "never beeps, from 20 or 99 heating:\n{}", outcome(refuted(&both, |k| !k.beeping)))?;
    let bounded = refuted(&near, |k: &Model| k.temperature <= 100);
    write!(out, "at most boiling, from 97:\n{}", outcome(bounded))?;
    let early = refuted(&both.budget(12), |k: &Model| !k.beeping);
    write!(out, "never beeps, from 20 or 99 heating, budget 12:\n{}", outcome(early))?;
    let cut = refuted(&Space::from([at(20, false, false)]).budget(10), |k: &Model| !k.beeping);
    write!(out, "never beeps, from 20, budget 10:\n{}", outcome(cut))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
