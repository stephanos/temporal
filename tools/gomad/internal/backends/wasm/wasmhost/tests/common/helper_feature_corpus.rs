use super::*;

const EXECUTION_PHASES: [&str; 6] = [
    "module_read",
    "module_hash",
    "engine_init",
    "compile",
    "instantiate",
    "execute",
];

fn feature_dir() -> PathBuf {
    corpus_dir().with_file_name("helper_features")
}

pub(super) fn append_cases(files: &mut Vec<PathBuf>) {
    if BUILD != "default" {
        let mut additions: Vec<_> = std::fs::read_dir(feature_dir().join("cases"))
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|ext| ext == "json"))
            .collect();
        additions.sort();
        files.extend(additions);
    }
}

pub(super) fn check_expected(
    file: &Path,
    source: &str,
    case: &Value,
    actual: &Value,
    regenerate: bool,
) -> Result<(), String> {
    if encode(case) != source {
        return Err("feature source is not canonically encoded; preserve source and review".into());
    }
    check_semantic_baseline(file, case, actual)?;
    let name = file.file_name().ok_or("missing feature case name")?;
    let path = feature_dir().join(BUILD).join(name);
    let identity = format!("{:x}", Sha256::digest(source.as_bytes()));
    let record = json!({"schema":"gomad.helper-feature-expectation/v1",
        "case":name.to_str().ok_or("feature name is not UTF8")?,
        "source_sha256":identity,"build":BUILD,"expected":actual});
    let text = match std::fs::read_to_string(&path) {
        Ok(text) => Some(text),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
        Err(error) => return Err(error.to_string()),
    };
    let saved = text
        .as_ref()
        .map(|text| validate_sidecar(text, &record))
        .transpose()?;
    if regenerate {
        std::fs::create_dir_all(path.parent().unwrap()).map_err(|error| error.to_string())?;
        return write_case(&path, text.as_deref().unwrap_or(""), &record);
    }
    let expected = if let Some(saved) = saved {
        saved["expected"].clone()
    } else {
        let mut expected = case["expected"].clone();
        for (key, value) in case["expected_builds"][BUILD]
            .as_object()
            .into_iter()
            .flatten()
        {
            expected[key] = value.clone();
        }
        expected
    };
    if expected != *actual {
        return Err(format!(
            "{BUILD} feature expectation differs (regenerate sidecar and review): {}",
            first_difference(&expected, actual)
        ));
    }
    Ok(())
}

fn validate_sidecar(text: &str, record: &Value) -> Result<Value, String> {
    let saved: Value = serde_json::from_str(text).map_err(|error| error.to_string())?;
    let mut expected_record = record.clone();
    expected_record["expected"] = saved["expected"].clone();
    if saved != expected_record || encode(&saved) != text {
        return Err("feature sidecar identity/schema/encoding differs".into());
    }
    Ok(saved)
}

#[test]
fn feature_sidecar_rejects_schema_source_build_case_and_unknown_field_drift() {
    let record = json!({"schema":"gomad.helper-feature-expectation/v1","case":"original.json",
        "source_sha256":"original-byte-digest","build":"diagnostic-profile","expected":{}});
    validate_sidecar(&encode(&record), &record).unwrap();
    for key in ["schema", "case", "source_sha256", "build", "unknown"] {
        let mut changed = record.clone();
        changed[key] = json!("changed");
        assert!(
            validate_sidecar(&encode(&changed), &record).is_err(),
            "accepted {key} drift"
        );
    }
}

