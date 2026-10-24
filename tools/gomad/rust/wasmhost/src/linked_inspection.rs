use crate::model::inspection::{
    self, BOUNDARY_ENCODING, LIFECYCLE_ENCODING, STATE_ENCODING, WORLD_ENCODING, WORLD_HEADER,
};
pub use crate::model::inspection::{
    Boundary, Command, Cut, Evidence, Handle, MAX_METADATA, Registry, Request, Witness,
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use std::collections::{BTreeMap, BTreeSet};
use wasmtime::{Error, Result};

pub struct Stop {
    pub kind: String,
    pub value: u64,
}

struct Reader<'a>(&'a [u8]);
impl<'a> Reader<'a> {
    fn take(&mut self, count: usize) -> Result<&'a [u8]> {
        if count > self.0.len() {
            return Err(Error::msg("invalid inspection binary length"));
        }
        let (value, rest) = self.0.split_at(count);
        self.0 = rest;
        Ok(value)
    }
    fn u32(&mut self) -> Result<u32> {
        Ok(u32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }
    fn u64(&mut self) -> Result<u64> {
        Ok(u64::from_le_bytes(self.take(8)?.try_into().unwrap()))
    }
    fn handle(&mut self) -> Result<Handle> {
        let count = self.u32()? as usize;
        if count > 1024 {
            return Err(Error::msg("invalid inspection handle"));
        }
        let node = std::str::from_utf8(self.take(count)?)
            .map_err(Error::new)?
            .to_owned();
        if node.is_empty() || node.contains(['\0', '\r', '\n']) {
            return Err(Error::msg("invalid inspection handle"));
        }
        Ok(Handle {
            node,
            incarnation: self.u64()?,
        })
    }
}

impl Evidence {
    /// # Errors
    /// Decodes bounded helper-owned metadata and authenticates binary cut coordinates.
    pub fn decode(raw: &[u8]) -> Result<Self> {
        if raw.is_empty() || raw.len() > MAX_METADATA {
            return Err(Error::msg(
                "unsupported or incomplete WASM inspection evidence",
            ));
        }
        let mut evidence: Self = serde_json::from_slice(raw).map_err(Error::new)?;
        if evidence.schema != "gomad.wasm-inspection/v3"
            || evidence.boundary_encoding != BOUNDARY_ENCODING
            || evidence.lifecycle_encoding != LIFECYCLE_ENCODING
            || evidence.state_encoding != STATE_ENCODING
            || evidence.world_encoding != WORLD_ENCODING
            || evidence.incarnations.is_empty()
            || evidence.incarnations.len() > 65536
            || !evidence.world.starts_with(WORLD_HEADER)
            || !evidence.lifecycle.starts_with(b"GMWIL\x01\0\0")
        {
            return Err(Error::msg(
                "unsupported or incomplete WASM inspection evidence",
            ));
        }
        let mut reader = Reader(&evidence.boundary_frames);
        if reader.take(8)? != b"GMWIB\x01\0\0" {
            return Err(Error::msg("invalid inspection boundary header"));
        }
        let count = reader.u32()?;
        if count > 65536 {
            return Err(Error::msg("inspection boundary capacity"));
        }
        for _ in 0..count {
            let size = reader.u32()? as usize;
            let mut frame = Reader(reader.take(size)?);
            let mut boundary = Boundary {
                handle: frame.handle()?,
                decision: frame.u64()?,
                trace: frame.u64()?,
                order: frame.u64()?,
                time_nanos: frame.u64()?,
                world_bytes: frame.u64()?,
                lifecycle_bytes: frame.u64()?,
                cuts: Vec::new(),
            };
            let cuts = frame.u32()?;
            if cuts == 0 || cuts > 65536 {
                return Err(Error::msg("invalid inspection cut count"));
            }
            for _ in 0..cuts {
                let mut cut = Cut {
                    handle: frame.handle()?,
                    state_encoding: STATE_ENCODING.into(),
                    choice_records: frame.u64()?,
                    choice_bytes: frame.u64()?,
                    wasi_bytes: frame.u64()?,
                    network_bytes: frame.u64()?,
                    storage_bytes: frame.u64()?,
                    stdout_bytes: frame.u64()?,
                    stderr_bytes: frame.u64()?,
                    ..Cut::default()
                };
                let clock = frame.u32()? as usize;
                if clock > 1024 {
                    return Err(Error::msg("invalid inspection clock run"));
                }
                cut.clock_run = frame.take(clock)?.to_vec();
                cut.network_state = frame.take(32)?.to_vec();
                cut.storage_state = frame.take(32)?.to_vec();
                cut.recovery_state = frame.take(32)?.to_vec();
                cut.witness = Witness::decode(frame.take(128)?)?;
                boundary.cuts.push(cut);
            }
            if !frame.0.is_empty() {
                return Err(Error::msg("inspection boundary trailing material"));
            }
            evidence.boundaries.push(boundary);
        }
        if !reader.0.is_empty() {
            return Err(Error::msg("inspection boundaries trailing material"));
        }
        validate(&evidence)?;
        Ok(evidence)
    }
}

