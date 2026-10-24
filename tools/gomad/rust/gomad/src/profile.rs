//! Diagnostic-only linked execution capture and independent attribution validation.
#[cfg(feature = "helper-diagnostic-profile")]
use gomad_wasmhost::diagnostic::LinkedProfile;
use gomad_wasmhost::profile_inventory as diagnostic;
use serde_json::Value;
use std::collections::BTreeSet;
mod json;
pub mod producer;

pub use crate::campaign::prepare::{Prepared as PreparedGuest, prepare as prepare_guest};

/// # Errors
/// Refuses oversized JSON, duplicate or unknown keys and invalid attribution.
pub fn decode(raw: &[u8]) -> Result<Value, String> {
    json::decode(raw)
}

fn number(value: &Value, field: &str) -> Result<u64, String> {
    value[field]
        .as_u64()
        .ok_or_else(|| format!("diagnostic {field}: missing unsigned integer"))
}
fn sum(mut values: impl Iterator<Item = Result<u64, String>>) -> Result<u64, String> {
    values.try_fold(0u64, |total, value| {
        total
            .checked_add(value?)
            .ok_or("diagnostic total overflow".into())
    })
}
const ABI_FIELDS: &str = "count wall_nanos instantiate_wall_nanos execute_wall_nanos";
const SERIES_FIELDS: &str = "count wall_nanos exclusive_wall_nanos instantiate_wall_nanos execute_wall_nanos execute_exclusive_wall_nanos min_nanos median_nanos p99_nanos max_nanos";
const ROW_FIELDS: &str = "execute_wall_nanos attributed_host_wall_nanos guest_residual_wall_nanos outer_host_wall_nanos difference_nanos tolerance_nanos";
fn fields<'a>(value: &'a Value, names: &str) -> Result<&'a serde_json::Map<String, Value>, String> {
    let entries = value.as_object().ok_or("diagnostic missing object")?;
    if entries.len() != names.split_whitespace().count()
        || !names
            .split_whitespace()
            .all(|name| entries.contains_key(name))
    {
        return Err(format!("diagnostic field inventory: {names}"));
    }
    Ok(entries)
}
fn numeric_fields(value: &Value, names: &str) -> Result<(), String> {
    fields(value, names)?;
    for name in names.split_whitespace() {
        number(value, name)?;
    }
    Ok(())
}
fn legacy(report: &Value, ops: &serde_json::Map<String, Value>) -> Result<(), String> {
    fields(
        report,
        "schema diagnostic_only backend host_os host_arch phases abi nested residual abi_overlap_errors abi_excluded abi_ops owners reconciliation threads threads_omitted unattributed_imports unframed_imports span_errors elapsed_wall_nanos wall_basis report_rows_dropped",
    )?;
    if report["backend"] != "cranelift-jit"
        || !matches!(report["host_os"].as_str(), Some("linux" | "macos"))
        || !matches!(report["host_arch"].as_str(), Some("aarch64" | "x86_64"))
    {
        return Err("diagnostic identity".into());
    }
    let phases = fields(
        &report["phases"],
        "module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute",
    )?;
    for phase in phases.values() {
        numeric_fields(
            phase,
            "count wall_nanos user_nanos system_nanos cpu_samples peak_rss_bytes_high_water",
        )?;
        let samples = number(phase, "cpu_samples")?;
        if samples > number(phase, "count")?
            || (samples == 0
                && (number(phase, "user_nanos")? != 0 || number(phase, "system_nanos")? != 0))
        {
            return Err("diagnostic CPU sample claim".into());
        }
    }
    let abi = fields(
        &report["abi"],
        "clock scheduler select observation idle other_local remote_callback",
    )?;
    for bucket in abi.values().chain(std::iter::once(&report["abi_excluded"])) {
        numeric_fields(bucket, ABI_FIELDS)?;
        let combined = number(bucket, "instantiate_wall_nanos")?
            .checked_add(number(bucket, "execute_wall_nanos")?)
            .ok_or("diagnostic ABI wall overflow")?;
        if combined > number(bucket, "wall_nanos")? {
            return Err("diagnostic ABI wall attribution".into());
        }
    }
    for name in ABI_FIELDS.split_whitespace() {
        let calls = sum(ops.values().map(|op| number(op, name)))?;
        let buckets = sum(abi
            .values()
            .chain(std::iter::once(&report["abi_excluded"]))
            .map(|bucket| number(bucket, name)))?;
        if calls != buckets {
            return Err(format!(
                "diagnostic abi_ops differ from legacy buckets: {name}"
            ));
        }
    }
    let nested = fields(
        &report["nested"],
        "alternative_hash encoding transcript_hash flush remote_write remote_read_wait",
    )?;
    for entry in nested.values() {
        numeric_fields(entry, "count wall_nanos")?;
    }
    numeric_fields(
        &report["residual"],
        "instantiate_wall_nanos execute_wall_nanos",
    )?;
    for (phase, name) in [
        ("instantiate", "instantiate_wall_nanos"),
        ("execute", "execute_wall_nanos"),
    ] {
        // Legacy residuals exclude abi_excluded, preserving the original v1 denominator.
        let total = sum(abi
            .values()
            .chain(std::iter::once(&report["residual"]))
            .map(|bucket| number(bucket, name)))?;
        if total != number(&phases[phase], "wall_nanos")? {
            return Err(format!("diagnostic residual attribution: {phase}"));
        }
    }
    numeric_fields(
        &report["reconciliation"],
        &format!("{ROW_FIELDS} distribution_relative_error_ppm"),
    )?;
    Ok(())
}
fn row(value: &Value) -> Result<[u64; 4], String> {
    let execute = number(value, "execute_wall_nanos")?;
    let attributed = number(value, "attributed_host_wall_nanos")?;
    let guest = number(value, "guest_residual_wall_nanos")?;
    let outer = number(value, "outer_host_wall_nanos")?;
    if guest.checked_add(outer) != Some(execute) {
        return Err("diagnostic guest residual plus outer host differs from execute".into());
    }
    let accounted = attributed
        .checked_add(guest)
        .ok_or("diagnostic total overflow")?;
    let difference = execute.abs_diff(accounted);
    let tolerance = 50_000.max(execute / 200);
    if number(value, "difference_nanos")? != difference
        || number(value, "tolerance_nanos")? != tolerance
    {
        return Err("diagnostic reconciliation row differs from recomputed values".into());
    }
    if difference > tolerance {
        return Err("diagnostic reconciliation miss".into());
    }
    Ok([execute, attributed, guest, outer])
}

