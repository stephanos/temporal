use super::*;
use serde_json::json;
use sha2::Digest as _;

#[test]
fn sqlite_oracle_preserves_acknowledgment_integrity_and_exact_history() {
    let intent = Workload::SQLite { selection: None };
    let mut evidence = BTreeMap::from([
        ("cluster/sqlite/2/stdout".into(), b"ACK 2\n".to_vec()),
        (
            "cluster/sqlite/3/stdout".into(),
            b"INTEGRITY ok\nHISTORY 1:committed,2:committed\n".to_vec(),
        ),
    ]);
    assert!(intent.observe(&evidence).unwrap().passed());
    for output in [
        "INTEGRITY corrupt\nHISTORY 1:committed,2:committed\n",
        "INTEGRITY ok\nHISTORY 1:committed\n",
        "INTEGRITY ok\nHISTORY 1:committed,2:wrong\n",
        "INTEGRITY ok\nHISTORY 2:committed,2:committed\n",
    ] {
        evidence.insert("cluster/sqlite/3/stdout".into(), output.as_bytes().to_vec());
        assert!(!intent.observe(&evidence).unwrap().passed(), "{output}");
    }
    evidence.insert(
        "cluster/sqlite/3/stdout".into(),
        b"HISTORY x\nHISTORY y\n".to_vec(),
    );
    assert!(intent.observe(&evidence).is_err());
}

#[test]
fn temporal_oracle_checks_both_recoveries_and_independent_convergence() {
    let row = |cluster: &str, history: &str| {
        format!("GOMAD_TEMPORAL_OBSERVATION {}\n", json!({
        "cluster_id":cluster,"history":history,"event_ids":[1,2,3,4,5],"types":["a","b","c","d","e"]
    })).into_bytes()
    };
    let mut evidence = BTreeMap::new();
    for role in ["active", "standby"] {
        for incarnation in [1, 2] {
            evidence.insert(
                format!("cluster/{role}/{incarnation}/stdout"),
                row(role, "aGlzdG9yeQ=="),
            );
        }
    }
    assert!(
        Workload::TemporalReplication { canary: false }
            .observe(&evidence)
            .unwrap()
            .passed()
    );
    assert!(
        !Workload::TemporalReplication { canary: true }
            .observe(&evidence)
            .unwrap()
            .passed()
    );
    evidence.insert(
        "cluster/standby/2/stdout".into(),
        row("standby", "b3RoZXI="),
    );
    assert!(
        !Workload::TemporalReplication { canary: false }
            .observe(&evidence)
            .unwrap()
            .passed()
    );
}

#[test]
fn storage_state_authenticates_sections_and_rejects_trailing_or_changed_bytes() {
    let sections = [b"[]".as_slice(), b"[]", b"{}", b"sha256:x", &[0; 40], b"{}"];
    let mut raw = b"GMSTORE\x01".to_vec();
    raw.extend_from_slice(&3u32.to_le_bytes());
    raw.extend_from_slice(&6u32.to_le_bytes());
    for (tag, bytes) in sections.iter().enumerate() {
        raw.extend_from_slice(&(u32::try_from(tag).unwrap() + 1).to_le_bytes());
        raw.extend_from_slice(&(bytes.len() as u64).to_le_bytes());
        raw.extend_from_slice(bytes);
    }
    let length = raw.len() as u64;
    let digest = sha2::Sha256::digest(&raw);
    raw.extend_from_slice(&length.to_le_bytes());
    raw.extend_from_slice(&digest);
    assert_eq!(native_crash::sections(&raw).unwrap().len(), 6);
    let mut changed = raw.clone();
    changed[32] ^= 1;
    assert!(native_crash::sections(&changed).is_err());
    raw.push(0);
    assert!(native_crash::sections(&raw).is_err());
}

