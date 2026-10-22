use super::{Bound, canonical, helper, native_crash};
use serde_json::json;
use sha2::Digest as _;
use std::collections::BTreeMap;

pub(super) fn sqlite(
    bound: &mut Bound,
    selection: Option<&native_crash::Selection>,
) -> Result<(), String> {
    let handles = bound
        .request
        .incarnations
        .iter()
        .map(|item| (item.handle.node.as_str(), item.handle.incarnation))
        .collect::<Vec<_>>();
    if handles != [("sqlite", 1), ("sqlite", 2), ("sqlite", 3)] {
        return Err("SQLite qualification requires its original three incarnations".into());
    }
    for item in &mut bound.request.incarnations {
        let model = item
            .request
            .model
            .as_ref()
            .ok_or("SQLite cooperative model missing")?;
        let operation = match item.handle.incarnation {
            1 => "setup-live",
            2 if selection.is_none()
                && model.arguments.get(1).is_some_and(|arg| arg == "unsynced") =>
            {
                "unsynced"
            }
            2 => "append-crash",
            3 => "recover",
            _ => return Err("unknown SQLite phase".into()),
        };
        if model.arguments.get(1..) != Some(&[operation.to_owned(), "WAL".to_owned()][..])
            || item.request.limits.memory_bytes != 256 << 20
            || item.request.limits.fuel != 4_000_000_000
            || model.configuration.limits.calls != 100_000
            || model.configuration.limits.transcript_bytes != 32 << 20
            || model.configuration.limit != 8 << 20
        {
            return Err("SQLite qualification changed original phase or runtime bounds".into());
        }
        let storage = item
            .request
            .storage
            .as_mut()
            .ok_or("SQLite durable model missing")?;
        let mut value = serde_json::to_value(&*storage).map_err(|error| error.to_string())?;
        let storage_model = &value["descriptor"]["model"];
        if storage_model["volumes"]
            != json!([{"ID":"sqlite","Path":"/volume","CapacityBytes":4<<20}])
            || storage_model["limits"] != json!({"PendingOperations":10000,"Transitions":100_000})
            || storage_model["sqlite_lock_databases"] != json!(["/volume/db"])
            || storage_model["faults"]
                != json!([{"name":"acknowledged-crash","resource":"/volume/crash-trigger","operation":"open","occurrence":1,"action":"crash-before","bytes":0}])
        {
            return Err("SQLite qualification changed original durable model".into());
        }
        value["descriptor"]["crash_limits"] =
            json!({"States":32,"Operations":8,"Depth":8,"Bytes":4<<20,"Work":1<<16});
        if item.handle.incarnation == 2
            && let Some(selection) = selection
        {
            let context = canonical::encode(&BTreeMap::from([("sqlite", &selection.context)]))?;
            value["descriptor"]["crash_selection"] =
                json!({"sqlite": selection.state.selected_operations});
            value["descriptor"]["crash_context"] = json!({"bytes":context.len(),"chunks":context.len().div_ceil(65536),"sha256":canonical::digest(&context)});
            value["context"] = json!(context);
        }
        *storage = serde_json::from_value(value).map_err(|error| error.to_string())?;
        bound
            .configuration
            .incarnations
            .iter_mut()
            .find(|retained| retained.handle == item.handle)
            .ok_or("missing retained SQLite boot")?
            .storage = Some(storage.clone());
    }

    Ok(())
}

