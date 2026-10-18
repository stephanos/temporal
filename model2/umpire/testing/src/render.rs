use crate::Files;
use std::{collections::BTreeMap, fmt::Write};
use umpire_checker::Result as Exploration;
use umpire_dsl::{Machine, Spell};

/// Render deterministic exploration results and counterexamples.
pub fn render<M: Machine>(result: &Exploration<M>) -> Result<Files, umpire_checker::Refusal> {
    let report = result.verify_all()?;
    let mut report_text = String::new();
    report.spell(&mut report_text);
    let mut files = BTreeMap::from([("report.txt".into(), report_text)]);
    let mut states = String::new();
    for (id, state) in result.states() {
        write!(states, "{}: ", id.0).ok();
        state.spell(&mut states);
        states.push('\n');
    }
    files.insert("states.txt".into(), states);
    if result.counts.reached <= 3 {
        let mut rows = String::new();
        for (id, _) in result.states() {
            for edge in result.rows(id)? {
                write!(rows, "{}: ", id.0).ok();
                edge.action.spell(&mut rows);
                write!(rows, " choice={} outcome=", edge.choice).ok();
                edge.emitted.0.spell(&mut rows);
                rows.push_str(" facts=");
                edge.emitted.1.spell(&mut rows);
                rows.push_str(" -> ");
                edge.emitted.2.spell(&mut rows);
                rows.push('\n');
            }
        }
        files.insert("rows.txt".into(), rows);
    }
    add_paths(&mut files, report);
    Ok(files)
}

/// Add counterexample paths from a report.
pub fn add_paths<M: Machine>(files: &mut Files, report: umpire_checker::Report<M>) {
    for claim in report.claims {
        if let Some(path) = claim.path {
            let mut text = String::new();
            path.spell(&mut text);
            files.insert(format!("paths/{}.txt", claim.name), text);
        }
    }
}
