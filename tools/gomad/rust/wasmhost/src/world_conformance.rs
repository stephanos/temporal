use crate::world;
use base64::{Engine, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::Digest;
use std::path::PathBuf;
use world::digest::hex;

fn corpora() -> Vec<(String, Value)> {
    let directory =
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/world_conformance");
    let pin = std::fs::read_to_string(directory.join("PINNED_GO_WORLD_SHA256")).unwrap();
    let mut files: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| {
            path.extension()
                .is_some_and(|extension| extension == "json")
        })
        .collect();
    files.sort();
    files
        .into_iter()
        .map(|path| {
            let name = path.file_stem().unwrap().to_str().unwrap().to_owned();
            let value: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            assert_eq!(value["schema"], "gomad.world-conformance/v1", "{name}");
            assert_eq!(value["name"], name.as_str());
            assert_eq!(value["world_implementation_sha256"], pin.trim(), "{name}");
            (name, value)
        })
        .collect()
}

#[test]
fn world_corpus_headers_match_the_go_pin() {
    let corpora = corpora();
    for name in [
        "unicode-equivalence-class-seed-41",
        "unicode-equivalence-class-seed-18446744073709551615",
    ] {
        assert!(
            corpora.iter().any(|(actual, _)| actual == name),
            "missing {name} World corpus"
        );
    }
    for source in ["scripted", "generated", "recorded"] {
        assert!(
            corpora.iter().any(|(_, corpus)| corpus["source"] == source),
            "missing {source} World corpus"
        );
    }
}

fn bytes(value: &Value) -> Vec<u8> {
    STANDARD.decode(value.as_str().unwrap()).unwrap()
}

fn text(value: &Value) -> String {
    value.as_str().unwrap().to_owned()
}

fn config(value: &Value) -> world::Config {
    world::Config {
        seed: value["seed"].as_u64().unwrap(),
        limits: world::Limits {
            max_requests: value["max_requests"].as_u64().unwrap(),
            max_events: value["max_events"].as_u64().unwrap(),
            max_queued_events: value["max_queued_events"].as_u64().unwrap(),
            max_transitions: value["max_transitions"].as_u64().unwrap(),
            max_payload_bytes: value["max_payload_bytes"].as_u64().unwrap(),
            max_string_bytes: u32::try_from(value["max_string_bytes"].as_u64().unwrap()).unwrap(),
        },
    }
}

fn error(error: &world::Error) -> Value {
    json!({"class": error.class(), "message": error.to_string()})
}

fn quiescence(result: &world::Quiescence) -> Value {
    json!({
        "kind": result.kind.as_str(),
        "before": result.before,
        "after": result.after,
        "deliveries": result.deliveries.iter().map(|delivery| json!({
            "request_id": delivery.request_id,
            "event_id": delivery.event_id,
            "at": delivery.at,
            "kind": delivery.kind,
            "payload_base64": STANDARD.encode(&delivery.payload),
        })).collect::<Vec<_>>(),
        "blocked": result.blocked,
    })
}

/// Rebuilds the Go `worldConformanceState` JSON after one step.
fn state(model: &world::Model, step: &Value) -> Value {
    let input = &step["expected"]["state"];
    let cardinality = model.cardinality();
    let mut state = json!({
        "now": cardinality.now,
        "requests": cardinality.requests,
        "events": cardinality.events,
        "queued_events": cardinality.queued_events,
        "transitions": cardinality.transitions,
        "payload_bytes": cardinality.payload_bytes,
        "transcript_digest": hex(&cardinality.transcript_digest),
    });
    if input.get("state_digest").is_some() {
        let (request, event, transition) = model.next_ids();
        state["next_request_id"] = request.into();
        state["next_event_id"] = event.into();
        state["next_transition"] = transition.into();
        state["state_digest"] = hex(&model.state_digest()).into();
    }
    state
}

