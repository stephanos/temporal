use super::*;

fn corpus() -> (Vec<u8>, BrokerConfig, Value) {
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../wasmhost/tests/testdata/world_conformance");
    let doc: Value = serde_json::from_slice(
        &std::fs::read(root.join("broker-incarnation-pattern.json")).unwrap(),
    )
    .unwrap();
    let raw =
        crate::qualify::cells::base64(doc["final"]["expanded_base64"].as_str().unwrap()).unwrap();
    let c = &doc["config"];
    let broker=serde_json::from_value(json!({"seed":c["seed"],"limits":{
        "max_requests":c["max_requests"],"max_events":c["max_events"],"max_queued_events":c["max_queued_events"],
        "max_transitions":c["max_transitions"],"max_payload_bytes":c["max_payload_bytes"],"max_string_bytes":c["max_string_bytes"]}})).unwrap();
    (raw, broker, doc)
}

#[test]
fn admitted_world_projects_canonical_owner_and_refuses_changed_input() {
    let (raw, broker, doc) = corpus();
    let digest = sha(&raw);
    let admitted = world(&raw, "", &broker, &digest).unwrap();
    let summary = admitted.summary();
    let expected = &doc["steps"].as_array().unwrap().last().unwrap()["expected"]["state"];
    for key in [
        "now",
        "requests",
        "events",
        "queued_events",
        "transitions",
        "payload_bytes",
        "transcript_digest",
        "state_digest",
    ] {
        assert_eq!(summary[key], expected[key], "{key}");
    }
    assert_eq!(
        admitted.requests().len() as u64,
        summary["requests"].as_u64().unwrap()
    );
    assert_eq!(
        admitted.events().len() as u64,
        summary["events"].as_u64().unwrap()
    );
    let encoding = "gomad/world/compressed-snapshot/v2";
    let encoded = codec::encode_world(&raw, encoding).unwrap();
    assert_eq!(
        world(&encoded, encoding, &broker, &sha(&encoded))
            .unwrap()
            .summary(),
        summary
    );
    let mut bad = broker.clone();
    bad.seed += 1;
    assert!(
        world(&raw, "", &bad, &digest)
            .err()
            .unwrap()
            .contains("configuration mismatch")
    );
    assert_eq!(
        world(&raw, "", &broker, &"0".repeat(64)).err().unwrap(),
        "World retained digest"
    );
    let mut trailing = raw.clone();
    trailing.push(0);
    assert!(
        world(&trailing, "", &broker, &sha(&trailing))
            .err()
            .unwrap()
            .contains("trailing")
    );
    for at in [0, 8, 96, raw.len() - 1] {
        let mut changed = raw.clone();
        changed[at] ^= 1;
        assert!(
            world(&changed, "", &broker, &sha(&changed)).is_err(),
            "offset {at}"
        );
    }
    let mut changed = encoded;
    let last = changed.len() - 1;
    changed[last] ^= 1;
    assert!(world(&changed, encoding, &broker, &sha(&changed)).is_err());
}

fn network_model() -> NetworkModel {
    serde_json::from_value(json!({"profile":"gomad.simulation-network-stream/v3","faults":[],"limits":{
        "listeners":128,"connections":128,"bytes":8<<20,"connection_bytes":4<<20,"stall_nanos":10000000000u64,"transitions":8192}})).unwrap()
}

#[test]
fn admitted_network_compares_observations_and_separates_shared_local_bounds() {
    let raw = include_bytes!("../../../../../wasmhost/tests/testdata/cluster_network/b.gmnetev1");
    let local = network_model();
    let mut shared = local.clone();
    shared.limits.transitions *= 2;
    shared.limits.bytes *= 2;
    shared.limits.listeners *= 2;
    shared.limits.connections *= 2;
    let admitted = codec::observe_network_evidence(raw, 1 << 20, &shared, &local).unwrap();
    let record = admitted.record().clone();
    let log = sha(admitted.log());
    let world_digest = sha(admitted.world());
    network(
        raw,
        1 << 20,
        &shared,
        &local,
        &sha(raw),
        &record,
        &log,
        &world_digest,
    )
    .unwrap();
    for end in 0..raw.len() {
        assert!(
            codec::observe_network_evidence(&raw[..end], 1 << 20, &shared, &local).is_err(),
            "prefix {end}"
        );
    }
    let mut wrong = record;
    wrong["snapshot"]["next_connection"] = json!(0);
    assert_eq!(
        network(
            raw,
            1 << 20,
            &shared,
            &local,
            &sha(raw),
            &wrong,
            &log,
            &world_digest
        )
        .err()
        .unwrap(),
        "network observed record"
    );
    let mut too_small = shared.clone();
    too_small.limits.listeners = 0;
    assert!(
        codec::observe_network_evidence(raw, 1 << 20, &too_small, &local)
            .err()
            .unwrap()
            .contains("record identity or bound")
    );
    let mut foreign = local.clone();
    foreign.limits.transitions += 1;
    assert!(
        codec::observe_network_evidence(raw, 1 << 20, &shared, &foreign)
            .err()
            .unwrap()
            .contains("configuration mismatch")
    );
    let mut trailing = raw.to_vec();
    trailing.push(0);
    assert!(
        codec::observe_network_evidence(&trailing, 1 << 20, &shared, &local)
            .err()
            .unwrap()
            .contains("trailing")
    );
    assert!(
        codec::observe_network_evidence(raw, raw.len() as u64 - 1, &shared, &local)
            .err()
            .unwrap()
            .contains("byte bound")
    );
}