pub(super) fn temporal_replication(bound: &mut Bound) -> Result<(), String> {
    if bound.request.incarnations.len() != 4 || bound.request.nodes != ["active", "standby"] {
        return Err(
            "Temporal qualification requires independent active/standby commit and reopen boots"
                .into(),
        );
    }
    for item in &mut bound.request.incarnations {
        let peer = match item.handle.node.as_str() {
            "active" => "10.0.0.2",
            "standby" => "10.0.0.1",
            _ => return Err("unknown Temporal role".into()),
        };
        let phase = match item.handle.incarnation {
            1 => "commit",
            2 => "reopen",
            _ => return Err("unknown Temporal phase".into()),
        };
        let model = item
            .request
            .model
            .as_mut()
            .ok_or("Temporal model missing")?;
        model.configuration.limit = 64 << 20;
        model.configuration.entropy_key = sha2::Sha256::digest(format!(
            "gomad-wasm-temporal-replication/v1/{}",
            item.handle.node
        ))
        .into();
        let frontend = format!("{peer}:7233");
        let http = format!("{peer}:7243");
        let history = format!("{peer}:7234");
        let overrides = BTreeMap::from([
            ("GOMAD_TEMPORAL_CLUSTER", item.handle.node.as_str()),
            ("GOMAD_TEMPORAL_PHASE", phase),
            ("GOMAD_TEMPORAL_REMOTE_FRONTEND", frontend.as_str()),
            ("GOMAD_TEMPORAL_REMOTE_HTTP", http.as_str()),
            ("GOMAD_TEMPORAL_REMOTE_HISTORY", history.as_str()),
        ]);
        model.environment.retain(|entry| {
            !overrides
                .keys()
                .any(|key| entry.starts_with(&format!("{key}=")))
        });
        model.environment.extend(
            overrides
                .iter()
                .map(|(key, value)| format!("{key}={value}")),
        );
        model.environment.sort();
        let retained = bound
            .configuration
            .incarnations
            .iter_mut()
            .find(|retained| retained.handle == item.handle)
            .ok_or("missing retained Temporal boot")?;
        retained.model["entropy_key"] = json!(model.configuration.entropy_key);
        retained.model["limit"] = json!(64 << 20);
        retained.environment = model
            .environment
            .iter()
            .map(|entry| {
                entry
                    .split_once('=')
                    .map(|(key, value)| (key.to_owned(), value.to_owned()))
                    .ok_or("invalid Temporal environment")
            })
            .collect::<Result<_, _>>()?;
    }
    let limits = helper::WorldLimits {
        max_requests: 2 << 20,
        max_events: 2 << 20,
        max_queued_events: 250_000,
        max_transitions: 8 << 20,
        max_payload_bytes: 64 << 20,
        max_string_bytes: u32::MAX,
    };
    bound.request.broker.limits = limits.clone();
    bound.configuration.broker.limits = limits;
    bound.request.boundaries.clear();
    bound.configuration.boundaries.clear();
    let group = helper::CrashGroup {
        nodes: vec!["active".into(), "standby".into()],
        marker: b"GOMAD_TEMPORAL_OBSERVATION ".to_vec(),
    };
    bound.request.crash_groups = vec![group.clone()];
    bound.configuration.crash_groups = vec![group];

    Ok(())
}

