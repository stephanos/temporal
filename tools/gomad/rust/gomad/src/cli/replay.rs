use crate::{
    evidence::canonical,
    replay::{divergence, inspection, preflight},
};
use std::{io::Write, path::Path};

fn incompatible(message: &str) -> super::Error {
    super::Error {
        status: 2,
        message: format!("incompatible replay artifact: {message}"),
    }
}

/// # Errors
/// Refuses malformed commands or incompatible artifacts; execution errors retain status 3.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    run_with_installation(arguments, output, None)
}

/// # Errors
/// Refuses incompatible explicitly selected installations after retained input admission.
pub fn run_with_installation(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    let arguments = parse(arguments)?;
    let inspection = arguments.inspection.request(arguments.path)?;
    if inspection.is_some() && (arguments.verify || arguments.observed_directory.is_some()) {
        return Err("inspection cannot combine with --verify-only or --observed".into());
    }
    if arguments.determinism_fixture && (arguments.stress_jitter || inspection.is_none()) {
        return Err("--determinism-fixture requires replay inspection with --stop and cannot combine with --stress-jitter".into());
    }
    if arguments.stress_jitter && inspection.is_none() {
        return Err("--stress-jitter requires replay inspection with --stop".into());
    }
    let mut prepared = preflight::open(arguments.path).map_err(|error| incompatible(&error))?;
    if matches!(&prepared.execution, preflight::Execution::Cluster(_))
        && (arguments.verify || arguments.observed_directory.is_some())
    {
        return Err(incompatible(
            "retained WASM cluster replay does not support --verify-only, --observed, or --toolchain-root",
        ));
    }
    if let Some(path) = installation {
        let description = crate::install::manifest::resolve(Some(path))
            .map_err(|message| super::Error { status: 3, message })?;
        validate_installation(&description.manifest, &prepared.opened)
            .map_err(|message| super::Error { status: 3, message })?;
        prepared
            .compile_cache(
                crate::campaign::prepare::compile_cache_root(&description.manifest)
                    .map_err(|message| super::Error { status: 3, message })?
                    .as_deref(),
            )
            .map_err(|message| super::Error { status: 3, message })?;
    }
    if let Some(mut request) = inspection {
        request.conditions = crate::replay::conditions::select(
            &prepared.opened.manifest().record.profile,
            arguments.stress_jitter,
            arguments.determinism_fixture,
        )?;
        return inspection::run(&prepared, &request, output);
    }
    if arguments.verify {
        writeln!(output, "gomad: verified {}", arguments.path.display())
            .map_err(|error| write_error(&error))?;
        return Ok(0);
    }
    run_prepared(prepared, arguments.observed_directory, output)
}

pub(crate) fn validate_installation(
    installed: &crate::install::manifest::Installation,
    opened: &crate::evidence::store::Opened,
) -> Result<(), String> {
    let mismatch = || {
        "runner_failure: selected installation differs from retained implementation context"
            .to_owned()
    };
    let file = if opened
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE)
    {
        crate::replay::cluster::FILE
    } else {
        preflight::FILE
    };
    let bytes = opened.payload(file, 128 << 20)?;
    let inputs: serde_json::Value = crate::qualify::spec::strict(&bytes)?;
    if installed.profile != opened.manifest().record.profile
        || inputs["host_sha256"] != installed.host_sha256
        || inputs["helper_sha256"] != installed.helper_sha256
        || inputs["runtime_implementation_sha256"]
            != serde_json::json!(installed.runtime_implementation_sha256)
    {
        return Err(mismatch());
    }
    if opened
        .manifest()
        .files
        .contains_key("target/provenance.json")
    {
        let bytes = opened.payload("target/provenance.json", 128 << 20)?;
        let provenance: serde_json::Value = crate::qualify::spec::strict(&bytes)?;
        if provenance["inputs"]["compiler_sha256"] != installed.compiler_sha256
            || provenance["inputs"]["runtime_implementation"]
                != serde_json::json!(installed.runtime_implementation_sha256)
        {
            return Err(mismatch());
        }
    }
    Ok(())
}

struct Arguments<'a> {
    verify: bool,
    observed_directory: Option<&'a Path>,
    path: &'a Path,
    inspection: inspection::Options,
    stress_jitter: bool,
    determinism_fixture: bool,
}

