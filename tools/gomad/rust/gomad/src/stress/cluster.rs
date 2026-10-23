use crate::{
    campaign::{execution, ledger::Job, prepare::Prepared, scenario},
    evidence::store::Publication,
};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde_json::{Value, json};
use sha2::{Digest as _, Sha256};
use std::{collections::BTreeMap, time::Duration};

const OBSERVATION: &str = "GOMAD_TEMPORAL_OBSERVATION ";

pub(crate) fn original_limits() -> Value {
    json!({"nodes":64,"directional_links":4096,"volumes":64,"scenario_actions":4096,
        "scenario_decisions":4096,"fault_actions":4096,"history_operations":65536,
        "observations":65536,"oracle_results":4096,"scenario_evidence_bytes":64 << 20,
        "boot_config_bytes":1 << 20,"volume_capacity_bytes":1 << 30,"observation_bytes":64 << 20,
        "network_listeners":4096,"network_connections":65536,"network_bytes":1 << 30,
        "network_connection_bytes":4 << 20,"network_stall_nanos":10_000_000_000_u64,
        "network_transitions":1 << 20,"volume_operations":1 << 20,"volume_transitions":1 << 20,
        "crash_states":65536,"crash_depth":256,"crash_bytes":128 << 20,"crash_wall_nanos":60_000_000_000_u64})
}

pub(super) fn validate_scenario(raw: &[u8]) -> Result<(), String> {
    let scenario: Value = crate::qualify::spec::strict(raw)?;
    if scenario["spec"]["limits"] != original_limits() {
        return Err("stress requires original two-cluster scenario bounds".into());
    }
    Ok(())
}

pub(crate) fn bind(bound: &mut scenario::Bound) -> Result<(), String> {
    if bound.request.nodes != ["active", "standby"] || bound.request.incarnations.len() != 4 {
        return Err("stress requires original active/standby four-incarnation scenario".into());
    }
    for incarnation in &mut bound.request.incarnations {
        let node = incarnation.handle.node.as_str();
        let peer = match node {
            "active" => "10.0.0.2",
            "standby" => "10.0.0.1",
            _ => return Err("unknown stress cluster node".into()),
        };
        let phase = match incarnation.handle.incarnation {
            1 => "commit",
            2 => "reopen",
            _ => return Err("unknown stress incarnation".into()),
        };
        let input = incarnation
            .request
            .model
            .as_mut()
            .ok_or("stress cluster requires cooperative model")?;
        for (name, value) in [
            ("GOMAD_TEMPORAL_CLUSTER", node.to_owned()),
            ("GOMAD_TEMPORAL_PHASE", phase.to_owned()),
            ("GOMAD_TEMPORAL_REMOTE_FRONTEND", format!("{peer}:7233")),
            ("GOMAD_TEMPORAL_REMOTE_HTTP", format!("{peer}:7243")),
            ("GOMAD_TEMPORAL_REMOTE_HISTORY", format!("{peer}:7234")),
        ] {
            if input
                .environment
                .iter()
                .any(|entry| entry.starts_with(&format!("{name}=")))
            {
                return Err("stress role environment already bound".into());
            }
            input.environment.push(format!("{name}={value}"));
        }
        input.configuration.entropy_key =
            Sha256::digest(format!("gomad-wasm-temporal-replication/v1/{node}").as_bytes()).into();
        let retained = bound
            .configuration
            .incarnations
            .iter_mut()
            .find(|entry| entry.handle == incarnation.handle)
            .ok_or("stress retained incarnation missing")?;
        retained.environment = input
            .environment
            .iter()
            .map(|entry| {
                entry
                    .split_once('=')
                    .map(|(name, value)| (name.to_owned(), value.to_owned()))
                    .ok_or("invalid stress environment")
            })
            .collect::<Result<BTreeMap<_, _>, _>>()?;
        retained.model["entropy_key"] = json!(input.configuration.entropy_key);
    }
    bound.request.boundaries.clear();
    bound.configuration.boundaries.clear();
    let group = helper::CrashGroup {
        nodes: vec!["active".into(), "standby".into()],
        marker: OBSERVATION.as_bytes().to_vec(),
    };
    bound.configuration.crash_groups = vec![group.clone()];
    bound.request.crash_groups = vec![group];
    Ok(())
}