fn check_semantic_baseline(file: &Path, case: &Value, actual: &Value) -> Result<(), String> {
    if file.parent() == Some(feature_dir().join("cases").as_path()) {
        return Ok(());
    }
    if (cfg!(feature = "diagnostic-fuel-off") || cfg!(feature = "guest-profile"))
        && case["request"]["type"] == "cluster_execute"
    {
        let frames = actual["frames"]
            .as_array()
            .ok_or("missing cluster refusal")?;
        if actual["exit_code"] != 0
            || actual["stderr"] != ""
            || actual["evidence"] != json!({})
            || frames.len() != 1
            || frames[0]["termination"] != "unsupported"
        {
            return Err("diagnostic cluster must refuse before execution without evidence".into());
        }
        return Ok(());
    }
    let mut baseline = case["expected"].clone();
    let overrides = if case["request"]["type"] == "cluster_execute"
        && !cfg!(feature = "diagnostic-fuel-off")
        && !cfg!(feature = "guest-profile")
    {
        &Value::Null
    } else {
        &case["expected_builds"][BUILD]
    };
    for (key, value) in overrides.as_object().into_iter().flatten() {
        baseline[key] = value.clone();
    }
    if overrides.get("frames").is_none() {
        for frame in baseline["frames"]
            .as_array_mut()
            .ok_or("baseline frames missing")?
        {
            if let Some(configuration) = frame["engine"]["configuration"].as_str() {
                let mut configuration = configuration.to_owned();
                if cfg!(feature = "guest-profile") {
                    configuration.push_str("-guest-profile");
                }
                if cfg!(feature = "diagnostic-fuel-off") {
                    configuration.push_str("-fuel-off");
                }
                frame["engine"]["configuration"] = json!(configuration);
            }
            if cfg!(feature = "diagnostic-fuel-off") && frame.get("fuel_remaining").is_some() {
                frame["fuel_remaining"] = fuel_without_metering(frame, &case["request"]["fuel"])?;
            }
        }
    }
    let mut observed = actual.clone();
    if baseline["diagnostic_profile"]["schema"] == "gomad.wasm-profile/v1"
        && observed["diagnostic_profile"]["schema"] == "gomad.wasm-profile/v2"
    {
        let old = baseline["diagnostic_profile"].as_object().unwrap();
        let current = observed["diagnostic_profile"].as_object_mut().unwrap();
        current.retain(|key, _| old.contains_key(key));
        current.insert("schema".into(), json!("gomad.wasm-profile/v1"));
    }
    if baseline.get("diagnostic_profile").is_none() {
        observed
            .as_object_mut()
            .unwrap()
            .remove("diagnostic_profile");
    }
    if baseline != observed {
        return Err(format!(
            "feature changed preserved semantic baseline: {}",
            first_difference(&baseline, &observed)
        ));
    }
    Ok(())
}

fn fuel_without_metering(baseline: &Value, offered: &Value) -> Result<Value, String> {
    let metrics = &baseline["metrics"];
    if metrics["instantiate_nanos"] == "nonzero"
        && (metrics["execute_nanos"] == "nonzero" || metrics["execute_nanos"] == 0)
    {
        // Failed instantiation also reaches terminal_outcome before execution starts.
        return offered
            .as_u64()
            .filter(|fuel| *fuel > 0)
            .map(|fuel| json!(fuel))
            .ok_or("terminal baseline lacks positive offered fuel".into());
    }
    if (metrics.is_null() || (metrics["instantiate_nanos"] == 0 && metrics["execute_nanos"] == 0))
        && baseline["fuel_remaining"] == 0
    {
        return Ok(json!(0));
    }
    Err("fuel-off baseline has an unrecognized admission/store stage".into())
}

#[test]
fn fuel_off_stage_comes_from_preserved_baseline_including_failed_instantiation() {
    let offered = json!(100);
    for metrics in [
        Value::Null,
        json!({"instantiate_nanos":0,"execute_nanos":0}),
    ] {
        let mut baseline = json!({"fuel_remaining":0,"metrics":metrics});
        assert_eq!(fuel_without_metering(&baseline, &offered).unwrap(), 0);
        baseline["fuel_remaining"] = json!(100);
        assert!(fuel_without_metering(&baseline, &offered).is_err());
    }
    for execute in [json!(0), json!("nonzero")] {
        let baseline = json!({"fuel_remaining":9,
            "metrics":{"instantiate_nanos":"nonzero","execute_nanos":execute}});
        assert_eq!(fuel_without_metering(&baseline, &offered).unwrap(), 100);
    }
    for metrics in [
        json!({}),
        json!({"instantiate_nanos":0,"execute_nanos":"nonzero"}),
        json!({"instantiate_nanos":"unknown","execute_nanos":0}),
    ] {
        let baseline = json!({"fuel_remaining":0,"metrics":metrics});
        assert!(fuel_without_metering(&baseline, &offered).is_err());
    }
}

