use crate::{
    campaign::{execution, ledger::Job, prepare::Prepared},
    evidence::{
        canonical,
        inspection::Details,
        record::Record,
        store::{Opened, Publication},
    },
    replay::{
        divergence,
        preflight::{self, Inputs},
    },
};
use gomad_wasmhost::linked;
use std::path::Path;

pub(crate) const ORIGINAL: &str = "minimization/original-record.json";

pub(crate) struct Parent {
    pub opened: Opened,
    prepared: Prepared,
    pub original: Record,
    pub trace: Vec<u8>,
    pub identity: linked::ChoiceIdentity,
}

impl Parent {
    pub fn from_replay(replay: preflight::Prepared) -> Result<Self, String> {
        if !replay.exact
            || !replay.available
            || matches!(&replay.execution, preflight::Execution::Cluster(_))
        {
            return Err(
                "minimization requires a complete standalone cooperative failure tape".into(),
            );
        }
        let opened = replay.opened;
        let record = &opened.manifest().record;
        if record.outcome.class != "target_failure" {
            return Err("minimization requires a replayable target failure".into());
        }
        let details = Details::read(&opened)?.ok_or("minimization choice evidence is missing")?;
        if details.simulation.is_some() {
            return Err("minimization requires a standalone backend choice failure without simulation evidence".into());
        }
        let choices = details
            .choices
            .as_ref()
            .ok_or("minimization choice evidence is missing")?;
        let trace = opened.payload(&choices.trace.file, choices.trace.limit)?;
        let identity = choices.identity.clone();
        let provenance = if opened
            .manifest()
            .files
            .contains_key("target/provenance.json")
        {
            opened.payload("target/provenance.json", 16 << 20)?
        } else {
            b"{}".to_vec()
        };
        let source: serde_json::Value =
            serde_json::from_slice(&provenance).map_err(|error| error.to_string())?;
        let configuration = if let Some(value) = source.get("configuration") {
            serde_json::from_value(value.clone()).map_err(|error| error.to_string())?
        } else {
            let inputs: Inputs =
                serde_json::from_slice(&opened.payload(preflight::FILE, 16 << 20)?)
                    .map_err(|error| error.to_string())?;
            inputs.configuration
        };
        let soft_assertion_report_only = source["inputs"]["soft_assertion_report_only"]
            .as_bool()
            .unwrap_or(false);
        let network = source
            .get("network")
            .map(|value| serde_json::from_value(value.clone()))
            .transpose()
            .map_err(|error| error.to_string())?
            .unwrap_or_else(|| configuration.network.clone());
        let prepared = Prepared {
            auxiliary: std::collections::BTreeMap::new(),
            compile_cache_root: None,
            module: opened.payload("target/module.wasm", 512 << 20)?,
            network,
            configuration,
            toolchain: record.identity.toolchain.clone(),
            choice_identity: Some(identity.clone()),
            scenario: source
                .get("scenario")
                .map(|value| serde_json::from_value(value.clone()))
                .transpose()
                .map_err(|error| error.to_string())?
                .flatten(),
            provenance,
            soft_assertion_report_only,
        };
        let original = behavioral_record(&opened)?;
        if prepared.scenario.is_some() && details.simulation.is_none() {
            return Err("cluster minimization requires retained simulation evidence".into());
        }
        if original.identity != record.identity
            || original.seed != record.seed
            || original.outcome != record.outcome
        {
            return Err("minimization original predicate identity mismatch".into());
        }
        if let Some(metadata) = &details.minimization
            && metadata.predicate.failure_signature != original.failure_signature()?
        {
            return Err("minimization retained predicate signature mismatch".into());
        }
        Ok(Self {
            opened,
            prepared,
            original,
            trace,
            identity,
        })
    }

