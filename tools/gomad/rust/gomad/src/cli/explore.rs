use std::{
    collections::BTreeSet,
    io::Write,
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
#[expect(
    clippy::struct_excessive_bools,
    reason = "Independent original CLI switches retain their separate meanings"
)]
pub struct ExploreOptions {
    pub installation: Option<PathBuf>,
    pub seeds: String,
    pub parallel: usize,
    pub execution_timeout: Duration,
    pub overall_timeout: Duration,
    pub terminate_grace: Duration,
    pub on_failure: String,
    pub failure_budget: u64,
    pub artifacts: PathBuf,
    pub json: bool,
    pub choices: bool,
    pub diagnostics: bool,
    pub coverage: String,
    pub keep_successes: String,
    pub success_limit: u64,
    pub success_bytes: u64,
    pub output_limit: u64,
    pub world_transition_limit: u64,
    pub choice_bytes: u64,
    pub environment: Vec<String>,
    pub build_tags: Vec<String>,
    pub soft_assertion_report_only: bool,
    pub working_dir: PathBuf,
    pub target_kind: String,
    pub source: String,
    pub provenance: String,
    pub target_args: Vec<String>,
    pub scenario: Option<Vec<u8>>,
}

/// # Errors
/// Reports malformed campaign requests and execution failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    run_with_installation(arguments, output, None)
}

/// # Errors
/// Reports malformed campaign requests and execution failures.
pub fn run_with_installation(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    if let Some(result) = super::exploration::try_run(arguments, output, installation) {
        return result;
    }
    let mut json = false;
    let mut options = match parse(arguments, &mut json) {
        Ok(options) => options,
        Err(error) => return report_error(&error, json, output),
    };
    options.installation = installation.map(Path::to_path_buf);
    match crate::campaign::explore(&options, output) {
        Ok(status) => Ok(status),
        Err(error) => report_error(&error, json, output),
    }
}

pub(crate) fn parse(arguments: &[String], json: &mut bool) -> Result<ExploreOptions, String> {
    parse_for_strategy(arguments, json, false)
}

