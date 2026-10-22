use crate::evidence::{canonical, store::Opened};
use gomad_wasmhost::linked_inspection;
use serde::Deserialize;
use serde_json::{Value, json};

#[derive(Deserialize)]
#[serde(tag = "type", rename_all = "lowercase", deny_unknown_fields)]
enum Frame {
    Header {
        schema: String,
    },
    Event {
        order: u64,
        kind: String,
        instance: String,
        incarnation: u64,
        ordinal: u64,
        trace: u64,
        decision: u64,
        time_nanos: i64,
        value: Value,
    },
    Boundary {
        order: u64,
        instance: String,
        incarnation: u64,
        trace: u64,
        decision: u64,
        time_nanos: i64,
        inspection_order: u64,
    },
    Complete {
        events: u64,
        bytes: u64,
    },
}

pub(super) struct Log {
    events: Vec<Value>,
    boundaries: Vec<Value>,
}

pub(super) fn parse(raw: &[u8]) -> Result<Log, String> {
    if raw.is_empty() || raw.len() > 32 << 20 || !raw.ends_with(b"\n") {
        return Err("missing or bounded complete supplemental event-order diagnostic".into());
    }
    let text = std::str::from_utf8(raw).map_err(|error| error.to_string())?;
    let mut log = Log {
        events: Vec::new(),
        boundaries: Vec::new(),
    };
    let mut header = false;
    let mut complete = false;
    let mut counted = 0;
    for line in text.split_inclusive('\n') {
        if complete {
            return Err("event-order frame follows terminal".into());
        }
        let payload = line
            .trim_end_matches('\n')
            .strip_prefix("gomad-wasm-event-order-v1 ")
            .ok_or("invalid event-order prefix")?;
        let frame: Frame = crate::qualify::spec::strict(payload.as_bytes())?;
        match frame {
            Frame::Header { schema } if !header && schema == "gomad.wasm-event-order/v1" => {
                header = true;
            }
            Frame::Event {
                order,
                kind,
                instance,
                incarnation,
                ordinal,
                trace,
                decision,
                time_nanos,
                value,
            } if header
                && order == log.events.len() as u64 + 1
                && log.events.len() < 131_072
                && !instance.is_empty()
                && (incarnation != 0 || instance == "controller")
                && matches!(
                    kind.as_str(),
                    "choice"
                        | "draw-counter"
                        | "clock"
                        | "entropy"
                        | "wasi-op"
                        | "model-transition"
                        | "output"
                        | "world-order"
                ) =>
            {
                log.events.push(json!({"order":order,"kind":kind,"instance":instance,"incarnation":incarnation,"ordinal":ordinal,"trace":trace,"decision":decision,"time_nanos":time_nanos,"value":value}));
            }
            Frame::Boundary {
                order,
                instance,
                incarnation,
                trace,
                decision,
                time_nanos,
                inspection_order,
            } if header
                && order == log.events.len() as u64
                && !instance.is_empty()
                && incarnation != 0
                && log.boundaries.len() < 131_072
                && log.boundaries.last().is_none_or(|previous| {
                    previous["inspection_order"].as_u64().unwrap() < inspection_order
                        && previous["order"].as_u64().unwrap() <= order
                }) =>
            {
                log.boundaries.push(json!({"order":order,"instance":instance,"incarnation":incarnation,"trace":trace,"decision":decision,"time_nanos":time_nanos,"inspection_order":inspection_order}));
            }
            Frame::Complete { events, bytes }
                if header && events == log.events.len() as u64 && bytes == counted as u64 =>
            {
                complete = true;
                continue;
            }
            _ => {
                return Err("invalid or incomplete event-order frame/count/byte commitment".into());
            }
        }
        counted += line.len();
    }
    if !complete || log.events.is_empty() {
        return Err("missing complete supplemental event-order diagnostic".into());
    }
    Ok(log)
}

pub(super) fn first(left: &Log, right: &Log) -> Option<Value> {
    (0..left.events.len().max(right.events.len())).find_map(|index| {
        let a = left.events.get(index);
        let b = right.events.get(index);
        if a == b { return None; }
        let at = a.or(b).unwrap();
        Some(json!({"kind":at["kind"],"instance":at["instance"],"incarnation":at["incarnation"],"ordinal":at["ordinal"],
            "coordinate":{"Shared":at["order"],"ChoiceOrdinal":at["trace"],"VirtualTime":at["time_nanos"]},
            "field":"ordered-event","expected":a.map_or_else(|| "absent".into(), Value::to_string),"actual":b.map_or_else(|| "absent".into(), Value::to_string)}))
    })
}

