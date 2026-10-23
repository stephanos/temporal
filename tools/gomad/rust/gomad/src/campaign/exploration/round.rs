use super::{
    choice::{ChoiceConfig, ChoiceRoundSegment, ChoiceState},
    simulation::{SimulationConfig, SimulationRoundSegment, SimulationState},
};
use crate::{
    campaign::ledger::{self, Execution, validate_execution},
    evidence::{
        canonical,
        record::Record,
        store::{Opened, open},
    },
};
use rustix::fs::{self, AtFlags, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    fs::File,
    io::Write,
    path::{Path, PathBuf},
};

const FORMAT: &str = "gomad.exploration-round-storage/v1";
const MAX_PLAN_BYTES: u64 = 16 << 20;

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct RoundPlan {
    pub prepared: Value,
    pub strategy: RoundStrategy,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(
    tag = "kind",
    content = "config",
    rename_all = "snake_case",
    deny_unknown_fields
)]
pub(crate) enum RoundStrategy {
    Choice(ChoiceConfig),
    Simulation(SimulationConfig),
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "snake_case", deny_unknown_fields)]
pub(crate) enum RoundEvent {
    Choice {
        executions: Vec<Execution>,
        segment: ChoiceRoundSegment,
    },
    Simulation {
        executions: Vec<Execution>,
        segment: SimulationRoundSegment,
        evidence: Vec<SimulationEvidence>,
    },
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SimulationEvidence {
    pub record: Vec<u8>,
    pub runtime_trace: Vec<u8>,
}

impl RoundEvent {
    pub(crate) fn executions(&self) -> &[Execution] {
        match self {
            Self::Choice { executions, .. } | Self::Simulation { executions, .. } => executions,
        }
    }
}

#[derive(Clone)]
pub(crate) enum RoundState {
    Choice(Box<ChoiceState>),
    Simulation(Box<SimulationState>),
}

impl RoundState {
    fn new(plan: &RoundPlan) -> Result<Self, String> {
        let prepared = &plan.prepared;
        super::plan::duration_metadata(prepared)?;
        for key in ["module_sha256", "toolchain", "host_sha256", "helper_sha256"] {
            if !prepared[key].as_str().is_some_and(canonical::valid_digest) {
                return Err(format!("invalid exploration prepared {key}"));
            }
        }
        if !prepared["prepared_artifact"]
            .as_str()
            .is_some_and(|value| value.starts_with("prepared/"))
            || !prepared["configuration"].is_object()
            || !prepared["choice_identity"].is_object()
            || prepared["journal_bytes"]
                .as_u64()
                .is_none_or(|value| value == 0)
        {
            return Err("incomplete exploration prepared plan".into());
        }
        match &plan.strategy {
            RoundStrategy::Choice(config) => {
                if serde_json::to_value(&config.execution).map_err(|error| error.to_string())?
                    != prepared["choice_identity"]
                    || super::choice::hex(&config.execution.target_sha256)
                        != prepared["module_sha256"]
                            .as_str()
                            .ok_or("missing prepared module")?
                    || config.execution.toolchain_build_key
                        != prepared["toolchain"]
                            .as_str()
                            .ok_or("missing prepared toolchain")?
                    || config.execution.runtime_implementation_sha256()
                        != prepared["configuration"]["runtime_sha256"]
                            .as_str()
                            .ok_or("missing prepared runtime")?
                {
                    return Err("choice exploration identity differs from its prepared plan".into());
                }
                Ok(Self::Choice(Box::new(ChoiceState::new(config.clone())?)))
            }
            RoundStrategy::Simulation(config) => {
                let keys = [
                    "module_sha256",
                    "toolchain",
                    "configuration",
                    "scenario",
                    "network",
                    "choice_identity",
                    "soft_assertion_report_only",
                    "host_sha256",
                    "helper_sha256",
                ];
                let identity: serde_json::Map<String, Value> = keys
                    .into_iter()
                    .map(|key| (key.into(), prepared[key].clone()))
                    .collect();
                if config.execution_sha256
                    != super::hash("gomad-simulation-exploration-execution/v1", &identity)?
                {
                    return Err(
                        "simulation exploration identity differs from its prepared plan".into(),
                    );
                }
                Ok(Self::Simulation(Box::new(SimulationState::new(
                    config.clone(),
                )?)))
            }
        }
    }

