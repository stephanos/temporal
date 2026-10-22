//! Existing mandatory workloads, their sealed boot inputs and output oracles.

use crate::{campaign::scenario::Bound, evidence::canonical};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::collections::BTreeMap;

mod boots;
pub(crate) mod native_crash;
#[cfg(test)]
mod tests;

pub(crate) const ORACLES: &str = "workload-oracles.json";

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(tag = "kind", deny_unknown_fields)]
pub enum Workload {
    #[serde(rename = "sqlite-durable-recovery")]
    SQLite {
        selection: Option<native_crash::Selection>,
    },
    #[serde(rename = "independent-temporal-replication-durable-restart")]
    TemporalReplication { canary: bool },
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Observation {
    schema: String,
    checks: BTreeMap<String, bool>,
}

impl Observation {
    pub(crate) fn passed(&self) -> bool {
        !self.checks.is_empty() && self.checks.values().all(|value| *value)
    }
}

fn line<'a>(bytes: &'a [u8], prefix: &str) -> Result<&'a str, String> {
    let output = std::str::from_utf8(bytes).map_err(|_| "workload observation is not UTF-8")?;
    let mut matches = output.lines().filter_map(|line| line.strip_prefix(prefix));
    let value = matches
        .next()
        .filter(|line| !line.is_empty() && line.len() <= 1 << 20)
        .ok_or("missing or oversized workload observation")?;
    if matches.next().is_some() {
        return Err("ambiguous workload observation".into());
    }
    Ok(value)
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct History {
    cluster_id: String,
    #[serde(rename = "history")]
    payload: String,
    event_ids: Vec<i64>,
    types: Vec<String>,
}

fn oracle_bytes(lengths: impl IntoIterator<Item = usize>) -> Result<(), String> {
    let total = lengths
        .into_iter()
        .try_fold(0usize, usize::checked_add)
        .ok_or("history oracle byte overflow")?;
    if total > 1 << 20 {
        return Err("history oracle evidence exceeds original 1MiB bound".into());
    }
    Ok(())
}

fn history_operation(history: &str) -> Result<Vec<u8>, String> {
    canonical::encode(
        &json!([{"id":"committed","actor":"workflow","kind":"workflow-history","invocation":1,"completion":2,"output":history}]),
    )
}

impl Workload {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if let Self::SQLite {
            selection: Some(selection),
        } = self
        {
            selection.validate()?;
        }
        Ok(())
    }

    pub(crate) fn bind(&self, bound: &mut Bound) -> Result<(), String> {
        self.validate()?;
        match self {
            Self::SQLite { selection } => boots::sqlite(bound, selection.as_ref()),
            Self::TemporalReplication { .. } => boots::temporal_replication(bound),
        }
    }

    pub(crate) fn observe(
        &self,
        streams: &BTreeMap<String, Vec<u8>>,
    ) -> Result<Observation, String> {
        let output = |node: &str, incarnation: u64| {
            streams
                .get(&format!("cluster/{node}/{incarnation}/stdout"))
                .map(Vec::as_slice)
                .ok_or_else(|| "workload stdout missing".to_owned())
        };
        let mut checks = BTreeMap::new();
        match self {
            Self::SQLite { .. } => {
                let ack = line(output("sqlite", 2)?, "ACK ")?;
                let recovered = output("sqlite", 3)?;
                let integrity = line(recovered, "INTEGRITY ")?;
                let history = line(recovered, "HISTORY ")?;
                let rows = history
                    .split(',')
                    .map(|row| row.split_once(':').ok_or("invalid SQLite history"))
                    .collect::<Result<Vec<_>, _>>()?;
                if rows.iter().any(|(id, value)| {
                    id.parse::<u64>()
                        .ok()
                        .filter(|id| *id > 0)
                        .map(|id| id.to_string())
                        .as_deref()
                        != Some(*id)
                        || value.is_empty()
                        || value.contains([':', '\r', '\n'])
                }) {
                    return Err("invalid SQLite history".into());
                }
                checks.insert(
                    "acknowledged-durable-recovery.events".into(),
                    rows.iter().map(|(id, _)| *id).collect::<Vec<_>>() == ["1", "2"],
                );
                checks.insert(
                    "acknowledged-durable-recovery.integrity".into(),
                    integrity == "ok",
                );
                checks.insert(
                    "acknowledged-durable-recovery.acknowledged".into(),
                    ack == "2" && rows.iter().all(|(_, value)| *value == "committed"),
                );
            }
            Self::TemporalReplication { canary } => {
                let mut recovered = Vec::new();
                for role in ["active", "standby"] {
                    let read = |incarnation| -> Result<History, String> {
                        let value: History = super::spec::strict(
                            line(output(role, incarnation)?, "GOMAD_TEMPORAL_OBSERVATION ")?
                                .as_bytes(),
                        )?;
                        if value.cluster_id.is_empty()
                            || value.payload.is_empty()
                            || value.event_ids.is_empty()
                            || value.event_ids.len() != value.types.len()
                            || value.event_ids.iter().any(|id| *id <= 0)
                            || value.types.iter().any(String::is_empty)
                            || super::cells::base64(&value.payload)
                                .is_none_or(|bytes| bytes.is_empty())
                        {
                            return Err("incomplete Temporal history observation".into());
                        }
                        Ok(value)
                    };
                    let expected = read(1)?;
                    let actual = read(2)?;
                    oracle_bytes([
                        history_operation(&expected.payload)?.len(),
                        history_operation(&actual.payload)?.len(),
                    ])?;
                    oracle_bytes([expected.cluster_id.len(), actual.cluster_id.len()])?;
                    checks.insert(
                        format!("{role}-durable-recovery.history"),
                        expected.payload == actual.payload,
                    );
                    checks.insert(
                        format!("{role}-durable-recovery.events"),
                        !canary && actual.event_ids == [1, 2, 3, 4, 5],
                    );
                    checks.insert(
                        format!("{role}-durable-recovery.identity"),
                        expected.cluster_id == actual.cluster_id,
                    );
                    recovered.push(actual.payload);
                }
                oracle_bytes(recovered.iter().map(|history| {
                    super::cells::base64(history).map_or(usize::MAX, |bytes| bytes.len())
                }))?;
                checks.insert(
                    "independent-replication-convergence".into(),
                    recovered[0] == recovered[1],
                );
            }
        }
        Ok(Observation {
            schema: "gomad.rust-workload-oracles/v1".into(),
            checks,
        })
    }
}