// Original standalone measurement requests permit a 1 GiB choice trace.
// The ordinary campaign CLI keeps its independent 64 MiB admission bound.
pub(crate) fn parse_measurement(
    arguments: &[String],
    json: &mut bool,
) -> Result<ExploreOptions, String> {
    parse_with_choice_maximum(arguments, json, false, 1 << 30)
}
pub(super) fn parse_for_strategy(
    arguments: &[String],
    json: &mut bool,
    simulation: bool,
) -> Result<ExploreOptions, String> {
    parse_with_choice_maximum(arguments, json, simulation, 64 << 20)
}
#[expect(
    clippy::too_many_lines,
    reason = "One parser preserves the original ordered flag and target grammar"
)]
fn parse_with_choice_maximum(
    arguments: &[String],
    json: &mut bool,
    simulation: bool,
    choice_maximum: u64,
) -> Result<ExploreOptions, String> {
    let mut options = ExploreOptions {
        installation: None,
        seeds: "1".into(),
        parallel: std::thread::available_parallelism()
            .map_or(1, usize::from)
            .min(8),
        execution_timeout: Duration::from_secs(30),
        overall_timeout: Duration::from_secs(600),
        terminate_grace: Duration::from_secs(2),
        on_failure: "first".into(),
        failure_budget: 1,
        artifacts: ".gomad/artifacts".into(),
        json: false,
        choices: false,
        diagnostics: false,
        coverage: "none".into(),
        keep_successes: "none".into(),
        success_limit: 0,
        success_bytes: 0,
        output_limit: 8 << 20,
        world_transition_limit: 64 << 20,
        choice_bytes: 8 << 20,
        environment: Vec::new(),
        build_tags: Vec::new(),
        soft_assertion_report_only: false,
        working_dir: PathBuf::new(),
        target_kind: String::new(),
        source: String::new(),
        provenance: String::new(),
        target_args: Vec::new(),
        scenario: None,
    };
    let mut seeds_set = false;
    let mut count = None;
    let mut choice_bytes_set = false;
    let mut scenario = None;
    let mut index = 0;
    while let Some((name, supplied)) = option(arguments, &mut index) {
        let name = name?;
        let boolean_flag = matches!(
            name,
            "json" | "choices" | "diagnostics" | "soft-assertion-report-only"
        );
        if boolean_flag {
            let value = boolean(supplied.unwrap_or("true"))?;
            match name {
                "json" => {
                    *json = value;
                    options.json = value;
                }
                "choices" => options.choices = value,
                "diagnostics" => options.diagnostics = value,
                _ => options.soft_assertion_report_only = value,
            }
            continue;
        }
        if !matches!(
            name,
            "strategy"
                | "seeds"
                | "count"
                | "parallel"
                | "execution-timeout"
                | "overall-timeout"
                | "terminate-grace"
                | "on-failure"
                | "failure-budget"
                | "artifacts"
                | "coverage"
                | "keep-successes"
                | "success-limit"
                | "output-limit"
                | "world-transition-limit"
                | "success-bytes"
                | "choice-bytes"
                | "env"
                | "build-tag"
                | "working-dir"
                | "scenario"
                | "max-executions"
                | "max-choice-depth"
                | "choice-start-ordinal"
                | "site-scope"
                | "max-exploration-bytes"
        ) {
            return Err(format!("flag provided but not defined: -{name}"));
        }
        let value = value(arguments, &mut index, name, supplied)?;
        match name {
            "strategy" if value != "seed" => {
                return Err("this command requires the seed strategy".into());
            }
            "strategy" => {}
            "seeds" => {
                options.seeds = value.into();
                seeds_set = true;
            }
            "count" => count = Some(unsigned(value)?),
            "parallel" => {
                options.parallel = value
                    .parse::<usize>()
                    .map_err(|_| "parallelism must be positive")?;
            }
            "execution-timeout" => options.execution_timeout = duration(value)?,
            "overall-timeout" => options.overall_timeout = duration(value)?,
            "terminate-grace" => options.terminate_grace = duration(value)?,
            "on-failure" => options.on_failure = value.into(),
            "failure-budget" => options.failure_budget = unsigned(value)?,
            "artifacts" => options.artifacts = value.into(),
            "coverage" if !matches!(value, "none" | "choice") => {
                return Err("coverage must be one of none, choice".into());
            }
            "coverage" => options.coverage = value.into(),
            "keep-successes" => options.keep_successes = value.into(),
            "success-limit" => options.success_limit = unsigned(value)?,
            "output-limit" => options.output_limit = bytes(value)?,
            "world-transition-limit" => options.world_transition_limit = bytes(value)?,
            "success-bytes" => options.success_bytes = bytes(value)?,
            "choice-bytes" => {
                options.choice_bytes = bytes(value)?;
                choice_bytes_set = true;
            }
            "env" => options.environment.push(value.into()),
            "build-tag" => options.build_tags.push(value.into()),
            "working-dir" => options.working_dir = value.into(),
            "scenario" => scenario = Some(value.to_owned()),
            "choice-start-ordinal" => {
                return Err("--choice-start-ordinal requires --strategy=choice-exploration".into());
            }
            "site-scope" => {
                return Err("--site-scope requires --strategy=choice-exploration".into());
            }
            _ => return Err("exploration bounds require --strategy=choice-exploration".into()),
        }
    }
    if let Some(count) = count {
        if seeds_set {
            return Err("--count and --seeds are mutually exclusive".into());
        }
        if count == 0 {
            return Err("--count must be greater than zero".into());
        }
        options.seeds = if count == 1 {
            "0".into()
        } else {
            format!("0-{}", count - 1)
        };
    }
    crate::campaign::seeds::SeedSelection::parse(&options.seeds)?;
    options.choices |= options.diagnostics;
    if !options.choices {
        if choice_bytes_set {
            return Err("--choice-bytes requires --choices".into());
        }
        options.choice_bytes = 0;
    } else if !(160..=choice_maximum).contains(&options.choice_bytes) {
        return Err(if choice_maximum == 64 << 20 {
            "--choice-bytes must be between 160 bytes and 64MiB"
        } else {
            "measurement --choice-bytes must be between 160 bytes and 1GiB"
        }
        .into());
    }
    if options.coverage == "choice" && !options.choices {
        return Err("--coverage=choice requires --choices".into());
    }
    parse_target(&arguments[index..], &mut options)?;
    validate(&options)?;
    if let Some(path) = scenario {
        if options.choices && !simulation || options.coverage != "none" {
            return Err("--scenario requires seed exploration without choices, diagnostics, coverage, or guidance".into());
        }
        let file = std::fs::File::open(path).map_err(|error| format!("open scenario: {error}"))?;
        let metadata = file
            .metadata()
            .map_err(|error| format!("read scenario metadata: {error}"))?;
        if !metadata.is_file() || metadata.len() > 256 << 10 {
            return Err("scenario must be a regular JSON file of at most 256 KiB".into());
        }
        let mut bytes = Vec::new();
        std::io::Read::read_to_end(&mut std::io::Read::take(file, (256 << 10) + 1), &mut bytes)
            .map_err(|error| format!("read scenario: {error}"))?;
        if bytes.len() > 256 << 10 || serde_json::from_slice::<serde_json::Value>(&bytes).is_err() {
            return Err("scenario must contain bounded valid JSON".into());
        }
        options.scenario = Some(bytes);
    }
    if options.working_dir.as_os_str().is_empty() {
        options.working_dir = std::env::current_dir()
            .map_err(|error| format!("runner_failure: resolve working directory: {error}"))?;
    } else {
        validate_directory(&options.working_dir)?;
    }
    Ok(options)
}

