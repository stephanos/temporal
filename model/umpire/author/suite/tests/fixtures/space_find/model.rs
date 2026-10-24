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
    let room = Space::from([at(20, false, false)]);
    let overall = find(&room, path![at(20, false, false), .., |k: &Model| k.heating_on, ..=1, |k| k.temperature == 22]);
    write!(out, "heating, then 22 within 1 (the first heating state is too far):\n{}", outcome(overall))?;
    let settled = find(&room.budget(10), path![at(20, false, false), ..=2, |k: &Model| k.beeping]);
    write!(out, "beeping within 2, from 20, budget 10:\n{}", outcome(settled))?;
    let starts = find(&either, path![|k: &Model| k.temperature < 0, .., |k| k.beeping]);
    write!(out, "below freezing, from 20 or 98:\n{}", outcome(starts))?;
    let duplicate = Space::from([at(20, false, false), at(20, false, false), at(98, false, false)]).budget(1);
    write!(out, "duplicate starts, below freezing, budget 1:\n{}", outcome(find(&duplicate, path![false, .., true])))?;
    for (name, pattern) in [
        ("predicate at start", path![|_: &Model| panic!("start predicate"), .., true]),
        ("predicate after a step", path![at(97, false, false), .., |k: &Model| {
            assert!(k.temperature != 98, "later predicate");
            false
        }]),
        ("non-string predicate panic", path![at(97, false, false), .., |_: &Model| std::panic::panic_any(7u8)]),
    ] {
        let answer = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| find(&near, pattern)));
        let text = answer.map_or_else(|_| "escaped predicate panic\n".into(), outcome);
        write!(out, "{name}:\n{text}")?;
    }
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