#[test]
fn boot_binding_expected_catalog_is_closed() {
    assert_eq!(BOOT_VARIANTS.len(), 13);
    assert!(boot_expected("foreign").is_err());
    for variant in BOOT_VARIANTS {
        let (caller, declared) = boot_expected(variant).unwrap();
        assert_eq!(
            caller["selected"].as_array().unwrap().len(),
            declared["selected"].as_array().unwrap().len()
        );
    }
    assert!(boot_rows(b"").err().unwrap().contains("missing"));
}

#[test]
#[ignore = "requires independently captured genuine selected-network harness stdout"]
fn receive_genuine_boot_rows_and_targeted_mutations() {
    let path = std::env::var("GOMAD_B06_BOOT_LOG").expect("GOMAD_B06_BOOT_LOG");
    let raw = std::fs::read(path).unwrap();
    boot_rows(&raw).unwrap();
    let text = std::str::from_utf8(&raw).unwrap();
    for line in text.lines() {
        let Some((_, raw)) = line.split_once("GOMAD_CORE_OBSERVATION ") else {
            continue;
        };
        let row = crate::cycle::custody::strict_json(raw.as_bytes()).unwrap();
        if row["case"] != "boot-network.selected-model-binding" {
            continue;
        }
        boot_row(&row).unwrap();
        let mut changed = row.clone();
        changed["inputs"]["caller_after"]["selected"][0]["incarnation"] = json!(99);
        assert_eq!(
            boot_row(&changed).unwrap_err(),
            "boot raw selected input or caller ownership"
        );
        let mut changed = row;
        changed["observed"]["error"] = json!("substituted");
        assert!(boot_row(&changed).is_err());
    }
}

#[test]
#[ignore = "requires independently captured genuine World owner harness stdout"]
fn receive_genuine_world_owner_and_targeted_workload_mutations() {
    let path = std::env::var("GOMAD_B06_WORLD_LOG").expect("GOMAD_B06_WORLD_LOG");
    let raw = std::fs::read(path).unwrap();
    let mut owner = None;
    for line in std::str::from_utf8(&raw).unwrap().lines() {
        let Some((_, raw)) = line.split_once("GOMAD_CORE_OBSERVATION ") else {
            continue;
        };
        let row = crate::cycle::custody::strict_json(raw.as_bytes()).unwrap();
        if row["case"] == "binary_world_owner_profile_round_trip" {
            assert!(owner.replace(row).is_none());
        }
    }
    let row = owner.expect("genuine owner observation");
    let o = &row["observed"];
    let expanded = crate::qualify::cells::base64(o["expanded_base64"].as_str().unwrap()).unwrap();
    let restored = crate::qualify::cells::base64(o["restored_base64"].as_str().unwrap()).unwrap();
    let envelope = crate::qualify::cells::base64(o["envelope_base64"].as_str().unwrap()).unwrap();
    assert_eq!(expanded, restored);
    assert_eq!(
        row["inputs"],
        json!({"seed":7,"requests":3,"wire_sha256":sha(&expanded)})
    );
    let summary = measured(&expanded, "", 3, &sha(&expanded)).unwrap();
    assert_eq!(
        measured(
            &envelope,
            "gomad/world/compressed-snapshot/v2",
            3,
            &sha(&envelope)
        )
        .unwrap(),
        summary
    );
    assert_eq!(o["envelope_sha256"], sha(&envelope));
    assert_eq!(o["restored_sha256"], sha(&restored));
    let mut broker = measured_config(3).unwrap();
    broker.limits.max_requests += 1;
    assert!(
        world(&expanded, "", &broker, &sha(&expanded))
            .err()
            .unwrap()
            .contains("configuration mismatch")
    );
    let mut trailing = expanded.clone();
    trailing.push(0);
    assert!(
        measured(&trailing, "", 3, &sha(&trailing))
            .err()
            .unwrap()
            .contains("trailing")
    );
    for at in [0, 8, 96, expanded.len() - 1] {
        let mut changed = expanded.clone();
        changed[at] ^= 1;
        assert!(measured(&changed, "", 3, &sha(&changed)).is_err(), "{at}");
    }
}

fn load_corpus(kind: &str) -> corpus::Corpus {
    let folder = match kind {
        "world" => "world_conformance",
        "clock" => "clock_conformance",
        "network" => "network_cluster_cases",
        "broker" => "broker_orders",
        _ => panic!("kind"),
    };
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../wasmhost/tests/testdata")
        .join(folder);
    let raw = std::fs::read_dir(root)
        .unwrap()
        .map(|e| e.unwrap().path())
        .filter(|p| p.extension().is_some_and(|s| s == "json"))
        .map(|p| {
            (
                p.file_stem().unwrap().to_str().unwrap().to_owned(),
                std::fs::read(&p).unwrap(),
            )
        })
        .collect::<std::collections::BTreeMap<_, _>>();
    let frozen = raw
        .iter()
        .map(|(name, raw)| (name.clone(), sha(raw)))
        .collect();
    corpus::Corpus::admit(kind, &raw, &frozen, None).unwrap()
}