#[test]
fn sqlite_selection_refuses_unbound_native_state_identities() {
    let selection: native_crash::Selection = serde_json::from_value(json!({
        "primary_result_sha256":"a".repeat(64),
        "state":{"Volume":"sqlite","PendingSHA256":"unbound","SelectedOperations":[],"Entries":[],"Identity":"unbound"},
        "context":{"PendingSHA256":"unbound","Operations":[]}
    })).unwrap();
    assert!(
        selection.validate().is_err(),
        "unbound native state/context must be refused before execution"
    );
}

fn bound(raw: &[u8], sqlite: bool) -> Bound {
    bound_with_choice_limit(raw, sqlite, if sqlite { 8 << 20 } else { 64 << 20 })
}

fn bound_with_choice_limit(raw: &[u8], sqlite: bool, limit: u64) -> Bound {
    let module = b"\0asm\x01\0\0\0\x05\x03\x01\x00\x01";
    let mut model = json!({"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/workspace"),
        "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
        "mode":1,"limit":limit,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":0,
        "limits":{"calls":100_000,"transcript_bytes":32<<20,"pending_events":4096}});
    if !sqlite {
        model["limits"] =
            json!({"calls":4_000_000,"transcript_bytes":128<<20,"pending_events":4096});
    }
    let base = serde_json::from_value(json!({
        "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1",
        "limits":{"fuel":if sqlite {4_000_000_000u64} else {1u64<<50},"memory_bytes":if sqlite {256u64<<20} else {4u64<<30},"wall_millis":60_000,"output_bytes":1<<20,"evidence_bytes":64<<20},
        "arguments":["gomad-target"],"environment":{"PWD":"/workspace","TZ":"UTC"},"runtime_sha256":"a".repeat(64),"model":model,
        "namespace":{"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],
            "limits":{"files":1024,"descriptors":128,"filesystem_bytes":8<<20,"output_bytes":1<<20},"epoch_nanos":0}
    })).unwrap();
    let identity = serde_json::from_value(json!({"TargetSHA256":sha2::Sha256::digest(module).to_vec(),
        "ToolchainBuildKey":"a".repeat(64),"GOOS":"wasip1","GOARCH":"wasm","ImplementationSHA256":vec![0u8;32]})).unwrap();
    let network = (!sqlite && serde_json::from_slice::<Value>(raw).unwrap()["workload"]["kind"]=="independent-temporal-replication-durable-restart").then(||serde_json::from_value(json!({"profile":"gomad.simulation-network-stream/v3",
        "limits":{"listeners":128,"connections":4096,"bytes":128<<20,"connection_bytes":4<<20,"stall_nanos":10_000_000_000u64,"transitions":1<<20}})).unwrap());
    crate::campaign::scenario::bind(raw, &base, module, &identity, network.as_ref(), 7).unwrap()
}

#[test]
fn sqlite_bound_seals_original_three_boots_and_append_only_selection() {
    let scenario: Value = serde_json::from_slice(include_bytes!("testdata/sqlite.json")).unwrap();
    let selection:native_crash::Selection = serde_json::from_value(json!({"primary_result_sha256":"a".repeat(64),
        "state":{"Volume":"sqlite","PendingSHA256":format!("sha256:{}","b".repeat(64)),"SelectedOperations":[],"Entries":[],"Identity":format!("sha256:{}","c".repeat(64))},
        "context":{"PendingSHA256":format!("sha256:{}","b".repeat(64)),"Operations":[]}})).unwrap();
    let mut actual = bound(&canonical::encode(&scenario).unwrap(), true);
    boots::sqlite(&mut actual, Some(&selection)).unwrap();
    for (index, item) in actual.request.incarnations.iter().enumerate() {
        let storage = serde_json::to_value(item.request.storage.as_ref().unwrap()).unwrap();
        let retained = serde_json::to_value(
            actual.configuration.incarnations[index]
                .storage
                .as_ref()
                .unwrap(),
        )
        .unwrap();
        assert_eq!(storage, retained);
        assert_eq!(
            storage["descriptor"]["crash_limits"],
            json!({"States":32,"Operations":8,"Depth":8,"Bytes":4<<20,"Work":1<<16})
        );
        assert_eq!(
            !storage["descriptor"]["crash_selection"].is_null(),
            index == 1
        );
        assert_eq!(!storage["context"].is_null(), index == 1);
    }
}

