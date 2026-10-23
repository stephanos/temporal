use super::preflight;

#[path = "inspection_search.rs"]
mod search;
use crate::{cli, evidence::store::Opened};
use gomad_wasmhost::{linked, linked_inspection as helper};
use serde::Serialize;
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs::{self, OpenOptions},
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Component, Path, PathBuf},
    time::{Duration, Instant},
};

#[derive(Default)]
pub struct Options {
    values: BTreeMap<String, String>,
    dumps: Vec<String>,
    probes: Vec<String>,
}

#[derive(Serialize)]
pub struct InspectionRequest {
    #[serde(skip_serializing_if = "String::is_empty")]
    pub instance: String,
    #[serde(skip_serializing_if = "is_zero")]
    pub incarnation: u64,
    pub stop: Stop,
    pub dumps: Vec<String>,
    pub probes: Vec<String>,
    pub maximum_bytes: u64,
    pub per_dump_bytes: u64,
    pub output_directory: PathBuf,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub search: Option<Search>,
    #[serde(skip)]
    artifact: PathBuf,
    #[serde(skip)]
    pub conditions: linked::Conditions,
}
#[expect(
    clippy::trivially_copy_pass_by_ref,
    reason = "Serde skip_serializing_if requires a borrowed field"
)]
fn is_zero(value: &u64) -> bool {
    *value == 0
}

#[derive(Serialize)]
pub struct Stop {
    pub kind: String,
    pub value: u64,
}

#[derive(Serialize)]
pub struct Search {
    pub check: String,
    pub first: u64,
    pub last: u64,
    pub mode: String,
    pub attempt_budget: u64,
    pub wall_time: u64,
}

impl Options {
    /// # Errors
    /// Parses the shipped inspection flag values before artifact admission.
    pub fn set(&mut self, name: &str, value: &str) -> Result<bool, String> {
        match name {
            "dump" => self.dumps.push(value.into()),
            "probe" => self.probes.push(value.into()),
            "stop" | "instance" | "incarnation" | "debug-output" | "debug-bytes" | "dump-bytes"
            | "check" | "choice-range" | "search-mode" | "search-attempts" | "search-timeout" => {
                self.values.insert(name.into(), value.into());
            }
            _ => return Ok(false),
        }
        if matches!(name, "incarnation" | "search-attempts") {
            unsigned(value)?;
        }
        if matches!(name, "debug-bytes" | "dump-bytes") {
            byte_size(value)?;
        }
        if name == "search-timeout" {
            duration(value)?;
        }
        Ok(true)
    }

