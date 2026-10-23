mod recording;

use super::{
    BTreeMap, BTreeSet, Candidate, Configuration, FileBinding, MAXIMUM_LIST, Opened, Path, PathBuf,
    Value, absolute, bound_file, canonical, digest, inventory, json, open, read, source, spec,
};

pub(super) fn validate(path: &Path, sha256: &str, candidate: &Candidate) -> Result<(), String> {
    let bytes = bound_file(
        &FileBinding {
            path: path.to_path_buf(),
            sha256: sha256.into(),
        },
        MAXIMUM_LIST,
    )?;
    let value: Value = spec::strict(&bytes)?;
    let string = |value: &Value| {
        value
            .as_str()
            .map(str::to_owned)
            .ok_or_else(|| "inspection receipt field missing".to_owned())
    };
    if value["schema"] != "gomad.rust-temporal-inspection/v1"
        || value["owner"] != "onebox"
        || value["binary_sha256"] != candidate.executable.sha256
        || value["subgates"]
            != json!([{"id":"TestTemporalInspectionRetainedFailure/onebox","status":"PASS","executed":true,"exit":0}])
    {
        return Err(
            "support inspection lacks executed original onebox subgate or candidate binding".into(),
        );
    }
    for field in ["source_detached", "private_inputs_detached"] {
        let detached = PathBuf::from(string(&value[field])?);
        absolute(&detached)?;
        let original = detached
            .to_str()
            .and_then(|text| text.strip_suffix("-unavailable"))
            .ok_or("inspection detached input reference invalid")?;
        if Path::new(original).exists() {
            return Err("inspection source/compiler/runtime inputs were restored".into());
        }
    }
    let artifact = PathBuf::from(string(&value["artifact"])?);
    let opened = Opened::open(&artifact)?;
    opened.authenticate_all()?;
    let cluster_inputs = crate::replay::cluster::inputs(&opened)?;
    crate::replay::cluster::bind(&opened, &cluster_inputs)?;
    preparation(&value, &opened, candidate)?;
    let manifest = opened.manifest();
    if value["record_sha256"] != manifest.record_sha256
        || manifest.record.seed != 7
        || manifest.record.outcome.class != "target_failure"
        || manifest.record.identity.implementation
            != digest(&json!({"host":candidate.host_sha256,"helper":candidate.helper_sha256}))?
        || manifest.record.identity.platform
            != canonical::digest(
                format!("{}/{}", candidate.host_os, candidate.host_arch).as_bytes(),
            )
    {
        return Err("support inspection record or implementation candidate changed".into());
    }
    opened.payload("target/module.wasm", 4 << 30)?;
    let stdout = opened.payload("evidence/cluster/history/1/stdout", 1 << 20)?;
    let stdout = std::str::from_utf8(&stdout).map_err(|_| "inspection stdout is not UTF-8")?;
    if !stdout.contains("workflow completed [step 0 attempt 2 step 1 attempt 2]")
        || !stdout.contains(
            "Temporal inspection canary after workflow timers and activity retries completed",
        )
    {
        return Err("support inspection lacks completed workflow timers/retries witness".into());
    }
    let cluster: Value =
        spec::strict(&opened.payload("evidence/cluster-replay.json", MAXIMUM_LIST)?)?;
    let config = &cluster["configuration"];
    let incarnations = config["incarnations"]
        .as_array()
        .ok_or("inspection incarnations missing")?;
    if config["workload"] != json!({"kind":"temporal-onebox-stopped-inspection"})
        || config["nodes"] != json!(["history"])
        || incarnations.len() != 1
        || incarnations[0]["handle"] != json!({"node":"history","incarnation":1})
        || incarnations[0]["arguments"]
            != json!([
                "gomad-target",
                "-test.run=^TestReplayInspectionWorkflowFailure$",
                "-test.v"
            ])
        || incarnations[0]["model"]["limit"] != 1_u64 << 30
    {
        return Err("support inspection changed original cell selection or topology".into());
    }
    for limits in [&config["limits"], &incarnations[0]["limits"]] {
        if limits["memory_bytes"] != 4_u64 << 30
            || limits["fuel"] != 1_u64 << 50
            || limits["output_bytes"] != 1_u64 << 20
            || limits["wall_millis"] != 1_800_000
        {
            return Err("support inspection original execution bounds changed".into());
        }
    }
    let receipt_root = path.parent().ok_or("inspection receipt parent missing")?;
    diagnostics(&value, &opened, &artifact, receipt_root)?;
    commands(&value, candidate, receipt_root)?;
    opened.authenticate_all()?;
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "One preparation admission binds the raw plan, artifact, provenance and explicit inventory aliases"
)]
fn preparation(value: &Value, failure: &Opened, candidate: &Candidate) -> Result<(), String> {
    let preparation = &value["preparation"];
    let path = |value: &Value| {
        value
            .as_str()
            .map(PathBuf::from)
            .ok_or_else(|| "inspection preparation path missing".to_owned())
    };
    let text = |value: &Value| {
        value
            .as_str()
            .map(str::to_owned)
            .ok_or_else(|| "inspection preparation identity missing".to_owned())
    };
    let campaign = path(&preparation["campaign"])?;
    absolute(&campaign)?;
    let plan_path = path(&preparation["plan"]["path"])?;
    if plan_path != campaign.join("plan.json") {
        return Err("inspection preparation plan escaped its campaign".into());
    }
    let plan_bytes = bound_file(
        &FileBinding {
            path: plan_path,
            sha256: text(&preparation["plan"]["sha256"])?,
        },
        MAXIMUM_LIST,
    )?;
    let plan: Value = spec::strict(&plan_bytes)?;
    if canonical::encode(&plan)? != plan_bytes {
        return Err("inspection preparation plan is not canonical".into());
    }
    let metadata = plan.get("prepared").unwrap_or(&plan);
    if plan["schema"] != "gomad.seed-campaign-plan/v2"
        || plan["selection"] != "7"
        || plan["selection_count"] != 1
        || plan["parallel"] != 1
        || plan["on_failure"] != "first"
        || plan["failure_budget"] != 1
        || plan["overall_nanos"] != 2_700_000_000_000_u64
        || plan["terminate_grace_nanos"] != 1_000_000_000
    {
        return Err("inspection original campaign envelope changed".into());
    }
    let reference = text(&metadata["prepared_artifact"])?;
    open::components(&reference)?;
    let prepared_path = path(&preparation["artifact"])?;
    if campaign.join(&reference) != prepared_path {
        return Err("inspection prepared artifact reference changed".into());
    }
    let opened = Opened::open(&prepared_path)?;
    opened.authenticate_all()?;
    let record = &opened.manifest().record;
    if preparation["record_sha256"] != opened.manifest().record_sha256
        || record.outcome.class != "prepared"
        || record.outcome.termination != "prepared"
        || record.outcome.exit_code.is_some()
        || !record.outcome.reason.is_empty()
        || record.seed != 0
        || record.virtual_time_nanos != 0
        || record.identity.target != failure.manifest().record.identity.target
        || record.identity.toolchain != failure.manifest().record.identity.toolchain
    {
        return Err("inspection prepared artifact differs from failure or ran a guest".into());
    }
    let raw = opened.payload("target/provenance.json", 16 << 20)?;
    if canonical::digest(&raw) != text(&preparation["provenance_sha256"])? {
        return Err("inspection raw provenance hash changed".into());
    }
    let provenance: Value = spec::strict(&raw)?;
    crate::campaign::prepare::typed_provenance(&provenance)?;
    let inputs = &provenance["inputs"];
    let module = opened.payload("target/module.wasm", 512 << 20)?;
    let module_metadata: Option<gomad_wasmhost::linked::ModuleMetadata> =
        serde_json::from_value(provenance["module_metadata"].clone())
            .map_err(|error| error.to_string())?;
    crate::campaign::prepare::authenticate_module_metadata(
        module_metadata.as_ref(),
        inputs,
        &module,
    )?;
    if canonical::encode(&provenance)? != raw
        || provenance["build_key"] != digest(inputs)?
        || provenance["build_key"] != record.identity.toolchain
        || provenance["module_sha256"] != record.identity.target
        || canonical::digest(&module) != record.identity.target
        || inputs["compiler_sha256"] != candidate.compiler_sha256
        || inputs["helper"] != candidate.helper_sha256
        || inputs["host"] != candidate.host_sha256
        || inputs["runtime_implementation"] != candidate.runtime_implementation_sha256
        || metadata["toolchain"] != record.identity.toolchain
        || metadata["module_sha256"] != record.identity.target
    {
        return Err("inspection prepared provenance is not bound to this candidate".into());
    }
    let configuration: Configuration = serde_json::from_value(provenance["configuration"].clone())
        .map_err(|error| error.to_string())?;
    if record.identity.configuration != digest(&configuration)?
        || record.identity.implementation
            != digest(&json!({"host":candidate.host_sha256,"helper":candidate.helper_sha256}))?
        || record.identity.platform
            != canonical::digest(
                format!("{}/{}", candidate.host_os, candidate.host_arch).as_bytes(),
            )
        || record.arguments != configuration.arguments
        || record.environment != configuration.environment
        || record.limits != configuration.limits
        || record.profile != configuration.profile
    {
        return Err("inspection prepared record is not bound to its full configuration".into());
    }
    let staged = path(&value["source_detached"])?;
    let staged = PathBuf::from(
        staged
            .to_str()
            .and_then(|path| path.strip_suffix("-unavailable"))
            .ok_or("inspection staged source alias invalid")?,
    );
    source::inspection_staged(&provenance, &staged, candidate)?;
    policy(&provenance, metadata, &staged, candidate)?;
    if provenance["scenario"] != json!(recording::scenario(&value["recording_scenario"])?) {
        return Err(
            "inspection recording scenario differs from authenticated prepared provenance".into(),
        );
    }
    let overlay: inventory::Overlay = serde_json::from_value(value["compiler_overlay"].clone())
        .map_err(|error| error.to_string())?;
    let inventory = inventory::from_provenance(&provenance, Some(overlay))?;
    let private = path(&value["private_inputs_detached"])?;
    let private = PathBuf::from(
        private
            .to_str()
            .and_then(|path| path.strip_suffix("-unavailable"))
            .ok_or("inspection private alias root invalid")?,
    );
    let cache = path(&value["cache_origin"])?;
    absolute(&cache)?;
    let candidate_cache = candidate
        .inventory
        .overlay
        .path
        .parent()
        .and_then(Path::parent)
        .and_then(Path::parent)
        .ok_or("candidate overlay cache root missing")?;
    let receipt_root = Path::new(
        value["recording_scenario"]["path"]
            .as_str()
            .ok_or("inspection scenario path missing")?,
    )
    .parent()
    .ok_or("inspection scenario root missing")?;
    if cache != receipt_root.join("preparation-cache")
        || path(&value["cache_request_origin"])? != candidate_cache
        || inventory.compiler_path != private.join("compiler/bin/go")
        || inventory.runtime_root != private.join("runtime")
        || path(&value["compiler_origin"])? != candidate.inventory.compiler_path
        || path(&value["runtime_origin"])? != candidate.inventory.runtime_root
        || inventory.overlay.path
            != cache
                .join("runtime-overlay")
                .join(&inventory.runtime_identity)
                .join("overlay.json")
        || inventory::view(&inventory)? != inventory::view(&candidate.inventory)?
    {
        return Err(
            "inspection compiler/runtime/overlay alias inventory differs from candidate".into(),
        );
    }
    opened.authenticate_all()?;
    Ok(())
}