#[test]
fn temporal_bound_seals_role_entropy_peer_services_and_grouped_crash() {
    let actual = bound(include_bytes!("testdata/replication.json"), false);
    for (index, item) in actual.request.incarnations.iter().enumerate() {
        let retained = &actual.configuration.incarnations[index];
        let model = item.request.model.as_ref().unwrap();
        let peer = if item.handle.node == "active" {
            "10.0.0.2"
        } else {
            "10.0.0.1"
        };
        for (suffix, port) in [("FRONTEND", 7233), ("HTTP", 7243), ("HISTORY", 7234)] {
            assert_eq!(
                retained.environment[&format!("GOMAD_TEMPORAL_REMOTE_{suffix}")],
                format!("{peer}:{port}")
            );
        }
        assert_eq!(
            retained.environment["GOMAD_TEMPORAL_CLUSTER"],
            item.handle.node
        );
        assert_eq!(
            retained.environment["GOMAD_TEMPORAL_PHASE"],
            if item.handle.incarnation == 1 {
                "commit"
            } else {
                "reopen"
            }
        );
        assert_eq!(
            retained.model["entropy_key"],
            json!(model.configuration.entropy_key)
        );
    }
    assert_ne!(
        actual.configuration.incarnations[0].model["entropy_key"],
        actual.configuration.incarnations[1].model["entropy_key"]
    );
    assert_eq!(actual.request.crash_groups.len(), 1);
    assert_eq!(actual.request.crash_groups[0].nodes, ["active", "standby"]);
    assert_eq!(
        actual.request.crash_groups[0].marker,
        b"GOMAD_TEMPORAL_OBSERVATION "
    );
}

#[test]
fn temporal_history_oracle_keeps_original_one_mib_evidence_bound() {
    let mut evidence = BTreeMap::new();
    for role in ["active", "standby"] {
        for incarnation in [1, 2] {
            let row = json!({"cluster_id":role,"history":"eHh4".repeat(140_000),"event_ids":[1,2,3,4,5],"types":["a","b","c","d","e"]});
            evidence.insert(
                format!("cluster/{role}/{incarnation}/stdout"),
                format!("GOMAD_TEMPORAL_OBSERVATION {row}\n").into_bytes(),
            );
        }
    }
    assert!(
        Workload::TemporalReplication { canary: false }
            .observe(&evidence)
            .is_err(),
        "separate bounded lines must not bypass original combined oracle-evidence budget"
    );
}

#[test]
fn runtime_selection_requires_closed_typed_intent_and_original_context() {
    let unknown = json!({"kind":"sqlite-durable-selection","primary_result":"/actual/result.json","selection":{},"foreign":true});
    assert!(serde_json::from_value::<recovery::Request>(unknown).is_err());
    let request = json!({"kind":"temporal-committed-workflow-crash-reopen","primary":{"artifact_path":"relative/foreign","record_sha256":"a".repeat(64),"prefix_sha256":"b".repeat(64),"storage_state_sha256":"c".repeat(64),
        "state":{"Volume":"temporal","PendingSHA256":format!("sha256:{}","d".repeat(64)),"SelectedOperations":[],"Entries":[],"Identity":format!("sha256:{}","e".repeat(64))},
        "context":{"PendingSHA256":format!("sha256:{}","d".repeat(64)),"Operations":[]}}});
    let recovery::Request::TemporalRecovery { mut primary } =
        serde_json::from_value::<recovery::Request>(request).unwrap()
    else {
        panic!("wrong request")
    };
    assert!(primary.validate().is_err());
    primary.artifact_path = std::path::PathBuf::from("/actual/primary");
    primary.validate().unwrap();
    primary.context.pending_sha256 = format!("sha256:{}", "f".repeat(64));
    assert!(primary.validate().is_err());
    let mut scenario: Value =
        serde_json::from_slice(include_bytes!("testdata/sqlite.json")).unwrap();
    scenario["workload"]["selection"] = json!({"primary_result_sha256":"a".repeat(64),
        "state":{"Volume":"sqlite","PendingSHA256":format!("sha256:{}","b".repeat(64)),"SelectedOperations":[],"Entries":[],"Identity":format!("sha256:{}","c".repeat(64))},
        "context":{"PendingSHA256":format!("sha256:{}","b".repeat(64)),"Operations":[]}});
    assert!(
        serde_json::from_value::<Workload>(scenario["workload"].clone())
            .unwrap()
            .validate()
            .is_err()
    );
}