    /// # Errors
    /// Resolves default selections and refuses incompatible or unbounded requests.
    #[expect(
        clippy::too_many_lines,
        reason = "Ordered admission keeps the shipped flag defaults and incompatibility checks together"
    )]
    pub fn request(&self, artifact: &Path) -> Result<Option<InspectionRequest>, String> {
        if self.values.is_empty() && self.dumps.is_empty() && self.probes.is_empty() {
            return Ok(None);
        }
        let get = |name: &str| self.values.get(name).map_or("", String::as_str);
        let instance = get("instance").to_owned();
        let incarnation = if get("incarnation").is_empty() {
            0
        } else {
            unsigned(get("incarnation"))?
        };
        if instance.is_empty() != (incarnation == 0) {
            return Err("inspection target requires both instance and positive incarnation".into());
        }
        let maximum_bytes = if get("debug-bytes").is_empty() {
            8 << 20
        } else {
            byte_size(get("debug-bytes"))?
        };
        let per_dump_bytes = if get("dump-bytes").is_empty() {
            4 << 20
        } else {
            byte_size(get("dump-bytes"))?
        };
        if maximum_bytes == 0
            || maximum_bytes > 64 << 20
            || per_dump_bytes == 0
            || per_dump_bytes > maximum_bytes
        {
            return Err(
                "inspection requires positive debug byte bounds, with per-dump <= total <= 64MiB"
                    .into(),
            );
        }
        let search_options = [
            "check",
            "choice-range",
            "search-mode",
            "search-attempts",
            "search-timeout",
        ]
        .iter()
        .any(|name| self.values.contains_key(*name));
        let search = if search_options {
            if get("check").is_empty() || get("choice-range").is_empty() || !get("stop").is_empty()
            {
                return Err("search requires --check and --choice-range, without --stop".into());
            }
            let (first, last) = get("choice-range")
                .split_once(':')
                .ok_or("choice range must be FIRST:LAST")?;
            Some(Search {
                check: get("check").into(),
                first: unsigned(first).map_err(|e| format!("invalid first choice: {e}"))?,
                last: unsigned(last).map_err(|e| format!("invalid last choice: {e}"))?,
                mode: if self.values.contains_key("search-mode") {
                    get("search-mode").into()
                } else {
                    "scan".into()
                },
                attempt_budget: if get("search-attempts").is_empty() {
                    128
                } else {
                    unsigned(get("search-attempts"))?
                },
                wall_time: if get("search-timeout").is_empty() {
                    60_000_000_000
                } else {
                    duration(get("search-timeout"))?
                },
            })
        } else {
            None
        };
        let stop = if search.is_some() {
            Stop {
                kind: String::new(),
                value: 0,
            }
        } else {
            parse_stop(if get("stop").is_empty() {
                "failure"
            } else {
                get("stop")
            })?
        };
        let mut dumps = self.dumps.clone();
        if search.is_none() && dumps.is_empty() {
            dumps = vec!["stacks".into(), "heap".into()];
        }
        if dumps.len() > 256 || self.probes.len() > 256 {
            return Err("inspection dump or probe selection exceeds registration capacity".into());
        }
        if dumps
            .iter()
            .chain(&self.probes)
            .any(|name| name.is_empty() || name.len() > 128)
        {
            return Err("inspection names and patterns must contain 1 to 128 bytes".into());
        }
        if let Some(search) = &search {
            if !dumps.is_empty() || !self.probes.is_empty() {
                return Err(
                    "invariant search cannot combine with a stop, dump or probe selection".into(),
                );
            }
            if search.check.len() > 128
                || search.first > search.last
                || search.attempt_budget == 0
                || search.wall_time == 0
                || !matches!(search.mode.as_str(), "scan" | "bisect")
            {
                return Err("invariant search requires a check, inclusive range, scan or bisect mode and positive bounds".into());
            }
        }
        let artifact = absolute(artifact)?;
        let output_directory = absolute(&if get("debug-output").is_empty() {
            let mut path = artifact.as_os_str().to_os_string();
            path.push(".inspection");
            PathBuf::from(path)
        } else {
            PathBuf::from(get("debug-output"))
        })?;
        if output_directory == Path::new("/") {
            return Err(
                "inspection output directory must be an absolute clean non-root path".into(),
            );
        }
        Ok(Some(InspectionRequest {
            instance,
            incarnation,
            stop,
            dumps,
            probes: self.probes.clone(),
            maximum_bytes,
            per_dump_bytes,
            output_directory,
            search,
            artifact,
            conditions: linked::Conditions::default(),
        }))
    }
}

fn absolute(path: &Path) -> Result<PathBuf, String> {
    let path = if path.is_absolute() {
        path.to_owned()
    } else {
        std::env::current_dir()
            .map_err(|e| e.to_string())?
            .join(path)
    };
    let mut clean = PathBuf::new();
    for part in path.components() {
        match part {
            Component::CurDir => {}
            Component::ParentDir => {
                clean.pop();
            }
            _ => clean.push(part.as_os_str()),
        }
    }
    Ok(clean)
}

