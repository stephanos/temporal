use super::{
    BTreeMap, BTreeSet, Path, PathBuf, RetainedSession, Value, argument, canonical, custody,
    digest, open, require, text, unsigned,
};
use serde::Deserialize;
use serde_json::json;

#[derive(Clone, Debug, PartialEq)]
pub(super) struct PreparedObservation {
    pub sidecar: PathBuf,
    pub provenance: Value,
    pub owner: u64,
    pub campaigns: BTreeSet<PathBuf>,
    pub installation: PathBuf,
}
pub(super) struct Pending {
    pub campaign: PathBuf,
    pub provenance: Value,
    pub owner: u64,
    pub campaigns: BTreeSet<PathBuf>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Seal {
    source_sha256: String,
    binary_sha256: String,
    campaigns: Vec<Campaign>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Campaign {
    path: PathBuf,
    receipt_path: PathBuf,
    receipt_sha256: String,
    files: BTreeMap<String, String>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Blob {
    bytes: u64,
    sha256: String,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Receipt {
    schema: String,
    owner_pid: u64,
    host_sha256: String,
    helper_sha256: String,
    prepared_record_sha256: String,
    module_sha256: String,
    provenance_sha256: String,
    build_key: String,
    compiler_sha256: String,
    sources_sha256: String,
    files: BTreeMap<String, Blob>,
}
impl RetainedSession {
    #[expect(
        clippy::too_many_lines,
        reason = "One preparation transaction retains all candidate and producer bindings until admission"
    )]
    pub(crate) fn capture_preparations(&mut self, row: &Value) -> Result<(), String> {
        let Some(seal) = row.get("private_observations").filter(|v| !v.is_null()) else {
            return Ok(());
        };
        let seal: Seal = crate::qualify::spec::strict(&canonical(seal)?)?;
        require(
            seal.source_sha256 == self.config["source_sha256"]
                && seal.binary_sha256 == self.config["binary_sha256"],
            "preparation outer seal belongs to foreign source/binary",
        )?;
        let argv: Vec<String> =
            serde_json::from_value(row["argv"].clone()).map_err(|e| e.to_string())?;
        let runtime = argv.len() == 6 && argv[1] == "produce";
        let installation = if runtime {
            require(
                self.config["stages"]
                    .as_object()
                    .is_some_and(|stages| stages.values().any(|stage| stage["harness"] == argv[0])),
                "preparation runtime is not frozen release harness",
            )?;
            require(
                self.config["files"][&argv[0]]
                    == digest(&custody::read_file(Path::new(&argv[0]), 512 << 20)?),
                "preparation runtime role changed",
            )?;
            require(
                seal.campaigns.len() == 1
                    && seal.campaigns[0].path == Path::new(&argv[5]).join("preparation"),
                "runtime preparation root changed",
            )?;
            PathBuf::from(&argv[2])
        } else {
            require(
                argv.first().is_some_and(|v| self.config["binary"] == *v),
                "preparation command uses foreign host",
            )?;
            PathBuf::from(argument(&argv, "--installation")?)
        };
        let campaigns = seal
            .campaigns
            .iter()
            .map(|v| v.path.clone())
            .collect::<BTreeSet<_>>();
        require(
            !campaigns.is_empty()
                && campaigns.len() == seal.campaigns.len()
                && campaigns.len() <= 32,
            "preparation campaign set missing/duplicate/oversized",
        )?;
        let mut admitted = Vec::new();
        for campaign in &seal.campaigns {
            self.contained(&campaign.path)?;
            let sidecar = campaign.path.join("preparation-observations");
            require(
                campaign.receipt_path == sidecar.join("receipt.json"),
                "preparation receipt belongs to another campaign",
            )?;
            let raw = self.read(&campaign.receipt_path, 16 << 20)?;
            require(
                digest(&raw) == campaign.receipt_sha256,
                "preparation receipt differs from actual command seal",
            )?;
            let value = custody::strict_json(&raw)?;
            let receipt: Receipt = crate::qualify::spec::strict(&raw)?;
            require(
                raw == canonical(&value)?
                    && receipt.schema == "gomad.native-preparation-observation-receipt/v1"
                    && receipt.owner_pid > 0
                    && receipt.owner_pid == unsigned(row, "pid")?
                    && receipt.host_sha256 == self.config["identity"]["host"]
                    && receipt.helper_sha256 == self.config["identity"]["helper"],
                "preparation receipt/current command owner mismatch",
            )?;
            require(
                !receipt.files.is_empty() && receipt.files.len() <= 32,
                "preparation bounded raw file table missing",
            )?;
            let mut files = BTreeMap::new();
            for (name, entry) in &receipt.files {
                open::components(name)?;
                require(
                    name.starts_with("target/preparation-inputs/") && entry.bytes <= 128 << 20,
                    "preparation unsafe/oversized raw file table",
                )?;
                let target = sidecar.join(name);
                let bytes = self.read(&target, 128 << 20)?;
                require(
                    bytes.len() as u64 == entry.bytes && digest(&bytes) == entry.sha256,
                    "preparation raw sidecar file changed",
                )?;
                files.insert(target.to_string_lossy().into_owned(), entry.sha256.clone());
            }
            require(
                files == campaign.files,
                "preparation exact outer-sealed raw file set differs",
            )?;
            let plan = self.json(&campaign.path.join("plan.json"), 16 << 20)?;
            let plan = plan.get("prepared").unwrap_or(&plan);
            let reference = text(plan, "prepared_artifact")?;
            open::components(reference)?;
            let artifact = self.artifact(&campaign.path.join(reference))?;
            crate::campaign::authenticate_preparation_observations(
                &self.directory(&campaign.path)?,
                plan,
            )?;
            let provenance_raw = artifact
                .payloads
                .get("target/provenance.json")
                .ok_or("prepared provenance missing")?;
            let provenance = custody::strict_json(provenance_raw)?;
            let inputs = &provenance["inputs"];
            let configuration = &provenance["configuration"];
            let record = &artifact.manifest.record;
            require(
                artifact.manifest.record_sha256 == receipt.prepared_record_sha256
                    && digest(
                        artifact
                            .payloads
                            .get("target/module.wasm")
                            .ok_or("prepared module missing")?,
                    ) == receipt.module_sha256
                    && provenance["module_sha256"] == receipt.module_sha256
                    && digest(provenance_raw) == receipt.provenance_sha256
                    && provenance["build_key"] == receipt.build_key
                    && digest(&canonical(&inputs["compiler"])?) == receipt.compiler_sha256
                    && digest(&canonical(&inputs["sources"])?) == receipt.sources_sha256,
                "preparation receipt stable commitments changed",
            )?;
            require(
                record.seed == 0
                    && record.virtual_time_nanos == 0
                    && record.outcome.class == "prepared"
                    && record.outcome.termination == "prepared"
                    && record.outcome.exit_code.is_none()
                    && record.outcome.reason.is_empty(),
                "prepared record role/outcome changed",
            )?;
            let expected_identity = json!({"target":receipt.module_sha256,"toolchain":receipt.build_key,"configuration":digest(&canonical(configuration)?),"implementation":digest(&canonical(&self.config["identity"])?),"platform":digest(format!("{}/{}",std::env::consts::OS,std::env::consts::ARCH).as_bytes())});
            let actual_record = serde_json::to_value(record).map_err(|e| e.to_string())?;
            require(
                actual_record["identity"] == expected_identity
                    && ["profile", "arguments", "environment", "limits"]
                        .iter()
                        .all(|key| actual_record[key] == configuration[key]),
                "prepared record role/configuration changed",
            )?;
            let replay = custody::strict_json(
                artifact
                    .payloads
                    .get("evidence/replay.json")
                    .ok_or("prepared replay missing")?,
            )?;
            self.prepared_replay(&replay, &provenance)?;
            self.provenance(&provenance)?;
            let value = PreparedObservation {
                sidecar,
                provenance,
                owner: receipt.owner_pid,
                campaigns: campaigns.clone(),
                installation: installation.clone(),
            };
            require(
                self.preparations
                    .get(&campaign.path)
                    .is_none_or(|previous| *previous == value),
                "preparation campaign borrowed conflicting producer",
            )?;
            admitted.push((campaign.path.clone(), value));
        }
        self.preparations.extend(admitted);
        Ok(())
    }
    pub(super) fn prepared_replay(&self, replay: &Value, provenance: &Value) -> Result<(), String> {
        require(
            replay
                == &json!({"schema":"gomad.replay-inputs/v1","helper_sha256":self.config["identity"]["helper"],"host_sha256":self.config["identity"]["host"],"toolchain_sha256":provenance["build_key"],"runtime_implementation_sha256":provenance["inputs"]["runtime_implementation"],"configuration":provenance["configuration"]}),
            "prepared replay preimage changed",
        )
    }
    pub(crate) fn prepared_observation(
        &self,
        campaign: &Path,
        provenance: &Value,
    ) -> Result<(&Path, u64), String> {
        self.contained(campaign)?;
        let value = self
            .preparations
            .get(campaign)
            .ok_or("preparation lacks actual command observation seal")?;
        require(
            value.provenance == *provenance,
            "execution borrowed different prepared provenance",
        )?;
        Ok((&value.sidecar, value.owner))
    }
    pub(crate) fn verify_pending_preparations(&mut self) -> Result<(), String> {
        for pending in &self.pending_preparations {
            let (_, owner) = self.prepared_observation(&pending.campaign, &pending.provenance)?;
            let observed = self
                .preparations
                .get(&pending.campaign)
                .ok_or("missing preparation")?;
            require(
                owner == pending.owner && observed.campaigns == pending.campaigns,
                "Result producer/complete campaign set differs",
            )?;
            let installation_raw = custody::read_file(&observed.installation, 16 << 20)?;
            let installation = custody::strict_json(&installation_raw)?;
            let typed = crate::qualify::spec::strict(&installation_raw)?;
            crate::install::manifest::validate_for_executable(
                &typed,
                text(&self.config, "binary_sha256")?,
            )?;
            let inputs = &pending.provenance["inputs"];
            let compiler = installation["admission"]["compiler_inputs"]
                .as_object()
                .ok_or("complete installation compiler inventory missing")?;
            require(
                installation["profile"] == pending.provenance["configuration"]["profile"]
                    && installation["compiler_path"] == inputs["compiler_path"]
                    && installation["compiler_sha256"] == inputs["compiler_sha256"]
                    && installation["runtime_implementation_sha256"]
                        == inputs["runtime_implementation"]
                    && installation["admission"]["host_executable_sha256"]
                        == self.config["binary_sha256"]
                    && ["host", "helper"].iter().all(|role| {
                        installation[format!("{role}_sha256")] == self.config["identity"][role]
                    })
                    && compiler
                        .iter()
                        .all(|(path, sha)| self.config["files"][path] == *sha)
                    && inputs["compiler"].as_object().is_some_and(|map| {
                        map.iter()
                            .all(|(path, sha)| compiler.get(path) == Some(sha))
                    }),
                "actual installation/complete compiler inventory differs",
            )?;
            crate::cycle::preparation::inputs::verify_preparation(
                &self.config,
                &observed.sidecar,
                &installation,
                inputs,
                Some(owner),
            )?;
        }
        self.pending_preparations.clear();
        Ok(())
    }
}
