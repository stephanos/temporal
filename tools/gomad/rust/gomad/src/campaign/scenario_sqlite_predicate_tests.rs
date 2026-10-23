use super::*;
use crate::qualify::workloads::Workload;

fn streams(ack: &[u8], recovered: &[u8]) -> BTreeMap<String, Vec<u8>> {
    BTreeMap::from([
        ("cluster/sqlite/2/stdout".into(), ack.to_vec()),
        ("cluster/sqlite/3/stdout".into(), recovered.to_vec()),
    ])
}

#[test]
fn sqlite_fixed_lines_reject_every_original_ambiguous_observation() {
    let workload = Workload::SQLite { selection: None };
    let good = b"INTEGRITY ok\nHISTORY 1:committed,2:committed\n";
    let observation = workload.observe(&streams(b"ACK 2\n", good)).unwrap();
    assert!(observation.passed());
    let checks = serde_json::to_value(&observation).unwrap();
    let checks = checks["checks"].as_object().unwrap();
    assert_eq!(checks.len(), 3);
    assert!(checks.values().all(|value| value == &json!(true)));
    println!(
        "{}",
        json!({
            "schema":"gomad.native-lifecycle-predicate/v1",
            "original_root":"TestSQLiteObservationsForwardToNativeOracleOwner",
            "variant":"three-native-checks","observed":observation
        })
    );
    for (variant, output) in [
        ("empty", ""),
        ("missing-history", "INTEGRITY ok\n"),
        (
            "duplicate-history",
            "INTEGRITY ok\nHISTORY 1:committed\nHISTORY 2:committed\n",
        ),
        (
            "duplicate-integrity",
            "INTEGRITY ok\nINTEGRITY ok\nHISTORY 1:committed\n",
        ),
        ("nonnumeric-event", "INTEGRITY ok\nHISTORY x:committed\n"),
        ("trailing-comma", "INTEGRITY ok\nHISTORY 1:committed,\n"),
    ] {
        let error = workload
            .observe(&streams(b"ACK 2\n", output.as_bytes()))
            .unwrap_err();
        println!(
            "{}",
            json!({
                "schema":"gomad.native-lifecycle-predicate/v1",
                "original_root":"TestSQLiteObservationUsesFreshFixedLines",
                "variant":variant,"observed":{"error":error}
            })
        );
    }
    let duplicate = workload.observe(&streams(b"ACK 2\nACK 2\n", good));
    assert!(duplicate.is_err());
    println!(
        "{}",
        json!({
            "schema":"gomad.native-lifecycle-predicate/v1",
            "original_root":"TestSQLiteObservationUsesFreshFixedLines",
            "variant":"duplicate-ack","observed":{"error":duplicate.unwrap_err()}
        })
    );
}

#[test]
fn every_selected_incarnation_receives_the_same_spec_bound_network_limits() {
    let base = serde_json::to_value(configuration()).unwrap();
    let owner = &base["incarnations"][0];
    let mut input = json!({
        "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1",
        "limits":base["limits"],"arguments":["guest"],"environment":{"PWD":"/"},
        "runtime_sha256":"b".repeat(64),"namespace":owner["namespace"],
        "model":{"entropy_key":vec![0u8;32],"working_directory_sha256":"c".repeat(64),
            "arguments":{"bytes":0,"count":0,"chunks":0},
            "environment":{"bytes":0,"count":0,"chunks":0},
            "mode":1,"limit":1<<20,"diagnostics":false,"epoch_nanos":0,
            "read_step_nanos":0,"now":1,
            "limits":{"calls":1<<20,"transcript_bytes":1<<20,"pending_events":1024}}
    });
    let base: preflight::Configuration = serde_json::from_value(input.take()).unwrap();
    let identity: linked::ChoiceIdentity = serde_json::from_value(base_choice_identity()).unwrap();
    let mut scenario = operator_fixture();
    scenario["spec"]["limits"]["network_connection_bytes"] = json!(123);
    scenario["spec"]["limits"]["network_stall_nanos"] = json!(456);
    scenario["actions"].as_array_mut().unwrap().extend([
        json!({"action":"restart","handle":{"node":"a","incarnation":2}}),
        json!({"action":"wait","handle":{"node":"a","incarnation":2}}),
    ]);
    let module = b"\0asm\x01\0\0\0\x05\x03\x01\x00\x01";
    let raw = normalize(&canonical::encode(&scenario).unwrap()).unwrap();
    for (variant, connection_bytes, stall_nanos) in [
        ("matching", 123, 456),
        ("byte-mismatch", 124, 456),
        ("stall-mismatch", 123, 457),
    ] {
        let network: helper::NetworkModel = serde_json::from_value(json!({
            "profile":"gomad.simulation-network-stream/v3",
            "limits":{"listeners":8,"connections":8,"bytes":1024,
                "connection_bytes":connection_bytes,"stall_nanos":stall_nanos,"transitions":128}
        }))
        .unwrap();
        let result = bind(&raw, &base, module, &identity, Some(&network), 17);
        if variant == "matching" {
            let bound = result.unwrap();
            assert_eq!(bound.configuration.incarnations.len(), 3);
            assert_eq!(bound.request.incarnations.len(), 3);
            for incarnation in &bound.configuration.incarnations {
                let limits = &incarnation.network.as_ref().unwrap().limits;
                assert_eq!(limits.connection_bytes, 123);
                assert_eq!(limits.stall_nanos, 456);
            }
            for incarnation in &bound.request.incarnations {
                let limits = &incarnation.network.as_ref().unwrap().limits;
                assert_eq!(limits.connection_bytes, 123);
                assert_eq!(limits.stall_nanos, 456);
            }
        } else {
            assert_eq!(
                result.err().unwrap(),
                "WASM boot network limits differ from simulation specification"
            );
        }
        println!(
            "{}",
            json!({
                "schema":"gomad.native-lifecycle-predicate/v1",
                "original_root":"TestProviderBindsEverySelectedBootNetworkLimit",
                "variant":variant,"observed":{"connection_bytes":connection_bytes,
                    "stall_nanos":stall_nanos,"selected":["a/1","b/1","a/2"],
                    "accepted":variant=="matching"}
            })
        );
    }
}

fn base_choice_identity() -> Value {
    json!({"TargetSHA256":vec![0u8;32],"ToolchainBuildKey":"c".repeat(64),
        "GOOS":"wasip1","GOARCH":"wasm","ImplementationSHA256":vec![0u8;32]})
}
