use super::{
    inputs::{CheckoutProof, Invocation},
    module,
};
use crate::evidence::canonical;
use serde_json::{Value, json};
use std::{
    fs,
    path::{Path, PathBuf},
};

/// # Errors
/// Refuses a malformed original cluster scenario or changed lifecycle declaration.
pub fn scenario_bytes() -> Result<Vec<u8>, String> {
    let raw = include_str!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../internal/backends/wasm/command/testdata/scenario/network.json"
    ))
    .replace("\"a\"", "\"active\"")
    .replace("\"b\"", "\"standby\"");
    let mut scenario: Value = serde_json::from_str(&raw).map_err(|error| error.to_string())?;
    scenario["actions"] = json!(
        [("start", 1), ("crash", 1), ("restart", 2), ("wait", 2)]
            .into_iter()
            .flat_map(|(action, incarnation)| ["active", "standby"].map(
                |node| json!({"action":action,"handle":{"node":node,"incarnation":incarnation}})
            ))
            .collect::<Vec<_>>()
    );
    scenario["output_crashes"] = json!(["active","standby"].map(|node|json!({"handle":{"node":node,"incarnation":1},"marker":b"GOMAD_TEMPORAL_OBSERVATION "})));
    scenario["spec"]["limits"] = crate::stress::cluster::original_limits();
    let arguments = json!([
        "-test.v",
        "-test.count=1",
        "-test.timeout=1h",
        "-test.run=^TestIndependentClusterReplicationRecovery$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite"
    ]);
    scenario["boots"] = json!(["active","standby"].into_iter().flat_map(|node| ["commit","reopen"].map(|phase| json!({"id":format!("{node}-{phase}"),"args":arguments,"storage":{"volumes":[{"ID":node,"Path":"/volume","CapacityBytes":256 << 20}],"limits":{"PendingOperations":100_000,"Transitions":10_000_000},"sqlite_lock_databases":["/volume/temporal.db"]}}))).collect::<Vec<_>>());
    scenario["spec"]["nodes"] = json!([("active","10.0.0.1"),("standby","10.0.0.2")].map(|(node,address)| json!({"id":node,"boot":format!("{node}-commit"),"address":address,"volumes":[{"volume":node,"path":"/volume"}]})));
    scenario["spec"]["volumes"] =
        json!(["active", "standby"].map(|node| json!({"id":node,"capacity_bytes":256 << 20})));
    scenario["incarnation_boots"] = json!(["active", "standby"].map(
        |node| json!({"handle":{"node":node,"incarnation":2},"boot":format!("{node}-reopen")})
    ));
    scenario["network_faults"] = json!([("partition",1_u64),("heal",30_000_000_000)].map(|(action,at)| json!({"Name":format!("temporal-active-to-standby-{action}"),"Action":action,"From":"active","To":"standby","AtNanos":at,"Symmetric":false})));
    crate::campaign::scenario::normalize(&canonical::encode(&scenario)?)
}

/// # Errors
/// Refuses changed fixture inputs, compiler admission or guest preparation.
pub fn prepare(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
) -> Result<module::Prepared, String> {
    let source = stage_broker(proof.repo())?;
    module::prepare_scenario(proof, call, root, &source, scenario_bytes()?)
}

pub(super) fn stage_broker(repo: &Path) -> Result<String, String> {
    let sources = [
        (
            "recovery_test.go",
            "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go",
        ),
        (
            "replication_test.go",
            "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go",
        ),
    ];
    let bodies = sources
        .iter()
        .map(|(_, path)| read(&repo.join(path), 8 << 20))
        .collect::<Result<Vec<_>, _>>()?;
    let combined: Vec<_> = bodies.iter().flatten().copied().collect();
    let relative = PathBuf::from(".tmp/wasm-backend-temporal").join(format!(
        "temporal_broker-{}",
        &canonical::digest(&combined)[..16]
    ));
    let destination = repo.join(&relative);
    fs::create_dir_all(&destination).map_err(|error| error.to_string())?;
    for entry in fs::read_dir(&destination).map_err(|error| error.to_string())? {
        let entry = entry.map_err(|error| error.to_string())?;
        if !entry
            .file_type()
            .map_err(|error| error.to_string())?
            .is_file()
            || !sources.iter().any(|(name, _)| entry.file_name() == *name)
        {
            return Err("unexpected staged broker source entry".into());
        }
    }
    for ((name, _), body) in sources.iter().zip(bodies) {
        let path = destination.join(name);
        if path.exists() {
            if read(&path, 8 << 20)? != body {
                return Err("staged broker source differs from immutable original".into());
            }
        } else {
            write(&path, &body)?;
        }
    }
    Ok(format!("./{}", relative.to_string_lossy()))
}

fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    use std::io::Read;
    let mut bytes = Vec::new();
    fs::File::open(path)
        .map_err(|error| error.to_string())?
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > limit {
        return Err("cluster evidence input exceeds bound".into());
    }
    Ok(bytes)
}
fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    use std::{io::Write, os::unix::fs::OpenOptionsExt};
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())
}

/// # Errors
/// Refuses changed module bytes or invalid retained per-incarnation configuration.
pub fn native_input(prepared: &module::Prepared, seed: u64) -> Result<Value, String> {
    module::authenticate_prepared(prepared)?;
    let identity = prepared
        .guest
        .choice_identity
        .as_ref()
        .ok_or("cluster choice identity absent")?;
    let mut bound = crate::campaign::scenario::bind(
        prepared
            .guest
            .scenario
            .as_deref()
            .ok_or("cluster scenario absent")?,
        &prepared.guest.configuration,
        &prepared.guest.module,
        identity,
        prepared.guest.network.as_ref(),
        seed,
    )?;
    crate::stress::cluster::bind(&mut bound)?;
    let incarnations = bound.request.incarnations.into_iter().map(|incarnation| -> Result<Value,String> {
        let guest = incarnation.request;
        let retained = bound.configuration.incarnations.iter().find(|entry| entry.handle == incarnation.handle).ok_or("bound incarnation configuration absent")?;
        let model = guest.model.map(|model| json!({"configuration":retained.model,"arguments":model.arguments,"environment":model.environment,"replay_trace":model.replay_trace}));
        Ok(json!({"handle":incarnation.handle,"network":incarnation.network,"guest":{"module":prepared.module_path,
            "module_sha256":guest.module_sha256,"limits":{"memory_bytes":guest.limits.memory_bytes,"fuel":guest.limits.fuel,"output_bytes":guest.limits.output_bytes,
            "evidence_bytes":guest.limits.evidence_bytes,"wall_millis":guest.limits.wall_timeout.as_millis()},"model":model,"namespace":guest.namespace,"storage":guest.storage}}))
    }).collect::<Result<Vec<_>,String>>()?;
    Ok(
        json!({"kind":"cluster","nodes":bound.request.nodes,"broker":bound.request.broker,"network":bound.request.network,
        "incarnations":incarnations,"steps":bound.request.steps,"boundaries":bound.request.boundaries,"crash_groups":bound.request.crash_groups,
        "wall_millis":bound.request.wall_timeout.as_millis(),"evidence_bytes":bound.request.evidence_bytes}),
    )
}

fn input<'a>(call: &'a Invocation, name: &str) -> &'a str {
    call.inputs.get(name).map_or("", String::as_str)
}

fn role(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    reference: bool,
) -> Result<super::workloads::Executable, String> {
    let diagnostic = call.name != "matched-release-candidate" && !reference;
    let name = if reference {
        "GOMAD_WASM_CLUSTER_OWNER_REFERENCE_HELPER"
    } else {
        "GOMAD_WASM_TEST_HELPER"
    };
    let receipt = input(
        call,
        if diagnostic {
            "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"
        } else {
            "GOMAD_WASM_CLUSTER_HELPER_ADMISSION"
        },
    );
    super::workloads::admit_executable(
        call,
        proof,
        root,
        if reference { "reference" } else { "candidate" },
        PathBuf::from(input(call, name)),
        if diagnostic {
            super::build::Role::Warm
        } else {
            super::build::Role::Release
        },
        (!receipt.is_empty()).then(|| PathBuf::from(receipt)),
    )
}

