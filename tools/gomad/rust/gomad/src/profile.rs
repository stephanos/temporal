//! Diagnostic-only linked execution capture and independent attribution validation.
use gomad_wasmhost::diagnostic::{self, LinkedProfile};
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
    if totals[0] == 0 {
        return Err("diagnostic no execute interval".into());
    }
    Ok(())
}

/// Runs a request with a bounded capture. Finalization also runs on ordinary
/// execution errors; callers retain the raw profile before enforcing validation.
/// # Errors
/// Refuses overlapping capture scopes or an oversized report.
pub fn capture<T>(execution: impl FnOnce() -> T) -> Result<(T, Value), String> {
    let session = LinkedProfile::start()?;
    let result = execution();
    let report = session.finish()?;
    Ok((result, report))
}

#[cfg(test)]
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
