use super::{case_files, closed_cluster_guards, corpus_dir, guest_path, inspection, read_case};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};

/// Concatenated evidence stream of a producer's default expected block.
fn producer_stream(name: &str, stream: &str) -> Result<Vec<u8>, String> {
    let (_, producer) = read_case(&corpus_dir().join(format!("{name}.json")))?;
    let mut bytes = Vec::new();
    for frame in producer["expected"]["frames"]
        .as_array()
        .ok_or("producer has no expected frames")?
    {
        if frame["type"] == "evidence" && frame["stream"] == stream {
            bytes.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap_or_default())
                    .map_err(|error| error.to_string())?,
            );
        }
    }
    Ok(bytes)
}

/// The JSON crash context section (tag 6) of a `GMSTORE` storage state.
fn crash_context(state: &[u8]) -> Result<Value, String> {
    let word = |at: usize| {
        state
            .get(at..at + 4)
            .map(|b| u32::from_le_bytes(b.try_into().unwrap()))
    };
    let long = |at: usize| {
        state
            .get(at..at + 8)
            .map(|b| u64::from_le_bytes(b.try_into().unwrap()))
    };
    let mut offset = 16;
    for tag in 1..=6u32 {
        if word(offset) != Some(tag) {
            return Err(format!("storage state section {tag} missing"));
        }
        let length = usize::try_from(long(offset + 4).ok_or("storage state truncated")?)
            .map_err(|_| "storage state length exceeds host width")?;
        offset += 12;
        let body = state
            .get(offset..offset + length)
            .ok_or("storage state truncated")?;
        if tag == 6 {
            return serde_json::from_slice(body).map_err(|error| error.to_string());
        }
        offset += length;
    }
    unreachable!()
}

fn chunk_frames(kind: &str, bytes: &[u8]) -> Vec<Value> {
    let mut frames: Vec<Value> = bytes
        .chunks(65536)
        .enumerate()
        .map(|(sequence, chunk)| json!({"type":format!("{kind}_input"),"sequence":sequence,"data_base64":STANDARD.encode(chunk)}))
        .collect();
    frames.push(json!({"type":format!("{kind}_inputs_end")}));
    frames
}

fn declaration(bytes: &[u8]) -> Value {
    json!({"chunks":bytes.len().div_ceil(65536),"bytes":bytes.len(),"sha256":format!("{:x}", Sha256::digest(bytes))})
}

