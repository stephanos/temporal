use super::{
    evidence::{self, ComparisonMask, NativeRun, VariantRole},
    operations::Counts,
    statistics::{self, NamespaceSample},
};
use crate::evidence::canonical;
use serde_json::{Value, json};
use std::collections::{BTreeMap, BTreeSet};

pub const SEEDS: [u64; 4] = [0, 1_000_003, 2_000_006, 3_000_009];
pub const COUNTER_DOMAIN: &str =
    "NoRemote::write callback-frame write attempts; not completed filesystem calls";
pub const NAMESPACE_BOUNDARY: &str =
    "helper-owned ABI namespace dispatch; fallback callbacks write through NoRemote";

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Role {
    Baseline,
    Candidate,
}
impl Role {
    fn name(self) -> &'static str {
        match self {
            Self::Baseline => "baseline",
            Self::Candidate => "candidate",
        }
    }
}

pub struct Common {
    pub requests: BTreeMap<u64, Value>,
    pub environment_identity: String,
    pub runtime_identity: String,
    pub choice_identity: gomad_wasmhost::linked::ChoiceIdentity,
    pub current_module_producer_sha256: String,
    pub native_role: VariantRole,
}

pub struct Observation {
    pub run: NativeRun,
    pub fresh_store_id: u64,
    /// Every first repetition carries an actual full-tape native replay.
    pub replay: Option<NativeRun>,
}

pub struct Series {
    pub common: Common,
    pub role: Role,
    pub observations: Vec<Observation>,
    pub samples: Vec<NamespaceSample>,
}

fn validate_common(common: &Common) -> Result<(), String> {
    if common.requests.keys().copied().collect::<Vec<_>>() != SEEDS
        || common.environment_identity.is_empty()
        || common.runtime_identity.is_empty()
        || common.choice_identity.toolchain_build_key != common.runtime_identity
        || !canonical::valid_digest(&common.current_module_producer_sha256)
    {
        return Err("namespace collection requires four immutable prepared seed requests and source/runtime bindings".into());
    }
    let target = super::workloads::hex(&common.choice_identity.target_sha256);
    let mut immutable = None;
    for (seed, request) in &common.requests {
        if request["guest"]["module_sha256"] != target
            || request["kind"] != "standalone"
            || request["guest"]["model"]["configuration"]["seed"] != *seed
            || !request["guest"]["module_sha256"]
                .as_str()
                .is_some_and(canonical::valid_digest)
            || request["guest"]["namespace"].is_null()
        {
            return Err("namespace sample requires an actual prepared standalone namespace".into());
        }
        let normalized = normalize_seed_request(request, *seed)?;
        if immutable.as_ref().is_some_and(|prior| prior != &normalized) {
            return Err("namespace seeds changed common immutable request inputs".into());
        }
        immutable = Some(normalized);
    }
    Ok(())
}

fn normalize_seed_request(request: &Value, seed: u64) -> Result<Value, String> {
    let mut normalized = request.clone();
    normalized["guest"]["model"]["configuration"]["seed"] = 0.into();
    if let Some(environment) = normalized["guest"]["model"].get_mut("environment") {
        let entries = environment
            .as_array_mut()
            .ok_or("namespace environment must be explicit strings")?;
        let mut seed_present = false;
        let mut original_bytes = 0usize;
        for entry in entries.iter_mut() {
            let value = entry
                .as_str()
                .ok_or("namespace environment entry must be a string")?;
            original_bytes = original_bytes
                .checked_add(value.len() + 1)
                .ok_or("namespace environment size overflow")?;
            if let Some(value) = value.strip_prefix("GOMADSEED=") {
                if seed_present || value != seed.to_string() {
                    return Err("namespace derived seed environment mismatch".into());
                }
                seed_present = true;
                *entry = "GOMADSEED=0".into();
            }
        }
        let normalized_bytes = entries
            .iter()
            .map(|value| value.as_str().unwrap_or_default().len() + 1)
            .sum::<usize>();
        let count = entries.len();
        let declaration = &mut normalized["guest"]["model"]["configuration"]["environment"];
        if !declaration.is_null() && *declaration != json!({"bytes":0,"count":0,"chunks":0}) {
            if *declaration
                != json!({"bytes":original_bytes,"count":count,"chunks":original_bytes.div_ceil(64*1024)})
            {
                return Err("namespace derived environment declaration mismatch".into());
            }
            *declaration = json!({"bytes":normalized_bytes,"count":count,"chunks":normalized_bytes.div_ceil(64*1024)});
        }
    }
    Ok(normalized)
}

