//! Receives actual native lifecycle observations after command and source custody.
#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle reader routing belongs to B15")
)]
use super::core_world::{bytes, fields, integer, require, sha};
use gomad_wasmhost::linked_inspection::{self, Handle};
use serde_json::{Value, json};
use std::collections::{BTreeMap, BTreeSet};

mod capacity;
mod cases;
#[cfg(test)]
mod tests;

pub(crate) const SCHEMA: &str = "gomad.native-lifecycle-predicate/v1";
pub(crate) const OPERATOR: &str = "TestOperatorStopsActionsAfterFailedWait";
pub(crate) const LAZY: &str = "TestProviderClusterStartsOnlyRequestedIncarnations";
pub(crate) const HOST: &str = "TestClusterHostTerminationStatus";
pub(crate) const CLEANUP: &str = "TestProviderMandatoryCleanupReapsWithoutLifecycleActions";
pub(crate) const MODEL: &str = "TestClusterRejectsResultWithoutModelEvidence";
pub(crate) const ATTRIBUTION: &str = "TestClusterRejectsUnattributableFrames";
pub(crate) const CALLBACK: &str = "TestClusterRejectsGuestCallbacks";
pub(crate) const DRAIN: &str = "TestClusterShutdownWaitsForObservedFailureDrain";
pub(crate) const DEADLINE: &str = "TestOpenClusterRecordsHelperExecutionDeadline";
pub(crate) const RESTART: &str = "TestRejectedRestartLeavesNoUnownedBrokerSession";
pub(crate) const RETRY: &str = "TestClusterRefusedRestartLeavesIncarnationRetryable";
pub(crate) const INTERNAL: &str = "TestClusterInternalDeadlineIsWatchdog";
pub(crate) const GRACE: &str = "TestClusterShutdownBoundsFinishedClusterByExecutionDeadline";
pub(crate) const NETWORK: &str = "TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses";
pub(crate) const SQLITE: &str = "TestSQLiteObservationUsesFreshFixedLines";
pub(crate) const ORACLE: &str = "TestSQLiteObservationsForwardToNativeOracleOwner";
pub(crate) const BOOT: &str = "TestProviderBindsEverySelectedBootNetworkLimit";
pub(crate) const STORAGE: &[&str] = &[
    "TestStorageCrashBoundOnlyByCrashBefore",
    "TestOperatorAcceptsDeclaredStorageCrash",
];

pub(crate) fn catalog() -> BTreeSet<(String, String)> {
    [
        (OPERATOR, &["trap", "exit"][..]),
        (LAZY, &["native-lazy-store"]),
        (
            HOST,
            &[
                "cancelled-before-start",
                "watchdog",
                "cancelled",
                "parent deadline",
            ],
        ),
        (
            CLEANUP,
            &[
                "shutdown",
                "scenario-error",
                "context-cancel",
                "action-limit",
                "context-cancel-after-actual-output",
            ],
        ),
        (
            MODEL,
            &[
                "missing-model",
                "memory-capacity-before-bootstrap",
                "memory-capacity-after-bootstrap",
            ],
        ),
        (
            ATTRIBUTION,
            &[
                "missing-namespace",
                "unknown-step",
                "zero-incarnation",
                "duplicate-handle",
                "model-seed",
            ],
        ),
        (CALLBACK, &["call", "sync"]),
        (DRAIN, &["trap", "exit", "held-peer-release"]),
        (DEADLINE, &["native-sealed-deadline"]),
        (
            RESTART,
            &[
                "missing-storage-image",
                "stale-incarnation",
                "live-incarnation",
            ],
        ),
        (RETRY, &["recovery-image-refusal-and-retry"]),
        (INTERNAL, &["internal deadline"]),
        (GRACE, &["live-noncooperative-actual-five-second-grace"]),
        (
            NETWORK,
            &[
                "valid",
                "node without profile",
                "node local faults",
                "stock node",
                "no cluster network",
                "restarted incarnation",
                "faults without network",
            ],
        ),
        (
            SQLITE,
            &[
                "empty",
                "missing-history",
                "duplicate-history",
                "duplicate-integrity",
                "nonnumeric-event",
                "trailing-comma",
                "duplicate-ack",
            ],
        ),
        (ORACLE, &["three-native-checks"]),
        (BOOT, &["matching", "byte-mismatch", "stall-mismatch"]),
        (
            "TestStorageCrashBoundOnlyByCrashBefore|TestOperatorAcceptsDeclaredStorageCrash",
            &["crash-before", "crash-after", "io-error", "no-storage"],
        ),
    ]
    .into_iter()
    .flat_map(|(root, variants)| {
        variants
            .iter()
            .map(move |v| (root.to_owned(), (*v).to_owned()))
    })
    .collect()
}

fn handle(node: &str) -> Value {
    json!({"node":node,"incarnation":1})
}
fn step(action: &str, node: &str) -> Value {
    json!({"action":action,"handle":handle(node)})
}