fn success(run: &super::evidence::NativeRun) -> Result<(), String> {
    if run.root.outcome["termination"] != "exit"
        || run.root.outcome["exit_code"] != 0
        || run.incarnations.len() != 4
    {
        return Err(
            "two-cluster scenario failed or did not retain all four actual incarnations".into(),
        );
    }
    let mut seen = std::collections::BTreeSet::new();
    let mut initial = std::collections::BTreeMap::new();
    let mut observations = std::collections::BTreeMap::new();
    let mut history = None;
    for incarnation in &run.incarnations {
        let handle = incarnation
            .handle
            .as_ref()
            .ok_or("cluster handle missing")?;
        let node = handle["node"].as_str().ok_or("cluster node absent")?;
        let generation = handle["incarnation"]
            .as_u64()
            .ok_or("cluster incarnation absent")?;
        if !["active", "standby"].contains(&node)
            || ![1, 2].contains(&generation)
            || !seen.insert((node, generation))
        {
            return Err("cluster actual incarnation membership differs".into());
        }
        let value = crate::stress::cluster::observation(
            incarnation
                .streams
                .get("stdout")
                .ok_or("cluster stdout absent")?,
        )?;
        if generation == 1 {
            if incarnation.outcome["termination"] != "crash"
                || !incarnation.streams.contains_key("storage_image")
            {
                return Err(
                    "committed cluster incarnation did not retain crashed durable state".into(),
                );
            }
            if history
                .as_ref()
                .is_some_and(|expected| expected != &value["history"])
                || initial
                    .values()
                    .any(|previous: &Value| previous["cluster_id"] == value["cluster_id"])
            {
                return Err(
                    "independent histories did not converge or cluster identities coincide".into(),
                );
            }
            history = Some(value["history"].clone());
            initial.insert(node, value.clone());
        } else if incarnation.outcome["termination"] != "exit"
            || incarnation.outcome["exit_code"] != 0
        {
            return Err("cluster reopened guest did not complete".into());
        }
        observations.insert((node, generation), value);
    }
    if initial.len() != 2 {
        return Err("missing independent original cluster commits".into());
    }
    for (node, original) in initial {
        let value = observations
            .get(&(node, 2))
            .ok_or("missing reopened cluster observation")?;
        crate::stress::cluster::history_oracle(&original, value)?;
        for field in ["history", "cluster_id", "event_ids", "types"] {
            if value[field] != original[field] {
                return Err("committed cluster history or identity changed after crash".into());
            }
        }
    }
    Ok(())
}

/// # Errors
/// Refuses source or role admission, failed guest oracles and divergent retained replay.
pub fn run(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let result = run_inner(call, proof, root);
    if let Err(error) = &result {
        let _ = write(&root.join("failure-report.json"), &canonical::encode(&json!({"schema":"gomad.native-cluster-failure/v1","producer":call.name,"error":error,"runtime_status":"actual failed observations retained"})).unwrap_or_default());
    }
    result
}

