use super::evidence::{self, ComparisonMask, NativeRun};
use super::statistics::{TimingRange, timing};
use crate::evidence::canonical;
use gomad_wasmhost::linked::Execution;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::time::Instant;

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Role {
    Baseline,
    Candidate,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Binding {
    pub source_commit: String,
    pub source_root: String,
    pub source_inputs: BTreeMap<String, String>,
    pub driver_sha256: String,
    pub host_sha256: String,
    pub module_sha256: String,
    pub config_sha256: String,
    pub model_sha256: String,
    pub runtime_identity: String,
    pub host_boot_id: String,
    pub host_cpu: String,
    pub host_os: String,
    pub host_arch: String,
    pub engine_version: String,
    pub engine_configuration: String,
    pub module_producer_sha256: String,
    pub seed: u64,
    pub artifact_bytes: u64,
    pub envelope_bytes: u64,
}

#[derive(Clone, Copy, Debug, Default, Eq, PartialEq, Serialize)]
pub struct Phase {
    pub count: u64,
    pub wall_nanos: u64,
}

pub struct TeardownInterval {
    pub terminal: Instant,
    pub returned: Instant,
    pub recorded_nanos: u64,
}

pub struct Observation {
    pub execution: Execution,
    pub phases: BTreeMap<String, Phase>,
    pub wall_nanos: u64,
    pub teardown: TeardownInterval,
    pub fresh_store_id: u64,
}

#[derive(Serialize)]
pub struct Run {
    pub native: Option<Box<NativeRun>>,
    pub raw_outcome: serde_json::Value,
    pub raw_evidence: BTreeMap<String, Vec<u8>>,
    pub event_order: Option<Vec<u8>>,
    pub phases: BTreeMap<String, Phase>,
    pub wall_nanos: u64,
    pub teardown_nanos: u64,
    /// Zero means no independent clock observation for a retained native monotonic interval.
    pub observed_teardown_nanos: u64,
    pub fresh_store_id: u64,
    pub evidence_sha256: String,
    pub prepare_nanos: u64,
    pub compile_nanos: u64,
    pub cache_hit: bool,
}

#[derive(Serialize)]
pub struct Series {
    pub schema: &'static str,
    pub role: Role,
    pub binding: Binding,
    pub warmup: Run,
    pub runs: Vec<Run>,
}

fn hex(value: &str, size: usize) -> bool {
    value.len() == size && value.bytes().all(|byte| byte.is_ascii_hexdigit())
}

fn validate_binding(binding: &Binding) -> Result<(), String> {
    if !hex(&binding.source_commit, 40)
        || binding.source_root.is_empty()
        || binding.source_inputs.is_empty()
        || binding
            .source_inputs
            .iter()
            .any(|(path, digest)| path.is_empty() || !hex(digest, 64))
        || [
            &binding.driver_sha256,
            &binding.host_sha256,
            &binding.module_sha256,
            &binding.config_sha256,
            &binding.model_sha256,
            &binding.module_producer_sha256,
        ]
        .into_iter()
        .any(|digest| !hex(digest, 64))
        || [
            &binding.runtime_identity,
            &binding.host_boot_id,
            &binding.host_cpu,
            &binding.host_os,
            &binding.host_arch,
            &binding.engine_configuration,
        ]
        .into_iter()
        .any(String::is_empty)
        || binding.engine_version != "47.0.3"
        || binding.artifact_bytes == 0
        || binding.artifact_bytes >= binding.envelope_bytes
    {
        return Err(
            "warm preparation lacks source, host, engine, module or producer closure".into(),
        );
    }
    Ok(())
}

fn behavior(
    outcome: &serde_json::Value,
    evidence: &BTreeMap<String, Vec<u8>>,
    event_order: Option<&[u8]>,
) -> Result<String, String> {
    let mut semantic = outcome.clone();
    semantic
        .as_object_mut()
        .ok_or("native outcome must be an object")?
        .remove("metrics");
    let mut hash = Sha256::new();
    let encoded = serde_json::to_vec(&semantic).map_err(|error| error.to_string())?;
    for bytes in [
        b"gomad.native-warm-behavior/v1".as_slice(),
        encoded.as_slice(),
    ] {
        hash.update((bytes.len() as u64).to_be_bytes());
        hash.update(bytes);
    }
    for (name, bytes) in evidence {
        hash.update((name.len() as u64).to_be_bytes());
        hash.update(name.as_bytes());
        hash.update((bytes.len() as u64).to_be_bytes());
        hash.update(bytes);
    }
    hash.update([u8::from(event_order.is_some())]);
    if let Some(bytes) = event_order {
        hash.update((bytes.len() as u64).to_be_bytes());
        hash.update(bytes);
    }
    Ok(format!("{:x}", hash.finalize()))
}

pub const NATIVE_TIMING_SCHEMA: &str = "gomad.native-linked-timing/v1";
pub const NATIVE_TIMING_SCOPE: &str = "validated-native-terminal-to-linked-return";

fn retained_timing(run: &NativeRun) -> Result<(u64, u64), String> {
    let timing = &run.report["timing"];
    let teardown = run.report["terminal_to_return_nanos"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("actual native terminal-to-return interval absent")?;
    let store = run.report["fresh_store_id"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("actual native fresh Store process identity absent")?;
    if timing["schema"] != NATIVE_TIMING_SCHEMA
        || timing["scope"] != NATIVE_TIMING_SCOPE
        || timing["clock"] != "monotonic"
        || timing["terminal_to_return_nanos"].as_u64() != Some(teardown)
        || timing["fresh_store_id"].as_u64() != Some(store)
        || timing["implementation_sha256"] != run.report["linked_implementation_sha256"]
        || !timing["implementation_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
    {
        return Err(
            "native monotonic teardown interval lost its implementation/process binding".into(),
        );
    }
    Ok((teardown, store))
}
fn retained_phases(run: &NativeRun) -> Result<BTreeMap<String, Phase>, String> {
    run.profile["phases"]
        .as_object()
        .ok_or("actual native warm diagnostic phases absent")?
        .iter()
        .map(|(name, row)| {
            Ok((
                name.clone(),
                Phase {
                    count: row["count"].as_u64().ok_or("native phase count absent")?,
                    wall_nanos: row["wall_nanos"]
                        .as_u64()
                        .ok_or("native phase timing absent")?,
                },
            ))
        })
        .collect()
}

/// Digest of actual native guest configuration, excluding its separately bound module reference.
/// # Errors
/// Refuses a missing guest configuration object.
pub fn retained_configuration_sha256(request: &serde_json::Value) -> Result<String, String> {
    let mut guest = request["guest"]
        .as_object()
        .ok_or("native warm guest configuration absent")?
        .clone();
    guest.remove("module");
    guest.remove("module_sha256");
    Ok(canonical::digest(&canonical::encode(&guest)?))
}
fn validate_retained_binding(binding: &Binding, run: &Run) -> Result<(), String> {
    let Some(native) = &run.native else {
        return Ok(());
    };
    if native.request["kind"] != "standalone"
        || native.request["guest"]["module_sha256"] != binding.module_sha256
        || native.report["host_sha256"] != binding.host_sha256
        || native.report["host_os"] != binding.host_os
        || native.report["host_arch"] != binding.host_arch
        || native.report["configuration"] != binding.engine_configuration
        || native.report["diagnostic_only"] != true
        || native.report["features"] != serde_json::json!(["diagnostic-profile"])
        || native.request["compile_cache"]
            .as_str()
            .is_none_or(str::is_empty)
        || retained_configuration_sha256(&native.request)? != binding.config_sha256
    {
        return Err("retained native warm witness changed module, helper, configuration or compiled role closure".into());
    }
    evidence::compare(
        native,
        native,
        ComparisonMask::Variant {
            baseline: evidence::VariantRole::DiagnosticProfile,
            candidate: evidence::VariantRole::DiagnosticProfile,
            gc: None,
        },
    )?;
    Ok(())
}

impl Run {
    fn observed(observation: Observation) -> Result<Self, String> {
        let execution = observation.execution;
        let timing = execution
            .timing
            .as_ref()
            .ok_or("actual native terminal timing absent")?;
        if timing.terminal_classified < timing.authorization_started
            || timing.returned < timing.terminal_classified
            || timing.terminal_classified != observation.teardown.terminal
            || timing.returned != observation.teardown.returned
            || execution.authorization_to_terminal_nanos
                != Some(timing.authorization_to_terminal_nanos())
        {
            return Err("actual native terminal timing is inconsistent".into());
        }
        #[cfg(feature = "helper-guest-profile")]
        if let Some(error) = &execution.guest_profile_error {
            return Err(format!("native guest profile finalization failed: {error}"));
        }
        if execution.outcome.termination != "exit"
            || execution.outcome.exit_code != Some(0)
            || execution.outcome.divergence.is_some()
        {
            return Err("warm preparation native execution did not exit successfully".into());
        }
        let metrics = execution
            .outcome
            .metrics
            .as_ref()
            .ok_or("native prepare metrics absent")?;
        let (prepare_nanos, compile_nanos, cache_hit) = (
            metrics.prepare_nanos,
            metrics.compile_nanos,
            metrics.cache_hit,
        );
        let raw_outcome =
            serde_json::to_value(&execution.outcome).map_err(|error| error.to_string())?;
        let evidence_sha256 = behavior(
            &raw_outcome,
            &execution.evidence,
            execution.event_order.as_deref(),
        )?;
        let elapsed = observation
            .teardown
            .returned
            .checked_duration_since(observation.teardown.terminal)
            .ok_or("native teardown clocks are reversed")?;
        let observed_teardown_nanos =
            u64::try_from(elapsed.as_nanos()).map_err(|_| "native teardown interval overflow")?;
        if observed_teardown_nanos != observation.teardown.recorded_nanos
            || !super::statistics::reap_consistent(
                observed_teardown_nanos,
                observation.teardown.recorded_nanos,
            )
        {
            return Err("native terminal-to-teardown interval is inconsistent".into());
        }
        Ok(Self {
            native: None,
            raw_outcome,
            raw_evidence: execution.evidence,
            event_order: execution.event_order,
            phases: observation.phases,
            wall_nanos: observation.wall_nanos,
            teardown_nanos: observed_teardown_nanos,
            observed_teardown_nanos,
            fresh_store_id: observation.fresh_store_id,
            evidence_sha256,
            prepare_nanos,
            compile_nanos,
            cache_hit,
        })
    }

    fn validate(&self, candidate: bool, warm: bool) -> Result<(), String> {
        let interval_valid = if let Some(native) = &self.native {
            let (teardown, store) = retained_timing(native)?;
            if native.root.outcome != self.raw_outcome
                || native.root.streams != self.raw_evidence
                || self.observed_teardown_nanos != 0
                || self.event_order.is_some()
                || retained_phases(native)? != self.phases
                || native.profile["elapsed_wall_nanos"].as_u64() != Some(self.wall_nanos)
            {
                return Err("retained native warm observation changed its raw witness".into());
            }
            teardown == self.teardown_nanos
                && store == self.fresh_store_id
                && teardown <= self.wall_nanos
        } else {
            self.observed_teardown_nanos == self.teardown_nanos
                && super::statistics::reap_consistent(
                    self.observed_teardown_nanos,
                    self.teardown_nanos,
                )
        };
        let metrics = &self.raw_outcome["metrics"];
        if self.raw_outcome["termination"] != "exit"
            || self.raw_outcome["exit_code"] != 0
            || metrics["prepare_nanos"].as_u64() != Some(self.prepare_nanos)
            || metrics["compile_nanos"].as_u64() != Some(self.compile_nanos)
            || metrics["cache_hit"].as_bool() != Some(self.cache_hit)
            || self.wall_nanos == 0
            || self.prepare_nanos == 0
            || self.fresh_store_id == 0
            || !interval_valid
            || candidate && self.teardown_nanos > 1_000_000_000
            || self.evidence_sha256
                != behavior(
                    &self.raw_outcome,
                    &self.raw_evidence,
                    self.event_order.as_deref(),
                )?
        {
            return Err(
                "native warm observation lost its raw outcome, interval or evidence binding".into(),
            );
        }
        if warm {
            if !self.cache_hit || self.compile_nanos != 0 {
                return Err("timed run is not a warm native cache hit".into());
            }
            for phase in [
                "cache_read",
                "envelope_verify",
                "deserialize",
                "cache_identity",
            ] {
                let phase = self
                    .phases
                    .get(phase)
                    .ok_or("warm cache diagnostic phase absent")?;
                if phase.count == 0
                    || phase.wall_nanos == 0
                    || phase.wall_nanos > self.prepare_nanos
                {
                    return Err("warm cache diagnostic phase is inconsistent".into());
                }
            }
        }
        Ok(())
    }
}

/// Runs a cache-priming execution followed by fresh measured executions.
/// The callback owns native preparation, execution and explicit Store teardown observation.
/// # Errors
/// Refuses missing native observations, reused stores or inconsistent behavior.
pub fn collect(
    binding: Binding,
    role: Role,
    repetitions: usize,
    mut execute: impl FnMut(usize) -> Result<Observation, String>,
) -> Result<Series, String> {
    validate_binding(&binding)?;
    if repetitions < 5 {
        return Err("warm preparation needs at least five fresh repetitions".into());
    }
    let warmup = Run::observed(execute(0)?)?;
    let mut series = Series {
        schema: "gomad.native-warm-preparation/v1",
        role,
        binding,
        warmup,
        runs: Vec::with_capacity(repetitions),
    };
    for index in 1..=repetitions {
        series.runs.push(Run::observed(execute(index)?)?);
    }
    series.validate()?;
    Ok(series)
}

/// Collects already retained native helper runs with their actual monotonic teardown observation.
/// Failed callback artifacts remain owned by the callback's retained run directory.
/// # Errors
/// Refuses missing timing provenance, changed raw bindings or nonwarm measured repetitions.
pub fn collect_retained(
    binding: Binding,
    role: Role,
    repetitions: usize,
    mut execute: impl FnMut(usize) -> Result<NativeRun, String>,
) -> Result<Series, String> {
    validate_binding(&binding)?;
    if repetitions < 5 {
        return Err("warm preparation needs at least five fresh repetitions".into());
    }
    let mut observed = |index| -> Result<Run, String> {
        let native = execute(index)?;
        let (teardown_nanos, fresh_store_id) = retained_timing(&native)?;
        let phases = retained_phases(&native)?;
        let wall_nanos = native.profile["elapsed_wall_nanos"]
            .as_u64()
            .filter(|value| *value > 0)
            .ok_or("actual native warm wall interval absent")?;
        let raw_outcome = native.root.outcome.clone();
        let metrics = &raw_outcome["metrics"];
        let prepare_nanos = metrics["prepare_nanos"]
            .as_u64()
            .ok_or("native preparation metric absent")?;
        let compile_nanos = metrics["compile_nanos"]
            .as_u64()
            .ok_or("native compilation metric absent")?;
        let cache_hit = metrics["cache_hit"]
            .as_bool()
            .ok_or("native cache-hit metric absent")?;
        let raw_evidence = native.root.streams.clone();
        let evidence_sha256 = behavior(&raw_outcome, &raw_evidence, None)?;
        Ok(Run {
            native: Some(Box::new(native)),
            raw_outcome,
            raw_evidence,
            event_order: None,
            phases,
            wall_nanos,
            teardown_nanos,
            observed_teardown_nanos: 0,
            fresh_store_id,
            evidence_sha256,
            prepare_nanos,
            compile_nanos,
            cache_hit,
        })
    };
    let warmup = observed(0)?;
    let mut series = Series {
        schema: "gomad.native-warm-preparation/v1",
        role,
        binding,
        warmup,
        runs: Vec::with_capacity(repetitions),
    };
    for index in 1..=repetitions {
        series.runs.push(observed(index)?);
    }
    series.validate()?;
    Ok(series)
}

impl Series {
    /// # Errors
    /// Refuses incomplete, inconsistent or out-of-bound native observations.
    pub fn validate(&self) -> Result<(), String> {
        validate_binding(&self.binding)?;
        if self.schema != "gomad.native-warm-preparation/v1" || self.runs.len() < 5 {
            return Err("incomplete native warm series".into());
        }
        self.warmup.validate(false, false)?;
        validate_retained_binding(&self.binding, &self.warmup)?;
        let mut stores = BTreeSet::from([self.warmup.fresh_store_id]);
        for run in &self.runs {
            run.validate(self.role == Role::Candidate, true)?;
            validate_retained_binding(&self.binding, run)?;
            match (&self.warmup.native, &run.native) {
                (Some(before), Some(after)) => {
                    evidence::compare(before, after, ComparisonMask::Exact)?;
                }
                (None, None) => {}
                _ => {
                    return Err(
                        "warm series mixed typed and retained native observation sources".into(),
                    );
                }
            }
            if !stores.insert(run.fresh_store_id)
                || run.evidence_sha256 != self.warmup.evidence_sha256
            {
                return Err("warm preparation reused a Store or changed native behavior".into());
            }
        }
        if self.role == Role::Candidate && self.prepare_timing().median > 1_000_000_000 {
            return Err("warm preparation median exceeds one second".into());
        }
        Ok(())
    }

    #[must_use]
    pub fn prepare_timing(&self) -> TimingRange {
        timing(
            &self
                .runs
                .iter()
                .map(|run| run.prepare_nanos)
                .collect::<Vec<_>>(),
        )
    }
}

/// # Errors
/// Refuses mismatched bindings, roles or native warm series.
pub fn compare(before: &Series, after: &Series) -> Result<(), String> {
    before.validate()?;
    after.validate()?;
    if before.role != Role::Baseline || after.role != Role::Candidate {
        return Err("warm comparison needs baseline and candidate roles".into());
    }
    let b = &before.binding;
    let a = &after.binding;
    if b.host_boot_id != a.host_boot_id
        || b.host_cpu != a.host_cpu
        || b.host_os != a.host_os
        || b.host_arch != a.host_arch
        || b.engine_version != a.engine_version
        || b.engine_configuration != a.engine_configuration
        || b.module_sha256 != a.module_sha256
        || b.seed != a.seed
        || b.config_sha256 != a.config_sha256
        || b.runtime_identity != a.runtime_identity
        || b.model_sha256 != a.model_sha256
        || b.module_producer_sha256 != a.module_producer_sha256
    {
        return Err("warm comparison changed host, module, configuration, runtime or current producer closure".into());
    }
    if before.warmup.evidence_sha256 != after.warmup.evidence_sha256 {
        return Err("warm comparison changed canonical native behavior".into());
    }
    Ok(())
}

#[cfg(test)]
#[path = "warm_tests.rs"]
mod tests;

#[cfg(all(test, feature = "helper-determinism-fixture"))]
#[path = "terminal_controls.rs"]
mod terminal_controls;