fn steps(actual: &Value, expected: &Value) -> Result<(), String> {
    let items = actual
        .as_array()
        .ok_or("missing authored lifecycle steps")?;
    for item in items {
        fields(item, &["action", "handle"])?;
        fields(&item["handle"], &["node", "incarnation"])?;
        require(
            item["action"].is_string()
                && item["handle"]["node"].is_string()
                && integer(&item["handle"]["incarnation"])? > 0,
            "foreign authored lifecycle step",
        )?;
    }
    require(actual == expected, "authored lifecycle action order")
}

fn activity(value: &Value, launched: u64, callbacks: u64) -> Result<(), String> {
    fields(value, &["launched_stores", "remote_callback_attempts"])?;
    require(
        integer(&value["launched_stores"])? == launched
            && integer(&value["remote_callback_attempts"])? == callbacks,
        "native activity ownership",
    )
}

fn stores<'a>(
    value: &'a Value,
    owner_activity: &Value,
    owners: Option<&[&str]>,
    require_created: bool,
) -> Result<&'a [Value], String> {
    let rows = value.as_array().ok_or("missing native store results")?;
    let mut seen = Vec::new();
    let mut created = 0;
    for item in rows {
        fields(
            item,
            &[
                "handle",
                "terminal",
                "exit_code",
                "message",
                "authorization_to_terminal_nanos",
                "interrupted",
                "stdout",
            ],
        )?;
        fields(&item["handle"], &["node", "incarnation"])?;
        let node = item["handle"]["node"]
            .as_str()
            .ok_or("foreign lifecycle node")?;
        require(
            !node.is_empty() && node.len() <= 128 && node.is_ascii(),
            "foreign lifecycle node",
        )?;
        let incarnation = integer(&item["handle"]["incarnation"])?;
        require(incarnation > 0, "zero result incarnation")?;
        let id = format!("{node}/{incarnation}");
        require(
            seen.last().is_none_or(|prior| prior < &id),
            "duplicate or unordered native store owner",
        )?;
        seen.push(id);
        require(
            item["terminal"].is_string()
                && item["message"].is_string()
                && item["interrupted"].is_boolean(),
            "native store result types",
        )?;
        if !item["exit_code"].is_null() {
            require(
                u32::try_from(integer(&item["exit_code"])?).is_ok(),
                "exit code range",
            )?;
        }
        if item["authorization_to_terminal_nanos"].is_null() {
            require(
                !require_created && item["terminal"] == "inspection_aborted",
                "store lost drop observation",
            )?;
        } else {
            require(
                integer(&item["authorization_to_terminal_nanos"])? > 0,
                "store drop duration",
            )?;
            created += 1;
        }
        if !item["stdout"].is_null() {
            bytes(&item["stdout"])?;
        }
    }
    if let Some(owners) = owners {
        require(seen == owners, "joined store owners")?;
    }
    activity(owner_activity, created, 0)?;
    Ok(rows)
}

fn recovery(o: &Value) -> Result<(), String> {
    require(
        o["storage_before"] == o["storage_after_refusal"],
        "refused restart changed storage",
    )?;
    let before = &o["storage_before"];
    let retry = &o["retry_storage"];
    fields(before, &["descriptor", "image", "context"])?;
    fields(retry, &["descriptor", "image", "context"])?;
    let raw = bytes(&o["recovery_image"])?;
    require(
        !raw.is_empty()
            && retry["image"] == o["recovery_image"]
            && before["image"].is_null()
            && before["context"] == retry["context"]
            && o["recovery_input_unchanged"] == true,
        "restart recovery preimages",
    )?;
    require(
        before["descriptor"]["model"]
            == json!({
                "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":4096}],
                "limits":{"PendingOperations":100,"Transitions":1000},
                "faults":null,"sqlite_lock_databases":null
            }),
        "restart declared storage model",
    )?;
    let input: gomad_wasmhost::linked::StorageInput =
        serde_json::from_value(before.clone()).map_err(|e| e.to_string())?;
    let recovered = super::core_codec::recovery(&input, &raw, &sha(&raw))?;
    let volumes = recovered.volumes();
    require(
        recovered.model_identity()
            == before["descriptor"]["model_identity"]
                .as_str()
                .ok_or("storage model identity")?
            && volumes.len() == 1
            && volumes[0].id() == "data"
            && volumes[0].mount() == "/workspace"
            && volumes[0].capacity_bytes() == 4096,
        "restart recovery owner",
    )?;
    let mut old = before["descriptor"].clone();
    let mut new = retry["descriptor"].clone();
    let old_image = old
        .as_object_mut()
        .ok_or("storage descriptor")?
        .remove("image");
    let image = new
        .as_object_mut()
        .ok_or("storage descriptor")?
        .remove("image");
    require(
        old_image.is_none_or(|v| v.is_null()) && old == new,
        "restart non-image declaration",
    )?;
    require(
        image
            == Some(
                json!({"bytes":raw.len(),"chunks":raw.len().div_ceil(65536),"sha256":sha(&raw)}),
            ),
        "restart recovery identity",
    )
}

