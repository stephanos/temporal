use super::{
    BTreeSet, Path, PathBuf, RetainedSession, Value, canonical, custody, digest, preparation,
    require, text, unsigned,
};
use crate::qualify::result::{self, Manifest, QualificationResult};

pub(super) const CHOICE_SOURCES: [&str; 7] = [
    "choice/schema/choicewire.json",
    "choice/schema/choicewire.go.tmpl",
    "choice/schema/choicewire_runtime.go.tmpl",
    "choice/schema/choicewire_test.go.tmpl",
    "runtimesource/gomad_semantics.go.txt",
    "choice/trace.go",
    "choice/tape.go",
];
impl RetainedSession {
    pub(crate) fn provenance(&self, provenance: &Value) -> Result<Manifest, String> {
        let inputs = &provenance["inputs"];
        let configuration = &provenance["configuration"];
        require(
            inputs["host"] == self.config["identity"]["host"]
                && inputs["helper"] == self.config["identity"]["helper"],
            "prepared provenance belongs to foreign release/helper",
        )?;
        let compiler = text(inputs, "compiler_path")?;
        let frozen = self.config["files"].get(compiler).and_then(Value::as_str);
        let alias = self.private_inputs.get(compiler);
        let expected = frozen.or_else(|| alias.map(|value| value.sha256.as_str()));
        require(
            expected == Some(text(inputs, "compiler_sha256")?),
            "qualification compiler path/hash not frozen",
        )?;
        if frozen.is_some() || !alias.is_some_and(|value| value.detached) {
            require(
                digest(&custody::read_file(Path::new(compiler), 512 << 20)?)
                    == text(inputs, "compiler_sha256")?,
                "actual frozen compiler changed",
            )?;
        }
        if configuration["profile"] == "gomad.wasi-cooperative/v1" {
            let inventory = crate::cycle::preparation::inputs::Inventory::new(&self.config)?;
            let root = Path::new(text(&self.config, "repo")?).join("tools/gomad/internal/model");
            for name in CHOICE_SOURCES {
                inventory.file_bytes(&root.join(name))?;
            }
        }
        let manifest = Manifest {
            schema: result::MANIFEST_SCHEMA.into(),
            provisional: false,
            integrated_source_manifest_path: None,
            id: "receiving".into(),
            workload: "receiving".into(),
            cell: None,
            integrated_source_sha256: text(&self.config, "source_sha256")?.into(),
            workload_source_sha256: digest(&canonical(&inputs["sources"])?),
            runtime_sha256: digest(&canonical(&inputs["compiler"])?),
            compiler_sha256: text(inputs, "compiler_sha256")?.into(),
            helper_sha256: text(inputs, "helper")?.into(),
            host_sha256: text(inputs, "host")?.into(),
            model_sha256: digest(&canonical(&configuration["model"])?),
            engine_sha256: digest(text(configuration, "engine")?.as_bytes()),
            host_config_sha256: digest(&canonical(configuration)?),
            module_sha256: text(provenance, "module_sha256")?.into(),
            toolchain_sha256: text(provenance, "build_key")?.into(),
            scenario_sha256: if provenance["scenario"].is_null() {
                None
            } else {
                Some(digest(
                    &serde_json::from_value::<Vec<u8>>(provenance["scenario"].clone())
                        .map_err(|e| e.to_string())?,
                ))
            },
            host_os: std::env::consts::OS.into(),
            host_arch: std::env::consts::ARCH.into(),
            seed: 7,
            repeat: 2,
            profile: result::profile_name(text(configuration, "profile")?)?.into(),
            fuel: unsigned(&configuration["limits"], "fuel")?,
            memory_bytes: unsigned(&configuration["limits"], "memory_bytes")?,
            unsupported: Vec::new(),
        };
        result::bind_provenance(&manifest, provenance)?;
        Ok(manifest)
    }
    pub(super) fn final_contract(&self, retained: &QualificationResult) -> Result<(), String> {
        let manifest = &retained.manifest;
        let source_path = Path::new(text(&self.config, "source_manifest")?);
        let source = custody::read_file(source_path, 16 << 20)?;
        require(
            digest(&source) == self.config["source_sha256"]
                && retained
                    .integrated_source_manifest
                    .as_deref()
                    .map(str::as_bytes)
                    == Some(source.as_slice()),
            "Result integrated source differs from frozen candidate",
        )?;
        require(
            !manifest.provisional
                && manifest.seed == 7
                && manifest.repeat == 2
                && manifest.integrated_source_sha256 == self.config["source_sha256"]
                && manifest.host_sha256 == self.config["identity"]["host"]
                && manifest.helper_sha256 == self.config["identity"]["helper"],
            "foreign/final Result source/role/seed/repeat",
        )?;
        self.provenance(&retained.provenance)?;
        let report = &retained.report;
        let success = report
            .executions
            .iter()
            .all(|row| row.record.outcome.class == "success");
        require(
            report.executions.len() == 2
                && report.executions.iter().map(|row| row.iteration).eq([1, 2])
                && report.seed == 7
                && report.repeat == 2
                && !report.command.is_empty()
                && report.failure.is_none()
                && report.deterministic
                && report.qualified == success
                && report.target_success == success
                && report.first_divergence.is_none(),
            "Result lacks complete deterministic final outcomes",
        )?;
        require(
            retained.cost_failure.is_none()
                && retained.cost.is_some()
                && retained.budget_failure.is_none()
                && retained
                    .budget
                    .as_ref()
                    .is_some_and(|budget| budget.within_budget),
            "Result lacks complete within-budget raw accounting",
        )?;
        for execution in &report.executions {
            Self::stream_owners(&execution.record)?;
            for phase in ["execute", "replay"] {
                require(
                    retained.observations.samples.iter().any(|sample| {
                        sample.phase == phase && sample.iteration == execution.iteration
                    }),
                    "Result raw store sample coverage differs",
                )?;
            }
        }
        Ok(())
    }
    pub(super) fn stream_owners(record: &crate::evidence::record::Record) -> Result<(), String> {
        for name in record.streams.keys() {
            if let Some(owner) = name.strip_prefix("evidence/cluster/") {
                let parts = owner.splitn(3, '/').collect::<Vec<_>>();
                require(
                    parts.len() == 3 && parts.iter().all(|part| !part.is_empty()),
                    "malformed cluster stream owner",
                )?;
            }
        }
        Ok(())
    }
    pub(super) fn final_publication(
        retained: &QualificationResult,
        receipt: &result::Receipt,
        terminal: &crate::qualify::cost::Terminal,
    ) -> Result<(), String> {
        require(
            receipt.linked_after.is_some()
                && receipt.operator_peak_rss_bytes.is_some_and(|peak| {
                    peak > 0
                        && peak <= 2 << 30
                        && retained
                            .observations
                            .operator_peak_rss_bytes
                            .is_some_and(|before| before > 0 && before <= peak)
                })
                && terminal.status == "within",
            "final Result publication lacks attributed within-budget measurement",
        )
    }
    #[expect(
        clippy::too_many_lines,
        reason = "One Result transaction independently reopens all execution and journal owners"
    )]
    pub(crate) fn result(
        &mut self,
        path: &Path,
        reported: &Value,
    ) -> Result<QualificationResult, String> {
        self.contained(path)?;
        self.directory(path.parent().ok_or("Result parent missing")?)?;
        let retained = result::open(path)?;
        let (receipt, terminal) = result::open_receipt(path, &retained)?;
        require(
            reported["result"] == serde_json::to_value(&retained).map_err(|e| e.to_string())?
                && reported["receipt"]
                    == serde_json::to_value(&receipt).map_err(|e| e.to_string())?
                && reported["publication"]
                    == serde_json::to_value(&terminal).map_err(|e| e.to_string())?,
            "public report differs from raw Result/receipt/publication",
        )?;
        Self::final_publication(&retained, &receipt, &terminal)?;
        self.final_contract(&retained)?;
        let manifest = &retained.manifest;
        let report = &retained.report;
        let campaigns = report
            .executions
            .iter()
            .map(|row| PathBuf::from(&row.campaign_path))
            .collect::<BTreeSet<_>>();
        let mut paths = BTreeSet::new();
        let mut pending = Vec::new();
        for execution in &report.executions {
            let path = Path::new(
                execution
                    .artifact_path
                    .as_deref()
                    .ok_or("Result execution artifact missing")?,
            );
            require(
                paths.insert(path.to_owned()),
                "duplicate qualification artifact",
            )?;
            let artifact = self.artifact(path)?;
            require(
                artifact.manifest.record == execution.record
                    && artifact.manifest.record_sha256 == execution.record_sha256,
                "Result differs from independently opened artifact",
            )?;
            let module = artifact
                .payloads
                .get("target/module.wasm")
                .ok_or("Result raw module missing")?;
            let metadata =
                gomad_wasmhost::linked::module_metadata(module).map_err(|e| e.to_string())?;
            require(
                serde_json::to_value(metadata).map_err(|e| e.to_string())?
                    == retained.provenance["module_metadata"],
                "Result module metadata differs from raw module",
            )?;
            let replay = execution
                .replay
                .as_ref()
                .ok_or("Result full replay missing")?;
            require(
                replay.attempted
                    && replay.matched
                    && replay.divergence.is_none()
                    && replay.choice_replay
                        == if manifest.profile == "cooperative" {
                            "exact"
                        } else {
                            "none"
                        },
                "incomplete full-tape Result replay",
            )?;
            let campaign = Path::new(&execution.campaign_path);
            self.contained(campaign)?;
            let journal = crate::campaign::ledger::Journal::open(campaign, false)?;
            let matches = journal
                .events()
                .iter()
                .filter_map(|event| {
                    if let crate::campaign::ledger::Event::Receipt { execution } = event {
                        Some(execution)
                    } else {
                        None
                    }
                })
                .filter(|row| row.record.hash().as_deref() == Ok(execution.record_sha256.as_str()))
                .collect::<Vec<_>>();
            require(
                matches.len() == 1,
                "Result journal execution missing/duplicate",
            )?;
            crate::campaign::ledger::authenticate_execution(
                journal.plan(),
                journal.root(),
                matches[0],
            )?;
            require(
                journal
                    .events()
                    .iter()
                    .any(|event| matches!(event, crate::campaign::ledger::Event::Complete { .. })),
                "Result campaign incomplete",
            )?;
            pending.push(preparation::Pending {
                campaign: campaign.to_owned(),
                provenance: retained.provenance.clone(),
                owner: retained.observations.operator_process_id,
                campaigns: campaigns.clone(),
            });
        }
        require(campaigns.len() == 2, "duplicate Result campaign")?;
        self.pending_preparations.extend(pending);
        self.check_root()?;
        Ok(retained)
    }
}
