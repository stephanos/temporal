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

/// One fresh guest store of an execution or replay. A cluster execution has
/// one sample per incarnation; they share its wall time and boundary counters
/// because the stores were resident concurrently in one linked process.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Sample {
    pub phase: String,
    pub iteration: u64,
    /// `node/incarnation` for cluster stores; absent for a standalone store.
    #[serde(default)]
    pub incarnation: Option<String>,
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
    /// The linked image high water of this store's window alone (Wasmtime
    /// compilation, instantiation and guest execution), when attributed.
    #[serde(default)]
    pub store_peak_rss_bytes: Option<u64>,
    /// Whether the window loaded a cached compiled module (warm) rather than
    /// compiling it (cold); absent when the helper reported no metrics.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cache_hit: Option<bool>,
}

/// Raw, retained measurements. Missing platform observations stay `None`.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Observations {
    pub operator_process_id: u64,
    /// The largest number of concurrently resident guests (cluster nodes).
    pub resident_stores: u64,
    pub start: Option<Snapshot>,
    pub prepared: Option<Snapshot>,
    pub executed: Option<Snapshot>,
    pub preparation_wall_nanos: u64,
    /// Compiler arguments of every command the cell's initial preparation
    /// actually ran, in order; a compile-cache hit runs no build.
    #[serde(default)]
    pub initial_compiler_commands: Vec<Vec<String>>,
    /// Whether that preparation compiled the module (Go `InitialPreparationCompiled`).
    #[serde(default)]
    pub initial_preparation_compiled: bool,
    pub samples: Vec<Sample>,
    /// The largest linked high water of the operator phases outside every
    /// store window, when phase attribution held for the whole interval.
    #[serde(default)]
    pub operator_peak_rss_bytes: Option<u64>,
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

/// One logical execution or replay: its concurrently resident stores.
struct Group<'a> {
    phase: &'a str,
    wall_nanos: u64,
    /// The linked lifetime peaks at the boundaries around the window.
    peaks: (u64, u64),
    store_peak_rss_bytes: Option<u64>,
    cache_hit: Option<bool>,
    virtual_time_nanos: u64,
    residents: u64,
    stores: Vec<&'a Sample>,
}

const MAXIMUM_RESIDENT_GUESTS: u64 = 2;

/// Groups store samples by logical execution and validates them: phase,
/// required per-store values, one shared window per execution, and windows
/// that run one after another inside the measured execution interval.
fn groups(
    observations: &Observations,
    prepared: Snapshot,
    executed: Snapshot,
) -> Result<Vec<Group<'_>>, String> {
    if observations.samples.is_empty() {
        return Err("actual guest store samples are required".into());
    }
    let mut groups: Vec<Group<'_>> = Vec::new();
    let mut previous = prepared.linked;
    let mut iterations = [0_u64; 2];
    let mut index = 0;
    let samples = &observations.samples;
    while index < samples.len() {
        let first = &samples[index];
        let end = index
            + samples[index..]
                .iter()
                .take_while(|sample| {
                    sample.phase == first.phase && sample.iteration == first.iteration
                })
                .count();
        let stores = samples[index..end].iter().collect::<Vec<_>>();
        index = end;
        let phase = match first.phase.as_str() {
            "execute" => 0,
            "replay" => 1,
            _ => return Err("actual WASM execution or replay phase is required".into()),
        };
        iterations[phase] += 1;
        if first.iteration != iterations[phase] {
            return Err("qualification sample order differs from its iterations".into());
        }
        let (Some(before), Some(after)) = (first.linked_before, first.linked_after) else {
            return Err("actual store boundary resource observations are required".into());
        };
        let mut names = BTreeSet::new();
        let mut nodes = BTreeSet::new();
        for store in &stores {
            if store.wall_nanos == 0
                || store.guest_peak_memory_bytes == 0
                || store.guest_memory_limit_bytes == 0
            {
                return Err(
                    "actual store wall and guest peak memory measurements are required".into(),
                );
            }
            if store.wall_nanos != first.wall_nanos
                || store.virtual_time_nanos != first.virtual_time_nanos
                || store.linked_before != first.linked_before
                || store.linked_after != first.linked_after
                || store.store_peak_rss_bytes != first.store_peak_rss_bytes
                || store.cache_hit != first.cache_hit
            {
                return Err("concurrent stores must share their execution window".into());
            }
            match (&store.incarnation, stores.len()) {
                (None, 1) => {}
                (Some(name), _) => {
                    let (node, incarnation) = name
                        .split_once('/')
                        .filter(|(node, incarnation)| !node.is_empty() && !incarnation.is_empty())
                        .ok_or("store measurement requires a node/incarnation handle")?;
                    if !incarnation.bytes().all(|byte| byte.is_ascii_digit()) || !names.insert(name)
                    {
                        return Err("cluster store handles must be unique node/incarnation".into());
                    }
                    nodes.insert(node);
                }
                (None, _) => return Err("concurrent stores require incarnation handles".into()),
            }
        }
        if !before.not_before(previous) || !after.not_before(before) {
            return Err("guest store observations overlap or are out of order".into());
        }
        previous = after;
        groups.push(Group {
            phase: &first.phase,
            wall_nanos: first.wall_nanos,
            peaks: (before.peak_rss_bytes, after.peak_rss_bytes),
            store_peak_rss_bytes: first.store_peak_rss_bytes,
            cache_hit: first.cache_hit,
            virtual_time_nanos: first.virtual_time_nanos,
            residents: (nodes.len() as u64).max(1),
            stores,
        });
    }
    if !executed.linked.not_before(previous) {
        return Err("guest store observations exceed the measured execution interval".into());
    }
    let residents = groups
        .iter()
        .map(|group| group.residents)
        .max()
        .unwrap_or(0);
    if residents == 0 || residents != observations.resident_stores {
        return Err("retained resident guest count differs from its store samples".into());
    }
    Ok(groups)
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

