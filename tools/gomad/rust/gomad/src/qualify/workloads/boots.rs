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