fn diagnostics(
    value: &Value,
    opened: &Opened,
    artifact: &Path,
    receipt_root: &Path,
) -> Result<(), String> {
    let string = |value: &Value| {
        value
            .as_str()
            .map(str::to_owned)
            .ok_or_else(|| "inspection diagnostic field missing".to_owned())
    };
    let report = &value["report"];
    if report["status"] != "inspected_prefix"
        || report["prefix_verified"] != true
        || report["witness_match"] != true
        || report["truncated"] != false
        || report["resolved"]["instance"] != "history"
        || report["resolved"]["incarnation"] != 1
    {
        return Err("support inspection prefix/witness does not match".into());
    }
    let debug_path = PathBuf::from(string(&report["output_path"])?);
    if debug_path != receipt_root.join("inspection/debug.json") {
        return Err("inspection diagnostic output escaped its retained root".into());
    }
    let debug: Value = spec::strict(&read(&debug_path, 8 << 20)?)?;
    if debug["stacks"].as_array().is_none_or(Vec::is_empty) {
        return Err("support inspection stacks missing".into());
    }
    let probes = debug["probes"]
        .as_array()
        .ok_or("support inspection probes missing")?;
    for name in ["temporal.shard.snapshot", "temporal.mutable_state.snapshot"] {
        if !probes.iter().any(|probe| probe["name"] == name) {
            return Err("support inspection snapshot probe missing".into());
        }
    }
    let dumps = debug["dumps"]
        .as_array()
        .ok_or("support inspection dumps missing")?;
    for name in ["temporal.mutable_states", "temporal.shards"] {
        let rows: Vec<_> = dumps.iter().filter(|dump| dump["name"] == name).collect();
        if rows.len() != 1 || rows[0]["status"] != "ok" {
            return Err("support inspection useful snapshot dump missing".into());
        }
        let body = crate::qualify::cells::base64(&string(&rows[0]["data_base64"])?)
            .ok_or("inspection dump base64 invalid")?;
        if body.is_empty()
            || body.len() > 4 << 20
            || !body
                .windows(b"snapshot=".len())
                .any(|window| window == b"snapshot=")
            || name == "temporal.mutable_states"
                && !body
                    .windows(b"values may precede stop".len())
                    .any(|window| window == b"values may precede stop")
        {
            return Err("support inspection useful snapshot content missing".into());
        }
    }
    let diagnostic_root = debug_path
        .parent()
        .ok_or("inspection diagnostic parent missing")?;
    let heap = read(&diagnostic_root.join("heap.pprof"), 4 << 20)?;
    gomad_wasmhost::linked_inspection::validate_heap_profile(&heap, 4 << 20)
        .map_err(|error| error.to_string())?;
    for (root, field) in [
        (artifact, "artifact_sha256"),
        (diagnostic_root, "diagnostic_sha256"),
    ] {
        let inventory = value[field]
            .as_object()
            .ok_or("inspection inventory missing")?;
        let expected_paths: BTreeSet<_> = inventory.keys().cloned().collect();
        if inventory.is_empty() || inventory_files(root)? != expected_paths {
            return Err("inspection complete retained inventory membership changed".into());
        }
        for (name, expected) in inventory {
            open::components(name)?;
            let actual = if field == "artifact_sha256" {
                match opened.manifest().files.get(name) {
                    Some(entry) => entry.sha256.clone(),
                    None => canonical::digest(&read(&root.join(name), MAXIMUM_LIST)?),
                }
            } else {
                canonical::digest(&read(&root.join(name), 8 << 20)?)
            };
            if actual != string(expected)? {
                return Err("inspection retained inventory changed".into());
            }
        }
    }
    Ok(())
}

