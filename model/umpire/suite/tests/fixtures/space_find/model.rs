use std::fmt::Write;

use umpire_suite::{Space, find, path};

use crate::kettle::{at, kettle::Model};

fn outcome<T: std::fmt::Display, E: std::fmt::Display>(answer: Result<T, E>) -> String {
    answer.map_or_else(|refusal| format!("refused: {refusal}\n"), |path| path.to_string())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let near = Space::from([at(97, false, false)]);
    let beeping = find(&near, path![at(97, false, false), .., |k: &Model| k.beeping]);
    write!(out, "beeping, from 97:\n{}", outcome(beeping))?;
    let either = Space::from([at(20, false, false), at(98, false, false)]);
    let hot = find(&either, path![|k: &Model| k.temperature > 90, .., |k| k.temperature == 100]);
    write!(out, "boiling, from whichever of 20 and 98 is above 90:\n{}", outcome(hot))?;
    let story = find(&near, path![at(97, false, false), .., |k: &Model| k.heating_on, ..=2, at(99, true, false), .., |k| !k.heating_on]);
    write!(out, "heating, 99 within 2, then off:\n{}", outcome(story))?;
    let tight = find(&near, path![at(97, false, false), ..=4, |k: &Model| k.beeping]);
    write!(out, "beeping within 4 steps:\n{}", outcome(tight))?;
    let freezing = find(&near, path![at(97, false, false), .., |k: &Model| k.temperature < 0]);
    write!(out, "below freezing:\n{}", outcome(freezing))?;
    let cut = find(&Space::from([at(20, false, false)]).budget(10), path![at(20, false, false), .., |k: &Model| k.beeping]);
    write!(out, "beeping, from 20, budget 10:\n{}", outcome(cut))?;
    let elsewhere = find(&near, path![at(20, false, false), .., |k: &Model| k.beeping]);
    write!(out, "beeping, from 20 outside the space:\n{}", outcome(elsewhere))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