fn unsigned(value: &str) -> Result<u64, String> {
    if value.is_empty() || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err(format!("invalid unsigned integer {value:?}"));
    }
    value
        .parse()
        .map_err(|_| format!("invalid unsigned integer {value:?}"))
}
fn byte_size(value: &str) -> Result<u64, String> {
    let (number, multiplier) = [("KiB", 1u64 << 10), ("MiB", 1 << 20), ("GiB", 1 << 30)]
        .into_iter()
        .find_map(|(suffix, multiplier)| {
            value
                .strip_suffix(suffix)
                .map(|number| (number, multiplier))
        })
        .unwrap_or((value, 1));
    let invalid = || format!("invalid byte size {value:?}");
    if number.len() > 1 && number.starts_with('0') {
        return Err(invalid());
    }
    unsigned(number)
        .map_err(|_| invalid())?
        .checked_mul(multiplier)
        .filter(|size| *size > 0)
        .ok_or_else(invalid)
}

/// # Errors
/// Parses Go duration units, including fractional and compound coordinates.
pub fn duration(value: &str) -> Result<u64, String> {
    let invalid = || format!("invalid nonnegative duration {value:?}");
    let negative = value.starts_with('-');
    let mut rest = value.strip_prefix(['+', '-']).unwrap_or(value);
    if rest == "0" {
        return Ok(0);
    }
    if rest.is_empty() {
        return Err(invalid());
    }
    let mut total = 0u64;
    while !rest.is_empty() {
        let count = rest.bytes().take_while(u8::is_ascii_digit).count();
        let whole = if count == 0 {
            0
        } else {
            unsigned(&rest[..count]).map_err(|_| invalid())?
        };
        rest = &rest[count..];
        let mut fraction = "";
        if let Some(after_dot) = rest.strip_prefix('.') {
            let digits = after_dot.bytes().take_while(u8::is_ascii_digit).count();
            fraction = &after_dot[..digits];
            rest = &after_dot[digits..];
        }
        if count == 0 && fraction.is_empty() {
            return Err(invalid());
        }
        let (unit, multiplier) = [
            ("ns", 1u64),
            ("us", 1000),
            ("µs", 1000),
            ("μs", 1000),
            ("ms", 1_000_000),
            ("s", 1_000_000_000),
            ("m", 60_000_000_000),
            ("h", 3_600_000_000_000),
        ]
        .into_iter()
        .find(|(unit, _)| rest.starts_with(unit))
        .ok_or_else(invalid)?;
        let nanos = whole
            .checked_mul(multiplier)
            .and_then(|value| value.checked_add(fraction_nanos(fraction, multiplier)))
            .ok_or_else(invalid)?;
        total = total
            .checked_add(nanos)
            .filter(|total| i64::try_from(*total).is_ok())
            .ok_or_else(invalid)?;
        rest = &rest[unit.len()..];
    }
    if negative && total != 0 {
        return Err(invalid());
    }
    Ok(total)
}

#[expect(
    clippy::cast_precision_loss,
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    reason = "The pinned Go duration parser uses this positive fractional conversion, bounded by one hour"
)]
fn fraction_nanos(fraction: &str, unit: u64) -> u64 {
    let mut value = 0u64;
    let mut scale = 1f64;
    for byte in fraction.bytes() {
        if value > (i64::MAX as u64) / 10 {
            break;
        }
        let next = value * 10 + u64::from(byte - b'0');
        if next > 1u64 << 63 {
            break;
        }
        value = next;
        scale *= 10.;
    }
    (value as f64 * (unit as f64 / scale)) as u64
}
fn parse_stop(value: &str) -> Result<Stop, String> {
    if value == "failure" {
        return Ok(Stop {
            kind: "failure".into(),
            value: 0,
        });
    }
    let (kind, coordinate) = value
        .split_once('=')
        .filter(|(kind, _)| matches!(*kind, "choice" | "time"))
        .ok_or("stop must be choice=N, time=T or failure")?;
    let ordinal = unsigned(coordinate)
        .or_else(|_| {
            if kind == "time" {
                duration(coordinate)
            } else {
                Err(String::new())
            }
        })
        .map_err(|_| format!("invalid nonnegative {kind} stop {coordinate:?}"))?;
    Ok(Stop {
        kind: kind.into(),
        value: ordinal,
    })
}

