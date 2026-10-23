//! Supplemental comparison of authenticated native inspection cuts. A cut
//! bounds a difference; it does not supply an unobserved event or clock value.
use crate::evidence::{canonical, store::Opened};
use gomad_wasmhost::linked_inspection::{self, Boundary, Evidence, Handle};
use serde_json::{Value, json};
use std::collections::BTreeMap;

pub(super) fn compare(left: &Opened, right: &Opened) -> Result<Value, String> {
    let read = |opened: &Opened| {
        let file = if opened
            .manifest()
            .files
            .contains_key(crate::replay::cluster::FILE)
        {
            "evidence/inspection_metadata"
        } else {
            "evidence/inspection"
        };
        let raw = opened.payload(file, linked_inspection::MAX_METADATA as u64)?;
        let evidence = Evidence::decode(&raw).map_err(|error| error.to_string())?;
        crate::replay::inspection::authenticate_streams(opened, &evidence)
            .map_err(|error| error.message)?;
        Ok::<_, String>((evidence, raw))
    };
    let (a, _) = read(left)?;
    let (b, raw) = read(right)?;
    let world = |opened: &Opened| {
        if opened
            .manifest()
            .record
            .streams
            .contains_key("evidence/world")
        {
            opened.payload("evidence/world", 384 << 20)
        } else {
            Ok(Vec::new())
        }
    };
    diagnostic(&a, &b, &raw, &world(left)?, &world(right)?)
}

fn value(bytes: &[u8]) -> Value {
    json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)})
}

fn difference(
    handle: &Handle,
    coordinate: (u64, Option<u64>, Option<u64>),
    field: &str,
    a: Value,
    b: Value,
    relation: &str,
) -> Value {
    let (order, time, trace) = coordinate;
    json!({"instance":handle.node,"incarnation":handle.incarnation,"field":field,
        "coordinate":{"Shared":order,"ChoiceOrdinal":trace,"VirtualTime":time},
        "coordinate_availability":{"shared":true,"choice":trace.is_some(),"virtual_time":time.is_some(),"event":false},
        "coordinate_relation":relation,"expected":a,"actual":b})
}

fn prefix(bytes: &[u8], length: u64) -> Result<&[u8], String> {
    let length = usize::try_from(length).map_err(|_| "inspection cut capacity")?;
    bytes
        .get(..length)
        .ok_or_else(|| "inspection cut exceeds retained stream".into())
}

fn changed(a: &[u8], b: &[u8], budget: &mut usize) -> Result<bool, String> {
    let bytes = a
        .len()
        .checked_add(b.len())
        .ok_or("inspection comparison capacity")?;
    *budget = budget
        .checked_sub(bytes)
        .ok_or("inspection comparison capacity")?;
    Ok(a != b)
}