    pub(crate) fn summary(&self) -> Result<Value, String> {
        match self {
            Self::Choice(state) => Ok(state.summary()),
            Self::Simulation(state) => {
                serde_json::to_value(state.summary()).map_err(|error| error.to_string())
            }
        }
    }

    #[expect(
        clippy::too_many_lines,
        reason = "authenticate all round evidence before applying its state transition"
    )]
    fn apply(&mut self, event: &RoundEvent, plan: &RoundPlan) -> Result<(), String> {
        match (&*self, event) {
            (
                Self::Choice(state),
                RoundEvent::Choice {
                    executions,
                    segment,
                },
            ) => {
                let round = state.next_round().ok_or("choice round follows its stop")?;
                if executions.len() != round.candidates.len()
                    || executions.len() != segment.results.len()
                {
                    return Err("choice round execution count differs from its segment".into());
                }
                for (index, (execution, result)) in
                    executions.iter().zip(&segment.results).enumerate()
                {
                    validate_observation(
                        execution,
                        state.logical_executions,
                        index,
                        state.config.base_seed,
                    )?;
                    if execution.divergence != result.divergence
                        || result.failed
                            != (is_failure(execution) && execution.divergence.is_none())
                        || result.failed
                            && result.failure_sha256
                                != format!("sha256:{}", execution.record.failure_signature()?)
                        || result.divergence.is_none()
                            && result.outcome_sha256
                                != super::hash(
                                    "gomad-choice-exploration-outcome/v2",
                                    &execution.record,
                                )?
                    {
                        return Err("choice round result differs from its execution record".into());
                    }
                    let has_trace = execution.record.streams.contains_key("evidence/choice");
                    if result.divergence.is_none() && has_trace != result.source_trace.is_some() {
                        return Err("choice round result omits its retained source trace".into());
                    }
                }
                *self = Self::Choice(Box::new(state.replay_segment(segment)?));
            }
            (
                Self::Simulation(state),
                RoundEvent::Simulation {
                    executions,
                    segment,
                    evidence,
                },
            ) => {
                let round = state
                    .next_round()
                    .ok_or("simulation round follows its stop")?;
                if executions.len() != round.candidates.len()
                    || executions.len() != segment.results.len()
                    || executions.len() != evidence.len()
                {
                    return Err("simulation round execution count differs from its segment".into());
                }
                for (index, (execution, result)) in
                    executions.iter().zip(&segment.results).enumerate()
                {
                    validate_observation(
                        execution,
                        state.logical_executions,
                        index,
                        state.config.base_seed,
                    )?;
                    if result.failed
                        && result.failure_sha256
                            != format!("sha256:{}", execution.record.failure_signature()?)
                    {
                        return Err(
                            "simulation round result differs from its execution record".into()
                        );
                    }
                    let raw = &evidence[index];
                    validate_stream(&execution.record, "simulation/record.json", &raw.record)?;
                    validate_stream(&execution.record, "evidence/choice", &raw.runtime_trace)?;
                    let identity = serde_json::from_value(plan.prepared["choice_identity"].clone())
                        .map_err(|error| error.to_string())?;
                    let control = super::simulation_record_wire::execution_for_candidate(
                        &state.config,
                        &round.candidates[index],
                        &identity,
                    )?;
                    validate_stream(
                        &execution.record,
                        "simulation/plan.json",
                        &control.simulation_plan,
                    )?;
                    let mut runtime =
                        super::simulation_record::runtime_decisions(&raw.runtime_trace, &identity)?;
                    for decision in &mut runtime {
                        decision.source_trace.clone_from(&raw.runtime_trace);
                    }
                    let mut regenerated = super::simulation_record::result_for_record(
                        &state.config,
                        &round.candidates[index],
                        &raw.record,
                        &runtime,
                    )?;
                    if is_failure(execution)
                        && execution.divergence.is_none()
                        && !regenerated.diverged
                    {
                        regenerated.failed = true;
                        regenerated.failure_sha256 =
                            format!("sha256:{}", execution.record.failure_signature()?);
                    }
                    regenerated.diverged |= execution.divergence.is_some();
                    if &regenerated != result {
                        let expected = serde_json::to_value(&regenerated)
                            .map_err(|error| error.to_string())?;
                        let observed =
                            serde_json::to_value(result).map_err(|error| error.to_string())?;
                        let fields = expected
                            .as_object()
                            .ok_or("simulation result is not an object")?
                            .iter()
                            .filter(|(name, value)| observed.get(*name) != Some(*value))
                            .map(|(name, _)| name.as_str())
                            .collect::<Vec<_>>();
                        return Err(format!(
                            "simulation segment differs from authenticated raw decisions: {}",
                            fields.join(", ")
                        ));
                    }
                }
                *self = Self::Simulation(Box::new(state.replay_segment(segment)?));
            }
            _ => return Err("exploration round strategy differs from its plan".into()),
        }
        Ok(())
    }
}