fn error(status: i32, message: impl Into<String>) -> cli::Error {
    cli::Error {
        status,
        message: message.into(),
    }
}
fn preflight_error(message: impl std::fmt::Display) -> cli::Error {
    error(2, format!("inspection preflight: {message}"))
}
fn execution_error(message: impl std::fmt::Display) -> cli::Error {
    error(3, message.to_string())
}

fn resolve(
    evidence: &helper::Evidence,
    request: &InspectionRequest,
    choice: Option<u64>,
) -> Result<helper::Request, cli::Error> {
    let stop = choice.map_or_else(
        || helper::Stop {
            kind: request.stop.kind.clone(),
            value: request.stop.value,
        },
        |choice| helper::Stop {
            kind: "choice".into(),
            value: choice,
        },
    );
    helper::resolve(
        evidence,
        helper::Handle {
            node: request.instance.clone(),
            incarnation: request.incarnation,
        },
        &stop,
        &request.dumps,
        &request.probes,
        request
            .search
            .as_ref()
            .map_or("", |search| search.check.as_str()),
    )
    .map_err(preflight_error)
}

enum EvaluationError {
    Host(cli::Error),
    Evaluation(cli::Error),
}

impl From<cli::Error> for EvaluationError {
    fn from(error: cli::Error) -> Self {
        Self::Evaluation(error)
    }
}

impl From<EvaluationError> for cli::Error {
    fn from(error: EvaluationError) -> Self {
        match error {
            EvaluationError::Host(error) | EvaluationError::Evaluation(error) => error,
        }
    }
}

fn inspection_execution(
    execution: Result<linked::Execution, String>,
) -> Result<linked::Execution, EvaluationError> {
    match execution {
        Ok(execution) if crate::execute::cache_failure(&execution.outcome) => Err(
            EvaluationError::Host(execution_error(execution.outcome.message)),
        ),
        Err(message) if crate::execute::cache_error(&message) => {
            Err(EvaluationError::Host(execution_error(message)))
        }
        Err(message) => Err(EvaluationError::Evaluation(execution_error(message))),
        Ok(execution) => Ok(execution),
    }
}

fn evaluate(
    opened: &Opened,
    evidence: &helper::Evidence,
    control: helper::Request,
    request: &InspectionRequest,
    timeout: Option<Duration>,
    compile_cache: Option<&str>,
) -> Result<helper::Diagnostics, EvaluationError> {
    let target = control.target.clone();
    let expected = control.expected.clone();
    let registry = evidence
        .incarnations
        .iter()
        .find(|owner| owner.handle == target)
        .ok_or_else(|| preflight_error("unknown inspection target"))?
        .registry
        .clone();
    let mut prepared =
        preflight::prepare_with_inspection(opened.try_clone().map_err(preflight_error)?, control)
            .map_err(preflight_error)?;
    prepared
        .compile_cache(compile_cache)
        .map_err(preflight_error)?;
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::execution_started()?;
    let observed = inspection_execution(preflight::execute_inspection(
        prepared,
        &target,
        request.conditions,
        timeout,
    ))?;
    if observed.outcome.termination != "inspected_prefix" {
        let kind = observed.outcome.termination;
        if matches!(
            kind,
            "debug_error" | "probe_fuel_limit" | "fuel" | "capacity"
        ) {
            return Err(execution_error(format!(
                "inspection evaluation {kind}: {}",
                observed.outcome.message
            ))
            .into());
        }
        return Err(execution_error(format!(
            "instrumentation divergence: inspection terminal {kind:?}: {}",
            observed.outcome.message
        ))
        .into());
    }
    helper::verify_prefix(
        evidence,
        observed
            .evidence
            .get("inspection")
            .map_or(&[], Vec::as_slice),
        &expected,
    )
    .map_err(execution_error)?;
    helper::debug_output(
        observed.evidence.get("debug").map_or(&[], Vec::as_slice),
        &target,
        &registry,
        request.per_dump_bytes,
    )
    .map_err(execution_error)
    .map_err(EvaluationError::from)
}

