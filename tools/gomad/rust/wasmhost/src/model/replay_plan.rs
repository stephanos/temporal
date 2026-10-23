use super::{CreationSite, Decision, get32, get64, set32, set64};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use super::choice_wire_generated::{
    TAPE_CHECKSUM_OFFSET, TAPE_HEADER_BYTES as HEADER_BYTES, TAPE_HEADER_WIRE_BYTES, TAPE_MAGIC,
    TAPE_RECORD_BYTES,
};

pub const RECORD_BYTES: usize = super::choice_wire_generated::RECORD_BYTES;

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
pub struct ExecutionIdentity {
    #[serde(rename = "TargetSHA256")]
    pub target_sha256: [u8; 32],
    pub toolchain_build_key: String,
    #[serde(rename = "GOOS")]
    pub goos: String,
    #[serde(rename = "GOARCH")]
    pub goarch: String,
    #[serde(rename = "ImplementationSHA256")]
    pub implementation_sha256: [u8; 32],
}

impl ExecutionIdentity {
    #[must_use]
    pub fn runtime_implementation_sha256(&self) -> String {
        crate::world::digest::hex(&self.implementation_sha256)
    }
}

pub struct ReplayPlan {
    pub decisions: Vec<Decision>,
    pub bytes: Vec<u8>,
    pub sha256: [u8; 32],
}

#[derive(Serialize)]
pub struct TraceInspection {
    pub records: u64,
    pub decisions: u64,
    pub branching_records: u64,
    pub runnable: u64,
    pub select_poll: u64,
    pub select_result: u64,
    pub sites: Vec<TraceSite>,
    pub replay_decisions: Vec<TraceDecision>,
    pub tape_sha256: String,
}

#[derive(Serialize)]
pub struct TraceSite {
    pub fingerprint: String,
    pub kind: &'static str,
    pub count: u64,
    pub maximum_alternatives: u32,
    #[serde(skip)]
    pub site_offset: u64,
    #[serde(skip)]
    pub site_missing: bool,
}

#[derive(Serialize)]
pub struct TraceDecision {
    pub ordinal: u64,
    pub kind: &'static str,
    pub site_offset: u64,
    pub site_missing: bool,
    pub alternatives: u32,
    pub selected: u32,
}

fn kind_name(kind: u8) -> &'static str {
    match kind {
        1 => "runnable",
        2 => "select-poll",
        3 => "select-result",
        4 => "goroutine-creation",
        _ => unreachable!("validated choice kind"),
    }
}

pub(super) fn site_fingerprint(
    target: [u8; 32],
    kind: u8,
    offset: u64,
    missing: bool,
) -> Option<String> {
    if missing {
        return None;
    }
    let mut hash = Sha256::new();
    hash.update(target);
    hash.update([kind]);
    hash.update(offset.to_be_bytes());
    Some(format!("{:x}", hash.finalize()))
}

/// # Errors
/// Refuses malformed traces and invalid replay identities using the replay projector.
pub fn inspect(
    trace: &[u8],
    identity: &ExecutionIdentity,
) -> Result<TraceInspection, &'static str> {
    let plan = project(trace, identity)?;
    let mut sites = std::collections::BTreeMap::<(u8, String), TraceSite>::new();
    let mut counts = [0; 3];
    for (ordinal, bytes) in trace.as_chunks::<RECORD_BYTES>().0.iter().enumerate() {
        let record = decode_record(bytes, ordinal)?;
        let decision = &record.decision;
        if decision.kind <= 3 {
            counts[usize::from(decision.kind - 1)] += 1;
        }
        let fingerprint = if decision.site_missing {
            "missing".into()
        } else {
            let mut hash = Sha256::new();
            hash.update(identity.target_sha256);
            hash.update([decision.kind]);
            hash.update(decision.site_offset.to_be_bytes());
            format!("{:x}", hash.finalize())
        };
        let site = sites
            .entry((decision.kind, fingerprint.clone()))
            .or_insert(TraceSite {
                fingerprint,
                kind: kind_name(decision.kind),
                count: 0,
                maximum_alternatives: 0,
                site_offset: decision.site_offset,
                site_missing: decision.site_missing,
            });
        site.count += 1;
        site.maximum_alternatives = site.maximum_alternatives.max(decision.alternatives);
    }
    Ok(TraceInspection {
        records: trace.len() as u64 / RECORD_BYTES as u64,
        decisions: plan.decisions.len() as u64,
        branching_records: plan.decisions.len() as u64,
        runnable: counts[0],
        select_poll: counts[1],
        select_result: counts[2],
        sites: sites.into_values().collect(),
        replay_decisions: plan
            .decisions
            .iter()
            .map(|decision| TraceDecision {
                ordinal: decision.ordinal,
                kind: kind_name(decision.kind),
                site_offset: decision.site_offset,
                site_missing: decision.site_missing,
                alternatives: decision.alternatives,
                selected: decision.selected,
            })
            .collect(),
        tape_sha256: crate::world::digest::hex(&plan.sha256),
    })
}