pub(super) fn validate_original(bound: &scenario::Bound) -> Result<(), String> {
    let steps = [
        (helper::Action::Start, 1),
        (helper::Action::Crash, 1),
        (helper::Action::Restart, 2),
        (helper::Action::Wait, 2),
    ]
    .into_iter()
    .flat_map(|(action, incarnation)| {
        ["active", "standby"].map(move |node| helper::Step {
            action,
            handle: helper::Handle {
                node: node.into(),
                incarnation,
            },
        })
    })
    .collect::<Vec<_>>();
    if bound.request.steps != steps {
        return Err("stress two-cluster lifecycle changed".into());
    }
    let network = bound
        .request
        .network
        .as_ref()
        .ok_or("stress two-cluster network missing")?;
    let expected = json!({"model":{"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":4096,"bytes":128 << 20,"connection_bytes":4 << 20,"stall_nanos":10_000_000_000_u64,"transitions":1 << 20},"faults":[]},
        "addresses":{"active":"10.0.0.1","standby":"10.0.0.2"},"strict_topology":true,
        "links":[{"from":"active","to":"standby","enabled":true,"delay_nanos":0},{"from":"standby","to":"active","enabled":true,"delay_nanos":0}],
        "faults":[{"name":"temporal-active-to-standby-partition","action":"partition","from":"active","to":"standby","at_nanos":1,"delay_nanos":0,"symmetric":false},{"name":"temporal-active-to-standby-heal","action":"heal","from":"active","to":"standby","at_nanos":30_000_000_000_u64,"delay_nanos":0,"symmetric":false}]});
    if serde_json::to_value(network).map_err(|error| error.to_string())? != expected {
        return Err("stress two-cluster topology, fault schedule or network bounds changed".into());
    }
    for item in &bound.request.incarnations {
        let model = item
            .request
            .model
            .as_ref()
            .ok_or("missing stress boot model")?;
        let args = [
            "-test.v",
            "-test.count=1",
            "-test.timeout=1h",
            "-test.run=^TestIndependentClusterReplicationRecovery$",
            "-persistenceType=sql",
            "-persistenceDriver=sqlite",
        ];
        if model
            .arguments
            .get(1..)
            .is_none_or(|arguments| arguments != args)
        {
            return Err("stress two-cluster boot arguments changed".into());
        }
        let storage = serde_json::to_value(
            item.request
                .storage
                .as_ref()
                .ok_or("missing stress durable volume")?,
        )
        .map_err(|error| error.to_string())?;
        let storage_model = &storage["descriptor"]["model"];
        if storage_model["volumes"]
            != json!([{"ID":item.handle.node,"Path":"/volume","CapacityBytes":256 << 20}])
            || storage_model["limits"]
                != json!({"PendingOperations":100_000,"Transitions":10_000_000})
            || storage_model["sqlite_lock_databases"] != json!(["/volume/temporal.db"])
            || storage_model["faults"]
                .as_array()
                .is_some_and(|faults| !faults.is_empty())
            || !storage["image"].is_null()
            || !storage["context"].is_null()
        {
            return Err("stress two-cluster durable inputs changed".into());
        }
    }
    Ok(())
}

pub(super) fn execute(
    prepared: &Prepared,
    seed: u64,
    timeout: Duration,
) -> Result<execution::Completion, String> {
    let mut bound = scenario::bind(
        prepared
            .scenario
            .as_deref()
            .ok_or("missing stress scenario")?,
        &prepared.configuration,
        &prepared.module,
        prepared
            .choice_identity
            .as_ref()
            .ok_or("missing cooperative identity")?,
        prepared.network.as_ref(),
        seed,
    )?;
    validate_original(&bound)?;
    bind(&mut bound)?;
    bound.request.wall_timeout = bound.request.wall_timeout.min(timeout);
    let mut admitted = helper::prepare(bound.request).map_err(|error| error.to_string())?;
    admitted.conditions(linked::Conditions::compiled());
    let observed = crate::backstop::within(admitted.wall_timeout(), || {
        helper::execute_prepared(admitted)
    })
    .map_err(|error| error.to_string())?;
    let oracle = verify(&observed);
    let result = scenario::publication(
        prepared,
        &Job { ordinal: 1, seed },
        bound.configuration,
        observed,
        false,
    )?;
    let mut result = match result {
        execution::Completion::Finalized(result) => result,
        execution::Completion::Failed(failure) => {
            return Ok(execution::Completion::Failed(failure));
        }
    };
    let raw = crate::evidence::canonical::encode(
        &json!({"passed":oracle.is_ok(),"reason":oracle.err()}),
    )?;
    result.publication.record.streams.insert(
        "evidence/stress-oracles.json".into(),
        crate::evidence::record::Stream {
            bytes: raw.len() as u64,
            sha256: crate::evidence::canonical::digest(&raw),
            records: 1,
        },
    );
    result
        .publication
        .payloads
        .insert("evidence/stress-oracles.json".into(), raw);
    result
        .publication
        .payloads
        .insert("record.json".into(), result.publication.record.bytes()?);
    Ok(execution::Completion::Finalized(result))
}