fn series(
    ops: &serde_json::Map<String, Value>,
    owners: &serde_json::Map<String, Value>,
) -> Result<(), String> {
    let names = diagnostic::operation_names();
    for (entries, operations) in [(ops, true), (owners, false)] {
        for (name, entry) in entries {
            if if operations {
                !names.contains(name)
            } else {
                !diagnostic::owner_names().contains(&name.as_str())
            } {
                return Err(format!("diagnostic unknown series: {name}"));
            }
            fields(
                entry,
                &if operations {
                    format!("{SERIES_FIELDS} served_by")
                } else {
                    SERIES_FIELDS.into()
                },
            )?;
            let count = number(entry, "count")?;
            let wall = number(entry, "wall_nanos")?;
            let execute = number(entry, "execute_wall_nanos")?;
            let minimum = number(entry, "min_nanos")?;
            let median = number(entry, "median_nanos")?;
            let p99 = number(entry, "p99_nanos")?;
            let maximum = number(entry, "max_nanos")?;
            if count == 0
                || number(entry, "exclusive_wall_nanos")? > wall
                || number(entry, "execute_exclusive_wall_nanos")? > execute
                || execute > wall
                || number(entry, "instantiate_wall_nanos")? > wall - execute
                || maximum > wall
                || !(minimum <= median && median <= p99 && p99 <= maximum)
            {
                return Err(format!("diagnostic invalid series: {name}"));
            }
            if operations {
                let served = entry["served_by"]
                    .as_object()
                    .ok_or("diagnostic missing served_by")?;
                let served_count = sum(served.iter().map(|(owner, count)| {
                    if !owners.contains_key(owner) {
                        return Err(format!("diagnostic missing serving owner: {owner}"));
                    }
                    count
                        .as_u64()
                        .filter(|count| *count > 0)
                        .ok_or("diagnostic invalid served count".into())
                }))?;
                if served_count != count {
                    return Err(format!("diagnostic op calls without owner: {name}"));
                }
            }
        }
    }
    Ok(())
}

