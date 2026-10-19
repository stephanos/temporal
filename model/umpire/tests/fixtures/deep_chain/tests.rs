use crate::deep_chain::World;
use std::{cell::Cell, collections::BTreeMap, rc::Rc};
use umpire_checker::Checker;

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