/// # Errors
/// Runs retained inspection or bounded fresh invariant evaluations using authenticated handles.
pub fn run(
    prepared: &preflight::Prepared,
    request: &InspectionRequest,
    output: &mut dyn Write,
) -> Result<i32, cli::Error> {
    if !prepared.exact || !prepared.available {
        return Err(preflight_error(
            "inspection requires cooperative exact replay",
        ));
    }
    if !prepared
        .opened
        .manifest()
        .record
        .streams
        .contains_key("evidence/inspection")
    {
        return Err(preflight_error(
            "unsupported or incomplete WASM inspection evidence",
        ));
    }
    let raw = prepared
        .opened
        .payload("evidence/inspection", 64 << 20)
        .map_err(preflight_error)?;
    let evidence = helper::Evidence::decode(&raw).map_err(preflight_error)?;
    authenticate_streams(&prepared.opened, &evidence)?;
    if prepared
        .opened
        .manifest()
        .files
        .contains_key(super::cluster::FILE)
        && (request.instance.is_empty() || request.incarnation == 0)
    {
        return Err(preflight_error(
            "inspection requires an explicit instance and incarnation",
        ));
    }
    let control = resolve(
        &evidence,
        request,
        request.search.as_ref().map(|search| search.first),
    )?;
    let parent = request
        .output_directory
        .parent()
        .ok_or_else(|| preflight_error("invalid inspection output directory"))?
        .canonicalize()
        .map_err(preflight_error)?;
    let destination = parent.join(
        request
            .output_directory
            .file_name()
            .ok_or_else(|| preflight_error("invalid inspection output directory"))?,
    );
    let artifact = request.artifact.canonicalize().map_err(preflight_error)?;
    if destination.starts_with(&artifact) {
        return Err(preflight_error(
            "inspection output must be outside artifact",
        ));
    }
    let at = &control.expected;
    let mut report = json!({"status":"", "artifact_id":prepared.opened.manifest().record_sha256,"requested":request,
        "resolved":{"instance":control.target.node,"incarnation":control.target.incarnation,"choice":at.decision,"trace":at.trace,"order":at.order,"time_nanos":at.time_nanos},
        "prefix_verified":false,"witness_match":false,"prefix_cuts":null,"output_path":"","truncated":false});
    let cache = prepared.compile_cache_directory();
    if let Some(search) = &request.search {
        let result = search_run(&prepared.opened, &evidence, request, search, cache)?;
        let status = i32::from(result["status"] != "no_failure");
        report["status"] = "invariant_search".into();
        report["search"] = result;
        serde_json::to_writer(&mut *output, &report).map_err(execution_error)?;
        writeln!(output).map_err(execution_error)?;
        return Ok(status);
    }
    let cuts: Vec<_> = at.cuts.iter().map(|cut| json!({"instance":cut.handle.node,"incarnation":cut.handle.incarnation,
        "choice_records":cut.choice_records,"choice_bytes":cut.choice_bytes,"wasi_bytes":cut.wasi_bytes,
        "network_bytes":cut.network_bytes,"storage_bytes":cut.storage_bytes,"stdout_bytes":cut.stdout_bytes,"stderr_bytes":cut.stderr_bytes})).collect();
    let diagnostics = evaluate(&prepared.opened, &evidence, control, request, None, cache)?;
    let (debug, heap, truncated) = diagnostics
        .bound(request.maximum_bytes)
        .map_err(execution_error)?;
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&destination)
        .map_err(execution_error)?;
    let debug_path = destination.join("debug.json");
    write_private(&debug_path, &debug)?;
    if !heap.is_empty() {
        write_private(&destination.join("heap.pprof"), &heap)?;
    }
    report["status"] = "inspected_prefix".into();
    report["prefix_verified"] = true.into();
    report["witness_match"] = true.into();
    report["prefix_cuts"] = cuts.into();
    report["output_path"] = debug_path.to_string_lossy().into_owned().into();
    report["truncated"] = truncated.into();
    serde_json::to_writer(&mut *output, &report).map_err(execution_error)?;
    writeln!(output).map_err(execution_error)?;
    Ok(0)
}

