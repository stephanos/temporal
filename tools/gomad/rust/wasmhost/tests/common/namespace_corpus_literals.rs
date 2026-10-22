use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

const NAMES: [&str; 5] = [
    "namespace_fd_fdstat_set_flags_unsupported_digest_only_limit1",
    "namespace_path_open_unsupported_digest_only_limit1",
    "namespace_path_readlink_unsupported_digest_only_limit1",
    "namespace_two_max_paths_before_transcript",
    "namespace_output_ab_x_refuse_y_at3",
];
const RENAME_WAT: &str = "(module (import \"wasi_snapshot_preview1\" \"path_rename\" (func $op (param i32 i32 i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 128) (func (export \"_start\") i32.const 0 i32.const 120 i32.const 4194304 memory.fill i32.const 4194304 i32.const 121 i32.const 4194304 memory.fill i32.const 3 i32.const 0 i32.const 4194304 i32.const 3 i32.const 4194304 i32.const 4194304 call $op drop))";
const OUTPUT_WAT: &str = "(module (import \"wasi_snapshot_preview1\" \"fd_write\" (func $op (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 16) \"ab\") (data (i32.const 32) \"x\") (data (i32.const 48) \"y\") (func (export \"_start\") i32.const 64 i32.const 16 i32.store i32.const 68 i32.const 2 i32.store i32.const 1 i32.const 64 i32.const 1 i32.const 72 call $op if unreachable end i32.const 64 i32.const 32 i32.store i32.const 68 i32.const 1 i32.store i32.const 2 i32.const 64 i32.const 1 i32.const 72 call $op if unreachable end i32.const 64 i32.const 48 i32.store i32.const 1 i32.const 64 i32.const 1 i32.const 72 call $op drop))";

fn reference(name: &str) -> Result<Value, String> {
    super::read_case(&super::corpus_dir().join(format!("{name}.json"))).map(|(_, case)| case)
}

fn digest_reference(name: &str) -> Option<(&'static str, &'static str)> {
    match name {
        "namespace_fd_fdstat_set_flags_unsupported_digest_only_limit1" => Some((
            "model_unsupported_flags_75",
            "021fc1b524f3de06c7ec6f153fe35e759e0638c4344695691f245fdd37bbcb85",
        )),
        "namespace_path_open_unsupported_digest_only_limit1" => Some((
            "model_unsupported_open_96",
            "c84a28ccb59f8f6dfb31c41a395e457b92525f608c1b2375749af3e299dfda99",
        )),
        "namespace_path_readlink_unsupported_digest_only_limit1" => Some((
            "model_unsupported_readlink_92",
            "98b3235789b889d5047cb8d10be284a3f68f527bc6b9353000ac11cb03c56b86",
        )),
        _ => None,
    }
}

fn expected_input(name: &str) -> Result<Value, String> {
    if let Some((source, _)) = digest_reference(name) {
        let mut case = reference(source)?;
        case["request"]["local_model"]["digest_only"] = json!(true);
        case["request"]["local_model"]["limits"]["transcript_bytes"] = json!(1);
        return Ok(case);
    }
    let mut case = reference("model_namespace_chunk_before_remote")?;
    case["replies"]
        .as_array_mut()
        .ok_or("reference replies")?
        .truncate(1);
    case["reply_policy"] = json!("deny");
    match name {
        "namespace_two_max_paths_before_transcript" => {
            case["guest"] = json!({"wat":RENAME_WAT});
            case["request"]["memory_bytes"] = json!(128 * 65536);
            case["request"]["fuel"] = json!(100_000_000);
            case["request"]["local_model"]["limits"]["transcript_bytes"] = json!(1);
        }
        "namespace_output_ab_x_refuse_y_at3" => {
            case["guest"] = json!({"wat":OUTPUT_WAT});
            case["request"]["memory_bytes"] = json!(131_072);
            case["request"]["output_bytes"] = json!(3);
            let chunk = &mut case["replies"][0]["send"][1];
            let mut image = STANDARD
                .decode(chunk["data_base64"].as_str().ok_or("namespace image")?)
                .map_err(|error| error.to_string())?;
            image
                .get_mut(32..40)
                .ok_or("short namespace header")?
                .copy_from_slice(&3u64.to_le_bytes());
            chunk["data_base64"] = json!(STANDARD.encode(&image));
            case["request"]["namespace"]["sha256"] = json!(format!("{:x}", Sha256::digest(&image)));
        }
        _ => return Err(format!("unknown namespace literal case {name}")),
    }
    Ok(case)
}