fn held_drain(o: &Value) -> Result<(), String> {
    fields(
        o,
        &[
            "terminal",
            "exit_code",
            "message",
            "failed_module_bytes",
            "failed_module_sha256",
            "peer_module_bytes",
            "peer_module_sha256",
            "held_peer_result",
            "store_results",
            "activity",
            "probe_millis",
            "probe_outcome",
            "probe_elapsed_nanos",
            "release_to_return_nanos",
            "join_error",
            "native_control_entry_points",
            "before_control_attempts",
            "held_control_attempts",
            "after_control_attempts",
            "before_lifecycle_bytes",
            "held_lifecycle_bytes",
            "after_lifecycle_bytes",
        ],
    )?;
    for (owner, body) in [("failed", "unreachable"), ("peer", capacity::FINISH)] {
        let raw = bytes(&o[format!("{owner}_module_bytes")])?;
        require(
            raw == capacity::module(body) && o[format!("{owner}_module_sha256")] == sha(&raw),
            "held drain module preimage",
        )?;
    }
    require(
        o["terminal"] == "trap"
            && o["exit_code"].is_null()
            && o["message"].as_str().is_some_and(|s| !s.is_empty()),
        "drain original failure",
    )?;
    let actual = stores(
        &o["store_results"],
        &o["activity"],
        Some(&["a/1", "b/1"]),
        true,
    )?;
    require(
        actual[0]["terminal"] == "trap"
            && actual[0]["exit_code"].is_null()
            && actual[0]["message"] == o["message"]
            && actual[1] == o["held_peer_result"]
            && actual[1]["terminal"] == "exit"
            && actual[1]["exit_code"] == 0
            && actual[1]["stdout"] == json!(b"mark"),
        "drain terminal ownership",
    )?;
    require(
        o["probe_millis"] == 50
            && o["probe_outcome"] == "timeout"
            && integer(&o["probe_elapsed_nanos"])? >= 50_000_000
            && integer(&o["release_to_return_nanos"])? < 5_000_000_000
            && o["join_error"].is_null(),
        "drain release/join bounds",
    )?;
    require(
        o["native_control_entry_points"] == json!(["activate", "run", "command"])
            && [
                "before_control_attempts",
                "held_control_attempts",
                "after_control_attempts",
            ]
            .iter()
            .all(|k| o[k] == 4),
        "post-failure native control attempt",
    )?;
    let raw = bytes(&o["before_lifecycle_bytes"])?;
    require(
        o["held_lifecycle_bytes"] == o["before_lifecycle_bytes"]
            && o["after_lifecycle_bytes"] == o["before_lifecycle_bytes"],
        "drain lifecycle changed",
    )?;
    let handles = ["a", "b"]
        .map(|node| Handle {
            node: node.into(),
            incarnation: 1,
        })
        .into_iter()
        .collect();
    let events = linked_inspection::lifecycle_records(&raw, &handles).map_err(|e| e.to_string())?;
    let starts = events
        .values()
        .filter(|(_, tag, _)| matches!(tag, 1 | 3))
        .map(|(_, tag, h)| (*tag, h.node.as_str()))
        .collect::<Vec<_>>();
    require(
        starts == [(1, "b"), (3, "b"), (1, "a"), (3, "a")],
        "held peer terminal lifecycle",
    )
}

/// Receives a closed fixture catalog. Unknown, duplicate and missing observations fail closed.
pub(crate) fn observations(
    stdout: &[u8],
    required: &BTreeSet<(String, String)>,
) -> Result<(), String> {
    require(
        !required.is_empty() && required.is_subset(&catalog()),
        "foreign lifecycle selection",
    )?;
    let text = std::str::from_utf8(stdout).map_err(|e| e.to_string())?;
    let mut seen = BTreeSet::new();
    for line in text.lines().filter(|line| line.contains(SCHEMA)) {
        let start = line.find('{').ok_or("native lifecycle row lacks JSON")?;
        let row = crate::cycle::custody::strict_json(&line.as_bytes()[start..])?;
        let root = if row.get("original_roots").is_some() {
            require(
                row["original_roots"] == json!(STORAGE),
                "foreign lifecycle roots",
            )?;
            STORAGE.join("|")
        } else {
            let root = row["original_root"].as_str().ok_or("lifecycle root")?;
            require(!root.contains('|'), "foreign lifecycle roots")?;
            root.to_owned()
        };
        let variant = row["variant"]
            .as_str()
            .ok_or("lifecycle variant")?
            .to_owned();
        let input = matches!(root.as_str(), SQLITE | ORACLE | BOOT);
        let key = if row.get("original_roots").is_some() {
            "original_roots"
        } else {
            "original_root"
        };
        let mut names = vec!["schema", key, "variant", "observed"];
        if input {
            names.push("inputs");
        }
        fields(&row, &names)?;
        require(
            row["schema"] == SCHEMA
                && required.contains(&(root.clone(), variant.clone()))
                && seen.insert((root.clone(), variant.clone())),
            "unknown or duplicate lifecycle case",
        )?;
        cases::validate(&root, &variant, &row["observed"], row.get("inputs"))?;
    }
    require(&seen == required, "lifecycle harness omitted cases")
}
