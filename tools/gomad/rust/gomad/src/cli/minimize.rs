use super::explore::{boolean, option, value};
use std::{io::Write, path::Path};

struct Arguments<'a> {
    path: &'a Path,
    artifacts: &'a Path,
    attempt_budget: u64,
    maximum_bytes: Option<u64>,
    json: bool,
    resume: bool,
}

/// # Errors
/// Reports malformed commands, incompatible artifacts and minimization failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let arguments = parse(arguments)?;
    let prepared = crate::replay::preflight::open(arguments.path)
        .map_err(|message| super::Error { status: 2, message })?;
    let directory = std::path::absolute(arguments.artifacts)
        .map_err(|error| format!("resolve artifact directory: {error}"))?
        .join("minimized");
    let result = crate::minimize::run(
        prepared,
        arguments.path,
        &directory,
        arguments.attempt_budget,
        arguments.maximum_bytes,
        arguments.resume,
    )
    .map_err(|message| super::Error { status: 3, message })?;
    if arguments.json {
        serde_json::to_writer(&mut *output, &result).map_err(|error| super::Error {
            status: 3,
            message: format!("encode minimization result: {error}"),
        })?;
        writeln!(output).map_err(|error| write_error(&error))?;
    } else {
        writeln!(
            output,
            "gomad minimize: changed={} attempts={}/{} accepted={} stop={} artifact={}",
            result["changed"].as_bool().unwrap_or(false),
            result["attempts"].as_u64().unwrap_or(0),
            result["attempt_budget"]
                .as_u64()
                .unwrap_or(arguments.attempt_budget),
            result["accepted"].as_array().map_or(0, Vec::len),
            result["stop_reason"].as_str().unwrap_or_default(),
            result["artifact_path"].as_str().unwrap_or_default(),
        )
        .map_err(|error| write_error(&error))?;
    }
    Ok(0)
}

fn parse(arguments: &[String]) -> Result<Arguments<'_>, super::Error> {
    let mut result = Arguments {
        path: Path::new(""),
        artifacts: Path::new(".gomad/artifacts"),
        attempt_budget: 64,
        maximum_bytes: None,
        json: false,
        resume: false,
    };
    let mut index = 0;
    while let Some((name, supplied)) = option(arguments, &mut index) {
        let name = name?;
        match name {
            "json" => result.json = boolean(supplied.unwrap_or("true"))?,
            "resume" => result.resume = boolean(supplied.unwrap_or("true"))?,
            "artifacts" => {
                result.artifacts = Path::new(value(arguments, &mut index, name, supplied)?);
            }
            "attempt-budget" => {
                let number = value(arguments, &mut index, name, supplied)?;
                result.attempt_budget = number
                    .parse()
                    .map_err(|_| format!("invalid value {number:?} for -attempt-budget"))?;
            }
            "max-bytes" => {
                result.maximum_bytes =
                    Some(byte_size(value(arguments, &mut index, name, supplied)?)?);
            }
            _ => return Err(format!("flag provided but not defined: -{name}").into()),
        }
    }
    if arguments.len() - index != 1 || result.attempt_budget == 0 {
        return Err("usage: gomad minimize [--json] [--resume] [--attempt-budget N] [--artifacts DIR] ARTIFACT_DIR".into());
    }
    result.path = Path::new(&arguments[index]);
    Ok(result)
}

fn byte_size(value: &str) -> Result<u64, super::Error> {
    let (number, factor) = [("KiB", 1_u64 << 10), ("MiB", 1 << 20), ("GiB", 1 << 30)]
        .into_iter()
        .find_map(|(suffix, factor)| value.strip_suffix(suffix).map(|number| (number, factor)))
        .unwrap_or((value, 1));
    if number.len() > 1 && number.starts_with('0') {
        return Err(format!("invalid byte size {value:?}").into());
    }
    number
        .parse::<u64>()
        .ok()
        .and_then(|number| number.checked_mul(factor))
        .filter(|number| *number > 0)
        .ok_or_else(|| format!("invalid byte size {value:?}").into())
}

fn write_error(error: &std::io::Error) -> super::Error {
    super::Error {
        status: 3,
        message: format!("write minimization report: {error}"),
    }
}
