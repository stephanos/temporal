use std::{
    cell::Cell,
    collections::BTreeMap,
    hash::{Hash, Hasher},
    rc::Rc,
};

use umpire_checker::Checker;
use umpire_dsl::{Claim, Machine, Steps, Verdict};

pub struct World(pub Rc<Cell<usize>>);

pub struct State {
    pub value: usize,
    pub clones: Rc<Cell<usize>>,
}

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("State { value: ")?;
        write!(out, "{:?}", self.value)?;
        out.write_str(", clones: ")?;
        write!(out, "{}", self.clones.get())?;
        out.write_str(" }")?;
        Ok(())
    }
}

impl Clone for State {
    fn clone(&self) -> Self {
        self.clones.set(self.clones.get() + 1);
        Self { value: self.value, clones: self.clones.clone() }
    }
}

impl PartialEq for State {
    fn eq(&self, other: &Self) -> bool {
        self.value == other.value
    }
}
impl Eq for State {}
impl Hash for State {
    fn hash<H: Hasher>(&self, hasher: &mut H) {
        self.value.hash(hasher);
    }
}

#[derive(umpire_dsl::Action)]
pub enum Action {
    Advance,
}

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        vec![State { value: 0, clones: self.0.clone() }]
    }
    fn actions(&self) -> Vec<Action> {
        Action::actions()
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &State, _: Action) -> Steps<State, (), ()> {
        let mut steps = Steps::new();
        if state.value < 1000 {
            steps.push(((), vec![], State { value: state.value + 1, clones: self.0.clone() }));
        }
        steps
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("safe", |_, _| Verdict::Holds)]
    }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let world = World(Rc::new(Cell::new(0)));
    let result = Checker::explore(&world, 1001)?;
    let work = if world.0.get() <= 4 * result.counts.reached { "linear" } else { "excessive" };
    let mut report = result.verify_all()?.to_string();
    report.push_str("clone_work: ");
    report.push_str(work);
    report.push('\n');
    Ok(BTreeMap::from([("report.txt".into(), report)]))
}