fn authenticate_streams(opened: &Opened, evidence: &helper::Evidence) -> Result<(), cli::Error> {
    if opened.manifest().files.contains_key(super::cluster::FILE) {
        return super::cluster::authenticate_inspection(opened, evidence).map_err(preflight_error);
    }
    if evidence.incarnations.len() != 1 {
        return Err(preflight_error(
            "unsupported or incomplete WASM inspection evidence",
        ));
    }
    let streams = &evidence.incarnations[0].streams;
    if !streams.network.is_empty() {
        return Err(preflight_error(
            "canonical retained inspection stream identity",
        ));
    }
    for (name, bytes) in [
        ("evidence/choice", &streams.choice),
        ("evidence/transcript", &streams.wasi),
        ("evidence/storage", &streams.storage),
        ("output/stdout", &streams.stdout),
        ("output/stderr", &streams.stderr),
    ] {
        let actual = if let Some(stream) = opened.manifest().record.streams.get(name) {
            opened
                .payload(name, stream.bytes)
                .map_err(preflight_error)?
        } else {
            Vec::new()
        };
        if *bytes != actual {
            return Err(preflight_error(
                "canonical retained inspection stream identity",
            ));
        }
    }
    Ok(())
}

fn write_private(path: &Path, bytes: &[u8]) -> Result<(), cli::Error> {
    OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .and_then(|mut file| file.write_all(bytes))
        .map_err(execution_error)
}

fn search_run(
    opened: &Opened,
    evidence: &helper::Evidence,
    request: &InspectionRequest,
    search: &Search,
    compile_cache: Option<&str>,
) -> Result<Value, cli::Error> {
    let started = Instant::now();
    let first = resolve(evidence, request, Some(search.first))?;
    resolve(evidence, request, Some(search.last))?;
    let owner = evidence
        .incarnations
        .iter()
        .find(|owner| owner.handle == first.target)
        .ok_or_else(|| preflight_error("unknown inspection target"))?;
    let registration = owner
        .registry
        .entries
        .iter()
        .find(|entry| {
            entry.kind == "check"
                && entry.name == search.check
                && entry.available_order <= first.expected.order
        })
        .ok_or_else(|| {
            preflight_error(format!(
                "unknown or unavailable inspection check {:?}",
                search.check
            ))
        })?;
    if search.mode == "bisect" && !registration.monotonic {
        return Err(preflight_error(
            "bisect requires a retained monotonic failure declaration",
        ));
    }
    search::run(
        search,
        || started.elapsed(),
        |choice, remaining| {
            let evaluation = resolve(evidence, request, Some(choice))
                .map_err(EvaluationError::from)
                .and_then(|control| {
                    evaluate(
                        opened,
                        evidence,
                        control,
                        request,
                        Some(remaining),
                        compile_cache,
                    )
                });
            predicate_evaluation(evaluation, &search.check)
        },
    )
}

fn predicate_evaluation(
    evaluation: Result<helper::Diagnostics, EvaluationError>,
    check: &str,
) -> Result<search::Evaluation, cli::Error> {
    let (outcome, detail, verified) = match evaluation {
        Ok(output) => {
            let checks = output.output["checks"].as_array();
            if let Some(checks) = checks
                && checks.len() == 1
                && checks[0]["name"] == check
            {
                let status = checks[0]["status"].as_str().unwrap_or("");
                (
                    match status {
                        "true" => "pass",
                        "false" => "fail",
                        _ => "error",
                    },
                    status.to_owned(),
                    true,
                )
            } else {
                (
                    "error",
                    "inspection did not return exactly one invariant result".into(),
                    false,
                )
            }
        }
        Err(EvaluationError::Host(error)) => return Err(error),
        Err(EvaluationError::Evaluation(error)) => {
            let divergence = error.message.contains("instrumentation divergence");
            (
                if divergence { "divergence" } else { "error" },
                error.message,
                false,
            )
        }
    };
    Ok(search::Evaluation {
        outcome,
        detail,
        prefix_verified: verified,
    })
}

#[cfg(test)]
#[path = "inspection_cache_tests.rs"]
mod cache_tests;