/// Go `preparationCompiled`: a `go build` or `go test -c` command compiled the module.
pub(crate) fn preparation_compiled(commands: &[Vec<String>]) -> bool {
    commands.iter().any(|arguments| {
        arguments.first().is_some_and(|verb| verb == "build")
            || arguments.len() > 1 && arguments[0] == "test" && arguments[1] == "-c"
    })
}

/// Go `measuredCost`: the retained cold/warm preparation flag must be the
/// classification of the compiler commands that preparation actually ran.
fn initial_preparation(observations: &Observations) -> Result<(), String> {
    if observations.initial_compiler_commands.is_empty()
        || preparation_compiled(&observations.initial_compiler_commands)
            != observations.initial_preparation_compiled
    {
        return Err(
            "initial preparation command classification differs from actual observations".into(),
        );
    }
    Ok(())
}

/// Recomputes costs from raw observations and the canonical report.
pub(crate) fn cost(observations: &Observations, report: &Report) -> Result<Cost, String> {
    let (start, prepared, executed) = interval(observations)?;
    if observations.preparation_wall_nanos == 0 {
        return Err("actual preparation wall measurement is required".into());
    }
    initial_preparation(observations)?;
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
    let groups = groups(observations, prepared, executed)?;
    for group in &groups {
        for store in &group.stores {
            cost.guest_peak_memory_bytes = cost
                .guest_peak_memory_bytes
                .max(store.guest_peak_memory_bytes);
        }
        if group.phase == "replay" {
            cost.replays += 1;
            add(&mut cost.replay_wall_nanos, group.wall_nanos)?;
            continue;
        }
        cost.executions += 1;
        add(&mut cost.execution_wall_nanos, group.wall_nanos)?;
        add(&mut cost.virtual_time_nanos, group.virtual_time_nanos)?;
        for store in &group.stores {
            add(&mut cost.fuel_used, store.fuel_used)?;
            add(&mut cost.trace_bytes, store.trace_bytes)?;
        }
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
    for (group, execution) in groups
        .iter()
        .filter(|group| group.phase == "execute")
        .zip(&report.executions)
    {
        if group.virtual_time_nanos != execution.record.virtual_time_nanos
            || group.stores.len() == 1
                && group.stores[0].incarnation.is_none()
                && group.stores[0].guest_memory_limit_bytes != execution.record.limits.memory_bytes
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
    let (start, prepared, executed) = interval(observations)?;
    let groups = groups(observations, prepared, executed)?;
    if observations.preparation_wall_nanos == 0 || executed.children.peak_rss_bytes == 0 {
        return Err("measured budget requires preparation wall and compiler peak RSS".into());
    }
    if warm_residents(&groups) != observations.resident_stores {
        return Err(
            "actual warm helper peak measurement is required for each resident guest".into(),
        );
    }
    let largest_wall = groups
        .iter()
        .map(|group| group.wall_nanos)
        .max()
        .ok_or("measured budget requires guest store samples")?;
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
        check(
            "resident guests",
            "exact",
            observations.resident_stores,
            MAXIMUM_RESIDENT_GUESTS,
            true,
        ),
    ];
    for sample in &observations.samples {
        if sample.guest_peak_memory_bytes > sample.guest_memory_limit_bytes {
            let store = sample
                .incarnation
                .as_deref()
                .map_or_else(String::new, |name| format!(" {name}"));
            checks.push(check(
                &format!(
                    "guest linear memory {}:{}{store}",
                    sample.phase, sample.iteration
                ),
                "exact",
                sample.guest_peak_memory_bytes,
                sample.guest_memory_limit_bytes,
                true,
            ));
        }
    }
    checks.extend(memory_checks(
        observations,
        &groups,
        (
            start.linked.peak_rss_bytes,
            prepared.linked.peak_rss_bytes,
            executed.linked.peak_rss_bytes,
        ),
    )?);
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

/// Go `EvaluateBudget`: every resident guest needs an actual warm (cache-hit)
/// helper measurement. A cluster cache-hit window measures every node it
/// holds. Whether the warm peak itself is exact or only upper-bounded stays
/// the warm check's status.
fn warm_residents(groups: &[Group<'_>]) -> u64 {
    let mut residents = BTreeSet::new();
    for group in groups.iter().filter(|group| group.cache_hit == Some(true)) {
        for store in &group.stores {
            residents.insert(store.incarnation.as_deref().map_or("", |name| {
                name.split_once('/').map_or(name, |(node, _)| node)
            }));
        }
    }
    residents.len() as u64
}

/// Maps process memory onto the ceilings that historically bounded each
/// phase (QUALIFICATION.md#measured-costs; Go `EvaluateBudget`):
///
/// - operator work outside guest stores: operator lifetime peak, 2 GiB;
/// - a single-guest store window: the cold-cache-miss helper peak, 6 GiB,
///   bounds every such window;
/// - a single-guest window that loaded its compiled module from the compile
///   cache: the warm-cache-hit helper peak, 4 GiB. When no window reports a
///   cache hit, the largest window peak is only an upper bound of it;
/// - a cluster window, which holds every resident guest at once: the resident
///   helper peak, 8 GiB, never the single-guest limits.
///
/// Without attributed phases, the lifetime linked peak upper-bounds every
/// component and proves a limit only when it fits under it. A zero phase peak
/// is a missing measurement, never an attributed one.
///
/// Attributed phases must account for the lifetime peak: every window lies
/// within it and the largest phase reaches it (each phase high water is folded
/// into the lifetime peak; the operator figure is read last, so it may exceed
/// it). Each window also lies within the lifetime peak observed after it, and
/// any rise across its boundaries is explained by the window or by operator
/// work. Phases that understate the lifetime peak refuse the budget.
///
/// A cluster window is one whose stores carry `node/incarnation` handles, even
/// a single-node one; only standalone stores meet the single-guest ceilings.
fn memory_checks(
    observations: &Observations,
    groups: &[Group<'_>],
    boundaries: (u64, u64, u64),
) -> Result<Vec<Check>, String> {
    let linked = boundaries.2;
    let windows = groups
        .iter()
        .map(|group| {
            group
                .store_peak_rss_bytes
                .filter(|peak| *peak > 0)
                .map(|peak| {
                    (
                        group.stores[0].incarnation.is_none(),
                        peak,
                        group.cache_hit == Some(true),
                    )
                })
        })
        .collect::<Option<Vec<_>>>();
    let operator = observations
        .operator_peak_rss_bytes
        .filter(|peak| *peak > 0);
    // Even when attribution is incomplete, every measured window must fit the
    // lifetime peaks around it before the fallback figure is used.
    let (start, prepared, executed) = boundaries;
    if groups.iter().any(|group| {
        group
            .store_peak_rss_bytes
            .is_some_and(|window| window > group.peaks.1 || window > executed)
    }) {
        return Err(
            "phase memory attribution is inconsistent with the lifetime linked peak".into(),
        );
    }
    let (Some(operator), Some(windows)) = (operator, windows) else {
        return Ok(fallback_checks(groups, operator, linked));
    };
    let largest = windows.iter().map(|(_, peak, _)| *peak).max().unwrap_or(0);
    // Follow the lifetime peak from the operator baseline through every
    // boundary: a rise inside a window is explained by that window or by
    // operator work around it, any other rise (preparation, between windows,
    // after the last one) by operator work alone.
    // The operator phases begin before the baseline snapshot, so the
    // operator figure covers the initial peak and every later rise outside
    // windows.
    let mut previous = prepared;
    let mut unexplained = start > operator || prepared > start.max(operator);
    for (group, (_, window, _)) in groups.iter().zip(&windows) {
        let (before, after) = group.peaks;
        unexplained |= before > previous.max(operator)
            || *window > after
            || after > before.max(*window).max(operator);
        previous = after;
    }
    unexplained |= executed > previous.max(operator);
    if unexplained || largest > linked || largest.max(operator) < linked {
        return Err(
            "phase memory attribution is inconsistent with the lifetime linked peak".into(),
        );
    }
    let single = windows
        .iter()
        .filter(|(single, _, _)| *single)
        .map(|(_, peak, _)| *peak)
        .max()
        .unwrap_or(0);
    let warm = windows
        .iter()
        .filter(|(single, _, warm)| *single && *warm)
        .map(|(_, peak, _)| *peak)
        .max();
    let cluster = windows
        .iter()
        .filter(|(single, _, _)| !*single)
        .map(|(_, peak, _)| *peak)
        .max();
    let mut checks = vec![
        check(
            "operator lifetime peak RSS",
            "operator phases",
            operator,
            2 * GIB,
            true,
        ),
        warm_check(warm, single),
        check(
            "individual cold helper peak RSS",
            "single store window",
            single,
            6 * GIB,
            true,
        ),
    ];
    if let Some(cluster) = cluster {
        checks.push(check(
            "conservative resident helper peak upper bound",
            "cluster store window",
            cluster,
            8 * GIB,
            true,
        ));
    }
    Ok(checks)
}

/// A cache-hit window measures the warm phase exactly; otherwise the largest
/// single-guest window only upper-bounds it.
fn warm_check(warm: Option<u64>, single: u64) -> Check {
    match warm {
        Some(warm) => check(
            "individual warm helper peak RSS",
            "single cache-hit store window",
            warm,
            4 * GIB,
            true,
        ),
        None => check(
            "individual warm helper peak RSS",
            "single store window upper bound",
            single,
            4 * GIB,
            false,
        ),
    }
}

/// Without full attribution: a measured phase that already exceeds its
/// ceiling stays a violation, since a missing measurement never erases it;
/// otherwise the lifetime peak upper-bounds the phase.
fn fallback_checks(groups: &[Group<'_>], operator: Option<u64>, linked: u64) -> Vec<Check> {
    let known = |cluster: bool| {
        groups
            .iter()
            .filter(|group| group.stores[0].incarnation.is_some() == cluster)
            .filter_map(|group| group.store_peak_rss_bytes.filter(|peak| *peak > 0))
            .max()
    };
    [
        (
            "operator lifetime peak RSS",
            "operator phases",
            operator,
            2 * GIB,
        ),
        ("individual warm helper peak RSS", "", None, 4 * GIB),
        (
            "individual cold helper peak RSS",
            "single store window",
            known(false),
            6 * GIB,
        ),
        (
            "conservative resident helper peak upper bound",
            "cluster store window",
            known(true),
            8 * GIB,
        ),
    ]
    .into_iter()
    .map(|(name, quantity, measured, limit)| match measured {
        Some(peak) if peak > limit => check(name, quantity, peak, limit, true),
        _ => check(name, "linked process upper bound", linked, limit, false),
    })
    .collect()
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

/// The publication receipt's terminal self observation, measured after the
/// sealed Result. Missing or backward counters are integrity failures; a
/// terminal linked peak over the operator-only ceiling is not attributable and
/// is reported as such rather than passed.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Terminal {
    pub cpu_nanos: u64,
    pub peak_rss_bytes: u64,
    pub status: String,
}

pub(crate) fn terminal(
    observations: &Observations,
    after: Usage,
    operator_peak: Option<u64>,
) -> Result<Terminal, String> {
    let (_, _, executed) = interval(observations)?;
    if after.peak_rss_bytes == 0 {
        return Err("terminal linked-process peak RSS is unavailable".into());
    }
    // With attributed phases the operator ceiling applies to operator phases
    // only; otherwise the lifetime linked peak is its upper bound.
    let attributed = operator_peak.filter(|_| {
        observations
            .operator_peak_rss_bytes
            .is_some_and(|retained| retained > 0)
    });
    if attributed.is_some_and(|peak| peak < observations.operator_peak_rss_bytes.unwrap_or(0)) {
        return Err("terminal operator phase peak moved backwards".into());
    }
    // After the sealed Result only operator work runs, so an attributed
    // terminal operator phase must explain any rise of the lifetime peak.
    if attributed
        .is_some_and(|peak| after.peak_rss_bytes > peak.max(executed.linked.peak_rss_bytes))
    {
        return Err("terminal lifetime peak rise is not attributed to the operator phase".into());
    }
    // A retained operator overrun stays exceeded even when the receipt lacks
    // its own operator measurement.
    // So does a receipt operator overrun when the Result lacks its own
    // operator figure: a known measurement is never discarded.
    let retained = observations
        .operator_peak_rss_bytes
        .filter(|peak| *peak > 2 * GIB)
        .or(operator_peak.filter(|peak| *peak > 2 * GIB));
    let (peak, exact) = match (attributed, retained) {
        (Some(peak), _) | (None, Some(peak)) => (peak, true),
        (None, None) => (after.peak_rss_bytes, false),
    };
    Ok(Terminal {
        cpu_nanos: cpu(after, executed.linked)?,
        peak_rss_bytes: peak,
        status: if peak <= 2 * GIB {
            "within"
        } else if exact {
            "exceeded"
        } else {
            "unattributed"
        }
        .into(),
    })
}

#[cfg(test)]
mod warm_window_tests {
    use super::*;

    const MIB: u64 = 1 << 20;

    fn usage(cpu: u64, peak: u64) -> Usage {
        Usage {
            user_nanos: cpu,
            system_nanos: cpu,
            peak_rss_bytes: peak,
        }
    }

    fn sample(
        phase: &str,
        iteration: u64,
        cpu: u64,
        peaks: (u64, u64, u64),
        hit: Option<bool>,
    ) -> Sample {
        Sample {
            phase: phase.into(),
            iteration,
            incarnation: None,
            wall_nanos: 1_000_000_000,
            fuel_used: 1,
            guest_peak_memory_bytes: 200 * MIB,
            guest_memory_limit_bytes: 4 * GIB,
            trace_bytes: 1,
            virtual_time_nanos: 1,
            linked_before: Some(usage(cpu, peaks.0)),
            linked_after: Some(usage(cpu + 1, peaks.1)),
            store_peak_rss_bytes: Some(peaks.2),
            cache_hit: hit,
        }
    }

    /// Shaped on an actual onebox cell: one cold compile above the warm
    /// ceiling, then cache-hit executions and replays well below it.
    fn observations(hits: [Option<bool>; 4]) -> Observations {
        let cold = 5_700 * MIB;
        let warm = 2_000 * MIB;
        let snapshot = |cpu, peak| Snapshot {
            linked: usage(cpu, peak),
            children: usage(cpu, 4_400 * MIB),
        };
        Observations {
            operator_process_id: 1,
            resident_stores: 1,
            start: Some(snapshot(0, 3 * MIB)),
            prepared: Some(snapshot(1, 700 * MIB)),
            executed: Some(snapshot(9, cold)),
            preparation_wall_nanos: 1,
            initial_compiler_commands: vec![vec!["test".into(), "-c".into()]],
            initial_preparation_compiled: true,
            samples: vec![
                sample("execute", 1, 1, (700 * MIB, cold, cold), hits[0]),
                sample("execute", 2, 3, (cold, cold, warm), hits[1]),
                sample("replay", 1, 5, (cold, cold, warm), hits[2]),
                sample("replay", 2, 7, (cold, cold, warm), hits[3]),
            ],
            operator_peak_rss_bytes: Some(1_100 * MIB),
        }
    }

    fn warm_check(budget: &Budget) -> &Check {
        budget
            .checks
            .iter()
            .find(|check| check.name == "individual warm helper peak RSS")
            .unwrap()
    }

    #[test]
    fn cache_hit_windows_attribute_the_warm_helper_peak_exactly() {
        let budget = budget(&observations([
            Some(false),
            Some(true),
            Some(true),
            Some(true),
        ]))
        .unwrap();
        let warm = warm_check(&budget);
        assert_eq!(
            (warm.quantity.as_str(), warm.observed, warm.status.as_str()),
            ("single cache-hit store window", 2_000 * MIB, "within")
        );
        assert!(budget.within_budget, "{budget:?}");
    }

    #[test]
    fn a_cold_warm_window_still_exceeds() {
        let mut cold_replay = observations([Some(false), Some(true), Some(true), Some(true)]);
        cold_replay.samples[3].store_peak_rss_bytes = Some(5_600 * MIB);
        let budget = budget(&cold_replay).unwrap();
        assert_eq!(warm_check(&budget).status, "exceeded");
        assert!(!budget.within_budget);
    }

    const WARM_REQUIRED: &str =
        "actual warm helper peak measurement is required for each resident guest";

    /// Go `EvaluateBudget`: without an actual cache-hit measurement the warm
    /// ceiling is not met by an upper bound; the budget is refused.
    #[test]
    fn every_resident_guest_requires_an_actual_warm_measurement() {
        for hits in [[None; 4], [Some(false); 4]] {
            assert_eq!(budget(&observations(hits)), Err(WARM_REQUIRED.into()));
        }
        // A cache-hit window without its own attributed peak is a warm
        // measurement whose peak stays only upper-bounded.
        let mut unattributed = observations([Some(false), Some(true), Some(true), Some(true)]);
        for sample in &mut unattributed.samples[1..] {
            sample.store_peak_rss_bytes = None;
        }
        let budget = budget(&unattributed).unwrap();
        assert_eq!(warm_check(&budget).status, "unattributed");
    }

    /// Go `measuredCost`: a cold flag needs an actual `go build`/`go test -c`
    /// among the commands preparation ran, and a warm flag needs none.
    #[test]
    fn initial_preparation_flag_matches_its_actual_compiler_commands() {
        let refused = Err(
            "initial preparation command classification differs from actual observations".into(),
        );
        let classify = |commands: &[&[&str]], compiled: bool| {
            let mut cell = observations([Some(false), Some(true), Some(true), Some(true)]);
            cell.initial_compiler_commands = commands
                .iter()
                .map(|command| command.iter().map(|value| (*value).to_owned()).collect())
                .collect();
            cell.initial_preparation_compiled = compiled;
            initial_preparation(&cell)
        };
        let listed: &[&str] = &["list", "-json", "./tests"];
        for build in [
            &["test", "-c", "-o", "m.wasm"][..],
            &["build", "-o", "m.wasm"][..],
        ] {
            assert_eq!(classify(&[listed, build], true), Ok(()));
            assert_eq!(classify(&[listed, build], false), refused);
        }
        assert_eq!(classify(&[listed], false), Ok(()));
        assert_eq!(classify(&[listed], true), refused);
        // `go test` without -c runs a test; it is not the module build.
        assert_eq!(classify(&[&["test", "-run", "x"]], true), refused);
        assert_eq!(classify(&[], false), refused);
    }

    /// A cluster cache-hit window measures every node it holds; a node only
    /// ever measured cold is refused.
    #[test]
    fn cluster_warm_windows_must_cover_every_resident_node() {
        let cluster = |warm_nodes: &[&str]| {
            let mut cell = observations([Some(false), Some(true), Some(true), Some(true)]);
            cell.resident_stores = 2;
            let mut samples = Vec::new();
            for (index, sample) in cell.samples.iter().enumerate() {
                let nodes: &[&str] = if index == 0 { &["a", "b"] } else { warm_nodes };
                for node in nodes {
                    let mut store = sample.clone();
                    store.incarnation = Some(format!("{node}/1"));
                    samples.push(store);
                }
            }
            cell.samples = samples;
            budget(&cell)
        };
        assert!(cluster(&["a", "b"]).is_ok());
        assert_eq!(cluster(&["a"]), Err(WARM_REQUIRED.into()));
    }
}
