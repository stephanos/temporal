use super::{
    inspection, model_corpus_literals::check_model_literals,
    write_corpus_literals::check_scatter_tuple_literals,
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

pub(super) fn check_storage_context_input(name: &str, case: &Value) -> Result<(), String> {
    if !name.starts_with("admission_storage_context_") {
        return Ok(());
    }
    let declared = &case["request"]["storage"]["crash_context"];
    let mut bytes = Vec::new();
    let mut chunks = 0;
    let mut ends = 0;
    for frame in case["replies"][0]["send"]
        .as_array()
        .ok_or("missing storage input script")?
    {
        if frame["type"] == "storage_context_input" {
            if frame["sequence"] != chunks {
                return Err("storage chunk sequence drift".into());
            }
            bytes.extend(
                STANDARD
                    .decode(
                        frame["data_base64"]
                            .as_str()
                            .ok_or("missing context bytes")?,
                    )
                    .map_err(|error| error.to_string())?,
            );
            chunks += 1;
        } else if frame["type"] == "storage_context_inputs_end" {
            ends += 1;
        }
    }
    if bytes.len() != 1_259_101
        || chunks != 20
        || declared["bytes"] != 1_259_101
        || declared["chunks"] != 20
        || declared["sha256"] != format!("{:x}", Sha256::digest(&bytes))
        || ends != usize::from(!name.ends_with("missing_end"))
        || case["request"]["storage"]["model"]["limits"]["PendingOperations"]
            != if name.ends_with("399") { 399 } else { 400 }
    {
        return Err(format!("{name}: exact chunk/count declaration differs"));
    }
    let context: Value = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    let operations = context["data"]["Operations"]
        .as_array()
        .ok_or("missing context operations")?;
    if operations.len() != 400
        || context["data"]["PendingSHA256"] != format!("sha256:{}", "a".repeat(64))
    {
        return Err(format!("{name}: exact400-operation context differs"));
    }
    for (index, operation) in operations.iter().enumerate() {
        let id = index + 1;
        if operation
            != &json!({"ID":id,"Kind":"namespace","Dependencies":null,"Inode":id+1,
            "Offset":0,"Bytes":0,"PayloadSHA256":"","EffectSHA256":"","Path":format!("/{}","f".repeat(3000)),"Destination":""})
        {
            return Err(format!("{name}: operation{id} differs"));
        }
    }
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
pub(super) fn check_admission_literals(name: &str, actual: &Value) -> Result<(), String> {
    if !name.starts_with("admission_") {
        return Ok(());
    }
    let frames = actual["frames"].as_array().ok_or("missing frames")?;
    let result = frames.last().ok_or("missing result")?;
    let callbacks: Vec<_> = frames
        .iter()
        .filter(|frame| frame["type"] == "call")
        .collect();
    if name.starts_with("admission_storage_") {
        let message = match name {
            "admission_storage_model_identity" => "storage model identity mismatch",
            "admission_storage_image_descriptor" => "invalid storage image descriptor",
            "admission_storage_unknown_field" => "unknown field `extra`,",
            "admission_storage_image_capacity" => "WASI capacity: storage-image-bytes",
            "admission_storage_context_400" => {
                "storage crash context diverged from prior observation"
            }
            "admission_storage_context_399" => "WASI invalid: storage crash context inventory",
            "admission_storage_context_missing_end" => "unexpected protocol EOF",
            _ => return Err(format!("{name}: missing storage admission literal")),
        };
        let observed = result["message"]
            .as_str()
            .ok_or("missing storage refusal message")?;
        if result["termination"]
            != if name == "admission_storage_image_capacity" {
                "capacity"
            } else if name == "admission_storage_unknown_field" {
                "unsupported"
            } else {
                "invalid"
            }
            || !callbacks.is_empty()
            || if name.ends_with("unknown_field") {
                !observed.starts_with(message)
            } else {
                observed != message
            }
        {
            return Err(format!("{name}: wrong storage admission stage: {result}"));
        }
        let mut stdout = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == "stdout" {
                stdout.extend(
                    STANDARD
                        .decode(
                            frame["data_base64"]
                                .as_str()
                                .ok_or("missing stdout bytes")?,
                        )
                        .map_err(|error| error.to_string())?,
                );
            }
        }
        if stdout
            != if name.ends_with("context_400") {
                b"x".as_slice()
            } else {
                &[][..]
            }
        {
            return Err(format!("{name}: wrong execution marker: {stdout:?}"));
        }
        if name == "admission_storage_image_capacity" && result["local_model"]["calls"] != 800 {
            return Err(format!(
                "{name}: not all400 opens/closes completed: {result}"
            ));
        }
        if !name.starts_with("admission_storage_context_")
            && name != "admission_storage_image_capacity"
            && frames.iter().any(|frame| frame["type"] == "started")
        {
            return Err(format!("{name}: descriptor error occurred after start"));
        }
    } else if name == "admission_protocol_namespace_without_owner" {
        if result["termination"] != "invalid"
            || result["message"] != "namespace requires local model ownership"
            || !callbacks.is_empty()
            || frames.iter().any(|frame| frame["type"] == "started")
        {
            return Err(format!(
                "{name}: namespace ownership was not refused before start"
            ));
        }
    } else if name == "admission_raw_write_then_pwrite" {
        if result["termination"] != "exit"
            || result["output_bytes"] != 0
            || !result["local_model"].is_null()
            || callbacks.len() != 2
            || callbacks[0]
                != &json!({"type":"call","id":1,"op":"fd_write","input":{"fd":13,"data_base64":"AP8="}})
            || callbacks[1]
                != &json!({"type":"call","id":2,"op":"fd_pwrite","input":{"fd":13,"data_base64":"AP8=","offset":u64::MAX}})
        {
            return Err(format!(
                "{name}: exact model-free raw callback sequence differs: {result}"
            ));
        }
    } else if let Some(carriage) = name.strip_prefix("admission_model_input_") {
        let (message, prefix) = match carriage {
            "missing_nul" => ("invalid local model input chunk length", false),
            "sequence_gap" | "undeclared_arguments" => ("invalid local model input chunk", false),
            "truncated_missing_end" | "truncated_no_inputs" => ("unexpected protocol EOF", false),
            "unknown_field" => (
                "unknown field `unknown`, expected one of `type`, `stream`, `sequence`, `data_base64`",
                true,
            ),
            _ => return Err(format!("{name}: unknown input carriage")),
        };
        let actual = result["message"].as_str().ok_or("missing input refusal")?;
        if result["termination"] != "invalid"
            || !callbacks.is_empty()
            || frames.iter().any(|frame| frame["type"] == "started")
            || if prefix {
                !actual.starts_with(message)
            } else {
                actual != message
            }
        {
            return Err(format!("{name}: wrong input framing refusal: {result}"));
        }
    } else if matches!(
        name,
        "admission_exact_cpu_max_id"
            | "admission_exact_monotonic_max_precision"
            | "admission_exact_cpu_two"
            | "admission_exact_realtime_zero"
    ) {
        let cpu = matches!(
            name,
            "admission_exact_cpu_max_id" | "admission_exact_cpu_two"
        );
        let clock_id = if name == "admission_exact_cpu_max_id" {
            u32::MAX
        } else if cpu {
            2
        } else {
            u32::from(name != "admission_exact_realtime_zero")
        };
        let precision = if name == "admission_exact_monotonic_max_precision" {
            u64::MAX
        } else {
            0
        };
        let metadata = &result["local_model"];
        if !callbacks.is_empty()
            || result["termination"] != if cpu { "unsupported" } else { "exit" }
            || result["message"]
                != if cpu {
                    "clock_time_get: CPU clocks outside stock WASI profile"
                } else {
                    ""
                }
            || metadata["now"] != if cpu { 1 } else { 4 }
            || metadata["advances"] != 0
            || metadata["calls"] != 7
        {
            return Err(format!(
                "{name}: full-width clock outcome differs: {result}"
            ));
        }
        let mut literal = b"GMWT\x01\0\0\0".to_vec();
        for id in 1..=6u64 {
            literal.extend_from_slice(&[2, 0, 0, 0]);
            literal.extend_from_slice(&id.to_le_bytes());
            literal.extend_from_slice(&0u32.to_le_bytes());
        }
        literal.extend_from_slice(&[if cpu { 1 } else { 33 }, u8::from(cpu), 0, 0]);
        literal.extend_from_slice(&7u64.to_le_bytes());
        literal.extend_from_slice(&(if cpu { 53u32 } else { 28 }).to_le_bytes());
        literal.extend_from_slice(&clock_id.to_le_bytes());
        literal.extend_from_slice(&precision.to_le_bytes());
        if cpu {
            literal.extend_from_slice(b"\x25\0\0\0CPU clocks outside stock WASI profile");
        } else {
            literal.extend_from_slice(&1u64.to_le_bytes());
            literal.extend_from_slice(&1u64.to_le_bytes());
        }
        let mut emitted = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == "transcript" {
                emitted.extend(
                    STANDARD
                        .decode(
                            frame["data_base64"]
                                .as_str()
                                .ok_or("missing transcript bytes")?,
                        )
                        .map_err(|error| error.to_string())?,
                );
            }
        }
        if emitted != literal
            || metadata["transcript_bytes"] != literal.len() as u64
            || metadata["transcript_sha256"] != format!("{:x}", Sha256::digest(&literal))
        {
            return Err(format!("{name}: full-width clock literal differs"));
        }
    } else if name.starts_with("admission_exact_") {
        let clock = name == "admission_exact_clock_remote_tuple";
        let refused = name == "admission_exact_remote_json_short";
        let id = if clock { 3 } else { 1 };
        let raw = format!(
            "{{\"type\":\"call\",\"id\":{id},\"op\":\"fd_sync\",\"input\":{{\"fd\":3}}}}\n{{\"type\":\"reply\",\"id\":{id},\"errno\":0,\"output\":{{}}}}\n"
        );
        let metadata = &result["local_model"];
        if callbacks.len() != 1
            || callbacks[0]["id"] != id
            || callbacks[0]["op"] != "fd_sync"
            || callbacks[0]["input"] != json!({"fd":3})
            || callbacks[0]["local_state"]
                != if clock {
                    json!({"now":4,"advances":0,"transcript_bytes":52})
                } else {
                    json!({"now":1,"advances":0,"transcript_bytes":0})
                }
            || result["termination"]
                != if clock {
                    "unsupported"
                } else if refused {
                    "capacity"
                } else {
                    "exit"
                }
            || result["message"]
                != if clock {
                    "clock_time_get: CPU clocks outside stock WASI profile"
                } else if refused {
                    "WASI capacity: transcript-bytes"
                } else {
                    ""
                }
            || metadata["calls"] != if clock { 5 } else { i32::from(!refused) }
            || metadata["now"]
                != if clock {
                    23
                } else if refused {
                    1
                } else {
                    20
                }
            || metadata["advances"] != if refused { 0 } else { 3 }
            || result["output_bytes"] != 0
        {
            return Err(format!(
                "{name}: exact clock/raw-splice tuple differs: {result}"
            ));
        }
        let mut literal = Vec::new();
        if clock {
            literal.extend_from_slice(b"GMWT\x01\0\0\0");
            let run = |bytes: &mut Vec<u8>, id: u64, clock: u32, precision: u64, now: u64| {
                bytes.extend_from_slice(&[33, 0, 0, 0]);
                bytes.extend_from_slice(&id.to_le_bytes());
                bytes.extend_from_slice(&28u32.to_le_bytes());
                bytes.extend_from_slice(&clock.to_le_bytes());
                bytes.extend_from_slice(&precision.to_le_bytes());
                bytes.extend_from_slice(&now.to_le_bytes());
                bytes.extend_from_slice(&1u64.to_le_bytes());
            };
            run(&mut literal, 2, 0, u64::MAX, 1);
            literal.extend_from_slice(raw.as_bytes());
            run(&mut literal, 4, 1, 0, 20);
            literal.extend_from_slice(&[1, 1, 0, 0]);
            literal.extend_from_slice(&5u64.to_le_bytes());
            literal.extend_from_slice(&53u32.to_le_bytes());
            literal.extend_from_slice(&2u32.to_le_bytes());
            literal.extend_from_slice(&0u64.to_le_bytes());
            literal.extend_from_slice(b"\x25\0\0\0CPU clocks outside stock WASI profile");
        } else if !refused {
            literal.extend_from_slice(raw.as_bytes());
        }
        let mut emitted = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == "transcript" {
                emitted.extend(
                    STANDARD
                        .decode(
                            frame["data_base64"]
                                .as_str()
                                .ok_or("missing transcript bytes")?,
                        )
                        .map_err(|error| error.to_string())?,
                );
            }
        }
        if emitted != literal
            || metadata["transcript_bytes"] != literal.len() as u64
            || metadata["transcript_sha256"] != format!("{:x}", Sha256::digest(&literal))
        {
            return Err(format!(
                "{name}: raw JSON splice/canonical clock records differ"
            ));
        }
    } else if name.starts_with("admission_debug_") {
        let message = if name == "admission_debug_invalid_frame" {
            "invalid debug exchange frame"
        } else {
            "inspection unsupported without helper-owned runtime"
        };
        if result["termination"] != "debug_error"
            || result["message"] != message
            || !callbacks.is_empty()
            || !result["local_model"].is_null()
        {
            return Err(format!("{name}: wrong public debug refusal: {result}"));
        }
    } else if name.starts_with("admission_remote_") {
        let invalid = name == "admission_remote_invalid_clock";
        let metadata = &result["local_model"];
        if callbacks.len() != 1
            || result["termination"] != if invalid { "invalid" } else { "exit" }
            || result["message"]
                != if invalid {
                    "invalid remote clock state"
                } else {
                    ""
                }
            || metadata["now"] != if invalid { 1 } else { 7 }
            || metadata["advances"] != if invalid { 0 } else { 3 }
            || metadata["calls"] != u64::from(!invalid)
            || metadata["transcript_bytes"] != if invalid { 0 } else { 2 }
            || result["output_bytes"] != 0
        {
            return Err(format!(
                "{name}: wrong remote merge/refusal state: {result}"
            ));
        }
        let input = &callbacks[0]["input"];
        let bytes = STANDARD
            .decode(input["data_base64"].as_str().ok_or("missing write bytes")?)
            .map_err(|error| error.to_string())?;
        let expected = if invalid {
            vec![]
        } else if name.ends_with("_two") {
            vec![0, 255]
        } else {
            vec![0; 32]
        };
        let plain = name == "admission_remote_inline_write_two";
        if input["fd"] != if plain { u64::from(u32::MAX) } else { 13 }
            || callbacks[0]["op"] != if plain { "fd_write" } else { "fd_pwrite" }
            || callbacks[0]["id"] != 1
            || callbacks[0]["local_state"] != json!({"now":1,"advances":0,"transcript_bytes":0})
            || if plain {
                !input["offset"].is_null()
            } else {
                input["offset"].as_u64()
                    != Some(if name == "admission_remote_chunk_write32_offset_zero" {
                        0
                    } else {
                        u64::MAX
                    })
            }
            || bytes != expected
        {
            return Err(format!("{name}: wrong pwrite bytes or unsigned offset"));
        }
        let mut emitted = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == "transcript" {
                emitted.extend(
                    STANDARD
                        .decode(frame["data_base64"].as_str().unwrap())
                        .map_err(|error| error.to_string())?,
                );
            }
        }
        if emitted != if invalid { &[][..] } else { &[1, 2][..] }
            || metadata["transcript_sha256"] != format!("{:x}", Sha256::digest(&emitted))
        {
            return Err(format!("{name}: two-byte remote transcript differs"));
        }
    } else if name == "admission_entropy_two_full_counters" {
        check_model_literals("model_entropy_two_full_counters", actual)?;
    } else if name == "admission_runtime_forced_rank_full_answer" {
        check_model_literals("model_runtime_forced_rank", actual)?;
    } else if name == "admission_dormant_predecision_prefix" {
        let metadata = &result["local_model"];
        if result["termination"] != "exit"
            || !result["divergence"].is_null()
            || !callbacks.is_empty()
            || metadata["calls"] != 1
            || metadata["configured"] != true
            || metadata["records"] != 0
            || metadata["cursor"] != 0
            || metadata["finished"] != false
            || metadata["inspection_sealed"] != false
            || metadata["transcript_bytes"] != 0
            || metadata["now"] != 1
            || metadata["advances"] != 0
        {
            return Err(format!(
                "{name}: dormant debug changed the immediate prefix state: {result}"
            ));
        }
    } else if name.starts_with("admission_clock_capacity_") {
        let primed = name == "admission_clock_capacity_callbacks_step_three";
        let overflow = name == "admission_clock_capacity_overflow_step_three";
        let message = if primed {
            "WASI capacity: callbacks"
        } else if overflow {
            "WASI capacity: clock-nanoseconds"
        } else {
            "WASI capacity: transcript-bytes"
        };
        let now = if primed {
            4u64
        } else if overflow {
            9_223_372_036_854_774_807
        } else {
            1
        };
        let metadata = &result["local_model"];
        let mut transcript = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == "transcript" {
                transcript.extend(
                    STANDARD
                        .decode(
                            frame["data_base64"]
                                .as_str()
                                .ok_or("missing clock evidence")?,
                        )
                        .map_err(|error| error.to_string())?,
                );
            }
        }
        let expected = if primed {
            b"GMWT\x01\0\0\0\x21\0\0\0\x01\0\0\0\0\0\0\0\x1c\0\0\0\x01\0\0\0\0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0".as_slice()
        } else {
            &[][..]
        };
        if result["termination"] != "capacity"
            || result["message"] != message
            || !callbacks.is_empty()
            || result["output_bytes"] != 0
            || metadata["calls"] != u64::from(primed)
            || metadata["now"] != now
            || metadata["advances"] != 0
            || metadata["transcript_bytes"] != expected.len() as u64
            || transcript != expected
            || metadata["transcript_sha256"] != format!("{:x}", Sha256::digest(expected))
        {
            return Err(format!(
                "{name}: original step-three clock refusal state differs: {result}"
            ));
        }
    } else if name.starts_with("admission_reference_decoder_")
        || name.starts_with("admission_shared_decoder_")
    {
        let shared = name.starts_with("admission_shared_decoder_");
        let unsupported = name.ends_with("_unsupported");
        let stop = name.ends_with("_stop");
        let message = if name.ends_with("_wrong_id") {
            "invalid callback reply type, ID or output"
        } else if name.ends_with("_missing_state") {
            "missing remote model state"
        } else if name.ends_with("_invalid_clock") {
            "invalid remote clock state"
        } else if unsupported {
            "fd_pwrite: outside profile"
        } else if stop {
            "modeled"
        } else {
            "remote transcript chunk capacity"
        };
        let expected = json!({"type":"call","id":1,"op":"fd_pwrite","input":{"fd":13,"offset":if shared {u64::MAX}else{0},"data_base64":if shared{"AP8="}else{""}},"local_state":{"advances":0,"now":1,"transcript_bytes":0}});
        let metadata = &result["local_model"];
        if callbacks.len() != 1
            || callbacks[0] != &expected
            || result["termination"]
                != if unsupported {
                    "unsupported"
                } else if stop {
                    "stop"
                } else {
                    "invalid"
                }
            || result["message"] != message
            || result["output_bytes"] != 0
            || metadata["calls"] != u64::from(unsupported)
            || metadata["now"] != if unsupported { 7 } else { 1 }
            || metadata["advances"] != if unsupported { 3 } else { 0 }
            || metadata["transcript_bytes"] != 0
            || metadata["transcript_sha256"] != format!("{:x}", Sha256::digest(b""))
        {
            return Err(format!(
                "{name}: exact original decoder tuple or atomic state differs: {result}"
            ));
        }
    } else if name.starts_with("admission_scatter_") {
        check_scatter_tuple_literals(name, actual)?;
    } else if name == "admission_mode_one_inspection" {
        if result["termination"] != "invalid"
            || result["message"] != "invalid local model configuration"
            || !callbacks.is_empty()
            || frames.iter().any(|frame| frame["type"] == "started")
        {
            return Err(format!(
                "{name}: inspection mode was not refused before start: {result}"
            ));
        }
    } else if name == "admission_handoff_boundary_documents" {
        if cfg!(feature = "diagnostic-fuel-off") || cfg!(feature = "guest-profile") {
            if actual["exit_code"] != 0
                || actual["stderr"] != ""
                || actual["evidence"] != json!({})
                || actual["termination"] != "unsupported"
                || frames.len() != 1
                || result
                    != &json!({"type":"cluster_result","instances":[],
                        "message":if cfg!(feature = "diagnostic-fuel-off") {"diagnostic fuel-off helper builds do not run clusters"} else {"diagnostic helper builds do not run clusters"},
                        "termination":"unsupported","virtual_elapsed_nanos":0})
            {
                return Err(format!(
                    "{name}: exact diagnostic handoff refusal differs: {actual}"
                ));
            }
            return Ok(());
        }
        if result["type"] != "cluster_result"
            || result["termination"] != "exit"
            || !frames.iter().any(|frame| {
                frame["type"] == "result"
                    && frame["instance"] == "a-terminal"
                    && frame["termination"] == "exit"
            })
        {
            let context = serde_json::to_string(&json!({
                "exit_code":actual["exit_code"], "stderr":actual["stderr"],
                "last_frame":result, "observed_projected_record":actual,
            }))
            .map_err(|error| format!("{name}: handoff failure context: {error}"))?;
            let bounded: String = context.chars().take(12 * 1024).collect();
            return Err(format!(
                "{name}: handoff prefix did not finish; observed_projected_record={bounded}"
            ));
        }
        let stream = |kind: &str, owner: Option<&str>| -> Result<Vec<u8>, String> {
            let mut bytes = Vec::new();
            for frame in frames {
                if frame["type"] == "evidence"
                    && frame["stream"] == kind
                    && frame["instance"].as_str() == owner
                {
                    bytes.extend(
                        STANDARD
                            .decode(
                                frame["data_base64"]
                                    .as_str()
                                    .ok_or("missing evidence bytes")?,
                            )
                            .map_err(|error| error.to_string())?,
                    );
                }
            }
            Ok(bytes)
        };
        let world = stream("world", None)?;
        let network = stream("network", Some("a-terminal"))?;
        let evidence: Value = serde_json::from_slice(&stream("inspection", None)?)
            .map_err(|error| error.to_string())?;
        let boundaries = inspection::boundaries(&evidence);
        if boundaries.len() != 1 || world.is_empty() || network.is_empty() {
            return Err(format!("{name}: missing exact boundary documents"));
        }
        let boundary = &boundaries[0];
        let cut = boundary["cuts"]
            .as_array()
            .and_then(|cuts| {
                cuts.iter()
                    .find(|cut| cut["handle"]["node"] == "a-terminal")
            })
            .ok_or("missing handoff network cut")?;
        let commitment = |domain: &str, bytes: &[u8]| -> Vec<u8> {
            let mut hash = Sha256::new();
            if domain == "world" {
                hash.update(b"gomad.inspection-world-sha256/v2\0");
            } else if domain == "network" {
                hash.update(b"gomad.inspection-state-sha256/v2/network\0");
            } else {
                hash.update(b"gomad.inspection-state-sha256/v1/");
                hash.update(domain.as_bytes());
                hash.update([0]);
            }
            hash.update((bytes.len() as u64).to_le_bytes());
            hash.update(bytes);
            hash.finalize().to_vec()
        };
        if world.len() < 208 || &world[..8] != b"GMWSNAP1" {
            return Err(format!("{name}: missing canonical World header"));
        }
        // Reconstruct v2 material independently from the emitted full wire.
        let mut chain = b"gomad/world/inspection-chain/v2\0".to_vec();
        chain.extend_from_slice(&world[8..104]);
        chain.extend_from_slice(&(world.len() as u64).to_be_bytes());
        chain.extend_from_slice(&64u64.to_be_bytes());
        for byte in &world[120..152] {
            chain.extend_from_slice(format!("{byte:02x}").as_bytes());
        }
        let mut combined = Sha256::digest(chain).to_vec();
        combined.push(1);
        let network_material = super::network_inspection::material(&network)?;
        combined.extend_from_slice(&commitment("network", &network_material));
        let recorded = STANDARD
            .decode(
                evidence["world"]
                    .as_str()
                    .ok_or("missing boundary World stream")?,
            )
            .map_err(|error| error.to_string())?;
        let length = usize::try_from(
            boundary["world_bytes"]
                .as_u64()
                .ok_or("missing World prefix length")?,
        )
        .map_err(|_| "World prefix exceeds host width")?;
        if length < 40
            || length != recorded.len()
            || recorded[length - 40..length - 32] != 32u64.to_le_bytes()
            || recorded[length - 32..] != commitment("world", &combined)
            || cut["network_state"] != STANDARD.encode(commitment("network", &network_material))
            || cut["network_bytes"] != 0
        {
            return Err(format!(
                "{name}: handoff World/network commitment does not bind the emitted documents: world_prefix_bytes={length}/{} current_network_bytes={} retained_network_bytes={} network_matches={} world_matches={}",
                recorded.len(),
                network.len(),
                cut["network_bytes"],
                cut["network_state"] == STANDARD.encode(commitment("network", &network_material)),
                length >= 32
                    && recorded.get(length - 32..length)
                        == Some(commitment("world", &combined).as_slice())
            ));
        }
    } else if name == "admission_terminal_network_document" {
        if cfg!(feature = "diagnostic-fuel-off") || cfg!(feature = "guest-profile") {
            if frames.len() != 1 || result["termination"] != "unsupported" {
                return Err(format!(
                    "{name}: diagnostic cluster admission differs: {result}"
                ));
            }
            return Ok(());
        }
        if result["type"] != "cluster_result" || result["termination"] != "exit" {
            return Err(format!("{name}: cluster did not finish: {result}"));
        }
        for node in ["a-terminal", "b-boundary"] {
            if !frames.iter().any(|frame| {
                frame["type"] == "result"
                    && frame["instance"] == node
                    && frame["termination"] == "exit"
            }) {
                return Err(format!("{name}: missing successful instance {node}"));
            }
        }
        let stream = |kind: &str, owner: Option<&str>| -> Result<Vec<u8>, String> {
            let mut bytes = Vec::new();
            for frame in frames {
                if frame["type"] == "evidence"
                    && frame["stream"] == kind
                    && frame["instance"].as_str() == owner
                {
                    bytes.extend(
                        STANDARD
                            .decode(
                                frame["data_base64"]
                                    .as_str()
                                    .ok_or("missing evidence bytes")?,
                            )
                            .map_err(|error| error.to_string())?,
                    );
                }
            }
            Ok(bytes)
        };
        let terminal = stream("network", Some("a-terminal"))?;
        let evidence: Value = serde_json::from_slice(&stream("inspection", None)?)
            .map_err(|error| error.to_string())?;
        let boundaries = inspection::boundaries(&evidence);
        let cut = |owner: &str| -> Result<&Value, String> {
            boundaries
                .iter()
                .find(|boundary| boundary["handle"]["node"] == owner)
                .and_then(|boundary| boundary["cuts"].as_array())
                .and_then(|cuts| {
                    cuts.iter()
                        .find(|cut| cut["handle"]["node"] == "a-terminal")
                })
                .ok_or_else(|| format!("{name}: missing {owner} cut for a-terminal"))
        };
        let mut hash = Sha256::new();
        hash.update(b"gomad.inspection-state-sha256/v2/network\0");
        hash.update((terminal.len() as u64).to_le_bytes());
        hash.update(&terminal);
        let digest = STANDARD.encode(hash.finalize());
        let retained = evidence["incarnations"]
            .as_array()
            .and_then(|owners| {
                owners
                    .iter()
                    .find(|owner| owner["handle"]["node"] == "a-terminal")
            })
            .ok_or("missing retained terminal incarnation")?;
        if terminal.is_empty()
            || cut("b-boundary")?["network_bytes"] != terminal.len() as u64
            || cut("b-boundary")?["network_state"] != digest
            || cut("a-terminal")?["network_state"] == digest
            || retained["streams"]["network"] != STANDARD.encode(&terminal)
        {
            return Err(format!(
                "{name}: terminal network document or commitment was not retained: bytes={} expected_bytes={} state={} expected_state={} old_state={} retained_matches={}",
                cut("b-boundary")?["network_bytes"],
                terminal.len(),
                cut("b-boundary")?["network_state"],
                digest,
                cut("a-terminal")?["network_state"],
                retained["streams"]["network"] == STANDARD.encode(&terminal)
            ));
        }
    } else if name == "admission_namespace_shared_literal_hi" {
        let metadata = &result["local_model"];
        if result["termination"] != "exit"
            || !callbacks.is_empty()
            || metadata["calls"] != 1
            || metadata["now"] != 1
            || metadata["advances"] != 0
            || metadata["transcript_bytes"] != 38
        {
            return Err(format!(
                "{name}: shared literal stdin was not read: {result}"
            ));
        }
    } else if name.starts_with("admission_namespace_") {
        let descriptor = name.starts_with("admission_namespace_descriptor_");
        let message = if descriptor {
            "invalid namespace descriptor"
        } else if name == "admission_namespace_cwd_identity" {
            "invalid namespace image digest or length"
        } else {
            "invalid namespace image"
        };
        if result["termination"] != "invalid"
            || result["message"] != message
            || !callbacks.is_empty()
            || frames.iter().any(|frame| frame["type"] == "started") == descriptor
        {
            return Err(format!("{name}: wrong namespace admission stage: {result}"));
        }
    } else {
        return Err(format!("{name}: missing admission literals"));
    }
    Ok(())
}