fn validate(evidence: &Evidence) -> Result<()> {
    let invalid = || Error::msg("unsupported or incomplete WASM inspection evidence");
    let mut handles = BTreeSet::new();
    for owner in &evidence.incarnations {
        if owner.handle.node.is_empty()
            || owner.handle.node.len() > 1024
            || owner.handle.node.contains(['\0', '\r', '\n'])
            || owner.handle.incarnation == 0
            || !handles.insert(owner.handle.clone())
            || (evidence.complete || owner.terminal_order != 0)
                && owner.terminal_order < owner.started_order
            || owner
                .failure_order
                .is_some_and(|order| order > owner.terminal_order)
            || owner.registry.entries.len() > 256
            || owner.registry.closed
                && (owner.registry.closed_order < owner.started_order
                    || owner.terminal_order != 0
                        && owner.registry.closed_order > owner.terminal_order)
        {
            return Err(invalid());
        }
        validate_registry(owner)?;
    }
    validate_boundaries(evidence, &handles)
}

fn validate_registry(owner: &inspection::Incarnation) -> Result<()> {
    let invalid = || Error::msg("unsupported or incomplete WASM inspection evidence");
    let mut names = BTreeSet::new();
    for (index, entry) in owner.registry.entries.iter().enumerate() {
        if entry.id as usize != index
            || entry.name.is_empty()
            || entry.name.len() > 128
            || entry.name.contains(['\0', '\r', '\n'])
            || !names.insert(&entry.name)
            || matches!(entry.name.as_str(), "stacks" | "heap")
            || !matches!(entry.kind.as_str(), "dump" | "check" | "probe")
            || entry.monotonic && entry.kind != "check"
            || entry.available_order < owner.started_order
            || owner.terminal_order != 0 && entry.available_order > owner.terminal_order
            || owner.registry.closed && entry.available_order > owner.registry.closed_order
            || index > 0
                && entry.available_order < owner.registry.entries[index - 1].available_order
        {
            return Err(invalid());
        }
    }
    Ok(())
}

fn lifecycle_events(
    evidence: &Evidence,
    handles: &BTreeSet<Handle>,
) -> Result<BTreeMap<u64, (u64, u8, Handle)>> {
    lifecycle_records(&evidence.lifecycle, handles)
}

/// Reads bounded lifecycle bytes through the inspection boundary's owner parser.
///
/// # Errors
/// Rejects malformed framing, unsupported tags and undeclared owners.
pub fn lifecycle_records(
    raw: &[u8],
    handles: &BTreeSet<Handle>,
) -> Result<BTreeMap<u64, (u64, u8, Handle)>> {
    let invalid = || Error::msg("unsupported or incomplete WASM inspection evidence");
    if raw.len() > MAX_METADATA || !raw.starts_with(b"GMWIL\x01\0\0") {
        return Err(invalid());
    }
    let mut lifecycle = Reader(&raw[8..]);
    let mut events = BTreeMap::new();
    let mut order = 0u64;
    while !lifecycle.0.is_empty() {
        let action = lifecycle.take(1)?[0];
        let handle = lifecycle.handle()?;
        order += 1;
        if !(1..=10).contains(&action)
            || action != 7 && !handles.contains(&handle)
            || action == 7 && (handle.node != "controller" || handle.incarnation != 0)
        {
            return Err(invalid());
        }
        events.insert(
            (raw.len() - lifecycle.0.len()) as u64,
            (order, action, handle),
        );
    }
    Ok(events)
}

fn validate_boundaries(evidence: &Evidence, handles: &BTreeSet<Handle>) -> Result<()> {
    let invalid = || Error::msg("unsupported or incomplete WASM inspection evidence");
    let events = lifecycle_events(evidence, handles)?;
    let mut decisions = BTreeMap::new();
    let mut prior_order = 0;
    let mut prior_time = 0;
    let mut world_bytes = WORLD_HEADER.len();
    let mut previous: BTreeMap<Handle, Cut> = BTreeMap::new();
    for boundary in &evidence.boundaries {
        let owner = evidence
            .incarnations
            .iter()
            .find(|owner| owner.handle == boundary.handle)
            .ok_or_else(invalid)?;
        let decision = decisions.entry(boundary.handle.clone()).or_insert(0);
        if boundary.decision != *decision
            || boundary.order < owner.started_order
            || (evidence.complete || owner.terminal_order != 0)
                && boundary.order > owner.terminal_order
            || boundary.order <= prior_order
            || boundary.time_nanos < prior_time
            || events.get(&boundary.lifecycle_bytes)
                != Some(&(boundary.order, 2, boundary.handle.clone()))
            || world_bytes + 40 > evidence.world.len()
            || evidence.world[world_bytes..world_bytes + 8] != 32u64.to_le_bytes()
        {
            return Err(invalid());
        }
        world_bytes += 40;
        if boundary.world_bytes != world_bytes as u64 {
            return Err(invalid());
        }
        *decision += 1;
        prior_time = boundary.time_nanos;
        let mut cuts = BTreeSet::new();
        for cut in &boundary.cuts {
            let owner = evidence
                .incarnations
                .iter()
                .find(|owner| owner.handle == cut.handle)
                .ok_or_else(invalid)?;
            if !cuts.insert(cut.handle.clone())
                || owner.started_order > boundary.order
                || cut.handle == boundary.handle && cut.choice_records != boundary.trace
            {
                return Err(invalid());
            }
            for (length, bytes) in streams(cut, &owner.streams) {
                if length > bytes.len() as u64
                    || owner.terminal_order != 0
                        && owner.terminal_order < boundary.order
                        && length != bytes.len() as u64
                {
                    return Err(invalid());
                }
            }
            if owner.terminal_order != 0
                && owner.terminal_order < boundary.order
                && cut.network_bytes != 0
                && cut.network_state != inspection::commitment("network", &owner.streams.network)
            {
                return Err(Error::msg("inspection terminal network state commitment"));
            }
            if let Some(prior) = previous.insert(cut.handle.clone(), cut.clone())
                && (cut.choice_records < prior.choice_records
                    || cut.choice_bytes < prior.choice_bytes
                    || cut.wasi_bytes < prior.wasi_bytes
                    || cut.network_bytes < prior.network_bytes
                    || cut.storage_bytes < prior.storage_bytes
                    || cut.stdout_bytes < prior.stdout_bytes
                    || cut.stderr_bytes < prior.stderr_bytes
                    || owner.terminal_order != 0
                        && owner.terminal_order < prior_order
                        && *cut != prior)
            {
                return Err(invalid());
            }
        }
        if evidence
            .incarnations
            .iter()
            .any(|owner| owner.started_order <= boundary.order && !cuts.contains(&owner.handle))
        {
            return Err(invalid());
        }
        prior_order = boundary.order;
    }
    if world_bytes != evidence.world.len() {
        return Err(invalid());
    }
    Ok(())
}

