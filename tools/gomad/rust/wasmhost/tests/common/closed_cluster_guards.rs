use super::*;

#[test]
fn manual_restart_raw_result_guards_reject_wrong_kind_and_missing_nodes() {
    let case = producer("closed_cluster_manual_turn_direct").unwrap();
    let frames = expected_frames(&case).unwrap().to_vec();
    let raw: Vec<_> = frames
        .iter()
        .map(|frame| {
            if frame["type"] == "result" && frame["incarnation"] == 2 {
                let mut rest = frame.clone();
                let fields = rest.as_object_mut().unwrap();
                fields.remove("instance");
                fields.remove("incarnation");
                fields.remove("type");
                format!(
                    "{{\"instance\":{},\"incarnation\":2,\"type\":\"result\",{}",
                    frame["instance"],
                    &rest.to_string()[1..]
                )
            } else {
                frame.to_string()
            }
        })
        .collect();
    let valid = Run {
        frames,
        raw,
        stderr: String::new(),
        exit_code: Some(0),
    };
    check(&case, &valid).unwrap();
    for node in ["a", "b"] {
        let index = valid
            .frames
            .iter()
            .position(|frame| {
                frame["instance"] == node && frame["incarnation"] == 2 && frame["type"] == "result"
            })
            .unwrap();
        let mut changed = valid.clone();
        changed.raw[index] =
            changed.raw[index].replacen("\"type\":\"result\"", "\"type\":\"call\"", 1);
        assert!(
            check(&case, &changed).is_err(),
            "wrong raw RESULT2 kind accepted for {node}"
        );
        let mut missing = valid.clone();
        missing.raw[index].clear();
        assert!(
            check(&case, &missing).is_err(),
            "missing raw RESULT2 accepted for {node}"
        );
    }
}

#[test]
fn network_bound_midpoint_rejects_changed_limit_from_committed_streams() {
    let free = producer("closed_cluster_network_bound_free").unwrap();
    let bounded = producer("closed_cluster_network_bound_bounded").unwrap();
    let frames = expected_frames(&free).unwrap().to_vec();
    let run = Run {
        raw: frames.iter().map(Value::to_string).collect(),
        frames,
        stderr: String::new(),
        exit_code: Some(0),
    };
    check_actual_network_bound(&run, &bounded).unwrap();
    let mut changed = bounded.clone();
    let limit = bounded["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"]
        .as_u64()
        .unwrap();
    changed["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"] =
        json!(limit + 1);
    assert!(
        check_actual_network_bound(&run, &changed).is_err(),
        "changed actual network midpoint limit accepted"
    );
}

