use super::*;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::Value;

fn bytes(value: &Value) -> Vec<u8> {
    value
        .as_str()
        .map(|value| STANDARD.decode(value).unwrap())
        .unwrap_or_default()
}

fn observe(
    endpoint: &Endpoint,
    now: u64,
    result: Result<Option<Vec<u8>>, Failure>,
    expected: &Value,
    context: &str,
) {
    let error = result.as_ref().err();
    assert_eq!(
        error.map(Failure::message).unwrap_or_default(),
        expected["error"],
        "{context} error"
    );
    let data = result.ok().flatten();
    assert_eq!(
        data.map_or(Value::Null, |data| Value::String(STANDARD.encode(data))),
        expected["data"],
        "{context} data"
    );
    assert_eq!(now, expected["now"], "{context} clock");
    assert_eq!(
        serde_json::to_value(lock(&endpoint.shared).record().unwrap()).unwrap(),
        expected["record"],
        "{context} owner"
    );
    assert_eq!(
        crate::linked_network::world_snapshot(&endpoint.world),
        expected["world"],
        "{context} World"
    );
    assert_eq!(
        serde_json::to_value(&endpoint.requests).unwrap(),
        expected["requests"],
        "{context} requests"
    );
    assert_eq!(
        lock(&endpoint.shared).owner.next_deadline(),
        expected["next_deadline"],
        "{context} deadline"
    );
    match endpoint.evidence() {
        Ok(actual) => {
            assert_eq!(expected["evidence_error"], "", "{context} evidence error");
            let expected_bytes = bytes(&expected["evidence"]);
            assert_eq!(
                actual.len(),
                expected_bytes.len(),
                "{context} evidence capacity accounting"
            );
            assert_eq!(
                serde_json::from_slice::<Value>(&actual).unwrap(),
                serde_json::from_slice::<Value>(&expected_bytes).unwrap(),
                "{context} evidence"
            );
        }
        Err(error) => assert_eq!(
            error.message(),
            expected["evidence_error"],
            "{context} evidence error"
        ),
    }
}

