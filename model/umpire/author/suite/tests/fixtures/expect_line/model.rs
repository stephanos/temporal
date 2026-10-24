use std::fmt::Write;

use umpire_suite::{Run, expect};

use crate::kettle::{at, kettle::User};

pub(crate) fn run() -> umpire_testing::RunResult {
    let run = Run::from(at(20, false, false)).step(User::PressButton)?;
    let mut out = String::new();
    let holds = crate::failure(|| {
        expect! { k = run.state();
            k.heating_on;
            !k.beeping;
        }
    });
    writeln!(out, "every line true: {holds}")?;
    let second = crate::failure(|| {
        expect! { k = run.state();
            k.heating_on;
            k.temperature == 100;
            k.beeping;
        }
    });
    writeln!(out, "second line false: {second}")?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