pub(crate) fn observation(stdout: &[u8]) -> Result<Value, String> {
    let text = std::str::from_utf8(stdout).map_err(|error| error.to_string())?;
    let rows = text
        .lines()
        .filter_map(|line| line.strip_prefix(OBSERVATION))
        .collect::<Vec<_>>();
    if rows.len() != 1 {
        return Err("exactly one detached Temporal observation required".into());
    }
    let value: Value = crate::qualify::spec::strict(rows[0].as_bytes())?;
    let ids = value["event_ids"]
        .as_array()
        .ok_or("missing Temporal history IDs")?;
    let types = value["types"]
        .as_array()
        .ok_or("missing Temporal history types")?;
    let history = value["history"]
        .as_str()
        .and_then(crate::qualify::cells::base64)
        .ok_or("invalid detached Temporal history base64")?;
    if history.is_empty()
        || value["cluster_id"].as_str().is_none_or(str::is_empty)
        || value["history"].as_str().is_none_or(str::is_empty)
        || ids.len() != types.len()
        || ids
            .iter()
            .enumerate()
            .any(|(index, id)| id.as_u64() != Some(index as u64 + 1))
    {
        return Err("duplicate or lost detached Temporal history event".into());
    }
    Ok(value)
}

pub(crate) fn history_oracle(expected: &Value, actual: &Value) -> Result<(), String> {
    let operation = |value: &Value| {
        json!([{"id":"committed","actor":"workflow",
        "kind":"workflow-history","invocation":1,"completion":2,"output":value["history"]}])
    };
    let ids = |value: &Value| {
        value["event_ids"]
            .as_array()
            .unwrap()
            .iter()
            .map(Value::to_string)
            .collect::<Vec<_>>()
    };
    let history_bytes = serde_json::to_vec(&operation(expected))
        .map_err(|error| error.to_string())?
        .len()
        + serde_json::to_vec(&operation(actual))
            .map_err(|error| error.to_string())?
            .len();
    let event_bytes = serde_json::to_vec(&ids(expected))
        .map_err(|error| error.to_string())?
        .len()
        + serde_json::to_vec(&ids(actual))
            .map_err(|error| error.to_string())?
            .len();
    if history_bytes > 1 << 20 || event_bytes > 1 << 20 {
        return Err(
            "detached Temporal history/event oracle exceeded original 1MiB evidence bound".into(),
        );
    }
    Ok(())
}

fn verify(observed: &helper::Execution) -> Result<(), String> {
    if observed.incarnations.len() != 4 {
        return Err("two-cluster stress requires four executed incarnations".into());
    }
    let mut initial = BTreeMap::new();
    let mut history = None;
    for incarnation in &observed.incarnations {
        let output = incarnation
            .execution
            .evidence
            .get("stdout")
            .ok_or("missing cluster stdout")?;
        let value = observation(output)?;
        let outcome = &incarnation.execution.outcome;
        if incarnation.handle.incarnation == 1 {
            if outcome.termination != "crash"
                || !incarnation.execution.evidence.contains_key("storage_image")
            {
                return Err("committed incarnation did not retain crashed durable state".into());
            }
            if history
                .as_ref()
                .is_some_and(|expected| expected != &value["history"])
                || initial
                    .values()
                    .any(|previous: &Value| previous["cluster_id"] == value["cluster_id"])
            {
                return Err("independent histories did not converge or identities coincide".into());
            }
            history = Some(value["history"].clone());
            initial.insert(incarnation.handle.node.clone(), value);
        }
    }
    if initial.len() != 2 {
        return Err("missing original active/standby commits".into());
    }
    for incarnation in &observed.incarnations {
        if incarnation.handle.incarnation != 2 {
            continue;
        }
        let value = observation(
            incarnation
                .execution
                .evidence
                .get("stdout")
                .ok_or("missing reopened stdout")?,
        )?;
        let original = initial
            .get(&incarnation.handle.node)
            .ok_or("reopened node lacks committed incarnation")?;
        history_oracle(original, &value)?;
        if incarnation.execution.outcome.termination != "exit"
            || incarnation.execution.outcome.exit_code != Some(0)
            || value["history"] != original["history"]
            || value["cluster_id"] != original["cluster_id"]
            || value["event_ids"] != original["event_ids"]
            || value["types"] != original["types"]
        {
            return Err(
                "committed history or cluster identity changed after whole-cluster crash".into(),
            );
        }
    }
    Ok(())
}

