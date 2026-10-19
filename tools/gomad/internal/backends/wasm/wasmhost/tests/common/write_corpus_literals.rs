use super::{corpus_dir, read_case};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};

pub(super) fn check_write_literals(name: &str, actual: &Value) -> Result<(), String> {
    let Some(variant) = name.strip_prefix("typed_write_") else {
        return Ok(());
    };
    let frames = actual["frames"].as_array().ok_or("missing frames")?;
    let result = frames.last().ok_or("missing result")?;
    let calls: Vec<_> = frames
        .iter()
        .filter(|frame| frame["type"] == "call")
        .collect();
    let (termination, message, completed, output, callbacks) = if variant.starts_with("scatter_") {
        check_scatter_tuple_literals(name, actual)?;
        (
            "exit",
            "",
            2,
            if variant.ends_with("_plain") { 4152 } else { 0 },
            2,
        )
    } else if variant == "max_buffer" {
        let call = calls.first().ok_or("missing maximum-buffer callback")?;
        let bytes = STANDARD
            .decode(
                call["input"]["data_base64"]
                    .as_str()
                    .ok_or("missing write bytes")?,
            )
            .map_err(|error| error.to_string())?;
        if call["id"] != 1
            || call["op"] != "fd_pwrite"
            || call["input"]["fd"] != 13
            || call["local_state"] != json!({"advances":0,"now":1,"transcript_bytes":0})
            || bytes.len() != 4_194_304
            || bytes.iter().any(|byte| *byte != 42)
            || call["input"]["offset"].as_u64() != Some(u64::MAX)
        {
            return Err("maximum write buffer or unsigned offset differs".into());
        }
        ("exit", "", 1, 0, 1)
    } else if variant == "output_budget" {
        ("capacity", "output capacity", 0, 0, 0)
    } else if variant == "callback_budget" {
        ("capacity", "WASI capacity: callbacks", 1, 32, 0)
    } else if let Some(reason) = variant.strip_prefix("refused_") {
        match reason {
            "count_priority" => ("capacity", "iovec capacity", 0, 0, 0),
            "buffer_capacity_priority" => ("capacity", "ABI buffer capacity", 0, 0, 0),
            "aggregate_buffer" => ("capacity", "iovec buffer capacity", 0, 0, 0),
            "table_pointer" | "buffer_pointer" | "output_pointer" => {
                ("trap", "guest memory range out of bounds", 0, 0, 0)
            }
            _ => return Err(format!("unknown typed write refusal {name}")),
        }
    } else if let Some(reason) = variant.strip_prefix("reply_") {
        match reason {
            "written_range" => ("invalid", "reply written out of range", 1, 32, 1),
            "written_type" => ("invalid", "invalid unsigned reply field written", 1, 32, 1),
            "reply_id" => (
                "invalid",
                "invalid callback reply type, ID or output",
                0,
                32,
                1,
            ),
            "missing_model_state" => ("invalid", "missing remote model state", 0, 32, 1),
            "unsupported" => ("unsupported", "fd_write: outside profile", 1, 32, 1),
            "stop" => ("stop", "modeled", 0, 32, 1),
            "crash" => ("crash", "modeled", 0, 32, 1),
            "empty_transcript_chunk" => ("invalid", "remote transcript chunk capacity", 0, 32, 1),
            "errno" => ("exit", "", 1, 32, 1),
            _ => return Err(format!("unknown typed write reply {name}")),
        }
    } else {
        return Err(format!("missing typed write literals for {name}"));
    };
    if result["termination"] != termination
        || result["message"] != message
        || result["output_bytes"] != output
        || calls.len() != callbacks
    {
        return Err(format!(
            "{name}: typed write literal outcome differs: {result}"
        ));
    }
    if !variant.starts_with("scatter_remote_") && result["local_model"]["calls"] != completed {
        return Err(format!("{name}: completed call count differs: {result}"));
    }
    Ok(())
}

