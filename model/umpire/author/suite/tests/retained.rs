//! Non-Clone outcomes and observable effects must retain the exact selected checker rows.

use std::{cell::Cell, fmt, rc::Rc};

use umpire_dsl::{ActionDomain, Machine, Path, Pattern, PatternError, Registered, Steps, Waypoint};
use umpire_suite::{Cases, Refusal, Run, Space, cover, find, refuted, sample, verify_all};

thread_local! {
    static WORLDS: Cell<usize> = const { Cell::new(0) };
    static STEPS: Cell<usize> = const { Cell::new(0) };
}

#[derive(Clone, PartialEq, Eq, Hash)]
struct Model;

impl fmt::Display for Model {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.write_str("Model")
    }
}

#[derive(Debug)]
struct Outcome(usize);

struct World {
    _not_send: Rc<()>,
}

impl Registered for Model {
    type World = World;
    fn world() -> World {
        WORLDS.set(WORLDS.get() + 1);
        World { _not_send: Rc::new(()) }
    }
}

impl Machine for World {
    type State = Model;
    type Action = u8;
    type Outcome = Outcome;
    type Fact = usize;
    fn starts(&self) -> Vec<Model> {
        vec![Model]
    }
    fn actions(&self) -> Vec<u8> {
        vec![0, 1]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, _: &Model, action: u8) -> Steps<Model, Outcome, usize> {
        STEPS.set(STEPS.get() + 1);
        let serial = STEPS.get() * 10;
        if action == 0 {
            [(Outcome(serial), vec![serial], Model), (Outcome(serial + 1), vec![serial + 1], Model)]
                .into_iter()
                .collect()
        } else {
            [(Outcome(serial), vec![serial], Model)].into_iter().collect()
        }
    }
}

fn action(value: u8, selected: Option<usize>) -> Pattern<Model, u8, Outcome> {
    Pattern::Action(
        ActionDomain::new(Ok(1), move |_| value),
        selected.map(|serial| {
            Rc::new(move |outcome: &Outcome| outcome.0 == serial) as Rc<dyn Fn(&Outcome) -> bool>
        }),
        None,
    )
}

#[test]
fn cover_and_along_never_replay_effects_or_rebuild_worlds() -> Result<(), Refusal> {
    WORLDS.set(0);
    STEPS.set(0);
    let covered = cover(
        &Space::from([Model]).budget(1),
        Cases(2),
        Path::pattern(Pattern::Either(vec![action(0, Some(11)), action(1, None)])),
    )?;
    assert_eq!(WORLDS.get(), 1);
    assert_eq!(STEPS.get(), 2);
    assert_eq!(covered.runs()[0].path().steps[0].choice, 1);
    assert!(covered.expansion().contains("Outcome(11), facts [11]"));
    assert!(covered.expansion().contains("Outcome(20), facts [20]"));
    assert!(covered.coverage().outcomes.is_none());
    WORLDS.set(0);
    STEPS.set(0);
    let along = Run::along(Path::pattern(Pattern::Sequence(vec![
        Pattern::State(Rc::new(Waypoint::Is(Model))),
        action(0, Some(11)),
    ])))?;
    assert_eq!(along.path().steps[0].choice, 1);
    assert_eq!(WORLDS.get(), 1);
    assert_eq!(STEPS.get(), 2);
    Ok(())
}

#[test]
fn refused_stories_do_not_construct_the_registered_world() {
    let paths = [
        Path::pattern(Pattern::action(
            ActionDomain::new(Err(PatternError::ExpansionOverflow), |_| 0),
            None,
            None,
        )),
        Path::pattern(Pattern::Either(vec![action(0, None), action(1, None)])),
        Path::pattern(action(0, None)),
        Path::pattern(Pattern::State(Rc::new(Waypoint::judged(|_| true)))),
    ];
    for path in paths {
        WORLDS.set(0);
        assert!(Run::along_with_budget(path, 1).is_err());
        assert_eq!(WORLDS.get(), 0);
    }
}

fn capped<S>(space: Space<S>) -> Space<S> {
    space.budget(10).alternative_budget(20).walk_bound(30)
}

fn copied<S: Clone>(space: &Space<S>) -> Space<S> {
    space.clone()
}

#[test]
fn legacy_spaces_preserve_unconstrained_configuration_and_auto_traits() {
    #[derive(Debug)]
    struct Unregistered;

    fn assert_send_sync<T: Send + Sync>() {}

    assert_send_sync::<Space<Model>>();
    WORLDS.set(0);
    let _ = copied(&Space::from([Model]));
    assert_eq!(WORLDS.get(), 0);
    let space = capped(Space::from(vec![Unregistered]));
    assert_eq!(
        format!("{space:?}"),
        "Space { starts: [Unregistered], budget: 10, alternatives: 20, walk_bound: 30 }"
    );
    let _ = capped(Space::from([Unregistered]));
}

#[test]
fn queries_take_the_state_as_their_one_explicit_generic_argument() -> Result<(), Refusal> {
    let space = Space::from([Model]);
    assert_eq!(verify_all::<Model>(&space)?.counts.reached, 1);
    assert_eq!(refuted::<Model>(&space, |_| false)?.steps.len(), 0);
    let pattern =
        || Path::<Model, u8, Outcome>::pattern(Pattern::State(Rc::new(Waypoint::Is(Model))));
    assert_eq!(find::<Model>(&space, pattern())?.steps.len(), 0);
    assert_eq!(cover::<Model>(&space, Cases(1), pattern())?.len(), 1);
    assert_eq!(sample::<Model>(&space, pattern(), 1, 7)?.len(), 1);
    Ok(())
}