pub(super) fn canonical_outcome(name: &str, publication: &Publication) -> Result<(), String> {
    let outcome = &publication.record.outcome;
    let intentional_canary = name == "runtime"
        && outcome.class == "target_failure"
        && outcome.termination == "exit"
        && outcome.exit_code == Some(17)
        && !publication
            .payloads
            .contains_key("evidence/soft-assertions.json")
        && publication
            .payloads
            .get("output/stdout")
            .is_some_and(|stdout| {
                stdout
                    .windows(b"canary 2\n".len())
                    .any(|window| window == b"canary 2\n")
            });
    if !intentional_canary
        && (outcome.class != "success"
            || outcome.termination != "exit"
            || outcome.exit_code != Some(0))
    {
        return Err(
            "stress requires a successful canonical outcome or the original runtime exit17 canary"
                .into(),
        );
    }
    Ok(())
}

pub(super) fn outcome(
    name: &str,
    prepared: &Prepared,
    publication: &Publication,
) -> Result<(), String> {
    canonical_outcome(name, publication)?;
    if name == "two-cluster" {
        let report: Value = crate::qualify::spec::strict(
            publication
                .payloads
                .get("evidence/stress-oracles.json")
                .ok_or("missing stress cluster oracles")?,
        )?;
        if report["passed"] != true {
            return Err(format!("two-cluster oracle failed: {}", report["reason"]));
        }
        return Ok(());
    }
    let outcome = &publication.record.outcome;
    let stdout = publication
        .payloads
        .get("output/stdout")
        .ok_or("stress guest omitted stdout")?;
    let text = String::from_utf8_lossy(stdout);
    if name == "runtime" {
        for witness in [
            "entropy ",
            "map ",
            "runq ",
            "timer callbacks ",
            "timer parking",
            "note negative park wake",
            "collector automatic finalizer cleanup 1 1",
            "canary ",
        ] {
            if !text.contains(witness) {
                return Err(format!("runtime stress lacks {witness:?}"));
            }
        }
        if outcome.exit_code == Some(17) && !text.contains("canary 2\n") {
            return Err("runtime exit17 lacks intentional canary witness".into());
        }
        let diagnostic = publication
            .payloads
            .get("evidence/diagnostic")
            .ok_or("missing runtime diagnostic trace")?;
        if diagnostic.len() < 96 || diagnostic.len() % 96 != 0 {
            return Err("invalid runtime diagnostic trace".into());
        }
        let trace = publication
            .payloads
            .get("evidence/choice")
            .ok_or("missing runtime choice trace")?;
        let inspected = linked::inspect_choices(
            trace,
            prepared
                .choice_identity
                .as_ref()
                .ok_or("missing runtime choice identity")?,
        )?;
        if inspected.runnable == 0 || inspected.select_poll == 0 || inspected.select_result == 0 {
            return Err("runtime stress lacks runnable/select hook witnesses".into());
        }
        if trace.len() != diagnostic.len() {
            return Err("runtime diagnostic count differs from choice count".into());
        }
        for (ordinal, row) in diagnostic.as_chunks::<96>().0.iter().enumerate() {
            if u64::from_be_bytes(row[..8].try_into().unwrap()) != ordinal as u64
                || row[28] > 2
                || row[29..32] != [0; 3]
                || row[36..40] != [0; 4]
            {
                return Err("invalid runtime diagnostic record".into());
            }
        }
        let last = &diagnostic[diagnostic.len() - 96..];
        if u32::from_be_bytes(last[24..28].try_into().unwrap()) == 0 {
            return Err("runtime stress lacks collector cycle witness".into());
        }
        for at in [16, 40, 56, 64, 80] {
            if u64::from_be_bytes(last[at..at + 8].try_into().unwrap()) == 0 {
                return Err("runtime stress lacks collector or seeded draw witness".into());
            }
        }
    } else {
        let witness = if name == "workflow" {
            "--- PASS: TestWorkflowTypeEncodingSuite"
        } else {
            "--- PASS: TestActivityTestSuite"
        };
        if !text.contains(witness) || text.contains("--- SKIP:") {
            return Err("unchanged stress selection did not execute fully".into());
        }
    }
    Ok(())
}