/// # Errors
/// Refuses malformed traces, invalid replay identities and unsupported terminals.
pub fn coverage_features(
    trace: &[u8],
    identity: &ExecutionIdentity,
    terminal: &str,
) -> Result<Vec<String>, &'static str> {
    if !matches!(terminal, "complete" | "overflow" | "diverged") {
        return Err("malformed_trace");
    }
    project(trace, identity)?;
    let mut features = std::collections::BTreeSet::new();
    let mut maximums = std::collections::BTreeMap::<String, u32>::new();
    let mut pairs = std::collections::BTreeSet::new();
    let mut previous = None;
    for (ordinal, bytes) in trace.as_chunks::<RECORD_BYTES>().0.iter().enumerate() {
        let record = decode_record(bytes, ordinal)?;
        let decision = &record.decision;
        if decision.kind == 4 {
            continue;
        }
        let role = if record.observation {
            "observation"
        } else {
            "decision"
        };
        let kind = kind_name(decision.kind);
        let site = if decision.site_missing {
            "missing".into()
        } else {
            let mut hash = Sha256::new();
            hash.update(identity.target_sha256);
            hash.update([decision.kind]);
            hash.update(decision.site_offset.to_be_bytes());
            format!("{:x}", hash.finalize())
        };
        let alternative = if decision.alternatives == 1 {
            "only"
        } else if decision.selected == 0 {
            "first"
        } else if decision.selected == decision.alternatives - 1 {
            "last"
        } else {
            "interior"
        };
        features.insert(format!("record_kind/{role}/{kind}"));
        let site_identity = format!("{kind}/{site}");
        features.insert(format!("site/{site_identity}"));
        let record_identity = format!("{role}/{site_identity}/{alternative}");
        features.insert(format!("selected_alternative/{record_identity}"));
        if let Some(previous) = previous {
            let pair = format!("{previous}->{record_identity}");
            if pairs.len() < 4096 {
                pairs.insert(pair);
            }
        }
        previous = Some(record_identity);
        if !record.observation {
            let maximum = maximums.entry(site_identity).or_default();
            *maximum = (*maximum).max(decision.alternatives);
        }
    }
    for (site, maximum) in maximums {
        if maximum > 1 {
            features.insert(format!("branching_site/{site}/max-alternatives={maximum}"));
        }
    }
    features.extend(
        pairs
            .into_iter()
            .map(|pair| format!("adjacent_pair/{pair}")),
    );
    features.insert(format!("terminal/{terminal}"));
    Ok(features.into_iter().collect())
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PrefixControl {
    pub cutoff: u64,
    pub r#override: Option<RankOverride>,
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RankOverride {
    pub ordinal: u64,
    pub rank: u32,
    pub reconstruct: bool,
}

impl ReplayPlan {
    #[must_use]
    pub fn no_op_select_polls(&self) -> Vec<bool> {
        let mut marks = vec![false; self.decisions.len()];
        let mut start = 0;
        while start < self.decisions.len() {
            let first = &self.decisions[start];
            if first.kind != 2 || first.alternatives != 2 {
                start += 1;
                continue;
            }
            let mut end = start + 1;
            while end < self.decisions.len() {
                let previous = &self.decisions[end - 1];
                let current = &self.decisions[end];
                if current.kind != 2
                    || current.site() != previous.site()
                    || previous.alternatives.checked_add(1) != Some(current.alternatives)
                {
                    break;
                }
                end += 1;
            }
            let readiness = |index: usize| {
                let offset = index
                    .checked_mul(RECORD_BYTES)?
                    .checked_add(HEADER_BYTES)?
                    .checked_add(10)?;
                self.bytes
                    .get(offset..)?
                    .first_chunk::<2>()
                    .map(|bytes| u16::from_be_bytes(*bytes))
            };
            let Some(word) = readiness(start) else {
                start = end;
                continue;
            };
            let known = word & 1 != 0;
            let ready = word >> 6;
            let flags = word & 0b11_1110;
            let proven_shape = matches!((ready, flags), (0, 0 | 2 | 8) | (1, 0 | 4 | 8 | 16));
            let eligible = known
                && proven_shape
                && self.decisions[end - 1].alternatives == 2
                && (start..end).all(|index| readiness(index) == Some(word));
            marks[start..end].fill(eligible);
            start = end;
        }
        marks
    }

    /// # Errors
    /// Refuses controls outside this validated tape or ranks outside the selected alternative set.
    pub fn prefix(mut self, control: &PrefixControl) -> Result<Self, &'static str> {
        let cutoff = usize::try_from(control.cutoff).map_err(|_| "invalid_replay_plan")?;
        if cutoff > self.decisions.len() {
            return Err("invalid_replay_plan");
        }
        self.decisions.truncate(cutoff);
        self.bytes.truncate(HEADER_BYTES + cutoff * RECORD_BYTES);
        if let Some(rank) = &control.r#override {
            if cutoff == 0 || rank.ordinal != control.cutoff - 1 {
                return Err("invalid_replay_plan");
            }
            let last = self.decisions.last_mut().ok_or("invalid_replay_plan")?;
            if rank.rank >= last.alternatives || (rank.rank == last.selected) != rank.reconstruct {
                return Err("invalid_replay_plan");
            }
            last.selected = rank.rank;
            last.selected_identity = [0; 32];
            last.rank_override = true;
            let offset = HEADER_BYTES + (cutoff - 1) * RECORD_BYTES;
            self.bytes[offset + 9] |= 8;
            set32(&mut self.bytes, offset + 16, rank.rank);
            self.bytes[offset + 32..offset + 64].fill(0);
            let mut source = Sha256::new();
            source.update(b"gomad-choice-rank-prefix/v1\0");
            for bytes in self.bytes[HEADER_BYTES..].as_chunks::<RECORD_BYTES>().0 {
                let mut record = *bytes;
                record[10..12].fill(0);
                source.update(record);
            }
            self.bytes[40..72].copy_from_slice(&source.finalize());
        }
        let length = self.bytes.len() as u64;
        set64(&mut self.bytes, 24, length);
        set64(&mut self.bytes, 32, control.cutoff);
        let payload = Sha256::digest(&self.bytes[HEADER_BYTES..]);
        self.bytes[200..232].copy_from_slice(&payload);
        let checksum = Sha256::digest(&self.bytes[..TAPE_CHECKSUM_OFFSET]);
        self.bytes[TAPE_CHECKSUM_OFFSET..HEADER_BYTES].copy_from_slice(&checksum);
        self.sha256 = Sha256::digest(&self.bytes).into();
        Ok(self)
    }
}