fn fresh(opened: &Opened) -> Result<(), String> {
    let clustered = opened
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE);
    let file = if clustered {
        crate::replay::cluster::FILE
    } else {
        crate::replay::preflight::FILE
    };
    let raw = opened.payload(file, 16 << 20)?;
    let inputs: Value = crate::qualify::spec::strict(&raw)?;
    let configuration = &inputs["configuration"];
    let models = if clustered {
        configuration["incarnations"]
            .as_array()
            .ok_or("missing stress incarnations")?
            .iter()
            .map(|item| &item["model"])
            .collect::<Vec<_>>()
    } else {
        vec![&configuration["model"]]
    };
    if models.is_empty()
        || models.iter().any(|model| {
            model["mode"] != 1
                || model.get("seed").is_some()
                || model.get("replay").is_some_and(|value| !value.is_null())
                || model.get("tape").is_some_and(|value| !value.is_null())
        })
    {
        return Err("forced retained execution is not a stress cell".into());
    }
    if canonical::digest(&canonical::encode(configuration)?)
        != opened.manifest().record.identity.configuration
    {
        return Err("retained stress configuration identity changed".into());
    }
    Ok(())
}

fn diagnostics(opened: &Opened) -> Result<(), String> {
    let clustered = opened
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE);
    let file = if clustered {
        crate::replay::cluster::FILE
    } else {
        crate::replay::preflight::FILE
    };
    let inputs: Value = crate::qualify::spec::strict(&opened.payload(file, 16 << 20)?)?;
    let prefixes = if clustered {
        inputs["configuration"]["incarnations"]
            .as_array()
            .ok_or("missing diagnostic incarnations")?
            .iter()
            .map(|item| {
                if item["model"]["diagnostics"] != true {
                    return Err("missing diagnostic trace; rerun with --diagnostics");
                }
                Ok(format!(
                    "evidence/cluster/{}/{}/",
                    item["handle"]["node"]
                        .as_str()
                        .ok_or("missing diagnostic owner")?,
                    item["handle"]["incarnation"]
                ))
            })
            .collect::<Result<Vec<_>, _>>()?
    } else {
        if inputs["configuration"]["model"]["diagnostics"] != true {
            return Err("missing diagnostic trace; rerun with --diagnostics".into());
        }
        vec!["evidence/".into()]
    };
    for prefix in prefixes {
        let choice = opened.payload(&format!("{prefix}choice"), 1 << 30)?;
        let diagnostic = opened.payload(&format!("{prefix}diagnostic"), 1 << 30)?;
        if diagnostic.is_empty() || diagnostic.len() % 96 != 0 || diagnostic.len() != choice.len() {
            return Err("missing or incomplete diagnostic trace; rerun with --diagnostics".into());
        }
    }
    Ok(())
}

fn metadata(opened: &Opened) -> Result<Vec<u8>, String> {
    let name = if opened.manifest().files.contains_key("evidence/inspection") {
        "evidence/inspection"
    } else {
        "evidence/inspection_metadata"
    };
    opened.payload(name, linked_inspection::MAX_METADATA as u64)
}