fn streams(actual: &Value) -> Result<std::collections::BTreeMap<String, Vec<u8>>, String> {
    let mut streams = std::collections::BTreeMap::<String, Vec<u8>>::new();
    for frame in actual["frames"].as_array().ok_or("missing frames")? {
        match frame["type"].as_str() {
            Some("started" | "result") => {}
            Some("evidence") => {
                if frame.get("instance").is_some() {
                    return Err("unexpected namespace evidence instance".into());
                }
                let stream = frame["stream"].as_str().ok_or("missing stream")?;
                let bytes = STANDARD
                    .decode(
                        frame["data_base64"]
                            .as_str()
                            .ok_or("missing evidence bytes")?,
                    )
                    .map_err(|error| error.to_string())?;
                if bytes.is_empty() {
                    return Err("empty evidence frame".into());
                }
                streams.entry(stream.to_owned()).or_default().extend(bytes);
            }
            _ => return Err("unexpected callback/namespace frame".into()),
        }
    }
    Ok(streams)
}

fn completed_output_transcript() -> Vec<u8> {
    let mut bytes = b"GMWT\x01\0\0\0".to_vec();
    for (id, fd, data) in [(1u64, 1u32, b"ab".as_slice()), (2, 2, b"x".as_slice())] {
        bytes.extend_from_slice(&[8, 0, 0, 0]);
        bytes.extend_from_slice(&id.to_le_bytes());
        bytes.extend_from_slice(&(12 + u32::try_from(data.len()).unwrap()).to_le_bytes());
        bytes.extend_from_slice(&fd.to_le_bytes());
        bytes.extend_from_slice(&(u32::try_from(data.len()).unwrap()).to_le_bytes());
        bytes.extend_from_slice(data);
        bytes.extend_from_slice(&(u32::try_from(data.len()).unwrap()).to_le_bytes());
    }
    bytes
}

fn metadata(calls: u64, transcript: &[u8]) -> Value {
    json!({"advances":0,"calls":calls,"configured":false,"cursor":0,"elapsed":0,
        "finished":false,"inspection_sealed":false,"now":1,"peak":0,"records":0,
        "trace_sha256":format!("{:x}", Sha256::digest([])),"branching_records":0,
        "tape_records":0,"tape_sha256":"","next_expected":null,
        "terminal_base64":"","transcript_bytes":transcript.len(),
        "transcript_sha256":format!("{:x}", Sha256::digest(transcript))})
}

#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
pub fn check(name: &str, case: &Value, actual: &Value) -> Result<(), String> {
    if !NAMES.contains(&name) {
        return Ok(());
    }
    let expected = expected_input(name)?;
    for key in [
        "schema",
        "guest",
        "request",
        "inputs",
        "replies",
        "reply_policy",
        "close_input",
        "builds",
        "derived",
    ] {
        if case.get(key) != expected.get(key) {
            return Err(format!("{name}: literal input {key} differs"));
        }
    }
    let frames = actual["frames"]
        .as_array()
        .ok_or("missing namespace frames")?;
    let result = frames.last().ok_or("missing namespace result")?;
    if frames.first().map(|frame| &frame["type"]) != Some(&json!("started"))
        || frames
            .iter()
            .filter(|frame| frame["type"] == "started")
            .count()
            != 1
        || frames
            .iter()
            .filter(|frame| frame["type"] == "result")
            .count()
            != 1
        || result["type"] != "result"
        || actual["exit_code"] != 0
        || actual["stderr"] != ""
        || actual["termination"] != result["termination"]
    {
        return Err(format!("{name}: namespace completion shape differs"));
    }
    let streams = streams(actual)?;
    if let Some((source, sha)) = digest_reference(name) {
        let base = reference(source)?;
        let reference_result = base["expected"]["frames"]
            .as_array()
            .and_then(|frames| frames.last())
            .ok_or("missing retained reference result")?;
        if reference_result["local_model"]["transcript_sha256"] != sha
            || !streams.is_empty()
            || frames.len() != 2
        {
            return Err(format!(
                "{name}: digest-only retained bytes/reference SHA differs"
            ));
        }
        for key in ["termination", "message", "local_model", "output_bytes"] {
            if result.get(key) != reference_result.get(key) {
                return Err(format!("{name}: retained-reference terminal {key} differs"));
            }
        }
    } else {
        let (message, calls, transcript, output) =
            if name == "namespace_two_max_paths_before_transcript" {
                ("WASI capacity: callback-frame", 0, Vec::new(), 0)
            } else {
                (
                    "WASI capacity: offered-output-bytes",
                    2,
                    completed_output_transcript(),
                    3,
                )
            };
        if result["termination"] != "capacity"
            || result["message"] != message
            || result["output_bytes"] != output
            || result["local_model"] != metadata(calls, &transcript)
        {
            return Err(format!(
                "{name}: namespace boundary/count/transcript literal differs"
            ));
        }
        if calls == 0 {
            if !streams.is_empty() || frames.len() != 2 {
                return Err(format!("{name}: uncommitted namespace emitted evidence"));
            }
        } else {
            let want = std::collections::BTreeMap::from([
                ("stdout".to_owned(), b"ab".to_vec()),
                ("stderr".to_owned(), b"x".to_vec()),
                ("transcript".to_owned(), transcript),
            ]);
            if streams != want {
                return Err(format!("{name}: exact completed output/records differ"));
            }
        }
    }
    let expected_evidence = streams
        .into_iter()
        .map(|(stream, bytes)| {
            let mut value =
                json!({"bytes":bytes.len(),"sha256":format!("{:x}",Sha256::digest(&bytes))});
            if matches!(stream.as_str(), "stdout" | "stderr") {
                value["text"] = json!(String::from_utf8(bytes).map_err(|error| error.to_string())?);
            }
            Ok((stream, value))
        })
        .collect::<Result<serde_json::Map<String, Value>, String>>()?;
    if actual["evidence"] != Value::Object(expected_evidence) {
        return Err(format!(
            "{name}: evidence projection differs from literal frames"
        ));
    }
    Ok(())
}