#[cfg(feature = "diagnostic-fuel-off")]
#[test]
fn fuel_off_sidecars_require_early_zero_and_terminal_offered_fuel() {
    let file = corpus_dir().join("fuel-stage-guard.json");
    for (instantiate, execute, baseline_fuel, expected_fuel) in [
        (json!(0), json!(0), 0, 0),
        (json!("nonzero"), json!(0), 9, 100),
        (json!("nonzero"), json!("nonzero"), 9, 100),
    ] {
        let baseline = json!({"frames":[{"type":"result","fuel_remaining":baseline_fuel,
            "metrics":{"instantiate_nanos":instantiate,"execute_nanos":execute}}]});
        let case = json!({"request":{"type":"execute","fuel":100},"expected":baseline});
        let mut actual = baseline;
        actual["frames"][0]["fuel_remaining"] = json!(expected_fuel);
        check_semantic_baseline(&file, &case, &actual).unwrap();
        actual["frames"][0]["fuel_remaining"] = json!(if expected_fuel == 0 { 100 } else { 0 });
        assert!(check_semantic_baseline(&file, &case, &actual).is_err());
    }
}

pub(super) fn check_raw_profile(case: &Value, run: &Run) -> Result<(), String> {
    if !case["intent"]
        .as_str()
        .unwrap_or("")
        .starts_with("feature_hooks_")
    {
        return Ok(());
    }
    let document = run
        .stderr
        .strip_prefix("gomad-wasm-profile-v1 ")
        .ok_or("feature hooks profile prefix missing")?;
    if run.stderr.len() >= 48 * 1024
        || run.stderr.bytes().filter(|byte| *byte == b'\n').count() != 1
    {
        return Err("feature hooks profile must be one bounded line".into());
    }
    let profile: Value = serde_json::from_str(document).map_err(|error| error.to_string())?;
    if profile["schema"] != "gomad.wasm-profile/v2" {
        return Err("feature hooks profile schema differs".into());
    }
    for phase in EXECUTION_PHASES {
        if profile["phases"][phase]["wall_nanos"]
            .as_u64()
            .is_none_or(|n| n == 0)
        {
            return Err(format!(
                "feature hooks {phase} must measure positive wall time"
            ));
        }
    }
    Ok(())
}

#[test]
fn feature_hooks_reject_missing_malformed_unbounded_and_zero_raw_profiles() {
    let case = json!({"intent":"feature_hooks_exit: guard"});
    let mut profile = json!({"schema":"gomad.wasm-profile/v2","phases":{}});
    for phase in EXECUTION_PHASES {
        profile["phases"][phase] = json!({"wall_nanos":1});
    }
    let run = |stderr| Run {
        frames: vec![],
        raw: vec![],
        stderr,
        exit_code: Some(0),
    };
    let valid = format!("gomad-wasm-profile-v1 {profile}\n");
    check_raw_profile(&case, &run(valid.clone())).unwrap();
    for invalid in [
        String::new(),
        "gomad-wasm-profile-v1 {}\n".into(),
        valid.trim_end().into(),
        valid.repeat(2),
        format!("{valid}{}", " ".repeat(48 * 1024)),
    ] {
        assert!(check_raw_profile(&case, &run(invalid)).is_err());
    }
    for phase in EXECUTION_PHASES {
        let mut zero = profile.clone();
        zero["phases"][phase]["wall_nanos"] = json!(0);
        assert!(check_raw_profile(&case, &run(format!("gomad-wasm-profile-v1 {zero}\n"))).is_err());
    }
}