/// # Errors
/// Refuses incomplete attribution, malformed distributions and reconciliation misses.
pub fn validate(report: &Value) -> Result<(), String> {
    validate_inner(report, false)
}

/// # Errors
/// Refuses malformed refusal attribution or any recorded instantiation, execution or import.
pub(crate) fn validate_admission_refusal(report: &Value) -> Result<(), String> {
    if ["instantiate", "execute"].into_iter().any(|phase| {
        report["phases"][phase]["count"] != 0 || report["phases"][phase]["wall_nanos"] != 0
    }) || report["abi_ops"]
        .as_object()
        .is_none_or(|ops| !ops.is_empty())
    {
        return Err("diagnostic admission refusal entered guest execution".into());
    }
    validate_inner(report, true)
}

fn validate_inner(report: &Value, admission_refusal: bool) -> Result<(), String> {
    if report["schema"] != "gomad.wasm-profile/v2" || report["diagnostic_only"] != true {
        return Err("diagnostic attribution schema".into());
    }
    for field in [
        "abi_overlap_errors",
        "span_errors",
        "unattributed_imports",
        "unframed_imports",
        "threads_omitted",
        "report_rows_dropped",
    ] {
        if number(report, field)? != 0 {
            return Err(format!("diagnostic attribution errors: {field}"));
        }
    }
    let ops = report["abi_ops"]
        .as_object()
        .ok_or("diagnostic missing abi_ops")?;
    let owners = report["owners"]
        .as_object()
        .ok_or("diagnostic missing owners")?;
    legacy(report, ops)?;
    series(ops, owners)?;
    let totals = row(&report["reconciliation"])?;
    let attributed = sum(owners
        .values()
        .map(|owner| number(owner, "execute_exclusive_wall_nanos")))?;
    let op_execute = sum(ops.values().map(|op| number(op, "execute_wall_nanos")))?;
    let glue = owners
        .get("instrumentation")
        .map_or(Ok(0), |owner| number(owner, "execute_exclusive_wall_nanos"))?;
    if attributed != totals[1]
        || attributed.checked_sub(glue) != Some(op_execute)
        || number(&report["phases"]["execute"], "wall_nanos")? != totals[0]
    {
        return Err("diagnostic owner/op/execute totals differ".into());
    }
    if number(&report["reconciliation"], "distribution_relative_error_ppm")? != 62_500 {
        return Err("diagnostic distribution bound".into());
    }
    let threads = report["threads"]
        .as_array()
        .ok_or("diagnostic missing threads")?;
    if threads.len() > 64 {
        return Err("diagnostic thread row capacity".into());
    }
    let mut identities = BTreeSet::new();
    let mut thread_totals = [0u64; 4];
    for thread in threads {
        numeric_fields(thread, &format!("slot incarnation {ROW_FIELDS}"))?;
        let identity = (number(thread, "slot")?, number(thread, "incarnation")?);
        if !identities.insert(identity) {
            return Err("diagnostic duplicate thread row".into());
        }
        for (total, value) in thread_totals.iter_mut().zip(row(thread)?) {
            *total = total
                .checked_add(value)
                .ok_or("diagnostic thread total overflow")?;
        }
    }
    if !threads.is_empty() && thread_totals != totals {
        return Err("diagnostic thread totals differ".into());
    }
    let elapsed = number(report, "elapsed_wall_nanos")?;
    let basis = if threads.is_empty() {
        "single-store elapsed"
    } else {
        "aggregate store thread-time; overlapping spans are not elapsed shares"
    };
    if elapsed == 0 || report["wall_basis"] != basis || (threads.is_empty() && totals[0] > elapsed)
    {
        return Err("diagnostic elapsed wall basis".into());
    }
    if totals[0] == 0 && !admission_refusal {
        return Err("diagnostic no execute interval".into());
    }
    Ok(())
}

