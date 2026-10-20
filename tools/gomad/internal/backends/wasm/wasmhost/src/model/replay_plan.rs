use super::{Decision, get32, get64, set32, set64};
use serde::Deserialize;
use sha2::{Digest, Sha256};

const RECORD_BYTES: usize = 96;
const HEADER_BYTES: usize = 264;

#[derive(Clone, Deserialize)]
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

#[allow(
    dead_code,
    reason = "The replay preflight will consume the projected plan."
)]
pub struct ReplayPlan {
    pub decisions: Vec<Decision>,
    pub bytes: Vec<u8>,
    pub sha256: [u8; 32],
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
        let checksum = Sha256::digest(&self.bytes[..232]);
        self.bytes[232..264].copy_from_slice(&checksum);
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
    if !(1..=3).contains(&kind)
        || flags & !7 != 0
        || flags & 3 != if observation { 2 } else { 1 }
        || observation != (kind == 3)
        || alternatives == 0
        || selected >= alternatives
        || site_missing && site_offset != 0
        || get64(bytes, 0) != ordinal as u64
        || !observation
            && (readiness != 0 || selected_identity == [0; 32] || alternative_set_digest == [0; 32])
        || observation && (bytes[40..].iter().any(|byte| *byte != 0) || origin > ordinal as u64)
        || observation
            && (readiness & 1 == 0
                || u32::from(readiness >> 6) > alternatives - u32::from(readiness & 2 != 0))
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

#[allow(dead_code, reason = "The replay preflight will call this projection.")]
pub fn project(trace: &[u8], identity: &ExecutionIdentity) -> Result<ReplayPlan, &'static str> {
    let (raw_records, remainder) = trace.as_chunks::<RECORD_BYTES>();
    if !remainder.is_empty() {
        return Err("malformed_trace");
    }
    let records: Vec<_> = raw_records
        .iter()
        .enumerate()
        .map(|(ordinal, bytes)| decode_record(bytes, ordinal))
        .collect::<Result<_, _>>()?;
    let mut decisions = Vec::new();
    let mut indices = vec![None; records.len()];
    let mut payload = Vec::new();
    for (ordinal, record) in records.iter().enumerate() {
        if record.observation || record.decision.alternatives < 2 {
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
    bytes[..8].copy_from_slice(b"GOMADTP\x03");
    set32(&mut bytes, 8, 3);
    set32(&mut bytes, 12, 264);
    set32(&mut bytes, 16, 96);
    set64(&mut bytes, 24, (HEADER_BYTES + payload.len()) as u64);
    set64(&mut bytes, 32, decisions.len() as u64);
    bytes[40..72].copy_from_slice(&Sha256::digest(trace));
    bytes[72..104].copy_from_slice(&identity.target_sha256);
    bytes[104..136].copy_from_slice(&identity.implementation_sha256);
    bytes[136..168].copy_from_slice(&key);
    bytes[168..200].copy_from_slice(&platform.finalize());
    bytes[200..232].copy_from_slice(&Sha256::digest(&payload));
    let checksum = Sha256::digest(&bytes[..232]);
    bytes[232..264].copy_from_slice(&checksum);
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