/// Counts retained native operations without converting missing bytes into an empty counter.
/// # Errors
/// Refuses missing or malformed bounded canonical transcripts.
pub fn operation_retention(run: &NativeRun) -> Result<Counts, String> {
    let mut aggregate = Counts::default();
    let mut found = false;
    for execution in std::iter::once(&run.root).chain(&run.incarnations) {
        let Some(raw) = execution.streams.get("transcript") else {
            continue;
        };
        found = true;
        let counts = super::count_operations(raw, 64 << 20)?;
        aggregate.calls = aggregate
            .calls
            .checked_add(counts.calls)
            .ok_or("native operation count overflow")?;
        aggregate.local = aggregate
            .local
            .checked_add(counts.local)
            .ok_or("native local count overflow")?;
        aggregate.remote = aggregate
            .remote
            .checked_add(counts.remote)
            .ok_or("native remote count overflow")?;
        for (name, count) in counts.operations {
            let total = aggregate.operations.entry(name).or_default();
            *total = total
                .checked_add(count)
                .ok_or("native operation count overflow")?;
        }
    }
    if !found {
        return Err("blocked: actual retained native transcript is absent".into());
    }
    Ok(aggregate)
}

fn counter(run: &NativeRun, store_id: u64) -> Result<BTreeMap<String, u64>, String> {
    let counter = run
        .report
        .get("native_namespace_counter")
        .filter(|value| !value.is_null())
        .ok_or("blocked: actual native namespace/external-owner counter seam is absent")?;
    if counter["schema"] != "gomad.native-namespace-counter/v1"
        || counter["counter_domain"] != COUNTER_DOMAIN
        || counter["namespace_boundary"] != NAMESPACE_BOUNDARY
        || counter["implementation_sha256"] != run.report["linked_implementation_sha256"]
        || !counter["implementation_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
        || counter["fresh_store_id"] != store_id
        || store_id == 0
    {
        return Err(
            "native namespace counter implementation or fresh Store binding mismatch".into(),
        );
    }
    let owner_calls = run.profile["owners"]["namespace"]["count"]
        .as_u64()
        .filter(|count| *count > 0)
        .ok_or("actual positive native namespace ownership attribution is absent")?;
    let abi_calls = run.profile["abi_ops"]
        .as_object()
        .ok_or("actual namespace ABI attribution is absent")?
        .values()
        .try_fold(0u64, |sum, row| {
            let count = row["served_by"]
                .get("namespace")
                .map(|value| value.as_u64().ok_or("invalid namespace ABI counter"))
                .transpose()?
                .unwrap_or(0);
            sum.checked_add(count)
                .ok_or("namespace owner count overflow")
        })?;
    if abi_calls == 0 || counter["namespace_calls"].as_u64() != Some(owner_calls) {
        return Err(
            "actual positive native namespace ownership attribution is absent or inconsistent"
                .into(),
        );
    }
    serde_json::from_value(
        counter
            .get("external_owner_calls")
            .ok_or("blocked: actual external-owner counter field is absent")?
            .clone(),
    )
    .map_err(|error| format!("invalid actual external-owner counter: {error}"))
}

fn semantic_digest(run: &NativeRun) -> Result<String, String> {
    let mut outcome = run.root.outcome.clone();
    let object = outcome
        .as_object_mut()
        .ok_or("native outcome object absent")?;
    for key in ["metrics", "fuel_remaining", "peak_memory_bytes"] {
        object.remove(key);
    }
    Ok(canonical::digest(&canonical::encode(
        &json!({"outcome":outcome,"streams":run.root.streams}),
    )?))
}

fn sample(
    common: &Common,
    role: Role,
    seed: u64,
    repetition: usize,
    observation: &Observation,
) -> Result<NamespaceSample, String> {
    if common.requests.get(&seed) != Some(&observation.run.request) {
        return Err("namespace sample changed immutable prepared request".into());
    }
    if observation.run.root.outcome["termination"] != "exit"
        || observation.run.root.outcome["exit_code"] != 0
    {
        return Err("namespace sample native terminal failed".into());
    }
    let mut selected = super::workloads::RunSpec::new(
        "namespace-observation",
        "^TestActivityTestSuite$",
        seed,
        role.name(),
    );
    selected.warm = true;
    super::workloads::success(&observation.run, &selected)?;
    operation_retention(&observation.run)?;
    evidence::compare(
        &observation.run,
        &observation.run,
        ComparisonMask::Variant {
            baseline: common.native_role,
            candidate: common.native_role,
            gc: None,
        },
    )?;
    let external = counter(&observation.run, observation.fresh_store_id)?;
    if role == Role::Candidate && !external.is_empty() {
        return Err("namespace candidate attempted a helper external callback".into());
    }
    let authorized = observation.run.report["authorization_to_terminal_nanos"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("blocked: actual native authorization-to-terminal interval is absent")?;
    let model = &observation.run.root.outcome["local_model"];
    let calls = model["calls"]
        .as_u64()
        .ok_or("actual model call count absent")?;
    let advances = model["advances"]
        .as_u64()
        .ok_or("actual model advance count absent")?;
    if repetition == 0 {
        verify_full_replay(
            common,
            &observation.run,
            observation
                .replay
                .as_ref()
                .ok_or("per-seed native full-tape replay absent")?,
        )?;
    }
    Ok(NamespaceSample {
        role: role.name().into(),
        seed,
        repetition,
        authorized_nanos: authorized,
        evidence_sha256: semantic_digest(&observation.run)?,
        environment_identity: common.environment_identity.clone(),
        calls,
        time_advances: advances,
        counter_present: true,
        external_filesystem_calls: external,
    })
}

fn verify_full_replay(
    common: &Common,
    recorded: &NativeRun,
    replay: &NativeRun,
) -> Result<(), String> {
    let declaration: gomad_wasmhost::linked::ReplayRequest =
        serde_json::from_value(replay.request["guest"]["model"]["configuration"]["replay"].clone())
            .map_err(|error| error.to_string())?;
    if declaration.execution_identity.toolchain_build_key != common.runtime_identity {
        return Err("native replay changed the immutable runtime identity".into());
    }
    evidence::compare_replay(recorded, replay, &common.choice_identity)
}

/// Collects four matched seeds and five fresh executions per seed from actual retained observations.
/// # Errors
/// Blocks on missing native counter/timing seams and refuses changed behavior or incomplete replays.
pub fn collect(
    common: Common,
    role: Role,
    mut execute: impl FnMut(u64, usize) -> Result<Observation, String>,
) -> Result<Series, String> {
    validate_common(&common)?;
    let mut observations = Vec::new();
    let mut samples = Vec::new();
    let mut stores = BTreeSet::new();
    for seed in SEEDS {
        for repetition in 0..5 {
            let observation = execute(seed, repetition)?;
            if !stores.insert(observation.fresh_store_id) {
                return Err("namespace measurement reused a native Store".into());
            }
            let row = sample(&common, role, seed, repetition, &observation)?;
            if let Some(first) = samples
                .iter()
                .find(|prior: &&NamespaceSample| prior.seed == seed)
                && (row.evidence_sha256 != first.evidence_sha256
                    || row.calls != first.calls
                    || row.time_advances != first.time_advances)
            {
                return Err(
                    "namespace repetition changed canonical evidence or model counts".into(),
                );
            }
            samples.push(row);
            observations.push(observation);
        }
    }
    Ok(Series {
        common,
        role,
        observations,
        samples,
    })
}

/// # Errors
/// Refuses series with changed immutable bindings, invalid samples or failed comparison gates.
pub fn compare(before: &Series, after: &Series) -> Result<(), String> {
    validate_common(&before.common)?;
    validate_common(&after.common)?;
    if before.role != Role::Baseline
        || after.role != Role::Candidate
        || before.common.environment_identity != after.common.environment_identity
        || before.common.runtime_identity != after.common.runtime_identity
        || canonical::encode(&before.common.choice_identity)?
            != canonical::encode(&after.common.choice_identity)?
        || before.common.current_module_producer_sha256
            != after.common.current_module_producer_sha256
    {
        return Err(
            "namespace comparison changed immutable module, runtime, producer or environment"
                .into(),
        );
    }
    if before.observations.len() != 20 || after.observations.len() != 20 {
        return Err("namespace comparison requires actual twenty-run retained cohorts".into());
    }
    let mut baseline = Vec::new();
    let mut candidate = Vec::new();
    let mut before_stores = BTreeSet::new();
    let mut after_stores = BTreeSet::new();
    for (index, (b, a)) in before
        .observations
        .iter()
        .zip(&after.observations)
        .enumerate()
    {
        if !before_stores.insert(b.fresh_store_id) || !after_stores.insert(a.fresh_store_id) {
            return Err("namespace comparison reused a native Store".into());
        }
        let seed = SEEDS[index / 5];
        let repetition = index % 5;
        baseline.push(sample(&before.common, Role::Baseline, seed, repetition, b)?);
        candidate.push(sample(&after.common, Role::Candidate, seed, repetition, a)?);
        evidence::compare(
            &b.run,
            &a.run,
            ComparisonMask::Variant {
                baseline: before.common.native_role,
                candidate: after.common.native_role,
                gc: None,
            },
        )?;
    }
    statistics::compare_namespace(&baseline, &candidate)
}

#[cfg(test)]
#[path = "namespace_tests.rs"]
mod tests;
