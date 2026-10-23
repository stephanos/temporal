use super::seeds::SeedSelection;
use crate::evidence::{
    canonical,
    record::Record,
    store::{Opened, open},
};
use rustix::fs::{self, AtFlags, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::File,
    io::Write,
    path::{Path, PathBuf},
};

const FORMAT: &str = "gomad.campaign-storage/v2";
const MAX_PLAN_BYTES: u64 = 16 << 20;

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Job {
    pub ordinal: u64,
    pub seed: u64,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Execution {
    pub ordinal: u64,
    pub seed: u64,
    pub record: Record,
    pub artifact: Option<String>,
    pub stored_bytes: u64,
    pub semantic_probes: Vec<String>,
    pub choice_features: Vec<String>,
    pub divergence: Option<gomad_wasmhost::linked::Divergence>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub soft_assertions: Option<Value>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scenario_steps: Option<Vec<gomad_wasmhost::linked_cluster::Step>>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "snake_case", deny_unknown_fields)]
#[expect(
    clippy::large_enum_variant,
    reason = "The receipt owns one canonical execution observation"
)]
pub enum Event {
    Admit {
        jobs: Vec<Job>,
    },
    Receipt {
        execution: Execution,
    },
    Outcome {
        executions: Vec<Execution>,
        admitted: Vec<Job>,
        stop_reason: Option<String>,
    },
    Complete {
        stop_reason: String,
    },
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
    event: Event,
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

#[derive(Clone)]
struct State {
    selection: SeedSelection,
    parallel: u64,
    shard_count: u64,
    next: Option<u64>,
    missing: Vec<OrdinalRange>,
    policy: String,
    launcher: bool,
    budget: u64,
    pending: BTreeMap<u64, u64>,
    receipts: BTreeMap<u64, Execution>,
    failures: BTreeSet<String>,
    stopped: Option<String>,
    complete: bool,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct OrdinalRange {
    start: u64,
    end: u64,
}

impl State {
    fn new(plan: &Value) -> Result<Self, String> {
        let selection = SeedSelection::parse(
            plan["selection"]
                .as_str()
                .ok_or("campaign selection missing")?,
        )?;
        if plan["selection_count"].as_u64() != Some(selection.count()) {
            return Err("campaign selection count mismatch".into());
        }
        let parallel = plan["parallel"]
            .as_u64()
            .filter(|value| *value > 0)
            .ok_or("campaign parallel bound missing")?;
        let (index, count) = if plan["shard"].is_null() {
            (0, 1)
        } else {
            let index = plan["shard"]["index"]
                .as_u64()
                .ok_or("invalid campaign shard")?;
            let count = plan["shard"]["count"]
                .as_u64()
                .ok_or("invalid campaign shard")?;
            if count == 0 || index >= count {
                return Err("invalid campaign shard".into());
            }
            (index, count)
        };
        let policy = plan["on_failure"]
            .as_str()
            .ok_or("campaign failure policy missing")?;
        let budget = plan["failure_budget"]
            .as_u64()
            .ok_or("campaign failure budget missing")?;
        if !matches!(policy, "first" | "budget" | "all") || policy == "budget" && budget == 0 {
            return Err("invalid campaign failure policy".into());
        }
        if plan["launcher"] == true
            && (parallel != 1 || policy != "first" || plan["origin"]["origin"] != "intercepted-go")
        {
            return Err("invalid intercepted launcher campaign policy".into());
        }
        let missing: Vec<OrdinalRange> = if plan["schema"] == "gomad.merged-seed-campaign-plan/v2" {
            if !plan["shard"].is_null()
                || policy != "all"
                || !plan["plan_sha256"]
                    .as_str()
                    .is_some_and(canonical::valid_digest)
                || plan["merge_sources"].as_array().is_none_or(Vec::is_empty)
            {
                return Err("invalid merged campaign plan identity".into());
            }
            serde_json::from_value(plan["merge_missing"].clone())
                .map_err(|error| error.to_string())?
        } else {
            if !plan["merge_missing"].is_null() || !plan["merge_sources"].is_null() {
                return Err("sparse ordinals require a merged campaign plan".into());
            }
            Vec::new()
        };
        if missing
            .iter()
            .any(|range| range.start > range.end || range.end >= selection.count())
            || missing.windows(2).any(|ranges| {
                ranges[0]
                    .end
                    .checked_add(1)
                    .is_none_or(|end| end >= ranges[1].start)
            })
        {
            return Err("merged campaign missing ordinal ranges are not canonical".into());
        }
        let mut state = Self {
            next: (index < selection.count()).then_some(index),
            missing,
            selection,
            parallel,
            shard_count: count,
            policy: policy.into(),
            launcher: plan["launcher"].as_bool().unwrap_or(false),
            budget,
            pending: BTreeMap::new(),
            receipts: BTreeMap::new(),
            failures: BTreeSet::new(),
            stopped: None,
            complete: false,
        };
        state.next = state.next.and_then(|ordinal| state.covered(ordinal));
        Ok(state)
    }