#[test]
fn named_reset_requires_actual_established_connection_and_inflight_retry() {
    let observation = json!({"workflow_id":"workflow","run_id":"run","result":"durable timer survived rpc-reset","history":"AQ==","event_ids":[1],"types":["completed"],
        "connections":[{"opened_nanos":1,"local_port":100,"remote_port":7233},{"opened_nanos":40_000_000_000u64,"local_port":101,"remote_port":7233}],
        "attempts":[{"started_nanos":39_000_000_000u64,"ended_nanos":41_000_000_000u64,"code":"Unavailable"},{"started_nanos":42_000_000_000u64,"ended_nanos":43_000_000_000u64,"code":"OK"}]});
    let stdout = format!(
        "GOMAD_NETWORK_TEMPORAL_OBSERVATION {}\n",
        serde_json::to_string(&observation).unwrap()
    );
    let operations = json!([
        {"at":1,"request":{"op":"dial","request_id":1},"result":{"state":"completed","handle":7,"local":{"port":100}}},
        {"at":40_000_000_000u64,"fault":{"name":"rpc-reset","action":"reset","at_nanos":40_000_000_000u64},"request":{},"result":{}},
        {"at":41_000_000_000u64,"request":{"op":"read","request_id":2,"handle":7},"result":{"error":"connection reset by peer"}},
        {"at":42_000_000_000u64,"request":{"op":"dial","request_id":3},"result":{"state":"completed","handle":8,"local":{"port":101}}}
    ]);
    let network = canonical::encode(&json!({"operations":operations})).unwrap();
    retry::witness(stdout.as_bytes(), &network).unwrap();
    for index in [0usize, 1, 2, 3] {
        let mut changed = operations.as_array().unwrap().clone();
        changed.remove(index);
        assert!(
            retry::witness(
                stdout.as_bytes(),
                &canonical::encode(&json!({"operations":changed})).unwrap()
            )
            .is_err()
        );
    }
    let mut changed = observation;
    changed["attempts"][0]["code"] = json!("OK");
    assert!(
        retry::witness(
            format!(
                "GOMAD_NETWORK_TEMPORAL_OBSERVATION {}\n",
                serde_json::to_string(&changed).unwrap()
            )
            .as_bytes(),
            &network
        )
        .is_err()
    );
}

#[test]
fn native_context_allows_persisted_dependencies_but_rejects_invalid_pending_order() {
    let mut state:native_crash::State=serde_json::from_value(json!({"Volume":"sqlite","PendingSHA256":format!("sha256:{}","b".repeat(64)),"SelectedOperations":[2],"Entries":[],"Identity":format!("sha256:{}","c".repeat(64))})).unwrap();
    let mut context:native_crash::Context=serde_json::from_value(json!({"PendingSHA256":state.pending_sha256,"Operations":[{"ID":2,"Kind":"write","Dependencies":[1],"Inode":1,"Offset":0,"Bytes":1,"PayloadSHA256":"","EffectSHA256":"","Path":"/db","Destination":""}]})).unwrap();
    native_crash::validate_state(&state, &context, "sqlite", 8).unwrap();
    context.operations[0].dependencies = Some(vec![2]);
    assert!(native_crash::validate_state(&state, &context, "sqlite", 8).is_err());
    context.operations[0].dependencies = Some(vec![1]);
    let mut earlier = context.operations[0].clone();
    earlier.id = 1;
    earlier.dependencies = None;
    context.operations.push(earlier);
    state.selected_operations = vec![1, 2];
    assert!(native_crash::validate_state(&state, &context, "sqlite", 8).is_err());
    context.operations.sort_by_key(|operation| operation.id);
    native_crash::validate_state(&state, &context, "sqlite", 8).unwrap();
    state.selected_operations = vec![2];
    assert!(native_crash::validate_state(&state, &context, "sqlite", 8).is_err());
}