    pub fn choice_request(&self, length: u64) -> Result<linked::ReplayRequest, String> {
        let plan = linked::project_choices(&self.trace, &self.identity)?.prefix(
            &linked::PrefixControl {
                cutoff: length,
                r#override: None,
            },
        )?;
        Ok(linked::ReplayRequest {
            schema: "gomad.replay-trace/v1".into(),
            execution_identity: self.identity.clone(),
            trace_sha256: canonical::digest(&self.trace),
            tape_sha256: crate::campaign::exploration::choice::hex(&plan.sha256),
            decisions: plan.decisions.len() as u64,
            trace_bytes: self.trace.len() as u64,
            trace_frames: (self.trace.len() as u64).div_ceil(65536),
            prefix_control: Some(linked::PrefixControl {
                cutoff: length,
                r#override: None,
            }),
        })
    }

    pub fn choice(&self, length: u64, ordinal: u64) -> Result<execution::Result, String> {
        let mut trial = execution::run_with_replay(
            &self.prepared,
            &Job {
                ordinal,
                seed: self.original.seed,
            },
            preflight::wall_timeout(
                &self.prepared.configuration.limits,
                self.prepared.configuration.wall_nanos,
            )?,
            &linked::Cancellation::default(),
            Some(&self.choice_request(length)?),
            Some(&self.trace),
        )?;
        retain_provenance_presence(&self.opened, &mut trial)?;
        Ok(trial)
    }

    pub fn matches(&self, candidate: &Record) -> Result<bool, String> {
        Ok(candidate.outcome == self.original.outcome
            && candidate.outcome.class == "target_failure"
            && candidate.failure_signature()? == self.original.failure_signature()?)
    }
}

fn retain_provenance_presence(
    parent: &Opened,
    trial: &mut execution::Result,
) -> Result<(), String> {
    const PROVENANCE: &str = "target/provenance.json";
    if parent.manifest().files.contains_key(PROVENANCE) {
        return Ok(());
    }
    let expected = crate::evidence::record::Stream {
        bytes: 2,
        sha256: canonical::digest(b"{}"),
        records: 1,
    };
    if trial
        .publication
        .payloads
        .get(PROVENANCE)
        .map(Vec::as_slice)
        != Some(b"{}")
        || trial.publication.record.streams.get(PROVENANCE) != Some(&expected)
        || trial.execution.record != trial.publication.record
        || trial.publication.payloads.get("record.json") != Some(&trial.publication.record.bytes()?)
    {
        return Err("minimizer generated provenance differs from absent parent fallback".into());
    }
    trial.publication.payloads.remove(PROVENANCE);
    trial.publication.record.streams.remove(PROVENANCE);
    trial
        .publication
        .payloads
        .insert("record.json".into(), trial.publication.record.bytes()?);
    trial.execution.record.clone_from(&trial.publication.record);
    Ok(())
}

pub(crate) fn behavioral_record(opened: &Opened) -> Result<Record, String> {
    for name in [ORIGINAL, "simulation/behavioral-record.json"] {
        if opened.manifest().files.contains_key(name) {
            if !opened.manifest().record.streams.contains_key(name) {
                return Err("minimization predicate record is not bound by its artifact".into());
            }
            return Record::decode(&opened.payload(name, 16 << 20)?);
        }
    }
    Ok(opened.manifest().record.clone())
}

pub(crate) fn publication(opened: &Opened) -> Result<Publication, String> {
    opened.authenticate_all()?;
    let payloads = opened
        .manifest()
        .files
        .iter()
        .map(|(name, entry)| Ok((name.clone(), opened.payload(name, entry.bytes)?)))
        .collect::<Result<_, String>>()?;
    Ok(Publication {
        record: opened.manifest().record.clone(),
        payloads,
    })
}

pub(crate) fn exact(path: &Path) -> Result<bool, String> {
    let prepared = preflight::open(path)?;
    if !prepared.exact || !prepared.available {
        return Ok(false);
    }
    let expected = prepared.opened.manifest().record.clone();
    let observed = preflight::execute(prepared.execution)?;
    Ok(divergence::first(&expected, &observed).is_none())
}

#[cfg(test)]
#[path = "predicate_tests.rs"]
mod tests;