    fn covered(&self, ordinal: u64) -> Option<u64> {
        let next = match self
            .missing
            .iter()
            .find(|range| range.start <= ordinal && ordinal <= range.end)
        {
            Some(range) => range.end.checked_add(1)?,
            None => ordinal,
        };
        (next < self.selection.count()).then_some(next)
    }

    fn pending_stop(&self) -> Result<Option<String>, String> {
        let Some((&ordinal, _)) = self.pending.first_key_value() else {
            return Ok(None);
        };
        let Some(execution) = self.receipts.get(&ordinal) else {
            return Ok(None);
        };
        if self.launcher && execution.record.outcome.class != "success" {
            return Ok(Some("first_failure".into()));
        }
        if !is_failure(execution) {
            return Ok(None);
        }
        let signature = execution.record.failure_signature()?;
        let distinct = self.failures.len() as u64 + u64::from(!self.failures.contains(&signature));
        Ok(match self.policy.as_str() {
            "first" => Some("first_failure".into()),
            "budget" if distinct >= self.budget => Some("failure_budget".into()),
            _ => None,
        })
    }

    fn admit(&mut self, jobs: &[Job]) -> Result<(), String> {
        if self.stopped.is_some() || self.pending_stop()?.is_some() {
            return Err("stopped campaign admits no ordinal".into());
        }
        if self.pending.len() as u64 + jobs.len() as u64 > self.parallel {
            return Err("campaign admissions exceed parallel bound".into());
        }
        for job in jobs {
            if self.next != Some(job.ordinal)
                || self.selection.seed_at(job.ordinal) != Some(job.seed)
            {
                return Err("campaign admission seed, order or shard mismatch".into());
            }
            self.pending.insert(job.ordinal, job.seed);
            self.next = job
                .ordinal
                .checked_add(self.shard_count)
                .and_then(|ordinal| self.covered(ordinal));
        }
        Ok(())
    }