pub(crate) fn project(workload: &Workload, observed: &mut linked::Execution) -> Result<(), String> {
    let observation = workload.observe(&observed.evidence)?;
    if !observation.passed()
        && observed.outcome.termination == "exit"
        && observed.outcome.exit_code == Some(0)
    {
        observed.outcome.message = "mandatory workload oracle failed".into();
    }
    observed
        .evidence
        .insert(ORACLES.into(), canonical::encode(&observation)?);
    Ok(())
}

pub(crate) fn retained(
    result: &super::result::QualificationResult,
) -> Result<
    (
        Vec<native_crash::Catalogue>,
        Option<native_crash::Selection>,
    ),
    String,
> {
    use crate::evidence::store::Opened;
    let scenario = &result.provenance["scenario"];
    if scenario.is_null() {
        return Ok((Vec::new(), None));
    }
    let raw: Vec<u8> =
        serde_json::from_value(scenario.clone()).map_err(|error| error.to_string())?;
    let scenario: Value = super::spec::strict(&raw)?;
    let workload: Option<Workload> =
        serde_json::from_value(scenario["workload"].clone()).map_err(|error| error.to_string())?;
    let Some(workload) = workload else {
        return Ok((Vec::new(), None));
    };
    workload.validate()?;
    let mut catalogues = Vec::new();
    let selection = match &workload {
        Workload::SQLite { selection } => selection.clone(),
        Workload::TemporalReplication { .. } => None,
    };
    for execution in &result.report.executions {
        let path = execution
            .artifact_path
            .as_ref()
            .ok_or("mandatory workload execution was not retained")?;
        let opened = Opened::open(std::path::Path::new(path))?;
        if opened.manifest().record != execution.record {
            return Err("mandatory workload record differs from its artifact".into());
        }
        let inputs: crate::replay::cluster::Inputs =
            super::spec::strict(&opened.payload(crate::replay::cluster::FILE, 64 << 20)?)?;
        if inputs.configuration.workload.as_ref() != Some(&workload) {
            return Err("retained workload intent differs from frozen scenario".into());
        }
        let mut streams = BTreeMap::new();
        for name in execution
            .record
            .streams
            .keys()
            .filter_map(|name| name.strip_prefix("evidence/"))
        {
            if name.ends_with("/stdout") {
                streams.insert(
                    name.to_owned(),
                    opened.payload(
                        &format!("evidence/{name}"),
                        execution.record.limits.output_bytes,
                    )?,
                );
            }
        }
        let observation = workload.observe(&streams)?;
        let expected = canonical::encode(&observation)?;
        if opened.payload(&format!("evidence/{ORACLES}"), 1 << 20)? != expected
            || (execution.record.outcome.class == "success") != observation.passed()
        {
            return Err("mandatory workload oracle differs from authenticated outputs".into());
        }
        if matches!(&workload, Workload::SQLite { .. })
            && result.manifest.cell.as_deref() != Some("canary")
        {
            let state = opened.payload("evidence/cluster/sqlite/2/storage_state", 8 << 20)?;
            let log = opened.payload("evidence/cluster/sqlite/2/storage", 32 << 20)?;
            let catalogue = native_crash::capture(&state, &log)?;
            if let Some(first) = catalogues.first() {
                if first != &catalogue {
                    return Err("SQLite native catalogue differs across fresh repeats".into());
                }
            } else {
                catalogues.push(catalogue);
            }
            if let Some(selection) = &selection {
                let catalogue = &catalogues[0];
                if catalogue.context != selection.context
                    || !catalogue
                        .enumeration
                        .states
                        .as_ref()
                        .is_some_and(|states| states.contains(&selection.state))
                {
                    return Err(
                        "selected SQLite state/context differs from actual catalogue".into(),
                    );
                }
                let recovered =
                    opened.payload("evidence/cluster/sqlite/3/storage_state", 8 << 20)?;
                native_crash::marker(&recovered, &selection.state)?;
            }
        }
    }
    Ok((catalogues, selection))
}