/// Runs a request with a bounded capture. Finalization also runs on ordinary
/// execution errors; callers retain the raw profile before enforcing validation.
/// # Errors
/// Refuses overlapping capture scopes or an oversized report.
#[cfg(feature = "helper-diagnostic-profile")]
pub fn capture<T>(execution: impl FnOnce() -> T) -> Result<(T, Value), String> {
    let session = LinkedProfile::start()?;
    let result = execution();
    let report = session.finish()?;
    Ok((result, report))
}

#[cfg(all(test, feature = "helper-diagnostic-profile"))]
mod tests {
    use super::*;
    use gomad_wasmhost::diagnostic::{ImportBoundary, Phase, PhaseSpan};

    // CLI processes cannot observe cross-request TLS/sink isolation inside one host.
    #[test]
    fn simultaneous_captures_do_not_share_profiles() {
        let barrier = std::sync::Arc::new(std::sync::Barrier::new(2));
        let threads: Vec<_> = [1, 3]
            .into_iter()
            .map(|count| {
                let barrier = barrier.clone();
                std::thread::spawn(move || {
                    let ((), profile) = capture(|| {
                        barrier.wait();
                        let _execute = PhaseSpan::new(Phase::Execute);
                        // Boundary-only spans must stay unattributed, independently per request.
                        for _ in 0..count {
                            let _boundary = ImportBoundary::new();
                        }
                    })
                    .unwrap();
                    assert_eq!(profile["unframed_imports"], count);
                    assert!(validate(&profile).is_err());
                })
            })
            .collect();
        for thread in threads {
            thread.join().unwrap();
        }
        assert!(LinkedProfile::start().unwrap().finish().is_ok());
    }

    // No CLI command can create nested request scopes on the same host thread.
    #[test]
    fn overlapping_capture_is_refused_without_clobbering_the_owner() {
        let session = LinkedProfile::start().unwrap();
        assert!(LinkedProfile::start().is_err());
        {
            let _execute = PhaseSpan::new(Phase::Execute);
        }
        let report = session.finish().unwrap();
        validate(&report).unwrap();
        let ((), report) = capture(|| {
            let _execute = PhaseSpan::new(Phase::Execute);
        })
        .unwrap();
        validate(&report).unwrap();
    }
}

#[cfg(test)]
mod attribution_tests {
    use super::validate;
    use serde_json::{Value, json};

    const CLOCK: &str = "wasi_snapshot_preview1.clock_time_get";
    type InvalidAttribution = (&'static str, &'static str, fn(&mut Value));

    fn fixture() -> Value {
        let mut profile = json!({"schema":"gomad.wasm-profile/v2","diagnostic_only":true,"backend":"cranelift-jit","host_os":"linux","host_arch":"aarch64",
            "phases":{},"abi":{},"nested":{},"residual":{"instantiate_wall_nanos":0,"execute_wall_nanos":70},"abi_overlap_errors":0,
            "abi_excluded":{"count":0,"wall_nanos":0,"instantiate_wall_nanos":0,"execute_wall_nanos":0},"abi_ops":{},"owners":{},
            "reconciliation":{"execute_wall_nanos":100,"attributed_host_wall_nanos":30,"guest_residual_wall_nanos":70,"outer_host_wall_nanos":30,"difference_nanos":0,"tolerance_nanos":50000,"distribution_relative_error_ppm":62500},
            "threads":[],"threads_omitted":0,"unattributed_imports":0,"unframed_imports":0,"span_errors":0,"elapsed_wall_nanos":100,"wall_basis":"single-store elapsed","report_rows_dropped":0});
        for name in "module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute".split_whitespace() {
            profile["phases"][name] = json!({"count":0,"wall_nanos":0,"user_nanos":0,"system_nanos":0,"cpu_samples":0,"peak_rss_bytes_high_water":0});
        }
        profile["phases"]["execute"] = json!({"count":1,"wall_nanos":100,"user_nanos":70,"system_nanos":10,"cpu_samples":1,"peak_rss_bytes_high_water":1024});
        for name in
            "clock scheduler select observation idle other_local remote_callback".split_whitespace()
        {
            profile["abi"][name] = profile["abi_excluded"].clone();
        }
        profile["abi"]["clock"] =
            json!({"count":1,"wall_nanos":30,"instantiate_wall_nanos":0,"execute_wall_nanos":30});
        for name in "alternative_hash encoding transcript_hash flush remote_write remote_read_wait"
            .split_whitespace()
        {
            profile["nested"][name] = json!({"count":0,"wall_nanos":0});
        }
        let series = json!({"count":1,"wall_nanos":30,"exclusive_wall_nanos":30,"instantiate_wall_nanos":0,"execute_wall_nanos":30,"execute_exclusive_wall_nanos":30,"min_nanos":30,"median_nanos":30,"p99_nanos":30,"max_nanos":30});
        profile["owners"]["clock"] = series.clone();
        profile["abi_ops"][CLOCK] = series;
        profile["abi_ops"][CLOCK]["served_by"] = json!({"clock":1});
        profile
    }