struct Record {
    decision: Decision,
    observation: bool,
    readiness: u16,
    origin: u64,
}

fn decode_record(bytes: &[u8; RECORD_BYTES], ordinal: usize) -> Result<Record, &'static str> {
    let kind = bytes[8];
    let flags = bytes[9];
    let readiness = u16::from_be_bytes(bytes[10..12].try_into().unwrap());
    let alternatives = get32(bytes, 12);
    let selected = get32(bytes, 16);
    let site_offset = get64(bytes, 24);
    let observation = flags & 2 != 0;
    let site_missing = flags & 4 != 0;
    let selected_identity = bytes[32..64].try_into().unwrap();
    let alternative_set_digest = bytes[64..96].try_into().unwrap();
    let origin = if observation { get64(bytes, 32) } else { 0 };
    let creation = kind == 4;
    if !(1..=4).contains(&kind)
        || flags & !7 != 0
        || flags & 3
            != if creation {
                0
            } else if observation {
                2
            } else {
                1
            }
        || observation != (kind == 3)
        || alternatives == 0
        || selected >= alternatives
        || site_missing && site_offset != 0
        || get64(bytes, 0) != ordinal as u64
        || !observation
            && !creation
            && (readiness != 0 || selected_identity == [0; 32] || alternative_set_digest == [0; 32])
        || observation && (bytes[40..].iter().any(|byte| *byte != 0) || origin > ordinal as u64)
        || observation
            && (readiness & 1 == 0
                || u32::from(readiness >> 6) > alternatives - u32::from(readiness & 2 != 0))
        || creation
            && (flags & !4 != 0
                || !(2..=256).contains(&alternatives)
                || readiness != 0
                || get32(bytes, 20) != 0
                || selected_identity == [0; 32]
                || alternative_set_digest != [0; 32])
    {
        return Err("malformed_trace");
    }
    Ok(Record {
        decision: Decision {
            ordinal: ordinal as u64,
            kind,
            site_offset,
            site_missing,
            rank_override: false,
            alternatives,
            selected,
            data: get32(bytes, 20),
            selected_identity: if observation {
                [0; 32]
            } else {
                selected_identity
            },
            alternative_set_digest,
            creation_sites: Vec::new(),
        },
        observation,
        readiness,
        origin,
    })
}