#[expect(
    clippy::too_many_lines,
    reason = "Each bounded cut compares coordinates, retained prefixes and state in causal order"
)]
fn boundary_difference(
    a: &Evidence,
    b: &Evidence,
    boundary: &Boundary,
    owners_a: &BTreeMap<Handle, usize>,
    owners_b: &BTreeMap<Handle, usize>,
    boundaries_b: &BTreeMap<(Handle, u64), usize>,
    budget: &mut usize,
) -> Result<Option<Value>, String> {
    let at = |handle: &Handle, trace, field, x, y| {
        difference(
            handle,
            (boundary.order, Some(boundary.time_nanos), trace),
            field,
            x,
            y,
            "inspection-boundary-upper-bound",
        )
    };
    let Some(index) = boundaries_b.get(&(boundary.handle.clone(), boundary.decision)) else {
        return Ok(Some(at(
            &boundary.handle,
            Some(boundary.trace),
            "boundary-presence",
            json!("present"),
            json!("absent"),
        )));
    };
    let other = &b.boundaries[*index];
    let other_cuts: BTreeMap<_, _> = other.cuts.iter().map(|cut| (&cut.handle, cut)).collect();
    for (field, x, y) in [
        ("boundary-order", boundary.order, other.order),
        ("boundary-trace", boundary.trace, other.trace),
        ("boundary-time", boundary.time_nanos, other.time_nanos),
        ("world-cut", boundary.world_bytes, other.world_bytes),
        (
            "lifecycle-cut",
            boundary.lifecycle_bytes,
            other.lifecycle_bytes,
        ),
    ] {
        if x != y {
            return Ok(Some(at(
                &boundary.handle,
                Some(boundary.trace),
                field,
                json!(x),
                json!(y),
            )));
        }
    }
    for cut in &boundary.cuts {
        let Some(other_cut) = other_cuts.get(&cut.handle).copied() else {
            return Ok(Some(at(
                &cut.handle,
                None,
                "cut-presence",
                json!("present"),
                json!("absent"),
            )));
        };
        let owner = &a.incarnations[*owners_a
            .get(&cut.handle)
            .ok_or("missing inspection owner")?];
        let Some(other_index) = owners_b.get(&cut.handle) else {
            continue;
        };
        let other_owner = &b.incarnations[*other_index];
        // Compare actual retained prefixes before their commitments. Output
        // committed before a later clock/choice is attributed to its first cut.
        for (field, length, other_length, x, y, choice) in [
            (
                "choice",
                cut.choice_bytes,
                other_cut.choice_bytes,
                &owner.streams.choice,
                &other_owner.streams.choice,
                true,
            ),
            (
                "wasi",
                cut.wasi_bytes,
                other_cut.wasi_bytes,
                &owner.streams.wasi,
                &other_owner.streams.wasi,
                false,
            ),
            (
                "network",
                cut.network_bytes,
                other_cut.network_bytes,
                &owner.streams.network,
                &other_owner.streams.network,
                false,
            ),
            (
                "storage",
                cut.storage_bytes,
                other_cut.storage_bytes,
                &owner.streams.storage,
                &other_owner.streams.storage,
                false,
            ),
            (
                "output:stdout",
                cut.stdout_bytes,
                other_cut.stdout_bytes,
                &owner.streams.stdout,
                &other_owner.streams.stdout,
                false,
            ),
            (
                "output:stderr",
                cut.stderr_bytes,
                other_cut.stderr_bytes,
                &owner.streams.stderr,
                &other_owner.streams.stderr,
                false,
            ),
        ] {
            let (x, y) = (prefix(x, length)?, prefix(y, other_length)?);
            if changed(x, y, budget)? {
                let trace = choice.then(|| {
                    x.iter()
                        .zip(y)
                        .position(|(x, y)| x != y)
                        .unwrap_or(x.len().min(y.len())) as u64
                        / 96
                });
                return Ok(Some(at(&cut.handle, trace, field, value(x), value(y))));
            }
        }
        if cut != other_cut {
            let field = if cut.recovery_state != other_cut.recovery_state {
                "recovery-state"
            } else if cut.witness != other_cut.witness {
                "runtime-witness"
            } else {
                "cut-state"
            };
            return Ok(Some(at(
                &cut.handle,
                None,
                field,
                serde_json::to_value(cut).map_err(|e| e.to_string())?,
                serde_json::to_value(other_cut).map_err(|e| e.to_string())?,
            )));
        }
    }
    for (field, x, y) in [
        (
            "world",
            prefix(&a.world, boundary.world_bytes)?,
            prefix(&b.world, other.world_bytes)?,
        ),
        (
            "lifecycle",
            prefix(&a.lifecycle, boundary.lifecycle_bytes)?,
            prefix(&b.lifecycle, other.lifecycle_bytes)?,
        ),
    ] {
        if changed(x, y, budget)? {
            return Ok(Some(at(
                &boundary.handle,
                Some(boundary.trace),
                field,
                value(x),
                value(y),
            )));
        }
    }
    Ok(None)
}

