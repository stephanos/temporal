use std::fmt::Write;

use umpire_suite::Run;

use crate::coin::{Coin, Toss};

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let Err(tossed) = Run::from(Coin::Unseen).step(Toss) else { return Err("toss".into()) };
    writeln!(out, "tossed: {tossed}")?;
    let Err(again) = Run::from(Coin::Heads).step(Toss) else { return Err("toss again".into()) };
    writeln!(out, "tossed a seen coin: {again}")?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
