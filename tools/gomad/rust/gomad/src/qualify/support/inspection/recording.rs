use super::{Candidate, Path, Value, absolute, canonical, json, spec};

fn original() -> Value {
    json!({"schema":"gomad-wasm.scenario/v1","workload":{"kind":"temporal-onebox-stopped-inspection"},
        "spec":{"schema":"gomad.simulation-spec/v8","backend":"wasm","fidelity":"simulation_model","seed":0,
            "limits":{"nodes":64,"directional_links":4096,"volumes":64,"scenario_actions":4096,"scenario_decisions":4096,
                "fault_actions":4096,"history_operations":65536,"observations":65536,"oracle_results":4096,
                "scenario_evidence_bytes":64<<20,"boot_config_bytes":1<<20,"volume_capacity_bytes":1<<30,
                "observation_bytes":64<<20,"network_listeners":128,"network_connections":4096,"network_bytes":128<<20,
                "network_connection_bytes":4<<20,"network_stall_nanos":10_000_000_000_u64,"network_transitions":1<<20,
                "volume_operations":100_000,"volume_transitions":1<<20,"crash_states":32,"crash_depth":8,
                "crash_bytes":4<<20,"crash_wall_nanos":1_000_000_000},
            "nodes":[{"id":"history","boot":"main","address":"127.0.0.1"}],"links":[],"volumes":[]},
        "boots":[{"id":"main","args":["-test.run=^TestReplayInspectionWorkflowFailure$","-test.v"]}],
        "actions":[{"action":"start","handle":{"node":"history","incarnation":1}},
                   {"action":"wait","handle":{"node":"history","incarnation":1}}],
        "incarnation_boots":[],"output_crashes":[],"output_stops":[],"network_faults":[]})
}

pub(super) fn scenario(witness: &Value) -> Result<Vec<u8>, String> {
    let witness: super::super::SpecWitness =
        serde_json::from_value(witness.clone()).map_err(|error| error.to_string())?;
    absolute(&witness.path)?;
    if witness.bytes_base64.len() > (256_usize << 10).div_ceil(3) * 4 {
        return Err("inspection scenario witness exceeds original256KiB bound".into());
    }
    let bytes = crate::qualify::cells::base64(&witness.bytes_base64)
        .filter(|bytes| bytes.len() <= 256 << 10)
        .ok_or("invalid bounded scenario witness")?;
    let raw: Value = spec::strict(&bytes)?;
    if canonical::digest(&bytes) != witness.sha256 || raw != original() {
        return Err("inspection original raw scenario hash or complete envelope changed".into());
    }
    crate::campaign::scenario::normalize(&bytes)
}

fn explore(candidate: &Candidate, root: &Path, stage: &Path) -> Result<Vec<String>, String> {
    let source = stage
        .strip_prefix(&candidate.source_root)
        .map_err(|_| "inspection stage escapes source")?;
    let mut command = vec![
        candidate.executable.path.to_string_lossy().into_owned(),
        "--installation".into(),
        root.join("installation.json")
            .to_string_lossy()
            .into_owned(),
    ];
    command.extend(
        [
            "explore",
            "--json",
            "--seeds",
            "7",
            "--parallel",
            "1",
            "--execution-timeout",
            "30m",
            "--overall-timeout",
            "45m",
            "--terminate-grace",
            "1s",
            "--on-failure",
            "first",
            "--failure-budget",
            "1",
            "--output-limit",
            "1048576",
            "--world-transition-limit",
            "1048576",
            "--scenario",
        ]
        .map(str::to_owned),
    );
    command.push(root.join("scenario.json").to_string_lossy().into_owned());
    command.push("--artifacts".into());
    command.push(root.join("artifacts").to_string_lossy().into_owned());
    for environment in [
        "GOMAD_DEBUG_CANARY=1",
        "TEMPORAL_ROOT=/workspace",
        "TMPDIR=/tmp",
        "TEMPORAL_TEST_LOG_LEVEL=warn",
    ] {
        command.extend(["--env".into(), environment.into()]);
    }
    for tag in super::super::TAGS.into_iter().chain(["gomad"]) {
        command.extend(["--build-tag".into(), tag.into()]);
    }
    command.extend([
        "--working-dir".into(),
        candidate.source_root.to_string_lossy().into_owned(),
        "go-test".into(),
        format!("./{}", source.display()),
        "--".into(),
        "-test.run=^TestReplayInspectionWorkflowFailure$".into(),
        "-test.v".into(),
    ]);
    Ok(command)
}

