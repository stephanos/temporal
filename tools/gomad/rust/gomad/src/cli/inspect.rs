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
        append_cluster_text(&opened, &mut rendered)?;
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

fn append_cluster_text(opened: &Opened, output: &mut Vec<u8>) -> Result<(), super::Error> {
    if !opened
        .manifest()
        .files
        .contains_key("evidence/cluster_summary")
    {
        return Ok(());
    }
    let raw = opened.payload("evidence/cluster_summary", 128 << 20)?;
    let summary: serde_json::Value = crate::qualify::spec::strict(&raw)?;
    if summary["schema"] != "gomad.cluster-observation/v1" {
        return Err("unsupported cluster observation".into());
    }
    let elapsed = summary["virtual_elapsed_nanos"]
        .as_u64()
        .ok_or("missing cluster elapsed time")?;
    writeln!(output, "cluster: virtual-elapsed-nanos={elapsed}")
        .map_err(|error| error.to_string())?;
    let record = &opened.manifest().record;
    if record.outcome.termination == "capacity"
        && let Some(bound) = record.outcome.reason.strip_prefix("WASI capacity: ")
    {
        writeln!(output, "exhausted={}", bound.escape_default())
            .map_err(|error| error.to_string())?;
    }
    for row in summary["incarnations"]
        .as_array()
        .ok_or("missing cluster instances")?
    {
        let node = row["handle"]["node"]
            .as_str()
            .ok_or("missing cluster node")?;
        let incarnation = row["handle"]["incarnation"]
            .as_u64()
            .ok_or("missing cluster incarnation")?;
        let (counts, tail) = cluster_output(opened, node, incarnation, &row["streams"])?;
        let tail = serde_json::to_string(&String::from_utf8_lossy(&tail))
            .map_err(|error| error.to_string())?;
        writeln!(output, "cluster-instance: node={} incarnation={incarnation} stdout-bytes={} stderr-bytes={} stdout-tail={tail}",
            node.escape_default(), counts[0], counts[1]).map_err(|error| error.to_string())?;
    }
    Ok(())
}

fn cluster_output(
    opened: &Opened,
    node: &str,
    incarnation: u64,
    streams: &serde_json::Value,
) -> Result<([u64; 2], Vec<u8>), String> {
    let streams = streams
        .as_object()
        .ok_or("missing cluster instance streams")?;
    let prefix = format!("evidence/cluster/{node}/{incarnation}/");
    let limits = &opened.manifest().record.limits;
    let mut evidence_bytes = 0_u64;
    let mut output_bytes = 0_u64;
    let mut counts = [0; 2];
    // run() already authenticated every manifest payload with the shared streaming verifier.
    for (name, descriptor) in streams {
        crate::evidence::store::open::components(name)?;
        let bytes = descriptor["bytes"]
            .as_u64()
            .ok_or("missing cluster stream byte count")?;
        evidence_bytes = evidence_bytes
            .checked_add(bytes)
            .ok_or("cluster evidence bytes overflow")?;
        if evidence_bytes > limits.evidence_bytes {
            return Err("cluster evidence exceeds retained evidence budget".into());
        }
        let output = match name.as_str() {
            "stdout" => Some(0),
            "stderr" => Some(1),
            _ => None,
        };
        let limit = if let Some(index) = output {
            output_bytes = output_bytes
                .checked_add(bytes)
                .ok_or("cluster output bytes overflow")?;
            if output_bytes > limits.output_bytes {
                return Err("cluster output exceeds retained output budget".into());
            }
            counts[index] = bytes;
            limits.output_bytes
        } else {
            limits.evidence_bytes
        };
        crate::replay::cluster::Payload {
            file: format!("{prefix}{name}"),
            bytes,
            sha256: descriptor["sha256"]
                .as_str()
                .ok_or("missing cluster stream digest")?
                .to_owned(),
            limit,
        }
        .bind(opened, limit)?;
    }
    if opened
        .manifest()
        .files
        .keys()
        .filter_map(|name| name.strip_prefix(&prefix))
        .any(|name| !streams.contains_key(name))
    {
        return Err("cluster instance output descriptor/payload mismatch".into());
    }
    let tail = if streams.contains_key("stdout") {
        opened.payload_tail(&format!("{prefix}stdout"), limits.output_bytes, 4096)?
    } else {
        Vec::new()
    };
    Ok((counts, tail))
}
