use super::{
    build,
    evidence::{self, ComparisonMask, NativeRun, VariantRole},
    host,
    inputs::{CheckoutProof, Invocation},
    module::{self, Selection},
    namespace::{self, Common, Observation, Role, SEEDS},
    workloads::{self, Executable, RunSpec},
};
use crate::evidence::{canonical, store::open};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs,
    io::{Read, Write},
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
    time::Duration,
};
const ACTIVITY: &str = "^TestActivityTestSuite$";
const COMMON_SCHEMA: &str = "gomad.native-namespace-common/v1";

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct CommonReceipt {
    schema: String,
    parent_root: PathBuf,
    module: PathBuf,
    provenance: PathBuf,
    installation: PathBuf,
    module_sha256: String,
    provenance_sha256: String,
    choice_identity: linked::ChoiceIdentity,
    environment_identity: String,
    requests: BTreeMap<u64, Value>,
}
impl CommonReceipt {
    fn common(&self, cache: &Path) -> Common {
        let mut requests = self.requests.clone();
        for request in requests.values_mut() {
            request["compile_cache"] = json!(cache);
        }
        Common {
            requests,
            environment_identity: self.environment_identity.clone(),
            runtime_identity: self.choice_identity.toolchain_build_key.clone(),
            choice_identity: self.choice_identity.clone(),
            current_module_producer_sha256: self.provenance_sha256.clone(),
            native_role: VariantRole::DiagnosticProfile,
        }
    }
}

