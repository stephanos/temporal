use crate::evidence::inspection::{Choices, Details, Minimization, Output, Simulation, World};
use crate::evidence::symbolize::{self, Site, Symbol, Symbols};
use crate::evidence::{canonical, record::Record, store::Opened};
use gomad_wasmhost::linked;
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
    minimization: Option<&'a Minimization>,
}

#[derive(Serialize)]
struct Symbolized<'a, T> {
    #[serde(flatten)]
    value: T,
    symbol: &'a Symbol,
}

#[derive(Serialize)]
struct Trace<'a> {
    records: u64,
    decisions: u64,
    branching_records: u64,
    runnable: u64,
    select_poll: u64,
    select_result: u64,
    sites: Vec<Symbolized<'a, linked::TraceSite>>,
    replay_decisions: Vec<Symbolized<'a, linked::TraceDecision>>,
    tape_sha256: String,
}

fn choice_symbols(choices: &Choices, opened: &Opened) -> Result<Symbols, String> {
    symbolize::resolve(
        opened,
        choices
            .trace
            .sites
            .iter()
            .map(|site| Site {
                value: site.site_offset,
                missing: site.site_missing,
            })
            .chain(choices.trace.replay_decisions.iter().map(|decision| Site {
                value: decision.site_offset,
                missing: decision.site_missing,
            })),
    )
}

fn symbolize_choices(choices: Choices, symbols: &Symbols) -> Choices<Trace<'_>> {
    let trace = choices.trace;
    Choices {
        schema: choices.schema,
        profile: choices.profile,
        implementation_sha256: choices.implementation_sha256,
        limit: choices.limit,
        payload_bytes: choices.payload_bytes,
        sha256: choices.sha256,
        terminal_state: choices.terminal_state,
        exact_replay_available: choices.exact_replay_available,
        trace: Trace {
            records: trace.records,
            decisions: trace.decisions,
            branching_records: trace.branching_records,
            runnable: trace.runnable,
            select_poll: trace.select_poll,
            select_result: trace.select_result,
            sites: trace
                .sites
                .into_iter()
                .map(|value| Symbolized {
                    symbol: symbols.get(Site {
                        value: value.site_offset,
                        missing: value.site_missing,
                    }),
                    value,
                })
                .collect(),
            replay_decisions: trace
                .replay_decisions
                .into_iter()
                .map(|value| Symbolized {
                    symbol: symbols.get(Site {
                        value: value.site_offset,
                        missing: value.site_missing,
                    }),
                    value,
                })
                .collect(),
            tape_sha256: trace.tape_sha256,
        },
    }
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

/// # Errors
/// Refuses invalid flags, unsupported artifacts and unauthenticated retained evidence.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<(), super::Error> {
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
    let path = path.as_path();
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
    let symbols = projected_choices
        .as_ref()
        .filter(|_| choices)
        .map(|choices| choice_symbols(choices, &opened))
        .transpose()?;
    let projected_choices = projected_choices
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
            minimization: details
                .as_ref()
                .and_then(|details| details.minimization.as_ref()),
        },
    };
    let bytes = if json {
        canonical::encode(&report)?
    } else {
        format!(
            "gomad inspect: kind=artifact path={}\nidentity: record={} execution={} seed={}\noutcome: class={} termination={} signature={}\nrecord: {}",
            path.display(), manifest.record_sha256, manifest.record.execution_identity,
            manifest.record.seed, manifest.record.outcome.class, manifest.record.outcome.termination,
            report.artifact.failure_signature, String::from_utf8(canonical::encode(&report.artifact)?).map_err(|error| error.to_string())?
        ).into_bytes()
    };
    output
        .write_all(&bytes)
        .and_then(|()| output.write_all(b"\n"))
        .map_err(|error| super::Error {
            status: 3,
            message: format!("write inspection report: {error}"),
        })
}
