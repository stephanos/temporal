//! Original named-reset witness over the actual retained network owner transcript.
use crate::{
    evidence::store::Opened,
    qualify::{result::QualificationResult, spec},
};
use serde_json::Value;

fn number(value: &Value, key: &str) -> Result<u64, String> {
    value[key]
        .as_u64()
        .ok_or_else(|| format!("invalid reset witness {key}"))
}

fn observation(raw: &[u8]) -> Result<Value, String> {
    let text = std::str::from_utf8(raw).map_err(|_| "Temporal reset output is not UTF-8")?;
    let mut lines = text
        .lines()
        .filter_map(|line| line.strip_prefix("GOMAD_NETWORK_TEMPORAL_OBSERVATION "));
    let value: Value = spec::strict(
        lines
            .next()
            .ok_or("Temporal reset observation absent")?
            .as_bytes(),
    )?;
    let ids = value["event_ids"]
        .as_array()
        .ok_or("Temporal reset events absent")?;
    if lines.next().is_some()
        || value["workflow_id"].as_str().is_none_or(str::is_empty)
        || value["run_id"].as_str().is_none_or(str::is_empty)
        || value["result"] != "durable timer survived rpc-reset"
        || value["history"]
            .as_str()
            .and_then(super::super::cells::base64)
            .is_none_or(|bytes| bytes.is_empty())
        || ids.is_empty()
        || value["types"]
            .as_array()
            .is_none_or(|types| types.len() != ids.len())
        || ids
            .iter()
            .enumerate()
            .any(|(index, id)| id.as_u64() != Some(index as u64 + 1))
    {
        return Err("invalid independent Temporal reset history observation".into());
    }
    Ok(value)
}

pub(super) fn witness(stdout: &[u8], network: &[u8]) -> Result<(), String> {
    let value = observation(stdout)?;
    let connections = value["connections"]
        .as_array()
        .ok_or("Temporal reset connections absent")?;
    let ports = connections
        .iter()
        .filter(|connection| {
            connection["opened_nanos"]
                .as_u64()
                .is_some_and(|at| at < 39_999_999_999)
        })
        .map(|connection| number(connection, "local_port"))
        .collect::<Result<Vec<_>, _>>()?;
    if ports.is_empty() {
        return Err("no SDK connection existed before the named reset".into());
    }
    let transcript: Value = spec::strict(network)?;
    let operations = transcript["operations"]
        .as_array()
        .ok_or("Temporal reset network operations absent")?;
    let mut handles = std::collections::BTreeSet::new();
    let mut pending = std::collections::BTreeMap::new();
    let (mut fault, mut reset_error, mut reconnected) = (false, false, false);
    for operation in operations {
        let at = number(operation, "at")?;
        let request = &operation["request"];
        let result = &operation["result"];
        fault |= operation["fault"]["name"] == "rpc-reset"
            && operation["fault"]["action"] == "reset"
            && operation["fault"]["at_nanos"] == 40_000_000_000u64;
        if request["op"] == "dial"
            && result["state"] == "completed"
            && result["error"].as_str().unwrap_or("").is_empty()
            && result["handle"].as_u64().is_some_and(|handle| handle != 0)
        {
            let port = number(&result["local"], "port")?;
            if at < 40_000_000_000 && ports.contains(&port) {
                handles.insert(number(result, "handle")?);
            }
            reconnected |= at >= 40_000_000_000
                && connections.iter().any(|connection| {
                    connection["opened_nanos"]
                        .as_u64()
                        .is_some_and(|at| at >= 39_999_999_999)
                        && connection["local_port"].as_u64() == Some(port)
                });
        }
        if request["op"] == "read" || request["op"] == "write" {
            pending.insert(number(request, "request_id")?, number(request, "handle")?);
        }
        let handle = if request["op"] == "poll" {
            pending
                .get(&number(request, "request_id")?)
                .copied()
                .unwrap_or(0)
        } else {
            request["handle"].as_u64().unwrap_or(0)
        };
        reset_error |= at >= 40_000_000_000
            && handles.contains(&handle)
            && matches!(request["op"].as_str(), Some("read" | "write" | "poll"))
            && matches!(
                result["error"].as_str(),
                Some("use of closed network connection" | "connection reset by peer")
            );
    }
    let mut failed = false;
    let mut retried = false;
    for attempt in value["attempts"]
        .as_array()
        .ok_or("Temporal reset RPC attempts absent")?
    {
        let started = number(attempt, "started_nanos")?;
        let ended = number(attempt, "ended_nanos")?;
        failed |=
            started < 39_999_999_999 && ended >= 39_999_999_999 && attempt["code"] == "Unavailable";
        retried |= failed && started >= 39_999_999_999 && attempt["code"] == "OK";
    }
    if !fault || !reset_error || !reconnected || !retried {
        return Err("named reset lacks original established stream error, fresh SDK connection and in-flight RPC retry".into());
    }
    Ok(())
}

pub(super) fn retained(result: &QualificationResult) -> Result<(), String> {
    for execution in &result.report.executions {
        let opened = Opened::open(std::path::Path::new(
            execution
                .artifact_path
                .as_deref()
                .ok_or("Temporal retry artifact absent")?,
        ))?;
        opened.authenticate_all()?;
        if opened.manifest().record != execution.record
            || execution.record.outcome.class != "success"
            || execution
                .replay
                .as_ref()
                .is_none_or(|replay| !replay.matched)
        {
            return Err("Temporal named-reset recording/replay failed".into());
        }
        let inputs: crate::replay::preflight::Inputs =
            spec::strict(&opened.payload(crate::replay::preflight::FILE, 16 << 20)?)?;
        if crate::evidence::canonical::digest(&crate::evidence::canonical::encode(
            &inputs.configuration,
        )?) != execution.record.identity.configuration
        {
            return Err("named reset actual configuration binding changed".into());
        }
        super::recovery::validate_retry(&inputs.configuration)?;
        witness(
            &opened.payload("evidence/stdout", 256 << 20)?,
            &opened.payload("evidence/network", 1 << 30)?,
        )?;
    }
    Ok(())
}