    fn remove(value: &mut Value, field: &str) {
        value.as_object_mut().unwrap().remove(field);
    }

    const INVALID_ATTRIBUTIONS: [InvalidAttribution; 23] = [
        ("unknown-op", "unknown series", |d| {
            let op = d["abi_ops"][CLOCK].clone();
            remove(&mut d["abi_ops"], CLOCK);
            d["abi_ops"]["wasi_snapshot_preview1.fd_invented"] = op;
        }),
        ("unknown-owner", "unknown series", |d| {
            d["owners"]["kernel"] = d["owners"]["clock"].clone();
        }),
        (
            "unknown-served-owner",
            "missing serving owner: kernel",
            |d| {
                d["abi_ops"][CLOCK]["served_by"] = json!({"kernel":1});
            },
        ),
        ("missing-op-field", "field inventory", |d| {
            remove(&mut d["abi_ops"][CLOCK], "p99_nanos");
        }),
        ("missing-reconciliation-field", "field inventory", |d| {
            remove(&mut d["reconciliation"], "tolerance_nanos");
        }),
        ("missing-counter", "span_errors", |d| {
            remove(d, "span_errors");
        }),
        ("missing-thread-field", "field inventory", |d| {
            d["threads"] = json!([{"slot":1,"incarnation":2,"execute_wall_nanos":1}]);
        }),
        ("served-exceeds-calls", "calls without owner", |d| {
            d["abi_ops"][CLOCK]["served_by"] = json!({"clock":2});
        }),
        (
            "ops-differ-from-buckets",
            "differ from legacy buckets",
            |d| {
                d["abi_ops"][CLOCK]["count"] = json!(2);
            },
        ),
        ("exclusive-exceeds-wall", "invalid series: clock", |d| {
            d["owners"]["clock"]["exclusive_wall_nanos"] = json!(31);
        }),
        ("unordered-distribution", "invalid series", |d| {
            d["abi_ops"][CLOCK]["median_nanos"] = json!(31);
        }),
        (
            "execute-row-mismatch",
            "guest residual plus outer host",
            |d| {
                d["reconciliation"]["execute_wall_nanos"] = json!(99);
            },
        ),
        ("unattributed-import", "attribution errors", |d| {
            d["unattributed_imports"] = json!(1);
        }),
        ("unframed-import", "attribution errors", |d| {
            d["unframed_imports"] = json!(1);
        }),
        ("span-error", "attribution errors", |d| {
            d["span_errors"] = json!(1);
        }),
        ("threads-omitted", "attribution errors", |d| {
            d["threads_omitted"] = json!(1);
        }),
        ("op-without-owner", "calls without owner", |d| {
            d["abi_ops"][CLOCK]["served_by"] = json!({});
        }),
        (
            "missing-serving-owner",
            "missing serving owner: clock",
            |d| {
                remove(&mut d["owners"], "clock");
            },
        ),
        ("owner-wall-short", "totals differ", |d| {
            d["owners"]["clock"]["execute_exclusive_wall_nanos"] = json!(29);
        }),
        ("row-disagrees", "row differs from recomputed", |d| {
            d["reconciliation"]["difference_nanos"] = json!(3);
        }),
        ("thread-miss", "reconciliation miss", |d| {
            d["threads"] = json!([{"slot":1,"incarnation":2,"execute_wall_nanos":1_000_000,"attributed_host_wall_nanos":100_000,"guest_residual_wall_nanos":800_000,"outer_host_wall_nanos":200_000,"difference_nanos":100_000,"tolerance_nanos":50_000}]);
            d["wall_basis"] =
                json!("aggregate store thread-time; overlapping spans are not elapsed shares");
        }),
        (
            "guest-gaps-do-not-telescope",
            "guest residual plus outer host",
            |d| {
                d["reconciliation"]["outer_host_wall_nanos"] = json!(33);
            },
        ),
        ("reconciliation-miss", "reconciliation miss", |d| {
            let execute = 20_000_000;
            d["phases"]["execute"]["wall_nanos"] = json!(execute);
            d["residual"]["execute_wall_nanos"] = json!(execute - 30);
            d["reconciliation"] = json!({"execute_wall_nanos":execute,"attributed_host_wall_nanos":30,
                "guest_residual_wall_nanos":execute - 200_000,"outer_host_wall_nanos":200_000,
                "difference_nanos":200_000 - 30,"tolerance_nanos":100_000,"distribution_relative_error_ppm":62500});
        }),
    ];

