use crate::Files;
use std::{
    collections::BTreeMap,
    fmt::{self, Write},
};
use umpire_checker::Result as Exploration;
use umpire_dsl::Machine;

/// Render deterministic exploration results and counterexamples.
pub fn render<M>(result: &Exploration<M>) -> Files
where
    M: Machine,
    M::State: fmt::Display,
    M::Outcome: fmt::Debug,
    M::Fact: fmt::Debug,
{
    let mut files = BTreeMap::from([("report.txt".into(), result.verify_all().to_string())]);
    let mut states = String::new();
    for (id, state) in result.states() {
        writeln!(states, "{}: {state}", id.0).ok();
    }
    files.insert("states.txt".into(), states);
    if result.counts.reached <= 3 {
        let mut rows = String::new();
        for (id, _) in result.states() {
            for edge in result.rows(id) {
                writeln!(
                    rows,
                    "{}: {:?} choice={} outcome={:?} facts={:?} -> {}",
                    id.0, edge.action, edge.choice, edge.emitted.0, edge.emitted.1, edge.emitted.2
                )
                .ok();
            }
        }
        files.insert("rows.txt".into(), rows);
    }
    add_paths(&mut files, result.verify_all());
    files
}

/// Add counterexample paths from a report.
pub fn add_paths<M: Machine>(files: &mut Files, report: umpire_checker::Report<M>)
where
    M::State: fmt::Display,
{
    for claim in report.claims {
        if let Some(path) = claim.path {
            files.insert(format!("paths/{}.txt", claim.name), path.to_string());
        }
    }
}