fn streams<'a>(cut: &Cut, streams: &'a inspection::Streams) -> [(u64, &'a [u8]); 6] {
    [
        (cut.choice_bytes, &streams.choice),
        (cut.wasi_bytes, &streams.wasi),
        (cut.network_bytes, &streams.network),
        (cut.storage_bytes, &streams.storage),
        (cut.stdout_bytes, &streams.stdout),
        (cut.stderr_bytes, &streams.stderr),
    ]
}

/// # Errors
/// Refuses missing decisions, observation coordinates and cut lengths that disagree with the retained raw choice stream.
pub fn validate_choice_mapping(evidence: &Evidence) -> Result<()> {
    if !evidence.complete {
        return Ok(());
    }
    for owner in &evidence.incarnations {
        let mapped = crate::model::replay_plan::inspection_coordinates(&owner.streams.choice)
            .map_err(Error::msg)?;
        let mut decisions = 0usize;
        for boundary in &evidence.boundaries {
            if boundary.handle == owner.handle {
                if mapped.get(decisions) != Some(&boundary.trace)
                    || boundary.decision != decisions as u64
                {
                    return Err(Error::msg("inspection decision/trace mapping"));
                }
                decisions += 1;
            }
            for cut in boundary
                .cuts
                .iter()
                .filter(|cut| cut.handle == owner.handle)
            {
                if cut
                    .choice_records
                    .checked_mul(crate::model::replay_plan::RECORD_BYTES as u64)
                    != Some(cut.choice_bytes)
                    || cut.choice_bytes > owner.streams.choice.len() as u64
                {
                    return Err(Error::msg("inspection choice prefix length"));
                }
            }
        }
        if decisions != mapped.len() {
            return Err(Error::msg(
                "incomplete inspection decision boundary inventory",
            ));
        }
    }
    Ok(())
}

/// # Errors
/// Resolves retained predecision coordinates and constructs the existing typed helper control.
pub fn resolve(
    evidence: &Evidence,
    mut target: Handle,
    stop: &Stop,
    dumps: &[String],
    probes: &[String],
    check: &str,
) -> Result<Request> {
    if !evidence.complete {
        return Err(Error::msg(
            "unsupported or incomplete WASM inspection evidence",
        ));
    }
    if target.node.is_empty() && target.incarnation == 0 && evidence.incarnations.len() == 1 {
        target = evidence.incarnations[0].handle.clone();
    }
    if target.node.is_empty() || target.incarnation == 0 {
        return Err(Error::msg(
            "inspection requires an explicit instance and incarnation",
        ));
    }
    let owner = evidence
        .incarnations
        .iter()
        .find(|owner| owner.handle == target)
        .ok_or_else(|| Error::msg("unknown inspection instance or incarnation"))?;
    let candidates = evidence
        .boundaries
        .iter()
        .filter(|boundary| boundary.handle == target);
    let boundary = match stop.kind.as_str() {
        "choice" => candidates
            .into_iter()
            .find(|boundary| boundary.decision == stop.value),
        "time" => candidates
            .into_iter()
            .find(|boundary| boundary.time_nanos >= stop.value),
        "failure" if stop.value == 0 => candidates.into_iter().rfind(|boundary| {
            owner
                .failure_order
                .is_some_and(|order| boundary.order <= order)
        }),
        _ => return Err(Error::msg("invalid inspection stop selector")),
    }
    .ok_or_else(|| Error::msg("inspection stop is outside retained inspectable boundaries"))?;
    let commands = select_commands(&owner.registry, boundary.order, dumps, check)?;
    let enabled = select_probes(&owner.registry, boundary.order, probes)?;
    resolve_prefix(evidence, target, boundary, commands, enabled)
}

fn select_commands(
    registry: &Registry,
    order: u64,
    dumps: &[String],
    check: &str,
) -> Result<Vec<Command>> {
    let mut commands = Vec::new();
    for name in dumps {
        let command = match name.as_str() {
            "stacks" => Command { kind: 1, id: 0 },
            "heap" => Command { kind: 2, id: 0 },
            _ => {
                if !registry.closed {
                    return Err(Error::msg(
                        "unsupported or incomplete WASM inspection evidence: registry is not closed",
                    ));
                }
                let entry = registry
                    .entries
                    .iter()
                    .find(|entry| {
                        entry.kind == "dump"
                            && entry.name == *name
                            && entry.available_order <= order
                    })
                    .ok_or_else(|| {
                        Error::msg(format!("unknown or unavailable inspection dump {name:?}"))
                    })?;
                Command {
                    kind: 3,
                    id: entry.id,
                }
            }
        };
        commands.push(command);
    }
    if !check.is_empty() {
        let entry = registry
            .entries
            .iter()
            .find(|entry| {
                entry.kind == "check" && entry.name == check && entry.available_order <= order
            })
            .ok_or_else(|| {
                Error::msg(format!("unknown or unavailable inspection check {check:?}"))
            })?;
        if !registry.closed {
            return Err(Error::msg("inspection invariant registry is not closed"));
        }
        commands.push(Command {
            kind: 4,
            id: entry.id,
        });
    }
    Ok(commands)
}

fn select_probes(registry: &Registry, order: u64, probes: &[String]) -> Result<Vec<u32>> {
    let mut enabled = BTreeSet::new();
    for pattern in probes {
        if !registry.closed {
            return Err(Error::msg(
                "unsupported or incomplete WASM inspection evidence: registry is not closed",
            ));
        }
        let mut found = false;
        for entry in &registry.entries {
            if glob(pattern, &entry.name)?
                && entry.kind == "probe"
                && entry.available_order <= order
            {
                enabled.insert(entry.id);
                found = true;
            }
        }
        if !found {
            return Err(Error::msg(format!(
                "unknown or unavailable inspection probe pattern {pattern:?}"
            )));
        }
    }
    Ok(enabled.into_iter().collect())
}

fn prefix(bytes: &[u8], count: u64) -> Result<&[u8]> {
    usize::try_from(count)
        .ok()
        .and_then(|count| bytes.get(..count))
        .ok_or_else(|| Error::msg("inspection prefix exceeds retained stream"))
}

fn resolve_prefix(
    evidence: &Evidence,
    target: Handle,
    boundary: &Boundary,
    commands: Vec<Command>,
    probes: Vec<u32>,
) -> Result<Request> {
    let mut stream_sha256 = Vec::new();
    let mut registries = Vec::new();
    for cut in &boundary.cuts {
        let owner = evidence
            .incarnations
            .iter()
            .find(|owner| owner.handle == cut.handle)
            .ok_or_else(|| Error::msg("inspection cut without owner"))?;
        let names = ["choice", "wasi", "network", "storage", "stdout", "stderr"];
        stream_sha256.push(
            names
                .into_iter()
                .zip(streams(cut, &owner.streams))
                .map(|(name, (count, bytes))| {
                    Ok((name.into(), inspection::digest(prefix(bytes, count)?)))
                })
                .collect::<Result<_>>()?,
        );
        let closed = owner.registry.closed && owner.registry.closed_order <= boundary.order;
        registries.push(Registry {
            closed,
            closed_order: if closed {
                owner.registry.closed_order
            } else {
                0
            },
            entries: owner
                .registry
                .entries
                .iter()
                .filter(|entry| entry.available_order <= boundary.order)
                .cloned()
                .collect(),
        });
    }
    Ok(Request {
        target,
        decision: boundary.decision,
        expected: boundary.clone(),
        world_sha256: inspection::digest(prefix(&evidence.world, boundary.world_bytes)?),
        lifecycle_sha256: inspection::digest(prefix(
            &evidence.lifecycle,
            boundary.lifecycle_bytes,
        )?),
        stream_sha256,
        probes,
        commands,
        registries,
    })
}

/// # Errors
/// Verifies the returned prefix against retained owner bytes and witnesses.
pub fn verify_prefix(retained: &Evidence, observed_raw: &[u8], expected: &Boundary) -> Result<()> {
    let observed = Evidence::decode(observed_raw)
        .map_err(|error| Error::msg(format!("instrumentation divergence: {error}")))?;
    let divergent = || Error::msg("instrumentation divergence: retained inspection prefix");
    let actual = observed
        .boundaries
        .iter()
        .find(|boundary| {
            boundary.handle == expected.handle && boundary.decision == expected.decision
        })
        .ok_or_else(divergent)?;
    if actual.trace != expected.trace
        || actual.order != expected.order
        || actual.time_nanos != expected.time_nanos
        || actual.world_bytes != expected.world_bytes
        || actual.lifecycle_bytes != expected.lifecycle_bytes
        || actual.cuts != expected.cuts
        || prefix(&retained.world, expected.world_bytes)?
            != prefix(&observed.world, actual.world_bytes)?
        || prefix(&retained.lifecycle, expected.lifecycle_bytes)?
            != prefix(&observed.lifecycle, actual.lifecycle_bytes)?
    {
        return Err(divergent());
    }
    for cut in &expected.cuts {
        let a = retained
            .incarnations
            .iter()
            .find(|owner| owner.handle == cut.handle)
            .ok_or_else(divergent)?;
        let b = observed
            .incarnations
            .iter()
            .find(|owner| owner.handle == cut.handle)
            .ok_or_else(divergent)?;
        if a.started_order != b.started_order
            || cut.network_bytes != 0 && a.streams.network_encoding != b.streams.network_encoding
        {
            return Err(divergent());
        }
        for ((length, a), (_, b)) in streams(cut, &a.streams)
            .into_iter()
            .zip(streams(cut, &b.streams))
        {
            if prefix(a, length)? != prefix(b, length)? {
                return Err(divergent());
            }
        }
        let entries = |registry: &Registry| {
            registry
                .entries
                .iter()
                .filter(|entry| entry.available_order <= expected.order)
                .cloned()
                .collect::<Vec<_>>()
        };
        let a_closed = a.registry.closed && a.registry.closed_order <= expected.order;
        let b_closed = b.registry.closed && b.registry.closed_order <= expected.order;
        if entries(&a.registry) != entries(&b.registry)
            || a_closed != b_closed
            || a_closed && a.registry.closed_order != b.registry.closed_order
        {
            return Err(divergent());
        }
    }
    Ok(())
}

fn glob(pattern: &str, name: &str) -> Result<bool> {
    enum Part {
        Literal(char),
        Any,
        Star,
        Class(bool, Vec<(char, char)>),
    }
    fn escaped(input: &[char], index: &mut usize, class: bool) -> Result<char> {
        let invalid = || Error::msg("invalid inspection probe pattern");
        let mut character = *input.get(*index).ok_or_else(invalid)?;
        *index += 1;
        if character == '\\' {
            character = *input.get(*index).ok_or_else(invalid)?;
            *index += 1;
        } else if class && matches!(character, '-' | ']') {
            return Err(invalid());
        }
        Ok(character)
    }
    if pattern.len() > 128 || name.len() > 128 {
        return Err(Error::msg("invalid inspection probe pattern"));
    }
    let input: Vec<_> = pattern.chars().collect();
    let name: Vec<_> = name.chars().collect();
    let mut parts = Vec::new();
    let mut index = 0;
    while index < input.len() {
        let character = input[index];
        index += 1;
        parts.push(match character {
            '*' => Part::Star,
            '?' => Part::Any,
            '\\' => {
                let value = *input
                    .get(index)
                    .ok_or_else(|| Error::msg("invalid inspection probe pattern"))?;
                index += 1;
                Part::Literal(value)
            }
            '[' => {
                let negative = input.get(index) == Some(&'^');
                if negative {
                    index += 1;
                }
                let mut ranges = Vec::new();
                loop {
                    if input.get(index) == Some(&']') && !ranges.is_empty() {
                        index += 1;
                        break;
                    }
                    let lower = escaped(&input, &mut index, true)?;
                    let upper = if input.get(index) == Some(&'-') {
                        index += 1;
                        escaped(&input, &mut index, true)?
                    } else {
                        lower
                    };
                    ranges.push((lower, upper));
                }
                Part::Class(negative, ranges)
            }
            _ => Part::Literal(character),
        });
    }
    let mut prior = vec![false; name.len() + 1];
    prior[0] = true;
    for part in parts {
        let mut current = vec![false; name.len() + 1];
        if matches!(part, Part::Star) {
            current[0] = prior[0];
        }
        for index in 0..name.len() {
            let character = name[index];
            current[index + 1] = match &part {
                Part::Literal(value) => prior[index] && character == *value,
                Part::Any => prior[index] && character != '/',
                Part::Star => prior[index + 1] || current[index] && character != '/',
                Part::Class(negative, ranges) => {
                    prior[index]
                        && character != '/'
                        && ranges
                            .iter()
                            .any(|(lower, upper)| *lower <= character && character <= *upper)
                            != *negative
                }
            };
        }
        prior = current;
    }
    Ok(prior[name.len()])
}

pub struct Diagnostics {
    pub output: Value,
    pub heap: Vec<u8>,
}

impl Diagnostics {
    /// # Errors
    /// Truncates diagnostics in the shipped priority order within one aggregate byte bound.
    pub fn bound(mut self, maximum: u64) -> Result<(Vec<u8>, Vec<u8>, bool)> {
        let value = &mut self.output;
        let invalid = || Error::msg("invalid debug output");
        let mut truncated = value["stacks_truncated"] == true
            || value["heap_truncated"] == true
            || value["drops"]
                .as_array()
                .is_some_and(|drops| !drops.is_empty())
            || value["dumps"]
                .as_array()
                .is_some_and(|dumps| dumps.iter().any(|dump| dump["status"] == "truncated"));
        loop {
            value["truncated"] = truncated.into();
            let encoded = serde_json::to_vec(&value).map_err(Error::new)?;
            if encoded.len() as u64 + self.heap.len() as u64 <= maximum {
                return Ok((encoded, self.heap, truncated));
            }
            truncated = true;
            if !self.heap.is_empty() {
                self.heap.clear();
                value["heap_truncated"] = true.into();
                continue;
            }
            let mut removed = false;
            for name in ["stacks", "probes", "dumps", "drops"] {
                if let Some(values) = value[name].as_array_mut()
                    && !values.is_empty()
                {
                    if name == "dumps" {
                        let dump = values.last_mut().ok_or_else(invalid)?;
                        let bytes = STANDARD
                            .decode(dump["data_base64"].as_str().unwrap_or(""))
                            .map_err(Error::new)?;
                        if bytes.is_empty() {
                            values.pop();
                        } else {
                            dump["data_base64"] = STANDARD.encode(&bytes[..bytes.len() / 2]).into();
                            dump["status"] = "truncated".into();
                        }
                    } else {
                        values.pop();
                    }
                    if values.is_empty() {
                        value.as_object_mut().ok_or_else(invalid)?.remove(name);
                    }
                    if name == "stacks" {
                        value["stacks_truncated"] = true.into();
                    }
                    removed = true;
                    break;
                }
            }
            if !removed {
                return Err(Error::msg(
                    "inspection byte bound is too small for truncation metadata",
                ));
            }
        }
    }
}

#[derive(Default)]
struct HeapSample {
    numbers: [u64; 5],
    pcs: Vec<u64>,
    frames: Vec<Value>,
}

fn coordinates(target: &Handle, decision: u64, trace: u64, time: u64) -> Value {
    json!({"instance":target.node,"incarnation":target.incarnation,"decision":decision,"trace":trace,"virtual_time_nanos":time})
}

fn named(mut value: Value, id: u32, kind: &str, registry: &Registry) -> Value {
    value["id"] = id.into();
    if let Some(entry) = registry
        .entries
        .iter()
        .find(|entry| entry.id == id && entry.kind == kind)
    {
        value["name"] = entry.name.clone().into();
    }
    value
}

/// # Errors
/// Decodes bounded diagnostic records produced by the existing helper-owned debug exchange.
#[expect(
    clippy::too_many_lines,
    reason = "The bounded debug operation switch mirrors the existing runtime ABI"
)]
pub fn debug_output(
    raw: &[u8],
    target: &Handle,
    registry: &Registry,
    per_dump: u64,
) -> Result<Diagnostics> {
    let invalid = || Error::msg("invalid inspection debug output");
    if raw.len() > 16 << 20 {
        return Err(invalid());
    }
    let mut reader = Reader(raw);
    let mut stacks: Vec<Value> = Vec::new();
    let mut stack_positions = BTreeMap::new();
    let mut dumps = Vec::new();
    let mut probes = Vec::new();
    let mut checks = Vec::new();
    let mut drops: Vec<Value> = Vec::new();
    let mut samples: Vec<HeapSample> = Vec::new();
    let mut heap_coordinates = coordinates(target, 0, 0, 0);
    let (mut stack_seen, mut stack_done, mut heap_seen, mut heap_done) =
        (false, false, false, false);
    let (mut stacks_truncated, mut heap_truncated) = (false, false);
    let mut rate = None;
    while !reader.0.is_empty() {
        let operation = reader.u32()?;
        let length = reader.u32()? as usize;
        let at = coordinates(target, reader.u64()?, reader.u64()?, reader.u64()?);
        if length > 64 * 1024 + 16 {
            return Err(invalid());
        }
        let mut payload = Reader(reader.take(length)?);
        match operation {
            4 => {
                let kind = payload.u32()?;
                let id = payload.u32()?;
                let status = payload.u32()?;
                let size = payload.u32()? as usize;
                if id >= 256 || size != payload.0.len() {
                    return Err(invalid());
                }
                match kind {
                    1 => {
                        if size > 16 * 1024 - 16 || status > 1024 {
                            return Err(invalid());
                        }
                        let mut fields = Vec::new();
                        for _ in 0..status {
                            let tag = payload.take(1)?[0];
                            let size = payload.u32()? as usize;
                            let bytes = payload.take(size)?;
                            let mut field = json!({});
                            match tag {
                                1 | 2 | 4 if size == 8 => {
                                    let bits = u64::from_le_bytes(
                                        bytes.try_into().map_err(|_| invalid())?,
                                    );
                                    let (name, key) = match tag {
                                        1 => ("int64", "int64"),
                                        2 => ("uint64", "uint64"),
                                        _ => ("float64", "float64_bits"),
                                    };
                                    field["type"] = name.into();
                                    if bits != 0 {
                                        field[key] = if tag == 1 {
                                            bits.cast_signed().into()
                                        } else {
                                            bits.into()
                                        };
                                    }
                                }
                                3 if size == 1 && bytes[0] <= 1 => {
                                    field["type"] = "bool".into();
                                    if bytes[0] != 0 {
                                        field["bool"] = true.into();
                                    }
                                }
                                5 => {
                                    field["type"] = "string".into();
                                    if !bytes.is_empty() {
                                        field["bytes_base64"] = STANDARD.encode(bytes).into();
                                    }
                                }
                                _ => return Err(invalid()),
                            }
                            fields.push(field);
                        }
                        if !payload.0.is_empty() {
                            return Err(invalid());
                        }
                        let mut probe = named(at, id, "probe", registry);
                        probe["fields"] = if fields.is_empty() {
                            Value::Null
                        } else {
                            fields.into()
                        };
                        probes.push(probe);
                    }
                    2 if status <= 2 && size <= 65536 => {
                        let mut dump = named(at, id, "dump", registry);
                        let retained = size.min(usize::try_from(per_dump).unwrap_or(usize::MAX));
                        dump["status"] = if retained < size {
                            "truncated"
                        } else {
                            ["ok", "panic", "truncated"][status as usize]
                        }
                        .into();
                        dump["data_base64"] = STANDARD.encode(&payload.0[..retained]).into();
                        dumps.push(dump);
                    }
                    3 if status <= 2 && size == 0 => {
                        let mut check = named(at, id, "check", registry);
                        check["status"] = ["true", "false", "panic"][status as usize].into();
                        checks.push(check);
                    }
                    4 if id == 0 && status == 0 && size == 8 => {
                        let count = payload.u64()?;
                        if drops
                            .last()
                            .is_some_and(|drop| drop["count"].as_u64().unwrap_or(0) > count)
                        {
                            return Err(invalid());
                        }
                        let mut drop = at;
                        drop["count"] = count.into();
                        drops.push(drop);
                    }
                    _ => return Err(invalid()),
                }
            }
            8 | 11 => {
                let identity = payload.u64()?;
                let status = payload.u32()?;
                let index = payload.u32()?;
                let pc = payload.u64()?;
                let line = payload.u32()?;
                let function_size = payload.u32()? as usize;
                let file_size = payload.u32()? as usize;
                if function_size > 2048
                    || file_size > 2048
                    || payload.0.len() != function_size + file_size
                {
                    return Err(invalid());
                }
                let function = String::from_utf8_lossy(payload.take(function_size)?).into_owned();
                let file = String::from_utf8_lossy(payload.take(file_size)?).into_owned();
                let frame = json!({"pc":pc,"line":line,"function":function,"file":file});
                let truncated = status >> 31 != 0;
                if index == u32::MAX
                    && (identity != 0
                        || status & !(1 << 31) != 0
                        || pc != 0
                        || line != 0
                        || function_size != 0
                        || file_size != 0)
                {
                    return Err(invalid());
                }
                if operation == 8 {
                    stack_seen = true;
                    if stack_done {
                        return Err(invalid());
                    }
                    if index == u32::MAX {
                        stack_done = true;
                        stacks_truncated = truncated;
                        continue;
                    }
                    if identity == 0 {
                        return Err(invalid());
                    }
                    let position = if let Some(position) = stack_positions.get(&identity) {
                        *position
                    } else {
                        if stacks.len() >= 4096 {
                            return Err(invalid());
                        }
                        let position = stacks.len();
                        let mut stack = at.clone();
                        stack["goroutine"] = identity.into();
                        stack["status"] = (status & !(1 << 31)).into();
                        stack["frames"] = json!([]);
                        stacks.push(stack);
                        stack_positions.insert(identity, position);
                        position
                    };
                    let stack = &mut stacks[position];
                    if stack["frames"].as_array().map_or(usize::MAX, Vec::len) != index as usize
                        || index >= 1024
                        || stack["status"] != status & !(1 << 31)
                        || ["decision", "trace", "virtual_time_nanos"]
                            .iter()
                            .any(|key| stack[*key] != at[*key])
                    {
                        return Err(invalid());
                    }
                    stack["frames"]
                        .as_array_mut()
                        .ok_or_else(invalid)?
                        .push(frame);
                    if truncated {
                        stack["truncated"] = true.into();
                    }
                } else {
                    heap_seen = true;
                    heap_coordinates = at;
                    if heap_done {
                        return Err(invalid());
                    }
                    if index == u32::MAX {
                        heap_done = true;
                        heap_truncated |= truncated;
                        continue;
                    }
                    let sample = samples
                        .get_mut(usize::try_from(identity).map_err(|_| invalid())?)
                        .ok_or_else(invalid)?;
                    if index as usize != sample.frames.len() || index >= 1024 {
                        return Err(invalid());
                    }
                    sample.frames.push(frame);
                    heap_truncated |= truncated;
                }
            }
            9 => {
                heap_seen = true;
                heap_coordinates = at;
                if heap_done || samples.len() >= 4096 {
                    return Err(invalid());
                }
                let mut numbers = [0; 5];
                for number in &mut numbers {
                    *number = payload.u64()?;
                    if *number > i64::MAX as u64 {
                        return Err(invalid());
                    }
                }
                let id = payload.u32()?;
                let count = payload.u32()?;
                if rate.is_some_and(|rate| rate != numbers[0]) {
                    return Err(invalid());
                }
                rate = Some(numbers[0]);
                if id == u32::MAX {
                    if !payload.0.is_empty()
                        || count != 0
                        || numbers[1..].iter().any(|n| *n != 0)
                        || !samples.is_empty()
                    {
                        return Err(invalid());
                    }
                    continue;
                }
                if id as usize != samples.len()
                    || count > 32
                    || payload.0.len() != 8 * count as usize
                    || numbers[2] > numbers[1]
                    || numbers[4] > numbers[3]
                    || numbers[1] == 0 && numbers[3] != 0
                {
                    return Err(invalid());
                }
                let mut sample = HeapSample {
                    numbers,
                    ..HeapSample::default()
                };
                for _ in 0..count {
                    sample.pcs.push(payload.u64()?);
                }
                samples.push(sample);
            }
            _ => return Err(invalid()),
        }
    }
    stacks_truncated |= stack_seen && !stack_done;
    heap_truncated |= heap_seen && !heap_done;
    let mut output = json!({"instance":target.node,"incarnation":target.incarnation,
        "heap_coordinates":heap_coordinates,"heap_sample_rate":rate.unwrap_or(0)});
    for (name, values) in [
        ("stacks", stacks),
        ("dumps", dumps),
        ("probes", probes),
        ("checks", checks),
        ("drops", drops),
    ] {
        if !values.is_empty() {
            output[name] = values.into();
        }
    }
    if stacks_truncated {
        output["stacks_truncated"] = true.into();
    }
    if heap_truncated {
        output["heap_truncated"] = true.into();
    }
    Ok(Diagnostics {
        output,
        heap: if heap_seen {
            heap_profile(&samples, rate.unwrap_or(0))?
        } else {
            Vec::new()
        },
    })
}