struct NativeRole {
    executable: Executable,
    admission: PathBuf,
    source: PathBuf,
    historical: bool,
    identity: build::NativeIdentity,
}
impl NativeRole {
    fn verify(&self, proof: &CheckoutProof) -> Result<(), String> {
        let admission = if self.historical {
            build::validate_historical_admission(&self.source, &self.admission, &build::Role::Warm)?
        } else {
            build::validate_admission(proof, &self.admission, &build::Role::Warm)?
        };
        if admission.identity != self.identity
            || admission.executable != self.executable.path
            || admission.executable_sha256 != self.executable.sha256
        {
            return Err("namespace native source/helper admission changed".into());
        }
        self.executable.verify(proof)
    }
}
fn value<'a>(call: &'a Invocation, key: &str) -> &'a str {
    call.inputs.get(key).map_or("", String::as_str)
}
fn absolute(call: &Invocation, key: &str) -> Result<PathBuf, String> {
    let value = value(call, key);
    let path = PathBuf::from(value);
    clean(&path).map_err(|error| format!("{key}: {error}"))?;
    Ok(path)
}
fn clean(path: &Path) -> Result<(), String> {
    if !path.is_absolute()
        || path == Path::new("/")
        || path.components().any(|part| {
            !matches!(
                part,
                std::path::Component::RootDir | std::path::Component::Normal(_)
            )
        })
        || path.as_os_str() != path.components().collect::<PathBuf>().as_os_str()
    {
        return Err("explicit absolute clean non-root path required".into());
    }
    Ok(())
}
fn directory(path: &Path) -> Result<(), String> {
    clean(path)?;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(path)
        .map_err(|error| error.to_string())?;
    open::pin(path).map(|_| ())
}
fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let mut data = Vec::new();
    fs::File::open(path)
        .map_err(|error| error.to_string())?
        .take(limit + 1)
        .read_to_end(&mut data)
        .map_err(|error| error.to_string())?;
    if data.len() as u64 > limit {
        return Err("namespace input capacity".into());
    }
    Ok(data)
}
fn write(path: &Path, value: &impl Serialize) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(&canonical::encode(value)?)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())
}
fn environment(call: &Invocation) -> BTreeMap<String, String> {
    call.environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect()
}
fn admission(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    label: &str,
    receipt: PathBuf,
    source: PathBuf,
    historical: bool,
) -> Result<NativeRole, String> {
    let admitted = if historical {
        build::validate_historical_admission(&source, &receipt, &build::Role::Warm)?
    } else {
        build::validate_admission(proof, &receipt, &build::Role::Warm)?
    };
    let roles = root.join("roles");
    directory(&roles)?;
    let observed = build::command_bounded(
        &roles,
        &format!("{label}-identity"),
        proof.repo(),
        &environment(call),
        &[
            admitted.executable.to_string_lossy().into_owned(),
            "--identity".into(),
        ],
        Duration::from_secs(10),
        1 << 20,
    )?;
    let identity: build::NativeIdentity =
        serde_json::from_slice(&observed).map_err(|error| error.to_string())?;
    if identity != admitted.identity {
        return Err("actual namespace helper differs from its native admission receipt".into());
    }
    write(&roles.join(format!("{label}-admission.json")), &admitted)?;
    let role = NativeRole {
        executable: Executable {
            path: admitted.executable,
            sha256: admitted.executable_sha256,
            role: build::Role::Warm,
            // Historical sources use their own closed native receipt, not a forged selected-checkout proof.
            receipt: if historical {
                None
            } else {
                Some(receipt.clone())
            },
        },
        admission: receipt,
        source,
        historical,
        identity,
    };
    role.verify(proof)?;
    Ok(role)
}
fn prepared(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    selection: Selection,
) -> Result<module::Prepared, String> {
    let mut selected = call.clone();
    let stock = value(call, "GOMAD_WASM_NAMESPACE_STOCK_GO");
    if !stock.is_empty() {
        clean(Path::new(stock))?;
        selected
            .inputs
            .insert("GOMAD_STOCK_GO".into(), stock.into());
    }
    let prepared = module::prepare(proof, &selected, root, selection)?;
    module::authenticate_prepared(&prepared)?;
    Ok(prepared)
}
fn supplied(
    call: &Invocation,
    prepared: &module::Prepared,
    module_key: &str,
    receipt_key: Option<&str>,
) -> Result<PathBuf, String> {
    let module = absolute(call, module_key)?;
    if read(&module, 512 << 20)? != prepared.guest.module {
        return Err(
            "supplied immutable namespace/operation module differs from actual preparation".into(),
        );
    }
    if let Some(key) = receipt_key {
        let receipt = absolute(call, key)?;
        let raw = read(&receipt, 16 << 20)?;
        crate::campaign::prepare::typed_provenance(
            &serde_json::from_slice(&raw).map_err(|error| error.to_string())?,
        )?;
        if raw != prepared.guest.provenance {
            return Err(
                "supplied native module receipt differs from current immutable preparation".into(),
            );
        }
    }
    Ok(module)
}
fn common(
    prepared: &module::Prepared,
    module: PathBuf,
    parent_root: &Path,
) -> Result<CommonReceipt, String> {
    let identity = prepared
        .guest
        .choice_identity
        .as_ref()
        .ok_or("namespace cooperative choice identity absent")?
        .clone();
    let mut requests = BTreeMap::new();
    for seed in SEEDS {
        let mut request = prepared.native_input(seed)?;
        request["guest"]["module"] = json!(module);
        requests.insert(seed, request);
    }
    let environment_identity =
        canonical::digest(&canonical::encode(&prepared.guest.configuration)?);
    Ok(CommonReceipt {
        schema: COMMON_SCHEMA.into(),
        parent_root: parent_root.into(),
        module,
        module_sha256: canonical::digest(&prepared.guest.module),
        provenance: prepared.provenance_path.clone(),
        provenance_sha256: canonical::digest(&prepared.guest.provenance),
        installation: prepared.installation_path.clone(),
        choice_identity: identity,
        environment_identity,
        requests,
    })
}
fn row(role: &str, seed: u64, repetition: i32, replay: bool, warm: bool) -> RunSpec {
    let mut row = RunSpec::new(
        format!(
            "seed{seed}-{role}-rep{repetition}{}",
            if replay { "-replay" } else { "" }
        ),
        ACTIVITY,
        seed,
        role,
    );
    row.repetition = repetition;
    row.warm = warm;
    row.prepare = repetition < 0;
    row
}
fn execute(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    role: &NativeRole,
    row: &RunSpec,
    input: Value,
) -> Result<NativeRun, String> {
    role.verify(proof)?;
    let result = workloads::execute(call, proof, root, row, &role.executable, input);
    role.verify(proof)?;
    let result = result?;
    if result.report["host_sha256"] != role.executable.sha256
        || result.report["linked_implementation_sha256"]
            != role.identity.linked_implementation_sha256
        || result.report["host_implementation_sha256"] != role.identity.host_implementation_sha256
        || result.report["configuration"] != role.identity.configuration
    {
        return Err("namespace raw run changed admitted helper/source configuration".into());
    }
    workloads::success(&result, row)?;
    Ok(result)
}
#[expect(
    clippy::too_many_arguments,
    reason = "A namespace sample binds its admitted role, immutable common inputs and declared seed position"
)]
fn observe(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    role: &NativeRole,
    common: &CommonReceipt,
    cache: &Path,
    label: &str,
    seed: u64,
    repetition: i32,
    replay_from: Option<&NativeRun>,
    warm: bool,
) -> Result<NativeRun, String> {
    let mut input = common
        .requests
        .get(&seed)
        .ok_or("undeclared namespace seed")?
        .clone();
    input["compile_cache"] = json!(cache);
    if let Some(recorded) = replay_from {
        workloads::replay_input(&mut input, recorded, &common.choice_identity)?;
    }
    execute(
        call,
        proof,
        root,
        role,
        &row(label, seed, repetition, replay_from.is_some(), warm),
        input,
    )
}
fn sample_summary(series: &namespace::Series) -> Value {
    json!(series.samples.iter().map(|sample|json!({"role":sample.role,"seed":sample.seed,"repetition":sample.repetition,"authorized_nanos":sample.authorized_nanos,"evidence_sha256":sample.evidence_sha256,"environment_identity":sample.environment_identity,"calls":sample.calls,"time_advances":sample.time_advances,"counter_present":sample.counter_present,"external_owner_attempts":sample.external_filesystem_calls,"counter_domain":namespace::COUNTER_DOMAIN})).collect::<Vec<_>>())
}