pub(super) fn check_scatter_tuple_literals(name: &str, actual: &Value) -> Result<(), String> {
    let exact = name.starts_with("admission_scatter_");
    let modeled = name.contains("_modeled_");
    let positioned = name.contains("_positioned");
    let fd = if if exact {
        name.ends_with("_maxfd")
    } else {
        positioned
    } {
        u64::from(u32::MAX)
    } else {
        1
    };
    let frames = actual["frames"]
        .as_array()
        .ok_or("missing scatter frames")?;
    let result = frames.last().ok_or("missing scatter result")?;
    let callbacks: Vec<_> = frames
        .iter()
        .filter(|frame| frame["type"] == "call")
        .collect();
    if callbacks.len() != 2
        || result["termination"] != "exit"
        || result["output_bytes"] != if fd == 1 { 4152 } else { 0 }
        || if modeled {
            result["local_model"]["calls"] != 2
                || result["local_model"]["now"] != 1
                || result["local_model"]["advances"] != 0
                || result["local_model"]["transcript_bytes"] != 0
        } else {
            !result["local_model"].is_null()
        }
    {
        return Err(format!("{name}: wrong scatter completion or output charge"));
    }
    let payload = [vec![0; 32], vec![255; 24], vec![42; 4096]].concat();
    for (index, callback) in callbacks.iter().enumerate() {
        let mut input = json!({"fd":fd, "data_base64":STANDARD.encode(if index == 0 { payload.as_slice() } else { &[][..] })});
        if positioned {
            input["offset"] = json!(if index == 0 || !exact { u64::MAX } else { 0 });
        }
        let mut expected = json!({"type":"call", "id":index as u64 + 1, "op":if positioned { "fd_pwrite" } else { "fd_write" }, "input":input});
        if modeled {
            expected["local_state"] = json!({"advances":0, "now":1, "transcript_bytes":0});
        }
        if **callback != expected {
            return Err(format!(
                "{name}: exact scatter callback {} differs",
                index + 1
            ));
        }
    }
    Ok(())
}

pub(super) fn scatter_tuple_literals_reject_changed_payload_and_callback_ids() {
    for kind in [
        "remote_plain",
        "remote_positioned",
        "modeled_plain",
        "modeled_positioned",
    ] {
        let name = format!("typed_write_scatter_{kind}");
        let (_, case) = read_case(&corpus_dir().join(format!("{name}.json"))).unwrap();
        check_write_literals(&name, &case["expected"]).unwrap();
        for field in ["payload", "id", "local_state", "empty_offset"] {
            if field == "local_state" && !kind.starts_with("modeled") {
                continue;
            }
            if field == "empty_offset" && !kind.ends_with("positioned") {
                continue;
            }
            let mut actual = case["expected"].clone();
            let callbacks: Vec<_> = actual["frames"]
                .as_array_mut()
                .unwrap()
                .iter_mut()
                .filter(|frame| frame["type"] == "call")
                .collect();
            let mut callbacks = callbacks;
            match field {
                "payload" => {
                    callbacks[0]["input"]["data_base64"] = json!(STANDARD.encode([1; 4152]));
                }
                "id" => callbacks[1]["id"] = json!(3),
                "local_state" => callbacks[0]["local_state"]["now"] = json!(2),
                "empty_offset" => callbacks[1]["input"]["offset"] = json!(0),
                _ => unreachable!(),
            }
            assert!(
                check_write_literals(&name, &actual).is_err(),
                "{name} accepted wrong {field}"
            );
        }
    }
}

pub(super) fn max_buffer_literals_reject_wrong_callback_identity_and_local_state() {
    let (_, case) = read_case(&corpus_dir().join("typed_write_max_buffer.json")).unwrap();
    check_write_literals("typed_write_max_buffer", &case["expected"]).unwrap();
    for field in ["id", "fd", "op", "local_state"] {
        let mut actual = case["expected"].clone();
        let call = actual["frames"]
            .as_array_mut()
            .unwrap()
            .iter_mut()
            .find(|frame| frame["type"] == "call")
            .unwrap();
        match field {
            "id" => call["id"] = json!(2),
            "fd" => call["input"]["fd"] = json!(12),
            "op" => call["op"] = json!("fd_write"),
            "local_state" => call["local_state"]["now"] = json!(2),
            _ => unreachable!(),
        }
        assert!(
            check_write_literals("typed_write_max_buffer", &actual).is_err(),
            "maximum buffer accepted wrong {field}"
        );
    }
}