fn replay(name: &str, corpus: &Value) -> Result<Option<world::Model>, String> {
    let mut model = match world::Model::new(config(&corpus["config"])) {
        Ok(model) if corpus["creation_error"].is_null() => model,
        Ok(_) => return Err(format!("{name}: accepted an invalid config")),
        Err(failure) if error(&failure) == corpus["creation_error"] => {
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1","case":"world.corpus-replay","inputs":{"corpus":name,"document":corpus},"observed":{"steps":[],"creation_error":error(&failure)}})
            );
            return Ok(None);
        }
        Err(failure) => return Err(format!("{name}: creation: {failure}")),
    };
    let mut observed = Vec::new();
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
        let mut actual = json!({"error": Value::Null});
        let result = match step["op"].as_str().unwrap() {
            "register" => {
                let request = &step["request"];
                model
                    .register(world::Request {
                        kind: text(&request["kind"]),
                        resource: world::ResourceId {
                            adapter: text(&request["adapter"]),
                            kind: text(&request["resource_kind"]),
                            key: text(&request["key"]),
                        },
                        priority: u16::try_from(request["priority"].as_u64().unwrap()).unwrap(),
                        payload: bytes(&request["payload_base64"]),
                    })
                    .map(|id| ("request_id", Value::from(id)))
            }
            "ready" => {
                let readiness = &step["readiness"];
                model
                    .ready(world::Readiness {
                        request_id: readiness["request_id"].as_u64().unwrap(),
                        at: readiness["at"].as_i64().unwrap(),
                        kind: text(&readiness["kind"]),
                        payload: bytes(&readiness["payload_base64"]),
                        equivalence_class: text(&readiness["equivalence_class"]),
                    })
                    .map(|id| ("event_id", Value::from(id)))
            }
            "cancel" => model
                .cancel(step["request_id"].as_u64().unwrap_or(0))
                .map(|result| {
                    (
                        "cancellation",
                        json!({
                            "request_id": result.request_id,
                            "event_id": result.event_id,
                            "status": result.status.as_str(),
                        }),
                    )
                }),
            "quiesce" => model
                .quiesce()
                .map(|result| ("quiescence", quiescence(&result))),
            op => return Err(format!("{name} step {index}: unknown op {op}")),
        };
        actual["state"] = state(&model, step);
        match result {
            // Go omits zero identities (`omitempty`) on the result fields.
            Ok((field, value)) if value != 0 => actual[field] = value,
            Ok(_) => {}
            Err(failure) => actual["error"] = error(&failure),
        }
        observed.push(actual.clone());
        if actual != step["expected"] {
            return Err(format!(
                "{name} (origin {}) diverges at step {index} {}:\n  want {}\n  got  {}",
                corpus["origin"], step["op"], step["expected"], actual
            ));
        }
    }
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        json!({"schema":"gomad.core-predicate-observation/v1","case":"world.corpus-replay","inputs":{"corpus":name,"document":corpus},"observed":{"steps":observed,"creation_error":Value::Null}})
    );
    Ok(Some(model))
}

#[test]
fn world_corpus_replays_exactly() {
    let mut failures = Vec::new();
    let mut replayed = 0;
    let mut wires = 0;
    for (name, corpus) in corpora() {
        match replay(&name, &corpus) {
            Ok(model) => {
                replayed += 1;
                // fn-178.4: the generated writer reproduces Go's expanded wire.
                if let (Some(model), Some(expected)) =
                    (model, corpus["final"].get("expanded_base64"))
                {
                    let mut actual = Vec::new();
                    world::wire_generated::write_snapshot(&model, &mut actual).unwrap();
                    assert_eq!(
                        model.snapshot_bytes(),
                        actual.len() as u64,
                        "{name}: incremental expanded wire size"
                    );
                    let expected = bytes(expected);
                    if actual != expected {
                        let at = actual
                            .iter()
                            .zip(&expected)
                            .position(|(left, right)| left != right);
                        failures.push(format!(
                            "{name}: expanded World wire differs from Go (lengths {} vs {}, first difference at {at:?})",
                            actual.len(),
                            expected.len()
                        ));
                    }
                    let digest: [u8; 32] = sha2::Sha256::digest(&actual).into();
                    if hex(&digest) != corpus["final"]["expanded_sha256"] {
                        failures.push(format!("{name}: expanded World wire digest"));
                    }
                    wires += 1;
                }
            }
            Err(failure) => failures.push(failure),
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
    assert!(replayed >= 60, "only {replayed} World corpora replayed");
    assert!(wires >= 50, "only {wires} expanded World wires compared");
}

#[cfg(not(feature = "conformance"))]
#[test]
fn final_world_wires_round_trip_through_the_binary_owner() {
    let mut checked = 0;
    for (name, corpus) in corpora() {
        let Some(final_wire) = corpus["final"].get("expanded_base64") else {
            continue;
        };
        let expanded = bytes(final_wire);
        let config = config(&corpus["config"]);
        world::retained::validate(&expanded, &config)
            .unwrap_or_else(|error| panic!("{name}: {error}"));
        let envelope = world::retained::compress(&expanded).unwrap();
        let digest: [u8; 32] = sha2::Sha256::digest(&envelope).into();
        assert_eq!(hex(&digest), corpus["final"]["envelope_sha256"], "{name}");
        let restored = world::retained::expand(&envelope).unwrap();
        assert_eq!(restored, expanded, "{name}");
        world::retained::validate(&restored, &config).unwrap();
        assert_eq!(
            world::retained::compress(&restored).unwrap(),
            envelope,
            "{name}"
        );
        let model = replay(&name, &corpus).unwrap().unwrap();
        assert_eq!(
            hex(&model.state_digest()),
            corpus["steps"].as_array().unwrap().last().unwrap()["expected"]["state"]["state_digest"],
            "{name}"
        );
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"final_world_wire_owner_round_trip","inputs":{"corpus":name,"expanded_sha256":format!("{:x}",sha2::Sha256::digest(&expanded))},"observed":{"expanded_base64":STANDARD.encode(&expanded),"envelope_base64":STANDARD.encode(&envelope),"restored_base64":STANDARD.encode(&restored),"envelope_sha256":hex(&digest),"restored_sha256":format!("{:x}",sha2::Sha256::digest(&restored)),"state_digest":hex(&model.state_digest()),"next_ids":model.next_ids()}})
        );
        checked += 1;
    }
    assert!(checked >= 50, "only {checked} final World wires checked");
}
