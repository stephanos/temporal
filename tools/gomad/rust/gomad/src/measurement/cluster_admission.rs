use super::super::{build, host, inputs, module, workloads};
use crate::evidence::canonical;
use gomad_wasmhost::linked_cluster::{Handle, LifecycleObserver, LifecyclePhase};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    os::unix::fs::DirBuilderExt,
    path::{Path, PathBuf},
};

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct HostIdentity {
    os: String,
    arch: String,
    boot_id: String,
    witness: host::HostWitness,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct SourceIdentity {
    schema: String,
    #[serde(rename = "source_commit")]
    head: String,
    #[serde(rename = "source_dirty")]
    dirty: bool,
    status: String,
    #[serde(rename = "source_inputs")]
    inputs: BTreeMap<PathBuf, String>,
}
impl SourceIdentity {
    fn captured(source: &host::SourceSnapshot) -> Result<Self, String> {
        serde_json::from_value(serde_json::to_value(source).map_err(|error| error.to_string())?)
            .map_err(|error| error.to_string())
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Baseline {
    host: HostIdentity,
    source: SourceIdentity,
    host_inputs: BTreeMap<PathBuf, String>,
}
impl Baseline {
    pub(super) fn admitted(
        host: &host::HostEvidence,
        source: &host::SourceSnapshot,
    ) -> Result<Self, String> {
        Self::observed(host, SourceIdentity::captured(source)?)
    }
    fn observed(host: &host::HostEvidence, source: SourceIdentity) -> Result<Self, String> {
        if host.schema != "gomad.native-measurement-host/v1"
            || host.boot_id.is_empty()
            || source.schema != "gomad.native-measurement-source/v1"
            || source.dirty
            || !source.status.is_empty()
            || source.head.len() != 40
            || !source.head.bytes().all(|byte| byte.is_ascii_hexdigit())
            || source.inputs.is_empty()
        {
            return Err("diagnostic CPU/boot/source identity changed or worktree is dirty".into());
        }
        Ok(Self {
            host: HostIdentity {
                os: host.host_os.clone(),
                arch: host.host_arch.clone(),
                boot_id: host.boot_id.clone(),
                witness: host.witness.clone(),
            },
            source,
            host_inputs: host.inputs.clone(),
        })
    }
    fn matches(&self, host: &host::HostEvidence, source: SourceIdentity) -> Result<(), String> {
        let actual = Self::observed(host, source)?;
        if self.host != actual.host || self.source != actual.source {
            return Err("diagnostic CPU/boot/source identity changed or worktree is dirty".into());
        }
        Ok(())
    }
}

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
    baseline: Baseline,
}
impl Spec {
    pub(super) fn prepared(
        prepared: &module::Prepared,
        proof: &inputs::CheckoutProof,
        helper: &workloads::Executable,
        memory_mode: &str,
        replay: bool,
        baseline: &Baseline,
    ) -> Result<Self, String> {
        let provenance = module::authenticate_prepared(prepared)?;
        let mut files = prepared_inputs(&provenance)?;
        files.extend(baseline.host_inputs.clone());
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
            baseline: baseline.clone(),
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
    fn verify(&self, proof: &inputs::CheckoutProof) -> Result<(), String> {
        module::verify_inputs(&self.files)?;
        workloads::Executable {
            path: self.helper.clone(),
            sha256: self.helper_sha256.clone(),
            role: self.role.clone(),
            receipt: self.receipt.clone(),
        }
        .verify(proof)
    }
}

fn prepared_inputs(provenance: &Value) -> Result<BTreeMap<PathBuf, String>, String> {
    let mut files = BTreeMap::new();
    for name in ["compiler", "sources"] {
        files.extend(
            serde_json::from_value::<BTreeMap<PathBuf, String>>(provenance["inputs"][name].clone())
                .map_err(|error| error.to_string())?,
        );
    }
    let compiler = PathBuf::from(
        provenance["inputs"]["compiler_path"]
            .as_str()
            .ok_or("cluster phase compiler path absent")?,
    );
    let digest = provenance["inputs"]["compiler_sha256"]
        .as_str()
        .ok_or("cluster phase compiler digest absent")?;
    if !compiler.is_absolute() || !canonical::valid_digest(digest) {
        return Err("cluster phase absolute compiler path and digest required".into());
    }
    if files
        .insert(compiler, digest.into())
        .is_some_and(|previous| previous != digest)
    {
        return Err("cluster phase compiler pin conflicts with preparation inventory".into());
    }
    Ok(files)
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
        let proof = inputs::authenticate_checkout(&self.spec.repo)?;
        let identity = self.root.join(format!(
            "{label}-{}-identity",
            if before { "before" } else { "after" }
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&identity)
            .map_err(|error| error.to_string())?;
        let host = host::current(&identity, &self.environment)?;
        let source = host::source_snapshot(&proof, &identity, &self.environment)?;
        let mut receipts = BTreeMap::new();
        for (name, value) in [
            ("host.json", serde_json::to_value(&host)),
            ("source.json", serde_json::to_value(&source)),
        ] {
            let bytes = canonical::encode(&value.map_err(|error| error.to_string())?)?;
            super::write(&identity.join(name), &bytes)?;
            receipts.insert(name.to_owned(), canonical::digest(&bytes));
        }
        // Retain refused observations too, without publishing a passing phase receipt.
        self.spec
            .baseline
            .matches(&host, SourceIdentity::captured(&source)?)?;
        boundary(
            before,
            || self.spec.verify(&proof),
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
                    "handles":phase.handles,"process_id":std::process::id(),"unix_nanos":std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map_err(|error|error.to_string())?.as_nanos(),"passed":true,"identity_receipts":receipts}),
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
    if before {
        memory()?;
    }
    verify()?;
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
    let spec = Spec::read(
        &root
            .parent()
            .ok_or("cluster evidence parent absent")?
            .join(format!("{name}-phase-inputs.json")),
    )?;
    if spec.replay != replay {
        return Err("cluster admission replay identity differs".into());
    }
    module::verify_inputs(&spec.files)?;
    module::verify_inputs(&spec.baseline.source.inputs)?;
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
            let identity = root.join("phase-admission").join(format!(
                "{phase}-{}-identity",
                if before { "before" } else { "after" }
            ));
            check_identity(&spec.baseline, &identity, &value["identity_receipts"])?;
            last = time;
        }
    }
    Ok(())
}

fn check_identity(baseline: &Baseline, root: &Path, receipts: &Value) -> Result<(), String> {
    use std::os::unix::fs::MetadataExt;
    let metadata = std::fs::symlink_metadata(root).map_err(|error| error.to_string())?;
    if !metadata.is_dir() || metadata.mode() & 0o777 != 0o700 {
        return Err("private cluster identity directory required".into());
    }
    let receipts: BTreeMap<String, String> =
        serde_json::from_value(receipts.clone()).map_err(|error| error.to_string())?;
    if receipts.len() != 2 {
        return Err("cluster phase host/source receipts absent".into());
    }
    let mut documents: BTreeMap<&str, Value> = BTreeMap::new();
    for name in ["host.json", "source.json"] {
        let bytes = read_identity_file(&root.join(name), 16 << 20)?;
        if receipts.get(name) != Some(&canonical::digest(&bytes)) {
            return Err("cluster phase host/source receipt changed".into());
        }
        documents.insert(name, crate::qualify::spec::strict(&bytes)?);
    }
    let host: host::HostEvidence = serde_json::from_value(documents["host.json"].clone())
        .map_err(|error| error.to_string())?;
    let source: SourceIdentity = serde_json::from_value(documents["source.json"].clone())
        .map_err(|error| error.to_string())?;
    baseline.matches(&host, source)?;
    if host.inputs.keys().any(|path| path.parent() != Some(root)) {
        return Err("cluster phase raw host witness path differs".into());
    }
    if !host.inputs.contains_key(&root.join("host-witness.json")) {
        return Err("cluster phase host witness pin absent".into());
    }
    for (path, digest) in &host.inputs {
        if canonical::digest(&read_identity_file(path, 2 << 20)?) != *digest {
            return Err("cluster phase raw host witness changed".into());
        }
    }
    let witness: host::HostWitness = serde_json::from_value(crate::qualify::spec::strict(
        &read_identity_file(&root.join("host-witness.json"), 2 << 20)?,
    )?)
    .map_err(|error| error.to_string())?;
    if witness != host.witness {
        return Err("cluster phase raw host witness differs".into());
    }
    if host.host_os == "linux" {
        let cpu = read_identity_file(&root.join("host-cpuinfo.txt"), host::CPUINFO_LIMIT)?;
        let boot = read_identity_file(&root.join("host-boot-id.txt"), 128)?;
        if cpu != host.witness.cpuinfo.as_bytes()
            || std::str::from_utf8(&boot)
                .map_err(|error| error.to_string())?
                .trim()
                != host.boot_id
        {
            return Err("cluster phase raw CPU/boot witness differs".into());
        }
    }
    Ok(())
}

fn read_identity_file(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    use crate::evidence::store::open;
    let parent = path.parent().ok_or("cluster identity file parent absent")?;
    let root = open::pin(parent)?;
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("cluster identity file name absent")?;
    open::read(&root, name, limit, false)
}