#[expect(
    clippy::too_many_lines,
    reason = "Original cluster producer ordering binds each role and repetition to its replay before publication"
)]
fn run_inner(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    use super::{
        evidence::{self, ComparisonMask, VariantRole},
        workloads::{self, RunSpec},
    };
    if ![
        "cluster-owner-profile",
        "matched-release-candidate",
        "diagnostic-cluster-baseline",
    ]
    .contains(&call.name.as_str())
    {
        return Err("unsupported native cluster producer".into());
    }
    super::host::current(root, &call.environ)?;
    super::host::source_snapshot(proof, root, &call.environ)?;
    let mut seeds = vec![7, 19, 42];
    let mut repetitions = 3;
    if call.name == "cluster-owner-profile" {
        let raw = input(call, "GOMAD_WASM_CLUSTER_OWNER_SEEDS");
        if !raw.is_empty() {
            seeds = raw
                .split(',')
                .map(|seed| {
                    seed.parse::<u64>()
                        .map_err(|_| "invalid cluster seed".to_string())
                })
                .collect::<Result<_, _>>()?;
        }
        let raw = input(call, "GOMAD_WASM_CLUSTER_OWNER_REPETITIONS");
        if !raw.is_empty() {
            repetitions = raw
                .parse::<i32>()
                .map_err(|_| "invalid cluster repetitions")?;
        }
        if seeds.is_empty() || !(1..=20).contains(&repetitions) {
            return Err("cluster seeds/repetitions outside original bounds".into());
        }
    }
    let candidate = role(call, proof, root, false)?;
    let reference = if call.name == "cluster-owner-profile" {
        Some(role(call, proof, root, true)?)
    } else {
        None
    };
    if reference.as_ref().is_some_and(|reference| {
        reference.path == candidate.path || reference.sha256 == candidate.sha256
    }) {
        return Err("cluster owner roles must be distinct executables".into());
    }
    let prepared = prepare(call, proof, &root.join("preparation"))?;
    let expected = input(call, "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256");
    if !expected.is_empty() && expected != canonical::digest(&prepared.guest.module) {
        return Err("cluster expected module digest differs from actual preparation".into());
    }
    let mut rows = Vec::new();
    for seed in seeds {
        let reference_run = if let Some(reference) = &reference {
            let row = RunSpec::new(
                format!("seed{seed}-reference"),
                "cluster",
                seed,
                "reference",
            );
            let result = execute_replayed(
                call,
                proof,
                root,
                &row,
                reference,
                &native_input(&prepared, seed)?,
                prepared
                    .guest
                    .choice_identity
                    .as_ref()
                    .ok_or("cluster replay choice identity absent")?,
            )?;
            Some(result)
        } else {
            None
        };
        let count = if call.name == "diagnostic-cluster-baseline" {
            1
        } else {
            repetitions + 1
        };
        let mut baseline = None;
        for repetition in 0..count {
            let mut row = RunSpec::new(
                format!("seed{seed}-rep{repetition}"),
                "cluster",
                seed,
                "candidate",
            );
            row.repetition = repetition - 1;
            row.warm = repetition > 0;
            let mut request = native_input(&prepared, seed)?;
            let cache = root.join(format!("seed{seed}-compile-cache"));
            workloads::cache_directory(&cache)?;
            request["compile_cache"] = json!(cache);
            let result = execute_replayed(
                call,
                proof,
                root,
                &row,
                &candidate,
                &request,
                prepared
                    .guest
                    .choice_identity
                    .as_ref()
                    .ok_or("cluster replay choice identity absent")?,
            )?;
            if let Some(reference) = &reference_run {
                evidence::compare(
                    reference,
                    &result,
                    ComparisonMask::Variant {
                        baseline: VariantRole::Ordinary,
                        candidate: VariantRole::DiagnosticProfile,
                        gc: None,
                    },
                )?;
            }
            if let Some(baseline) = &baseline {
                evidence::compare(baseline, &result, ComparisonMask::Exact)?;
            }
            let mut witness = Vec::new();
            for incarnation in &result.incarnations {
                witness.extend(
                    incarnation
                        .streams
                        .get("choice")
                        .ok_or("cluster choice bytes absent")?,
                );
            }
            rows.push(json!({"seed":seed,"repetition":repetition,"warmup":repetition==0,"request":result.request,"report":result.report,"profile":result.profile,"choice_witness_sha256":canonical::digest(&witness)}));
            baseline = Some(result);
        }
    }
    module::authenticate_prepared(&prepared)?;
    candidate.verify(proof)?;
    if let Some(reference) = reference {
        reference.verify(proof)?;
    }
    let path = root.join("cluster-report.json");
    write(
        &path,
        &canonical::encode(
            &json!({"schema":"gomad.native-cluster-measurement/v1","producer":call.name,"module":prepared.module_path,"provenance":prepared.provenance_path,"scenario":prepared.scenario_path,"runs":rows,"passed":true}),
        )?,
    )?;
    Ok(path)
}