pub fn run(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    if proof.repo() != call.repo {
        return Err("namespace driver selected-checkout proof mismatch".into());
    }
    directory(root)?;
    let result = match call.name.as_str() {
        "namespace-activity" => activity(call, proof, root),
        "namespace-sample" => run_sample(call, proof, root),
        "operation-retention" => operations(call, proof, root),
        _ => Err("unknown native namespace/operation producer".into()),
    };
    if let Err(error) = &result {
        write(
            &root.join("namespace-failure.json"),
            &json!({"schema":"gomad.native-namespace-failure/v1","producer":call.name,"failure":error,"accepted":false,"raw_attempts":root.join("evidence")}),
        )?;
    }
    result
}
#[expect(
    clippy::too_many_lines,
    reason = "Paired namespace sampling preserves the original alternating twenty-sample order and all per-seed bounds"
)]
fn activity(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let baseline_source = absolute(call, "GOMAD_WASM_NAMESPACE_BASELINE_SOURCE")?;
    if fs::canonicalize(&baseline_source).map_err(|error| error.to_string())?
        == fs::canonicalize(proof.repo()).map_err(|error| error.to_string())?
    {
        return Err(
            "namespace baseline requires a separate immutable native source checkout".into(),
        );
    }
    let baseline_receipt = absolute(call, "GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT")?;
    let candidate_receipt = absolute(call, "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT")?;
    let baseline_cache = absolute(call, "GOMAD_WASM_NAMESPACE_BASELINE_CACHE")?;
    let candidate_cache = absolute(call, "GOMAD_WASM_NAMESPACE_CACHE")?;
    directory(&baseline_cache)?;
    directory(&candidate_cache)?;
    if fs::canonicalize(&baseline_cache).map_err(|error| error.to_string())?
        == fs::canonicalize(&candidate_cache).map_err(|error| error.to_string())?
    {
        return Err("paired namespace roles require separate actual cache directories".into());
    }
    let host = host::current(&root.join("host"), &call.environ)?;
    if !host::host_check(&host.host_os, &host.host_arch, &host.witness) {
        return Err(
            "namespace activity requires actual supported performance host evidence".into(),
        );
    }
    write(&root.join("host.json"), &host)?;
    let baseline = admission(
        call,
        proof,
        root,
        "baseline",
        baseline_receipt,
        baseline_source,
        true,
    )?;
    let candidate = admission(
        call,
        proof,
        root,
        "candidate",
        candidate_receipt,
        proof.repo().into(),
        false,
    )?;
    let prepared = prepared(call, proof, &root.join("module"), Selection::Activity)?;
    let module = supplied(
        call,
        &prepared,
        "GOMAD_WASM_NAMESPACE_MODULE",
        Some("GOMAD_WASM_NAMESPACE_MODULE_RECEIPT"),
    )?;
    let common = common(&prepared, module, root)?;
    write(&root.join("common.json"), &common)?;
    let mut before = BTreeMap::new();
    let mut after = BTreeMap::new();
    let mut order = Vec::new();
    for (seed_index, seed) in SEEDS.into_iter().enumerate() {
        let warm_before = observe(
            call,
            proof,
            root,
            &baseline,
            &common,
            &baseline_cache,
            "baseline",
            seed,
            -1,
            None,
            false,
        )?;
        let warm_after = observe(
            call,
            proof,
            root,
            &candidate,
            &common,
            &candidate_cache,
            "candidate",
            seed,
            -1,
            None,
            false,
        )?;
        evidence::compare(
            &warm_before,
            &warm_after,
            ComparisonMask::Variant {
                baseline: VariantRole::DiagnosticProfile,
                candidate: VariantRole::DiagnosticProfile,
                gc: None,
            },
        )?;
        for repetition in 0..5 {
            for turn in 0..2 {
                let candidate_turn = (turn + (seed_index * 5 + repetition) % 2) % 2 == 1;
                let (label, role, cache, warm) = if candidate_turn {
                    ("candidate", &candidate, &candidate_cache, &warm_after)
                } else {
                    ("baseline", &baseline, &baseline_cache, &warm_before)
                };
                module::authenticate_prepared(&prepared)?;
                let observed = observe(
                    call,
                    proof,
                    root,
                    role,
                    &common,
                    cache,
                    label,
                    seed,
                    i32::try_from(repetition).map_err(|_| "namespace repetition exceeds i32")?,
                    None,
                    true,
                )?;
                evidence::compare(warm, &observed, ComparisonMask::Exact)?;
                order.push(json!({"role":label,"seed":seed,"repetition":repetition,"raw":root.join("evidence").join(row(label,seed,i32::try_from(repetition).map_err(|_| "namespace repetition exceeds i32")?,false,true).name)}));
                if candidate_turn {
                    after.insert(
                        (seed, repetition),
                        Observation {
                            fresh_store_id: observed.report["fresh_store_id"]
                                .as_u64()
                                .ok_or("actual helper PID absent")?,
                            run: observed,
                            replay: None,
                        },
                    );
                } else {
                    before.insert(
                        (seed, repetition),
                        Observation {
                            fresh_store_id: observed.report["fresh_store_id"]
                                .as_u64()
                                .ok_or("actual helper PID absent")?,
                            run: observed,
                            replay: None,
                        },
                    );
                }
            }
        }
        let recorded = &before[&(seed, 0)].run;
        let replay_before = observe(
            call,
            proof,
            root,
            &baseline,
            &common,
            &baseline_cache,
            "baseline",
            seed,
            0,
            Some(recorded),
            true,
        )?;
        let replay_after = observe(
            call,
            proof,
            root,
            &candidate,
            &common,
            &candidate_cache,
            "candidate",
            seed,
            0,
            Some(recorded),
            true,
        )?;
        before
            .get_mut(&(seed, 0))
            .ok_or("baseline first repetition absent")?
            .replay = Some(replay_before);
        after
            .get_mut(&(seed, 0))
            .ok_or("candidate first repetition absent")?
            .replay = Some(replay_after);
    }
    write(&root.join("collection-order.json"), &order)?;
    let before = namespace::collect(
        common.common(&baseline_cache),
        Role::Baseline,
        |seed, repetition| {
            before
                .remove(&(seed, repetition))
                .ok_or("baseline retained sample absent".into())
        },
    )?;
    let after = namespace::collect(
        common.common(&candidate_cache),
        Role::Candidate,
        |seed, repetition| {
            after
                .remove(&(seed, repetition))
                .ok_or("candidate retained sample absent".into())
        },
    )?;
    let comparison = namespace::compare(&before, &after);
    module::authenticate_prepared(&prepared)?;
    baseline.verify(proof)?;
    candidate.verify(proof)?;
    let report = root.join("measurement.json");
    write(
        &report,
        &json!({"schema":"gomad.native-namespace-activity/v1","accepted":comparison.is_ok(),"failure":comparison.as_ref().err(),"baseline":sample_summary(&before),"candidate":sample_summary(&after),"common":root.join("common.json"),"counter_domain":namespace::COUNTER_DOMAIN,"namespace_boundary":namespace::NAMESPACE_BOUNDARY,"qualification":"native diagnostic comparison only; runtime qualification remains separate"}),
    )?;
    comparison?;
    Ok(report)
}

