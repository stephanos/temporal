//! Linked-process cost accounting. The former Go contract measured a separate
//! helper process per execution; the Rust host links the helper, so every
//! guest store, Wasmtime engine and operator step is inside one process self
//! figure. Compiler children are the only separately measured processes. CPU
//! is therefore `compiler children + linked self`, each counted exactly once,
//! and no per-sample helper CPU exists to add.

use super::{
    report::Report,
    usage::{Snapshot, Usage},
};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

const GIB: u64 = 1 << 30;
const MINUTE: u64 = 60_000_000_000;
const HOUR: u64 = 3_600_000_000_000;

/// One fresh guest store: an execution or a replay of its retained artifact.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Sample {
    pub phase: String,
    pub iteration: u64,
    pub wall_nanos: u64,
    pub fuel_used: u64,
    pub guest_peak_memory_bytes: u64,
    pub guest_memory_limit_bytes: u64,
    pub trace_bytes: u64,
    pub virtual_time_nanos: u64,
    /// Process self counters at the store's boundaries; they order the stores
    /// and are never summed into CPU, which the operator interval already holds.
    pub linked_before: Option<Usage>,
    pub linked_after: Option<Usage>,
}

/// Raw, retained measurements. Missing platform observations stay `None`.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Observations {
    pub operator_process_id: u64,
    pub resident_stores: u64,
    pub start: Option<Snapshot>,
    pub prepared: Option<Snapshot>,
    pub executed: Option<Snapshot>,
    pub preparation_wall_nanos: u64,
    pub samples: Vec<Sample>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Ratio {
    pub numerator: u64,
    pub denominator: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Cost {
    pub compiler_cpu_nanos: u64,
    pub linked_cpu_nanos: u64,
    pub cpu_nanos: u64,
    pub compiler_peak_rss_bytes: u64,
    pub linked_peak_rss_bytes: u64,
    pub preparation_wall_nanos: u64,
    pub executions: u64,
    pub replays: u64,
    pub distinct_outcomes: u64,
    pub replayable_bugs: u64,
    pub execution_wall_nanos: u64,
    pub replay_wall_nanos: u64,
    pub fuel_used: u64,
    pub trace_bytes: u64,
    pub virtual_time_nanos: u64,
    pub guest_peak_memory_bytes: u64,
    pub virtual_to_wall: Ratio,
    pub replayable_bugs_per_compute_hour: Ratio,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Check {
    pub name: String,
    pub quantity: String,
    pub observed: u64,
    pub limit: u64,
    pub status: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
#[expect(
    clippy::struct_field_names,
    reason = "The retained verdict keeps the existing within-budget name"
)]
pub(crate) struct Budget {
    pub resident_stores: u64,
    pub checks: Vec<Check>,
    pub violations: Vec<String>,
    pub pending: Vec<String>,
    pub within_budget: bool,
}

fn add(total: &mut u64, value: u64) -> Result<(), String> {
    *total = total
        .checked_add(value)
        .ok_or("qualification cost counter overflow")?;
    Ok(())
}

fn cpu(after: Usage, before: Usage) -> Result<u64, String> {
    if !after.not_before(before) {
        return Err("process resource counters decreased".into());
    }
    after
        .user_nanos
        .checked_sub(before.user_nanos)
        .zip(after.system_nanos.checked_sub(before.system_nanos))
        .and_then(|(user, system)| user.checked_add(system))
        .ok_or_else(|| "qualification cost counter overflow".into())
}

/// Returns the ordered, complete operator interval or the missing measurement.
fn interval(observations: &Observations) -> Result<(Snapshot, Snapshot, Snapshot), String> {
    let (Some(start), Some(prepared), Some(executed)) = (
        observations.start,
        observations.prepared,
        observations.executed,
    ) else {
        return Err("actual linked-process and compiler resource observations are required".into());
    };
    if executed.linked.peak_rss_bytes == 0 || observations.operator_process_id == 0 {
        return Err("actual linked-process identity and peak RSS are required".into());
    }
    for (later, earlier) in [(prepared, start), (executed, prepared)] {
        if !later.linked.not_before(earlier.linked) || !later.children.not_before(earlier.children)
        {
            return Err("process resource counters decreased".into());
        }
    }
    Ok((start, prepared, executed))
}

/// Validates per-store observations: phase, required values, guest bound, and
/// that stores ran one at a time inside the measured execution interval.
fn validate_samples(
    observations: &Observations,
    prepared: Snapshot,
    executed: Snapshot,
) -> Result<(), String> {
    if observations.resident_stores != 1 {
        return Err(
            "concurrent guest stores need per-store attribution; only one resident store is measured"
                .into(),
        );
    }
    let mut previous = prepared.linked;
    let mut iterations = [0_u64; 2];
    for sample in &observations.samples {
        let phase = match sample.phase.as_str() {
            "execute" => 0,
            "replay" => 1,
            _ => return Err("actual WASM execution or replay phase is required".into()),
        };
        iterations[phase] += 1;
        if sample.iteration != iterations[phase] {
            return Err("qualification sample order differs from its iterations".into());
        }
        if sample.wall_nanos == 0
            || sample.guest_peak_memory_bytes == 0
            || sample.guest_memory_limit_bytes == 0
        {
            return Err("actual store wall and guest peak memory measurements are required".into());
        }
        let (Some(before), Some(after)) = (sample.linked_before, sample.linked_after) else {
            return Err("actual store boundary resource observations are required".into());
        };
        if !before.not_before(previous) || !after.not_before(before) {
            return Err("guest store observations overlap or are out of order".into());
        }
        previous = after;
    }
    if !executed.linked.not_before(previous) {
        return Err("guest store observations exceed the measured execution interval".into());
    }
    Ok(())
}

/// Distinct canonical outcomes and distinct replayed failure signatures.
fn yields(report: &Report) -> Result<(u64, u64), String> {
    let mut outcomes = BTreeSet::new();
    let mut bugs = BTreeSet::new();
    for execution in &report.executions {
        outcomes.insert(&execution.record_sha256);
        if execution.record.outcome.class != "success"
            && execution
                .replay
                .as_ref()
                .is_some_and(|replay| replay.matched)
        {
            // The retained failure's signature is the bug identity; repeats and
            // replays do not inflate distinct bug yield.
            bugs.insert(execution.record.failure_signature()?);
        }
    }
    Ok((outcomes.len() as u64, bugs.len() as u64))
}

/// Recomputes costs from raw observations and the canonical report.
pub(crate) fn cost(observations: &Observations, report: &Report) -> Result<Cost, String> {
    let (start, prepared, executed) = interval(observations)?;
    validate_samples(observations, prepared, executed)?;
    if observations.preparation_wall_nanos == 0 {
        return Err("actual preparation wall measurement is required".into());
    }
    let compiler_cpu_nanos = cpu(executed.children, start.children)?;
    if compiler_cpu_nanos == 0 || executed.children.peak_rss_bytes == 0 {
        return Err("actual compiler resource usage is unavailable".into());
    }
    let linked_cpu_nanos = cpu(executed.linked, start.linked)?;
    let cpu_nanos = compiler_cpu_nanos
        .checked_add(linked_cpu_nanos)
        .ok_or("qualification cost counter overflow")?;
    let mut cost = Cost {
        compiler_cpu_nanos,
        linked_cpu_nanos,
        cpu_nanos,
        compiler_peak_rss_bytes: executed.children.peak_rss_bytes,
        linked_peak_rss_bytes: executed.linked.peak_rss_bytes,
        preparation_wall_nanos: observations.preparation_wall_nanos,
        executions: 0,
        replays: 0,
        distinct_outcomes: 0,
        replayable_bugs: 0,
        execution_wall_nanos: 0,
        replay_wall_nanos: 0,
        fuel_used: 0,
        trace_bytes: 0,
        virtual_time_nanos: 0,
        guest_peak_memory_bytes: 0,
        virtual_to_wall: Ratio {
            numerator: 0,
            denominator: 0,
        },
        replayable_bugs_per_compute_hour: Ratio {
            numerator: 0,
            denominator: cpu_nanos,
        },
    };
    for sample in &observations.samples {
        cost.guest_peak_memory_bytes = cost
            .guest_peak_memory_bytes
            .max(sample.guest_peak_memory_bytes);
        if sample.phase == "replay" {
            cost.replays += 1;
            add(&mut cost.replay_wall_nanos, sample.wall_nanos)?;
            continue;
        }
        cost.executions += 1;
        add(&mut cost.execution_wall_nanos, sample.wall_nanos)?;
        add(&mut cost.fuel_used, sample.fuel_used)?;
        add(&mut cost.trace_bytes, sample.trace_bytes)?;
        add(&mut cost.virtual_time_nanos, sample.virtual_time_nanos)?;
    }
    let replays = report
        .executions
        .iter()
        .filter(|execution| execution.replay.is_some())
        .count() as u64;
    if cost.executions == 0
        || cost.executions != report.executions.len() as u64
        || cost.replays != replays
    {
        return Err("store samples differ from the qualification report executions".into());
    }
    for (sample, execution) in observations
        .samples
        .iter()
        .filter(|sample| sample.phase == "execute")
        .zip(&report.executions)
    {
        if sample.virtual_time_nanos != execution.record.virtual_time_nanos
            || sample.guest_memory_limit_bytes != execution.record.limits.memory_bytes
        {
            return Err("store sample differs from its canonical execution record".into());
        }
    }
    (cost.distinct_outcomes, cost.replayable_bugs) = yields(report)?;
    cost.virtual_to_wall = Ratio {
        numerator: cost.virtual_time_nanos,
        denominator: cost.execution_wall_nanos,
    };
    cost.replayable_bugs_per_compute_hour.numerator = cost
        .replayable_bugs
        .checked_mul(HOUR)
        .ok_or("qualification cost counter overflow")?;
    Ok(cost)
}

fn check(name: &str, quantity: &str, observed: u64, limit: u64, exact: bool) -> Check {
    Check {
        name: name.into(),
        quantity: quantity.into(),
        observed,
        limit,
        status: if observed <= limit {
            "within"
        } else if exact {
            "exceeded"
        } else {
            "unattributed"
        }
        .into(),
    }
}

/// Evaluates the agreed 16 GiB host protocol. Exact quantities can exceed a
/// limit; the linked peak is an upper bound of each former operator-only or
/// helper-only figure, so it proves those limits only when it fits under them
/// and otherwise leaves them pending rather than being charged to one owner.
pub(crate) fn budget(observations: &Observations) -> Result<Budget, String> {
    let (_, prepared, executed) = interval(observations)?;
    validate_samples(observations, prepared, executed)?;
    if observations.preparation_wall_nanos == 0 || executed.children.peak_rss_bytes == 0 {
        return Err("measured budget requires preparation wall and compiler peak RSS".into());
    }
    let largest_wall = observations
        .samples
        .iter()
        .map(|sample| sample.wall_nanos)
        .max()
        .ok_or("measured budget requires guest store samples")?;
    let linked = executed.linked.peak_rss_bytes;
    let mut checks = vec![
        check(
            "preparation wall",
            "exact",
            observations.preparation_wall_nanos,
            10 * MINUTE,
            true,
        ),
        check(
            "execution/replay store wall",
            "exact",
            largest_wall,
            15 * MINUTE,
            true,
        ),
        check(
            "individual compiler peak RSS",
            "largest waited compiler child",
            executed.children.peak_rss_bytes,
            6 * GIB,
            true,
        ),
    ];
    for sample in &observations.samples {
        if sample.guest_peak_memory_bytes > sample.guest_memory_limit_bytes {
            checks.push(check(
                &format!("guest linear memory {}:{}", sample.phase, sample.iteration),
                "exact",
                sample.guest_peak_memory_bytes,
                sample.guest_memory_limit_bytes,
                true,
            ));
        }
    }
    for (name, limit) in [
        ("operator lifetime peak RSS", 2 * GIB),
        ("individual warm helper peak RSS", 4 * GIB),
        ("individual cold helper peak RSS", 6 * GIB),
        ("conservative resident helper peak upper bound", 8 * GIB),
    ] {
        checks.push(check(
            name,
            "linked process upper bound",
            linked,
            limit,
            false,
        ));
    }
    let violations = checks
        .iter()
        .filter(|check| check.status == "exceeded")
        .map(|check| check.name.clone())
        .collect::<Vec<_>>();
    let pending = checks
        .iter()
        .filter(|check| check.status == "unattributed")
        .map(|check| check.name.clone())
        .collect::<Vec<_>>();
    Ok(Budget {
        resident_stores: observations.resident_stores,
        within_budget: violations.is_empty() && pending.is_empty(),
        checks,
        violations,
        pending,
    })
}

/// Recomputes cost and budget, retaining a failure message instead of a
/// fabricated measurement when an observation is missing.
pub(crate) fn evaluate(
    observations: &Observations,
    report: &Report,
) -> (Option<Cost>, Option<String>, Option<Budget>, Option<String>) {
    let (cost, cost_failure) = match cost(observations, report) {
        Ok(cost) => (Some(cost), None),
        Err(error) => (None, Some(error)),
    };
    let (budget, budget_failure) = match budget(observations) {
        Ok(budget) => (Some(budget), None),
        Err(error) => (None, Some(error)),
    };
    (cost, cost_failure, budget, budget_failure)
}

/// The publication receipt's terminal self observation follows the sealed
/// Result; it applies the same attribution rule to the terminal linked peak.
pub(crate) fn terminal(observations: &Observations, after: Usage) -> Result<u64, String> {
    let (_, _, executed) = interval(observations)?;
    if after.peak_rss_bytes == 0 {
        return Err("terminal linked-process peak RSS is unavailable".into());
    }
    if after.peak_rss_bytes > 2 * GIB {
        return Err(
            "terminal linked-process peak exceeds the operator-only 2GiB ceiling and cannot be attributed"
                .into(),
        );
    }
    cpu(after, executed.linked)
}