pub(super) fn namespace_ownership_literal_rejects_wrong_admission_stage() {
    let name = "admission_protocol_namespace_without_owner";
    let mut actual = json!({"frames":[{"type":"result","termination":"invalid",
        "message":"namespace requires local model ownership"}]});
    check_admission_literals(name, &actual).unwrap();
    actual["frames"][0]["message"] = json!("invalid namespace image");
    assert!(check_admission_literals(name, &actual).is_err());
}

#[cfg(feature = "diagnostic-fuel-off")]
#[test]
fn diagnostic_handoff_refusal_rejects_changed_frame_process_or_evidence() {
    let name = "admission_handoff_boundary_documents";
    let actual = json!({"exit_code":0,"stderr":"","evidence":{},"termination":"unsupported",
        "frames":[{"type":"cluster_result","instances":[],
        "message":if cfg!(feature = "diagnostic-fuel-off") {"diagnostic fuel-off helper builds do not run clusters"} else {"diagnostic helper builds do not run clusters"},
        "termination":"unsupported","virtual_elapsed_nanos":0}]});
    check_admission_literals(name, &actual).unwrap();
    for (pointer, value) in [
        ("/exit_code", json!(1)),
        ("/stderr", json!("unexpected warning")),
        ("/evidence", json!({"world":{"bytes":1}})),
        ("/termination", json!("invalid")),
        ("/frames/0/type", json!("result")),
        ("/frames/0/instances", json!([{"id":"a-terminal"}])),
        ("/frames/0/message", json!("unrelated refusal")),
        ("/frames/0/termination", json!("invalid")),
        ("/frames/0/virtual_elapsed_nanos", json!(1)),
    ] {
        let mut changed = actual.clone();
        *changed.pointer_mut(pointer).unwrap() = value;
        assert!(
            check_admission_literals(name, &changed).is_err(),
            "accepted {pointer} drift"
        );
    }
    let mut extra = actual;
    extra["frames"]
        .as_array_mut()
        .unwrap()
        .push(json!({"type":"started"}));
    assert!(check_admission_literals(name, &extra).is_err());
}
