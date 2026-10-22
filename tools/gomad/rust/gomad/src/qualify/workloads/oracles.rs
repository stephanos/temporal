use super::{History, canonical, history_operation, line, oracle_bytes, read_history};
use std::collections::BTreeMap;

pub(super) fn sqlite(
    streams: &BTreeMap<String, Vec<u8>>,
) -> Result<BTreeMap<String, bool>, String> {
    let output = |node: &str, incarnation: u64| {
        streams
            .get(&format!("cluster/{node}/{incarnation}/stdout"))
            .map(Vec::as_slice)
            .ok_or_else(|| "workload stdout missing".to_owned())
    };
    let mut checks = BTreeMap::new();

    let ack = line(output("sqlite", 2)?, "ACK ")?;
    if ack
        .parse::<u64>()
        .ok()
        .filter(|value| *value > 0)
        .map(|value| value.to_string())
        .as_deref()
        != Some(ack)
    {
        return Err("invalid SQLite acknowledgment".into());
    }
    let recovered = output("sqlite", 3)?;
    let integrity = line(recovered, "INTEGRITY ")?;
    let history = line(recovered, "HISTORY ")?;
    oracle_bytes([ack.len(), "committed".len(), recovered.len()])?;
    let rows = history
        .split(',')
        .map(|row| row.split_once(':').ok_or("invalid SQLite history"))
        .collect::<Result<Vec<_>, _>>()?;
    if rows.iter().any(|(id, value)| {
        id.parse::<u64>()
            .ok()
            .filter(|id| *id > 0)
            .map(|id| id.to_string())
            .as_deref()
            != Some(*id)
            || value.is_empty()
            || value.contains([':', '\r', '\n'])
    }) {
        return Err("invalid SQLite history".into());
    }
    checks.insert("acknowledged-durable-recovery.events".into(), {
        let mut ids = rows.iter().map(|(id, _)| *id).collect::<Vec<_>>();
        ids.sort_unstable();
        ids == ["1", "2"]
    });
    checks.insert(
        "acknowledged-durable-recovery.integrity".into(),
        integrity == "ok",
    );
    checks.insert(
        "acknowledged-durable-recovery.acknowledged".into(),
        ack == "2" && rows.iter().all(|(_, value)| *value == "committed"),
    );

    Ok(checks)
}

pub(super) fn replication(
    streams: &BTreeMap<String, Vec<u8>>,
    canary: bool,
) -> Result<BTreeMap<String, bool>, String> {
    let output = |node: &str, incarnation: u64| {
        streams
            .get(&format!("cluster/{node}/{incarnation}/stdout"))
            .map(Vec::as_slice)
            .ok_or_else(|| "workload stdout missing".to_owned())
    };
    let mut checks = BTreeMap::new();

    let mut recovered = Vec::new();
    for role in ["active", "standby"] {
        let read = |incarnation| -> Result<History, String> {
            let value: History = super::super::spec::strict(
                line(output(role, incarnation)?, "GOMAD_TEMPORAL_OBSERVATION ")?.as_bytes(),
            )?;
            if value.cluster_id.is_empty()
                || value.payload.is_empty()
                || value.event_ids.is_empty()
                || value.event_ids.len() != value.types.len()
                || value.event_ids.iter().any(|id| *id <= 0)
                || value.types.iter().any(String::is_empty)
                || super::super::cells::base64(&value.payload).is_none_or(|bytes| bytes.is_empty())
            {
                return Err("incomplete Temporal history observation".into());
            }
            Ok(value)
        };
        let expected = read(1)?;
        let actual = read(2)?;
        oracle_bytes([
            history_operation(&expected.payload)?.len(),
            history_operation(&actual.payload)?.len(),
        ])?;
        oracle_bytes([expected.cluster_id.len(), actual.cluster_id.len()])?;
        checks.insert(
            format!("{role}-durable-recovery.history"),
            expected.payload == actual.payload,
        );
        checks.insert(
            format!("{role}-durable-recovery.events"),
            !canary && {
                let mut ids = actual.event_ids.clone();
                ids.sort_unstable();
                ids == [1, 2, 3, 4, 5]
            },
        );
        checks.insert(
            format!("{role}-durable-recovery.identity"),
            expected.cluster_id == actual.cluster_id,
        );
        recovered.push(actual.payload);
    }
    oracle_bytes(recovered.iter().map(|history| {
        super::super::cells::base64(history).map_or(usize::MAX, |bytes| bytes.len())
    }))?;
    checks.insert(
        "independent-replication-convergence".into(),
        recovered[0] == recovered[1],
    );

    Ok(checks)
}

pub(super) fn recovery(
    streams: &BTreeMap<String, Vec<u8>>,
    forcing: Option<&super::recovery::Forcing>,
) -> Result<BTreeMap<String, bool>, String> {
    let output = |node: &str, incarnation: u64| {
        streams
            .get(&format!("cluster/{node}/{incarnation}/stdout"))
            .map(Vec::as_slice)
            .ok_or_else(|| "workload stdout missing".to_owned())
    };
    let mut checks = BTreeMap::new();

    let expected = read_history(output("temporal", 1)?, "GOMAD_TEMPORAL_OBSERVATION ")?;
    let actual = read_history(output("temporal", 2)?, "GOMAD_TEMPORAL_OBSERVATION ")?;
    oracle_bytes([
        history_operation(&expected.payload)?.len(),
        history_operation(&actual.payload)?.len(),
    ])?;
    checks.insert(
        "temporal-recovery.history".into(),
        expected.payload == actual.payload,
    );
    checks.insert(
        "temporal-recovery.events".into(),
        expected.event_ids == actual.event_ids,
    );
    checks.insert(
        "temporal-recovery.identity".into(),
        expected.cluster_id == actual.cluster_id,
    );
    checks.insert(
        "temporal-recovery.types".into(),
        expected.types == actual.types,
    );
    if let Some(forcing) = forcing {
        for (name, expected) in [
            ("stdout", &forcing.stdout_sha256),
            ("storage", &forcing.storage_sha256),
            ("choice", &forcing.prefix_sha256),
        ] {
            let raw = streams
                .get(&format!("cluster/temporal/1/{name}"))
                .ok_or("Temporal forced prefix evidence missing")?;
            if canonical::digest(raw) != *expected {
                return Err(
                    "Temporal selected state failed its exact original committed prefix".into(),
                );
            }
        }
        if expected != forcing.expected {
            return Err("Temporal selected state changed the acknowledged original history".into());
        }
    }

    Ok(checks)
}

pub(super) fn inspection(
    streams: &BTreeMap<String, Vec<u8>>,
) -> Result<BTreeMap<String, bool>, String> {
    let output = |node: &str, incarnation: u64| {
        streams
            .get(&format!("cluster/{node}/{incarnation}/stdout"))
            .map(Vec::as_slice)
            .ok_or_else(|| "workload stdout missing".to_owned())
    };
    let mut checks = BTreeMap::new();

    let stdout =
        std::str::from_utf8(output("history", 1)?).map_err(|_| "onebox output is not UTF-8")?;
    checks.insert(
        "completed-timers-and-retries".into(),
        stdout.contains("workflow completed [step 0 attempt 2 step 1 attempt 2]"),
    );
    checks.insert(
        "canary-after-completed-snapshot".into(),
        stdout.contains(
            "Temporal inspection canary after workflow timers and activity retries completed",
        ),
    );

    Ok(checks)
}