#[test]
fn namespace_corpus_literal_guards_reject_input_and_result_drift() {
    let mut missed = Vec::new();
    for name in NAMES {
        let case = reference(name).unwrap();
        let mut actual = if let Some((source, _)) = digest_reference(name) {
            let mut actual = reference(source).unwrap()["expected"].clone();
            actual["frames"]
                .as_array_mut()
                .unwrap()
                .retain(|frame| frame["type"] != "evidence");
            actual["evidence"] = json!({});
            actual
        } else {
            let output = name == "namespace_output_ab_x_refuse_y_at3";
            let transcript = if output {
                completed_output_transcript()
            } else {
                Vec::new()
            };
            let mut frames = vec![json!({"type":"started"})];
            if output {
                for (stream, bytes) in [
                    ("stdout", b"ab".to_vec()),
                    ("stderr", b"x".to_vec()),
                    ("transcript", transcript.clone()),
                ] {
                    frames.push(json!({"type":"evidence","stream":stream,"data_base64":STANDARD.encode(bytes)}));
                }
            }
            frames.push(json!({"type":"result","termination":"capacity",
                "message":if output {"WASI capacity: offered-output-bytes"} else {"WASI capacity: callback-frame"},
                "output_bytes":if output {3} else {0},"local_model":metadata(if output {2} else {0},&transcript)}));
            json!({"frames":frames,"termination":"capacity","exit_code":0,"stderr":"","evidence":{}})
        };
        let mut evidence = serde_json::Map::new();
        for (stream, bytes) in streams(&actual).unwrap() {
            let mut summary =
                json!({"bytes":bytes.len(),"sha256":format!("{:x}",Sha256::digest(&bytes))});
            if matches!(stream.as_str(), "stdout" | "stderr") {
                summary["text"] = json!(String::from_utf8(bytes).unwrap());
            }
            evidence.insert(stream, summary);
        }
        actual["evidence"] = Value::Object(evidence);
        check(name, &case, &actual).unwrap();
        let mut wrong_input = case.clone();
        wrong_input["request"]["local_model"]["limits"]["transcript_bytes"] = json!(2);
        if check(name, &wrong_input, &actual).is_ok() {
            missed.push(format!("{name}: changed input accepted"));
        }
        actual["frames"].as_array_mut().unwrap().last_mut().unwrap()["local_model"]["transcript_sha256"] =
            json!("wrong");
        if check(name, &case, &actual).is_ok() {
            missed.push(format!("{name}: changed terminal SHA accepted"));
        }
    }
    assert!(missed.is_empty(), "{}", missed.join("\n"));
}