/// # Errors
/// Refuses missing incarnations, invalid traces or unprojectable helper-owned tapes.
pub fn replay_input(
    input: &mut Value,
    recorded: &super::evidence::NativeRun,
    identity: &gomad_wasmhost::linked::ChoiceIdentity,
) -> Result<(), String> {
    use gomad_wasmhost::linked;
    let incarnations = input["incarnations"]
        .as_array_mut()
        .ok_or("cluster replay incarnation declarations absent")?;
    if incarnations.len() != recorded.incarnations.len() {
        return Err("cluster replay incarnation membership differs".into());
    }
    for incarnation in incarnations {
        let retained = recorded
            .incarnations
            .iter()
            .find(|retained| retained.handle.as_ref() == Some(&incarnation["handle"]))
            .ok_or("cluster replay recorded handle absent")?;
        let trace = retained
            .streams
            .get("choice")
            .ok_or("cluster replay actual choice bytes absent")?;
        let plan = linked::project_choices(trace, identity).map_err(str::to_owned)?;
        let interrupted = matches!(
            retained.outcome["termination"].as_str(),
            Some("crash" | "storage_crash" | "stop")
        );
        let replay = json!({"schema":"gomad.replay-trace/v1","execution_identity":identity,"trace_sha256":canonical::digest(trace),"tape_sha256":canonical::digest(&plan.bytes),"decisions":plan.decisions.len(),"trace_bytes":trace.len(),"trace_frames":trace.len().div_ceil(65536),"prefix_control":if interrupted {Some(json!({"cutoff":plan.decisions.len(),"override":null}))} else {None}});
        incarnation["guest"]["model"]["configuration"]["mode"] =
            json!(if interrupted { 3 } else { 2 });
        incarnation["guest"]["model"]["configuration"]["replay"] = replay;
        incarnation["guest"]["model"]["replay_trace"] = json!(trace);
    }
    Ok(())
}

fn execute_replayed(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    row: &super::workloads::RunSpec,
    executable: &super::workloads::Executable,
    request: &Value,
    identity: &gomad_wasmhost::linked::ChoiceIdentity,
) -> Result<super::evidence::NativeRun, String> {
    let result = super::workloads::execute(call, proof, root, row, executable, request)?;
    success(&result)?;
    let mut request = result.request.clone();
    replay_input(&mut request, &result, identity)?;
    let mut replay_row = row.clone();
    replay_row.name.push_str("-replay");
    let replay = super::workloads::execute(call, proof, root, &replay_row, executable, &request)?;
    success(&replay)?;
    compare_replay(&result, &replay, identity)?;
    Ok(result)
}

fn compare_replay(
    before: &super::evidence::NativeRun,
    after: &super::evidence::NativeRun,
    identity: &gomad_wasmhost::linked::ChoiceIdentity,
) -> Result<(), String> {
    if before.incarnations.len() != after.incarnations.len()
        || before.request["incarnations"].as_array().map(Vec::len)
            != Some(before.incarnations.len())
    {
        return Err("cluster replay changed actual incarnation membership".into());
    }
    let mut normalized = after.clone();
    let declarations = normalized.request["incarnations"]
        .as_array_mut()
        .ok_or("cluster replay declarations absent")?;
    for (index, original) in before.incarnations.iter().enumerate() {
        let handle = original
            .handle
            .as_ref()
            .ok_or("cluster replay handle absent")?;
        let actual_index = after
            .incarnations
            .iter()
            .position(|value| value.handle.as_ref() == Some(handle))
            .ok_or("cluster replay handle changed")?;
        let original_request = before.request["incarnations"]
            .as_array()
            .unwrap()
            .iter()
            .find(|value| &value["handle"] == handle)
            .ok_or("cluster original declaration absent")?;
        let declaration = declarations
            .iter_mut()
            .find(|value| &value["handle"] == handle)
            .ok_or("cluster replay declaration changed")?;
        let mut left = before.clone();
        let mut right = after.clone();
        left.request = json!({"kind":"standalone","guest":original_request["guest"]});
        right.request = json!({"kind":"standalone","guest":declaration["guest"]});
        left.root = original.clone();
        right.root = after.incarnations[actual_index].clone();
        left.incarnations.clear();
        right.incarnations.clear();
        left.executed_steps = None;
        right.executed_steps = None;
        if matches!(
            original.outcome["termination"].as_str(),
            Some("crash" | "storage_crash" | "stop")
        ) {
            super::evidence::compare_prefix_replay(&left, &right, identity)?;
        } else {
            super::evidence::compare_replay(&left, &right, identity)?;
        }
        declaration["guest"] = original_request["guest"].clone();
        for field in ["cursor", "tape_records", "tape_sha256"] {
            normalized.incarnations[actual_index].outcome["local_model"][field] =
                original.outcome["local_model"][field].clone();
        }
        if actual_index != index {
            return Err("cluster replay changed retained incarnation order".into());
        }
    }
    super::evidence::compare(before, &normalized, super::evidence::ComparisonMask::Exact)
}

#[cfg(test)]
#[path = "cluster_driver_tests.rs"]
mod tests;