    /// Original `TestDiagnosticAttributionRequiresEveryImportAndReconciliation`:
    /// a v2 profile is attributed only when every import has a known serving
    /// owner, every distribution and row is consistent, no attribution error
    /// counter is set and the execute interval reconciles within tolerance.
    #[test]
    fn attribution_requires_every_import_and_reconciliation() {
        validate(&fixture()).unwrap();
        let mut v1 = fixture();
        v1["schema"] = json!("gomad.wasm-profile/v1");
        assert!(validate(&v1).unwrap_err().contains("attribution schema"));
        let mismatched: Vec<_> = INVALID_ATTRIBUTIONS
            .into_iter()
            .filter_map(|(name, expected, mutate)| {
                let mut profile = fixture();
                mutate(&mut profile);
                match validate(&profile) {
                    Err(error) if error.contains(expected) => None,
                    other => Some(format!("{name}: {other:?}")),
                }
            })
            .collect();
        assert!(mismatched.is_empty(), "{mismatched:#?}");
    }

    /// Original `TestDiagnosticThreadRowsBindAggregateAndFullIdentities`: a
    /// thread row keeps its full incarnation identity, and duplicate rows, a
    /// row missing its aggregate share and more than 64 rows are refused.
    #[test]
    fn thread_rows_bind_aggregate_and_full_identities() {
        let thread = json!({"slot":1,"incarnation":1_u64 << 40,"execute_wall_nanos":100,"attributed_host_wall_nanos":30,"guest_residual_wall_nanos":70,"outer_host_wall_nanos":30,"difference_nanos":0,"tolerance_nanos":50_000});
        let with = |rows: Value| {
            let mut profile = fixture();
            profile["threads"] = rows;
            profile["wall_basis"] =
                json!("aggregate store thread-time; overlapping spans are not elapsed shares");
            profile
        };
        validate(&with(json!([thread]))).unwrap();
        let share = json!({"slot":1,"incarnation":0,"execute_wall_nanos":50,"attributed_host_wall_nanos":0,"guest_residual_wall_nanos":50,"outer_host_wall_nanos":0,"difference_nanos":0,"tolerance_nanos":50_000});
        let many: Vec<_> = (0..65)
            .map(|slot| {
                let mut row = thread.clone();
                row["slot"] = json!(slot);
                row
            })
            .collect();
        for (name, rows, expected) in [
            ("duplicate", json!([thread, thread]), "duplicate thread row"),
            (
                "missing-aggregate-share",
                json!([share]),
                "thread totals differ",
            ),
            ("too-many", json!(many), "thread row capacity"),
        ] {
            let error = validate(&with(rows)).expect_err(name);
            assert!(error.contains(expected), "{name}: {error}");
        }
    }
}