fn varint(bytes: &mut Vec<u8>, mut value: u64) {
    while value >= 128 {
        bytes.push((value.to_le_bytes()[0] & 127) | 128);
        value >>= 7;
    }
    bytes.push(value.to_le_bytes()[0]);
}
fn proto_uint(bytes: &mut Vec<u8>, field: u64, value: u64) {
    varint(bytes, field << 3);
    varint(bytes, value);
}
fn proto_bytes(bytes: &mut Vec<u8>, field: u64, value: &[u8]) {
    varint(bytes, field << 3 | 2);
    varint(bytes, value.len() as u64);
    bytes.extend_from_slice(value);
}
fn value_type(kind: u64, unit: u64) -> Vec<u8> {
    let mut value = Vec::new();
    proto_uint(&mut value, 1, kind);
    proto_uint(&mut value, 2, unit);
    value
}
#[expect(
    clippy::cast_precision_loss,
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    reason = "Matches Go pprof floating-point sampling correction; positive results are bounded below the signed sample limit before truncation"
)]
fn scale(objects: u64, bytes: u64, rate: u64) -> Result<(u64, u64)> {
    if objects == 0 || bytes == 0 {
        return Ok((0, 0));
    }
    if rate <= 1 {
        return Ok((objects, bytes));
    }
    let factor = 1.0 / -(-(bytes as f64) / objects as f64 / rate as f64).exp_m1();
    let count = objects as f64 * factor;
    let size = bytes as f64 * factor;
    if count >= i64::MAX as f64 || size >= i64::MAX as f64 {
        return Err(Error::msg("debug scaled heap sample overflow"));
    }
    Ok((count as u64, size as u64))
}
fn heap_profile(samples: &[HeapSample], rate: u64) -> Result<Vec<u8>> {
    let mut strings = vec![
        String::new(),
        "alloc_objects".into(),
        "count".into(),
        "alloc_space".into(),
        "bytes".into(),
        "inuse_objects".into(),
        "inuse_space".into(),
        "space".into(),
    ];
    let mut profile = Vec::new();
    for kind in [1, 3, 5, 6] {
        proto_bytes(
            &mut profile,
            1,
            &value_type(kind, if kind == 3 || kind == 6 { 4 } else { 2 }),
        );
    }
    proto_bytes(&mut profile, 11, &value_type(7, 4));
    proto_uint(&mut profile, 14, 6);
    proto_uint(&mut profile, 12, rate);
    let mut next = 0;
    for sample in samples {
        let fallback: Vec<_> = sample
            .pcs
            .iter()
            .map(|pc| json!({"pc":pc,"line":0,"function":"","file":""}))
            .collect();
        let frames = if sample.frames.is_empty() {
            &fallback
        } else {
            &sample.frames
        };
        let mut locations = Vec::new();
        for frame in frames {
            next += 1;
            let mut function = Vec::new();
            proto_uint(&mut function, 1, next);
            for (field, name) in [(2, "function"), (3, "function"), (4, "file")] {
                let value = frame[name].as_str().unwrap_or("");
                let index = if let Some(index) = strings.iter().position(|string| string == value) {
                    index
                } else {
                    strings.push(value.to_owned());
                    strings.len() - 1
                };
                proto_uint(&mut function, field, index as u64);
            }
            proto_bytes(&mut profile, 5, &function);
            let mut line = Vec::new();
            proto_uint(&mut line, 1, next);
            proto_uint(&mut line, 2, frame["line"].as_u64().unwrap_or(0));
            let mut location = Vec::new();
            proto_uint(&mut location, 1, next);
            proto_uint(&mut location, 3, frame["pc"].as_u64().unwrap_or(0));
            proto_bytes(&mut location, 4, &line);
            proto_bytes(&mut profile, 4, &location);
            varint(&mut locations, next);
        }
        let n = sample.numbers;
        let allocated = scale(n[1], n[3], n[0])?;
        let inuse = scale(n[1] - n[2], n[3] - n[4], n[0])?;
        let mut values = Vec::new();
        for value in [allocated.0, allocated.1, inuse.0, inuse.1] {
            varint(&mut values, value);
        }
        let mut value = Vec::new();
        proto_bytes(&mut value, 1, &locations);
        proto_bytes(&mut value, 2, &values);
        proto_bytes(&mut profile, 2, &value);
    }
    for string in strings {
        proto_bytes(&mut profile, 6, string.as_bytes());
    }
    let mut gzip = vec![31, 139, 8, 0, 0, 0, 0, 0, 0, 255];
    let mut chunks = profile.chunks(65535).peekable();
    while let Some(chunk) = chunks.next() {
        gzip.push(u8::from(chunks.peek().is_none()));
        let length =
            u16::try_from(chunk.len()).map_err(|_| Error::msg("heap gzip block overflow"))?;
        gzip.extend_from_slice(&length.to_le_bytes());
        gzip.extend_from_slice(&(!length).to_le_bytes());
        gzip.extend_from_slice(chunk);
    }
    gzip.extend_from_slice(&crc32fast::hash(&profile).to_le_bytes());
    let length = u32::try_from(profile.len()).map_err(|_| Error::msg("heap gzip size overflow"))?;
    gzip.extend_from_slice(&length.to_le_bytes());
    Ok(gzip)
}

/// Checks the complete retained gzip heap payload without executing a guest.
///
/// # Errors
/// Refuses corrupt, empty, truncated, trailing or over-capacity payloads.
pub fn validate_heap_profile(bytes: &[u8], maximum_expanded: usize) -> Result<()> {
    let (raw, consumed) =
        crate::world::gzip::decode(bytes, maximum_expanded).map_err(Error::msg)?;
    if raw.is_empty() || consumed != bytes.len() {
        return Err(Error::msg(
            "heap gzip payload is empty or has trailing bytes",
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "linked_inspection_predicate_tests.rs"]
mod predicate_tests;