pub(super) fn toolchain_key(identity: &ExecutionIdentity) -> Result<[u8; 32], &'static str> {
    if identity.target_sha256 == [0; 32]
        || identity.implementation_sha256 == [0; 32]
        || identity.goos.is_empty()
        || identity.goarch.is_empty()
        || identity.toolchain_build_key.len() != 64
        || !identity
            .toolchain_build_key
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
    {
        return Err("invalid_replay_plan");
    }
    let mut key = [0; 32];
    for (index, byte) in key.iter_mut().enumerate() {
        *byte = u8::from_str_radix(&identity.toolchain_build_key[index * 2..index * 2 + 2], 16)
            .map_err(|_| "invalid_replay_plan")?;
    }
    if key == [0; 32] {
        return Err("invalid_replay_plan");
    }
    Ok(key)
}

fn project_creation_sites(records: &mut [Record]) -> Result<(), &'static str> {
    let mut pending = Vec::new();
    let mut pending_count = 0;
    for record in records {
        let decision = &mut record.decision;
        if decision.kind == 4 {
            if pending.is_empty() {
                pending_count = decision.alternatives;
            }
            if decision.selected as usize != pending.len() || decision.alternatives != pending_count
            {
                return Err("malformed_trace");
            }
            pending.push(CreationSite {
                identity: decision.selected_identity,
                site_offset: decision.site_offset,
                site_missing: decision.site_missing,
            });
        } else if !pending.is_empty() {
            if decision.kind != 1
                || pending.len() != decision.alternatives as usize
                || pending_count != decision.alternatives
                || pending
                    .windows(2)
                    .any(|sites| sites[0].identity >= sites[1].identity)
            {
                return Err("malformed_trace");
            }
            let mut hash = Sha256::new();
            hash.update(b"gomad-choice-alternative-set/v1\0");
            hash.update((pending.len() as u64).to_be_bytes());
            for site in &pending {
                hash.update(site.identity);
            }
            if <[u8; 32]>::from(hash.finalize()) != decision.alternative_set_digest {
                return Err("malformed_trace");
            }
            decision.creation_sites = std::mem::take(&mut pending);
        }
    }
    if !pending.is_empty() {
        return Err("malformed_trace");
    }
    Ok(())
}