#[expect(
    clippy::too_many_lines,
    reason = "The bounded comparison keeps presence, cut and terminal ordering in one pass"
)]
pub(super) fn diagnostic(
    a: &Evidence,
    b: &Evidence,
    b_raw: &[u8],
    world_a: &[u8],
    world_b: &[u8],
) -> Result<Value, String> {
    if !a.complete || !b.complete {
        return Err("incomplete retained inspection evidence".into());
    }
    let owners_a: BTreeMap<_, _> = a
        .incarnations
        .iter()
        .enumerate()
        .map(|(index, owner)| (owner.handle.clone(), index))
        .collect();
    let owners_b: BTreeMap<_, _> = b
        .incarnations
        .iter()
        .enumerate()
        .map(|(index, owner)| (owner.handle.clone(), index))
        .collect();
    let boundaries_a: BTreeMap<_, _> = a
        .boundaries
        .iter()
        .enumerate()
        .map(|(index, boundary)| ((boundary.handle.clone(), boundary.decision), index))
        .collect();
    let boundaries_b: BTreeMap<_, _> = b
        .boundaries
        .iter()
        .enumerate()
        .map(|(index, boundary)| ((boundary.handle.clone(), boundary.decision), index))
        .collect();
    let mut candidates = Vec::new();
    let mut budget = 64 << 20;
    for (expected, actual_owners, reverse) in [(a, &owners_b, false), (b, &owners_a, true)] {
        for owner in &expected.incarnations {
            if !actual_owners.contains_key(&owner.handle) {
                let (x, y) = if reverse {
                    (json!("absent"), json!("present"))
                } else {
                    (json!("present"), json!("absent"))
                };
                candidates.push(difference(
                    &owner.handle,
                    (owner.started_order, None, None),
                    "incarnation-presence",
                    x,
                    y,
                    "retained-incarnation-start",
                ));
            }
        }
    }
    for boundary in &a.boundaries {
        if let Some(first) = boundary_difference(
            a,
            b,
            boundary,
            &owners_a,
            &owners_b,
            &boundaries_b,
            &mut budget,
        )? {
            candidates.push(first);
        }
    }
    for boundary in &b.boundaries {
        if !boundaries_a.contains_key(&(boundary.handle.clone(), boundary.decision)) {
            candidates.push(difference(
                &boundary.handle,
                (
                    boundary.order,
                    Some(boundary.time_nanos),
                    Some(boundary.trace),
                ),
                "boundary-presence",
                json!("absent"),
                json!("present"),
                "inspection-boundary-upper-bound",
            ));
        }
    }
    for owner in &a.incarnations {
        let Some(other_index) = owners_b.get(&owner.handle) else {
            continue;
        };
        let other = &b.incarnations[*other_index];
        for (field, x, y) in [
            ("choice-tail", &owner.streams.choice, &other.streams.choice),
            ("wasi-tail", &owner.streams.wasi, &other.streams.wasi),
            (
                "network-tail",
                &owner.streams.network,
                &other.streams.network,
            ),
            (
                "storage-tail",
                &owner.streams.storage,
                &other.streams.storage,
            ),
            ("stdout-tail", &owner.streams.stdout, &other.streams.stdout),
            ("stderr-tail", &owner.streams.stderr, &other.streams.stderr),
        ] {
            if changed(x, y, &mut budget)? {
                candidates.push(difference(
                    &owner.handle,
                    (owner.terminal_order.max(other.terminal_order), None, None),
                    field,
                    value(x),
                    value(y),
                    "retained-terminal-upper-bound",
                ));
            }
        }
        if owner.started_order != other.started_order {
            candidates.push(difference(
                &owner.handle,
                (owner.started_order.min(other.started_order), None, None),
                "incarnation-start",
                json!(owner.started_order),
                json!(other.started_order),
                "retained-incarnation-start",
            ));
        }
        if owner.terminal_order != other.terminal_order || owner.registry != other.registry {
            candidates.push(difference(
                &owner.handle,
                (owner.terminal_order.max(other.terminal_order), None, None),
                "incarnation-terminal",
                json!({"terminal":owner.terminal_order,"registry":owner.registry}),
                json!({"terminal":other.terminal_order,"registry":other.registry}),
                "retained-terminal-upper-bound",
            ));
        }
    }
    let world_changed = changed(world_a, world_b, &mut budget)?;
    let lifecycle_changed = changed(&a.lifecycle, &b.lifecycle, &mut budget)?;
    if world_changed || lifecycle_changed {
        let owner = a
            .incarnations
            .iter()
            .chain(&b.incarnations)
            .max_by_key(|owner| owner.terminal_order)
            .ok_or("missing terminal inspection owner")?;
        candidates.push(difference(
            &owner.handle,
            (owner.terminal_order, None, None),
            if world_changed {
                "world-tail"
            } else {
                "lifecycle-tail"
            },
            value(if world_changed { world_a } else { &a.lifecycle }),
            value(if world_changed { world_b } else { &b.lifecycle }),
            "retained-terminal-upper-bound",
        ));
    }
    candidates.sort_by_key(|candidate| candidate["coordinate"]["Shared"].as_u64().unwrap());
    let mut report = json!({"schema":"gomad.inspection-first-difference/v1","diagnostic_only":true,"ordering":"inspection-boundary","status":"identical"});
    if let Some(first) = candidates.into_iter().next() {
        let order = first["coordinate"]["Shared"].as_u64().unwrap();
        report["status"] = json!("changed");
        report["virtual_time_available"] = first["coordinate_availability"]["virtual_time"].clone();
        report["first"] = first;
        if let Some(boundary) = a
            .boundaries
            .iter()
            .rev()
            .find(|boundary| boundary.order < order)
            .filter(|boundary| linked_inspection::verify_prefix(a, b_raw, boundary).is_ok())
        {
            report["preceding_verified_prefix"] = json!({"instance":boundary.handle.node,"incarnation":boundary.handle.incarnation,"decision":boundary.decision,"trace":boundary.trace,"order":boundary.order,"time_nanos":boundary.time_nanos});
        }
    }
    Ok(report)
}