fn commands(value: &Value, candidate: &Candidate, receipt_root: &Path) -> Result<(), String> {
    let string = |value: &Value| {
        value
            .as_str()
            .map(str::to_owned)
            .ok_or_else(|| "inspection command field missing".to_owned())
    };
    let commands = value["commands"]
        .as_object()
        .ok_or("inspection command receipts missing")?;
    let expected = [
        ("install", 0),
        ("explore", 1),
        ("detached-replay", 1),
        ("stopped-inspection", 0),
        ("reopened-replay", 1),
    ];
    if commands.len() != expected.len() {
        return Err("inspection command receipts incomplete".into());
    }
    for (name, exit) in expected {
        let binding = &commands[name];
        let command_bytes = bound_file(
            &FileBinding {
                path: PathBuf::from(string(&binding["path"])?),
                sha256: string(&binding["sha256"])?,
            },
            1 << 20,
        )?;
        let command: Value = spec::strict(&command_bytes)?;
        if binding["exit"] != exit
            || command["exit"] != exit
            || command["timed_out"] != false
            || command["name"] != name
            || command["command"][0] != candidate.executable.path.to_string_lossy().as_ref()
        {
            return Err("inspection command exit or executable changed".into());
        }
        for stream in ["stdout", "stderr"] {
            bound_file(
                &FileBinding {
                    path: PathBuf::from(string(&command[stream])?),
                    sha256: string(&command[format!("{stream}_sha256")])?,
                },
                MAXIMUM_LIST,
            )?;
        }
        if ["detached-replay", "stopped-inspection", "reopened-replay"].contains(&name)
            && command["environment_intent"]["detached"] != true
        {
            return Err("inspection detached replay intent missing".into());
        }
        if ["install", "explore"].contains(&name) {
            recording::command(name, &command, value, candidate, receipt_root)?;
        }
        if name == "explore" {
            recording::response(&command, value)?;
        }
        if name == "stopped-inspection" {
            let report: Value = spec::strict(&read(
                Path::new(&string(&command["stdout"])?),
                MAXIMUM_LIST,
            )?)?;
            let expected = json!([
                candidate.executable.path,
                "replay",
                "--stop=failure",
                "--instance=history",
                "--incarnation=1",
                "--dump=stacks",
                "--dump=heap",
                "--dump=temporal.mutable_states",
                "--dump=temporal.shards",
                "--probe=temporal.*.snapshot",
                "--debug-bytes",
                "8388608",
                "--dump-bytes",
                "4194304",
                "--debug-output",
                receipt_root.join("inspection"),
                value["artifact"]
            ]);
            if report != value["report"] || command["command"] != expected {
                return Err(
                    "inspection report or exact stopped command differs from raw process stdout"
                        .into(),
                );
            }
        }
        if name.ends_with("replay") {
            if command["command"] != json!([candidate.executable.path, "replay", value["artifact"]])
            {
                return Err("inspection detached replay did not select its own artifact".into());
            }
            let stdout = read(Path::new(&string(&command["stdout"])?), MAXIMUM_LIST)?;
            if stdout.strip_suffix(b"\n").unwrap_or(&stdout)!=b"gomad: reproduced=true diagnostic=false result=target_failure choice-replay=exact" {return Err("inspection full detached replay did not match".into());}
        }
    }
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "The original inspection policy is a closed complete configuration, checked as one admission"
)]
fn policy(
    provenance: &Value,
    metadata: &Value,
    staged: &Path,
    candidate: &Candidate,
) -> Result<(), String> {
    let configuration: Configuration = serde_json::from_value(provenance["configuration"].clone())
        .map_err(|error| error.to_string())?;
    let inputs = &provenance["inputs"];
    let key = provenance["build_key"]
        .as_str()
        .ok_or("inspection build key missing")?;
    let choice: gomad_wasmhost::linked::ChoiceIdentity =
        serde_json::from_value(provenance["choice_identity"].clone())
            .map_err(|error| error.to_string())?;
    let expected_implementation = crate::campaign::prepare::choice_implementation(key)?;
    let mut built = provenance["configuration"].clone();
    built["runtime_sha256"] = Value::Null;
    if choice.toolchain_build_key != key
        || choice.implementation_sha256.as_slice() != &expected_implementation[..]
        || choice.goos != "wasip1"
        || choice.goarch != "wasm"
        || configuration.runtime_sha256.as_deref()
            != Some(choice.runtime_implementation_sha256().as_str())
        || metadata["configuration"] != provenance["configuration"]
        || inputs["configuration"] != built
        || provenance["network"] != inputs["network"]
        || provenance["scenario"] != inputs["scenario"]
        || inputs["soft_assertion_report_only"] != false
        || inputs["go_version"] != "go1.27.1"
    {
        return Err("inspection full provenance/configuration identity changed".into());
    }
    if choice.target_sha256.as_slice()
        != super::hex_bytes(
            provenance["module_sha256"]
                .as_str()
                .ok_or("inspection module hash missing")?,
        )?
        .as_slice()
        || metadata["choice_identity"] != provenance["choice_identity"]
        || metadata["scenario"] != provenance["scenario"]
        || metadata["network"] != provenance["network"]
        || metadata["soft_assertion_report_only"] != false
    {
        return Err("inspection prepared plan does not bind its complete provenance".into());
    }
    let args = json!([
        "gomad-target",
        "-test.run=^TestReplayInspectionWorkflowFailure$",
        "-test.v"
    ]);
    let relative = staged
        .strip_prefix(&candidate.source_root)
        .map_err(|_| "inspection stage escaped source checkout")?;
    if inputs["kind"] != "go-test"
        || inputs["source"] != format!("./{}", relative.display())
        || inputs["directory"] != candidate.source_root.to_string_lossy().as_ref()
        || inputs["tags"]
            != json!([
                "disable_grpc_modules",
                "gomad",
                "gomad_wasm_slim",
                "sqlite3_dotlk",
                "test_dep"
            ])
        || serde_json::to_value(&configuration.arguments).map_err(|error| error.to_string())?
            != args
        || inputs["arguments"]
            != json!(["-test.run=^TestReplayInspectionWorkflowFailure$", "-test.v"])
        || configuration.storage.is_some()
        || configuration.profile != "gomad.wasi-cooperative/v1"
        || !configuration.derive_seed_environment
        || configuration.limits.fuel != 1 << 50
        || configuration.limits.memory_bytes != 4 << 30
        || configuration.limits.output_bytes != 1 << 20
        || configuration.limits.wall_millis != 1_800_000
        || configuration.wall_nanos != Some(1_800_000_000_000)
    {
        return Err("inspection original target, tags or campaign limits changed".into());
    }
    let model = json!({"entropy_key":vec![0;32],"working_directory_sha256":canonical::digest(b"/workspace"),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"mode":1,"limit":1_u64<<30,"diagnostics":false,"epoch_nanos":946_684_800_000_000_000_u64,"read_step_nanos":0,"now":1,"limits":{"calls":1<<28,"transcript_bytes":256<<20,"pending_events":4096}});
    let network = json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":4096,"bytes":128<<20,"connection_bytes":4<<20,"stall_nanos":10_000_000_000_u64,"transitions":1<<20}});
    let environment = BTreeMap::from([
        ("PWD".into(), "/workspace".into()),
        ("GOMAD_DEBUG_CANARY".into(), "1".into()),
        ("TEMPORAL_ROOT".into(), "/workspace".into()),
        ("TMPDIR".into(), "/tmp".into()),
        ("TZ".into(), "UTC".into()),
        ("TEMPORAL_TEST_LOG_LEVEL".into(), "warn".into()),
        ("GOMAD_IO_PROFILE".into(), "wasm".into()),
    ]);
    let namespace = configuration
        .namespace
        .as_ref()
        .ok_or("inspection schema namespace missing")?;
    if configuration.model.as_ref() != Some(&model)
        || configuration.environment != environment
        || serde_json::to_value(&configuration.network).map_err(|error| error.to_string())?
            != network
        || namespace.working_directory != b"/workspace"
        || namespace.writable_directories != [b"/workspace".to_vec(), b"/tmp".to_vec()]
        || namespace.preopens
            != [
                b"/tmp".to_vec(),
                b"/workspace".to_vec(),
                b"/workspace/schema/sqlite".to_vec(),
            ]
        || !namespace.stdin.is_empty()
        || namespace.epoch_nanos != 946_684_800_000_000_000
        || namespace.limits.files != 100_000
        || namespace.limits.descriptors != 4096
        || namespace.limits.filesystem_bytes != 512 << 20
        || namespace.limits.output_bytes != 1 << 20
    {
        return Err(
            "inspection complete original model, namespace, environment or network changed".into(),
        );
    }
    source::schema(namespace, candidate)
}