fn parse_target(arguments: &[String], options: &mut ExploreOptions) -> Result<(), String> {
    let kind = arguments.first().ok_or("target kind is required")?;
    options.target_kind.clone_from(kind);
    match kind.as_str() {
        "go-test" | "go-run" => {
            options.source.clone_from(
                arguments
                    .get(1)
                    .filter(|value| !value.is_empty())
                    .ok_or_else(|| format!("{kind} requires one package"))?,
            );
            if arguments.len() > 2 {
                if arguments[2] != "--" {
                    return Err(format!("{kind} target arguments require -- separator"));
                }
                options.target_args = arguments[3..].to_vec();
            }
        }
        "exec" => {
            if arguments.len() < 5
                || arguments[1] != "--provenance"
                || arguments[2].is_empty()
                || arguments[3] != "--"
                || arguments[4].is_empty()
            {
                return Err("exec requires --provenance FILE -- BINARY [ARG ...]".into());
            }
            options.provenance.clone_from(&arguments[2]);
            options.source.clone_from(&arguments[4]);
            options.target_args = arguments[5..].to_vec();
        }
        _ => return Err(format!("unknown target kind {kind:?}")),
    }
    Ok(())
}

fn validate(options: &ExploreOptions) -> Result<(), String> {
    if options.parallel == 0 {
        return Err("parallelism must be positive".into());
    }
    if options.execution_timeout.is_zero() || options.overall_timeout.is_zero() {
        return Err("execution and overall timeouts must be positive".into());
    }
    if options.terminate_grace > options.execution_timeout
        || options.terminate_grace > options.overall_timeout
    {
        return Err("termination grace must fit inside all deadlines".into());
    }
    if options.artifacts.as_os_str().is_empty() {
        return Err("artifact root is required".into());
    }
    match options.on_failure.as_str() {
        "first" | "all" if options.failure_budget != 1 => {
            return Err("failure budget is only configurable in budget mode".into());
        }
        "budget" if options.failure_budget == 0 => {
            return Err("failure budget must be positive".into());
        }
        "first" | "all" | "budget" => {}
        value => return Err(format!("unknown failure policy {value:?}")),
    }
    match options.keep_successes.as_str() {
        "none" if options.success_limit != 0 || options.success_bytes != 0 => {
            return Err("disabled success retention does not accept capacity limits".into());
        }
        "novel"
            if options.coverage == "none"
                || options.success_limit == 0
                || options.success_bytes == 0 =>
        {
            return Err(
                "novel success retention requires coverage and explicit count and byte limits"
                    .into(),
            );
        }
        "all" if options.success_limit == 0 || options.success_bytes == 0 => {
            return Err("success retention requires explicit count and byte limits".into());
        }
        "none" | "novel" | "all" => {}
        value => {
            return Err(format!(
                "unknown successful-execution retention policy {value:?}"
            ));
        }
    }
    let mut seen = BTreeSet::new();
    for entry in &options.environment {
        let (name, value) = entry
            .split_once('=')
            .ok_or_else(|| format!("invalid target environment entry {entry:?}"))?;
        if name.is_empty()
            || !name.bytes().enumerate().all(|(index, byte)| {
                byte.is_ascii_alphabetic() || byte == b'_' || index > 0 && byte.is_ascii_digit()
            })
            || value.contains('\0')
        {
            return Err(format!("invalid target environment entry {entry:?}"));
        }
        if reserved(name) {
            return Err(format!("target environment name {name:?} is reserved"));
        }
        if !seen.insert(name) {
            return Err(format!("duplicate target environment name {name:?}"));
        }
    }
    Ok(())
}

