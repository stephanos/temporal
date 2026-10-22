use crate::evidence::{
    canonical,
    inspection::{Details, Minimization},
    record::Record,
    store::{Opened, Publication},
};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};

pub(crate) const TRACE: &str = "minimization/source-choice.bin";
pub(crate) const DESCRIPTOR: &str = "minimization/source-choice.json";
const MAX_DESCRIPTOR: u64 = 16 << 20;

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Descriptor {
    schema: String,
    parent_record_sha256: String,
    identity: linked::ChoiceIdentity,
    limit: u64,
    trace_bytes: u64,
    trace_sha256: String,
    tape_sha256: String,
    decisions: u64,
    decisions_sha256: String,
}

pub(crate) struct Source {
    pub raw: Vec<u8>,
    pub identity: linked::ChoiceIdentity,
    pub limit: u64,
    pub parent_record_sha256: String,
}

impl Source {
    fn descriptor(&self) -> Result<Descriptor, String> {
        let plan = linked::project_choices(&self.raw, &self.identity)?;
        Ok(Descriptor {
            schema: "gomad.minimization-source-choice/v1".into(),
            identity: self.identity.clone(),
            parent_record_sha256: self.parent_record_sha256.clone(),
            limit: self.limit,
            trace_bytes: self.raw.len() as u64,
            trace_sha256: canonical::digest(&self.raw),
            tape_sha256: crate::campaign::exploration::choice::hex(&plan.sha256),
            decisions: plan.decisions.len() as u64,
            decisions_sha256: canonical::digest(&canonical::encode(&plan.decisions)?),
        })
    }

    pub(crate) fn attach(&self, publication: &mut Publication) -> Result<(), String> {
        if self.raw.len() as u64 > self.limit {
            return Err("minimization source trace exceeds parent bound".into());
        }
        let descriptor = canonical::encode(&self.descriptor()?)?;
        if descriptor.len() as u64 > MAX_DESCRIPTOR {
            return Err("minimization source descriptor exceeds bound".into());
        }
        publication.payloads.insert(TRACE.into(), self.raw.clone());
        publication.payloads.insert(DESCRIPTOR.into(), descriptor);
        Ok(())
    }
}

pub(crate) fn read(opened: &Opened) -> Result<Option<Source>, String> {
    let files = &opened.manifest().files;
    if !files.contains_key(TRACE) && !files.contains_key(DESCRIPTOR) {
        return Ok(None);
    }
    if opened.manifest().record.streams.contains_key(TRACE)
        || opened.manifest().record.streams.contains_key(DESCRIPTOR)
    {
        return Err("minimization source must be manifest-only provenance".into());
    }
    let bytes = opened.payload(DESCRIPTOR, MAX_DESCRIPTOR)?;
    let descriptor: Descriptor = serde_json::from_slice(&bytes)
        .map_err(|error| format!("decode minimization source descriptor: {error}"))?;
    let original = Record::decode(&opened.payload(super::predicate::ORIGINAL, 16 << 20)?)?;
    if descriptor.schema != "gomad.minimization-source-choice/v1"
        || canonical::encode(&descriptor)? != bytes
        || descriptor.limit > original.limits.evidence_bytes
        || descriptor.trace_bytes > descriptor.limit
        || descriptor.identity.toolchain_build_key != original.identity.toolchain
        || crate::campaign::exploration::choice::hex(&descriptor.identity.target_sha256)
            != original.identity.target
    {
        return Err("minimization source descriptor identity or bound mismatch".into());
    }
    let details = Details::read(opened)?.ok_or("minimization source inspection missing")?;
    let evidence = details
        .choices
        .as_ref()
        .ok_or("minimization source choice evidence missing")?;
    let metadata = details
        .minimization
        .as_ref()
        .ok_or("minimization source metadata missing")?;
    if canonical::encode(&descriptor.identity)? != canonical::encode(&evidence.identity)?
        || descriptor.decisions != metadata.original_forced_decisions
        || descriptor.parent_record_sha256 != metadata.parent_record_hash
        || !canonical::valid_digest(&descriptor.parent_record_sha256)
    {
        return Err("minimization source differs from original parent".into());
    }
    let source = Source {
        raw: opened.payload(TRACE, descriptor.limit)?,
        identity: descriptor.identity,
        limit: descriptor.limit,
        parent_record_sha256: descriptor.parent_record_sha256,
    };
    if canonical::encode(&source.descriptor()?)? != bytes {
        return Err("minimization source tape commitment mismatch".into());
    }
    let plan = linked::project_choices(&source.raw, &source.identity)?;
    let identities = plan
        .decisions
        .iter()
        .map(|decision| {
            (
                decision.ordinal(),
                crate::campaign::exploration::choice::hex(&decision.selected_identity()),
            )
        })
        .collect::<std::collections::BTreeMap<_, _>>();
    validate_candidates(plan, metadata)?;
    for decision in metadata
        .accepted
        .iter()
        .flat_map(|reduction| &reduction.removed)
    {
        if decision.dimension != "runtime"
            || identities.get(&decision.ordinal) != Some(&decision.identity)
        {
            return Err("minimization removed identity differs from source tape".into());
        }
    }
    Ok(Some(source))
}

fn validate_candidates(
    mut plan: linked::ChoicePlan,
    metadata: &Minimization,
) -> Result<(), String> {
    for (cutoff, expected) in [
        (
            metadata.original_forced_decisions,
            &metadata.original_candidate_sha256,
        ),
        (
            metadata.final_forced_decisions,
            &metadata.final_candidate_sha256,
        ),
    ] {
        plan = plan.prefix(&linked::PrefixControl {
            cutoff,
            r#override: None,
        })?;
        if super::proposals::choice_prefix_sha256(cutoff, &plan.sha256)
            .trim_start_matches("sha256:")
            != expected
        {
            return Err("minimization source candidate commitment mismatch".into());
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "source_tests.rs"]
mod tests;