    fn apply(&mut self, event: &Event) -> Result<(), String> {
        if self.complete {
            return Err("campaign event follows completion".into());
        }
        match event {
            Event::Admit { jobs } => {
                if jobs.is_empty() {
                    return Err("campaign admission is empty".into());
                }
                self.admit(jobs)?;
            }
            Event::Receipt { execution } => {
                validate_execution(execution)?;
                if self.pending.get(&execution.ordinal) != Some(&execution.seed)
                    || self.receipts.contains_key(&execution.ordinal)
                {
                    return Err("campaign receipt has no unique admission".into());
                }
                if execution.record.outcome.termination == "cancelled"
                    && !self.launcher
                    && self.pending_stop()?.as_deref() != Some("first_failure")
                {
                    return Err("campaign cancellation has no first-failure stop".into());
                }
                self.receipts.insert(execution.ordinal, execution.clone());
            }
            Event::Outcome {
                executions,
                admitted,
                stop_reason,
            } => {
                let pending_stop = self.pending_stop()?;
                if stop_reason != &pending_stop {
                    return Err("campaign outcome does not match pending policy stop".into());
                }
                let ordered: Vec<_> = self.pending.keys().copied().collect();
                if executions.is_empty()
                    || stop_reason.is_none() && executions.len() != 1
                    || stop_reason.is_some()
                        && (executions.len() != ordered.len() || !admitted.is_empty())
                {
                    return Err("campaign outcome has an incomplete ordered stop group".into());
                }
                for (index, execution) in executions.iter().enumerate() {
                    if ordered.get(index) != Some(&execution.ordinal) {
                        return Err("campaign outcomes are not in admission order".into());
                    }
                    let receipt = self
                        .receipts
                        .get(&execution.ordinal)
                        .ok_or("campaign outcome has no durable receipt")?;
                    let mut observation = execution.clone();
                    if observation.record.outcome.class == "success" || is_failure(&observation) {
                        observation.artifact.clone_from(&receipt.artifact);
                        observation.stored_bytes = receipt.stored_bytes;
                    }
                    if receipt != &observation {
                        return Err("campaign outcome differs from its durable receipt".into());
                    }
                    validate_execution(execution)?;
                }
                for execution in executions {
                    self.pending.remove(&execution.ordinal);
                    self.receipts.remove(&execution.ordinal);
                    if is_failure(execution) {
                        self.failures.insert(execution.record.failure_signature()?);
                    }
                }
                self.stopped.clone_from(stop_reason);
                if !admitted.is_empty() {
                    self.admit(admitted)?;
                }
            }
            Event::Complete { stop_reason } => {
                if !self.pending.is_empty()
                    || !self.receipts.is_empty()
                    || match &self.stopped {
                        Some(stopped) => stopped != stop_reason,
                        None => stop_reason != "seeds_exhausted" || self.next.is_some(),
                    }
                {
                    return Err(
                        "campaign completion has outstanding work or an invalid stop".into(),
                    );
                }
                self.complete = true;
            }
        }
        Ok(())
    }
}

fn is_failure(execution: &Execution) -> bool {
    matches!(
        execution.record.outcome.class.as_str(),
        "target_failure" | "watchdog"
    )
}

pub(crate) fn validate_execution(execution: &Execution) -> Result<(), String> {
    execution.record.validate()?;
    if let Some(report) = &execution.soft_assertions {
        let typed: super::soft_assertions::Report =
            serde_json::from_value(report.clone()).map_err(|error| error.to_string())?;
        let bytes = canonical::encode(&typed)?;
        let stream = execution
            .record
            .streams
            .get("evidence/soft-assertions.json")
            .ok_or("committed soft assertion observation has no bound stream")?;
        if serde_json::to_value(typed).map_err(|error| error.to_string())? != *report
            || stream.sha256 != canonical::digest(&bytes)
            || stream.bytes != bytes.len() as u64
        {
            return Err("committed soft assertion observation identity mismatch".into());
        }
    }
    if execution
        .scenario_steps
        .as_ref()
        .is_some_and(|steps| steps.len() > 65_536)
    {
        return Err("campaign scenario execution steps exceed their bound".into());
    }
    if execution.seed != execution.record.seed
        || execution.artifact.is_none() && execution.stored_bytes != 0
        || execution.artifact.is_some() && execution.stored_bytes == 0
    {
        return Err("campaign execution record identity mismatch".into());
    }
    for names in [&execution.semantic_probes, &execution.choice_features] {
        if names.iter().any(String::is_empty) || names.windows(2).any(|pair| pair[0] >= pair[1]) {
            return Err("campaign execution features are not canonical".into());
        }
    }
    let outcome = &execution.record.outcome;
    if let Some(divergence) = &execution.divergence {
        if divergence.reason_name() == "unknown"
            || outcome.class != "target_failure"
            || !matches!(outcome.termination.as_str(), "diverged" | "divergence")
        {
            return Err("campaign divergence does not match its execution outcome".into());
        }
    } else if matches!(outcome.termination.as_str(), "diverged" | "divergence") {
        return Err("campaign divergence outcome has no typed evidence".into());
    }
    match outcome.class.as_str() {
        "success" if outcome.termination == "exit" && outcome.exit_code == Some(0) => {}
        "target_failure"
            if execution.artifact.is_some()
                && (outcome.termination == "trap"
                    || outcome.termination == "exit"
                        && outcome.exit_code.is_some_and(|code| code != 0)
                    || outcome.termination == "exit"
                        && outcome.exit_code == Some(0)
                        && execution
                            .record
                            .streams
                            .contains_key("evidence/soft-assertions.json")
                    || execution.divergence.is_some()
                    || outcome.termination == "instrumentation_divergence"
                        && execution
                            .record
                            .streams
                            .contains_key("simulation/record.json")) => {}
        "watchdog" if execution.artifact.is_some() && outcome.termination == "watchdog_timeout" => {
        }
        "runner_failure"
            if execution.artifact.is_some()
                && execution.record.streams.contains_key(super::origin::FILE)
                && outcome.exit_code.is_none()
                && !matches!(
                    outcome.termination.as_str(),
                    "exit" | "trap" | "divergence" | "diverged"
                ) => {}
        "runner_failure"
            if outcome.termination == "cancelled"
                && outcome.reason == "runner_cancelled"
                && execution.artifact.is_none() => {}
        _ => return Err("invalid campaign terminal outcome".into()),
    }
    Ok(())
}

pub struct Journal {
    root: File,
    path: PathBuf,
    lock: Option<File>,
    writer: Option<File>,
    plan: Value,
    state: State,
    events: Vec<Event>,
    previous: String,
    bytes: u64,
    max_bytes: u64,
    poisoned: bool,
    observed: fs::Stat,
    recovery_changed: bool,
}

impl Journal {
    /// # Errors
    /// Refuses existing campaign state, invalid plans, unsafe entries and durability failures.
    pub fn create(path: &Path, plan: &Value, max_bytes: u64) -> Result<Self, String> {
        let state = State::new(plan)?;
        if max_bytes == 0 || plan["journal_bytes"].as_u64() != Some(max_bytes) {
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
        super::authenticate_preparation_observations(&root, plan)?;
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
        Ok(Self {
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
        })
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
                "evidence"
                | "prepared"
                | crate::campaign::prepare::OBSERVATION_DIRECTORY
                | crate::campaign::execution::OBSERVATION_DIRECTORY => {
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
        let plan: Value = serde_json::from_slice(&plan_bytes).map_err(|error| error.to_string())?;
        if canonical::encode(&plan)? != plan_bytes {
            return Err("campaign plan is not canonical".into());
        }
        if plan["journal_bytes"].as_u64() != Some(storage.max_bytes) {
            return Err("campaign journal bound differs from its plan".into());
        }
        let state = State::new(&plan)?;
        super::authenticate_preparation_observations(&root, &plan)?;
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
        let complete_bytes = data
            .iter()
            .rposition(|byte| *byte == b'\n')
            .map_or(0, |index| index + 1);
        if complete_bytes != 0 {
            for line in data[..complete_bytes - 1].split(|byte| *byte == b'\n') {
                journal.consume(line)?;
            }
        }
        journal.authenticate_events()?;
        if let Some(reference) = journal.plan["prepared_artifact"].as_str() {
            if !reference.starts_with("prepared/") {
                return Err("unsafe campaign prepared artifact reference".into());
            }
            let prepared = journal.open_artifact(reference)?;
            prepared.authenticate_all()?;
            journal.validate_plan_record(&prepared.manifest().record)?;
        }
        if complete_bytes != data.len() {
            let tail = &data[complete_bytes..];
            match serde_json::from_slice::<Envelope>(tail) {
                Ok(envelope) => {
                    journal.validate_envelope(&envelope, tail)?;
                    journal.state.clone().apply(&envelope.event)?;
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
        if writable {
            journal.recovery_changed |= journal.reconcile_artifacts()?;
        }
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

    fn consume(&mut self, bytes: &[u8]) -> Result<(), String> {
        let envelope: Envelope =
            serde_json::from_slice(bytes).map_err(|error| error.to_string())?;
        self.validate_envelope(&envelope, bytes)?;
        self.state.apply(&envelope.event)?;
        self.previous = envelope.sha256;
        self.events.push(envelope.event);
        Ok(())
    }

    fn authenticate_execution(&self, execution: &Execution) -> Result<(), String> {
        authenticate_execution(&self.plan, &self.root, execution)
    }

    fn validate_plan_record(&self, record: &Record) -> Result<(), String> {
        validate_plan_record(&self.plan, &self.root, record)
    }

    fn authenticate_events(&self) -> Result<(), String> {
        for event in &self.events {
            match event {
                Event::Receipt { execution } => {
                    validate_execution_plan_record(&self.plan, &self.root, execution)?;
                    if let Some(reference) = &execution.artifact {
                        let parts = artifact_reference(reference)?;
                        if parts[0] != "evidence" {
                            return Err("campaign receipt references non-execution evidence".into());
                        }
                    }
                }
                Event::Outcome { .. } => self.authenticate_event(event)?,
                _ => {}
            }
        }
        for execution in self.state.receipts.values() {
            self.authenticate_execution(execution)?;
        }
        Ok(())
    }

    fn authenticate_event(&self, event: &Event) -> Result<(), String> {
        match event {
            Event::Receipt { execution } => self.authenticate_execution(execution)?,
            Event::Outcome { executions, .. } => {
                for execution in executions {
                    self.authenticate_execution(execution)?;
                }
            }
            _ => {}
        }
        Ok(())
    }

    /// # Errors
    /// Refuses invalid transitions, unauthenticated references, capacity overflow and failed writes.
    pub fn append(&mut self, event: Event) -> Result<(), String> {
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
        next.apply(&event)?;
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
        if matches!(self.events.last(), Some(Event::Outcome { .. })) {
            self.reconcile_artifacts()?;
        }
        Ok(())
    }

    /// # Errors
    /// Refuses unlocked mutation, invalid live evidence and unauthenticated retirement entries.
    pub fn reconcile_artifacts(&self) -> Result<bool, String> {
        if self.writer.is_none() || self.lock.is_none() || self.poisoned {
            return Err("campaign artifact reconciliation requires its healthy writer lock".into());
        }
        verify_named(
            &self.root,
            ".lock",
            self.lock.as_ref().ok_or("campaign lock missing")?,
        )?;
        let mut live = BTreeSet::new();
        let mut retain = |execution: &Execution| -> Result<(), String> {
            if let Some(reference) = &execution.artifact {
                let parts = artifact_reference(reference)?;
                if parts[0] != "evidence" {
                    return Err("campaign execution references non-execution evidence".into());
                }
                live.insert(parts[1].to_owned());
            }
            Ok(())
        };
        for execution in self.state.receipts.values() {
            retain(execution)?;
        }
        for event in &self.events {
            if let Event::Outcome { executions, .. } = event {
                for execution in executions {
                    retain(execution)?;
                }
            }
        }
        let store = match open::child(&self.root, "evidence", true, false) {
            Ok(store) => store,
            Err(error) if error == rustix::io::Errno::NOENT.to_string() && live.is_empty() => {
                return Ok(false);
            }
            Err(error) => return Err(error),
        };
        match open::read(&store, ".store.json", 4096, false) {
            Ok(layout) if layout == open::LAYOUT => {}
            Err(error) if error == rustix::io::Errno::NOENT.to_string() && live.is_empty() => {
                for name in open::entries(&store)? {
                    match name.as_str() {
                        ".targets" => {
                            open::child(&store, &name, true, false)?;
                        }
                        ".lock" => {
                            open::child(&store, &name, false, false)?;
                        }
                        _ => {
                            return Err(
                                "incomplete artifact store contains unexpected entries".into()
                            );
                        }
                    }
                }
                return Ok(false);
            }
            _ => return Err("unsupported campaign artifact store layout".into()),
        }
        let before = open::entries(&store)?;
        crate::evidence::store::Store::retire(&store, &live)?;
        Ok(before != open::entries(&store)?)
    }

    /// # Errors
    /// Refuses unsafe references, substituted entries and unauthenticated artifact manifests.
    pub fn open_artifact(&self, reference: &str) -> Result<Opened, String> {
        open_artifact(&self.root, reference)
    }

    #[must_use]
    pub fn recovery_changed(&self) -> bool {
        self.recovery_changed
    }

    #[must_use]
    pub fn plan(&self) -> &Value {
        &self.plan
    }
    #[must_use]
    pub fn events(&self) -> &[Event] {
        &self.events
    }
    #[must_use]
    pub fn root(&self) -> &File {
        &self.root
    }
    #[must_use]
    pub fn path(&self) -> &Path {
        &self.path
    }

    pub(super) fn fingerprints(&self) -> Result<Value, String> {
        let plan = canonical::encode(&self.plan)?;
        let storage = canonical::encode(&Storage {
            format: FORMAT.into(),
            max_bytes: self.max_bytes,
            plan_sha256: canonical::digest(&plan),
        })?;
        if open::read(&self.root, "plan.json", MAX_PLAN_BYTES, false)? != plan
            || open::read(&self.root, "storage.json", 4096, false)? != storage
        {
            return Err("campaign source plan or storage changed after validation".into());
        }
        let before = fs::statat(&self.root, "journal.jsonl", AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if !open::same(&self.observed, &before) {
            return Err("campaign source journal changed after validation".into());
        }
        let journal = open::read(&self.root, "journal.jsonl", self.max_bytes, false)?;
        let after = fs::statat(&self.root, "journal.jsonl", AtFlags::SYMLINK_NOFOLLOW)
            .map_err(|error| error.to_string())?;
        if !open::same(&self.observed, &after) {
            return Err("campaign source journal changed while fingerprinting".into());
        }
        let receipts = self.events.iter().filter_map(|event| match event {
            Event::Receipt { execution } => Some(canonical::encode(event)
                .map(|bytes| serde_json::json!({"ordinal":execution.ordinal,"sha256":canonical::digest(&bytes)}))),
            _ => None,
        }).collect::<Result<Vec<_>, _>>()?;
        Ok(serde_json::json!({"plan_sha256":canonical::digest(&plan),
            "storage_sha256":canonical::digest(&storage),"journal_sha256":canonical::digest(&journal),
            "receipts":receipts}))
    }
}

pub(crate) fn authenticate_execution(
    plan: &Value,
    root: &File,
    execution: &Execution,
) -> Result<(), String> {
    validate_execution_plan_record(plan, root, execution)?;
    if let Some(report) = &execution.soft_assertions
        && report["report_only"] != plan["soft_assertion_report_only"]
    {
        return Err("committed soft assertion policy differs from campaign".into());
    }
    if let Some(reference) = &execution.artifact {
        if !reference.starts_with("evidence/") {
            return Err("campaign execution references a non-execution artifact".into());
        }
        let opened = open_artifact(root, reference)?;
        let retained = &opened.manifest().record;
        let matches = if is_failure(execution) {
            retained.execution_identity == execution.record.execution_identity
                && retained.failure_signature()? == execution.record.failure_signature()?
        } else {
            retained == &execution.record
        };
        if !matches || opened.stored_bytes() != execution.stored_bytes {
            return Err("campaign artifact record reference mismatch".into());
        }
        opened.authenticate_all()?;
        if !plan["scenario"].is_null() {
            super::scenario::authenticate_assertions(plan, &opened)?;
        } else if execution
            .record
            .streams
            .contains_key("evidence/soft-assertions.json")
        {
            let bytes = opened.payload("evidence/soft-assertions.json", 16 << 20)?;
            let report: super::soft_assertions::Report =
                serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
            let output = |name: &str| -> Result<Vec<u8>, String> {
                if opened.manifest().record.streams.contains_key(name) {
                    opened.payload(name, execution.record.limits.output_bytes)
                } else {
                    Ok(Vec::new())
                }
            };
            let stdout = output("output/stdout")?;
            let stderr = output("output/stderr")?;
            if canonical::encode(&report)? != bytes
                || plan["soft_assertion_report_only"].as_bool() != Some(report.report_only)
                || super::soft_assertions::observe(
                    &stdout,
                    &stderr,
                    matches!(
                        execution.record.outcome.termination.as_str(),
                        "exit" | "trap"
                    ),
                    report.report_only,
                )
                .as_ref()
                    != Some(&report)
            {
                return Err("campaign soft assertion evidence mismatch".into());
            }
        }
    }
    Ok(())
}

pub(crate) fn validate_plan_record(
    plan: &Value,
    root: &File,
    record: &Record,
) -> Result<(), String> {
    validate_plan_record_with_steps(plan, root, record, None)
}

fn validate_execution_plan_record(
    plan: &Value,
    root: &File,
    execution: &Execution,
) -> Result<(), String> {
    if plan["scenario"].is_null() != execution.scenario_steps.is_none() {
        return Err("campaign execution step prefix does not match its scenario plan".into());
    }
    validate_plan_record_with_steps(
        plan,
        root,
        &execution.record,
        execution.scenario_steps.as_deref(),
    )
}

fn validate_plan_record_with_steps(
    plan: &Value,
    root: &File,
    record: &Record,
    steps: Option<&[gomad_wasmhost::linked_cluster::Step]>,
) -> Result<(), String> {
    if let Some(origin) = plan.get("origin") {
        super::origin::validate_record(origin, record)?;
    }
    let scenario = !plan["scenario"].is_null() && record.outcome.class != "prepared";
    let configuration = if scenario {
        let raw: Vec<u8> =
            serde_json::from_value(plan["scenario"].clone()).map_err(|error| error.to_string())?;
        let base = serde_json::from_value(plan["configuration"].clone())
            .map_err(|error| error.to_string())?;
        let identity = serde_json::from_value(plan["choice_identity"].clone())
            .map_err(|error| error.to_string())?;
        let network: Option<gomad_wasmhost::linked_cluster::NetworkModel> =
            serde_json::from_value(plan["network"].clone()).map_err(|error| error.to_string())?;
        let prepared = open_artifact(
            root,
            plan["prepared_artifact"]
                .as_str()
                .ok_or("campaign prepared artifact missing")?,
        )?;
        let module = prepared.payload("target/module.wasm", 512 << 20)?;
        let mut bound = super::scenario::bind(
            &raw,
            &base,
            &module,
            &identity,
            network.as_ref(),
            record.seed,
        )?;
        if plan["simulation_exploration"] == true {
            super::scenario::simulation_crash_bounds(&raw, &mut bound)?;
        }
        if let Some(steps) = steps {
            super::scenario::retain_execution_prefix(
                &mut bound.configuration,
                steps,
                plan["simulation_exploration"] == true,
            )?;
        }
        serde_json::to_value(&bound.configuration).map_err(|error| error.to_string())?
    } else {
        plan["configuration"].clone()
    };
    if plan["module_sha256"].as_str() != Some(record.identity.target.as_str())
        || plan["toolchain"].as_str() != Some(record.identity.toolchain.as_str())
        || record.identity.platform != crate::replay::preflight::platform()
        || record.identity.implementation != crate::replay::preflight::implementation()?
        || canonical::digest(&canonical::encode(&configuration)?) != record.identity.configuration
        || configuration["profile"].as_str() != Some(record.profile.as_str())
        || serde_json::to_value(&record.limits).map_err(|error| error.to_string())?
            != configuration["limits"]
        || if scenario {
            !record.arguments.is_empty() || !record.environment.is_empty()
        } else {
            serde_json::to_value(&record.arguments).map_err(|error| error.to_string())?
                != configuration["arguments"]
                || serde_json::to_value(&record.environment).map_err(|error| error.to_string())?
                    != configuration["environment"]
        }
    {
        return Err("campaign record differs from its prepared plan".into());
    }
    Ok(())
}

pub(crate) fn open_artifact(root: &File, reference: &str) -> Result<Opened, String> {
    let parts = artifact_reference(reference)?;
    let store = open::child(root, parts[0], true, false)?;
    if open::read(&store, ".store.json", 4096, false)? != open::LAYOUT {
        return Err("unsupported campaign artifact store layout".into());
    }
    let pool = open::child(&store, ".targets", true, false)?;
    let artifact = open::child(&store, parts[1], true, false)?;
    let opened = Opened::from_root_with_pool(artifact, Some(&pool))?;
    if !crate::evidence::store::matches_name(parts[1], opened.manifest())? {
        return Err("campaign artifact name does not match its record".into());
    }
    Ok(opened)
}

fn artifact_reference(reference: &str) -> Result<Vec<&str>, String> {
    let parts = open::components(reference)?;
    if parts.len() != 2
        || !matches!(parts[0], "evidence" | "prepared")
        || !crate::evidence::store::artifact_name(parts[1])
    {
        return Err("unsafe campaign artifact reference".into());
    }
    Ok(parts)
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
mod live_backend_tests {
    #[test]
    fn unretained_soft_assertion_roundtrip() {
        let report = serde_json::json!({"messages":["invalid fixture state"],"report_only":true,"inconclusive":false});
        let mut value = serde_json::json!({
            "ordinal":0,"seed":7,"artifact":null,"stored_bytes":0,
            "semantic_probes":[],"choice_features":[],"divergence":null,
            "soft_assertions":report,
            "record":{"schema":"gomad.execution-record/v2","execution_identity":"",
                "identity":{"target":"","toolchain":"","platform":"","implementation":"","configuration":""},
                "profile":"gomad.wasi-cooperative/v1","seed":7,"arguments":[],"environment":{},
                "limits":{"fuel":4_000_000_000_u64,"memory_bytes":268_435_456,"wall_millis":300_000,
                    "output_bytes":1_048_576,"evidence_bytes":16_777_216},
                "virtual_time_nanos":0,"outcome":{"class":"success","termination":"exit","exit_code":0,"reason":""},"streams":{}}
        });
        for name in [
            "target",
            "toolchain",
            "platform",
            "implementation",
            "configuration",
        ] {
            value["record"]["identity"][name] = "0".repeat(64).into();
        }
        let identity: crate::evidence::identity::Identity =
            serde_json::from_value(value["record"]["identity"].clone()).unwrap();
        value["record"]["execution_identity"] = identity.digest().unwrap().into();
        let bytes = crate::evidence::canonical::encode(&report).unwrap();
        value["record"]["streams"]["evidence/soft-assertions.json"] = serde_json::json!({
            "bytes":bytes.len(), "sha256":crate::evidence::canonical::digest(&bytes), "records":1});
        let decoded = serde_json::from_value::<super::Execution>(value.clone());
        assert!(
            decoded.is_ok(),
            "committed unretained observation was lost: {decoded:?}"
        );
        let mut decoded = decoded.unwrap();
        assert_eq!(serde_json::to_value(&decoded).unwrap(), value);
        super::validate_execution(&decoded).unwrap();
        decoded.soft_assertions.as_mut().unwrap()["messages"] = serde_json::json!(["changed"]);
        assert!(super::validate_execution(&decoded).is_err());
    }
}
