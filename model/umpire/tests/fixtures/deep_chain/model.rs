use std::{
    cell::Cell,
    hash::{Hash, Hasher},
    rc::Rc,
};

#[cfg(test)]
use std::collections::BTreeMap;
#[cfg(test)]
use umpire_checker::Checker;
use umpire_dsl::{Claim, Machine, Spell, Steps, Verdict};

struct World(Rc<Cell<usize>>);

struct State {
    value: usize,
    clones: Rc<Cell<usize>>,
}

impl Spell for State {
    fn spell(&self, out: &mut String) {
        out.push_str("State { value: ");
        self.value.spell(out);
        out.push_str(", clones: ");
        self.clones.get().spell(out);
        out.push_str(" }");
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

impl Machine for World {
    type State = State;
    type Action = ();
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        vec![State { value: 0, clones: self.0.clone() }]
    }
    fn actions(&self) -> Vec<()> {
        vec![()]
    }
    fn internal(&self, (): &()) -> bool {
        false
    }
    fn step(&self, state: &State, (): ()) -> Steps<State, (), ()> {
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

#[cfg(test)]
pub(crate) fn run() -> umpire_testing::RunResult {
    let world = World(Rc::new(Cell::new(0)));
    let result = Checker::explore(&world, 1001)?;
    let work = if world.0.get() <= 4 * result.counts.reached { "linear" } else { "excessive" };
    let mut report = String::new();
    result.verify_all()?.spell(&mut report);
    report.push_str("clone_work: ");
    report.push_str(work);
    report.push('\n');
    Ok(BTreeMap::from([("report.txt".into(), report)]))
}