pub(super) fn check_literals(file: &Path, actual: &Value) -> Result<(), String> {
    let name = file.file_stem().unwrap().to_str().unwrap();
    if name.starts_with("feature_") && actual["exit_code"] != 0 {
        return Err(format!("{name}: helper process must exit successfully"));
    }
    let frames = actual["frames"]
        .as_array()
        .ok_or("feature frames missing")?;
    let result = frames.last().ok_or("feature result missing")?;
    let profile = &actual["diagnostic_profile"];
    let require = |ok, message: &str| {
        if ok {
            Ok(())
        } else {
            Err(format!("{name}: {message}"))
        }
    };
    if let Some(variant) = name.strip_prefix("feature_hooks_") {
        let (terminal, remote, now) = match variant {
            "exit" => ("exit", 1, 4),
            "invalid" => ("invalid", 1, 4),
            "trap" => ("trap", 0, 1),
            _ => return Err("unknown feature hooks case".into()),
        };
        require(
            result["termination"] == terminal && result["local_model"]["now"] == now,
            "terminal/clock tuple differs",
        )?;
        require(
            profile["schema"] == "gomad.wasm-profile/v2" && profile["abi_overlap_errors"] == 0,
            "profile schema/overlap differs",
        )?;
        for phase in EXECUTION_PHASES {
            require(
                profile["phases"][phase]["count"] == 1,
                "phase count differs",
            )?;
        }
        require(
            profile["abi"]["clock"]["count"] == 1
                && profile["abi"]["remote_callback"]["count"] == remote
                && profile["nested"]["remote_write"]["count"] == remote
                && profile["nested"]["remote_read_wait"]["count"] == remote,
            "exact ABI/nested counts differ",
        )?;
        let clock = b"\x47\x4d\x57\x54\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00";
        let mut expected: Vec<u8> = Vec::new();
        if terminal != "trap" {
            expected.extend(clock);
        }
        if terminal == "exit" {
            expected.extend(
                b"\x0d\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x03\x00\x00\x00",
            );
        }
        let mut transcript: Vec<u8> = Vec::new();
        for frame in frames.iter().filter(|frame| frame["type"] == "evidence") {
            require(
                frame["stream"] == "transcript",
                "unexpected evidence stream",
            )?;
            transcript.extend(
                STANDARD
                    .decode(
                        frame["data_base64"]
                            .as_str()
                            .ok_or("missing evidence bytes")?,
                    )
                    .map_err(|error| error.to_string())?,
            );
        }
        require(transcript == expected, "exact transcript bytes differ")?;
    } else if name == "feature_yield_poll" {
        require(
            result["termination"] == "exit"
                && profile["nested"]["encoding"]["count"] == 3
                && profile["abi"]["remote_callback"]["count"] == 0,
            "terminal encoding/callback counts differ",
        )?;
    } else if name == "feature_fuel_off" {
        require(
            result["termination"] == "exit"
                && result["fuel_remaining"] == 1
                && frames[0]["engine"]["configuration"]
                    .as_str()
                    .is_some_and(|value| value.ends_with("-fuel-off")),
            "offered fuel/configuration differs",
        )?;
    } else if name.starts_with("feature_guest_invalid_") {
        require(
            frames.len() == 1
                && result["termination"]
                    == if name == "feature_guest_invalid_unknown" {
                        "unsupported"
                    } else {
                        "invalid"
                    },
            "invalid profile must refuse before start",
        )?;
    }
    Ok(())
}

#[test]
fn feature_sidecars_cannot_replace_preserved_frames_evidence_or_profile_counts() {
    let file = corpus_dir().join("guard.json");
    let expected = json!({"frames":[{"type":"result","termination":"exit"}],
        "evidence":{"transcript":{"bytes":3,"sha256":"literal"}},
        "diagnostic_profile":{"abi":{"clock":{"count":1}}}});
    let case = json!({"request":{"type":"execute"},"expected":expected});
    check_semantic_baseline(&file, &case, &expected).unwrap();
    for changed in [
        {
            let mut value = expected.clone();
            value["frames"][0]["termination"] = json!("trap");
            value
        },
        {
            let mut value = expected.clone();
            value["evidence"]["transcript"]["sha256"] = json!("changed");
            value
        },
        {
            let mut value = expected;
            value["diagnostic_profile"]["abi"]["clock"]["count"] = json!(2);
            value
        },
    ] {
        assert!(check_semantic_baseline(&file, &case, &changed).is_err());
    }
}

#[test]
fn attribution_schema_extension_preserves_legacy_buckets() {
    let file = corpus_dir().join("attribution-guard.json");
    let expected = json!({"frames":[],"evidence":{},"diagnostic_profile":{
        "schema":"gomad.wasm-profile/v1","abi":{"clock":{"count":1}}}});
    let case = json!({"request":{"type":"execute"},"expected":expected});
    let mut actual = expected;
    actual["diagnostic_profile"]["schema"] = json!("gomad.wasm-profile/v2");
    actual["diagnostic_profile"]["abi_ops"] =
        json!({"wasi_snapshot_preview1.clock_time_get":{"count":1}});
    check_semantic_baseline(&file, &case, &actual).unwrap();
    actual["diagnostic_profile"]["abi"]["clock"]["count"] = json!(2);
    assert!(check_semantic_baseline(&file, &case, &actual).is_err());
}
