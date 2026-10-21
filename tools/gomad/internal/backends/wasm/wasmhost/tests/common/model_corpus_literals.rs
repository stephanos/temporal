use super::{corpus_dir, read_case};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

/// Independent literals are checked before regeneration can approve a snapshot.
#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
pub(super) fn check_model_literals(name: &str, actual: &Value) -> Result<(), String> {
    let refusal = match name {
        "model_inputs_refused_environment_count" => {
            Some(("invalid local model string count", false))
        }
        "model_inputs_refused_invalid_utf8" => {
            Some(("invalid utf-8 sequence of 1 bytes from index 0", false))
        }
        "model_inputs_refused_end_wrong_type" => Some(("invalid local model input end", false)),
        "model_inputs_refused_end_unknown_field" => {
            Some(("unknown field `extra`, expected `type`", true))
        }
        "model_inputs_refused_pwd_digest_mismatch" | "model_inputs_refused_environment_shape_3" => {
            Some(("invalid local model working directory identity", false))
        }
        "model_inputs_refused_arguments_over_bound"
        | "model_inputs_refused_environment_wrong_chunks" => {
            Some(("invalid local model configuration", false))
        }
        "model_inputs_refused_retired_network_field" => Some(("unknown field `network`,", true)),
        "model_inputs_refused_environment_shape_0" | "model_inputs_refused_environment_shape_2" => {
            Some(("invalid local model environment names", false))
        }
        "model_inputs_refused_environment_shape_1" => {
            Some(("invalid local model environment", false))
        }
        "model_inputs_refused_tape_capacity" => Some((
            "replay preflight framing: invalid replay trace byte/frame declaration",
            false,
        )),
        "model_clock_first_run_51"
        | "model_entropy_reservation_35"
        | "model_unsupported_flags_74"
        | "model_unsupported_open_95"
        | "model_unsupported_readlink_91"
        | "model_namespace_completed_before_capacity"
        | "model_completed_clock_before_capacity"
        | "model_remote_transcript_19" => Some(("WASI capacity: transcript-bytes", false)),
        "model_clock_callback_limit" => Some(("WASI capacity: callbacks", false)),
        "model_clock_overflow_atomic" => Some(("WASI capacity: clock-nanoseconds", false)),
        "model_runtime_choice_capacity" => Some(("WASI capacity: runtime-choice-bytes", false)),
        _ => None,
    };
    if let Some((expected, prefix)) = refusal {
        let message = actual["frames"]
            .as_array()
            .and_then(|frames| frames.last())
            .and_then(|result| result["message"].as_str())
            .ok_or("missing refusal message")?;
        if if prefix {
            !message.starts_with(expected)
        } else {
            message != expected
        } {
            return Err(format!(
                "{name}: wrong refusal stage: expected {expected:?}, got {message:?}"
            ));
        }
    }
    if name.starts_with("namespace_wire_") || name.starts_with("model_inputs_refused_") {
        let frames = actual["frames"].as_array().ok_or("missing frames")?;
        let result = frames.last().ok_or("missing result")?;
        let accepted = name.starts_with("namespace_wire_accepted_");
        let want = if accepted {
            "exit"
        } else if matches!(
            name,
            "model_inputs_refused_retired_network_field"
                | "namespace_wire_refused_descriptor_unknown"
        ) {
            "unsupported"
        } else {
            "invalid"
        };
        let callbacks = frames
            .iter()
            .filter(|frame| frame["type"] == "call")
            .count();
        if result["termination"] != want
            || callbacks != usize::from(name == "namespace_wire_accepted_root")
        {
            return Err(format!(
                "{name}: expected {want} without callbacks, got {result}"
            ));
        }
        if name == "model_inputs_refused_tape_capacity"
            && (result["message"]
                != "replay preflight framing: invalid replay trace byte/frame declaration"
                || result["error"] != "replay_preflight"
                || result["check"] != "framing")
        {
            return Err(format!("{name}: wrong refusal stage: {result}"));
        }
        if name == "namespace_wire_accepted_multi"
            && result["local_model"]["transcript_bytes"] != 131_111
        {
            return Err(format!("{name}: stdin transcript length differs: {result}"));
        }
        return Ok(());
    }
    if !name.starts_with("model_") {
        return Ok(());
    }
    let frames = actual["frames"].as_array().ok_or("missing frames")?;
    let result = frames.last().ok_or("missing result")?;
    let metadata = &result["local_model"];
    let stream = |name: &str| -> Result<Vec<u8>, String> {
        let mut bytes = Vec::new();
        for frame in frames {
            if frame["type"] == "evidence" && frame["stream"] == name {
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
    let check = |got: &Value, want: Value, field: &str| -> Result<(), String> {
        if *got != want {
            return Err(format!("{name}: literal {field}: want {want}, got {got}"));
        }
        Ok(())
    };
    if name == "model_namespace_chunk_before_remote" {
        check(&result["termination"], json!("exit"), "termination")?;
        check(&metadata["calls"], json!(4), "completed calls")?;
        check(&result["output_bytes"], json!(65540), "output bytes")?;
        let callback = frames
            .iter()
            .position(|frame| frame["type"] == "call")
            .ok_or("missing callback")?;
        let stdout: Vec<_> = frames
            .iter()
            .enumerate()
            .filter(|(_, frame)| frame["type"] == "evidence" && frame["stream"] == "stdout")
            .collect();
        if stdout.len() != 2 || stdout.iter().any(|(index, _)| *index >= callback) {
            return Err("stdout must flush in two chunks before callback".into());
        }
        let first = STANDARD
            .decode(stdout[0].1["data_base64"].as_str().unwrap())
            .map_err(|error| error.to_string())?;
        if first.len() != 65536
            || first[..65535].iter().any(|byte| *byte != b'x')
            || first[65535] != b'y'
        {
            return Err("full output chunk differs from x65535+y".into());
        }
        check(
            &stdout[1].1["data_base64"],
            json!("eg=="),
            "remaining stdout z",
        )?;
        if stream("stderr")? != b"err"
            || frames.iter().enumerate().any(|(index, frame)| {
                frame["type"] == "evidence"
                    && frame["stream"] == "stderr"
                    && (index >= callback || index <= stdout[1].0)
            })
        {
            return Err("stderr err must follow stdout z before callback".into());
        }
        return Ok(());
    }
    if name == "model_debug_dormant_predecision" {
        check(
            &result["termination"],
            json!("divergence"),
            "runtime termination",
        )?;
        check(
            &result["divergence"]["Reason"],
            json!(6),
            "ordinary tape exhaustion",
        )?;
        for field in ["records", "cursor", "transcript_bytes"] {
            check(&metadata[field], json!(0), field)?;
        }
        check(&metadata["calls"], json!(1), "completed calls")?;
        check(
            &metadata["inspection_sealed"],
            json!(false),
            "dormant prefix",
        )?;
        return Ok(());
    }
    if let Some(variant) = name.strip_prefix("model_runtime_") {
        let diverged = variant == "kind_divergence";
        let forced = variant.starts_with("forced_rank");
        check(
            &result["termination"],
            json!(if diverged {
                "divergence"
            } else if variant == "choice_capacity" {
                "capacity"
            } else {
                "exit"
            }),
            "runtime termination",
        )?;
        check(&metadata["configured"], json!(true), "configured")?;
        check(&metadata["records"], json!(u64::from(!diverged)), "records")?;
        check(&metadata["cursor"], json!(u64::from(forced)), "cursor")?;
        check(
            &metadata["calls"],
            json!(if diverged { 1 } else { 2 }),
            "completed calls",
        )?;
        let mut choice = vec![0u8; 96];
        choice[8] = 1;
        choice[9] = 1;
        choice[12..16].copy_from_slice(&2u32.to_be_bytes());
        choice[16..20].copy_from_slice(&u32::from(forced).to_be_bytes());
        choice[24..32].copy_from_slice(&42u64.to_be_bytes());
        choice[32..64].fill(if forced { 2 } else { 1 });
        choice[64..].copy_from_slice(&[
            224, 5, 66, 203, 233, 157, 29, 41, 169, 40, 120, 58, 85, 122, 108, 164, 246, 228, 204,
            79, 126, 196, 129, 255, 181, 238, 34, 125, 84, 164, 188, 128,
        ]);
        if diverged {
            choice.clear();
            check(&result["divergence"]["Reason"], json!(1), "kind divergence")?;
        }
        if stream("choice")? != choice || !stream("transcript")?.is_empty() {
            return Err(format!("{name}: literal choice record differs"));
        }
        return Ok(());
    }
    let mut literal = b"GMWT\x01\0\0\0".to_vec();
    let mut record = |tag: u8, unsupported: bool, id: u64, payload: &[u8]| {
        literal.extend_from_slice(&[tag, u8::from(unsupported), 0, 0]);
        literal.extend_from_slice(&id.to_le_bytes());
        literal.extend_from_slice(&(u32::try_from(payload.len()).unwrap()).to_le_bytes());
        literal.extend_from_slice(payload);
    };
    let (termination, calls, now) = if let Some(key) = name.strip_prefix("model_frozen_clock_") {
        let (count, clock, precision, now): (u64, u32, u64, u64) = match key {
            "base" => (3, 1, 7, 1),
            "count" => (2, 1, 7, 1),
            "id" => (3, 0, 7, 1),
            "precision" => (3, 1, 8, 1),
            "start" => (3, 1, 7, 2),
            _ => return Err(format!("unknown frozen-clock case {name}")),
        };
        let mut payload = clock.to_le_bytes().to_vec();
        payload.extend_from_slice(&precision.to_le_bytes());
        payload.extend_from_slice(&now.to_le_bytes());
        payload.extend_from_slice(&count.to_le_bytes());
        record(33, false, 1, &payload);
        ("exit", count, now)
    } else if name.starts_with("model_debug_pending_") || name == "model_debug_open_clock" {
        let open = name == "model_debug_open_clock";
        let sealed = name == "model_debug_pending_inspect";
        let mut payload = 1u32.to_le_bytes().to_vec();
        payload.extend_from_slice(&0u64.to_le_bytes());
        payload.extend_from_slice(&1u64.to_le_bytes());
        payload.extend_from_slice(&(if open { 2u64 } else { 1 }).to_le_bytes());
        record(33, false, if open { 2 } else { 1 }, &payload);
        check(
            &metadata["inspection_sealed"],
            json!(sealed),
            "inspection seal",
        )?;
        for field in ["records", "cursor"] {
            check(&metadata[field], json!(0), field)?;
        }
        check(
            &metadata["finished"],
            json!(false),
            "unfinished tape suffix",
        )?;
        if !open {
            let evidence: Value = serde_json::from_slice(&stream("inspection")?)
                .map_err(|error| error.to_string())?;
            check(
                &evidence["schema"],
                json!("gomad.wasm-inspection/v3"),
                "inspection schema",
            )?;
            check(
                &evidence["world_encoding"],
                json!("gomad.inspection-world-sha256/v2"),
                "inspection World encoding",
            )?;
            check(
                &evidence["complete"],
                json!(!sealed),
                "inspection completeness",
            )?;
        }
        (
            if sealed {
                "inspected_prefix"
            } else if name == "model_debug_pending_refused_stream" {
                "instrumentation_divergence"
            } else {
                "exit"
            },
            if open { 3 } else { 1 },
            1,
        )
    } else if name == "model_shared_clock_idle_poll_remote" {
        record(
            33,
            false,
            2,
            b"\x01\0\0\0\0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0",
        );
        let mut poll = 1u32.to_le_bytes().to_vec();
        poll.extend_from_slice(&7u64.to_le_bytes());
        poll.push(0);
        poll.extend_from_slice(&1u32.to_le_bytes());
        poll.extend_from_slice(&5u64.to_le_bytes());
        poll.extend_from_slice(&0u64.to_le_bytes());
        poll.extend_from_slice(&0u16.to_le_bytes());
        poll.extend_from_slice(&1u32.to_le_bytes());
        poll.extend_from_slice(b"\x07\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0");
        record(32, false, 4, &poll);
        literal.extend_from_slice(b"\x0d\0\0\0\x05\0\0\0\0\0\0\0\x04\0\0\0\x03\0\0\0");
        ("exit", 5, 20)
    } else if name.starts_with("model_remote_transcript_") {
        literal.clear();
        let accepted = name == "model_remote_transcript_20";
        if accepted {
            literal.extend_from_slice(b"\x0d\0\0\0\x01\0\0\0\0\0\0\0\x04\0\0\0\x03\0\0\0");
        }
        (
            if accepted { "exit" } else { "capacity" },
            u64::from(accepted),
            if accepted { 20 } else { 1 },
        )
    } else if name == "model_typed_clock_streaming_chunks" {
        for index in 0..4096u64 {
            let mut payload = ((index % 2) as u32).to_le_bytes().to_vec();
            payload.extend_from_slice(&(index * 17).to_le_bytes());
            payload.extend_from_slice(&(index * 3 + 1).to_le_bytes());
            payload.extend_from_slice(&1u64.to_le_bytes());
            record(33, false, index + 1 + u64::from(index > 2048), &payload);
        }
        let callback = frames
            .iter()
            .position(|frame| frame["type"] == "call")
            .ok_or("missing midpoint callback")?;
        let mut chunks = 0;
        let mut early = false;
        for (index, frame) in frames.iter().enumerate() {
            if frame["type"] == "evidence" && frame["stream"] == "transcript" {
                let bytes = STANDARD
                    .decode(frame["data_base64"].as_str().unwrap())
                    .map_err(|error| error.to_string())?;
                if bytes.len() > 65536 {
                    return Err("unbounded transcript chunk".into());
                }
                chunks += 1;
                early |= index < callback;
            }
        }
        if chunks < 3 || !early {
            return Err("transcript did not stream bounded chunks before midpoint".into());
        }
        ("exit", 4097, 12289)
    } else if name == "model_arguments_full_domain" {
        record(3, false, 1, &[]);
        record(4, false, 2, &[]);
        ("exit", 2, 1)
    } else if name == "model_local_yield_literal" {
        record(2, false, 1, &[]);
        ("exit", 1, 1)
    } else if matches!(
        name,
        "model_namespace_mixed_poll_clock_advance" | "model_namespace_mixed_poll_prefix"
    ) {
        let mut clock = 4u64.to_le_bytes().to_vec();
        clock.push(0);
        clock.extend_from_slice(&1u32.to_le_bytes());
        clock.extend_from_slice(&10u64.to_le_bytes());
        clock.extend_from_slice(&0u64.to_le_bytes());
        clock.extend_from_slice(&0u16.to_le_bytes());
        let mut mixed = 4u32.to_le_bytes().to_vec();
        for (userdata, kind, fd) in [(1u64, 1u8, 0u32), (2, 1, 99), (3, 2, 1)] {
            mixed.extend_from_slice(&userdata.to_le_bytes());
            mixed.push(kind);
            mixed.extend_from_slice(&fd.to_le_bytes());
        }
        mixed.extend_from_slice(&clock);
        mixed.extend_from_slice(&3u32.to_le_bytes());
        for (userdata, errno, kind, nbytes) in [(1u64, 0u16, 1u8, 1u64), (2, 8, 1, 0), (3, 0, 2, 0)]
        {
            mixed.extend_from_slice(&userdata.to_le_bytes());
            mixed.extend_from_slice(&errno.to_le_bytes());
            mixed.push(kind);
            mixed.extend_from_slice(&nbytes.to_le_bytes());
            mixed.extend_from_slice(&0u16.to_le_bytes());
        }
        record(32, false, 1, &mixed);
        if name == "model_namespace_mixed_poll_prefix" {
            ("exit", 1, 1)
        } else {
            let mut alone = 1u32.to_le_bytes().to_vec();
            alone.extend_from_slice(&clock);
            alone.extend_from_slice(&1u32.to_le_bytes());
            alone.extend_from_slice(b"\x04\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0");
            record(32, false, 2, &alone);
            ("exit", 2, 11)
        }
    } else if name == "model_entropy_full_counter_golden"
        || name == "model_entropy_two_full_counters"
    {
        let steps = if name == "model_entropy_two_full_counters" {
            vec![(1u64, 32u32, 0u64), (2, 32, 32), (3, 0, 64)]
        } else {
            vec![(1u64, 30u32, 0u64), (2, 6, 30), (3, 28, 36), (4, 0, 64)]
        };
        for (id, length, offset) in steps {
            let mut payload = length.to_le_bytes().to_vec();
            payload.extend_from_slice(&offset.to_le_bytes());
            record(7, false, id, &payload);
        }
        (
            "exit",
            if name == "model_entropy_two_full_counters" {
                3
            } else {
                4
            },
            1,
        )
    } else if name == "model_entropy_reservation_36" {
        record(7, false, 1, b"\x01\0\0\0\0\0\0\0\0\0\0\0");
        ("exit", 1, 1)
    } else if matches!(
        name,
        "model_clock_first_run_52"
            | "model_clock_first_run_digest_only"
            | "model_completed_clock_before_capacity"
            | "model_clock_callback_limit"
    ) {
        let mut payload = 1u32.to_le_bytes().to_vec();
        payload.extend_from_slice(
            &(if name == "model_completed_clock_before_capacity" {
                7u64
            } else {
                0
            })
            .to_le_bytes(),
        );
        payload.extend_from_slice(&1u64.to_le_bytes());
        payload.extend_from_slice(&1u64.to_le_bytes());
        record(33, false, 1, &payload);
        (
            if matches!(
                name,
                "model_completed_clock_before_capacity" | "model_clock_callback_limit"
            ) {
                "capacity"
            } else {
                "exit"
            },
            1,
            1,
        )
    } else if matches!(
        name,
        "model_clock_first_run_51" | "model_entropy_reservation_35" | "model_clock_overflow_atomic"
    ) {
        literal.clear();
        (
            "capacity",
            0,
            if name == "model_clock_overflow_atomic" {
                9_223_372_036_854_774_807
            } else {
                1
            },
        )
    } else if name == "model_namespace_completed_before_capacity" {
        record(8, false, 1, b"\x01\0\0\0\x02\0\0\0ab\x02\0\0\0");
        check(
            &json!(STANDARD.encode(stream("stdout")?)),
            json!("YWI="),
            "completed stdout",
        )?;
        ("capacity", 1, 1)
    } else if let Some(key) = name.strip_prefix("model_unsupported_") {
        let (op, limit) = key.rsplit_once('_').ok_or("missing unsupported limit")?;
        let limit: usize = limit.parse().map_err(|_| "invalid unsupported limit")?;
        let (tag, bound, mut payload, message) = match op {
            "flags" => (
                18,
                75,
                b"\x03\0\0\0\x02\0".to_vec(),
                "descriptor flags outside volatile profile",
            ),
            "open" => (
                26,
                96,
                b"\x03\0\0\0\0\0\0\0\x01\0\0\0x\0\0\x02\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\x02\0"
                    .to_vec(),
                "open flags outside volatile profile",
            ),
            "readlink" => (
                27,
                92,
                b"\x03\0\0\0\x01\0\0\0x\0\0\0\0".to_vec(),
                "operation outside stock WASI profile: path_readlink",
            ),
            _ => return Err(format!("unknown unsupported case {name}")),
        };
        if limit == bound {
            payload.extend_from_slice(&(u32::try_from(message.len()).unwrap()).to_le_bytes());
            payload.extend_from_slice(message.as_bytes());
            record(tag, true, 1, &payload);
            check(
                &result["message"],
                json!(format!(
                    "{}: {message}",
                    match op {
                        "flags" => "fd_fdstat_set_flags",
                        "open" => "path_open",
                        _ => "path_readlink",
                    }
                )),
                "unsupported message",
            )?;
            ("unsupported", 1, 1)
        } else {
            literal.clear();
            ("capacity", 0, 1)
        }
    } else if name.starts_with("model_typed_write_") {
        check(
            &result["termination"],
            json!("capacity"),
            "write termination",
        )?;
        check(
            &result["message"],
            json!("output capacity"),
            "write refusal",
        )?;
        check(&result["output_bytes"], json!(32), "offered output")?;
        check(
            &json!(
                frames
                    .iter()
                    .filter(|frame| frame["type"] == "call")
                    .count()
            ),
            json!(1),
            "callback count",
        )?;
        if name.contains("_local_") {
            check(&metadata["calls"], json!(1), "completed calls")?;
        }
        return Ok(());
    } else {
        return Err(format!("model literal assertions missing for {name}"));
    };
    check(&result["termination"], json!(termination), "termination")?;
    check(&metadata["calls"], json!(calls), "completed calls")?;
    check(&metadata["now"], json!(now), "time")?;
    check(
        &metadata["advances"],
        json!(match name {
            "model_namespace_mixed_poll_clock_advance" => 1,
            "model_shared_clock_idle_poll_remote" | "model_remote_transcript_20" => 3,
            _ => 0,
        }),
        "advances",
    )?;
    check(
        &metadata["transcript_bytes"],
        json!(literal.len()),
        "transcript bytes",
    )?;
    check(
        &metadata["transcript_sha256"],
        json!(format!("{:x}", Sha256::digest(&literal))),
        "transcript digest",
    )?;
    let emitted = stream("transcript")?;
    let expected = if name == "model_clock_first_run_digest_only" {
        &[][..]
    } else if name == "model_debug_pending_inspect" {
        &literal[..8]
    } else {
        &literal
    };
    if emitted != expected {
        return Err(format!("{name}: literal transcript bytes differ"));
    }
    Ok(())
}

pub(super) fn model_refusal_literals_reject_wrong_stage_messages() {
    for name in [
        "model_inputs_refused_environment_count",
        "model_inputs_refused_invalid_utf8",
        "model_inputs_refused_end_wrong_type",
        "model_inputs_refused_end_unknown_field",
        "model_inputs_refused_pwd_digest_mismatch",
        "model_inputs_refused_arguments_over_bound",
        "model_inputs_refused_environment_wrong_chunks",
        "model_inputs_refused_retired_network_field",
        "model_inputs_refused_environment_shape_0",
        "model_inputs_refused_environment_shape_1",
        "model_inputs_refused_environment_shape_2",
        "model_inputs_refused_environment_shape_3",
        "model_inputs_refused_tape_capacity",
        "model_clock_first_run_51",
        "model_entropy_reservation_35",
        "model_unsupported_flags_74",
        "model_unsupported_open_95",
        "model_unsupported_readlink_91",
        "model_namespace_completed_before_capacity",
        "model_completed_clock_before_capacity",
        "model_clock_callback_limit",
        "model_clock_overflow_atomic",
        "model_remote_transcript_19",
        "model_runtime_choice_capacity",
    ] {
        let (_, case) = read_case(&corpus_dir().join(format!("{name}.json"))).unwrap();
        let mut actual = case["expected"].clone();
        check_model_literals(name, &actual).unwrap();
        actual["frames"].as_array_mut().unwrap().last_mut().unwrap()["message"] =
            json!("wrong admission stage");
        assert!(
            check_model_literals(name, &actual).is_err(),
            "{name} accepted a different refusal stage"
        );
    }
}

pub(super) fn inspection_schema_literals_reject_wrong_schema_and_world_encoding() {
    let (_, case) = read_case(&corpus_dir().join("model_debug_pending_record.json")).unwrap();
    check_model_literals("model_debug_pending_record", &case["expected"]).unwrap();
    for field in ["schema", "world_encoding"] {
        let mut actual = case["expected"].clone();
        let frames = actual["frames"].as_array_mut().unwrap();
        let frame = frames
            .iter_mut()
            .find(|frame| frame["type"] == "evidence" && frame["stream"] == "inspection")
            .unwrap();
        let bytes = STANDARD
            .decode(frame["data_base64"].as_str().unwrap())
            .unwrap();
        let mut evidence: Value = serde_json::from_slice(&bytes).unwrap();
        evidence[field] = json!("wrong literal encoding");
        frame["data_base64"] = json!(STANDARD.encode(serde_json::to_vec(&evidence).unwrap()));
        assert!(
            check_model_literals("model_debug_pending_record", &actual).is_err(),
            "{field} accepted an unrelated literal"
        );
    }
}