// The Go corpus fixes intermediate owner/World state and exact capacity accounting,
// including synthetic World readiness and owner link changes unreachable through the CLI.
#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The corpus applies complete ordered Go request and state observations."
)]
fn go_standalone_network_adapter_corpus() {
    let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("tests/testdata/network_adapter_cases");
    let mut files: Vec<_> = std::fs::read_dir(directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .collect();
    files.sort();
    assert_eq!(files.len(), 62);
    for path in files {
        let fixture: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        assert_eq!(fixture["schema"], "gomad.network-adapter-corpus/v1");
        assert_eq!(
            fixture["implementation_sha256"],
            include_str!("../PINNED_GO_WASI_SHA256").trim()
        );
        if fixture["network"].is_null() {
            assert!(fixture["steps"].is_null());
            assert!(fixture["initial"]["record"].is_null());
            continue;
        }
        let configuration: ModelV2 = serde_json::from_value(fixture["network"].clone()).unwrap();
        if fixture["epoch"].as_u64().unwrap() > i64::MAX as u64 {
            assert_eq!(
                fixture["creation_error"],
                "WASI invalid: working directory or clock policy"
            );
            continue;
        }
        let mut endpoint = match Endpoint::standalone(
            configuration,
            fixture["epoch"].as_u64().unwrap(),
            8 << 20,
        ) {
            Ok(endpoint) => {
                assert_eq!(fixture["creation_error"], "", "{path:?}");
                endpoint
            }
            Err(error) => {
                assert_eq!(error.message(), fixture["creation_error"], "{path:?}");
                continue;
            }
        };
        let mut now = 1;
        let mut advances = 0;
        observe(
            &endpoint,
            now,
            Ok(None),
            &fixture["initial"],
            &format!("{path:?} initial"),
        );
        for (index, step) in fixture["steps"]
            .as_array()
            .map(Vec::as_slice)
            .unwrap_or_default()
            .iter()
            .enumerate()
        {
            let input = &step["input"];
            let nanos = input["nanos"].as_u64().unwrap_or_default();
            let result = match input["action"].as_str().unwrap() {
                "serve" => {
                    if nanos != 0 {
                        now = nanos;
                    }
                    let mut data = bytes(&input["data"]);
                    endpoint
                        .serve(
                            &mut data,
                            now,
                            !input["inactive"].as_bool().unwrap_or_default(),
                        )
                        .map(|()| Some(data))
                }
                "advance" => {
                    let shared = endpoint.shared.clone();
                    let result = endpoint.advance(&mut lock(&shared), nanos);
                    if result.is_ok() {
                        now = nanos;
                    }
                    result.map(|()| None)
                }
                "link" => {
                    let mut shared = lock(&endpoint.shared);
                    let symmetric = input["symmetric"].as_bool().unwrap_or_default();
                    match input["link"].as_str().unwrap() {
                        "partition" => shared.owner.partition(
                            &endpoint.client.node,
                            &endpoint.server.node,
                            symmetric,
                        ),
                        "heal" => shared.owner.heal(
                            &endpoint.client.node,
                            &endpoint.server.node,
                            symmetric,
                        ),
                        "delay" => shared.owner.delay(
                            &endpoint.client.node,
                            &endpoint.server.node,
                            nanos,
                            symmetric,
                        ),
                        other => panic!("unknown link {other}"),
                    }
                    .map(|()| None)
                    .map_err(infrastructure)
                }
                "idle" | "runtime_idle" => {
                    let mut data = vec![0; 16];
                    stream_abi::set_u32(&mut data, 0, 1);
                    stream_abi::set_u64(
                        &mut data,
                        8,
                        input["deadline"].as_u64().unwrap_or_default(),
                    );
                    let result = endpoint.idle(&mut data, &mut now, &mut advances);
                    if input["action"] == "idle" {
                        data[0..4].fill(0);
                    }
                    result.map(|()| Some(data))
                }
                "evidence_limit" => {
                    endpoint.transcript_limit = input["limit"].as_u64().unwrap();
                    Ok(None)
                }
                "poll_clock" => {
                    let mut data = vec![0; 0];
                    now = now.max(nanos);
                    data.extend_from_slice(
                        br#"{"events":[{"userdata":1,"errno":0,"type":0,"nbytes":0,"flags":0}]}"#,
                    );
                    Ok(Some(data))
                }
                "world_ready" => {
                    for _ in 0..input["count"].as_u64().unwrap() {
                        let request_id = endpoint
                            .world
                            .register(world::Request {
                                kind: "network".into(),
                                resource: world::ResourceId {
                                    adapter: "wasm".into(),
                                    kind: "stream".into(),
                                    key: "readiness".into(),
                                },
                                priority: 0,
                                payload: Vec::new(),
                            })
                            .unwrap();
                        endpoint
                            .world
                            .ready(world::Readiness {
                                request_id,
                                at: world::INITIAL_TIME + 1,
                                kind: "network".into(),
                                payload: Vec::new(),
                                equivalence_class: String::new(),
                            })
                            .unwrap();
                    }
                    endpoint
                        .world
                        .quiesce()
                        .map(|_| None)
                        .map_err(|error| infrastructure(error.to_string()))
                }
                other => panic!("unknown corpus action {other}"),
            };
            observe(
                &endpoint,
                now,
                result,
                &step["expected"],
                &format!("{path:?} step {index}"),
            );
        }
    }
}

#[test]
fn standalone_network_world_byte_bound_is_typed_and_preserves_the_rejected_world() {
    let fixture: Value = serde_json::from_str(include_str!(
        "../../../../tests/testdata/network_adapter_cases/pending-dial-timeout-0.json"
    ))
    .unwrap();
    for limit in [1, 1 << 20] {
        let mut endpoint = Endpoint::standalone(
            serde_json::from_value(fixture["network"].clone()).unwrap(),
            0,
            8 << 20,
        )
        .unwrap();
        endpoint.limit_transition_bytes(Some(limit)).unwrap();
        let mut refusal = false;
        for step in &fixture["steps"].as_array().unwrap()[..3] {
            let input = &step["input"];
            if input["action"] == "link" {
                lock(&endpoint.shared)
                    .owner
                    .partition(
                        &endpoint.client.node,
                        &endpoint.server.node,
                        input["symmetric"].as_bool().unwrap_or_default(),
                    )
                    .unwrap();
                continue;
            }
            let mut data = bytes(&input["data"]);
            let before = endpoint.world.state_digest();
            let observed = endpoint.serve(&mut data, input["nanos"].as_u64().unwrap_or(1), true);
            if let Err(failure) = observed {
                assert_eq!(limit, 1);
                assert!(matches!(
                    failure,
                    Failure::Exhausted {
                        dimension: "transition-bytes",
                        limit: 1,
                        ..
                    }
                ));
                assert_eq!(endpoint.world.state_digest(), before);
                assert_eq!(failure.termination(), "capacity");
                refusal = true;
                break;
            }
        }
        assert_eq!(refusal, limit == 1);
    }
}
