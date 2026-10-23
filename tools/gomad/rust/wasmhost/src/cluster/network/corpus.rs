use super::*;
use base64::Engine;
use serde_json::{Value, json};

fn decode_bytes(value: &Value) -> Vec<u8> {
    value
        .as_str()
        .map(|bytes| base64::prelude::BASE64_STANDARD.decode(bytes).unwrap())
        .unwrap_or_default()
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "The fixture transfers its error string into the complete serialized observation."
)]
fn observe(
    shared: &Arc<Mutex<Shared>>,
    error: String,
    result: Option<OperationResult>,
    data: Option<Vec<u8>>,
) -> Value {
    let shared = lock(shared);
    let mut output = json!({"error":error,"record":shared.record().unwrap(),"next_deadline":shared.deadline(),"model_deadline":shared.owner.next_deadline()});
    if let Some(result) = result {
        let mut value = serde_json::to_value(&result).unwrap();
        if !result.bytes.is_empty() {
            value["bytes"] = base64::prelude::BASE64_STANDARD.encode(result.bytes).into();
        }
        output["result"] = value;
    }
    if let Some(data) = data {
        output["data"] = base64::prelude::BASE64_STANDARD.encode(data).into();
    }
    output
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
fn go_network_cluster_corpus() {
    let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("tests/testdata/network_cluster_cases");
    let mut files: Vec<_> = std::fs::read_dir(directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .collect();
    files.sort();
    assert_eq!(files.len(), 51);
    for path in files {
        let corpus: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        assert_eq!(path.extension().unwrap(), "json");
        assert_eq!(corpus["name"], path.file_stem().unwrap().to_str().unwrap());
        assert_eq!(corpus["schema"], "gomad.network-cluster-corpus/v1");
        assert_eq!(
            corpus["implementation_sha256"],
            include_str!("PINNED_GO_WASI_SHA256").trim()
        );
        assert_eq!(
            corpus["network_implementation_sha256"],
            include_str!("../../network/PINNED_GO_NETWORK_SHA256").trim()
        );
        let config: Config = serde_json::from_value(corpus["config"].clone()).unwrap();
        let ids: Vec<String> = serde_json::from_value(corpus["nodes"].clone()).unwrap();
        let shared = match Shared::new(&config, &ids) {
            Ok(shared) => {
                assert_eq!(corpus["creation_error"], "", "{path:?}");
                Arc::new(Mutex::new(shared))
            }
            Err(error) => {
                assert_eq!(corpus["creation_error"], error.message(), "{path:?}");
                println!(
                    "GOMAD_CORE_OBSERVATION {}",
                    json!({"schema":"gomad.core-predicate-observation/v1","case":"network.corpus.creation","inputs":{"name":corpus["name"]},"observed":{"error":error.message()}})
                );
                continue;
            }
        };
        let mut endpoints: BTreeMap<String, Endpoint> = ids
            .iter()
            .enumerate()
            .map(|(index, id)| {
                (
                    id.clone(),
                    Endpoint::new(
                        shared.clone(),
                        id,
                        1,
                        index as u64 + 1,
                        &config.model,
                        946_684_800_000_000_000,
                        8 << 20,
                    )
                    .unwrap(),
                )
            })
            .collect();
        for endpoint in endpoints.values_mut() {
            endpoint.attach_at(1).unwrap();
        }
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"network.corpus.initial","inputs":{"name":corpus["name"]},"observed":observe(&shared, String::new(), None, None)})
        );
        assert_eq!(
            observe(&shared, String::new(), None, None),
            corpus["initial"],
            "{path:?} initial"
        );
        for id in &ids {
            let server = format!("{id}.server");
            let expected = corpus["initial"]["record"]["snapshot"]["nodes"]
                .as_array()
                .unwrap()
                .iter()
                .find(|node| node["node"] == server)
                .unwrap();
            assert_eq!(
                lock(&shared).address(id),
                expected["address"].as_str(),
                "{path:?}: {id} address"
            );
        }
        let steps = match corpus.get("steps") {
            Some(Value::Null) => &[][..],
            Some(Value::Array(steps)) => steps.as_slice(),
            _ => panic!("{path:?}: steps must be an array or Go's null empty slice"),
        };
        for (index, step) in steps.iter().enumerate() {
            let input = &step["input"];
            let node = input["node"].as_str().unwrap_or_default();
            let nanos = input["nanos"].as_u64().unwrap_or_default();
            let mut result = None;
            let mut data = None;
            let mut entry = None;
            let error = match input["action"].as_str().unwrap() {
                "advance" => lock(&shared)
                    .advance(nanos)
                    .err()
                    .map(|error| error.message()),
                "change_link" => lock(&shared)
                    .change_link(
                        input["link_action"].as_str().unwrap(),
                        input["from"].as_str().unwrap(),
                        input["to"].as_str().unwrap(),
                        nanos,
                        input["symmetric"].as_bool().unwrap_or_default(),
                    )
                    .err()
                    .map(|error| error.message()),
                "partition_domains" => lock(&shared)
                    .owner
                    .partition(
                        input["from"].as_str().unwrap(),
                        input["to"].as_str().unwrap(),
                        input["symmetric"].as_bool().unwrap_or_default(),
                    )
                    .err(),
                "attach_epoch" => {
                    let slot = ids.iter().position(|id| id == node).unwrap() as u64 + 1;
                    let mut fresh = Endpoint::new(
                        shared.clone(),
                        node,
                        1,
                        slot,
                        &config.model,
                        input["epoch"].as_u64().unwrap(),
                        8 << 20,
                    )
                    .unwrap();
                    let result = fresh.attach_at(1);
                    data = Some(vec![u8::from(result.is_ok())]);
                    result.err().map(|error| error.message())
                }
                "register" => lock(&shared)
                    .register(node, input["incarnation"].as_u64().unwrap())
                    .err(),
                "revoke" => lock(&shared)
                    .revoke(node, 1, input["graceful"].as_bool().unwrap_or_default())
                    .err(),
                "dispatch" => {
                    let mut request = input["request"].clone();
                    request["bytes"] =
                        serde_json::to_value(decode_bytes(&request["bytes"])).unwrap();
                    result = Some(
                        lock(&shared)
                            .owner
                            .dispatch(serde_json::from_value(request).unwrap()),
                    );
                    None
                }
                "serve" => {
                    let endpoint = endpoints.get_mut(node).unwrap();
                    if let Some(limit) = input["transcript_bytes"].as_u64() {
                        endpoint.transcript_limit = limit;
                    }
                    let mut buffer = decode_bytes(&input["data"]);
                    match endpoint.serve(
                        &mut buffer,
                        nanos,
                        !input["inactive"].as_bool().unwrap_or_default(),
                    ) {
                        Ok(()) => {
                            data = Some(buffer);
                            None
                        }
                        Err(error) => {
                            if matches!(&error, Failure::Capacity(_)) {
                                assert_eq!(error.termination(), "capacity");
                            }
                            Some(error.message())
                        }
                    }
                }
                "operation_entry" => {
                    let operation = &input["operation"];
                    let mut request = operation["request"].clone();
                    request["bytes"] =
                        serde_json::to_value(decode_bytes(&request["bytes"])).unwrap();
                    let result = &operation["result"];
                    let address = |value: &Value| crate::network::Address {
                        ip: value["IP"].as_str().unwrap().into(),
                        port: value["Port"].as_u64().unwrap(),
                    };
                    let result = OperationResult {
                        request_id: result["request_id"].as_u64().unwrap(),
                        state: result["state"].as_str().unwrap().into(),
                        handle: result["handle"].as_u64().unwrap_or_default(),
                        local: address(&result["local"]),
                        remote: address(&result["remote"]),
                        bytes: decode_bytes(&result["bytes"]),
                        count: result["count"].as_u64().unwrap_or_default(),
                        deadline: result["deadline"].as_u64().unwrap_or_default(),
                        error: result["error"].as_str().unwrap_or_default().into(),
                    };
                    entry = Some(evidence::operation_entry(
                        operation["at"].as_u64().unwrap(),
                        &serde_json::from_value(request).unwrap(),
                        &result,
                    ));
                    None
                }
                "quiescence_entry" => {
                    let q = &input["quiescence"];
                    let number = |value: &Value| value.as_str().unwrap().parse::<i64>().unwrap();
                    let deliveries = q["deliveries"]
                        .as_array()
                        .unwrap()
                        .iter()
                        .map(|delivery| world::Delivery {
                            request_id: number(&delivery["requestId"]).cast_unsigned(),
                            event_id: number(&delivery["eventId"]).cast_unsigned(),
                            at: number(&delivery["at"]),
                            kind: delivery["kind"].as_str().unwrap().into(),
                            payload: decode_bytes(&delivery["payload"]),
                        })
                        .collect();
                    entry = Some(evidence::quiescence_entry(&world::Quiescence {
                        kind: world::QuiescenceKind::Delivered,
                        before: number(&q["before"]),
                        after: number(&q["after"]),
                        deliveries,
                        blocked: Vec::new(),
                    }));
                    None
                }
                action => panic!("unknown action {action}"),
            }
            .unwrap_or_default();
            let mut actual = observe(&shared, error, result, data);
            if let Some((json, binary)) = entry {
                actual["entry_json"] = String::from_utf8(json).unwrap().into();
                actual["entry_tag"] = binary[0].into();
            }
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1","case":"network.corpus.step","inputs":{"name":corpus["name"],"index":index},"observed":actual})
            );
            assert_eq!(actual, step["expected"], "{path:?} step {index}");
        }
        if corpus["name"] == "cross-node-ready-poll-and-stale-cancel-binary-fixture" {
            // These files are the only writer-backed fixtures for the Go
            // decoder tests (`wasi/cluster_network_test.go`); after a reviewed
            // evidence or scenario change, regenerate them here and rerun Go.
            let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("tests/testdata/cluster_network");
            let regenerate =
                std::env::var("GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE").as_deref() == Ok("1");
            for (id, endpoint) in &endpoints {
                let path = directory.join(format!("{id}.gmnetev1"));
                let evidence = endpoint.evidence().unwrap();
                #[cfg(not(feature = "conformance"))]
                {
                    let decoded = evidence::decode::decode(
                        &evidence,
                        1 << 20,
                        &evidence::decode::owner_limits(&config.model.limits),
                        endpoint.world.config(),
                    )
                    .unwrap();
                    assert_eq!(decoded.record, shared.lock().unwrap().record().unwrap());
                    let mut world_wire = Vec::new();
                    world::wire_generated::write_snapshot(&endpoint.world, &mut world_wire)
                        .unwrap();
                    assert_eq!(decoded.world, world_wire);
                    println!(
                        "GOMAD_CORE_OBSERVATION {}",
                        json!({"schema":"gomad.core-predicate-observation/v1","case":"cluster_network_typed_owner_round_trip","inputs":{"corpus":corpus["name"],"node":id,"evidence_base64":base64::prelude::BASE64_STANDARD.encode(&evidence),"evidence_sha256":format!("{:x}",Sha256::digest(&evidence))},"observed":{"record":decoded.record,"log_sha256":format!("{:x}",Sha256::digest(&decoded.log)),"world_sha256":format!("{:x}",Sha256::digest(&decoded.world))}})
                    );
                }
                if regenerate {
                    std::fs::create_dir_all(&directory).unwrap();
                    std::fs::write(&path, &evidence).unwrap();
                    continue;
                }
                assert_eq!(
                    evidence,
                    std::fs::read(&path).unwrap(),
                    "stale {path:?}; rerun with GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE=1 and the Go decoder tests"
                );
            }
        }
        if corpus["name"] == "own-server-dial-and-bytes" {
            let endpoint = &endpoints["a"];
            assert_eq!(endpoint.handles().len(), 3);
            assert!(endpoint.evidence().unwrap().starts_with(evidence::MAGIC));
            assert_eq!(endpoint.entry_count, 5);
            assert!(endpoint.log_json_bytes > 0);
        }
    }
}
