mod cluster;
mod storage;

use crate::evidence::inspection::{Choices, Details, Output, Simulation, World};
use crate::evidence::symbolize::{
    self,
    inspection::{Trace, choice_sites, symbolize_choices},
};
use crate::evidence::{canonical, record::Record, store::Opened};
use serde::Serialize;
use std::{
    io::Write,
    path::{Component, Path, PathBuf},
};

#[derive(Serialize)]
pub struct Inspection<'a> {
    schema: &'static str,
    kind: &'static str,
    path: &'a Path,
    artifact: Artifact<'a>,
}

#[derive(Serialize)]
struct Artifact<'a> {
    record: &'a Record,
    record_sha256: &'a str,
    failure_signature: String,
    stored_bytes: u64,
    files: &'a std::collections::BTreeMap<String, crate::evidence::store::FileEntry>,
    stdout: Output,
    stderr: Output,
    choices: Option<Choices<Trace<'a>>>,
    world: Option<&'a World>,
    simulation: Option<&'a Simulation>,
    minimization: Option<crate::minimize::report::Report<'a>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    storage: Option<storage::Storage>,
    #[serde(skip_serializing_if = "Option::is_none")]
    cluster: Option<cluster::Cluster>,
}

fn artifact_path(operand: &Path) -> Result<PathBuf, String> {
    let absolute = std::path::absolute(operand).map_err(|error| error.to_string())?;
    let mut path = PathBuf::new();
    for component in absolute.components() {
        if component == Component::ParentDir {
            path.pop();
        } else {
            path.push(component.as_os_str());
        }
    }
    Ok(path)
}

fn parse(arguments: &[String]) -> Result<(bool, bool, PathBuf), super::Error> {
    let mut json = false;
    let mut choices = false;
    let mut positional = false;
    let mut path = None;
    for argument in arguments {
        if !positional && argument == "--" {
            positional = true;
        } else if !positional && argument.starts_with('-') {
            let flag = argument
                .strip_prefix("--")
                .or_else(|| argument.strip_prefix('-'))
                .ok_or("invalid inspect option")?;
            let (name, value) = flag.split_once('=').unwrap_or((flag, "true"));
            let destination = match name {
                "json" => &mut json,
                "choices" => &mut choices,
                _ => return Err(format!("unknown inspect option: {argument}").into()),
            };
            *destination = match value {
                "1" | "t" | "T" | "true" | "TRUE" | "True" => true,
                "0" | "f" | "F" | "false" | "FALSE" | "False" => false,
                _ => return Err(format!("invalid boolean inspect option: {argument}").into()),
            };
        } else if path.replace(Path::new(argument)).is_some() {
            return Err("usage: gomad inspect [--json] [--choices] ARTIFACT".into());
        } else {
            positional = true;
        }
    }
    let path = artifact_path(path.ok_or("usage: gomad inspect [--json] [--choices] ARTIFACT")?)?;
    Ok((json, choices, path))
}

/// # Errors
/// Refuses invalid flags, unsupported artifacts and unauthenticated retained evidence.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<(), super::Error> {
    let (json, choices, path) = parse(arguments)?;
    let path = path.as_path();
    if crate::campaign::inspect::is_campaign(path)? {
        return crate::campaign::inspect::run(path, json, choices, output).map_err(Into::into);
    }
    let opened =
        Opened::open(path).map_err(|error| format!("inspect {}: {error}", path.display()))?;
    if opened.manifest().files.contains_key("record.json")
        && Record::decode(&opened.payload("record.json", 16 << 20)?)? != opened.manifest().record
    {
        return Err("record payload mismatch".into());
    }
    let details = Details::read(&opened)?;
    let choice_evidence = details
        .as_ref()
        .and_then(|details| details.choices.as_ref());
    if choices && choice_evidence.is_none() {
        return Err("artifact has no choice trace".into());
    }
    let projected_choices = choice_evidence
        .map(|evidence| evidence.inspect(&opened))
        .transpose()?;
    opened.authenticate_all()?;
    let minimization = details
        .as_ref()
        .and_then(|details| details.minimization.as_ref());
    let minimization_projection = minimization
        .map(|metadata| crate::minimize::report::Projection::read(&opened, metadata))
        .transpose()?;
    let symbols = if choices || minimization.is_some() {
        Some(symbolize::resolve(
            &opened,
            projected_choices
                .as_ref()
                .into_iter()
                .filter(|_| choices)
                .flat_map(choice_sites)
                .chain(
                    minimization_projection
                        .as_ref()
                        .into_iter()
                        .flat_map(crate::minimize::report::Projection::sites),
                ),
        )?)
    } else {
        None
    };
    let minimization = minimization
        .zip(minimization_projection.as_ref())
        .zip(symbols.as_ref())
        .map(|((metadata, projection), symbols)| projection.report(metadata, symbols))
        .transpose()?;
    let projected_choices = projected_choices
        .filter(|_| choices)
        .zip(symbols.as_ref())
        .map(|(choices, symbols)| symbolize_choices(choices, symbols));
    let manifest = opened.manifest();
    let report = Inspection {
        schema: "gomad.inspect/v9",
        kind: "artifact",
        path,
        artifact: Artifact {
            record: &manifest.record,
            record_sha256: &manifest.record_sha256,
            failure_signature: manifest.record.failure_signature()?,
            stored_bytes: opened.stored_bytes(),
            files: &manifest.files,
            stdout: Output::read(&opened, details.as_ref(), "stdout")?,
            stderr: Output::read(&opened, details.as_ref(), "stderr")?,
            choices: projected_choices,
            world: details.as_ref().and_then(|details| details.world.as_ref()),
            simulation: details
                .as_ref()
                .and_then(|details| details.simulation.as_ref()),
            minimization,
            storage: storage::read(&opened)?,
            cluster: cluster::read(&opened)?,
        },
    };
    let bytes = if json {
        canonical::encode(&report)?
    } else {
        let mut rendered = format!(
            "gomad inspect: kind=artifact path={}\nidentity: record={} execution={} seed={}\noutcome: class={} termination={} signature={}\nrecord: {}",
            path.display(), manifest.record_sha256, manifest.record.execution_identity,
            manifest.record.seed, manifest.record.outcome.class, manifest.record.outcome.termination,
            report.artifact.failure_signature, String::from_utf8(canonical::encode(&report.artifact)?).map_err(|error| error.to_string())?
        ).into_bytes();
        if let Some(cluster) = &report.artifact.cluster {
            cluster.append_text(&mut rendered)?;
        }
        rendered
    };
    output
        .write_all(&bytes)
        .and_then(|()| output.write_all(b"\n"))
        .map_err(|error| super::Error {
            status: 3,
            message: format!("write inspection report: {error}"),
        })
}
