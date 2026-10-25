//! A state-only search over a space far larger than any fixed label cap, under the default
//! state budget.

use std::fmt;

use umpire_dsl::{Associated, Machine, Steps, path};
use umpire_suite::{Refusal, Space, find};

/// More states than a constant one-million label cap admits with two labels per state.
const LENGTH: u32 = 700_000;

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Model(u32);

impl fmt::Display for Model {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(out)
    }
}

struct Line;

impl Associated for Model {
    type World = Line;
}

impl Machine for Line {
    type State = Model;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Model> {
        vec![Model(0)]
    }

    fn actions(&self) -> Vec<u8> {
        vec![1]
    }

    fn internal(&self, _: &u8) -> bool {
        false
    }

    fn step(&self, state: &Model, _: u8) -> Steps<Model, (), ()> {
        if state.0 + 1 < LENGTH {
            [((), vec![], Model(state.0 + 1))].into_iter().collect()
        } else {
            Steps::new()
        }
    }
}

#[test]
fn state_only_find_answers_over_a_large_space_under_the_default_budget() -> Result<(), Refusal> {
    let space = Space::<Model>::new(Line);
    let found = find(&space, path![Model(0), .., Model(LENGTH - 1)])?;
    assert_eq!(found.steps.len(), LENGTH as usize - 1);
    assert_eq!(found.steps.last().map(|step| &step.after), Some(&Model(LENGTH - 1)));
    Ok(())
}
