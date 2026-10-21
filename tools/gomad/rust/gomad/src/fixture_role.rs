use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde_json::{Value, json};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::Path,
    sync::Barrier,
    time::Duration,
};

fn selected(role: &str) -> Result<linked::Conditions, String> {
    match role {
        "ordinary" => Ok(linked::Conditions::Ordinary),
        "stress-jitter" => Ok(linked::Conditions::stress_replay()),
        "determinism-fixture" => Ok(linked::Conditions::determinism_fixture_replay()),
        _ => Err("unknown linked diagnostic fixture role".into()),
    }
}

fn execute(module: &[u8], conditions: linked::Conditions) -> Result<linked::Execution, String> {
    let configuration = serde_json::from_value(json!({
        "entropy_key": vec![0_u8;32], "working_directory_sha256": canonical::digest(b"/"),
        "arguments": {"bytes":0,"count":0,"chunks":0},
        "environment": {"bytes":0,"count":0,"chunks":0},
        "seed":7,"mode":1,"limit":4096,"diagnostics":false,
        "epoch_nanos":0,"read_step_nanos":0,"now":1,
        "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}
    }))
    .map_err(|error| error.to_string())?;
    let mut prepared = linked::prepare(linked::Request {
        module: module.to_vec(),
        module_sha256: canonical::digest(module),
        limits: linked::Limits {
            memory_bytes: 65_536,
            fuel: 100_000,
            output_bytes: 1024,
            evidence_bytes: 8192,
            wall_timeout: Duration::from_secs(5),
        },
        model: Some(linked::ModelInput {
            configuration,
            arguments: Vec::new(),
            environment: vec!["PWD=/".into()],
            replay_trace: None,
        }),
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(),
            stdin: Vec::new(),
            limits: linked::NamespaceLimits {
                files: 16,
                descriptors: 16,
                filesystem_bytes: 4096,
                output_bytes: 1024,
            },
            epoch_nanos: 0,
        }),
        storage: None,
    })
    .map_err(|error| error.to_string())?;
    prepared.conditions(conditions);
    linked::execute_prepared(prepared).map_err(|error| error.to_string())
}

fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .and_then(|mut file| file.write_all(bytes))
        .map_err(|error| error.to_string())
}

fn diagnostics(raw: Option<&[u8]>, ordinary: bool) -> Result<Value, String> {
    if ordinary {
        if raw.is_some() {
            return Err("ordinary execution retained diagnostic-role events".into());
        }
        return Ok(Value::Null);
    }
    let raw = raw.ok_or("diagnostic role omitted event-order bytes")?;
    let text = std::str::from_utf8(raw).map_err(|error| error.to_string())?;
    let rows = text
        .lines()
        .map(|line| {
            serde_json::from_str::<Value>(
                line.strip_prefix("gomad-wasm-event-order-v1 ")
                    .ok_or("invalid event-order prefix")?,
            )
            .map_err(|error| error.to_string())
        })
        .collect::<Result<Vec<_>, String>>()?;
    if rows.first().and_then(|row| row["type"].as_str()) != Some("header")
        || rows.last().and_then(|row| row["type"].as_str()) != Some("complete")
        || rows.iter().any(|row| row["type"] == "incomplete")
    {
        return Err("diagnostic role retained incomplete event-order log".into());
    }
    let events = rows.iter().filter(|row| row["type"] == "event").count();
    if events == 0 || rows.last().unwrap()["events"].as_u64() != Some(events as u64) {
        return Err("event-order count mismatch".into());
    }
    for (ordinal, row) in rows.iter().filter(|row| row["type"] == "event").enumerate() {
        if row["order"].as_u64() != Some(ordinal as u64 + 1) {
            return Err("event-order commit sequence mismatch".into());
        }
    }
    Ok(json!({"bytes":raw.len(),"sha256":canonical::digest(raw),"events":events}))
}

/// # Errors
/// Requires an explicit matching role and compares fresh repeated and concurrent executions.
pub(crate) fn run(root: &Path, corpus: &Path, role: &str) -> Result<(), String> {
    let conditions = selected(role)?;
    conditions.validate().map_err(|error| error.to_string())?;
    let module =
        fs::read(corpus.join("targets/inspect-choice.wat")).map_err(|error| error.to_string())?;
    fs::DirBuilder::new()
        .mode(0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let first = execute(&module, conditions)?;
    let second = execute(&module, conditions)?;
    let barrier = Barrier::new(2);
    let concurrent = std::thread::scope(|scope| {
        let left = scope.spawn(|| {
            barrier.wait();
            execute(&module, conditions)
        });
        let right = scope.spawn(|| {
            barrier.wait();
            execute(&module, conditions)
        });
        Ok::<_, String>((
            left.join().map_err(|_| "role fixture worker panicked")??,
            right.join().map_err(|_| "role fixture worker panicked")??,
        ))
    })?;
    let mut executions = Vec::new();
    for (name, execution) in [
        ("first", &first),
        ("second", &second),
        ("concurrent-left", &concurrent.0),
        ("concurrent-right", &concurrent.1),
    ] {
        if execution.outcome.termination != "exit"
            || execution.outcome.exit_code != Some(0)
            || execution.evidence != first.evidence
            || execution.event_order != first.event_order
        {
            return Err(format!("{name}: fresh role execution differs"));
        }
        let events = diagnostics(
            execution.event_order.as_deref(),
            conditions == linked::Conditions::Ordinary,
        )?;
        if let Some(raw) = &execution.event_order {
            write(&root.join(format!("{name}-event-order.log")), raw)?;
        }
        executions.push(json!({"name":name,"termination":execution.outcome.termination,
            "exit_code":execution.outcome.exit_code,"event_order":events,
            "evidence":execution.evidence.iter().map(|(stream,raw)|
                (stream.clone(),json!({"bytes":raw.len(),"sha256":canonical::digest(raw)}))).collect::<std::collections::BTreeMap<_,_>>()}));
    }
    let report = json!({"schema":"gomad.replay-role-fixture/v1","role":role,
        "configuration":conditions.configuration(),"helper_sha256":linked::IMPLEMENTATION_SHA256,
        "module_sha256":canonical::digest(&module),"executions":executions});
    let raw = canonical::encode(&report)?;
    write(&root.join("role-result.json"), &raw)?;
    println!(
        "{}",
        String::from_utf8(raw).map_err(|error| error.to_string())?
    );
    Ok(())
}
