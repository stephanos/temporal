use super::case_files;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use std::path::PathBuf;

const REMOTE: &str = "residual_remote_yield";
const STORAGE: &str = "residual_storage_context_declared_overlimit";
const WAT: &str = r#"(module (import "wasi_snapshot_preview1" "sched_yield" (func $yield (result i32))) (memory (export "memory") 1) (func (export "_start") call $yield if unreachable end))"#;

pub fn input(name: &str, case: &Value) -> Result<(), String> {
    if name == REMOTE {
        if case["guest"]["wat"] != WAT
            || !case["request"]["local_model"].is_null()
            || !case["request"]["namespace"].is_null()
            || case["inputs"] != json!([])
            || case["replies"][1]
                != json!({"after":{"type":"call","op":"sched_yield","id":1},"send":[{"type":"reply","id":1,"errno":0,"output":{}}]})
            || case["replies"].as_array().map(Vec::len) != Some(2)
        {
            return Err("remote yield input/reply must be the exact unmodeled success".into());
        }
    } else if name == STORAGE {
        let send = case["replies"][0]["send"]
            .as_array()
            .ok_or("missing storage setup")?;
        if send.len() != 3
            || send[0]["type"] != "authorize"
            || send[1]["type"] != "namespace_input"
            || send[2] != json!({"type":"namespace_inputs_end"})
            || case["replies"].as_array().map(Vec::len) != Some(1)
            || case["close_input"] != "after_replies"
        {
            return Err(
                "overlimit case must send only authorization and namespace, zero context payload"
                    .into(),
            );
        }
        let image = STANDARD
            .decode(
                send[1]["data_base64"]
                    .as_str()
                    .ok_or("missing namespace image")?,
            )
            .map_err(|error| error.to_string())?;
        let field = image.get(24..32).ok_or("short namespace header")?;
        let filesystem_bytes = u64::from_le_bytes(field.try_into().unwrap());
        let declared = &case["request"]["storage"]["crash_context"];
        if filesystem_bytes != 1024
            || declared
                != &json!({"bytes":18_892_801,"chunks":289,"sha256":"0000000000000000000000000000000000000000000000000000000000000000"})
            || declared["bytes"].as_u64() != Some(18 * (filesystem_bytes + (1 << 20)) + 1)
        {
            return Err(
                "context declaration is not valid syntax at exact computed maximum+1".into(),
            );
        }
    }
    Ok(())
}

pub fn actual(name: &str, value: &Value) -> Result<(), String> {
    if name != REMOTE && name != STORAGE {
        return Ok(());
    }
    let frames = value["frames"]
        .as_array()
        .ok_or("missing residual frames")?;
    let result = frames.last().ok_or("missing residual result")?;
    if frames.first().map(|frame| frame["type"].as_str()) != Some(Some("started"))
        || result["type"] != "result"
        || frames
            .iter()
            .filter(|frame| frame["type"] == "started")
            .count()
            != 1
    {
        return Err("residual must have exactly one started and terminal result".into());
    }
    if name == REMOTE {
        if frames.len() != 3
            || frames[1]["type"] != "call"
            || frames[1]["op"] != "sched_yield"
            || frames[1]["id"] != 1
            || frames[1]["input"] != json!({})
            || result["termination"] != "exit"
            || result["exit_code"] != 0
            || result["message"] != ""
            || !result["local_model"].is_null()
        {
            return Err("remote yield must expose exact callback1/input{} then exit0".into());
        }
    } else if frames.len() != 2
        || result["termination"] != "invalid"
        || result["message"] != "WASI invalid: storage-context-bytes"
        || result["peak_memory_bytes"] != 0
        || result["fuel_remaining"] != 0
        || result["metrics"]["instantiate_nanos"] != 0
        || result["metrics"]["execute_nanos"] != 0
        || result["local_model"]["calls"] != 0
    {
        return Err("context must refuse exact byte bound after started, before instantiation/execution/callback".into());
    }
    Ok(())
}