fn load_prepared(common: &CommonReceipt) -> Result<module::Prepared, String> {
    if common.schema != COMMON_SCHEMA
        || canonical::digest(&read(&common.provenance, 16 << 20)?) != common.provenance_sha256
    {
        return Err("native namespace common/provenance binding changed".into());
    }
    let raw = read(&common.provenance, 16 << 20)?;
    let provenance: Value = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    crate::campaign::prepare::typed_provenance(&provenance)?;
    let module = read(&common.module, 512 << 20)?;
    if canonical::digest(&module) != common.module_sha256 {
        return Err("native namespace common module bytes changed".into());
    }
    let prepared = module::Prepared {
        guest: crate::campaign::prepare::Prepared {
            auxiliary: BTreeMap::new(),
            compile_cache_root: crate::campaign::prepare::compile_cache_root(
                &crate::install::manifest::resolve(Some(&common.installation))?.manifest,
            )?,
            module,
            configuration: serde_json::from_value(provenance["configuration"].clone())
                .map_err(|error| error.to_string())?,
            toolchain: provenance["build_key"]
                .as_str()
                .ok_or("native module build key absent")?
                .into(),
            choice_identity: serde_json::from_value(provenance["choice_identity"].clone())
                .map_err(|error| error.to_string())?,
            scenario: None,
            network: serde_json::from_value(
                provenance.get("network").cloned().unwrap_or(Value::Null),
            )
            .map_err(|error| error.to_string())?,
            soft_assertion_report_only: provenance["inputs"]["soft_assertion_report_only"]
                .as_bool()
                .ok_or("native assertion policy absent")?,
            provenance: raw,
        },
        module_path: common.module.clone(),
        provenance_path: common.provenance.clone(),
        installation_path: common.installation.clone(),
        module_metadata: provenance["module_metadata"].clone(),
        preparation_arguments: Vec::new(),
        scenario_path: None,
    };
    module::authenticate_prepared(&prepared)?;
    if canonical::encode(
        prepared
            .guest
            .choice_identity
            .as_ref()
            .ok_or("common native choice identity absent")?,
    )? != canonical::encode(&common.choice_identity)?
    {
        return Err("common native choice identity changed".into());
    }
    Ok(prepared)
}
fn parent_lease(root: &Path) -> Result<(), String> {
    clean(root)?;
    let canonical = fs::canonicalize(root).map_err(|error| error.to_string())?;
    if canonical != root {
        return Err("namespace parent root must be canonical".into());
    }
    let pinned = open::pin(root)?;
    if open::read(&pinned, ".pin", 8192, false)? != root.as_os_str().as_encoded_bytes() {
        return Err("namespace parent run identity changed".into());
    }
    let lease = open::child(&pinned, ".lease", false, false)?;
    match rustix::fs::flock(&lease, rustix::fs::FlockOperation::NonBlockingLockShared) {
        Err(rustix::io::Errno::WOULDBLOCK) => Ok(()),
        Ok(()) => {
            Err("namespace sample requires its active parent exclusive retention lease".into())
        }
        Err(error) => Err(error.to_string()),
    }
}

