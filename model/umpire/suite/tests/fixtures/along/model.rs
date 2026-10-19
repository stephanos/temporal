use std::fmt::Write;

use umpire_suite::{Refusal, Run, path};

use crate::{coin::Coin, kettle::{at, kettle::Model}};

fn outcome<S: umpire_dsl::Registered>(run: Result<Run<S>, Refusal>) -> String {
    match run {
        Ok(run) => run.to_string(),
        Err(refusal) => format!("refused: {refusal}\n"),
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let boils = Run::along(path![at(97, false, false), .., |k| k.heating_on, .., |k| k.beeping]);
    write!(out, "on, then beeping:\n{}", outcome(boils))?;
    let states = Run::along(path![at(97, false, false), ..=4, at(99, true, false), .., at(100, false, false)]);
    write!(out, "through states, then switched off at boiling:\n{}", outcome(states))?;
    let within = Run::along(path![at(97, false, false), ..=5, |k: &Model| k.beeping]);
    write!(out, "beeping within 5 steps:\n{}", outcome(within))?;
    let tight = Run::along(path![at(97, false, false), ..=4, |k: &Model| k.beeping]);
    write!(out, "beeping within 4 steps:\n{}", outcome(tight))?;
    let freezing = Run::along(path![at(97, false, false), .., |k: &Model| k.temperature < 0]);
    write!(out, "below freezing:\n{}", outcome(freezing))?;
    let cooled = Run::along(path![at(97, false, false), .., |k| k.beeping, .., |k| k.temperature < 100]);
    write!(out, "beeping, then cooled:\n{}", outcome(cooled))?;
    let tossed = Run::along(path![Coin::Unseen, .., Coin::Tails]);
    write!(out, "a toss showing tails:\n{}", outcome(tossed))?;
    let unanchored = Run::along(path![|k: &Model| !k.heating_on, .., |k| k.beeping]);
    write!(out, "from a predicate:\n{}", outcome(unanchored))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