fn reserved(name: &str) -> bool {
    name.starts_with("LD_")
        || name.starts_with("DYLD_")
        || matches!(
            name,
            "GOMAD_DIAGNOSTIC_PROFILE"
                | "GOMAD_DIAGNOSTIC_TRACE_FD"
                | "GOMAD_DIAGNOSTIC_TRACE_BYTES"
                | "GOMAD_DIAGNOSTIC_PERTURB_DRAW"
                | "GOMADSEED"
                | "GOMAD_CHILD_SEED"
                | "GOMAD_IO_PROFILE"
                | "GOMAD_CLOCK_TICK"
                | "GOMAD_CHOICE_PROFILE"
                | "GOMAD_CHOICE_MODE"
                | "GOMAD_CHOICE_TRACE_FD"
                | "GOMAD_CHOICE_TERMINAL_FD"
                | "GOMAD_CHOICE_TRACE_BYTES"
                | "GOMAD_CHOICE_TAPE_FD"
                | "GOMAD_CHOICE_TAPE_BYTES"
                | "GOMAD_SIMULATION_ROLE"
                | "GOMAD_SIMULATION_REQUEST_FD"
                | "GOMAD_SIMULATION_RESPONSE_FD"
                | "GOMAD_SIMULATION_BOOTSTRAP_FD"
                | "GOMAD_SIMULATION_CONTROL_FD"
                | "TZ"
                | "CGO_ENABLED"
                | "GODEBUG"
                | "GOMAXPROCS"
                | "GOEXPERIMENT"
                | "LIBPATH"
                | "SHLIB_PATH"
        )
}

fn validate_directory(path: &Path) -> Result<(), String> {
    let text = path.to_string_lossy();
    if !path.is_absolute()
        || text.contains("//")
        || text.ends_with('/') && text != "/"
        || text.split('/').any(|part| matches!(part, "." | ".."))
    {
        return Err(format!(
            "working directory {text:?} must be an absolute, clean path"
        ));
    }
    if !path.is_dir() {
        return Err(format!("working directory {text:?} is not a directory"));
    }
    if !path.join("go.mod").is_file() {
        return Err(format!(
            "working directory {text:?} is not a module root: it has no regular go.mod"
        ));
    }
    Ok(())
}

pub(super) fn option<'a>(
    arguments: &'a [String],
    index: &mut usize,
) -> Option<(Result<&'a str, String>, Option<&'a str>)> {
    let argument = arguments.get(*index)?;
    if argument == "--" {
        *index += 1;
        return None;
    }
    if !argument.starts_with('-') || argument == "-" {
        return None;
    }
    *index += 1;
    let flag = argument
        .strip_prefix("--")
        .or_else(|| argument.strip_prefix('-'))
        .unwrap_or(argument);
    let (name, value) = flag
        .split_once('=')
        .map_or((flag, None), |(name, value)| (name, Some(value)));
    Some((Ok(name), value))
}

pub(super) fn value<'a>(
    arguments: &'a [String],
    index: &mut usize,
    name: &str,
    supplied: Option<&'a str>,
) -> Result<&'a str, String> {
    if let Some(value) = supplied {
        return Ok(value);
    }
    let value = arguments
        .get(*index)
        .ok_or_else(|| format!("flag needs an argument: -{name}"))?;
    *index += 1;
    Ok(value)
}