fn prefix(left: &Opened, right: &Opened, a: &Log, b: &Log, before: u64) -> Result<Value, String> {
    let expected_raw = metadata(left)?;
    let observed_raw = metadata(right)?;
    let expected =
        linked_inspection::Evidence::decode(&expected_raw).map_err(|error| error.to_string())?;
    if left
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE)
    {
        crate::replay::cluster::authenticate_inspection(left, &expected)?;
    }
    let observed =
        linked_inspection::Evidence::decode(&observed_raw).map_err(|error| error.to_string())?;
    if right
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE)
    {
        crate::replay::cluster::authenticate_inspection(right, &observed)?;
    }
    if !expected.complete || !observed.complete {
        return Err("incomplete retained inspection evidence".into());
    }
    let marker = a
        .boundaries
        .iter()
        .rev()
        .find(|marker| marker["order"].as_u64().unwrap() < before)
        .ok_or("no authenticated inspection boundary precedes the first differing event")?;
    if !b.boundaries.contains(marker) {
        return Err("preceding event-order marker differs".into());
    }
    let boundary = expected
        .boundaries
        .iter()
        .find(|boundary| {
            boundary.order == marker["inspection_order"].as_u64().unwrap()
                && boundary.handle.node == marker["instance"]
                && boundary.handle.incarnation == marker["incarnation"].as_u64().unwrap()
                && boundary.trace == marker["trace"].as_u64().unwrap()
                && boundary.decision == marker["decision"].as_u64().unwrap()
                && i64::try_from(boundary.time_nanos).ok() == marker["time_nanos"].as_i64()
        })
        .ok_or("event-order marker lacks retained inspection boundary")?;
    linked_inspection::verify_prefix(&expected, &observed_raw, boundary)
        .map_err(|error| error.to_string())?;
    Ok(
        json!({"instance":boundary.handle.node,"incarnation":boundary.handle.incarnation,"stop":{"kind":"choice","value":boundary.decision},"dumps":["stacks","heap"],"probes":null,"maximum_bytes":64 << 20,"per_dump_bytes":64 << 20,"output_directory":""}),
    )
}

pub(super) fn compare(left: &Value, right: &Value) -> Result<Value, String> {
    super::comparable(left, right)?;
    let a = Opened::open(std::path::Path::new(
        left["artifact"].as_str().ok_or("missing stress artifact")?,
    ))?;
    let b = Opened::open(std::path::Path::new(
        right["artifact"]
            .as_str()
            .ok_or("missing stress artifact")?,
    ))?;
    a.authenticate_all()?;
    b.authenticate_all()?;
    fresh(&a)?;
    fresh(&b)?;

    if a.manifest().record.identity != b.manifest().record.identity
        || a.manifest().record.seed != b.manifest().record.seed
    {
        return Err("stress artifact identity or seed changed".into());
    }
    let read_log = |opened: &Opened| {
        diagnostics(opened)?;
        parse(&opened.payload("evidence/event-order", 32 << 20)?)
    };
    let (expected, actual) = match (read_log(&a), read_log(&b)) {
        (Ok(a), Ok(b)) => (a, b),
        (a, b) => {
            return Ok(
                json!({"schema":"gomad.wasm-first-divergence/v1","status":"undiagnosable","reason":format!("retain --diagnostics and complete event-order evidence: {:?} {:?}",a.err(), b.err())}),
            );
        }
    };
    let mut a_record = a.manifest().record.clone();
    let mut b_record = b.manifest().record.clone();
    a_record.streams.remove("evidence/event-order");
    b_record.streams.remove("evidence/event-order");
    let Some(difference) = first(&expected, &actual) else {
        return Ok(if a_record == b_record {
            json!({"schema":"gomad.wasm-first-divergence/v1","status":"identical"})
        } else {
            json!({"schema":"gomad.wasm-first-divergence/v1","status":"undiagnosable","reason":"canonical evidence differs without matching event-order difference"})
        });
    };
    let mut result = json!({"schema":"gomad.wasm-first-divergence/v1","status":"diverged","ordering":"helper-event","virtual_time_available":true,"first":difference,
        "reproducer":{"workload":left["workload"],"perturbation":right["axis"],"seed":left["seed"],"identities":left["identities"],"expected_digest":a.manifest().record_sha256,"actual_digest":b.manifest().record_sha256,
        "expected_command":left["command"],"actual_command":right["command"],"artifact":left["artifact"]}});
    match prefix(
        &a,
        &b,
        &expected,
        &actual,
        result["first"]["coordinate"]["Shared"].as_u64().unwrap(),
    ) {
        Ok(stop) => {
            result["reproducer"]["stop"] = stop.clone();
            result["reproducer"]["replay_command"] = json!([
                left["binary"],
                "replay",
                "--stress-jitter",
                "--stop",
                format!("choice={}", stop["stop"]["value"]),
                "--instance",
                stop["instance"],
                "--incarnation",
                stop["incarnation"].as_u64().unwrap().to_string(),
                left["artifact"]
            ]);
        }
        Err(reason) => {
            result["status"] = json!("undiagnosable");
            result["reproducer"]["prefix_unavailable"] = json!(reason);
        }
    }
    result["reproducer"]["evidence_digest"] =
        json!(canonical::digest(&canonical::encode(&a_record)?));
    Ok(result)
}