fn inventory_files(path: &Path) -> Result<BTreeSet<String>, String> {
    use rustix::fs::{self, AtFlags, FileType};
    let root = open::pin(path)?;
    let mut pending = vec![(root, String::new())];
    let mut files = BTreeSet::new();
    while let Some((directory, prefix)) = pending.pop() {
        for entry in fs::Dir::read_from(&directory).map_err(|error| error.to_string())? {
            let entry = entry.map_err(|error| error.to_string())?;
            let name = entry
                .file_name()
                .to_str()
                .map_err(|_| "inspection inventory filename is not UTF-8")?;
            if matches!(name, "." | "..") {
                continue;
            }
            let full = if prefix.is_empty() {
                name.to_owned()
            } else {
                format!("{prefix}/{name}")
            };
            open::components(&full)?;
            let stat = fs::statat(&directory, name, AtFlags::SYMLINK_NOFOLLOW)
                .map_err(|error| error.to_string())?;
            match FileType::from_raw_mode(stat.st_mode) {
                FileType::Directory => {
                    pending.push((open::child(&directory, name, true, false)?, full));
                }
                FileType::RegularFile => {
                    files.insert(full);
                }
                _ => return Err("inspection inventory contains a nonregular member".into()),
            }
            if files.len() + pending.len() > 100_000 {
                return Err("inspection inventory membership exceeds bound".into());
            }
        }
    }
    Ok(files)
}