/// Collects one source-admitted native sample under its active parent's retention lease.
/// # Errors
/// Refuses standalone roots, old Go receipts/tapes and changed immutable common inputs.
#[expect(
    clippy::too_many_lines,
    reason = "A child sample keeps parent retention, source admission and replay validation in one lease scope"
)]
pub fn run_sample(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
) -> Result<PathBuf, String> {
    let _lease = super::retention::ChildLease::for_path(root)?
        .ok_or("namespace sample requires its authenticated parent child lease")?;
    let common_path = absolute(call, "GOMAD_WASM_NAMESPACE_COMMON")?;
    let common: CommonReceipt = serde_json::from_slice(&read(&common_path, 32 << 20)?)
        .map_err(|error| error.to_string())?;
    parent_lease(&common.parent_root)?;
    if !root.starts_with(&common.parent_root)
        || root == common.parent_root
        || !common_path.starts_with(&common.parent_root)
    {
        return Err(
            "namespace sample output/common must remain under the authenticated parent run".into(),
        );
    }
    if root != absolute(call, "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT")? {
        return Err("namespace sample output root differs from requested sample root".into());
    }
    directory(root)?;
    let prepared = load_prepared(&common)?;
    let label = value(call, "GOMAD_WASM_NAMESPACE_ROLE");
    if !matches!(label, "baseline" | "candidate") {
        return Err("namespace sample requires explicit baseline/candidate role".into());
    }
    let seed: u64 = value(call, "GOMAD_WASM_NAMESPACE_SEED")
        .parse()
        .map_err(|_| "namespace sample seed absent")?;
    if !SEEDS.contains(&seed) {
        return Err("namespace sample seed outside frozen cohort".into());
    }
    let repetition: i32 = value(call, "GOMAD_WASM_NAMESPACE_REPETITION")
        .parse()
        .map_err(|_| "namespace sample repetition absent")?;
    if !(-1..=4).contains(&repetition) {
        return Err("namespace sample requires warmup -1 or repetition 0..4".into());
    }
    let receipt = absolute(call, "GOMAD_WASM_NAMESPACE_HELPER_RECEIPT")?;
    let claimed: build::Admission =
        serde_json::from_slice(&read(&receipt, 16 << 20)?).map_err(|error| error.to_string())?;
    let build_receipt: build::BuildReceipt =
        serde_json::from_slice(&read(&claimed.build_receipt, 16 << 20)?)
            .map_err(|error| error.to_string())?;
    let source = build_receipt
        .crate_root
        .ancestors()
        .nth(4)
        .ok_or("native namespace receipt source root absent")?
        .to_owned();
    if label == "candidate" && source != proof.repo() {
        return Err("namespace candidate receipt is not source-matched".into());
    }
    let historical = source != proof.repo();
    if label == "baseline" && !historical {
        return Err("namespace baseline requires separate admitted native source".into());
    }
    let role = admission(call, proof, root, label, receipt, source, historical)?;
    let cache = absolute(call, "GOMAD_WASM_NAMESPACE_CACHE")?;
    directory(&cache)?;
    let replay = value(call, "GOMAD_WASM_NAMESPACE_REPLAY_TAPE");
    let recorded = if replay.is_empty() {
        None
    } else {
        let path = Path::new(replay);
        clean(path)?;
        let recorded = evidence::read_profile_run(path, evidence::ReadLimits::default())?;
        if recorded.report["host_sha256"] != role.executable.sha256 {
            return Err(
                "namespace replay requires a retained native source run from this admitted helper"
                    .into(),
            );
        }
        Some(recorded)
    };
    let run = observe(
        call,
        proof,
        root,
        &role,
        &common,
        &cache,
        label,
        seed,
        repetition,
        recorded.as_ref(),
        repetition >= 0,
    )?;
    if let Some(recorded) = &recorded {
        evidence::compare_replay(recorded, &run, &common.choice_identity)?;
    }
    namespace::operation_retention(&run)?;
    let counter = &run.report["native_namespace_counter"];
    if counter["schema"] != "gomad.native-namespace-counter/v1"
        || counter["counter_domain"] != namespace::COUNTER_DOMAIN
        || counter["namespace_boundary"] != namespace::NAMESPACE_BOUNDARY
        || counter["implementation_sha256"] != role.identity.linked_implementation_sha256
        || label == "candidate"
            && counter["external_owner_calls"]
                .as_object()
                .is_none_or(|calls| !calls.is_empty())
    {
        return Err(
            "namespace source sample lacks its actual source-bound counter observation".into(),
        );
    }
    parent_lease(&common.parent_root)?;
    module::authenticate_prepared(&prepared)?;
    role.verify(proof)?;
    let output = root.join("sample.json");
    write(
        &output,
        &json!({"schema":"gomad.native-namespace-sample/v1","role":label,"seed":seed,"repetition":repetition,"replay":recorded.is_some(),"common_sha256":canonical::digest(&read(&common_path,32<<20)?),"raw_root":root.join("evidence").join(row(label,seed,repetition,recorded.is_some(),repetition>=0).name),"counter":counter,"authorization_to_terminal_nanos":run.report["authorization_to_terminal_nanos"],"source":role.source,"helper_sha256":role.executable.sha256}),
    )?;
    let index = value(call, "GOMAD_WASM_NAMESPACE_SAMPLE_INDEX");
    if !index.is_empty() {
        let index = Path::new(index);
        clean(index)?;
        if !index.starts_with(&common.parent_root) {
            return Err("namespace sample index must remain under its parent lease".into());
        }
        write(
            index,
            &json!({"receipt":output,"sha256":canonical::digest(&read(&output,16<<20)?)}),
        )?;
    }
    Ok(output)
}