#[test]
fn residual_helper_literals_reject_wrong_stage_and_callback_projection() {
    let remote = json!({"frames":[{"type":"started"},{"type":"call","op":"sched_yield","id":1,"input":{}},{"type":"result","termination":"exit","exit_code":0,"message":""}]});
    actual(REMOTE, &remote).unwrap();
    for field in ["op", "id", "input"] {
        let mut wrong = remote.clone();
        wrong["frames"][1][field] = match field {
            "op" => json!("clock_time_get"),
            "id" => json!(2),
            _ => json!({"unexpected":1}),
        };
        assert!(
            actual(REMOTE, &wrong).is_err(),
            "changed remote callback {field} accepted"
        );
    }
    let storage = json!({"frames":[{"type":"started"},{"type":"result","termination":"invalid","message":"WASI invalid: storage-context-bytes","peak_memory_bytes":0,"fuel_remaining":0,"metrics":{"instantiate_nanos":0,"execute_nanos":0},"local_model":{"calls":0}}]});
    actual(STORAGE, &storage).unwrap();
    let mut wrong = storage.clone();
    wrong["frames"][1]["message"] = json!("invalid storage crash context descriptor");
    assert!(
        actual(STORAGE, &wrong).is_err(),
        "wrong earlier storage stage accepted"
    );
    let mut wrong = storage;
    wrong["frames"].as_array_mut().unwrap().remove(0);
    assert!(
        actual(STORAGE, &wrong).is_err(),
        "impossible no-start byte-bound refusal accepted"
    );
    let directory =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/helper_transcript");
    for name in [REMOTE, STORAGE] {
        let case: Value =
            serde_json::from_slice(&std::fs::read(directory.join(format!("{name}.json"))).unwrap())
                .unwrap();
        input(name, &case).unwrap();
        let mut wrong = case.clone();
        if name == REMOTE {
            wrong["replies"][1]["send"][0]["errno"] = json!(1);
        } else {
            wrong["request"]["storage"]["crash_context"]["bytes"] = json!(18_892_800);
        }
        assert!(
            input(name, &wrong).is_err(),
            "changed {name} setup accepted"
        );
    }
}

/// Producers run before the cases whose inputs derive from their expected
/// blocks, so one regeneration pass refreshes both.
pub(super) fn residual_regeneration_files(
    files: Vec<PathBuf>,
    regenerate: bool,
    selected: Option<&str>,
) -> Result<Vec<PathBuf>, String> {
    let Some(selected) = selected else {
        return Ok(files);
    };
    if !regenerate {
        return Err("residual selection is only allowed during regeneration".into());
    }
    let names: Vec<_> = selected.split(',').collect();
    let expected = [
        "residual_remote_yield",
        "residual_storage_context_declared_overlimit",
    ];
    if names.len() != 2
        || names
            .iter()
            .any(|name| name.is_empty() || name.contains(['/', '\\']))
        || names
            .iter()
            .copied()
            .collect::<std::collections::BTreeSet<_>>()
            != expected.into_iter().collect()
    {
        return Err(
            "residual regeneration requires exactly the two approved distinct names".into(),
        );
    }
    let selected: Vec<_> = files
        .into_iter()
        .filter(|path| {
            path.file_stem()
                .and_then(|stem| stem.to_str())
                .is_some_and(|name| expected.contains(&name))
        })
        .collect();
    if selected.len() != 2 {
        return Err("approved residual input file is missing".into());
    }
    Ok(selected)
}

pub(super) fn residual_regeneration_selection_is_exact_and_generation_only() {
    let files = case_files();
    let expected: Vec<_> = files
        .iter()
        .filter(|path| {
            matches!(
                path.file_stem().unwrap().to_str().unwrap(),
                "residual_remote_yield" | "residual_storage_context_declared_overlimit"
            )
        })
        .cloned()
        .collect();
    assert_eq!(
        expected.len(),
        2,
        "both real newly authored inputs must exist"
    );
    for regenerate in [false, true] {
        assert_eq!(
            residual_regeneration_files(files.clone(), regenerate, None).unwrap(),
            files,
            "absent selection must retain ALL canonical cases"
        );
    }
    let names = "residual_remote_yield,residual_storage_context_declared_overlimit";
    assert_eq!(
        residual_regeneration_files(files.clone(), true, Some(names)).unwrap(),
        expected
    );
    for (regenerate, selected) in [
        (false, names),
        (false, ""),
        (true, ""),
        (true, "residual_remote_yield"),
        (true, "residual_remote_yield,residual_remote_yield"),
        (true, "residual_remote_yield,missing"),
        (
            true,
            "residual_remote_yield,../residual_storage_context_declared_overlimit",
        ),
        (
            true,
            "residual_remote_yield,residual_storage_context_declared_overlimit,",
        ),
    ] {
        assert!(
            residual_regeneration_files(files.clone(), regenerate, Some(selected)).is_err(),
            "invalid generation selection accepted: {selected:?}, regenerate={regenerate}"
        );
    }
    let missing: Vec<_> = files
        .into_iter()
        .filter(|path| path.file_stem().unwrap() != "residual_remote_yield")
        .collect();
    assert!(
        residual_regeneration_files(missing, true, Some(names)).is_err(),
        "missing real input was silently dropped"
    );
}