/// Rebuilds a derived case's inputs from its producer: `storage_image`
/// embeds the producer's returned recovery image; `crash_context` embeds the
/// crash context recorded in its storage state, with JSON-pointer `set`
/// edits and, under `select`, a crash selection of every recorded operation.
/// The inputs follow the authorize frame and namespace inputs, as Go sends
/// them.
#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
pub(super) fn derive_inputs(case: &Value, derived: &Value) -> Result<Value, String> {
    if let Some(case) = closed_cluster_guards::derive(case, derived)? {
        return Ok(case);
    }
    let from = derived["from"]
        .as_str()
        .ok_or("derived.from must name a case")?;
    let mut case = case.clone();
    if let Some(edit) = derived.get("inspection") {
        if !edit
            .as_object()
            .is_some_and(|fields| fields.keys().all(|key| key == "corrupt_stream"))
        {
            return Err("derived.inspection accepts only corrupt_stream".into());
        }
        let evidence: Value = serde_json::from_slice(&producer_stream(from, "inspection")?)
            .map_err(|error| error.to_string())?;
        let boundaries = inspection::boundaries(&evidence);
        let boundary = boundaries
            .first()
            .ok_or("producer has no inspection boundary")?;
        let cut = &boundary["cuts"][0];
        let prefix_digest = |encoded: &Value, length: &Value| -> Result<Value, String> {
            let bytes = STANDARD
                .decode(encoded.as_str().ok_or("inspection bytes missing")?)
                .map_err(|error| error.to_string())?;
            let length =
                usize::try_from(length.as_u64().ok_or("inspection prefix length missing")?)
                    .map_err(|_| "inspection prefix length exceeds host width")?;
            let prefix = bytes
                .get(..length)
                .ok_or("inspection prefix exceeds stream")?;
            Ok(json!(format!("{:x}", Sha256::digest(prefix))))
        };
        let mut streams = Map::new();
        for (name, field) in [
            ("choice", "choice_bytes"),
            ("wasi", "wasi_bytes"),
            ("network", "network_bytes"),
            ("storage", "storage_bytes"),
            ("stdout", "stdout_bytes"),
            ("stderr", "stderr_bytes"),
        ] {
            streams.insert(
                name.into(),
                prefix_digest(&evidence["incarnations"][0]["streams"][name], &cut[field])?,
            );
        }
        if let Some(stream) = edit.get("corrupt_stream") {
            let name = stream.as_str().ok_or("corrupt_stream must name a stream")?;
            let digest = streams.get_mut(name).ok_or("unknown corrupt_stream")?;
            *digest = json!(format!("{:x}", Sha256::digest(b"changed neighbor output")));
        }
        case["request"]["local_model"]["inspection"] = json!({
            "target":boundary["handle"], "decision":boundary["decision"], "expected":boundary,
            "world_sha256":prefix_digest(&evidence["world"], &boundary["world_bytes"])?,
            "lifecycle_sha256":prefix_digest(&evidence["lifecycle"], &boundary["lifecycle_bytes"])?,
            "stream_sha256":[streams], "probes":[], "commands":[],
            "registries":[evidence["incarnations"][0]["registry"]],
        });
        return Ok(case);
    }
    let mut inputs = Vec::new();
    if derived["storage_image"] == true {
        let image = producer_stream(from, "storage_image")?;
        if image.is_empty() {
            return Err(format!("{from} returned no storage image"));
        }
        inputs.extend(chunk_frames("storage_image", &image));
        case["request"]["storage"]["image"] = declaration(&image);
    }
    if let Some(edit) = derived.get("crash_context") {
        let mut context = crash_context(&producer_stream(from, "storage_state")?)?;
        for (pointer, value) in edit["set"].as_object().into_iter().flatten() {
            *context
                .pointer_mut(pointer)
                .ok_or_else(|| format!("crash context has no {pointer}"))? = value.clone();
        }
        let bytes = serde_json::to_vec(&context).unwrap();
        inputs.extend(chunk_frames("storage_context", &bytes));
        case["request"]["storage"]["crash_context"] = declaration(&bytes);
        if edit["select"] == true {
            let selection: Map<String, Value> = context
                .as_object()
                .ok_or("crash context must be an object")?
                .iter()
                .map(|(volume, state)| {
                    let ids = state["Operations"]
                        .as_array()
                        .into_iter()
                        .flatten()
                        .map(|operation| operation["ID"].clone());
                    (volume.clone(), Value::Array(ids.collect()))
                })
                .collect();
            case["request"]["storage"]["crash_selection"] = Value::Object(selection);
        }
    }
    let send = case["replies"][0]["send"]
        .as_array_mut()
        .ok_or("derived case needs an authorize reply")?;
    send.retain(|frame| {
        !frame["type"].as_str().is_some_and(|kind| {
            kind.starts_with("storage_image_input") || kind.starts_with("storage_context_input")
        })
    });
    send.extend(inputs);
    Ok(case)
}

