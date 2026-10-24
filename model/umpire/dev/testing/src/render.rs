use crate::Files;
use std::{collections::BTreeMap, fmt::Write};
use umpire_checker::Result as Exploration;
use umpire_dsl::Machine;

/// Render deterministic exploration results and counterexamples.
pub fn render<M: Machine>(result: &Exploration<M>) -> Result<Files, umpire_checker::Refusal> {
    let report = result.verify_all()?;
    let mut files = BTreeMap::from([("report.txt".into(), report.to_string())]);
    let mut states = String::new();
    for (id, state) in result.states() {
        writeln!(states, "{}: {state}", id.0)
            .unwrap_or_else(|error| panic!("state formatting failed: {error}"));
    }
    files.insert("states.txt".into(), states);
    if result.counts.reached <= 3 {
        files.insert("rows.txt".into(), render_rows(result)?);
    }
    add_paths(&mut files, report);
    Ok(files)
}

/// Render emitted rows in discovery, action and choice order for native/host agreement.
pub fn render_rows<M: Machine>(result: &Exploration<M>) -> Result<String, umpire_checker::Refusal> {
    let mut rows = String::new();
    for (id, _) in result.states() {
        for edge in result.rows(id)? {
            writeln!(
                rows,
                "{}: {} choice={} outcome={:?} facts={:?} -> {}",
                id.0, edge.action, edge.choice, edge.emitted.0, edge.emitted.1, edge.emitted.2
            )
            .unwrap_or_else(|error| panic!("row formatting failed: {error}"));
        }
    }
    Ok(rows)
}

/// Add counterexample paths from a report.
pub fn add_paths<M: Machine>(files: &mut Files, report: umpire_checker::Report<M>) {
    for claim in report.claims {
        if let Some(path) = claim.path {
            files.insert(format!("paths/{}.txt", claim.name), path.to_string());
        }
    }
}
