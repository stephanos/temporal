use super::super::{build, host, inputs, module, workloads};
use crate::evidence::canonical;
use gomad_wasmhost::linked_cluster::{Handle, LifecycleObserver, LifecyclePhase};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::{
    collections::BTreeMap,
    os::unix::fs::DirBuilderExt,
    path::{Path, PathBuf},
};

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Spec {
    schema: String,
    repo: PathBuf,
    files: BTreeMap<PathBuf, String>,
    helper: PathBuf,
    helper_sha256: String,
    role: build::Role,
    receipt: Option<PathBuf>,
    memory_mode: String,
    replay: bool,
}
impl Spec {
    pub(super) fn prepared(
        prepared: &module::Prepared,
        proof: &inputs::CheckoutProof,
        helper: &workloads::Executable,
        memory_mode: &str,
        replay: bool,
    ) -> Result<Self, String> {
        let provenance = module::authenticate_prepared(prepared)?;
        let mut files = BTreeMap::new();
        for name in ["compiler", "sources"] {
            files.extend(
                serde_json::from_value::<BTreeMap<PathBuf, String>>(
                    provenance["inputs"][name].clone(),
                )
                .map_err(|error| error.to_string())?,
            );
        }
        for path in [
            &prepared.module_path,
            &prepared.provenance_path,
            &prepared.installation_path,
        ]
        .into_iter()
        .chain(prepared.scenario_path.iter())
        {
            files.insert(
                path.clone(),
                canonical::digest(&std::fs::read(path).map_err(|error| error.to_string())?),
            );
        }
        Ok(Self {
            schema: "gomad.cluster-phase-admission/v1".into(),
            repo: proof.repo().into(),
            files,
            helper: helper.path.clone(),
            helper_sha256: helper.sha256.clone(),
            role: helper.role.clone(),
            receipt: helper.receipt.clone(),
            memory_mode: memory_mode.into(),
            replay,
        })
    }
    pub(crate) fn read(path: &Path) -> Result<Self, String> {
        let raw = super::read(path, 16 << 20)?;
        let value = crate::qualify::spec::strict(&raw)?;
        let spec: Self = serde_json::from_value(value).map_err(|error| error.to_string())?;
        if spec.schema != "gomad.cluster-phase-admission/v1" || spec.files.is_empty() {
            return Err("cluster admission specification absent".into());
        }
        Ok(spec)
    }
    fn verify(&self) -> Result<(), String> {
        let proof = inputs::authenticate_checkout(&self.repo)?;
        module::verify_inputs(&self.files)?;
        workloads::Executable {
            path: self.helper.clone(),
            sha256: self.helper_sha256.clone(),
            role: self.role.clone(),
            receipt: self.receipt.clone(),
        }
        .verify(&proof)
    }
}

pub(crate) struct Observer {
    spec: Spec,
    root: PathBuf,
    environment: Vec<String>,
}
impl Observer {
    pub(crate) fn new(spec: Spec, root: &Path) -> Result<Self, String> {
        let root = root.join("phase-admission");
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .map_err(|error| error.to_string())?;
        Ok(Self {
            spec,
            root,
            environment: std::env::vars()
                .map(|(name, value)| format!("{name}={value}"))
                .collect(),
        })
    }
    fn observe(&self, phase: &LifecyclePhase, before: bool) -> Result<(), String> {
        let label = phase_label(phase, self.spec.replay)?;
        boundary(
            before,
            || self.spec.verify(),
            || {
                host::admit_preparation(
                    &self.root,
                    &label,
                    &self.spec.memory_mode,
                    &self.environment,
                )
            },
            || {
                super::write(
                    &self.root.join(format!(
                        "{label}-{}.json",
                        if before { "before" } else { "after" }
                    )),
                    &canonical::encode(
                        &json!({"schema":"gomad.cluster-phase-observation/v1","phase":label,"before":before,
                    "handles":phase.handles,"process_id":std::process::id(),"unix_nanos":std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map_err(|error|error.to_string())?.as_nanos(),"passed":true}),
                    )?,
                )
            },
        )
    }
}
fn phase_label(phase: &LifecyclePhase, replay: bool) -> Result<String, String> {
    let expected: Vec<_> = ["active", "standby"]
        .map(|node| Handle {
            node: node.into(),
            incarnation: phase.incarnation,
        })
        .into();
    if phase.handles != expected {
        return Err("cluster phase pair membership differs".into());
    }
    Ok(match (phase.incarnation, replay) {
        (1, false) => "cluster-crash",
        (2, false) => "cluster-reopened",
        (1, true) => "cluster-prefix-replay",
        (2, true) => "cluster-full-replay",
        _ => return Err("cluster phase incarnation differs".into()),
    }
    .into())
}
fn boundary(
    before: bool,
    verify: impl FnOnce() -> Result<(), String>,
    memory: impl FnOnce() -> Result<(), String>,
    observe: impl FnOnce() -> Result<(), String>,
) -> Result<(), String> {
    verify()?;
    if before {
        memory()?;
    }
    observe()
}
impl LifecycleObserver for Observer {
    fn before(&self, phase: &LifecyclePhase) -> Result<(), String> {
        self.observe(phase, true)
    }
    fn after(&self, phase: &LifecyclePhase) -> Result<(), String> {
        self.observe(phase, false)
    }
}

#[cfg(test)]
#[path = "cluster_admission_tests.rs"]
mod tests;

pub(super) fn check_observations(root: &Path, replay: bool) -> Result<(), String> {
    let phases = if replay {
        ["cluster-prefix-replay", "cluster-full-replay"]
    } else {
        ["cluster-crash", "cluster-reopened"]
    };
    let name = root
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("cluster evidence name absent")?;
    let command: serde_json::Value = crate::qualify::spec::strict(&super::read(
        &root
            .parent()
            .ok_or("cluster evidence parent absent")?
            .join(format!("{name}-driver.json")),
        1 << 20,
    )?)?;
    let expected_pid = command["pid"]
        .as_u64()
        .ok_or("cluster command process absent")?;
    let mut last = 0;
    for (index, phase) in phases.into_iter().enumerate() {
        for before in [true, false] {
            let path = root.join("phase-admission").join(format!(
                "{phase}-{}.json",
                if before { "before" } else { "after" }
            ));
            let value: serde_json::Value =
                crate::qualify::spec::strict(&super::read(&path, 1 << 20)?)?;
            let observed_pid = value["process_id"]
                .as_u64()
                .filter(|value| *value > 0)
                .ok_or("cluster phase process absent")?;
            let time = value["unix_nanos"]
                .as_u64()
                .ok_or("cluster phase time absent")?;
            if value["schema"] != "gomad.cluster-phase-observation/v1"
                || value["phase"] != phase
                || value["before"] != before
                || value["passed"] != true
                || time <= last
                || observed_pid != expected_pid
                || value["handles"]
                    != json!(["active", "standby"].map(|node| Handle {
                        node: node.into(),
                        incarnation: index as u64 + 1
                    }))
            {
                return Err("cluster phase observation identity/order differs".into());
            }
            last = time;
        }
    }
    Ok(())
}