/// # Errors
/// Maps branching decisions to their runtime predecision coordinates, including interrupted select prefixes.
pub fn inspection_coordinates(trace: &[u8]) -> Result<Vec<u64>, &'static str> {
    let (raw, remainder) = trace.as_chunks::<RECORD_BYTES>();
    if !remainder.is_empty() {
        return Err("malformed_trace");
    }
    let mut records = raw
        .iter()
        .enumerate()
        .map(|(ordinal, bytes)| decode_record(bytes, ordinal))
        .collect::<Result<Vec<_>, _>>()?;
    project_creation_sites(&mut records)?;
    let mut coordinates = Vec::new();
    let mut presented_at = 0;
    for (index, record) in records.iter().enumerate() {
        if index == 0 || records[index - 1].decision.kind != 4 {
            presented_at = record.decision.ordinal;
        }
        if !record.observation && record.decision.kind != 4 && record.decision.alternatives >= 2 {
            coordinates.push(presented_at);
        }
    }
    Ok(coordinates)
}

/// # Errors
/// Refuses malformed raw records, inconsistent select observations and invalid execution identities.
pub fn project(trace: &[u8], identity: &ExecutionIdentity) -> Result<ReplayPlan, &'static str> {
    let (raw_records, remainder) = trace.as_chunks::<RECORD_BYTES>();
    if !remainder.is_empty() {
        return Err("malformed_trace");
    }
    let mut records: Vec<_> = raw_records
        .iter()
        .enumerate()
        .map(|(ordinal, bytes)| decode_record(bytes, ordinal))
        .collect::<Result<_, _>>()?;
    project_creation_sites(&mut records)?;
    let mut decisions = Vec::new();
    let mut indices = vec![None; records.len()];
    let mut payload = Vec::new();
    for (ordinal, record) in records.iter().enumerate() {
        if record.observation || record.decision.kind == 4 || record.decision.alternatives < 2 {
            continue;
        }
        let mut decision = record.decision.clone();
        decision.ordinal = decisions.len() as u64;
        indices[ordinal] = Some(decisions.len());
        let start = payload.len();
        payload.extend_from_slice(&raw_records[ordinal]);
        set64(&mut payload, start, decision.ordinal);
        decisions.push(decision);
    }
    for result in records
        .iter()
        .filter(|record| record.observation && record.decision.data >= 2)
    {
        let polled = u64::from(result.decision.data) - 1;
        if polled > result.decision.ordinal - result.origin {
            return Err("invalid_replay_plan");
        }
        for step in 0..polled {
            let ordinal =
                usize::try_from(result.origin + step).map_err(|_| "invalid_replay_plan")?;
            let poll = &records[ordinal].decision;
            if poll.kind != 2
                || records[ordinal].observation
                || poll.site_offset != result.decision.site_offset
                || poll.site_missing != result.decision.site_missing
                || u64::from(poll.alternatives) != step + 2
            {
                return Err("invalid_replay_plan");
            }
            let index = indices[ordinal].ok_or("invalid_replay_plan")?;
            let offset = index * RECORD_BYTES + 10;
            if payload[offset..offset + 2] != [0, 0] {
                return Err("invalid_replay_plan");
            }
            payload[offset..offset + 2].copy_from_slice(&result.readiness.to_be_bytes());
        }
    }
    let key = toolchain_key(identity)?;
    let mut platform = Sha256::new();
    platform.update(b"gomad-choice-platform/v1\0");
    platform.update(identity.goos.as_bytes());
    platform.update([0]);
    platform.update(identity.goarch.as_bytes());
    let mut bytes = vec![0; HEADER_BYTES];
    bytes[..8].copy_from_slice(&TAPE_MAGIC);
    set32(&mut bytes, 8, 3);
    set32(&mut bytes, 12, TAPE_HEADER_WIRE_BYTES);
    set32(&mut bytes, 16, TAPE_RECORD_BYTES);
    set64(&mut bytes, 24, (HEADER_BYTES + payload.len()) as u64);
    set64(&mut bytes, 32, decisions.len() as u64);
    bytes[40..72].copy_from_slice(&Sha256::digest(trace));
    bytes[72..104].copy_from_slice(&identity.target_sha256);
    bytes[104..136].copy_from_slice(&identity.implementation_sha256);
    bytes[136..168].copy_from_slice(&key);
    bytes[168..200].copy_from_slice(&platform.finalize());
    bytes[200..232].copy_from_slice(&Sha256::digest(&payload));
    let checksum = Sha256::digest(&bytes[..TAPE_CHECKSUM_OFFSET]);
    bytes[TAPE_CHECKSUM_OFFSET..HEADER_BYTES].copy_from_slice(&checksum);
    bytes.extend(payload);
    Ok(ReplayPlan {
        decisions,
        sha256: Sha256::digest(&bytes).into(),
        bytes,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{Engine as _, engine::general_purpose::STANDARD};
    use serde_json::Value;
    use std::collections::BTreeSet;
    use std::path::PathBuf;

    #[test]
    fn runnable_creation_metadata_projects_without_forcing_metadata() {
        let identity = ExecutionIdentity {
            target_sha256: [1; 32],
            toolchain_build_key: "02".repeat(32),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [3; 32],
        };
        let mut trace = [0; 288];
        for index in 0..2 {
            let start = index * 96;
            set64(&mut trace, start, index as u64);
            trace[start + 8] = 4;
            set32(&mut trace, start + 12, 2);
            set32(&mut trace, start + 16, u32::try_from(index).unwrap());
            set64(&mut trace, start + 24, 123 + index as u64);
            trace[start + 32..start + 64].fill(u8::try_from(index + 1).unwrap());
        }
        set64(&mut trace, 192, 2);
        trace[200] = 1;
        trace[201] = 5;
        set32(&mut trace, 204, 2);
        trace[224..256].fill(1);
        let mut hash = Sha256::new();
        hash.update(b"gomad-choice-alternative-set/v1\0");
        hash.update(2u64.to_be_bytes());
        hash.update([1; 32]);
        hash.update([2; 32]);
        trace[256..288].copy_from_slice(&hash.finalize());
        let plan = project(&trace, &identity).unwrap();
        assert_eq!(plan.decisions.len(), 1);
        assert_eq!(plan.decisions[0].ordinal(), 0);
    }

    #[test]
    fn replay_plan_go_corpus() {
        let directory =
            PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/replay_plan");
        let mut paths: Vec<_> = std::fs::read_dir(directory)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| {
                path.extension()
                    .is_some_and(|extension| extension == "json")
            })
            .collect();
        paths.sort();
        let mut names = BTreeSet::new();
        for path in paths {
            let corpus: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
            let name = corpus["name"].as_str().unwrap();
            assert_eq!(
                corpus["schema"], "gomad.replay-plan-conformance/v1",
                "{name}"
            );
            assert_eq!(path.file_stem().unwrap().to_str().unwrap(), name);
            assert!(names.insert(name.to_owned()));
            let identity = serde_json::from_value(corpus["identity"].clone()).unwrap();
            let trace = STANDARD.decode(corpus["trace"].as_str().unwrap()).unwrap();
            let expected = &corpus["expected"];
            let projected = project(&trace, &identity).and_then(|plan| {
                if let Some(control) = corpus.get("prefix_control") {
                    plan.prefix(&serde_json::from_value(control.clone()).unwrap())
                } else {
                    Ok(plan)
                }
            });
            match projected {
                Err(error) => assert_eq!(error, expected["error"].as_str().unwrap(), "{name}"),
                Ok(plan) => {
                    assert_eq!(expected["error"], "", "{name}");
                    assert_eq!(
                        plan.bytes,
                        STANDARD
                            .decode(expected["bytes"].as_str().unwrap())
                            .unwrap(),
                        "{name}"
                    );
                    assert_eq!(
                        format!("{:x}", sha2::Sha256::digest(&plan.bytes)),
                        expected["sha256"].as_str().unwrap(),
                        "{name}"
                    );
                    assert_eq!(
                        plan.sha256.as_slice(),
                        sha2::Sha256::digest(&plan.bytes).as_slice(),
                        "{name}"
                    );
                    assert_eq!(
                        serde_json::to_value(plan.decisions).unwrap(),
                        expected["decisions"],
                        "{name}"
                    );
                }
            }
        }
        for required in [
            "empty",
            "observation-only",
            "single-alternative",
            "select-readiness-origin",
            "unfinished-select-prefix",
            "double-named",
            "before-origin",
            "storage-crash-interrupted",
            "prefix-zero",
            "prefix-full",
            "prefix-short",
            "prefix-rank",
            "prefix-forced",
            "prefix-rank-not-final",
        ] {
            assert!(names.contains(required), "missing {required}");
        }
    }
}
