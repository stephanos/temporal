use crate::clock::{self, Answer, Clock, Event, Handle, Quiesced, TimeRequest};
use serde_json::{Value, json};
use std::collections::BTreeMap;
use std::path::PathBuf;

fn corpora() -> Vec<(String, Value)> {
    let directory =
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/clock_conformance");
    let pin = std::fs::read_to_string(directory.join("PINNED_GO_CLOCK_SHA256")).unwrap();
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
            assert_eq!(value["schema"], "gomad.clock-conformance/v1", "{name}");
            assert_eq!(value["name"], name.as_str());
            assert_eq!(value["clock_implementation_sha256"], pin.trim(), "{name}");
            (name, value)
        })
        .collect()
}

#[test]
fn clock_corpus_headers_match_the_go_pin() {
    let corpora = corpora();
    assert!(
        corpora
            .iter()
            .any(|(name, _)| name == "quoted-participant-names"),
        "missing quoted participant-name clock corpus"
    );
    for source in ["scripted", "generated", "recorded"] {
        assert!(
            corpora.iter().any(|(_, corpus)| corpus["source"] == source),
            "missing {source} clock corpus"
        );
    }
}

fn response(label: &str, result: &Result<clock::TimeResponse, String>) -> Value {
    match result {
        Ok(response) => json!({
            "handle": label,
            "generation": response.generation,
            "kind": response.kind,
            "time": response.time,
            "error": "",
        }),
        Err(error) => {
            json!({"handle": label, "generation": 0, "kind": 0, "time": 0, "error": error})
        }
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "The scenario retains its complete ordered setup and contract assertions."
)]
fn replay(name: &str, corpus: &Value) -> Result<Vec<Value>, String> {
    let mut observed = Vec::new();
    let mut clock = match (
        corpus["clock"]["kind"].as_str(),
        corpus["clock"]["forward"].as_bool(),
    ) {
        (Some("broker"), Some(false)) => Clock::broker(),
        (Some("process"), Some(forward)) => Clock::process(forward),
        kind => return Err(format!("{name}: clock kind {kind:?}")),
    };
    if corpus["initial"].as_i64() != Some(clock.current()) {
        return Err(format!("{name}: initial time {}", clock.current()));
    }
    let mut handles: BTreeMap<String, Handle> = BTreeMap::new();
    let mut labels: BTreeMap<Handle, String> = BTreeMap::new();
    for (index, step) in corpus["steps"].as_array().unwrap().iter().enumerate() {
        let field = |key: &str| step.get(key).and_then(Value::as_u64).unwrap_or(0);
        let signed = |key: &str| step.get(key).and_then(Value::as_i64).unwrap_or(0);
        let handle_of = |key: &str| {
            step.get(key)
                .and_then(Value::as_str)
                .and_then(|label| handles.get(label).copied())
        };
        let handle = handle_of("handle");
        let other = handle_of("other");
        let label = step.get("handle").and_then(Value::as_str).unwrap_or("");
        let mut error = String::new();
        let mut value = 0i64;
        let mut responses = Vec::new();
        let op = step["op"].as_str().unwrap();
        match op {
            "register" | "register_inactive" => {
                let registered = if op == "register" {
                    clock.register(step["name"].as_str().unwrap_or(""))
                } else {
                    clock.register_inactive(step["name"].as_str().unwrap_or(""))
                };
                match registered {
                    Ok(handle) => {
                        handles.insert(label.to_owned(), handle);
                        labels.insert(handle, label.to_owned());
                    }
                    Err(failure) => error = failure,
                }
            }
            "activate" => value = clock.activate(handle),
            "activate_at" => match clock.activate_at(handle, signed("current")) {
                Ok(current) => value = current,
                Err(failure) => error = failure,
            },
            "quiesce" => {
                let request = TimeRequest {
                    generation: field("generation"),
                    current: signed("current"),
                    deadline: signed("deadline"),
                    arrivals: u32::try_from(field("arrivals")).unwrap(),
                };
                match clock.quiesce(handle, request) {
                    Ok(Quiesced::Answered(answer)) => responses.push(response(label, &Ok(answer))),
                    Ok(Quiesced::Parked) => {}
                    Err(failure) => responses.push(response(label, &Err(failure))),
                }
            }
            "cancel" => {
                let Some(Answer { result, .. }) = clock.abandon(handle) else {
                    return Err(format!(
                        "{name} step {index}: cancel without a parked quiesce"
                    ));
                };
                responses.push(response(label, &result));
            }
            op => {
                let request = field("request");
                let arrivals = u32::try_from(field("arrivals")).unwrap();
                let event = match op {
                    "accepted" => Event::Accepted {
                        participant: handle,
                        request,
                        arrivals,
                    },
                    "delivered" => Event::Delivered {
                        participant: handle,
                        request,
                    },
                    "consume" => Event::Consume {
                        participant: handle,
                        arrivals,
                    },
                    "remove" => Event::Remove {
                        participant: handle,
                    },
                    "control_forwarded" => Event::ControlForwarded {
                        participant: handle,
                        node: other,
                        request,
                        arrivals,
                    },
                    "wait_accepted" => Event::WaitAccepted {
                        participant: handle,
                        request,
                        arrivals,
                    },
                    "wait_suspended" => Event::WaitSuspended {
                        participant: handle,
                        request,
                    },
                    "wait_resumed" => Event::WaitResumed {
                        participant: handle,
                        request,
                    },
                    "completion_reserved" => Event::CompletionReserved {
                        participant: handle,
                        request,
                    },
                    "request_accepted" => Event::RequestAccepted {
                        participant: handle,
                        arrivals,
                    },
                    "response_delivered" => Event::ResponseDelivered {
                        participant: handle,
                    },
                    "forwarded" => Event::Forwarded {
                        source: handle,
                        destination: other,
                        arrivals,
                        current: signed("current"),
                    },
                    "dispatched" => Event::Dispatched {
                        coordinator: handle,
                    },
                    "dispatch_unavailable" => Event::DispatchUnavailable {
                        coordinator: handle,
                    },
                    "response_arrived" => Event::ResponseArrived {
                        coordinator: handle,
                        node: other,
                        arrivals,
                    },
                    "abandoned_discarded" => Event::AbandonedResponseDiscarded {
                        coordinator: handle,
                        arrivals,
                    },
                    "runnable" => Event::Runnable {
                        participant: handle,
                    },
                    op => return Err(format!("{name} step {index}: unknown op {op}")),
                };
                if let Err(failure) = clock.apply(event) {
                    error = failure;
                }
            }
        }
        for answer in clock.take_answers() {
            responses.push(response(&labels[&answer.handle], &answer.result));
        }
        responses.sort_by(|left, right| left["handle"].as_str().cmp(&right["handle"].as_str()));
        let waiting: Vec<&String> = labels
            .iter()
            .filter(|(handle, _)| clock.is_quiescent(Some(**handle)))
            .map(|(_, label)| label)
            .collect();
        let mut actual = json!({
            "error": error,
            "responses": responses,
            "current": clock.current(),
            "waiting": waiting,
        });
        if value != 0 {
            actual["value"] = value.into();
        }
        if actual != step["expected"] {
            return Err(format!(
                "{name} (origin {}) diverges at step {index} {op}:\n  want {}\n  got  {}",
                corpus["origin"], step["expected"], actual
            ));
        }
        observed.push(actual);
    }
    Ok(observed)
}

#[test]
fn clock_corpus_replays_exactly() {
    let mut failures = Vec::new();
    let mut replayed = 0;
    for (name, corpus) in corpora() {
        match replay(&name, &corpus) {
            Ok(_) => replayed += 1,
            Err(failure) => failures.push(failure),
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
    assert!(replayed >= 50, "only {replayed} clock corpora replayed");
}

#[test]
fn generated_clock_runs_reproduce_every_observed_response() {
    let mut checked = 0;
    for (name, corpus) in corpora() {
        if corpus["source"] != "generated" {
            continue;
        }
        let first = replay(&name, &corpus).unwrap();
        let second = replay(&name, &corpus).unwrap();
        assert!(!first.is_empty(), "{name}: empty generated scenario");
        assert_eq!(first, second, "{name}: generated clock execution changed");
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"generated_clock_repeated_execution","inputs":{"corpus":name,"clock":corpus["clock"],"steps":corpus["steps"]},"observed":{"first":first,"second":second}})
        );
        checked += 1;
    }
    assert!(checked > 0, "no generated clock scenarios replayed");
}
