//! World and network receiving over independently reopened immutable buffers.
#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle reader routing belongs to B15")
)]
use gomad_wasmhost::linked_cluster::{self as codec, BrokerConfig, NetworkModel, WorldObservation};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

pub(crate) mod corpus;

pub(crate) const INITIAL_TIME: u64 = 946_684_800_000_000_000;

pub(crate) fn require(ok: bool, reason: &str) -> Result<(), String> {
    if ok { Ok(()) } else { Err(reason.into()) }
}

pub(crate) fn fields(value: &Value, names: &[&str]) -> Result<(), String> {
    let object = value.as_object().ok_or("observation is not an object")?;
    require(
        object.len() == names.len() && names.iter().all(|name| object.contains_key(*name)),
        "observation fields",
    )
}

pub(crate) fn integer(value: &Value) -> Result<u64, String> {
    value
        .as_u64()
        .ok_or_else(|| "observation unsigned integer".into())
}

pub(crate) fn bytes(value: &Value) -> Result<Vec<u8>, String> {
    value
        .as_array()
        .ok_or("missing raw bytes")?
        .iter()
        .map(|v| u8::try_from(integer(v)?).map_err(|_| "raw byte range".into()))
        .collect()
}

pub(crate) fn sha(raw: &[u8]) -> String {
    format!("{:x}", Sha256::digest(raw))
}

/// The caller supplies buffers independently reopened by RetainedSession.
pub(crate) fn world(
    raw: &[u8],
    encoding: &str,
    broker: &BrokerConfig,
    digest: &str,
) -> Result<WorldObservation, String> {
    require(sha(raw) == digest, "World retained digest")?;
    codec::observe_world(raw, encoding, broker)
}

pub(crate) fn network(
    raw: &[u8],
    limit: u64,
    shared: &NetworkModel,
    local: &NetworkModel,
    digest: &str,
    record: &Value,
    log_digest: &str,
    world_digest: &str,
) -> Result<codec::NetworkObservation, String> {
    require(sha(raw) == digest, "network retained digest")?;
    let admitted = codec::observe_network_evidence(raw, limit, shared, local)?;
    require(admitted.record() == record, "network observed record")?;
    require(
        sha(admitted.log()) == log_digest && sha(admitted.world()) == world_digest,
        "network observed stream identities",
    )?;
    Ok(admitted)
}

pub(crate) fn measured_config(requests: u64) -> Result<BrokerConfig, String> {
    require(
        matches!(requests, 3 | 20_000 | 200_000),
        "World measured cardinality",
    )?;
    serde_json::from_value(json!({"seed":7,"limits":{
        "max_requests":requests+1,"max_events":requests+1,"max_queued_events":1,
        "max_transitions":(requests*3+1).max(100_000),"max_payload_bytes":32<<20,
        "max_string_bytes":1024
    }}))
    .map_err(|e| e.to_string())
}

fn measured_request() -> Value {
    json!({"kind":"runtime_net_stream","adapter":"wasm-broker","resource_kind":"incarnation",
        "key":"active/1","priority":0,"payload":vec![0u8;32]})
}

fn measured_readiness(id: u64) -> Value {
    json!({"request_id":id,"at":INITIAL_TIME+id,"kind":"runtime_net_stream","payload":[],"equivalence_class":""})
}

/// Checks every admitted transition and terminal owner in the fixed measured workload.
pub(crate) fn measured(
    raw: &[u8],
    encoding: &str,
    requests: u64,
    digest: &str,
) -> Result<Value, String> {
    let observed = world(raw, encoding, &measured_config(requests)?, digest)?;
    let summary = observed.summary();
    require(
        summary["requests"] == requests
            && summary["events"] == requests
            && summary["transitions"] == requests * 3
            && summary["queued_events"] == 0
            && summary["now"] == INITIAL_TIME + requests
            && summary["next_ids"] == json!([requests + 1, requests + 1, requests * 3 + 1])
            && summary["payload_bytes"] == requests * 64,
        "World measured terminal counters",
    )?;
    for (index, (request, event)) in observed.requests().zip(observed.events()).enumerate() {
        let id = index as u64 + 1;
        require(
            request
                == json!({"id":id,"request":measured_request(),"state":"delivered","event_id":id})
                && event == json!({"id":id,"readiness":measured_readiness(id),"state":"delivered"}),
            "World measured terminal owner",
        )?;
    }
    for (index, transition) in observed.transitions().enumerate() {
        let id = index as u64 / 3 + 1;
        let at = INITIAL_TIME + id;
        let expected = match index % 3 {
            0 => json!({"kind":"register","request":measured_request(),"request_id":id}),
            1 => json!({"kind":"ready","readiness":measured_readiness(id),"event_id":id}),
            _ => json!({"kind":"quiesce","quiescence":{"kind":"delivered","before":at-1,"after":at,
                "deliveries":[{"request_id":id,"event_id":id,"at":at,"kind":"runtime_net_stream","payload":[]}],"blocked":[]}}),
        };
        require(
            transition["sequence"] == index as u64 + 1 && transition["body"] == expected,
            "World measured transition",
        )?;
    }
    Ok(summary)
}

pub(crate) const BOOT_VARIANTS: &[&str] = &[
    "matching",
    "network free",
    "first byte mismatch",
    "later byte mismatch",
    "first stall mismatch",
    "later stall mismatch",
    "restart byte mismatch",
    "restart stall mismatch",
    "matching explicit broker",
    "matching restart",
    "independent local capacities",
    "broker byte mismatch",
    "broker stall mismatch",
];

fn boot_model() -> Value {
    json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":8,"connections":8,
        "bytes":1024,"connection_bytes":123,"stall_nanos":456,"transitions":128},"faults":[]})
}

