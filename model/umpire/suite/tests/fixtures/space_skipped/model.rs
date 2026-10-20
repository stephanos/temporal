use std::fmt::Write;

use umpire_dsl::{Machine, Registered, Steps};
use umpire_suite::{Run, Space, find, path, refuted};

/// A dial turned from 0 up to 3 whose turn panics at 2, so 3 is never admitted.
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct Dial(u8);

impl std::fmt::Display for Dial {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "Dial({})", self.0)
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct Turn;

impl std::fmt::Display for Turn {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Turn")
    }
}

pub(crate) struct World;

impl Machine for World {
    type State = Dial;
    type Action = Turn;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Dial> {
        vec![Dial(0)]
    }
    fn actions(&self) -> Vec<Turn> {
        vec![Turn]
    }
    fn internal(&self, _: &Turn) -> bool {
        false
    }
    fn step(&self, state: &Dial, _: Turn) -> Steps<Dial, (), ()> {
        assert!(state.0 != 2, "stuck");
        let mut steps = Steps::new();
        if state.0 < 3 {
            steps.push(((), Vec::new(), Dial(state.0 + 1)));
        }
        steps
    }
}

impl Registered for Dial {
    type World = World;
    fn world() -> World {
        World
    }
}

fn outcome<T: std::fmt::Display, E: std::fmt::Display>(answer: Result<T, E>) -> String {
    answer.map_or_else(|refusal| format!("refused: {refusal}\n"), |path| path.to_string())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let space = Space::from([Dial(0)]);
    write!(out, "never 3:\n{}", outcome(refuted(&space, |dial| dial.0 < 3)))?;
    write!(out, "to 3:\n{}", outcome(find(&space, path![Dial(0), .., |dial: &Dial| dial.0 == 3])))?;
    write!(out, "to 1:\n{}", outcome(find(&space, path![Dial(0), .., |dial: &Dial| dial.0 == 1])))?;
    write!(out, "to 2 (panic at equal depth):\n{}", outcome(find(&space, path![Dial(0), .., Dial(2)])))?;
    write!(out, "refuted at 1:\n{}", outcome(refuted(&space, |dial| dial.0 < 1)))?;
    write!(out, "refuted at 2:\n{}", outcome(refuted(&space, |dial| dial.0 < 2)))?;
    write!(out, "along to 1:\n{}", outcome(Run::along(path![Dial(0), .., Dial(1)])))?;
    write!(out, "along to 2:\n{}", outcome(Run::along(path![Dial(0), .., Dial(2)])))?;
    let shallow = Space::from([Dial(2), Dial(0)]);
    write!(out, "to 1, panic at a shallower start:\n{}", outcome(find(&shallow, path![Dial(0), .., Dial(1)])))?;
    write!(out, "refuted at 1, panic at a shallower start:\n{}", outcome(refuted(&shallow, |dial| dial.0 != 1)))?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
