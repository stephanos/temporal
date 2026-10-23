use super::{exploration, ledger::Journal, prepare::Prepared};
use crate::qualify::usage::{self, Snapshot};
use crate::{
    cli::{exploration::ExplorationOptions, explore::ExploreOptions},
    evidence::{
        canonical,
        store::{Opened, open},
    },
};
use rustix::fs::{self, AtFlags, FileType, Mode, OFlags};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest as _, Sha256};
use std::{
    fs::File,
    io::{Read, Write},
    path::{Path, PathBuf},
    time::Instant,
};

const MAXIMUM: u64 = 32 << 20;

pub(crate) struct PreparedPlan {
    pub prepared: Prepared,
    pub campaign: PathBuf,
    pub plan: Value,
    pub guest_executions: u64,
    pub envelope: Option<Value>,
    preparation_start: Option<Snapshot>,
    preparation_end: Option<Snapshot>,
    preparation_elapsed_nanos: u64,
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct PreparationReceipt {
    pub schema: String,
    pub plan: PathBuf,
    pub plan_sha256: String,
    pub campaign: PathBuf,
    pub prepared_artifact: String,
    pub record_sha256: String,
    pub module_file: String,
    pub module_bytes: u64,
    pub module_sha256: String,
    pub provenance_file: String,
    pub provenance_bytes: u64,
    pub provenance_sha256: String,
    pub guest_executions: u64,
    pub command: Vec<String>,
    pub exit_code: i32,
    pub elapsed_nanos: u64,
    pub process_id: u32,
    pub compiler_cpu_nanos: Option<u64>,
    pub compiler_lifetime_peak_rss_bytes: Option<u64>,
    pub preparation_start: Option<Snapshot>,
    pub preparation_end: Option<Snapshot>,
}

pub(crate) fn retain_admitted(
    options: &ExploreOptions,
    prepared: Prepared,
) -> Result<PreparedPlan, String> {
    let started = Instant::now();
    let snapshot = usage::snapshot();
    let journal = super::seed_journal(options, &prepared)?;
    Ok(PreparedPlan {
        prepared,
        campaign: std::fs::canonicalize(journal.path()).map_err(|e| e.to_string())?,
        plan: journal.plan().clone(),
        guest_executions: 0,
        envelope: None,
        preparation_start: snapshot,
        preparation_end: snapshot,
        preparation_elapsed_nanos: elapsed(started)?,
    })
}

pub(crate) fn seed(options: &ExploreOptions) -> Result<PreparedPlan, String> {
    let started = Instant::now();
    let preparation_start = usage::snapshot();
    let prepared = super::prepare::prepare(options)?;
    let preparation_end = usage::snapshot();
    let preparation_elapsed_nanos = elapsed(started)?;
    let journal = super::seed_journal(options, &prepared)?;
    Ok(PreparedPlan {
        prepared,
        campaign: std::fs::canonicalize(journal.path()).map_err(|error| error.to_string())?,
        plan: journal.plan().clone(),
        guest_executions: 0,
        envelope: None,
        preparation_start,
        preparation_end,
        preparation_elapsed_nanos,
    })
}

pub(crate) fn choice(options: &ExplorationOptions) -> Result<PreparedPlan, String> {
    let started = Instant::now();
    let preparation_start = usage::snapshot();
    let prepared = super::prepare::prepare(&options.base)?;
    let preparation_end = usage::snapshot();
    let preparation_elapsed_nanos = elapsed(started)?;
    let (mut journal, initial) = super::choice::prepare_journal(options, &prepared, started)?;
    let guest_executions = u64::from(initial.is_some());
    if let Some(initial) = initial {
        let remaining = options
            .base
            .overall_timeout
            .checked_sub(started.elapsed())
            .ok_or("overall_timeout: choice plan deadline exceeded during preparation")?;
        super::choice::commit_planned_base(&mut journal, &prepared, remaining, initial)?;
    }
    Ok(PreparedPlan {
        prepared,
        campaign: std::fs::canonicalize(journal.path()).map_err(|error| error.to_string())?,
        plan: serde_json::to_value(journal.plan()).map_err(|error| error.to_string())?,
        guest_executions,
        envelope: Some(journal.summary()?),
        preparation_start,
        preparation_end,
        preparation_elapsed_nanos,
    })
}

fn elapsed(started: Instant) -> Result<u64, String> {
    u64::try_from(started.elapsed().as_nanos())
        .map_err(|_| "preparation elapsed measurement overflow".into())
}

fn compiler_cpu(start: Option<Snapshot>, end: Option<Snapshot>) -> Result<Option<u64>, String> {
    let (Some(start), Some(end)) = (start, end) else {
        return Ok(None);
    };
    if !end.children.not_before(start.children) || !end.linked.not_before(start.linked) {
        return Err("preparation resource counters regressed".into());
    }
    end.children
        .user_nanos
        .checked_sub(start.children.user_nanos)
        .and_then(|user| {
            end.children
                .system_nanos
                .checked_sub(start.children.system_nanos)
                .and_then(|system| user.checked_add(system))
        })
        .map(Some)
        .ok_or_else(|| "preparation compiler CPU measurement overflow".into())
}

fn metadata(plan: &Value) -> &Value {
    if plan["prepared"].is_object() {
        &plan["prepared"]
    } else {
        plan
    }
}

fn artifact(campaign: &Path, plan: &Value) -> Result<Opened, String> {
    let reference = metadata(plan)["prepared_artifact"]
        .as_str()
        .ok_or("prepared artifact missing")?;
    open::components(reference)?;
    if plan["schema"] == "gomad.seed-campaign-plan/v2" {
        let journal = Journal::open(campaign, false)?;
        if journal.plan() != plan || !journal.events().is_empty() {
            return Err("seed preparation journal differs".into());
        }
        journal.open_artifact(reference)
    } else {
        let typed: exploration::round::RoundPlan =
            serde_json::from_value(plan.clone()).map_err(|error| error.to_string())?;
        let exploration::round::RoundStrategy::Choice(config) = typed.strategy else {
            return Err("unsupported preparation plan strategy".into());
        };
        let journal = exploration::round::RoundJournal::open(campaign, false)?;
        let executions = journal
            .events()
            .iter()
            .flat_map(exploration::round::RoundEvent::executions)
            .count();
        if serde_json::to_value(journal.plan()).map_err(|error| error.to_string())? != *plan
            || executions != usize::from(config.site_scope.is_some())
        {
            return Err("choice preparation journal differs".into());
        }
        journal.open_artifact(reference)
    }
}

fn output_parts(path: &Path) -> Result<(File, String, PathBuf), String> {
    let parent = path
        .parent()
        .filter(|path| !path.as_os_str().is_empty())
        .unwrap_or(Path::new("."));
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or("invalid plan output path")?;
    open::components(name)?;
    let absolute = std::fs::canonicalize(parent)
        .map_err(|error| error.to_string())?
        .join(name);
    let before = fs::statat(fs::CWD, parent, AtFlags::SYMLINK_NOFOLLOW)
        .map_err(|error| error.to_string())?;
    let root = File::from(
        fs::open(
            parent,
            OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::empty(),
        )
        .map_err(|error| error.to_string())?,
    );
    let after = fs::statat(fs::CWD, parent, AtFlags::SYMLINK_NOFOLLOW)
        .map_err(|error| error.to_string())?;
    let pinned = fs::fstat(&root).map_err(|error| error.to_string())?;
    let identity = |stat: &rustix::fs::Stat| (stat.st_dev, stat.st_ino, stat.st_mode, stat.st_uid);
    if FileType::from_raw_mode(before.st_mode) != FileType::Directory
        || identity(&before) != identity(&after)
        || identity(&after) != identity(&pinned)
    {
        return Err("preparation output parent changed while opening".into());
    }
    Ok((root, name.into(), absolute))
}

fn read_document(root: &File, name: &str) -> Result<Vec<u8>, String> {
    let file = open::child(root, name, false, false)?;
    let before = fs::fstat(&file).map_err(|error| error.to_string())?;
    let mut bytes = Vec::new();
    (&file)
        .take(MAXIMUM + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    let after = fs::fstat(&file).map_err(|error| error.to_string())?;
    if bytes.len() as u64 > MAXIMUM || !open::same(&before, &after) {
        return Err("preparation document exceeds byte limit or changed while reading".into());
    }
    Ok(bytes)
}

fn absent(root: &File, name: &str) -> Result<(), String> {
    for destination in [name.to_owned(), format!("{name}.preparation.json")] {
        match fs::statat(root, destination.as_str(), AtFlags::SYMLINK_NOFOLLOW) {
            Err(rustix::io::Errno::NOENT) => {}
            Ok(_) => return Err("preparation output already exists".into()),
            Err(error) => return Err(error.to_string()),
        }
    }
    Ok(())
}

pub(crate) fn check_output(path: &Path) -> Result<(), String> {
    let (root, name, _) = output_parts(path)?;
    absent(&root, &name)
}

fn publish(root: &File, name: &str, bytes: &[u8]) -> Result<(), String> {
    if bytes.is_empty() || bytes.len() as u64 > MAXIMUM {
        return Err("preparation document exceeds byte limit".into());
    }
    let mut file = File::from(
        fs::openat(
            root,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())?;
    root.sync_all().map_err(|error| error.to_string())
}

impl PreparedPlan {
    pub(crate) fn publish(
        &self,
        output: &Path,
        command: Vec<String>,
    ) -> Result<(PathBuf, PreparationReceipt, String), String> {
        let (root, name, plan_path) = output_parts(output)?;
        let receipt_name = format!("{name}.preparation.json");
        absent(&root, &name)?;
        let opened = artifact(&self.campaign, &self.plan)?;
        opened.authenticate_all()?;
        let provenance = opened.payload("target/provenance.json", 16 << 20)?;
        let plan = canonical::encode(&self.plan)?;
        let receipt = PreparationReceipt {
            schema: "gomad.rust-preparation-receipt/v1".into(),
            plan: plan_path,
            plan_sha256: canonical::digest(&plan),
            campaign: self.campaign.clone(),
            prepared_artifact: metadata(&self.plan)["prepared_artifact"]
                .as_str()
                .ok_or("prepared artifact missing")?
                .into(),
            record_sha256: opened.manifest().record_sha256.clone(),
            module_file: "target/module.wasm".into(),
            module_bytes: self.prepared.module.len() as u64,
            module_sha256: canonical::digest(&self.prepared.module),
            provenance_file: "target/provenance.json".into(),
            provenance_bytes: provenance.len() as u64,
            provenance_sha256: canonical::digest(&provenance),
            guest_executions: self.guest_executions,
            command,
            exit_code: 0,
            elapsed_nanos: self.preparation_elapsed_nanos,
            process_id: std::process::id(),
            compiler_cpu_nanos: compiler_cpu(self.preparation_start, self.preparation_end)?,
            compiler_lifetime_peak_rss_bytes: self
                .preparation_end
                .map(|snapshot| snapshot.children.peak_rss_bytes),
            preparation_start: self.preparation_start,
            preparation_end: self.preparation_end,
        };
        validate(&receipt, &self.plan, &opened)?;
        let encoded = canonical::encode(&receipt)?;
        publish(&root, &name, &plan)?;
        publish(&root, &receipt_name, &encoded)?;
        Ok((
            receipt.plan.with_file_name(receipt_name),
            receipt,
            canonical::digest(&encoded),
        ))
    }
}

fn valid_inventory(value: &Value) -> bool {
    value.as_object().is_some_and(|entries| {
        !entries.is_empty()
            && entries.iter().all(|(path, digest)| {
                Path::new(path).is_absolute()
                    && digest.as_str().is_some_and(canonical::valid_digest)
            })
    })
}

fn authenticate_provenance_module(provenance: &Value, module: &[u8]) -> Result<(), String> {
    #[derive(Deserialize)]
    struct SourceInputs {
        kind: String,
        source: String,
        directory: PathBuf,
        arguments: Vec<String>,
        tags: Vec<String>,
        scenario: Option<Vec<u8>>,
        network: Option<gomad_wasmhost::linked_cluster::NetworkModel>,
        soft_assertion_report_only: bool,
    }
    super::prepare::typed_provenance(provenance)?;
    let source: SourceInputs = serde_json::from_value(provenance["inputs"].clone())
        .map_err(|error| format!("preparation source binding: {error}"))?;
    let arguments = std::iter::once("gomad-target".to_owned())
        .chain(source.arguments)
        .collect::<Vec<_>>();
    if !matches!(source.kind.as_str(), "go-test" | "go-run")
        || source.source.is_empty()
        || source.source.contains('\0')
        || !source.directory.is_absolute()
        || source.tags.iter().any(|tag| tag.contains('\0'))
        || source.tags.windows(2).any(|pair| pair[0] >= pair[1])
        || provenance["configuration"]["arguments"] != serde_json::json!(arguments)
        || provenance["scenario"] != serde_json::json!(source.scenario)
        || provenance["network"] != serde_json::json!(source.network)
        || source.soft_assertion_report_only && source.kind != "go-test"
    {
        return Err("preparation source binding mismatch".into());
    }
    let module_metadata: Option<gomad_wasmhost::linked::ModuleMetadata> =
        serde_json::from_value(provenance["module_metadata"].clone())
            .map_err(|error| error.to_string())?;
    super::prepare::authenticate_module_metadata(
        module_metadata.as_ref(),
        &provenance["inputs"],
        module,
    )
}

fn validate(
    receipt: &PreparationReceipt,
    plan: &Value,
    opened: &Opened,
) -> Result<Prepared, String> {
    if receipt.schema != "gomad.rust-preparation-receipt/v1"
        || receipt.exit_code != 0
        || receipt.process_id == 0
        || receipt.command.is_empty()
        || receipt.module_file != "target/module.wasm"
        || receipt.provenance_file != "target/provenance.json"
        || receipt.guest_executions > 1
        || metadata(plan)["prepared_artifact"] != receipt.prepared_artifact
        || receipt.record_sha256 != opened.manifest().record_sha256
        || receipt.plan_sha256 != canonical::digest(&canonical::encode(plan)?)
    {
        return Err("preparation receipt identity mismatch".into());
    }
    if receipt.compiler_cpu_nanos
        != compiler_cpu(receipt.preparation_start, receipt.preparation_end)?
        || receipt.compiler_lifetime_peak_rss_bytes
            != receipt
                .preparation_end
                .map(|snapshot| snapshot.children.peak_rss_bytes)
    {
        return Err("preparation receipt resource observation mismatch".into());
    }
    let prepared = super::restore_from_plan(metadata(plan), opened)?;
    let provenance: Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    authenticate_provenance_module(&provenance, &prepared.module)?;
    let mut configuration =
        serde_json::to_value(&prepared.configuration).map_err(|error| error.to_string())?;
    configuration["runtime_sha256"] = Value::Null;
    let runtime = (prepared.configuration.profile == "gomad.wasi-cooperative/v1")
        .then_some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"));
    if canonical::encode(&provenance)? != prepared.provenance
        || provenance["schema"] != "gomad.rust-wasm-provenance/v1"
        || provenance["inputs"]["schema"] != "gomad.rust-wasm-build-inputs/v1"
        || prepared.configuration.engine
            != gomad_wasmhost::linked::Conditions::compiled().configuration()
        || !provenance["inputs"]["compiler_path"]
            .as_str()
            .is_some_and(|path| Path::new(path).is_absolute())
        || !provenance["inputs"]["compiler_sha256"]
            .as_str()
            .is_some_and(canonical::valid_digest)
        || !matches!(
            provenance["inputs"]["go_version"].as_str(),
            Some("go1.27.0" | "go1.27.1")
        )
        || provenance["inputs"]["soft_assertion_report_only"] != prepared.soft_assertion_report_only
        || provenance["inputs"]["scenario"] != serde_json::json!(prepared.scenario)
        || provenance["inputs"]["network"] != serde_json::json!(prepared.network)
        || provenance["inputs"]["configuration"] != configuration
        || provenance["inputs"]["runtime_implementation"] != serde_json::json!(runtime)
        || provenance["scenario"] != serde_json::json!(prepared.scenario)
        || provenance["network"] != serde_json::json!(prepared.network)
        || provenance["choice_identity"] != serde_json::json!(prepared.choice_identity)
        || provenance["inputs"]["host"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || provenance["inputs"]["helper"] != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
        || !valid_inventory(&provenance["inputs"]["sources"])
        || !valid_inventory(&provenance["inputs"]["compiler"])
        || provenance["build_key"] != prepared.toolchain
        || provenance["build_key"] != canonical::digest(&canonical::encode(&provenance["inputs"])?)
        || provenance["configuration"]
            != serde_json::to_value(&prepared.configuration).map_err(|error| error.to_string())?
        || provenance["module_sha256"] != receipt.module_sha256
        || receipt.module_sha256 != canonical::digest(&prepared.module)
        || receipt.module_bytes != prepared.module.len() as u64
        || receipt.provenance_sha256 != canonical::digest(&prepared.provenance)
        || receipt.provenance_bytes != prepared.provenance.len() as u64
    {
        return Err("preparation receipt provenance mismatch".into());
    }
    if let Some(identity) = &prepared.choice_identity {
        if identity.target_sha256 != <[u8; 32]>::from(Sha256::digest(&prepared.module))
            || identity.toolchain_build_key != prepared.toolchain
            || identity.goos != "wasip1"
            || identity.goarch != "wasm"
            || prepared.configuration.runtime_sha256.as_deref()
                != Some(identity.runtime_implementation_sha256().as_str())
        {
            return Err("preparation receipt choice identity mismatch".into());
        }
    } else if runtime.is_some() {
        return Err("preparation receipt cooperative choice identity missing".into());
    }
    if plan["schema"] == "gomad.seed-campaign-plan/v2" {
        if receipt.guest_executions != 0 {
            return Err("seed preparation executed a guest".into());
        }
    } else {
        let typed: exploration::round::RoundPlan =
            serde_json::from_value(plan.clone()).map_err(|error| error.to_string())?;
        let exploration::round::RoundStrategy::Choice(config) = typed.strategy else {
            return Err("unsupported preparation plan strategy".into());
        };
        if u64::from(config.site_scope.is_some()) != receipt.guest_executions {
            return Err("choice preparation base execution count differs".into());
        }
    }
    Ok(prepared)
}

pub(crate) fn open_receipt(
    path: &Path,
    expected_sha256: &str,
) -> Result<(PreparationReceipt, Prepared), String> {
    let (root, name, _) = output_parts(path)?;
    let bytes = read_document(&root, &name)?;
    let receipt: PreparationReceipt =
        serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if canonical::digest(&bytes) != expected_sha256 || canonical::encode(&receipt)? != bytes {
        return Err("preparation receipt digest or canonical form differs".into());
    }
    let (plan_root, plan_name, _) = output_parts(&receipt.plan)?;
    let plan_bytes = read_document(&plan_root, &plan_name)?;
    let plan: Value = serde_json::from_slice(&plan_bytes).map_err(|error| error.to_string())?;
    if canonical::encode(&plan)? != plan_bytes {
        return Err("noncanonical preparation plan".into());
    }
    let opened = artifact(&receipt.campaign, &plan)?;
    let prepared = validate(&receipt, &plan, &opened)?;
    Ok((receipt, prepared))
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn provenance() -> Value {
        json!({
            "schema":"gomad.rust-wasm-provenance/v1", "build_key":"fixture",
            "module_sha256":canonical::digest(b"module"),
            "configuration":{
                "engine":"fixture", "profile":"stock", "limits":{
                    "fuel":1,"memory_bytes":1,"wall_millis":1,"output_bytes":1,"evidence_bytes":1
                },
                "arguments":["gomad-target","-test.v"],"environment":{},
                "runtime_sha256":null,"model":null,"namespace":null,"storage":null
            },
            "choice_identity":null,"scenario":null,"network":null,
            "inputs":{
                "kind":"go-test","source":"./tests","directory":"/retired/source",
                "arguments":["-test.v"],"tags":["test_dep"],
                "scenario":null,"network":null,"soft_assertion_report_only":false
            }
        })
    }

    #[test]
    fn preparation_restoration_refuses_authenticated_execution_records() {
        let provenance = provenance();
        let prepared = Prepared {
            module: b"module".to_vec(),
            configuration: serde_json::from_value(provenance["configuration"].clone()).unwrap(),
            toolchain: "a".repeat(64),
            choice_identity: None,
            scenario: None,
            provenance: canonical::encode(&provenance).unwrap(),
            network: None,
            soft_assertion_report_only: false,
        };
        let plan = json!({
            "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
            "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
            "configuration":prepared.configuration,"toolchain":prepared.toolchain,
            "module_sha256":canonical::digest(&prepared.module),
            "choice_identity":null,"scenario":null,"network":null,
            "soft_assertion_report_only":false
        });
        let store = crate::evidence::search_context::tests::store();
        let original = store
            .publish(super::super::prepared_publication(&prepared).unwrap())
            .unwrap();
        super::super::restore_from_plan(&plan, &Opened::open(&original.path).unwrap()).unwrap();
        let mut executed = super::super::prepared_publication(&prepared).unwrap();
        executed.record.outcome.class = "success".into();
        executed.record.outcome.termination = "exit".into();
        executed.record.outcome.exit_code = Some(0);
        executed
            .payloads
            .insert("record.json".into(), executed.record.bytes().unwrap());
        let executed = store.publish(executed).unwrap();
        assert!(
            super::super::restore_from_plan(&plan, &Opened::open(&executed.path).unwrap()).is_err()
        );
        std::fs::remove_dir_all(&store.root).unwrap();
    }

    #[test]
    fn retained_build_tag_tokens_preserve_go_compiler_grammar() {
        for tags in [json!(["gomad_fixture,test_dep"]), json!([""])] {
            let mut retained = provenance();
            retained["inputs"]["tags"] = tags;
            authenticate_provenance_module(&retained, b"module").unwrap();
        }
    }

    #[test]
    fn retained_source_bindings_refuse_inconsistent_or_malformed_inputs() {
        authenticate_provenance_module(&provenance(), b"module").unwrap();
        for (name, value) in [
            ("arguments", json!(["-test.list=."])),
            ("arguments", json!([1])),
            ("kind", json!("exec")),
            ("source", json!("")),
            ("directory", json!("relative/source")),
            ("tags", json!(["test_dep", "test_dep"])),
            ("tags", json!([7])),
            ("scenario", json!([1])),
            ("network", json!({})),
            ("soft_assertion_report_only", json!(null)),
        ] {
            let mut changed = provenance();
            changed["inputs"][name] = value;
            assert!(
                authenticate_provenance_module(&changed, b"module").is_err(),
                "accepted inconsistent {name}: {}",
                changed["inputs"][name]
            );
        }
    }
}