pub(crate) fn boot_expected(variant: &str) -> Result<(Value, Value), String> {
    require(
        BOOT_VARIANTS.contains(&variant),
        "foreign selected boot network variant",
    )?;
    let mut shared = boot_model();
    let mut selected = vec![
        json!({"node":"a","incarnation":1,"network":boot_model()}),
        json!({"node":"b","incarnation":1,"network":boot_model()}),
    ];
    if variant.starts_with("restart ") || variant == "matching restart" {
        selected.push(json!({"node":"a","incarnation":2,"network":boot_model()}));
    }
    let field = if variant.contains("stall") {
        "stall_nanos"
    } else {
        "connection_bytes"
    };
    if variant.starts_with("broker ") {
        let n = integer(&shared["limits"][field])?;
        shared["limits"][field] = json!(n + 1);
    }
    for (prefix, index) in [("first ", 0), ("later ", 1), ("restart ", 2)] {
        if variant.starts_with(prefix) {
            let n = integer(&selected[index]["network"]["limits"][field])?;
            selected[index]["network"]["limits"][field] = json!(n + 1);
        }
    }
    if variant == "independent local capacities" {
        for field in ["bytes", "transitions", "listeners", "connections"] {
            let n = integer(&selected[1]["network"]["limits"][field])?;
            selected[1]["network"]["limits"][field] = json!(n * 2);
        }
    }
    if variant == "network free" {
        shared = Value::Null;
        for item in &mut selected {
            item["network"] = Value::Null;
        }
    }
    let caller = json!({"shared":shared,"selected":selected});
    let config = if shared.is_null() {
        Value::Null
    } else {
        json!({"model":shared,"addresses":{},"links":[],"faults":[],"strict_topology":false})
    };
    let mut steps = Vec::new();
    for item in &selected {
        let h = json!({"node":item["node"],"incarnation":item["incarnation"]});
        steps.push(
            json!({"action":if item["incarnation"] == 1 {"start"} else {"restart"},"handle":h}),
        );
        steps.push(json!({"action":"wait","handle":h}));
    }
    let selected = selected
        .iter()
        .map(|item| {
            let mut item = item.clone();
            item["model_seed"] = json!(7);
            item
        })
        .collect::<Vec<_>>();
    Ok((
        caller,
        json!({"nodes":["a","b"],"shared":config,"broker_seed":7,"selected":selected,"steps":steps}),
    ))
}

pub(crate) fn boot_row(row: &Value) -> Result<(), String> {
    fields(row, &["schema", "case", "inputs", "observed"])?;
    require(
        row["schema"] == "gomad.core-predicate-observation/v1"
            && row["case"] == "boot-network.selected-model-binding",
        "boot envelope",
    )?;
    let input = &row["inputs"];
    fields(
        input,
        &[
            "original_root",
            "variant",
            "caller_before",
            "caller_after",
            "declared",
        ],
    )?;
    require(
        input["original_root"] == "TestProviderBindsEverySelectedBootNetworkLimit",
        "boot original root",
    )?;
    let variant = input["variant"].as_str().ok_or("boot variant")?;
    let (caller, declared) = boot_expected(variant)?;
    require(
        input["caller_before"] == caller
            && input["caller_after"] == caller
            && input["declared"] == declared,
        "boot raw selected input or caller ownership",
    )?;
    let selected = declared["selected"].as_array().ok_or("boot selections")?;
    let mismatch = selected.iter().any(|item| {
        !item["network"].is_null()
            && ["connection_bytes", "stall_nanos"].iter().any(|key| {
                item["network"]["limits"][key] != declared["shared"]["model"]["limits"][key]
            })
    });
    let actual = &row["observed"];
    if mismatch {
        require(
            actual == &json!({"error":"cluster boot network limits"}),
            "selected network mismatch admission",
        )
    } else {
        fields(
            actual,
            &[
                "error",
                "bound",
                "activity",
                "shared_json_bytes",
                "shared_json_sha256",
            ],
        )?;
        let mut selected = selected.iter().map(|item| json!({"node":item["node"],"incarnation":item["incarnation"],"network":item["network"]})).collect::<Vec<_>>();
        selected.sort_by_key(|item| {
            (
                item["node"].as_str().unwrap_or("").to_owned(),
                item["incarnation"].as_u64().unwrap_or(0),
            )
        });
        require(
            actual["error"] == ""
                && actual["bound"]
                    == json!({"shared":declared["shared"],"selected":selected,"owners":["a","b"]})
                && actual["activity"] == json!({"launched_stores":0,"remote_callback_attempts":0}),
            "boot binding owner/activity",
        )?;
        let raw = bytes(&actual["shared_json_bytes"])?;
        require(
            actual["shared_json_sha256"] == sha(&raw)
                && crate::cycle::custody::strict_json(&raw)? == declared["shared"],
            "boot shared identity",
        )
    }
}

pub(crate) fn boot_rows(stdout: &[u8]) -> Result<(), String> {
    let text = std::str::from_utf8(stdout).map_err(|e| e.to_string())?;
    let mut seen = std::collections::BTreeSet::new();
    for line in text.lines() {
        if let Some((_, raw)) = line.split_once("GOMAD_CORE_OBSERVATION ") {
            let row = crate::cycle::custody::strict_json(raw.as_bytes())?;
            if row["case"] != "boot-network.selected-model-binding" {
                continue;
            }
            boot_row(&row)?;
            let variant = row["inputs"]["variant"]
                .as_str()
                .ok_or("boot variant")?
                .to_owned();
            require(seen.insert(variant), "duplicate boot variant")?;
        }
    }
    require(
        seen == BOOT_VARIANTS.iter().map(|s| (*s).to_owned()).collect(),
        "missing boot variants",
    )
}

#[cfg(test)]
mod tests;