#[test]
fn current_corpus_inventory_refuses_tamper_omission_and_foreign_header() {
    let raw=std::collections::BTreeMap::from([("fixture".to_owned(),br#"{"schema":"gomad.network-cluster-corpus/v1","name":"fixture","creation_error":"invalid","steps":[]}"#.to_vec())]);
    let frozen = raw
        .iter()
        .map(|(name, raw)| (name.clone(), sha(raw)))
        .collect();
    let admitted = corpus::Corpus::admit("network", &raw, &frozen, None).unwrap();
    let row = json!({"schema":"gomad.core-predicate-observation/v1","case":"network.corpus.creation","inputs":{"name":"fixture"},"observed":{"error":"invalid"}});
    let text = format!("GOMAD_CORE_OBSERVATION {row}\n");
    admitted.receive(text.as_bytes()).unwrap();
    let mut changed = row;
    changed["observed"]["error"] = json!("substituted");
    assert_eq!(
        admitted
            .receive(format!("GOMAD_CORE_OBSERVATION {changed}\n").as_bytes())
            .unwrap_err(),
        "network corpus API response/state"
    );
    assert_eq!(
        admitted.receive(b"").unwrap_err(),
        "corpus harness omitted cases"
    );
    assert!(
        admitted
            .receive(format!("{text}{text}").as_bytes())
            .err()
            .unwrap()
            .contains("duplicate")
    );
    assert!(
        corpus::Corpus::admit("network", &std::collections::BTreeMap::new(), &frozen, None)
            .is_err()
    );
    let mut foreign = raw.clone();
    let mut doc: Value = serde_json::from_slice(&foreign["fixture"]).unwrap();
    doc["schema"] = json!("foreign");
    foreign.insert("fixture".into(), serde_json::to_vec(&doc).unwrap());
    let resealed = foreign
        .iter()
        .map(|(name, raw)| (name.clone(), sha(raw)))
        .collect();
    assert_eq!(
        corpus::Corpus::admit("network", &foreign, &resealed, None)
            .err()
            .unwrap(),
        "corpus name/schema"
    );
    let mut changed = raw;
    changed.get_mut("fixture").unwrap().push(b' ');
    assert!(corpus::Corpus::admit("network", &changed, &frozen, None).is_err());
}

#[test]
#[ignore = "requires independently captured genuine World, clock, network and broker corpus logs"]
fn receive_genuine_corpora() {
    let root = std::path::PathBuf::from(
        std::env::var("GOMAD_B06_CORPUS_LOGS").expect("GOMAD_B06_CORPUS_LOGS"),
    );
    for kind in ["world", "clock", "network", "broker"] {
        let admitted = load_corpus(kind);
        let raw = std::fs::read(root.join(format!("{kind}.log"))).unwrap();
        if kind == "broker" {
            let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../wasmhost/tests/testdata/broker_orders");
            let inputs = std::fs::read_dir(directory)
                .unwrap()
                .map(|e| e.unwrap().path())
                .filter(|p| p.extension().is_some_and(|s| s == "gmnetev1"))
                .map(|p| {
                    (
                        p.file_stem().unwrap().to_str().unwrap().to_owned(),
                        std::fs::read(&p).unwrap(),
                    )
                })
                .collect::<std::collections::BTreeMap<_, _>>();
            let frozen = inputs
                .iter()
                .map(|(name, raw)| (name.clone(), sha(raw)))
                .collect();
            admitted.receive_broker(&raw, &inputs, &frozen).unwrap();
        } else {
            admitted.receive(&raw).unwrap();
        }
    }
}

#[test]
#[ignore = "requires raw bytes retained by the genuine bounded 20k/200k World probes"]
fn receive_genuine_measured_world_bytes() {
    let root = std::path::PathBuf::from(
        std::env::var("GOMAD_B06_MEASURED_INPUTS").expect("GOMAD_B06_MEASURED_INPUTS"),
    );
    for requests in [20_000, 200_000] {
        let raw = std::fs::read(root.join(format!("world-{requests}.compressed.raw"))).unwrap();
        let envelope =
            std::fs::read(root.join(format!("world-{requests}.compressed.envelope"))).unwrap();
        let expanded = measured(&raw, "", requests, &sha(&raw)).unwrap();
        assert_eq!(
            measured(
                &envelope,
                "gomad/world/compressed-snapshot/v2",
                requests,
                &sha(&envelope)
            )
            .unwrap(),
            expanded
        );
        println!(
            "B06_MEASURED_CONTROL requests={requests} raw_sha256={} envelope_sha256={} summary={expanded}",
            sha(&raw),
            sha(&envelope)
        );
    }
}