fn digest_retention(control: &NativeRun, retained: &NativeRun) -> Result<(), String> {
    let transcript = retained
        .root
        .streams
        .get("transcript")
        .ok_or("operation retention actual canonical transcript absent")?;
    namespace::operation_retention(retained)?;
    if control.request["guest"]["model"]["configuration"]["digest_only"] != true
        || retained.request["guest"]["model"]["configuration"]["digest_only"] != false
        || control.root.streams.contains_key("transcript")
        || control.root.outcome["local_model"]["transcript_sha256"] != canonical::digest(transcript)
        || control.root.outcome["local_model"]["transcript_bytes"] != transcript.len() as u64
        || control.root.outcome["local_model"]["transcript_sha256"]
            != retained.root.outcome["local_model"]["transcript_sha256"]
        || control.root.outcome["local_model"]["transcript_bytes"]
            != retained.root.outcome["local_model"]["transcript_bytes"]
    {
        return Err(
            "native digest control does not bind the exact retained canonical transcript".into(),
        );
    }
    let mut projected = retained.clone();
    projected.root.streams.remove("transcript");
    projected.request["guest"]["model"]["configuration"]["digest_only"] = true.into();
    evidence::compare(control, &projected, ComparisonMask::Exact)
}
#[expect(
    clippy::too_many_lines,
    reason = "Operation retention compares the original digest, full-transcript and replay variants before accepting output"
)]
fn operations(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let controls = absolute(call, "GOMAD_WASM_OPERATION_CONTROL_ROOT")?;
    let cache = absolute(call, "GOMAD_WASM_MATCHED_CACHE")?;
    directory(&cache)?;
    let executable = workloads::admit(call, proof, root, "diagnostic")?;
    let identity = value(call, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID");
    if identity.is_empty() {
        return Err("operation retention requires explicit native runtime identity".into());
    }
    let mut rows = Vec::new();
    for (label, selection) in [
        (
            "workflow",
            "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
        ),
        ("child", "^TestChildWorkflowSuite$"),
        ("timers", "^TestUserTimersTestSuite$"),
    ] {
        let prepared = prepared(
            call,
            proof,
            &root.join(format!("module-{label}")),
            Selection::Selected(selection.into()),
        )?;
        let module = supplied(call, &prepared, "GOMAD_WASM_MATCHED_MODULE", None)?;
        if prepared
            .guest
            .choice_identity
            .as_ref()
            .ok_or("native operation choice identity absent")?
            .toolchain_build_key
            != identity
        {
            return Err(
                "operation retention native runtime identity differs from actual prepared module"
                    .into(),
            );
        }
        for seed in [0, 1_000_003] {
            module::authenticate_prepared(&prepared)?;
            let mut input = prepared.native_input(seed)?;
            input["guest"]["module"] = json!(module);
            input["compile_cache"] = json!(cache);
            let mut prime = RunSpec::new(
                format!("{label}-prime-seed{seed}"),
                selection,
                seed,
                "diagnostic",
            );
            prime.prepare = true;
            prime.warm = false;
            let observed =
                workloads::execute(call, proof, root, &prime, &executable, input.clone())?;
            workloads::success(&observed, &prime)?;
            let mut recorded = Vec::new();
            for (owner, digest_only) in [("control", true), ("retained", false)] {
                let row = RunSpec::new(
                    format!("{label}-{owner}-seed{seed}"),
                    selection,
                    seed,
                    "diagnostic",
                );
                let mut input = input.clone();
                input["guest"]["model"]["configuration"]["digest_only"] = digest_only.into();
                let observed = workloads::execute(call, proof, root, &row, &executable, input)?;
                workloads::success(&observed, &row)?;
                recorded.push(observed);
            }
            digest_retention(&recorded[0], &recorded[1])?;
            let reference_root = controls.join(format!("{label}-reference-seed{seed}"));
            let reference =
                evidence::read_profile_run(&reference_root, evidence::ReadLimits::default())?;
            for field in [
                "host_sha256",
                "host_implementation_sha256",
                "linked_implementation_sha256",
                "configuration",
                "features",
            ] {
                if reference.report[field] != recorded[0].report[field] {
                    return Err("operation native reference changed actual helper/model/configuration identity".into());
                }
            }
            evidence::compare(
                &reference,
                &recorded[0],
                ComparisonMask::Variant {
                    baseline: VariantRole::DiagnosticProfile,
                    candidate: VariantRole::DiagnosticProfile,
                    gc: None,
                },
            )?;
            let counts = namespace::operation_retention(&recorded[1])?;
            let mut names = BTreeMap::<String, u64>::new();
            for (key, count) in &counts.operations {
                let name = key
                    .split('/')
                    .nth(1)
                    .ok_or("invalid native counted operation")?;
                let total = names.entry(name.into()).or_default();
                *total = total
                    .checked_add(*count)
                    .ok_or("operation count overflow")?;
            }
            for (name, count) in names {
                if recorded[1].profile["abi_ops"][format!("wasi_snapshot_preview1.{name}")]["count"]
                    .as_u64()
                    .is_none_or(|observed| observed < count)
                {
                    return Err(
                        "actual native operation profile lost retained canonical operation calls"
                            .into(),
                    );
                }
            }
            let result = json!({"suite":label,"seed":seed,"calls":counts.calls,"local":counts.local,"canonical_nonlocal_categories":counts.remote,"operations":counts.operations,"counter_domain":"canonical operation categories; not completed external callback calls","raw_reference":reference_root,"raw_control":root.join("evidence").join(format!("{label}-control-seed{seed}")),"raw_retained":root.join("evidence").join(format!("{label}-retained-seed{seed}"))});
            write(
                &root.join(format!("{label}-seed{seed}-operation-counts.json")),
                &result,
            )?;
            rows.push(result);
        }
        module::authenticate_prepared(&prepared)?;
    }
    executable.verify(proof)?;
    let output = root.join("operation-retention.json");
    write(
        &output,
        &json!({"schema":"gomad.native-operation-retention/v1","accepted":true,"runs":rows,"boundary":"actual native digest/retained canonical bytes and role-bound native references; no historical Go public-only projection restored"}),
    )?;
    Ok(output)
}

#[cfg(test)]
#[path = "namespace_driver_tests.rs"]
mod tests;
