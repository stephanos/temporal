//! Stream the existing canonical terminal ledger while borrowing native evidence bytes.
use super::{
    build,
    evidence::{NativeExecution, NativeRun},
    terminal_collection::Record,
};
use serde::{Serialize, Serializer};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs::OpenOptions,
    io::{BufWriter, Write},
    os::unix::fs::OpenOptionsExt,
    path::Path,
};

#[derive(Serialize)]
#[serde(tag = "kind", rename_all = "kebab-case")]
pub(crate) enum Observation {
    Release {
        #[serde(serialize_with = "serialize_run")]
        observation: Box<NativeRun>,
        process_exit: Value,
    },
    AdmissionRefusal {
        observation: Value,
    },
}

#[derive(Serialize)]
struct RootView<'a> {
    handle: &'a Option<Value>,
    outcome: &'a Value,
    streams: &'a BTreeMap<String, Vec<u8>>,
}
impl<'a> From<&'a NativeExecution> for RootView<'a> {
    fn from(root: &'a NativeExecution) -> Self {
        Self {
            handle: &root.handle,
            outcome: &root.outcome,
            streams: &root.streams,
        }
    }
}
#[derive(Serialize)]
struct RunView<'a> {
    cutoff: &'a Option<Value>,
    executed_steps: &'a Option<Value>,
    incarnations: Vec<RootView<'a>>,
    modules: &'a [Value],
    profile: &'a Value,
    report: &'a Value,
    request: &'a Value,
    root: RootView<'a>,
}
impl<'a> From<&'a NativeRun> for RunView<'a> {
    fn from(run: &'a NativeRun) -> Self {
        Self {
            cutoff: &run.cutoff,
            executed_steps: &run.executed_steps,
            incarnations: run.incarnations.iter().map(RootView::from).collect(),
            modules: &run.modules,
            profile: &run.profile,
            report: &run.report,
            request: &run.request,
            root: RootView::from(&run.root),
        }
    }
}
fn serialize_run<S: Serializer>(run: &NativeRun, serializer: S) -> Result<S::Ok, S::Error> {
    RunView::from(run).serialize(serializer)
}

#[derive(Serialize)]
struct Document<'a> {
    accepted: bool,
    admission_refusals: Vec<&'a Value>,
    attempts: &'a [Record<Observation>],
    bound_nanos: u64,
    boundary: &'static str,
    diagnostic_control_admission: Option<Value>,
    failure_reasons: Option<&'a str>,
    helper_admission: Value,
    module_provenance: &'a Path,
    observations: Vec<RunView<'a>>,
    original_control_mapping: Value,
    process_exit_observations: Vec<&'a Value>,
    schema: &'static str,
}

pub(crate) fn publish(
    path: &Path,
    records: &[Record<Observation>],
    failure: Option<&str>,
    admission: &build::Admission,
    diagnostic: Option<&build::Admission>,
    provenance: &Path,
) -> Result<(), String> {
    let mut observations = Vec::new();
    let mut process_exit_observations = Vec::new();
    let mut admission_refusals = Vec::new();
    for record in records {
        match record.observation.as_ref() {
            Some(Observation::Release {
                observation,
                process_exit,
            }) => {
                observations.push(RunView::from(observation.as_ref()));
                process_exit_observations.push(process_exit);
            }
            Some(Observation::AdmissionRefusal { observation }) => {
                admission_refusals.push(observation);
            }
            None => {}
        }
    }
    let document = Document {
        accepted: failure.is_none(),
        admission_refusals,
        attempts: records,
        bound_nanos: 1_000_000_000,
        boundary: "actual classified native Outcome through completed linked Store/engine/watchdog cleanup; child supervisor wait retained separately",
        diagnostic_control_admission: diagnostic
            .map(serde_json::to_value)
            .transpose()
            .map_err(|e| e.to_string())?,
        failure_reasons: failure,
        helper_admission: serde_json::to_value(admission).map_err(|e| e.to_string())?,
        module_provenance: provenance,
        observations,
        original_control_mapping: json!({"success":"six admitted native release observations",
            "post-load-authorization-mismatch":"separate native diagnostic module-digest and loaded-module-descriptor admission controls; historical helper wire applicability remains separate"}),
        process_exit_observations,
        schema: "gomad.native-terminal-exit/v2",
    };
    let file = OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| e.to_string())?;
    let mut writer = BufWriter::with_capacity(64 << 10, file);
    serde_json::to_writer(&mut writer, &document).map_err(|e| e.to_string())?;
    writer.flush().map_err(|e| e.to_string())?;
    writer.get_ref().sync_all().map_err(|e| e.to_string())
}

#[cfg(test)]
#[path = "terminal_publication_tests.rs"]
pub(crate) mod tests;
