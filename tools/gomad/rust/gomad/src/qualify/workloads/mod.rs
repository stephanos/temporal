//! Existing mandatory workloads, their sealed boot inputs and output oracles.

use crate::{campaign::scenario::Bound, evidence::canonical};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::collections::BTreeMap;

mod boots;
pub(crate) mod native_crash;
mod oracles;
pub(crate) mod recovery;
mod retry;
pub(crate) mod sqlite;
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
    #[serde(rename = "temporal-committed-workflow-crash-reopen")]
    TemporalRecovery {
        forcing: Option<Box<recovery::Forcing>>,
    },
    #[serde(rename = "temporal-onebox-stopped-inspection")]
    TemporalInspection,
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

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct History {
    cluster_id: String,
    #[serde(rename = "history")]
    payload: String,
    event_ids: Vec<i64>,
    types: Vec<String>,
}

fn read_history(output: &[u8], prefix: &str) -> Result<History, String> {
    let value: History = super::spec::strict(line(output, prefix)?.as_bytes())?;
    if value.cluster_id.is_empty()
        || value.event_ids.is_empty()
        || value.event_ids.len() != value.types.len()
        || value.event_ids.iter().enumerate().any(|(index, id)| {
            i64::try_from(index)
                .ok()
                .and_then(|index| index.checked_add(1))
                != Some(*id)
        })
        || value.types.iter().any(String::is_empty)
        || super::cells::base64(&value.payload).is_none_or(|bytes| bytes.is_empty())
    {
        return Err("incomplete Temporal history observation".into());
    }
    Ok(value)
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
        if matches!(
            self,
            Self::SQLite { selection: Some(_) } | Self::TemporalRecovery { forcing: Some(_) }
        ) {
            return Err(
                "runtime crash controls require the registered post-preparation intent".into(),
            );
        }
        Ok(())
    }

    pub(crate) fn bind(&self, bound: &mut Bound) -> Result<(), String> {
        self.validate()?;
        match self {
            Self::SQLite { selection } => boots::sqlite(bound, selection.as_ref()),
            Self::TemporalReplication { .. } => boots::temporal_replication(bound),
            Self::TemporalRecovery { forcing: None } => boots::temporal_recovery(bound),
            Self::TemporalRecovery { forcing: Some(_) } => {
                Err("primary forcing requires the post-preparation runtime owner".into())
            }
            Self::TemporalInspection => boots::temporal_inspection(bound),
        }
    }

    pub(crate) fn observe(
        &self,
        streams: &BTreeMap<String, Vec<u8>>,
    ) -> Result<Observation, String> {
        let checks = match self {
            Self::SQLite { .. } => oracles::sqlite(streams)?,
            Self::TemporalReplication { canary } => oracles::replication(streams, *canary)?,
            Self::TemporalRecovery { forcing } => oracles::recovery(streams, forcing.as_deref())?,
            Self::TemporalInspection => oracles::inspection(streams)?,
        };
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
    let Some(workload) = retained_workload(result)? else {
        return Ok((Vec::new(), None));
    };
    let mut catalogues = Vec::new();
    let mut selection = match &workload {
        Workload::SQLite { selection } => selection.clone(),
        Workload::TemporalReplication { .. }
        | Workload::TemporalRecovery { .. }
        | Workload::TemporalInspection => None,
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
        let actual_workload = inputs.configuration.workload.as_ref();
        runtime_binding(result, &opened, actual_workload, &workload)?;
        let streams = retained_streams(&opened, &execution.record)?;
        if let Some(Workload::SQLite {
            selection: Some(actual),
        }) = actual_workload
        {
            if selection.as_ref().is_some_and(|prior| prior != actual) {
                return Err("SQLite runtime selection changed between repetitions".into());
            }
            selection = Some(actual.clone());
        }
        let observation = actual_workload
            .ok_or("retained workload missing")?
            .observe(&streams)?;
        let expected = canonical::encode(&observation)?;
        if opened.payload(&format!("evidence/{ORACLES}"), 1 << 20)? != expected
            || (execution.record.outcome.class == "success") != observation.passed()
        {
            return Err("mandatory workload oracle differs from authenticated outputs".into());
        }
        if matches!(&workload, Workload::SQLite { .. })
            && result.manifest.cell.as_deref() != Some("canary")
        {
            sqlite_catalogue(&opened, &mut catalogues, selection.as_ref())?;
        }
        if let Some(Workload::TemporalRecovery { forcing }) = actual_workload {
            temporal_catalogue(&opened, &mut catalogues, forcing.as_deref())?;
        }
    }
    Ok((catalogues, selection))
}

fn runtime_binding(
    result: &super::result::QualificationResult,
    opened: &crate::evidence::store::Opened,
    actual_workload: Option<&Workload>,
    workload: &Workload,
) -> Result<(), String> {
    let runtime = recovery::from_provenance(&canonical::encode(&result.provenance)?)?;
    if let Some(Workload::SQLite {
        selection: Some(selection),
    }) = actual_workload
    {
        if !matches!(&runtime,Some(recovery::Request::SQLiteSelection {selection:expected,..}) if expected==selection)
        {
            return Err(
                "SQLite selection differs from its authenticated post-preparation intent".into(),
            );
        }
        if result.manifest.id != "sqlite-durable-recovery" {
            return Err("SQLite runtime selection has a foreign registry identity".into());
        }
        sqlite::authenticate(opened, selection)?;
    } else if let Some(Workload::TemporalRecovery {
        forcing: Some(forcing),
    }) = actual_workload
    {
        if !runtime
            .as_ref()
            .is_some_and(|request| recovery::matches_request(request, forcing))
        {
            return Err("Temporal forcing differs from its authenticated runtime intent".into());
        }
        if result.manifest.id != "temporal-committed-workflow-crash-reopen" {
            return Err("Temporal forcing has a foreign registry identity".into());
        }
        recovery::authenticate(opened, forcing)?;
    } else if actual_workload != Some(workload) || runtime.is_some() {
        return Err("retained workload intent differs from frozen scenario".into());
    }
    Ok(())
}

