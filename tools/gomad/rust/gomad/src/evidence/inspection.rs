use super::{canonical, store::Opened};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Details {
    pub schema: String,
    pub outputs: Option<std::collections::BTreeMap<String, Output>>,
    pub choices: Option<ChoiceEvidence>,
    pub world: Option<World>,
    pub simulation: Option<Simulation>,
    pub minimization: Option<Minimization>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Output {
    pub full_sha256: String,
    pub total_bytes: u64,
    pub retained_bytes: u64,
    pub discarded_bytes: u64,
    pub truncated: bool,
}

impl Output {
    /// # Errors
    /// Refuses inconsistent output accounting or digest declarations.
    pub fn read(opened: &Opened, details: Option<&Details>, name: &str) -> Result<Self, String> {
        let stream = opened
            .manifest()
            .record
            .streams
            .get(&format!("output/{name}"));
        let size = stream.map_or(0, |stream| stream.bytes);
        let hash = stream.map_or_else(|| canonical::digest(&[]), |stream| stream.sha256.clone());
        let value = details
            .and_then(|details| details.outputs.as_ref())
            .and_then(|outputs| outputs.get(name));
        if let Some(value) = value {
            digest(&value.full_sha256)?;
            if value.retained_bytes != size
                || value.retained_bytes.checked_add(value.discarded_bytes)
                    != Some(value.total_bytes)
                || value.truncated != (value.discarded_bytes != 0)
                || !value.truncated && value.full_sha256 != hash
            {
                return Err("output stream accounting mismatch".into());
            }
            return Ok(Self {
                full_sha256: value.full_sha256.clone(),
                total_bytes: value.total_bytes,
                retained_bytes: size,
                discarded_bytes: value.discarded_bytes,
                truncated: value.truncated,
            });
        }
        Ok(Self {
            full_sha256: hash,
            total_bytes: size,
            retained_bytes: size,
            discarded_bytes: 0,
            truncated: false,
        })
    }
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Payload {
    pub schema: String,
    pub file: String,
    pub sha256: String,
    pub bytes: u64,
    pub limit: u64,
}

impl Payload {
    fn authenticate(&self, opened: &Opened) -> Result<Vec<u8>, String> {
        let bytes = opened.payload(&self.file, self.limit)?;
        if self.schema.is_empty()
            || bytes.len() as u64 != self.bytes
            || canonical::digest(&bytes) != self.sha256
        {
            return Err("inspection payload identity mismatch".into());
        }
        Ok(bytes)
    }
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ChoiceEvidence {
    pub profile: String,
    pub implementation_sha256: String,
    pub identity: linked::ChoiceIdentity,
    pub trace: Payload,
    pub records: u64,
    pub terminal_state: String,
    pub tape_sha256: Option<String>,
}

#[derive(Serialize)]
pub struct Choices<T = linked::TraceInspection> {
    pub schema: &'static str,
    pub profile: String,
    pub implementation_sha256: String,
    pub limit: u64,
    pub payload_bytes: u64,
    pub sha256: String,
    pub terminal_state: String,
    pub exact_replay_available: bool,
    #[serde(flatten)]
    pub trace: T,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct World {
    pub profile: String,
    pub implementation_sha256: String,
    pub seed: u64,
    pub now_nanos: i64,
    pub transitions: u64,
    pub terminal_kind: String,
    pub terminal_detail: String,
    pub history: Payload,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Simulation {
    pub profile: String,
    pub controller_sha256: String,
    pub execution_sha256: String,
    pub candidate_sha256: String,
    pub outcome_sha256: String,
    pub failure_sha256: Option<String>,
    pub plan: Payload,
    pub record: Payload,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Minimization {
    pub schema: String,
    pub implementation_sha256: String,
    pub parent_record_hash: String,
    pub parent_failure_signature: String,
    pub original_candidate_sha256: String,
    pub final_candidate_sha256: String,
    pub attempt_budget: u64,
    pub attempts: u64,
    pub original_forced_decisions: u64,
    pub final_forced_decisions: u64,
    pub accepted: Vec<Reduction>,
    pub predicate: Predicate,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Reduction {
    pub kind: String,
    pub before_sha256: String,
    pub after_sha256: String,
    pub removed: Vec<Decision>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Decision {
    pub dimension: String,
    pub ordinal: u64,
    pub identity: String,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Predicate {
    pub failure_signature: String,
    pub domain: String,
    pub reason: String,
    pub termination: String,
    pub replay_match: bool,
    pub choice_replay: String,
    pub simulation_replay: String,
}

fn digest(value: &str) -> Result<(), String> {
    if !canonical::valid_digest(value) {
        return Err("invalid inspection digest".into());
    }
    Ok(())
}

impl Details {
    /// # Errors
    /// Refuses unbound, malformed or inconsistent retained inspection evidence.
    pub fn read(opened: &Opened) -> Result<Option<Self>, String> {
        let name = "evidence/inspection.json";
        if !opened.manifest().files.contains_key(name) {
            return Ok(None);
        }
        if !opened.manifest().record.streams.contains_key(name) {
            return Err("inspection details are not bound by the record".into());
        }
        let bytes = opened.payload(name, 16 << 20)?;
        let details: Self = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
        if details.schema != "gomad.inspection-evidence/v1" || canonical::encode(&details)? != bytes
        {
            return Err("unsupported or noncanonical inspection evidence".into());
        }
        if details.outputs.as_ref().is_some_and(|outputs| {
            outputs
                .keys()
                .any(|name| !matches!(name.as_str(), "stdout" | "stderr"))
        }) {
            return Err("unknown output stream accounting".into());
        }
        if let Some(world) = &details.world {
            digest(&world.implementation_sha256)?;
            if world.history.schema != "gomad.world-expanded-snapshot/v1"
                || world.profile.is_empty()
                || world.terminal_kind.is_empty()
            {
                return Err("unsupported World inspection evidence".into());
            }
            let snapshot = world.history.authenticate(opened)?;
            if world.seed != opened.manifest().record.seed {
                return Err("World seed mismatch".into());
            }
            if snapshot.len() < 200
                || &snapshot[..12] != b"GMWSNAP1\0\0\0\x01"
                || u64::from_be_bytes(snapshot[12..20].try_into().map_err(|_| "World header")?)
                    != world.seed
                || i64::from_be_bytes(snapshot[64..72].try_into().map_err(|_| "World header")?)
                    != world.now_nanos
                || u64::from_be_bytes(snapshot[88..96].try_into().map_err(|_| "World header")?)
                    .checked_sub(1)
                    != Some(world.transitions)
            {
                return Err("World snapshot summary mismatch".into());
            }
        }
        if let Some(simulation) = &details.simulation {
            if simulation.profile.is_empty() {
                return Err("invalid simulation profile".into());
            }
            for hash in [
                &simulation.controller_sha256,
                &simulation.execution_sha256,
                &simulation.candidate_sha256,
                &simulation.outcome_sha256,
            ] {
                digest(hash)?;
            }
            if let Some(hash) = &simulation.failure_sha256 {
                digest(hash)?;
            }
            simulation.plan.authenticate(opened)?;
            simulation.record.authenticate(opened)?;
        }
        if let Some(minimization) = &details.minimization {
            minimization.validate()?;
        }
        Ok(Some(details))
    }
}

impl ChoiceEvidence {
    /// # Errors
    /// Refuses invalid metadata, malformed traces and inconsistent exact-tape declarations.
    pub fn inspect(&self, opened: &Opened) -> Result<Choices, String> {
        digest(&self.implementation_sha256)?;
        if self.profile != "gomad-choice-trace/v3"
            || self.trace.schema != self.profile
            || self.identity.toolchain_build_key != opened.manifest().record.identity.toolchain
            || !matches!(
                self.terminal_state.as_str(),
                "complete" | "overflow" | "interrupted" | "diverged"
            )
        {
            return Err("unsupported choice trace metadata".into());
        }
        let mut target = String::with_capacity(64);
        for byte in self.identity.target_sha256 {
            target.push(char::from(b"0123456789abcdef"[usize::from(byte >> 4)]));
            target.push(char::from(b"0123456789abcdef"[usize::from(byte & 15)]));
        }
        if target != opened.manifest().record.identity.target {
            return Err("choice target identity mismatch".into());
        }
        let outcome = &opened.manifest().record.outcome;
        let overflow = outcome.class == "runner_failure"
            && outcome.termination == "capacity"
            && outcome.exit_code.is_none()
            && outcome.reason == "WASI capacity: runtime-choice-bytes";
        let interrupted = outcome.class == "runner_failure"
            && outcome.exit_code.is_none()
            && (outcome.termination == "watchdog_timeout"
                || outcome.termination == "storage_crash"
                || outcome.termination == "cancelled" && outcome.reason == "runner_cancelled");
        let diverged = outcome.class == "target_failure"
            && matches!(outcome.termination.as_str(), "divergence" | "diverged")
            && outcome.exit_code.is_none();
        if match self.terminal_state.as_str() {
            "overflow" => !overflow,
            "interrupted" => !interrupted,
            "diverged" => !diverged,
            _ => overflow || interrupted || diverged,
        } {
            return Err("choice terminal outcome mismatch".into());
        }
        let bytes = self.trace.authenticate(opened)?;
        let mut trace = linked::inspect_choices(&bytes, &self.identity).map_err(str::to_owned)?;
        if trace.records != self.records || bytes.len() as u64 + 64 > self.trace.limit {
            return Err("choice trace bounds mismatch".into());
        }
        let exact = self.tape_sha256.is_some();
        if self.terminal_state == "complete" && !exact {
            return Err("complete choice tape is missing".into());
        }
        if let Some(tape) = &self.tape_sha256
            && (self.terminal_state != "complete" || tape != &trace.tape_sha256)
        {
            return Err("choice tape identity mismatch".into());
        }
        if !exact {
            trace.tape_sha256.clear();
            trace.replay_decisions.clear();
            trace.decisions = 0;
        }
        Ok(Choices {
            schema: "gomad.choice-inspection/v3",
            profile: self.profile.clone(),
            implementation_sha256: self.implementation_sha256.clone(),
            limit: self.trace.limit,
            payload_bytes: self.trace.bytes,
            sha256: self.trace.sha256.clone(),
            terminal_state: self.terminal_state.clone(),
            exact_replay_available: exact,
            trace,
        })
    }
}

impl Minimization {
    fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.minimization-evidence/v1"
            || self.attempt_budget == 0
            || self.attempts == 0
            || self.attempts > self.attempt_budget
            || self.final_forced_decisions >= self.original_forced_decisions
            || self.accepted.is_empty()
            || self.accepted.len() as u64 > self.attempts
            || self.parent_failure_signature != self.predicate.failure_signature
            || !self.predicate.replay_match
        {
            return Err("invalid minimization evidence".into());
        }
        for hash in [
            &self.implementation_sha256,
            &self.parent_record_hash,
            &self.parent_failure_signature,
            &self.original_candidate_sha256,
            &self.final_candidate_sha256,
            &self.predicate.failure_signature,
        ] {
            digest(hash)?;
        }
        let mut previous = &self.original_candidate_sha256;
        let mut removed = std::collections::BTreeSet::new();
        let mut candidates = std::collections::BTreeSet::from([previous]);
        let mut remaining = self.original_forced_decisions;
        for reduction in &self.accepted {
            digest(&reduction.before_sha256)?;
            digest(&reduction.after_sha256)?;
            if &reduction.before_sha256 != previous
                || reduction.removed.is_empty()
                || !matches!(
                    reduction.kind.as_str(),
                    "schedule_suffix" | "schedule_range" | "fault_entries"
                )
                || !candidates.insert(&reduction.after_sha256)
            {
                return Err("invalid minimization reduction chain".into());
            }
            remaining = remaining
                .checked_sub(reduction.removed.len() as u64)
                .ok_or("invalid minimization removal count")?;
            for (index, decision) in reduction.removed.iter().enumerate() {
                digest(&decision.identity)?;
                if !matches!(
                    decision.dimension.as_str(),
                    "runtime" | "scenario" | "network" | "storage" | "fault" | "crash"
                ) || !removed.insert((&decision.dimension, decision.ordinal))
                    || reduction.kind == "fault_entries" && decision.dimension != "fault"
                    || reduction.kind == "schedule_range" && decision.dimension == "fault"
                    || reduction.kind == "schedule_suffix"
                        && (decision.dimension != "runtime"
                            || self.predicate.simulation_replay == "not_present"
                                && decision.ordinal != remaining + index as u64)
                {
                    return Err("invalid minimization removed decision".into());
                }
            }
            previous = &reduction.after_sha256;
        }
        if previous != &self.final_candidate_sha256 {
            return Err("minimization final candidate mismatch".into());
        }
        if remaining != self.final_forced_decisions {
            return Err("invalid minimization final decision count".into());
        }
        Ok(())
    }
}