/// Derived inputs are a function of their producer: a corrupted copy is
/// detected and rebuilt to the committed bytes.
pub(super) fn derived_inputs_follow_their_producer() {
    let mut derived = 0;
    for file in case_files() {
        let (_, case) = read_case(&file).unwrap();
        let Some(spec) = case.get("derived") else {
            continue;
        };
        derived += 1;
        let mut corrupted = case.clone();
        if let Some(packet) = spec.get("cluster_packet") {
            if packet.get("bound_from").is_some() {
                let original =
                    case["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"]
                        .as_u64()
                        .unwrap();
                corrupted["request"]["instances"][1]["local_model"]["limits"]["transcript_bytes"] =
                    json!(original ^ 1);
                assert_ne!(corrupted, case);
                assert_eq!(
                    derive_inputs(&corrupted, spec).unwrap(),
                    case,
                    "{}",
                    file.display()
                );
            } else {
                assert_eq!(
                    derive_inputs(&case, spec).unwrap(),
                    case,
                    "{}",
                    file.display()
                );
                let assets = packet["assets"].as_object().unwrap();
                assert!(!assets.is_empty(), "cluster packet has no derivation");
                let path = guest_path().unwrap();
                let checked = (|| -> Result<_, String> {
                    let prepared = closed_cluster_guards::assets(&case, &path)?;
                    for name in assets.keys() {
                        std::fs::write(
                            path.parent().unwrap().join(format!("{name}.wat")),
                            b"corrupt asset",
                        )
                        .map_err(|error| error.to_string())?;
                    }
                    let rebuilt = closed_cluster_guards::assets(&case, &path)?;
                    let bytes = assets
                        .keys()
                        .map(|name| {
                            std::fs::read(path.parent().unwrap().join(format!("{name}.wat")))
                                .map_err(|error| error.to_string())
                        })
                        .collect::<Result<Vec<_>, _>>()?;
                    Ok((prepared, rebuilt, bytes))
                })();
                let cleanup = std::fs::remove_dir_all(path.parent().unwrap());
                let (prepared, rebuilt, bytes) = checked.unwrap();
                cleanup.unwrap();
                assert_eq!(rebuilt, prepared, "{}", file.display());
                for ((name, wat), bytes) in assets.iter().zip(bytes) {
                    assert_eq!(bytes, wat.as_str().unwrap().as_bytes(), "asset {name}");
                }
            }
            continue;
        }
        if spec.get("inspection").is_some() {
            corrupted["request"]["local_model"]["inspection"] = json!(null);
            assert_ne!(corrupted, case);
            assert_eq!(
                derive_inputs(&corrupted, spec).unwrap(),
                case,
                "{}",
                file.display()
            );
            continue;
        }
        corrupted["replies"][0]["send"]
            .as_array_mut()
            .unwrap()
            .push(json!({"type":"storage_context_input","sequence":9,"data_base64":""}));
        corrupted["request"]["storage"]["image"] = json!(null);
        corrupted["request"]["storage"]["crash_context"] = json!(null);
        assert_ne!(corrupted, case);
        let rebuilt = derive_inputs(&corrupted, spec).unwrap();
        let mut want = case.clone();
        for key in ["image", "crash_context"] {
            if want["request"]["storage"].get(key).is_none() {
                want["request"]["storage"][key] = json!(null);
            }
        }
        assert_eq!(rebuilt, want, "{}", file.display());
    }
    assert!(derived >= 5, "{derived} derived cases");
}

/// Relations between committed cases that the deleted engine.rs scenarios
/// asserted, checked over the expected blocks without running the helper.
pub(super) fn committed_cases_keep_their_cross_case_relations() {
    const DURABLE: &[u8] = br#""Path":"/db""#;
    let stream = |case: &str, name: &str| producer_stream(case, name).unwrap();
    // Split revocation preserves the same storage whether the revoked call
    // was replied to or completed by the interrupt (split_revocation_*).
    for kind in ["crash", "stop"] {
        for effect in ["remote_only", "local_write"] {
            for name in ["storage", "storage_state", "storage_image"] {
                let replied = stream(
                    &format!("split_revocation_{kind}_after_reply_{effect}"),
                    name,
                );
                let completed = stream(
                    &format!("split_revocation_{kind}_completed_interrupt_{effect}"),
                    name,
                );
                assert!(!replied.is_empty(), "{kind}/{effect}/{name} is empty");
                assert_eq!(replied, completed, "{kind}/{effect}/{name}");
            }
        }
    }
    // The sealed storage state binds the identity of the returned image.
    let state = stream("storage_volume_operations_local", "storage_state");
    let image = stream("storage_volume_operations_local", "storage_image");
    assert!(image.starts_with(b"GMVOLIMG"));
    let identity = format!("{:x}", Sha256::digest(&image));
    assert!(
        state
            .windows(identity.len())
            .any(|window| window == identity.as_bytes()),
        "storage state does not bind the image identity"
    );
    // Recovery from that image starts with the durable file.
    let recovered = stream("storage_recovery_from_returned_image", "storage_state");
    assert!(
        recovered
            .windows(DURABLE.len())
            .any(|window| window == DURABLE),
        "recovered state lacks /db"
    );
    // The read-only namespace run records path_open, poll_oneoff and fd_write
    // as calls 1, 2 and 3 in one transcript.
    let transcript = stream("namespace_rofs_and_mixed_poll_stay_local", "transcript");
    assert_eq!(&transcript[..8], b"GMWT\x01\0\0\0");
    let mut at = 8;
    for (id, tag) in [(1u64, 26u8), (2, 32), (3, 8)] {
        assert_eq!(transcript[at], tag, "record {id} tag");
        assert_eq!(
            &transcript[at + 4..at + 12],
            &id.to_le_bytes(),
            "record {id} id"
        );
        at += 16 + u32::from_le_bytes(transcript[at + 12..at + 16].try_into().unwrap()) as usize;
    }
    assert_eq!(at, transcript.len(), "trailing transcript records");
}
