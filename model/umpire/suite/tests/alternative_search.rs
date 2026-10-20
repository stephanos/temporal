//! A skipped effect is judged only after the global shortest alternative has been selected.

use std::fmt;

use umpire_dsl::{ActionDomain, Machine, Path, Pattern, Registered, Steps};
use umpire_suite::{Refusal, Space, find};

#[derive(Clone, PartialEq, Eq, Hash)]
struct Model(u8);

impl fmt::Display for Model {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(out, "{}", self.0)
    }
}

struct World;

impl Registered for Model {
    type World = World;
    fn world() -> World {
        World
    }
}

impl Machine for World {
    type State = Model;
    type Action = u8;
    type Outcome = ();
    type Fact = ();
    fn starts(&self) -> Vec<Model> {
        vec![Model(0)]
    }
    fn actions(&self) -> Vec<u8> {
        vec![0, 1, 2]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, state: &Model, action: u8) -> Steps<Model, (), ()> {
        match (state.0, action) {
            (0, 0) => [((), vec![], Model(10))].into_iter().collect(),
            (at @ 0..=5, 1) => [((), vec![], Model(at + 1))].into_iter().collect(),
            (5, 2) => panic!("skipped deep effect"),
            _ => Steps::new(),
        }
    }
}

fn action(value: u8) -> Pattern<Model, u8, ()> {
    Pattern::Action(ActionDomain::new(Ok(1), move |_| value), None, None)
}

#[test]
fn skipped_deep_effect_cannot_hide_a_shorter_alternative() -> Result<(), Refusal> {
    let space = Space::from([Model(0)]);
    let long = || Pattern::Repeat(Box::new(action(1)), 6, 6);
    let best = find(&space, Path::pattern(Pattern::Either(vec![long(), action(0)])))?;
    assert_eq!(best.steps.len(), 1);
    assert_eq!(best.steps[0].action, 0);
    assert!(matches!(find(&space, Path::pattern(long())), Err(Refusal::Skipped { .. })));
    let cut = space.budget(1);
    let empty =
        find(&cut, Path::pattern(Pattern::Either(vec![long(), Pattern::Sequence(vec![])])))?;
    assert!(empty.steps.is_empty());
    Ok(())
}