fn name(case: &Value) -> &str {
    case["intent"]
        .as_str()
        .unwrap_or("")
        .split(':')
        .next()
        .unwrap_or("")
}
fn stream(frames: &[Value], node: Option<&str>, kind: &str) -> Result<Vec<u8>, String> {
    let mut bytes = Vec::new();
    for frame in frames {
        if frame["type"] == "evidence"
            && frame["stream"] == kind
            && frame["instance"].as_str() == node
        {
            bytes.extend(
                STANDARD
                    .decode(
                        frame["data_base64"]
                            .as_str()
                            .ok_or("evidence data missing")?,
                    )
                    .map_err(|e| e.to_string())?,
            );
        }
    }
    Ok(bytes)
}
fn producer(name: &str) -> Result<Value, String> {
    Ok(read_case(&corpus_dir().join(format!("{name}.json")))?.1)
}
fn expected_frames(case: &Value) -> Result<&[Value], String> {
    case["expected"]["frames"]
        .as_array()
        .map(Vec::as_slice)
        .ok_or("producer expected frames missing".into())
}
fn varint(bytes: &[u8], at: &mut usize) -> Result<u64, String> {
    let mut value = 0u64;
    for shift in (0..64).step_by(7) {
        let byte = *bytes.get(*at).ok_or("short network evidence varint")?;
        *at += 1;
        value |= u64::from(byte & 127) << shift;
        if byte & 128 == 0 {
            return Ok(value);
        }
    }
    Err("network evidence varint overflow".into())
}
fn log_bytes(bytes: &[u8]) -> Result<u64, String> {
    if bytes.get(..8) != Some(b"GMNETEV1") {
        return Err("network evidence magic differs".into());
    }
    let mut at = 8;
    if varint(bytes, &mut at)? != 1 {
        return Err("network evidence version differs".into());
    }
    let len = usize::try_from(varint(bytes, &mut at)?)
        .map_err(|_| "profile length exceeds host width")?;
    at = at.checked_add(len).ok_or("profile length overflow")?;
    varint(bytes, &mut at)
}
pub(super) fn derive(case: &Value, derived: &Value) -> Result<Option<Value>, String> {
    let Some(spec) = derived.get("cluster_packet") else {
        return Ok(None);
    };
    let fields = spec.as_object().ok_or("cluster_packet must be an object")?;
    if fields
        .keys()
        .any(|key| !matches!(key.as_str(), "assets" | "bound_from"))
    {
        return Err("unknown cluster_packet derived field".into());
    }
    let mut out = case.clone();
    if let Some(from) = spec["bound_from"].as_str() {
        let source = producer(from)?;
        let bytes = stream(expected_frames(&source)?, Some("1-bound"), "network")?;
        let log = log_bytes(&bytes)?;
        if log >= bytes.len() as u64 {
            return Err("network log is not smaller than final document".into());
        }
        out["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"] =
            json!(u64::midpoint(log, bytes.len() as u64));
    }
    Ok(Some(out))
}
pub(super) fn assets(case: &Value, path: &Path) -> Result<Value, String> {
    let mut out = case.clone();
    if case["request"]["type"] == "cluster_execute"
        && (cfg!(feature = "diagnostic-fuel-off") || cfg!(feature = "guest-profile"))
    {
        out["replies"] = json!([]);
        out["close_input"] = json!(true);
    }
    for (name, wat) in case["derived"]["cluster_packet"]["assets"]
        .as_object()
        .into_iter()
        .flatten()
    {
        if name.is_empty() || !name.bytes().all(|b| b.is_ascii_alphanumeric() || b == b'_') {
            return Err("invalid cluster asset name".into());
        }
        let wat = wat.as_str().ok_or("cluster asset must be WAT text")?;
        let asset = path
            .parent()
            .ok_or("guest has no parent")?
            .join(format!("{name}.wat"));
        std::fs::write(&asset, wat).map_err(|e| e.to_string())?;
        replace(
            &mut out,
            &format!("$cluster_asset.{name}.path"),
            asset.to_str().ok_or("asset path not UTF8")?,
        );
    }
    if name(case) == "closed_cluster_network_bound_bounded" {
        require(
            out["request"] == case["request"],
            "bounded request changed during asset preparation",
        )?;
    }
    Ok(out)
}
fn replace(value: &mut Value, key: &str, replacement: &str) {
    match value {
        Value::String(text) if text == key => *text = replacement.into(),
        Value::Array(items) => {
            for item in items {
                replace(item, key, replacement);
            }
        }
        Value::Object(fields) => {
            for item in fields.values_mut() {
                replace(item, key, replacement);
            }
        }
        _ => {}
    }
}
fn require(ok: bool, message: &str) -> Result<(), String> {
    if ok { Ok(()) } else { Err(message.into()) }
}
fn results(frames: &[Value]) -> Vec<&Value> {
    frames.iter().filter(|f| f["type"] == "result").collect()
}
fn result<'a>(frames: &'a [Value], node: &str) -> Result<&'a Value, String> {
    results(frames)
        .into_iter()
        .find(|f| f["instance"] == node)
        .ok_or_else(|| format!("missing result {node}"))
}
fn tuple(frame: &Value) -> (Value, Value, Value, Value) {
    (
        frame["instance"].clone(),
        frame["incarnation"].clone(),
        frame["termination"].clone(),
        frame["local_model"]["now"].clone(),
    )
}
#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
pub(super) fn check(case: &Value, run: &Run) -> Result<(), String> {
    let name = name(case);
    if !name.starts_with("closed_cluster_") {
        return Ok(());
    }
    let stderr = if cfg!(feature = "diagnostic-profile") {
        split_profile(&run.stderr)?.0
    } else {
        run.stderr.clone()
    };
    require(
        run.exit_code == Some(0) && stderr.is_empty(),
        "cluster helper process/stderr differs",
    )?;
    let last = run
        .frames
        .last()
        .ok_or("cluster helper emitted no frames")?;
    if name == "closed_cluster_diagnostic_refusal" {
        if cfg!(feature = "diagnostic-fuel-off") || cfg!(feature = "guest-profile") {
            return require(
                run.frames.len() == 1 && last["termination"] == "unsupported",
                "diagnostic cluster refusal stage differs",
            );
        }
        let calls: Vec<_> = run
            .frames
            .iter()
            .filter(|frame| frame["type"] == "call")
            .collect();
        require(
            calls.len() == 2
                && calls[0]["instance"] == "a"
                && calls[0]["id"] == 1
                && calls[0]["op"] == "fd_write"
                && calls[0]["input"] == json!({"fd":1,"data_base64":"aGVsbG8K"})
                && calls[1]["instance"] == "a"
                && calls[1]["id"] == 2
                && calls[1]["op"] == "clock_time_get"
                && calls[1]["input"] == json!({"clock_id":0,"precision":0}),
            "default diagnostic fixture exact write/clock callbacks differ",
        )?;
        let completed = result(&run.frames, "a")?;
        return require(
            completed["termination"] == "exit"
                && completed["peak_memory_bytes"] == 196_608
                && last["termination"] == "exit"
                && last["instances"] == json!([{"id":"a","termination":"exit"}]),
            "default diagnostic fixture must complete its three-page guest",
        );
    }
    if name == "closed_cluster_tagged_oversized" {
        require(
            result(&run.frames, "a")?["termination"] == "capacity",
            "tagged oversized reply must be instance capacity",
        )?;
        require(
            result(&run.frames, "b")?["termination"] == "exit" && last["termination"] == "exit",
            "tagged oversized line broke attribution or survivor progress",
        )?;
        return require(
            run.raw
                .iter()
                .any(|line| line.starts_with("{\"instance\":\"a\",")),
            "raw tagged output prefix absent",
        );
    }
    if name.starts_with("closed_cluster_single_") {
        let (terminal, peak) = match name.strip_prefix("closed_cluster_single_").unwrap() {
            "0-trap" => ("trap", 65536),
            "1-memory" => ("capacity", 65536),
            "2-fuel" => ("capacity", 196_608),
            "3-exit" => ("exit", 196_608),
            _ => return Err("unknown single isolation case".into()),
        };
        require(
            last["termination"] == terminal && last["peak_memory_bytes"] == peak,
            "single isolation terminal/peak tuple differs",
        )?;
    }
    if name.starts_with("closed_cluster_neighbors_") {
        require(
            last["type"] == "cluster_result" && last["termination"] == "exit",
            "mixed cluster terminal differs",
        )?;
        require(
            run.frames.iter().filter(|f| f["type"] == "started").count() == 1,
            "cluster must compile/emit started exactly once",
        )?;
        require(
            last["metrics"]["peak_rss_bytes"]
                .as_u64()
                .is_some_and(|n| n > 0),
            "cluster peakRSS must be positive",
        )?;
        let healthy = name.ends_with("_healthy");
        let want = if healthy {
            json!([{"id":"0-trap","termination":"exit"},{"id":"1-memory","termination":"exit"},{"id":"2-fuel","termination":"exit"},{"id":"3-exit","termination":"exit"}])
        } else {
            json!([{"id":"0-trap","termination":"trap"},{"id":"1-memory","termination":"capacity"},{"id":"2-fuel","termination":"capacity"},{"id":"3-exit","termination":"exit"}])
        };
        require(
            last["instances"] == want,
            "complete mixed cluster instance terminal vector differs",
        )?;
        if !healthy {
            let memory = result(&run.frames, "1-memory")?;
            require(
                memory["peak_memory_bytes"] == 65536
                    && memory["message"]
                        .as_str()
                        .is_some_and(|m| m.contains("guest linear memory capacity")),
                "memory instance peak/message differs",
            )?;
            require(
                result(&run.frames, "3-exit")?["peak_memory_bytes"] == 196_608,
                "survivor grown peak differs",
            )?;
            require(
                result(&run.frames, "2-fuel")?["message"]
                    .as_str()
                    .is_some_and(|m| m.contains("fuel")),
                "fuel refusal message differs",
            )?;
        }
        require(
            run.frames.iter().any(|f| {
                f["type"] == "evidence" && f["instance"] == "3-exit" && f["stream"] == "transcript"
            }),
            "survivor transcript evidence absent",
        )?;
    }
    if name.starts_with("closed_cluster_manual_turn_") {
        let restarted: Vec<_> = run
            .frames
            .iter()
            .filter(|f| f["type"] == "result" && f["incarnation"] == 2)
            .collect();
        require(
            restarted
                .iter()
                .map(|frame| frame["instance"].as_str())
                .collect::<Vec<_>>()
                == vec![Some("a"), Some("b")],
            "raw RESULT2 guard requires both restarted nodes",
        )?;
        for frame in restarted {
            let index = run
                .frames
                .iter()
                .position(|f| std::ptr::eq(f, frame))
                .unwrap();
            let prefix = format!(
                "{{\"instance\":{},\"incarnation\":2,\"type\":\"result\",",
                frame["instance"]
            );
            require(
                run.raw
                    .get(index)
                    .is_some_and(|line| line.starts_with(&prefix)),
                "raw incarnation2 result prefix differs",
            )?;
        }
        let want = vec![
            (json!("a"), Value::Null, json!("exit"), json!(2000)),
            (json!("b"), Value::Null, json!("exit"), json!(2000)),
            (json!("a"), json!(2), json!("exit"), json!(4000)),
            (json!("b"), json!(2), json!("exit"), json!(4000)),
        ];
        require(
            results(&run.frames)
                .iter()
                .map(|f| tuple(f))
                .collect::<Vec<_>>()
                == want,
            "manual recovery exact node/incarnation/termination/clock tuples differ",
        )?;
        let commands: Vec<_> = case["replies"]
            .as_array()
            .unwrap()
            .iter()
            .flat_map(|script| script["send"].as_array().unwrap())
            .filter(|f| f.get("id").is_some())
            .collect();
        let acks: Vec<_> = run
            .frames
            .iter()
            .filter(|f| f["type"] == "control_result")
            .collect();
        require(
            commands.len() == acks.len(),
            "manual controller acknowledgment count differs",
        )?;
        for (command, ack) in commands.iter().zip(acks) {
            let invalid =
                command["type"] == "finish" && command["id"] != commands.last().unwrap()["id"];
            require(
                ack["id"] == command["id"]
                    && ack["termination"] == if invalid { "invalid" } else { "" },
                "manual command id/termination ordering differs",
            )?;
            if command["type"] == "run" {
                let ack_at = run
                    .frames
                    .iter()
                    .position(|f| std::ptr::eq(f, ack))
                    .unwrap();
                let result_at = run
                    .frames
                    .iter()
                    .position(|f| {
                        f["type"] == "result"
                            && f["instance"] == command["node"]
                            && f["incarnation"].as_u64().unwrap_or(1)
                                == command["incarnation"].as_u64().unwrap()
                    })
                    .ok_or("manual run result missing")?;
                require(
                    ack_at < result_at,
                    "guest escaped controller acknowledgment turn",
                )?;
            }
        }
        require(
            last["termination"] == "exit",
            "manual final cluster terminal differs",
        )?;
        for node in ["a", "b"] {
            require(
                !stream(&run.frames, Some(node), "network")?.is_empty(),
                "manual recovery network evidence missing",
            )?;
        }
        require(
            !stream(&run.frames, None, "world")?.is_empty(),
            "manual recovery World evidence missing",
        )?;
    }
    if name == "closed_cluster_manual_inspection_record" {
        require(
            last["termination"] == "exit",
            "manual inspection producer must exit",
        )?;
        require(
            results(&run.frames)
                .iter()
                .map(|f| (f["instance"].clone(), f["termination"].clone()))
                .collect::<Vec<_>>()
                == vec![(json!("a"), json!("exit")), (json!("b"), json!("exit"))],
            "recording a/b results differ",
        )?;
        require(
            run.frames.iter().any(|f| f["type"] == "started"),
            "manual inspection started frame absent",
        )?;
        let acks: Vec<_> = run
            .frames
            .iter()
            .filter(|f| f["type"] == "control_result")
            .collect();
        require(
            acks.iter()
                .enumerate()
                .all(|(i, f)| f["id"] == i + 1 && f["termination"] == ""),
            "manual inspection control acknowledgments differ",
        )?;
    }
    if name == "closed_cluster_network_bound_free" {
        require(
            result(&run.frames, "1-bound")?["termination"] == "exit",
            "free instance must exit",
        )?;
    }
    if name == "closed_cluster_network_bound_bounded" {
        let frame = result(&run.frames, "1-bound")?;
        require(
            frame["termination"] == "capacity"
                && frame["message"] == "WASI capacity: network-v2-evidence-bytes"
                && frame.get("exit_code").is_none(),
            "exact final network evidence capacity tuple differs",
        )?;
        require(
            stream(&run.frames, Some("1-bound"), "network")?.is_empty(),
            "capacity instance emitted partial final network evidence",
        )?;
    }
    if name == "closed_cluster_trap_pending_peer" {
        require(
            result(&run.frames, "a")?["termination"] == "trap"
                && result(&run.frames, "b")?["termination"] == "exit"
                && last["termination"] == "exit",
            "trap/pending-peer terminal tuple differs",
        )?;
        let directory = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/broker_orders");
        for node in ["a", "b"] {
            let want = std::fs::read(
                directory.join(format!("finish-with-pending-peer-read.{node}.gmnetev1")),
            )
            .map_err(|e| e.to_string())?;
            require(
                stream(&run.frames, Some(node), "network")? == want,
                "trap pending-read network bytes differ from Go Finished",
            )?;
        }
        let scenario: Value = serde_json::from_slice(
            &std::fs::read(directory.join("finish-with-pending-peer-read.json"))
                .map_err(|e| e.to_string())?,
        )
        .map_err(|e| e.to_string())?;
        require(
            STANDARD.encode(stream(&run.frames, None, "world")?)
                == scenario["world_base64"]
                    .as_str()
                    .ok_or("Go World missing")?,
            "trap World bytes differ from Go broker",
        )?;
    }
    Ok(())
}
fn stripped(run: &Run, node: &str) -> Result<(Vec<Value>, Vec<String>), String> {
    require(
        run.frames.len() == run.raw.len(),
        "raw/frame capture length differs",
    )?;
    let prefix = format!("{{\"instance\":{node:?},");
    let mut frames = Vec::new();
    let mut lines = Vec::new();
    for (frame, line) in run.frames.iter().zip(&run.raw) {
        if frame["instance"] == node {
            require(
                line.starts_with(&prefix),
                "actual cluster raw attribution prefix differs",
            )?;
            let mut frame = frame.clone();
            frame.as_object_mut().unwrap().remove("instance");
            frames.push(frame);
            lines.push(format!("{{{}", &line[prefix.len()..]));
        }
    }
    Ok((frames, lines))
}
fn singles(frames: &[Value]) -> Vec<Value> {
    frames
        .iter()
        .filter(|f| f["type"] != "started")
        .map(|f| {
            let mut f = f.clone();
            f.as_object_mut().unwrap().remove("metrics");
            f
        })
        .collect()
}
fn same_non_result(cluster: &Run, node: &str, single: &Run) -> Result<(), String> {
    let (frames, lines) = stripped(cluster, node)?;
    require(
        frames == singles(&single.frames),
        "complete instance_frames single/cluster normalization differs",
    )?;
    let cluster_lines: Vec<_> = lines
        .iter()
        .filter(|line| !line.starts_with("{\"type\":\"result\""))
        .collect();
    let single_lines: Vec<_> = single
        .frames
        .iter()
        .zip(&single.raw)
        .filter(|(frame, _)| frame["type"] != "started" && frame["type"] != "result")
        .map(|(_, line)| line)
        .collect();
    require(
        !single_lines.is_empty(),
        "single/cluster raw equality guard is vacuous",
    )?;
    require(
        cluster_lines == single_lines,
        "actual captured non-result raw lines differ after attribution stripping",
    )
}
fn check_actual_network_bound(free: &Run, bounded: &Value) -> Result<(), String> {
    let bytes = stream(&free.frames, Some("1-bound"), "network")?;
    let log = log_bytes(&bytes)?;
    require(
        log < bytes.len() as u64,
        "actual network log is not smaller than final document",
    )?;
    require(
        bounded["request"]["instances"][1]["id"] == "1-bound"
            && bounded["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"]
                == json!(u64::midpoint(log, bytes.len() as u64)),
        "actual network midpoint differs from driven bounded request",
    )
}
#[cfg(not(any(
    feature = "diagnostic-profile",
    feature = "diagnostic-fuel-off",
    feature = "guest-profile"
)))]
fn load_run(name: &str) -> Result<Run, String> {
    let case = producer(name)?;
    run_case(&case)
}
#[test]
#[cfg(not(any(
    feature = "diagnostic-profile",
    feature = "diagnostic-fuel-off",
    feature = "guest-profile"
)))]
fn closed_cluster_cross_case_raw_and_evidence_guards() {
    let mixed = load_run("closed_cluster_neighbors_failing").unwrap();
    for node in ["0-trap", "1-memory", "2-fuel", "3-exit"] {
        let single = load_run(&format!("closed_cluster_single_{node}")).unwrap();
        same_non_result(&mixed, node, &single).unwrap();
    }
    let healthy = load_run("closed_cluster_neighbors_healthy").unwrap();
    assert_eq!(
        stripped(&mixed, "3-exit").unwrap().1,
        stripped(&healthy, "3-exit").unwrap().1,
        "actual survivor raw frames changed beside failed neighbors"
    );
    let direct = load_run("closed_cluster_manual_turn_direct").unwrap();
    let delayed = load_run("closed_cluster_manual_turn_delayed").unwrap();
    let recovery = |run: &Run, node: &str| {
        let frames: Vec<_> = run
            .frames
            .iter()
            .filter(|f| f["incarnation"] == 2)
            .cloned()
            .collect();
        stream(&frames, Some(node), "network").unwrap()
    };
    for node in ["a", "b"] {
        let left = recovery(&direct, node);
        let right = recovery(&delayed, node);
        assert_ne!(left, [] as [u8; 0]);
        assert_eq!(left, right, "manual recovery actual network bytes differ");
    }
    let world = |run: &Run| {
        run.frames
            .iter()
            .filter(|f| f["stream"] == "world")
            .cloned()
            .collect::<Vec<_>>()
    };
    assert_ne!(world(&direct), [] as [serde_json::Value; 0]);
    assert_eq!(
        world(&direct),
        world(&delayed),
        "manual recovery actual World frames differ"
    );
    let free = load_run("closed_cluster_network_bound_free").unwrap();
    let bounded_case = producer("closed_cluster_network_bound_bounded").unwrap();
    let bounded = run_case(&bounded_case).unwrap();
    check_actual_network_bound(&free, &bounded_case).unwrap();
    assert_eq!(
        stream(&free.frames, Some("2-free"), "network").unwrap(),
        stream(&bounded.frames, Some("2-free"), "network").unwrap(),
        "final document capacity touched later neighbor"
    );
}
#[test]
fn closed_cluster_raw_guard_rejects_changed_literal_and_empty_probe() {
    let single = Run {
        frames: vec![json!({"type":"call","id":1})],
        raw: vec!["{\"type\":\"call\",\"id\":1}".into()],
        stderr: String::new(),
        exit_code: Some(0),
    };
    let cluster = Run {
        frames: vec![json!({"instance":"a","type":"call","id":1})],
        raw: vec!["{\"instance\":\"a\",\"type\":\"call\",\"id\":1}".into()],
        stderr: String::new(),
        exit_code: Some(0),
    };
    same_non_result(&cluster, "a", &single).unwrap();
    let mut changed = cluster;
    changed.raw[0] = changed.raw[0].replace(":\"call\"", ": \"call\"");
    assert!(same_non_result(&changed, "a", &single).is_err());
    let empty = Run {
        frames: Vec::new(),
        raw: Vec::new(),
        stderr: String::new(),
        exit_code: Some(0),
    };
    assert!(same_non_result(&empty, "a", &empty).is_err());
    let wrong_clock_case = json!({"intent":"closed_cluster_manual_turn_direct: mutation probe"});
    let wrong_clock = Run {
        frames: vec![
            json!({"type":"result","instance":"a","termination":"exit","local_model":{"now":2001}}),
            json!({"type":"cluster_result","termination":"exit"}),
        ],
        raw: vec![],
        stderr: String::new(),
        exit_code: Some(0),
    };
    assert!(check(&wrong_clock_case, &wrong_clock).is_err());
}
