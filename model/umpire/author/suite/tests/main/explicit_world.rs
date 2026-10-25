//! Selected world constants and ownership across suite queries and retained runs.

#![forbid(unsafe_code)]

use std::{cell::Cell, fmt, rc::Rc};

use umpire_dsl::{
    ActionDomain, Associated, CheckOutcome, Claim, Machine, Path, Pattern, Steps, path,
};
use umpire_suite::{Cases, Refusal, Run, Space, cover, find, refuted, sample, verify_all};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Model(u8);

impl fmt::Display for Model {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(out)
    }
}

struct World {
    start: u8,
    limit: u8,
    drops: Rc<Cell<usize>>,
}

impl Drop for World {
    fn drop(&mut self) {
        self.drops.set(self.drops.get() + 1);
    }
}

impl Associated for Model {
    type World = World;
}

impl Machine for World {
    type State = Model;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Model> {
        vec![Model(self.start)]
    }

    fn actions(&self) -> Vec<u8> {
        vec![1]
    }

    fn internal(&self, _: &u8) -> bool {
        false
    }

    fn step(&self, state: &Model, _: u8) -> Steps<Model, (), ()> {
        if state.0 < self.limit {
            [((), vec![], Model(state.0 + 1))].into_iter().collect()
        } else {
            Steps::new()
        }
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::new("starts_at_zero", |world, _| (world.start == 0).into())]
    }
}

fn world(start: u8, limit: u8, drops: &Rc<Cell<usize>>) -> World {
    World { start, limit, drops: drops.clone() }
}

#[test]
fn explicit_queries_use_selected_constants_and_starts() -> Result<(), Refusal> {
    let drops = Rc::new(Cell::new(0));
    let small = Space::<Model>::new(world(0, 1, &drops));
    let large = Space::<Model>::new(world(1, 2, &drops));
    assert_eq!(verify_all(&small)?.counts.reached, 2);
    assert_eq!(verify_all(&small)?.claims[0].outcome, CheckOutcome::Verified);
    assert_eq!(verify_all(&large)?.claims[0].outcome, CheckOutcome::Refuted);
    assert_eq!(verify_all(&large)?.counts.reached, 2);
    assert!(matches!(refuted(&small, |s| s.0 < 2), Err(Refusal::Holds { .. })));
    assert_eq!(refuted(&large, |s| s.0 < 2)?.steps[0].after, Model(2));
    assert!(find(&small, path![|_: &Model| true, .., Model(2)]).is_err());
    assert_eq!(find(&large, path![|_: &Model| true, .., Model(2)])?.start, Model(1));
    let action = || Path::pattern(Pattern::action(ActionDomain::new(Ok(1), |_| 1), None, None));
    assert_eq!(find(&large, action())?.steps[0].after, Model(2));
    assert!(matches!(
        Run::new(world(1, 1, &drops), Model(1)).step(1),
        Err(Refusal::Disabled { .. })
    ));
    Ok(())
}

#[test]
fn explicit_runs_retain_the_owned_world_after_space_is_dropped() -> Result<(), Refusal> {
    let drops = Rc::new(Cell::new(0));
    let (covered, sampled) = {
        let space = Space::with_world(world(0, 2, &drops), [Model(0)]);
        let covered = cover(&space, Cases(1), path![Model(0), .., Model(1)])?;
        let sampled = sample(&space, path![Model(0), .., Model(1)], 1, 7)?;
        (covered, sampled)
    };
    assert_eq!(drops.get(), 0);
    assert!(covered.all(|run| run.state() == &Model(1)));
    assert!(sampled.all(|run| run.state() == &Model(1)));
    drop(covered);
    assert_eq!(drops.get(), 0);
    drop(sampled);
    assert_eq!(drops.get(), 1);
    assert_eq!(Run::new(world(0, 1, &drops), Model(0)).step(1)?.state(), &Model(1));
    assert_eq!(
        Run::along_in(world(0, 1, &drops), path![Model(0), ..=1, Model(1)])?.state(),
        &Model(1)
    );
    assert_eq!(
        Run::along_in_with_budget(world(0, 1, &drops), path![Model(0), ..=1, Model(1)], 2)?.state(),
        &Model(1)
    );
    let empty = Space::with_world(world(0, 1, &drops), Vec::<Model>::new());
    assert_eq!(verify_all(&empty)?.counts.reached, 0);
    assert!(find(&empty, path![Model(0), .., Model(0)]).is_err());
    assert!(refuted(&empty, |_| false).is_err());
    assert!(cover(&empty, Cases(1), path![Model(0), .., Model(1)]).is_err());
    assert!(sample(&empty, path![Model(0), .., Model(1)], 1, 7).is_err());
    Ok(())
}

#[test]
fn cloning_and_formatting_spaces_require_no_world_traits() {
    let drops = Rc::new(Cell::new(0));
    let space = Space::with_world(world(0, 1, &drops), [Model(0)])
        .budget(2)
        .alternative_budget(3)
        .walk_bound(4);
    let copy = space.clone();
    assert_eq!(
        format!("{copy:?}"),
        "Space { starts: [Model(0)], budget: 2, alternatives: 3, walk_bound: 4 }"
    );
    drop(space);
    assert_eq!(drops.get(), 0);
    drop(copy);
    assert_eq!(drops.get(), 1);
}