#[test]
fn recovery_keeps_original_three_incarnations_and_complete_crash_bound() {
    let actual = bound(include_bytes!("testdata/recovery.json"), false);
    assert!(actual.request.network.is_none());
    assert_eq!(actual.request.incarnations.len(), 3);
    for (index, item) in actual.request.incarnations.iter().enumerate() {
        let model = item.request.model.as_ref().unwrap();
        assert_eq!(model.configuration.limit, 1 << 30);
        assert!(model.environment.contains(&format!(
            "GOMAD_TEMPORAL_PHASE={}",
            if index == 0 { "commit" } else { "reopen" }
        )));
        let storage = serde_json::to_value(item.request.storage.as_ref().unwrap()).unwrap();
        assert_eq!(!storage["descriptor"]["crash_limits"].is_null(), index == 0);
        if index == 0 {
            assert_eq!(
                storage["descriptor"]["crash_limits"],
                json!({"States":128,"Operations":256,"Depth":256,"Bytes":256<<20,"Work":1<<20})
            );
        }
    }
    for ids in [vec![1, 1], vec![2, 1], vec![1, 3]] {
        let value = json!({"cluster_id":"cluster","history":"AQ==","event_ids":ids,"types":["start","complete"]});
        assert!(
            read_history(
                format!("GOMAD_TEMPORAL_OBSERVATION {value}\n").as_bytes(),
                "GOMAD_TEMPORAL_OBSERVATION "
            )
            .is_err()
        );
    }
}

/// The scenario choice bound: an ordinary scenario defaults to 8MiB and keeps
/// an explicit 4096-byte bound, a registered Temporal workload requires its
/// original 1GiB bound, and the selected bound reaches every boot of a bound
/// ordinary scenario (registered workloads pin their own bound at binding).
#[test]
fn scenario_choice_bound_reaches_every_boot() {
    let sqlite = include_bytes!("testdata/sqlite.json");
    let recovery = include_bytes!("testdata/recovery.json");
    let mut ordinary: Value = serde_json::from_slice(sqlite).unwrap();
    ordinary.as_object_mut().unwrap().remove("workload");
    let ordinary = serde_json::to_vec(&ordinary).unwrap();
    let choice = crate::campaign::scenario::original_choice_bytes;
    for raw in [&sqlite[..], &ordinary[..]] {
        assert_eq!(choice(Some(raw), 0).unwrap(), 8 << 20);
        assert_eq!(choice(Some(raw), 4096).unwrap(), 4096);
    }
    assert_eq!(choice(None, 0).unwrap(), 160);
    assert_eq!(choice(Some(recovery), 0).unwrap(), 1 << 30);
    assert_eq!(choice(Some(recovery), 1 << 30).unwrap(), 1 << 30);
    assert!(choice(Some(recovery), 4096).is_err());
    for requested in [0, 4096] {
        let limit = choice(Some(&ordinary), requested).unwrap();
        let actual = bound_with_choice_limit(&ordinary, true, limit);
        assert_eq!(actual.request.incarnations.len(), 3);
        for incarnation in &actual.request.incarnations {
            let model = incarnation.request.model.as_ref().unwrap();
            assert_eq!(model.configuration.limit, limit, "{:?}", incarnation.handle);
        }
    }
}