fn boolean(value: &str) -> Result<bool, super::Error> {
    match value {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Ok(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Ok(false),
        _ => Err(format!("invalid boolean replay option: {value}").into()),
    }
}

fn parse(arguments: &[String]) -> Result<Arguments<'_>, super::Error> {
    let mut verify = false;
    let mut observed_directory = None;
    let mut inspection = inspection::Options::default();
    let mut stress_jitter = false;
    let mut determinism_fixture = false;
    let mut path = None;
    let mut positional = false;
    let mut arguments = arguments.iter();
    while let Some(argument) = arguments.next() {
        if !positional && argument == "--" {
            positional = true;
        } else if !positional && argument.starts_with('-') {
            let option = argument
                .strip_prefix("--")
                .or_else(|| argument.strip_prefix('-'))
                .ok_or("invalid replay option")?;
            let (name, supplied) = option
                .split_once('=')
                .map_or((option, None), |(name, value)| (name, Some(value)));
            if matches!(
                name,
                "verify-only" | "stress-jitter" | "determinism-fixture"
            ) {
                let value = boolean(supplied.unwrap_or("true"))?;
                match name {
                    "verify-only" => verify = value,
                    "stress-jitter" => stress_jitter = value,
                    _ => determinism_fixture = value,
                }
                continue;
            }
            if name == "toolchain-root" {
                return Err("flag provided but not defined: -toolchain-root".into());
            }
            let value = supplied
                .or_else(|| arguments.next().map(String::as_str))
                .ok_or_else(|| format!("flag needs an argument: -{name}"))?;
            if name == "observed" {
                observed_directory = (!value.is_empty()).then(|| Path::new(value));
            } else if !inspection.set(name, value)? {
                return Err(format!("unknown replay option: {argument}").into());
            }
        } else if path.replace(Path::new(argument)).is_some() {
            return Err("usage: gomad replay [--verify-only] ARTIFACT".into());
        } else {
            positional = true;
        }
    }
    let path = path.ok_or("usage: gomad replay [--verify-only] ARTIFACT")?;
    Ok(Arguments {
        verify,
        observed_directory,
        path,
        inspection,
        stress_jitter,
        determinism_fixture,
    })
}

pub(crate) fn run_prepared(
    prepared: preflight::Prepared,
    observed_directory: Option<&Path>,
    output: &mut dyn Write,
) -> Result<i32, super::Error> {
    let record = &prepared.opened.manifest().record;
    if !prepared.available {
        writeln!(output, "gomad: reproduced=false divergence=choice_profile.replay_unavailable choice-replay=unavailable").map_err(|error| write_error(&error))?;
        return Ok(1);
    }
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::execution_started()?;
    let observed = preflight::execute(prepared.execution).map_err(|error| super::Error {
        status: if error.starts_with("replay preflight ") {
            2
        } else {
            3
        },
        message: format!("execute replay target: {error}"),
    })?;
    if observed.outcome.termination == "infrastructure"
        || observed.outcome.termination == "capacity"
            && observed.outcome.message == "compile cache artifact capacity"
    {
        return Err(super::Error {
            status: 3,
            message: format!("execute replay target: {}", observed.outcome.message),
        });
    }
    if let Some(directory) = observed_directory {
        std::fs::create_dir_all(directory).map_err(|error| write_error(&error))?;
        for name in ["stdout", "stderr"] {
            std::fs::write(
                directory.join(name),
                observed
                    .evidence
                    .get(name)
                    .map_or(&[] as &[u8], Vec::as_slice),
            )
            .map_err(|error| write_error(&error))?;
        }
        std::fs::write(
            directory.join("backend-evidence.bin"),
            canonical::encode(&observed.outcome)?,
        )
        .map_err(|error| write_error(&error))?;
    }
    let divergence = divergence::first(record, &observed);
    let choice = if observed.outcome.divergence.is_some() {
        "diverged"
    } else if prepared.exact {
        "exact"
    } else {
        "none"
    };
    if let Some(divergence) = divergence {
        writeln!(
            output,
            "gomad: reproduced=false divergence={divergence} choice-replay={choice}"
        )
        .map_err(|error| write_error(&error))?;
        return Ok(1);
    }
    let success = record.outcome.class == "success";
    writeln!(
        output,
        "gomad: reproduced=true diagnostic=false result={} choice-replay={choice}",
        if success { "success" } else { "target_failure" }
    )
    .map_err(|error| write_error(&error))?;
    Ok(i32::from(!success))
}

fn write_error(error: &std::io::Error) -> super::Error {
    super::Error {
        status: 3,
        message: format!("write replay report: {error}"),
    }
}
