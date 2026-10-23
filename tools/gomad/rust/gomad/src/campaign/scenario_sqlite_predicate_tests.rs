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
