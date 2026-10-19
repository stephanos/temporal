use crate::Files;
use std::collections::BTreeMap;
use umpire_checker::Result as Exploration;
use umpire_dsl::Machine;

/// Render deterministic exploration results and counterexamples.
pub fn render<M: Machine>(result: &Exploration<M>) -> Result<Files, umpire_checker::Refusal> {
    let report = result.verify_all()?;
    let mut files = BTreeMap::from([("report.txt".into(), report.to_string())]);
    let mut states = String::new();
    for (id, state) in result.states() {
        states.push_str(&format!("{}: {state}\n", id.0));
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
            rows.push_str(&format!(
                "{}: {} choice={} outcome={:?} facts={:?} -> {}\n",
                id.0, edge.action, edge.choice, edge.emitted.0, edge.emitted.1, edge.emitted.2
            ));
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