pub(super) fn temporal_recovery(bound: &mut Bound) -> Result<(), String> {
    let limits = helper::WorldLimits {
        max_requests: 1_000_000,
        max_events: 1_000_000,
        max_queued_events: 65_536,
        max_transitions: 4_000_000,
        max_payload_bytes: 64 << 20,
        max_string_bytes: 1024,
    };
    bound.request.broker.limits = limits.clone();
    bound.configuration.broker.limits = limits;
    let handles = bound
        .request
        .incarnations
        .iter()
        .map(|item| (item.handle.node.as_str(), item.handle.incarnation))
        .collect::<Vec<_>>();
    if bound.request.network.is_some()
        || bound.configuration.network.is_some()
        || handles != [("temporal", 1), ("temporal", 2), ("temporal", 3)]
    {
        return Err(
            "Temporal recovery requires original commit and two fresh reopens of that same image"
                .into(),
        );
    }
    for item in &mut bound.request.incarnations {
        if item.request.limits.memory_bytes != 4 << 30 || item.request.limits.fuel != 1 << 50 {
            return Err("Temporal recovery changed original runtime bounds".into());
        }
        let model = item
            .request
            .model
            .as_mut()
            .ok_or("Temporal recovery model missing")?;
        recovery_selection(model)?;
        model.configuration.limit = 1 << 30;
        model
            .environment
            .retain(|entry| !entry.starts_with("GOMAD_TEMPORAL_PHASE="));
        model.environment.push(format!(
            "GOMAD_TEMPORAL_PHASE={}",
            if item.handle.incarnation == 1 {
                "commit"
            } else {
                "reopen"
            }
        ));
        model.environment.sort();
        let storage = item
            .request
            .storage
            .as_mut()
            .ok_or("Temporal recovery volume missing")?;
        let mut value = serde_json::to_value(&*storage).map_err(|error| error.to_string())?;
        let declared = &value["descriptor"]["model"];
        if declared["volumes"]
            != json!([{"ID":"temporal","Path":"/volume","CapacityBytes":256<<20}])
            || declared["limits"] != json!({"PendingOperations":100_000,"Transitions":10_000_000})
            || declared["sqlite_lock_databases"] != json!(["/volume/temporal.db"])
            || !declared["faults"].is_null()
        {
            return Err("Temporal recovery changed original durable storage model".into());
        }
        if item.handle.incarnation == 1 {
            value["descriptor"]["crash_limits"] =
                json!({"States":128,"Operations":256,"Depth":256,"Bytes":256<<20,"Work":1<<20});
        }
        *storage = serde_json::from_value(value).map_err(|error| error.to_string())?;
        let retained = bound
            .configuration
            .incarnations
            .iter_mut()
            .find(|retained| retained.handle == item.handle)
            .ok_or("Temporal recovery retained boot missing")?;
        retained.storage = Some(storage.clone());
        retained.model["limit"] = json!(1 << 30);
        retained.environment = model
            .environment
            .iter()
            .map(|entry| {
                entry
                    .split_once('=')
                    .map(|(key, value)| (key.to_owned(), value.to_owned()))
                    .ok_or("invalid Temporal recovery environment")
            })
            .collect::<Result<_, _>>()?;
    }
    Ok(())
}

fn recovery_selection(model: &gomad_wasmhost::linked::ModelInput) -> Result<(), String> {
    let expected = [
        "-test.v",
        "-test.count=1",
        "-test.timeout=1h",
        "-test.run=^TestCommittedWorkflowCrashRecovery$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite",
    ];
    if model
        .arguments
        .iter()
        .skip(1)
        .map(String::as_str)
        .ne(expected)
        || model.configuration.limits.calls != 4_000_000
        || model.configuration.limits.transcript_bytes != 128 << 20
    {
        return Err("Temporal recovery changed original selection or model limits".into());
    }
    Ok(())
}

pub(super) fn temporal_inspection(bound: &mut Bound) -> Result<(), String> {
    if bound.request.incarnations.len() != 1
        || bound.request.incarnations[0].handle.node != "history"
        || bound.request.incarnations[0].handle.incarnation != 1
    {
        return Err("Temporal onebox inspection requires the original history/1 owner".into());
    }
    let item = &mut bound.request.incarnations[0];
    if item.request.limits.memory_bytes != 4 << 30 || item.request.limits.fuel != 1 << 50 {
        return Err("Temporal inspection changed original runtime bounds".into());
    }
    let model = item
        .request
        .model
        .as_mut()
        .ok_or("Temporal inspection model missing")?;
    if model.arguments.get(1..)
        != Some(
            &[
                "-test.run=^TestReplayInspectionWorkflowFailure$".into(),
                "-test.v".into(),
            ][..],
        )
    {
        return Err("Temporal onebox inspection changed its original selection".into());
    }
    model.configuration.limit = 1 << 30;
    bound.configuration.incarnations[0].model["limit"] = json!(1 << 30);
    Ok(())
}
