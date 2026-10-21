use super::ledger::{Event, Journal};
use crate::evidence::{
    canonical,
    store::{Opened, open},
    symbolize::{
        self, Site,
        inspection::{choice_sites, symbolize_choices},
    },
};
use serde_json::{Value, json};
use std::{io::Write, path::Path};

pub(crate) fn is_campaign(path: &Path) -> Result<bool, String> {
    let root = open::pin(path)?;
    let storage = open::read(&root, "storage.json", 1024, false)?;
    let value: Value = serde_json::from_slice(&storage).map_err(|error| error.to_string())?;
    Ok(matches!(
        value["format"].as_str(),
        Some("gomad.campaign-storage/v2" | "gomad.exploration-round-storage/v1")
    ))
}

pub(crate) fn run(
    path: &Path,
    json_output: bool,
    choices: bool,
    output: &mut dyn Write,
) -> Result<(), String> {
    if super::exploration::plan::is_round_campaign(path)? {
        return run_round(path, json_output, choices, output);
    }
    let journal = Journal::open(path, false)?;
    let mut executions = Vec::new();
    let mut stop = None;
    for event in journal.events() {
        match event {
            Event::Outcome {
                executions: values,
                stop_reason,
                ..
            } => {
                executions.extend(values);
                if stop_reason.is_some() {
                    stop = stop_reason.as_deref();
                }
            }
            Event::Complete { stop_reason } => stop = Some(stop_reason.as_str()),
            _ => {}
        }
    }
    let mut projected = Vec::new();
    for execution in executions {
        let opened = execution
            .artifact
            .as_deref()
            .map(|reference| journal.open_artifact(reference))
            .transpose()?;
        projected.push(project_execution(execution, opened.as_ref(), choices)?);
    }
    render(path, json_output, output, &journal, projected, stop)
}

fn run_round(
    path: &Path,
    json_output: bool,
    choices: bool,
    output: &mut dyn Write,
) -> Result<(), String> {
    let journal = super::exploration::round::RoundJournal::open(path, false)?;
    let mut executions = Vec::new();
    for execution in journal
        .events()
        .iter()
        .flat_map(super::exploration::round::RoundEvent::executions)
    {
        let opened = execution
            .artifact
            .as_deref()
            .map(|reference| journal.open_artifact(reference))
            .transpose()?;
        executions.push(project_execution(execution, opened.as_ref(), choices)?);
    }
    let envelope = journal.summary()?;
    let report = json!({
        "schema":"gomad.inspect/v9","kind":"campaign","path":path,
        "campaign":{"plan":journal.plan(),"executions":executions,
            "stop_reason":envelope["stop_reason"],"events":journal.events().len(),
            "envelope":envelope}
    });
    if json_output {
        writeln!(
            output,
            "{}",
            String::from_utf8(canonical::encode(&report)?).map_err(|error| error.to_string())?
        )
        .map_err(|error| error.to_string())
    } else {
        writeln!(
            output,
            "gomad campaign: path={} attempted={} stop={}",
            path.display(),
            executions.len(),
            envelope["stop_reason"]
        )
        .map_err(|error| error.to_string())?;
        for execution in executions {
            writeln!(
                output,
                "{}",
                String::from_utf8(canonical::encode(&execution)?)
                    .map_err(|error| error.to_string())?
            )
            .map_err(|error| error.to_string())?;
        }
        Ok(())
    }
}

pub(super) fn project_execution(
    execution: &super::ledger::Execution,
    opened: Option<&Opened>,
    choices: bool,
) -> Result<Value, String> {
    let mut value = serde_json::to_value(execution).map_err(|error| error.to_string())?;
    let projected_choices = if let Some(opened) = opened
        && (choices || execution.divergence.is_some())
    {
        crate::evidence::inspection::Details::read(opened)?
            .and_then(|details| details.choices)
            .map(|evidence| evidence.inspect(opened))
            .transpose()?
    } else {
        None
    };
    let divergence_sites = execution.divergence.iter().flat_map(|divergence| {
        divergence
            .expected
            .iter()
            .chain(divergence.observed.iter())
            .map(|decision| {
                let (value, missing) = decision.site();
                Site { value, missing }
            })
    });
    let symbols = if projected_choices.is_some() || execution.divergence.is_some() {
        Some(symbolize::resolve(
            opened.ok_or("campaign divergence requires authenticated retained execution")?,
            projected_choices
                .iter()
                .flat_map(choice_sites)
                .chain(divergence_sites),
        )?)
    } else {
        None
    };
    if let Some(divergence) = &execution.divergence {
        let symbols = symbols.as_ref().ok_or("campaign symbols missing")?;
        let mut divergence_value = json!({"ordinal":divergence.ordinal,"reason":divergence.reason_name(),"tape_records":divergence.tape_records,"expected":divergence.expected,"observed":divergence.observed});
        for (name, decision) in [
            ("expected", divergence.expected.as_ref()),
            ("observed", divergence.observed.as_ref()),
        ] {
            if let Some(decision) = decision {
                let (value, missing) = decision.site();
                divergence_value[name]["symbol"] =
                    serde_json::to_value(symbols.get(Site { value, missing }))
                        .map_err(|error| error.to_string())?;
            }
        }
        value["divergence"] = divergence_value;
    }
    if let Some(opened) = opened
        && execution
            .record
            .streams
            .contains_key("evidence/soft-assertions.json")
    {
        value["soft_assertions"] =
            serde_json::from_slice(&opened.payload("evidence/soft-assertions.json", 16 << 20)?)
                .map_err(|error| error.to_string())?;
    }
    if choices && let Some((choices, symbols)) = projected_choices.zip(symbols.as_ref()) {
        value["choices"] = serde_json::to_value(symbolize_choices(choices, symbols))
            .map_err(|error| error.to_string())?;
    }
    Ok(value)
}

fn render(
    path: &Path,
    json_output: bool,
    output: &mut dyn Write,
    journal: &Journal,
    projected: Vec<Value>,
    stop: Option<&str>,
) -> Result<(), String> {
    if json_output {
        let report = json!({"schema":"gomad.inspect/v9","kind":"campaign","path":path,"campaign":{"plan":journal.plan(),"executions":projected,"stop_reason":stop,"events":journal.events().len()}});
        writeln!(
            output,
            "{}",
            String::from_utf8(canonical::encode(&report)?).map_err(|error| error.to_string())?
        )
        .map_err(|error| error.to_string())?;
    } else {
        writeln!(
            output,
            "gomad campaign: path={} selected={} attempted={} stop={}",
            path.display(),
            journal.plan()["selection_count"],
            projected.len(),
            stop.unwrap_or("interrupted")
        )
        .map_err(|error| error.to_string())?;
        for execution in projected {
            writeln!(
                output,
                "seed={} ordinal={} outcome={} artifact={}",
                execution["seed"],
                execution["ordinal"],
                execution["record"]["outcome"]["class"],
                execution["artifact"]
            )
            .map_err(|error| error.to_string())?;
        }
    }
    Ok(())
}