fn validate_stream(record: &Record, name: &str, bytes: &[u8]) -> Result<(), String> {
    let stream = record
        .streams
        .get(name)
        .ok_or_else(|| format!("exploration evidence has no recorded {name} stream"))?;
    if stream.bytes != bytes.len() as u64 || stream.sha256 != canonical::digest(bytes) {
        return Err(format!(
            "exploration evidence differs from its recorded {name} stream"
        ));
    }
    Ok(())
}

fn is_failure(execution: &Execution) -> bool {
    matches!(
        execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    )
}

fn validate_observation(
    execution: &Execution,
    logical: u64,
    index: usize,
    seed: u64,
) -> Result<(), String> {
    validate_execution(execution)?;
    if execution.ordinal
        != logical
            .checked_add(index as u64)
            .ok_or("round ordinal overflow")?
        || execution.seed != seed
        || execution.record.outcome.class == "runner_failure"
    {
        return Err(
            "exploration execution ordinal, seed or artifact differs from its round".into(),
        );
    }
    Ok(())
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Storage {
    format: String,
    max_bytes: u64,
    plan_sha256: String,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Envelope {
    ordinal: u64,
    previous_sha256: String,
    event: RoundEvent,
    sha256: String,
}

impl Envelope {
    fn digest(&self) -> Result<String, String> {
        Ok(canonical::digest(&canonical::encode(&serde_json::json!({
            "ordinal": self.ordinal,
            "previous_sha256": self.previous_sha256,
            "event": self.event,
        }))?))
    }
}

pub struct RoundJournal {
    root: File,
    path: PathBuf,
    lock: Option<File>,
    writer: Option<File>,
    plan: RoundPlan,
    state: RoundState,
    events: Vec<RoundEvent>,
    previous: String,
    bytes: u64,
    max_bytes: u64,
    poisoned: bool,
    observed: fs::Stat,
    recovery_changed: bool,
}

impl RoundJournal {
    /// # Errors
    /// Refuses existing campaign state, invalid plans, unsafe entries and durability failures.
    pub fn create(path: &Path, plan: &RoundPlan, max_bytes: u64) -> Result<Self, String> {
        let state = RoundState::new(plan)?;
        if max_bytes == 0 || plan.prepared["journal_bytes"].as_u64() != Some(max_bytes) {
            return Err("invalid campaign journal byte bound".into());
        }
        let plan_bytes = canonical::encode(plan)?;
        if plan_bytes.len() as u64 > MAX_PLAN_BYTES {
            return Err("campaign plan exceeds byte bound".into());
        }
        let parent_path = path
            .parent()
            .filter(|parent| !parent.as_os_str().is_empty())
            .unwrap_or_else(|| Path::new("."));
        let name = path
            .file_name()
            .and_then(|name| name.to_str())
            .ok_or("invalid campaign path")?;
        if open::components(name)?.len() != 1 {
            return Err("invalid campaign path".into());
        }
        let parent = File::from(
            fs::open(
                parent_path,
                OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                Mode::empty(),
            )
            .map_err(|error| error.to_string())?,
        );
        match fs::mkdirat(&parent, name, Mode::from_raw_mode(0o700)) {
            Ok(()) => {}
            Err(error) if error == rustix::io::Errno::EXIST => {}
            Err(error) => return Err(error.to_string()),
        }
        let root = open::child(&parent, name, true, false)?;
        for entry in open::entries(&root)? {
            if !matches!(
                entry.as_str(),
                "prepared" | crate::campaign::prepare::OBSERVATION_DIRECTORY
            ) {
                return Err(
                    "campaign creation requires an empty or authenticated preparation-only directory".into(),
                );
            }
            open::child(&root, &entry, true, false)?;
        }
        crate::campaign::authenticate_preparation_observations(&root, &plan.prepared)?;
        let lock = create_file(&root, ".lock", &[])?;
        fs::flock(&lock, FlockOperation::NonBlockingLockExclusive)
            .map_err(|error| error.to_string())?;
        let previous = canonical::digest(&plan_bytes);
        let storage = Storage {
            format: FORMAT.into(),
            max_bytes,
            plan_sha256: previous.clone(),
        };
        create_file(&root, "storage.json", &canonical::encode(&storage)?)?;
        create_file(&root, "plan.json", &plan_bytes)?;
        let writer = create_file(&root, "journal.jsonl", &[])?;
        let observed = fs::fstat(&writer).map_err(|error| error.to_string())?;
        root.sync_all().map_err(|error| error.to_string())?;
        parent.sync_all().map_err(|error| error.to_string())?;
        let mut journal = Self {
            root,
            path: path.into(),
            lock: Some(lock),
            writer: Some(writer),
            plan: plan.clone(),
            state,
            events: Vec::new(),
            previous,
            bytes: 0,
            max_bytes,
            poisoned: false,
            observed,
            recovery_changed: false,
        };
        journal.authenticate_prepared()?;
        journal.bind_scope_module()?;
        Ok(journal)
    }

    /// # Errors
    /// Refuses unsupported lineage, corrupt records, unsafe references and unavailable writer locks.
    #[expect(
        clippy::too_many_lines,
        reason = "Replay and validation must finish before a torn append can be repaired"
    )]
    pub fn open(path: &Path, writable: bool) -> Result<Self, String> {
        let root = open::pin(path)?;
        let storage_bytes = open::read(&root, "storage.json", 4096, false)
            .map_err(|error| format!("unsupported campaign storage format: {error}"))?;
        let storage: Storage = serde_json::from_slice(&storage_bytes)
            .map_err(|_| "unsupported campaign storage format")?;
        if storage.format != FORMAT || canonical::encode(&storage)? != storage_bytes {
            return Err("unsupported campaign storage format".into());
        }
        if storage.max_bytes == 0 || !canonical::valid_digest(&storage.plan_sha256) {
            return Err("invalid campaign storage bounds or identity".into());
        }
        for name in open::entries(&root)? {
            match name.as_str() {
                "storage.json" | "plan.json" | "journal.jsonl" | ".lock" => {
                    open::child(&root, &name, false, false)?;
                }
                "evidence" | "prepared" | crate::campaign::prepare::OBSERVATION_DIRECTORY => {
                    open::child(&root, &name, true, false)?;
                }
                _ => return Err(format!("unexpected campaign entry {name:?}")),
            }
        }
        let lock = open::child(&root, ".lock", false, false)?;
        if writable {
            fs::flock(&lock, FlockOperation::NonBlockingLockExclusive)
                .map_err(|error| format!("campaign writer lock unavailable: {error}"))?;
            verify_named(&root, ".lock", &lock)?;
        }
        let plan_bytes = open::read(&root, "plan.json", MAX_PLAN_BYTES, false)?;
        if canonical::digest(&plan_bytes) != storage.plan_sha256 {
            return Err("campaign plan hash mismatch".into());
        }
        let plan: RoundPlan =
            serde_json::from_slice(&plan_bytes).map_err(|error| error.to_string())?;
        if canonical::encode(&plan)? != plan_bytes {
            return Err("campaign plan is not canonical".into());
        }
        if plan.prepared["journal_bytes"].as_u64() != Some(storage.max_bytes) {
            return Err("campaign journal bound differs from its plan".into());
        }
        let state = RoundState::new(&plan)?;
        crate::campaign::authenticate_preparation_observations(&root, &plan.prepared)?;
        let writer = if writable {
            Some(writable_file(&root, "journal.jsonl")?)
        } else {
            None
        };
        let observed = fs::fstat(&open::child(&root, "journal.jsonl", false, false)?)
            .map_err(|error| error.to_string())?;
        let data = open::read(&root, "journal.jsonl", storage.max_bytes, false)?;
        let after_read = fs::statat(&root, "journal.jsonl", AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if !open::same(&observed, &after_read) {
            return Err("campaign journal changed while reading".into());
        }
        if let Some(file) = &writer {
            verify_named(&root, "journal.jsonl", file)?;
            if file.metadata().map_err(|error| error.to_string())?.len() != data.len() as u64 {
                return Err("campaign journal changed while opening writer".into());
            }
        }
        let mut journal = Self {
            root,
            path: path.into(),
            lock: writable.then_some(lock),
            writer,
            plan,
            state,
            events: Vec::new(),
            previous: storage.plan_sha256,
            bytes: 0,
            max_bytes: storage.max_bytes,
            poisoned: false,
            observed,
            recovery_changed: false,
        };
        if matches!(&journal.state, RoundState::Choice(state) if state.config.site_scope.is_some())
        {
            journal.authenticate_prepared()?;
            journal.bind_scope_module()?;
        }
        let complete_bytes = data
            .iter()
            .rposition(|byte| *byte == b'\n')
            .map_or(0, |index| index + 1);
        if complete_bytes != 0 {
            for line in data[..complete_bytes - 1].split(|byte| *byte == b'\n') {
                journal.consume(line)?;
            }
        }
        journal.authenticate_prepared()?;
        if complete_bytes != data.len() {
            let tail = &data[complete_bytes..];
            match serde_json::from_slice::<Envelope>(tail) {
                Ok(envelope) => {
                    journal.validate_envelope(&envelope, tail)?;
                    journal
                        .state
                        .clone()
                        .apply(&envelope.event, &journal.plan)?;
                    journal.authenticate_event(&envelope.event)?;
                }
                Err(error)
                    if error.is_eof()
                        && (b"{\"event\":".starts_with(tail)
                            || tail.starts_with(b"{\"event\":")) => {}
                _ => return Err("campaign journal has a malformed final record".into()),
            }
            if !writable {
                return Err(
                    "campaign journal has a torn final append; writable recovery required".into(),
                );
            }
            let writer = journal.writer.as_ref().ok_or("campaign writer missing")?;
            verify_named(&journal.root, "journal.jsonl", writer)?;
            if !open::same(
                &journal.observed,
                &fs::fstat(writer).map_err(|error| error.to_string())?,
            ) {
                return Err("campaign journal changed before torn-tail repair".into());
            }
            writer
                .set_len(complete_bytes as u64)
                .map_err(|error| error.to_string())?;
            writer.sync_all().map_err(|error| error.to_string())?;
            journal.observed = fs::fstat(writer).map_err(|error| error.to_string())?;
            journal.recovery_changed = true;
        }
        journal.bytes = complete_bytes as u64;
        Ok(journal)
    }

    fn validate_envelope(&self, envelope: &Envelope, bytes: &[u8]) -> Result<(), String> {
        if envelope.ordinal != self.events.len() as u64
            || envelope.previous_sha256 != self.previous
            || envelope.sha256 != envelope.digest()?
            || canonical::encode(envelope)? != bytes
        {
            return Err("campaign journal hash link or canonical record mismatch".into());
        }
        Ok(())
    }

    fn authenticate_prepared(&self) -> Result<(), String> {
        let reference = self.plan.prepared["prepared_artifact"]
            .as_str()
            .ok_or("exploration prepared artifact is missing")?;
        if !reference.starts_with("prepared/") {
            return Err("unsafe campaign prepared artifact reference".into());
        }
        let prepared = self.open_artifact(reference)?;
        prepared.authenticate_all()?;
        let bytes = prepared.payload(crate::replay::preflight::FILE, 16 << 20)?;
        let inputs: crate::replay::preflight::Inputs =
            serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
        if inputs.schema != "gomad.replay-inputs/v1"
            || canonical::encode(&inputs)? != bytes
            || inputs.helper_sha256 != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
            || inputs.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
            || self.plan.prepared["helper_sha256"] != inputs.helper_sha256
            || self.plan.prepared["host_sha256"] != inputs.host_sha256
            || self.plan.prepared["toolchain"] != inputs.toolchain_sha256
            || canonical::encode(&inputs.configuration)?
                != canonical::encode(&self.plan.prepared["configuration"])?
            || prepared.manifest().record.outcome.class != "prepared"
        {
            return Err(
                "exploration prepared inputs differ from its plan or implementation".into(),
            );
        }
        ledger::validate_plan_record(&self.plan.prepared, &self.root, &prepared.manifest().record)
    }

    fn bind_scope_module(&mut self) -> Result<(), String> {
        if !matches!(&self.state, RoundState::Choice(state) if state.config.site_scope.is_some()) {
            return Ok(());
        }
        let reference = self.plan.prepared["prepared_artifact"]
            .as_str()
            .ok_or("exploration prepared artifact is missing")?;
        let opened = self.open_artifact(reference)?;
        let entry = opened
            .manifest()
            .files
            .get("target/module.wasm")
            .ok_or("exploration prepared module is missing")?;
        let module = opened.payload("target/module.wasm", entry.bytes)?;
        if let RoundState::Choice(state) = &mut self.state {
            state.bind_scope_module(module.into())?;
        }
        Ok(())
    }

    fn consume(&mut self, bytes: &[u8]) -> Result<(), String> {
        let envelope: Envelope =
            serde_json::from_slice(bytes).map_err(|error| error.to_string())?;
        self.validate_envelope(&envelope, bytes)?;
        self.authenticate_event(&envelope.event)?;
        self.state.apply(&envelope.event, &self.plan)?;
        self.previous = envelope.sha256;
        self.events.push(envelope.event);
        Ok(())
    }

    pub(crate) fn search_context(
        &self,
        base_trace: Option<&[u8]>,
    ) -> Result<Option<crate::evidence::search_context::Context>, String> {
        let RoundState::Choice(state) = &self.state else {
            return Ok(None);
        };
        let Some(scope) = &state.config.site_scope else {
            return Ok(None);
        };
        let mut resolution = state
            .site_scope_resolution
            .clone()
            .ok_or("scoped round resolution missing")?;
        if state.committed_rounds == 0 {
            let reference = self.plan.prepared["prepared_artifact"]
                .as_str()
                .ok_or("prepared artifact missing")?;
            let prepared = self.open_artifact(reference)?;
            let entry = prepared
                .manifest()
                .files
                .get("target/module.wasm")
                .ok_or("prepared module missing")?;
            let module = prepared.payload("target/module.wasm", entry.bytes)?;
            if let Some(raw) = base_trace {
                resolution.observe(&module, raw, &state.config.execution, &scope.patterns)?;
            }
            if resolution.scope(&scope.patterns) != *scope {
                return Err("exploration plan identity or bounds changed: search context base resolution differs from plan scope".into());
            }
        }
        Ok(Some(crate::evidence::search_context::Context::new(
            scope.patterns.clone(),
            resolution,
            canonical::digest(&canonical::encode(&self.plan)?),
            state.committed_rounds,
            state.config.execution.clone(),
        )))
    }

    fn authenticate_event(&self, event: &RoundEvent) -> Result<(), String> {
        for execution in event.executions() {
            ledger::authenticate_execution(&self.plan.prepared, &self.root, execution)?;
        }
        if let RoundEvent::Choice {
            executions,
            segment,
        } = event
        {
            let base_trace = segment
                .results
                .first()
                .and_then(|result| result.source_trace.as_deref());
            let expected = self.search_context(base_trace)?;
            for execution in executions {
                if let Some(reference) = &execution.artifact {
                    let opened = self.open_artifact(reference)?;
                    if crate::evidence::search_context::read(&opened)? != expected {
                        return Err(
                            "search context differs from producing plan or pre-round snapshot"
                                .into(),
                        );
                    }
                }
            }
            for (execution, result) in executions.iter().zip(&segment.results) {
                if let Some(raw) = &result.source_trace {
                    let stream = execution
                        .record
                        .streams
                        .get("evidence/choice")
                        .ok_or("choice segment trace has no recorded stream")?;
                    if stream.bytes != raw.len() as u64 || stream.sha256 != canonical::digest(raw) {
                        return Err("choice segment trace differs from retained execution".into());
                    }
                    if let Some(reference) = execution.artifact.as_deref() {
                        let opened = self.open_artifact(reference)?;
                        if opened
                            .payload("evidence/choice", execution.record.limits.evidence_bytes)?
                            != *raw
                        {
                            return Err(
                                "choice segment trace differs from retained artifact".into()
                            );
                        }
                    }
                }
            }
        }
        Ok(())
    }

    /// # Errors
    /// Refuses invalid transitions, unauthenticated references, capacity overflow and failed writes.
    pub fn append(&mut self, event: RoundEvent) -> Result<(), String> {
        if self.writer.is_none() || self.lock.is_none() || self.poisoned {
            return Err("campaign journal is read-only or its writer is poisoned".into());
        }
        #[cfg(feature = "determinism-fixture")]
        let interrupt_after = std::env::var_os("GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER")
            .map(|raw| {
                let text = raw
                    .into_string()
                    .map_err(|_| "invalid campaign interruption ordinal")?;
                let ordinal = text
                    .parse::<u64>()
                    .map_err(|_| "invalid campaign interruption ordinal")?;
                if ordinal.to_string() != text {
                    return Err("invalid campaign interruption ordinal");
                }
                Ok(ordinal)
            })
            .transpose()?;
        verify_named(
            &self.root,
            ".lock",
            self.lock.as_ref().ok_or("campaign lock missing")?,
        )?;
        let mut next = self.state.clone();
        next.apply(&event, &self.plan)?;
        self.authenticate_event(&event)?;
        let mut envelope = Envelope {
            ordinal: self.events.len() as u64,
            previous_sha256: self.previous.clone(),
            event,
            sha256: String::new(),
        };
        envelope.sha256 = envelope.digest()?;
        let mut bytes = canonical::encode(&envelope)?;
        bytes.push(b'\n');
        let total = self
            .bytes
            .checked_add(bytes.len() as u64)
            .ok_or("campaign journal byte overflow")?;
        if total > self.max_bytes {
            return Err("campaign journal byte bound exceeded".into());
        }
        let writer = self.writer.as_mut().ok_or("campaign writer missing")?;
        verify_named(&self.root, "journal.jsonl", writer)?;
        if !open::same(
            &self.observed,
            &fs::fstat(&*writer).map_err(|error| error.to_string())?,
        ) {
            self.poisoned = true;
            return Err("campaign journal changed outside its writer".into());
        }
        self.poisoned = true;
        writer
            .write_all(&bytes)
            .and_then(|()| writer.sync_all())
            .map_err(|error| error.to_string())?;
        verify_named(&self.root, "journal.jsonl", writer)?;
        self.observed = fs::fstat(&*writer).map_err(|error| error.to_string())?;
        self.state = next;
        self.previous = envelope.sha256;
        self.events.push(envelope.event);
        self.bytes = total;
        self.poisoned = false;
        #[cfg(feature = "determinism-fixture")]
        if interrupt_after == Some(self.events.len() as u64 - 1) {
            std::process::exit(86);
        }
        Ok(())
    }

    /// # Errors
    /// Refuses unsafe references, substituted entries and unauthenticated artifact manifests.
    pub fn open_artifact(&self, reference: &str) -> Result<Opened, String> {
        ledger::open_artifact(&self.root, reference)
    }

    #[must_use]
    pub fn recovery_changed(&self) -> bool {
        self.recovery_changed
    }

    #[must_use]
    pub fn plan(&self) -> &RoundPlan {
        &self.plan
    }
    #[must_use]
    pub fn events(&self) -> &[RoundEvent] {
        &self.events
    }
    #[must_use]
    pub(crate) fn state(&self) -> &RoundState {
        &self.state
    }

    pub(crate) fn summary(&self) -> Result<Value, String> {
        self.state.summary()
    }

    #[must_use]
    pub fn path(&self) -> &Path {
        &self.path
    }
}

fn create_file(root: &File, name: &str, bytes: &[u8]) -> Result<File, String> {
    let mut file = File::from(
        fs::openat(
            root,
            name,
            OFlags::RDWR
                | OFlags::APPEND
                | OFlags::CREATE
                | OFlags::EXCL
                | OFlags::CLOEXEC
                | OFlags::NOFOLLOW,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())?;
    verify_named(root, name, &file)?;
    Ok(file)
}

fn writable_file(root: &File, name: &str) -> Result<File, String> {
    let checked = open::child(root, name, false, false)?;
    let file = File::from(
        fs::openat(
            root,
            name,
            OFlags::RDWR | OFlags::APPEND | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK,
            Mode::empty(),
        )
        .map_err(|error| error.to_string())?,
    );
    if !open::same(
        &fs::fstat(&checked).map_err(|error| error.to_string())?,
        &fs::fstat(&file).map_err(|error| error.to_string())?,
    ) {
        return Err("campaign journal substituted while opening writer".into());
    }
    verify_named(root, name, &file)?;
    Ok(file)
}

fn verify_named(root: &File, name: &str, file: &File) -> Result<(), String> {
    let checked = open::child(root, name, false, false)?;
    let stat = fs::fstat(file).map_err(|error| error.to_string())?;
    let named =
        fs::statat(root, name, AtFlags::SYMLINK_NOFOLLOW).map_err(|error| error.to_string())?;
    if !open::same(&stat, &named)
        || !open::same(
            &stat,
            &fs::fstat(&checked).map_err(|error| error.to_string())?,
        )
    {
        return Err("campaign journal entry substituted".into());
    }
    Ok(())
}

#[cfg(test)]
#[path = "round_tests.rs"]
mod tests;