pub(super) fn boolean(value: &str) -> Result<bool, String> {
    match value {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Ok(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Ok(false),
        _ => Err(format!("invalid boolean value {value:?}")),
    }
}

fn unsigned(value: &str) -> Result<u64, String> {
    if value.is_empty() || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err(format!("invalid unsigned integer {value:?}"));
    }
    value
        .parse()
        .map_err(|_| format!("invalid unsigned integer {value:?}"))
}

fn bytes(value: &str) -> Result<u64, String> {
    let (number, factor) = [("KiB", 1_u64 << 10), ("MiB", 1 << 20), ("GiB", 1 << 30)]
        .into_iter()
        .find_map(|(suffix, factor)| value.strip_suffix(suffix).map(|number| (number, factor)))
        .unwrap_or((value, 1));
    if number.len() > 1 && number.starts_with('0') {
        return Err(format!("invalid byte size {value:?}"));
    }
    unsigned(number)?
        .checked_mul(factor)
        .filter(|value| *value > 0)
        .ok_or_else(|| format!("invalid byte size {value:?}"))
}

fn duration(value: &str) -> Result<Duration, String> {
    if matches!(value, "0" | "+0" | "-0") {
        return Ok(Duration::ZERO);
    }
    let mut remaining = value.strip_prefix('+').unwrap_or(value);
    let mut nanoseconds = 0_u64;
    while !remaining.is_empty() {
        let length = remaining
            .bytes()
            .take_while(|byte| byte.is_ascii_digit() || *byte == b'.')
            .count();
        let number = &remaining[..length];
        let tail = &remaining[length..];
        let (unit, multiplier) = [
            ("ns", 1_u64),
            ("us", 1_000),
            ("µs", 1_000),
            ("μs", 1_000),
            ("ms", 1_000_000),
            ("s", 1_000_000_000),
            ("m", 60_000_000_000),
            ("h", 3_600_000_000_000),
        ]
        .into_iter()
        .find(|(unit, _)| tail.starts_with(unit))
        .ok_or_else(|| format!("invalid duration {value:?}"))?;
        let (whole, fraction) = number.split_once('.').unwrap_or((number, ""));
        if whole.is_empty() && fraction.is_empty()
            || !fraction.bytes().all(|byte| byte.is_ascii_digit())
        {
            return Err(format!("invalid duration {value:?}"));
        }
        let whole = if whole.is_empty() {
            0
        } else {
            unsigned(whole)?
        };
        let mut fraction_nanos = 0_u128;
        let mut denominator = 1_u128;
        for digit in fraction.bytes().take(19) {
            fraction_nanos = fraction_nanos * 10 + u128::from(digit - b'0');
            denominator *= 10;
        }
        let nanos = u128::from(whole) * u128::from(multiplier)
            + fraction_nanos * u128::from(multiplier) / denominator;
        nanoseconds = nanoseconds
            .checked_add(u64::try_from(nanos).map_err(|_| format!("invalid duration {value:?}"))?)
            .filter(|total| i64::try_from(*total).is_ok())
            .ok_or_else(|| format!("invalid duration {value:?}"))?;
        remaining = &tail[unit.len()..];
    }
    if value.is_empty() || value == "+" {
        return Err(format!("invalid duration {value:?}"));
    }
    Ok(Duration::from_nanos(nanoseconds))
}

pub(super) fn report_error(
    error: &str,
    json: bool,
    output: &mut dyn Write,
) -> Result<i32, super::Error> {
    let (classification, message) = error
        .split_once(": ")
        .filter(|(name, _)| {
            matches!(
                *name,
                "runner_failure"
                    | "capacity"
                    | "unsupported_target"
                    | "cancelled"
                    | "overall_timeout"
                    | "semantic_coverage_failure"
                    | "invalid_input"
            )
        })
        .unwrap_or(("invalid_input", error));
    let status = match classification {
        "semantic_coverage_failure" => 1,
        "invalid_input" | "unsupported_target" => 2,
        _ => 3,
    };
    let result = if json {
        writeln!(
            output,
            "{}",
            serde_json::json!({"schema":"gomad.explore-event/v3", "type":"error", "classification":classification, "message":message})
        )
    } else {
        writeln!(output, "gomad: {classification}: {message}")
    };
    result.map_err(|error| super::Error {
        status: 3,
        message: format!("write campaign report: {error}"),
    })?;
    Ok(status)
}