fn sqlite_catalogue(
    opened: &crate::evidence::store::Opened,
    catalogues: &mut Vec<native_crash::Catalogue>,
    selection: Option<&native_crash::Selection>,
) -> Result<(), String> {
    opened.payload("evidence/cluster/sqlite/2/storage_precrash_image", 9 << 20)?;
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
    if let Some(selection) = selection {
        let catalogue = &catalogues[0];
        if catalogue.context != selection.context
            || !catalogue
                .enumeration
                .states
                .as_ref()
                .is_some_and(|states| states.contains(&selection.state))
        {
            return Err("selected SQLite state/context differs from actual catalogue".into());
        }
        let recovered = opened.payload("evidence/cluster/sqlite/3/storage_state", 8 << 20)?;
        native_crash::marker(&recovered, &selection.state)?;
    }
    Ok(())
}

fn temporal_catalogue(
    opened: &crate::evidence::store::Opened,
    catalogues: &mut Vec<native_crash::Catalogue>,
    forcing: Option<&recovery::Forcing>,
) -> Result<(), String> {
    opened.payload(
        "evidence/cluster/temporal/1/storage_precrash_image",
        513 << 20,
    )?;
    let state = opened.payload("evidence/cluster/temporal/1/storage_state", 256 << 20)?;
    let log = opened.payload("evidence/cluster/temporal/1/storage", 128 << 20)?;
    let catalogue = native_crash::temporal(&state, &log)?;
    if let Some(first) = catalogues.first() {
        if first != &catalogue {
            return Err("Temporal complete native catalogue differs across fresh repeats".into());
        }
    } else {
        catalogues.push(catalogue);
    }
    if let Some(forcing) = forcing {
        let catalogue = &catalogues[0];
        if catalogue.context != forcing.context
            || !catalogue
                .enumeration
                .states
                .as_ref()
                .is_some_and(|states| states.contains(&forcing.state))
        {
            return Err(
                "Temporal selected state or context differs from actual complete catalogue".into(),
            );
        }
    }
    Ok(())
}

fn retained_workload(
    result: &super::result::QualificationResult,
) -> Result<Option<Workload>, String> {
    let scenario = &result.provenance["scenario"];
    if scenario.is_null() {
        match recovery::from_provenance(&canonical::encode(&result.provenance)?)? {
            Some(recovery::Request::TemporalRetry)
                if result.manifest.id == "temporal-named-network-reset" =>
            {
                retry::retained(result)?;
            }
            Some(_) => {
                return Err(
                    "runtime workload requires its registered base scenario and identity".into(),
                );
            }
            None if result.manifest.id == "temporal-named-network-reset" => {
                return Err(
                    "named reset Result omitted its authenticated runtime workload intent".into(),
                );
            }
            None => (),
        }
        return Ok(None);
    }
    let raw: Vec<u8> =
        serde_json::from_value(scenario.clone()).map_err(|error| error.to_string())?;
    let scenario: Value = super::spec::strict(&raw)?;
    let workload: Option<Workload> =
        serde_json::from_value(scenario["workload"].clone()).map_err(|error| error.to_string())?;
    let Some(workload) = workload else {
        if matches!(
            result.manifest.id.as_str(),
            "sqlite-durable-recovery"
                | "temporal-committed-workflow-crash-reopen"
                | "independent-temporal-replication-durable-restart"
        ) {
            return Err("registered mandatory Result omitted its known workload".into());
        }
        return Ok(None);
    };
    let expected = match &workload {
        Workload::SQLite { .. } => Some("sqlite-durable-recovery"),
        Workload::TemporalReplication { .. } => {
            Some("independent-temporal-replication-durable-restart")
        }
        Workload::TemporalRecovery { .. } => Some("temporal-committed-workflow-crash-reopen"),
        Workload::TemporalInspection => None,
    };
    if expected.is_some_and(|expected| result.manifest.id != expected) {
        return Err("mandatory workload Result has another registry identity".into());
    }
    workload.validate()?;
    Ok(Some(workload))
}

fn retained_streams(
    opened: &crate::evidence::store::Opened,
    record: &crate::evidence::record::Record,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let mut streams = BTreeMap::new();
    for file in record
        .streams
        .keys()
        .filter(|file| file.starts_with("evidence/cluster/"))
    {
        let name = file
            .strip_prefix("evidence/")
            .ok_or("cluster evidence prefix absent")?;
        if [
            "/stdout",
            "/storage",
            "/choice",
            "/storage_state",
            "/storage_precrash_image",
            "/recovery_image_primary",
            "/recovery_image_input",
        ]
        .iter()
        .any(|suffix| name.ends_with(suffix))
        {
            streams.insert(
                name.to_owned(),
                opened.payload(
                    file,
                    record.limits.evidence_bytes.max(record.limits.output_bytes),
                )?,
            );
        }
    }
    Ok(streams)
}
