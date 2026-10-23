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
    parse_frames(raw, false)
}

fn parse_frames(raw: &[u8], diagnostic_empty: bool) -> Result<Log, String> {
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
    if !complete || !diagnostic_empty && log.events.is_empty() {
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
        let field = match (a,b) {
            (Some(a),Some(b)) => match at["kind"].as_str().unwrap() {
                "draw-counter" => ["runq_draws","scheduler_draws","select_draws","runtime_rand_draws","runtime_cheaprand_draws","timer_draws","clock_tick_draws"].into_iter().find(|field| a["value"][field] != b["value"][field]).unwrap_or("ordered-event"),
                "clock" => "operation",
                "world-order" => "step",
                _=>"ordered-event",
            },
            _=>"presence",
        };
        let diagnostic_time = match (a,b) {
            (Some(a),Some(b)) if a["kind"]=="model-transition" && b["kind"]=="model-transition"
                && a["value"]["model"]=="runtime-witness" && b["value"]["model"]=="runtime-witness"
                && a["value"]["virtual_time"].as_i64().is_some() && b["value"]["virtual_time"].as_i64().is_some()
                && a["value"]["virtual_time"]!=b["value"]["virtual_time"] => Some(a["value"]["virtual_time"].clone()),
            _=>None,
        };
        let kind=if diagnostic_time.is_some() {json!("clock")} else {at["kind"].clone()};
        let field=if diagnostic_time.is_some() {"virtual_time"} else {field};
        let virtual_time=diagnostic_time.unwrap_or_else(||at["time_nanos"].clone());
        Some(json!({"kind":kind,"instance":at["instance"],"incarnation":at["incarnation"],"ordinal":at["ordinal"],
            "coordinate":{"Shared":at["order"],"ChoiceOrdinal":at["trace"],"VirtualTime":virtual_time},
            "field":field,"expected":a.map_or_else(|| "absent".into(), Value::to_string),"actual":b.map_or_else(|| "absent".into(), Value::to_string)}))
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
        if diagnostic.len() % 96 != 0 || diagnostic.len() != choice.len() {
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
    crate::replay::inspection::authenticate_streams(left, &expected)
        .map_err(|error| error.message)?;
    let observed =
        linked_inspection::Evidence::decode(&observed_raw).map_err(|error| error.to_string())?;
    crate::replay::inspection::authenticate_streams(right, &observed)
        .map_err(|error| error.message)?;
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
    compare_inner(left, right, None)
}

#[cfg(all(test, feature = "helper-determinism-fixture"))]
pub(super) fn compare_counterfactual(left: &Value, right: &Value) -> Result<Value, String> {
    let role = gomad_wasmhost::linked::Conditions::DeterminismFixtureReplay;
    role.validate().map_err(|error| error.to_string())?;
    compare_inner(left, right, Some(role))
}

fn compare_inner(
    left: &Value,
    right: &Value,
    required_role: Option<gomad_wasmhost::linked::Conditions>,
) -> Result<Value, String> {
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
    if let Some(required) = required_role
        && replay_role(&a, &b)? != required
    {
        return Err("counterfactual replay role is not the compiled fixture role".into());
    }
    let read_log = |opened: &Opened| {
        diagnostics(opened)?;
        parse(&opened.payload("evidence/event-order", 32 << 20)?)
    };
    let (a_opened, b_opened) = (&a, &b);
    let (expected, actual) = match (read_log(&a), read_log(&b)) {
        (Ok(a), Ok(b)) => (a, b),
        (a, b) => {
            let mut report = json!({"schema":"gomad.wasm-first-divergence/v1","status":"undiagnosable","reason":format!("retain --diagnostics and complete event-order evidence: {:?} {:?}",a.err(), b.err())});
            supplement_inspection(&mut report, a_opened, b_opened);
            return Ok(report);
        }
    };
    let mut a_record = a.manifest().record.clone();
    let mut b_record = b.manifest().record.clone();
    a_record.streams.remove("evidence/event-order");
    b_record.streams.remove("evidence/event-order");
    let Some(difference) = first(&expected, &actual) else {
        let mut report = if a_record == b_record {
            json!({"schema":"gomad.wasm-first-divergence/v1","status":"identical"})
        } else {
            json!({"schema":"gomad.wasm-first-divergence/v1","status":"undiagnosable","reason":"canonical evidence differs without matching event-order difference"})
        };
        if report["status"] == "undiagnosable" {
            supplement_inspection(&mut report, &a, &b);
        }
        return Ok(report);
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
    )
    .and_then(|stop| Ok((stop, replay_flag(replay_role(&a, &b)?)?)))
    {
        Ok((stop, flag)) => {
            result["reproducer"]["stop"] = stop.clone();
            let mut command = vec![left["binary"].clone(), json!("replay")];
            if let Some(flag) = flag {
                command.push(json!(flag));
            }
            command.extend([
                json!("--stop"),
                json!(format!("choice={}", stop["stop"]["value"])),
                json!("--instance"),
                stop["instance"].clone(),
                json!("--incarnation"),
                json!(stop["incarnation"].as_u64().unwrap().to_string()),
                left["artifact"].clone(),
            ]);
            result["reproducer"]["replay_command"] = json!(command);
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

pub(super) fn extract_diagnostic(raw: &[u8]) -> Result<Value, String> {
    const PREFIX: &str = "gomad-wasm-event-order-v1 ";
    if raw.len() > 32 << 20 {
        return Err("supplemental stderr diagnostic capacity".into());
    }
    let mut frames = String::new();
    let mut events = 0u64;
    let mut complete = false;
    let mut terminal = false;
    let mut reason = "missing complete supplemental event-order diagnostic".to_string();
    for line in raw.split_inclusive(|byte| *byte == b'\n') {
        let Some(payload) = line.strip_prefix(PREFIX.as_bytes()) else {
            continue;
        };
        if terminal {
            return Err("event-order frame follows terminal".into());
        }
        if !line.ends_with(b"\n") {
            reason = "truncated supplemental event-order frame".into();
            break;
        }
        let line = std::str::from_utf8(line).map_err(|error| error.to_string())?;
        let payload = std::str::from_utf8(payload).map_err(|error| error.to_string())?;
        let value: Value = crate::qualify::spec::strict(payload.trim_end_matches('\n').as_bytes())?;
        if value["type"] == "incomplete" {
            #[derive(Deserialize)]
            #[serde(deny_unknown_fields)]
            struct Incomplete {
                #[serde(rename = "type")]
                kind: String,
                reason: String,
            }
            let marker: Incomplete =
                crate::qualify::spec::strict(payload.trim_end_matches('\n').as_bytes())?;
            if marker.kind != "incomplete" || marker.reason.is_empty() || marker.reason.len() > 4096
            {
                return Err("invalid incomplete event-order diagnostic".into());
            }
            reason = marker.reason;
            terminal = true;
            continue;
        }
        let frame: Frame = crate::qualify::spec::strict(payload.trim_end_matches('\n').as_bytes())?;
        if matches!(frame, Frame::Event { .. }) {
            events += 1;
        }
        complete = matches!(frame, Frame::Complete { .. });
        terminal = complete;
        frames.push_str(line);
    }
    let log = if complete {
        parse_frames(frames.as_bytes(), true)?
    } else if !frames.is_empty() {
        // A synthetic commitment validates a diagnostic prefix with the same
        // strict frame rules. It is never retained or reported as a completion.
        let candidate = format!(
            "{frames}{PREFIX}{}\n",
            json!({"type":"complete","events":events,"bytes":frames.len()})
        );
        parse_frames(candidate.as_bytes(), true)?
    } else {
        Log {
            events: Vec::new(),
            boundaries: Vec::new(),
        }
    };
    Ok(
        json!({"schema":"gomad.stderr-event-order-diagnostic/v1","diagnostic_only":true,"complete":complete,"reason":if complete {""} else {&reason},"events":log.events,"boundaries":log.boundaries}),
    )
}

fn supplement_inspection(report: &mut Value, left: &Opened, right: &Opened) {
    match super::inspection::compare(left, right) {
        Ok(diagnostic) => report["inspection_diagnostic"] = diagnostic,
        Err(reason) => report["inspection_diagnostic_unavailable"] = json!(reason),
    }
}

pub(super) fn configuration_role(
    configuration: &Value,
    clustered: bool,
) -> Result<gomad_wasmhost::linked::Conditions, String> {
    use gomad_wasmhost::linked::Conditions;
    let models = if clustered {
        configuration["incarnations"]
            .as_array()
            .filter(|items| !items.is_empty() && items.len() <= 65_536)
            .ok_or("missing bounded cluster replay role")?
            .iter()
            .collect::<Vec<_>>()
    } else {
        vec![configuration]
    };
    let engine = models[0]["engine"]
        .as_str()
        .ok_or("missing retained replay role")?;
    if models
        .iter()
        .any(|model| model["engine"].as_str() != Some(engine))
    {
        return Err("cluster incarnations disagree on retained replay role".into());
    }
    let role = [
        Conditions::Ordinary,
        Conditions::StressReplay,
        Conditions::DeterminismFixtureReplay,
        Conditions::DiagnosticProfile,
    ]
    .into_iter()
    .find(|role| role.configuration() == engine)
    .ok_or("unknown retained replay role")?;
    role.validate()
        .map_err(|error| format!("incompatible compiled replay role: {error}"))?;
    if role == Conditions::DiagnosticProfile {
        return Err("diagnostic profile has no public replay role flag".into());
    }
    Ok(role)
}

fn replay_role(
    left: &Opened,
    right: &Opened,
) -> Result<gomad_wasmhost::linked::Conditions, String> {
    let read = |opened: &Opened| {
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
        configuration_role(&inputs["configuration"], clustered)
    };
    let a = read(left)?;
    if read(right)? != a {
        return Err("retained executions disagree on replay role".into());
    }
    Ok(a)
}

fn replay_flag(role: gomad_wasmhost::linked::Conditions) -> Result<Option<&'static str>, String> {
    use gomad_wasmhost::linked::Conditions;
    match role {
        Conditions::Ordinary => Ok(None),
        Conditions::StressReplay => Ok(Some("--stress-jitter")),
        Conditions::DeterminismFixtureReplay => Ok(Some("--determinism-fixture")),
        Conditions::DiagnosticProfile | Conditions::GuestProfile | Conditions::DiagnosticFuelOff | Conditions::GuestProfileFuelOff => {
            Err("diagnostic profile has no public replay role flag".into())
        }
    }
}
