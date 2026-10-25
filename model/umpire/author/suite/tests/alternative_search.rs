//! A skipped effect is judged only after the global shortest alternative has been selected.

#![forbid(unsafe_code)]

use std::{fmt, rc::Rc};

use umpire_dsl::{ActionDomain, Machine, Path, Pattern, Registered, Steps, Waypoint};
use umpire_suite::{Refusal, Run, Space, find, path};

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

#[test]
fn along_preserves_skipped_step_refusals_for_structural_and_legacy_paths() -> Result<(), Refusal> {
    let anchored = |start, next| {
        Path::pattern(Pattern::Sequence(vec![
            Pattern::State(Rc::new(Waypoint::Is(Model(start)))),
            action(next),
        ]))
    };
    let structural = Run::along(anchored(5, 2));
    let legacy = Run::along(path![Model(5), ..=1, Model(99)]);
    assert!(matches!(structural, Err(Refusal::Skipped { .. })));
    assert!(matches!(legacy, Err(Refusal::Skipped { .. })));
    let early = Run::along(anchored(0, 0))?;
    assert_eq!(early.path().steps.len(), 1);
    Ok(())
}

#[test]
fn along_keeps_predicate_failure_ahead_of_unrelated_skips() {
    let pattern = Path::pattern(Pattern::Sequence(vec![
        Pattern::State(Rc::new(Waypoint::Is(Model(0)))),
        action(0),
        Pattern::State(Rc::new(Waypoint::judged(|_| panic!("authored predicate")))),
    ]));
    assert!(matches!(Run::along(pattern), Err(Refusal::Predicate { .. })));
}