pub(super) fn command(
    name: &str,
    command: &Value,
    receipt: &Value,
    candidate: &Candidate,
    root: &Path,
) -> Result<(), String> {
    let original_stage = receipt["source_detached"]
        .as_str()
        .and_then(|path| path.strip_suffix("-unavailable"))
        .ok_or("inspection staged source path missing")?;
    let stage = candidate.source_root.join("service/history").join(format!(
        ".gomad-inspection-onebox-{}",
        &canonical::digest(root.to_string_lossy().as_bytes())[..16]
    ));
    if Path::new(original_stage) != stage
        || command["cwd"] != json!(candidate.source_root)
        || command["pid"].as_u64().is_none_or(|pid| pid == 0)
        || command["start_ns"].as_u64().is_none_or(|time| time == 0)
        || command["end_ns"].as_u64() < command["start_ns"].as_u64()
    {
        return Err("inspection recording source, process or wall-clock boundary changed".into());
    }
    let expected = match name {
        "explore" => {
            scenario(&receipt["recording_scenario"])?;
            if command["recording_scenario"] != receipt["recording_scenario"]
                || receipt["recording_scenario"]["path"] != json!(root.join("scenario.json"))
                || command["timeout_seconds"] != 2700
            {
                return Err(
                    "inspection recording command lost raw scenario or timeout witness".into(),
                );
            }
            explore(candidate, root, &stage)?
        }
        "install" => {
            let private = root.join("private-inputs");
            if receipt["private_inputs_detached"] != json!(root.join("private-inputs-unavailable"))
                || command["timeout_seconds"] != 600
            {
                return Err("inspection installation private alias or timeout changed".into());
            }
            vec![
                candidate.executable.path.to_string_lossy().into_owned(),
                "install".into(),
                "--compiler".into(),
                private
                    .join("compiler/bin/go")
                    .to_string_lossy()
                    .into_owned(),
                "--runtime-root".into(),
                private.join("runtime").to_string_lossy().into_owned(),
                "--cache".into(),
                root.join("preparation-cache")
                    .to_string_lossy()
                    .into_owned(),
                "--output".into(),
                root.join("installation.json")
                    .to_string_lossy()
                    .into_owned(),
                "--profile".into(),
                "gomad.wasi-cooperative/v1".into(),
                "--config".into(),
                root.join("policy.json").to_string_lossy().into_owned(),
                "--fuel".into(),
                (1_u64 << 50).to_string(),
                "--memory-bytes".into(),
                (4_u64 << 30).to_string(),
            ]
        }
        _ => return Err("unknown inspection recording command".into()),
    };
    if command["command"] != json!(expected) || command["argv"] != json!(expected) {
        return Err("inspection original exact recording command or bounds changed".into());
    }
    Ok(())
}

pub(super) fn response(command: &Value, receipt: &Value) -> Result<(), String> {
    let bytes = super::bound_file(
        &super::FileBinding {
            path: command["stdout"]
                .as_str()
                .ok_or("inspection recording stdout path missing")?
                .into(),
            sha256: command["stdout_sha256"]
                .as_str()
                .ok_or("inspection recording stdout hash missing")?
                .into(),
        },
        super::MAXIMUM_LIST,
    )?;
    let response: Value = spec::strict(&bytes)?;
    if response["attempted"] != 1
        || response["failures"] != 1
        || response["retained_evidence"] != 1
        || response["campaign_path"] != receipt["preparation"]["campaign"]
    {
        return Err("inspection recording stdout differs from retained preparation".into());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    fn witness(value: &Value) -> Value {
        let bytes = canonical::encode(value).unwrap();
        json!({"path":"/retained/inspection/scenario.json","sha256":canonical::digest(&bytes),
            "bytes_base64":crate::campaign::exploration::base64(&bytes)})
    }
    #[test]
    fn resealed_recording_scenario_refuses_changed_address_boot_actions_and_fault_bounds() {
        let original = original();
        scenario(&witness(&original)).unwrap();
        for (pointer, replacement) in [
            ("/spec/nodes/0/address", json!("127.0.0.2")),
            ("/boots/0/args", json!(["-test.run=OtherInspection"])),
            ("/spec/limits/network_bytes", json!(256 << 20)),
            (
                "/actions",
                json!([{"action":"start","handle":{"node":"history","incarnation":1}}]),
            ),
            (
                "/output_stops",
                json!([{"handle":{"node":"history","incarnation":1},"marker":"eA=="}]),
            ),
        ] {
            let mut changed = original.clone();
            *changed.pointer_mut(pointer).unwrap() = replacement;
            assert!(
                scenario(&witness(&changed)).is_err(),
                "accepted resealed {pointer}"
            );
        }
    }

    #[test]
    fn resealed_recording_command_refuses_changed_world_cap_and_extra_options() {
        let candidate = super::super::super::tests::source_candidate();
        let root = Path::new("/retained/inspection");
        let stage = candidate.source_root.join("service/history").join(format!(
            ".gomad-inspection-onebox-{}",
            &canonical::digest(root.to_string_lossy().as_bytes())[..16]
        ));
        let scenario = witness(&original());
        let receipt = json!({"source_detached":format!("{}-unavailable",stage.display()),"recording_scenario":scenario});
        let argv = explore(&candidate, root, &stage).unwrap();
        let observed = json!({"cwd":candidate.source_root,"pid":42,"start_ns":100,"end_ns":200,
            "timeout_seconds":2700,"command":argv,"argv":argv,"recording_scenario":receipt["recording_scenario"]});
        command("explore", &observed, &receipt, &candidate, root).unwrap();
        let position = argv
            .iter()
            .position(|arg| arg == "--world-transition-limit")
            .unwrap()
            + 1;
        let mut changed = observed.clone();
        changed["command"][position] = json!((2 << 20).to_string());
        changed["argv"] = changed["command"].clone();
        assert!(command("explore", &changed, &receipt, &candidate, root).is_err());
        let mut changed = observed;
        changed["command"]
            .as_array_mut()
            .unwrap()
            .push(json!("--choices"));
        changed["argv"] = changed["command"].clone();
        assert!(command("explore", &changed, &receipt, &candidate, root).is_err());
    }
}
