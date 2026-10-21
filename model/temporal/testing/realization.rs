//! Coverage of actual admitted realization contexts, using the runtime guard metadata.

use std::fmt::Write;
use umpire_temporal::{Context, Realization};
use umpire_temporal_api::RPC_BINDINGS;
use umpire_testing::{Consumer, Files};

pub(super) const COVERAGE: Consumer = Consumer {
    name: "realization",
    expected: &["realization.txt"],
    render: |_, folder| {
        if super::fixture_name(folder)? != "activity" {
            return Err("fixture has no registered realization".into());
        }
        Ok(Files::from([("realization.txt".into(), render(Realization::REALIZATION)?)]))
    },
};

fn render(contexts: &[Context]) -> Result<String, Box<dyn std::error::Error>> {
    let mut report = String::from(
        "Activity realization coverage\n\
         classifications: mapped | ignored (reason) | unsupported\n\
         mapped means read for incoming contexts and produced for outgoing requests.\n\
         A classified parent covers its subtree; opaque parents stop traversal.\n\
         Unsupported decoded presence refuses; proto3 scalar defaults have no wire presence.\n",
    );
    for context in contexts {
        writeln!(
            report,
            "\n{}: {}",
            context.kind,
            context.method.unwrap_or("ambiguous RPC owner")
        )?;
        writeln!(report, "message: {}", context.message)?;
        if context.fields.is_empty() {
            writeln!(report, "  (no fields)")?;
        }
        for field in context.fields {
            write!(report, "  {} {}", field.class, field.path)?;
            if let Some(reason) = field.reason {
                write!(report, " - {reason}")?;
            }
            writeln!(report)?;
        }
    }
    let actions = RPC_BINDINGS
        .iter()
        .filter(|(_, role, _)| *role == "Action")
        .map(|(rpc, _, action)| {
            let (service, method) = rpc.rsplit_once('.').ok_or("invalid admitted RPC binding")?;
            Ok((format!("/{service}/{method}"), *action))
        })
        .collect::<Result<Vec<_>, Box<dyn std::error::Error>>>()?;
    writeln!(report, "\nModel-action RPCs without an observation handler:")?;
    for (rpc, action) in &actions {
        let bound: Vec<_> = contexts.iter().filter(|context| context.method == Some(rpc)).collect();
        if !bound.iter().any(|context| context.kind == "observed response") {
            writeln!(report, "  {rpc} -> {action}")?;
            if bound.is_empty() {
                writeln!(report, "    no realization context")?;
            } else {
                for context in bound {
                    writeln!(report, "    covered only as {}", context.kind)?;
                }
            }
        }
    }
    writeln!(report, "\nModel-action RPCs without any realization context:")?;
    let unhandled: Vec<_> = actions
        .iter()
        .filter(|(rpc, _)| !contexts.iter().any(|context| context.method == Some(rpc)))
        .collect();
    if unhandled.is_empty() {
        writeln!(report, "  (none)")?;
    }
    for (rpc, action) in unhandled {
        writeln!(report, "  {rpc} -> {action}")?;
    }
    Ok(report)
}
