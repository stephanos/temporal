use super::{Entry as StrictEntry, Result, baseline, method, recipe};
use crate::foundation::dc::{BLOB_SIZE_ERROR, Limits, MAX_ID_LENGTH};
use std::{collections::BTreeSet, sync::OnceLock};
use umpire_annotations::{
    Expectation, GapEntry, MutationApplicability, ProbeResult, Violation, WireCase,
    mutation_entries, wire_cases,
};
use umpire_runtime::annotations::ProbeApplicability;
use umpire_runtime::annotations::ProbeCase;
use umpire_runtime::annotations::ProbeExpectation;
use umpire_runtime::annotations::ProbeObservation;
use umpire_runtime::annotations::ProbeResponse;
use umpire_temporal_api::{
    WorkflowServiceMethod, mutate_workflow_request, umpire::temporal::v1::ProbeRequest,
    workflow_request_fields,
};

pub(super) const MAX_CASES: usize = 32_768;
pub(super) const MAX_PREDICTIONS: usize = 256;

pub(super) struct Entry {
    pub method: WorkflowServiceMethod,
    prerequisite: &'static str,
    pub case: WireCase,
    canonical: OnceLock<ProbeCase>,
}

pub(super) fn inventory() -> Result<&'static [Entry]> {
    static ENTRIES: OnceLock<Vec<Entry>> = OnceLock::new();
    Ok(super::cached(&ENTRIES, build_inventory)?.as_slice())
}

fn build_inventory() -> Result<Vec<Entry>> {
    let limits = Limits::default();
    let bounds = [limits.get(&MAX_ID_LENGTH), limits.get(&BLOB_SIZE_ERROR)];
    let methods = [
        (WorkflowServiceMethod::StartActivityExecution, "start"),
        (WorkflowServiceMethod::DescribeActivityExecution, "scheduled"),
        (WorkflowServiceMethod::PollActivityExecution, "completed"),
        (WorkflowServiceMethod::RequestCancelActivityExecution, "scheduled"),
        (WorkflowServiceMethod::PollActivityTaskQueue, "poll"),
        (WorkflowServiceMethod::RespondActivityTaskCompleted, "token"),
        (WorkflowServiceMethod::RespondActivityTaskFailed, "token"),
        (WorkflowServiceMethod::RespondActivityTaskCanceled, "canceled_token"),
    ];
    let mut entries = Vec::new();
    for (method, prerequisite) in methods {
        let fields = workflow_request_fields(method);
        for case in wire_cases(&fields, &bounds) {
            entries.push(Entry { method, prerequisite, case, canonical: OnceLock::new() });
            if entries.len() > MAX_CASES {
                return Err("generic probe inventory exceeds metadata budget".into());
            }
        }
    }
    entries.sort_by_key(Entry::id);
    let ids: BTreeSet<_> = entries.iter().map(Entry::id).collect();
    if ids.len() != entries.len() {
        return Err("duplicate generic probe identity".into());
    }
    Ok(entries)
}

impl Entry {
    pub(super) fn id(&self) -> String {
        format!("generic::{}::{}", self.method.name(), self.case.id())
    }

    pub(super) fn metadata(&self) -> Result<ProbeCase> {
        let (applicability, mut refusal) = applicability(&self.case.applicability);
        if matches!(self.case.applicability, MutationApplicability::NotApplicable(_)) {
            refusal =
                format!("{} is not applicable to this wire carrier", self.case.mutation.name());
        }
        let case = ProbeCase {
            id: self.id(),
            method: method(self.method.name()),
            prerequisite: self.prerequisite.into(),
            generic: true,
            applicability: applicability as i32,
            refusal,
            ..Default::default()
        };
        if case.id.len() > 4096 || case.refusal.len() > 256 {
            return Err("generic probe identity exceeds metadata budget".into());
        }
        Ok(case)
    }

    pub(super) fn declaration(&self, strict: &[StrictEntry]) -> Result<ProbeCase> {
        let mut metadata = self.metadata()?;
        let canonical = self.canonical(strict)?;
        metadata.predictions.clone_from(&canonical.predictions);
        if canonical.applicability != ProbeApplicability::Unchanged as i32 {
            metadata.applicability = canonical.applicability;
            metadata.refusal.clone_from(&canonical.refusal);
        }
        Ok(metadata)
    }

    pub(super) fn prepare(
        &self,
        request: &ProbeRequest,
        strict: &[StrictEntry],
    ) -> Result<ProbeResponse> {
        let (metadata, bytes) = self.materialize(request, strict)?;
        if !request.baseline {
            let expected = self.canonical(strict)?;
            if !compatible_state(metadata.applicability, &metadata.refusal, expected)
                || metadata.predictions != expected.predictions
            {
                return Err("generic prediction context diverges from canonical preparation".into());
            }
        }
        let mut response = ProbeResponse {
            method: metadata.method.clone(),
            cases: vec![metadata],
            ..Default::default()
        };
        if bytes.len() > 512 * 1024 {
            response.recipe = Some(recipe(&bytes)?);
        } else {
            response.message = bytes;
        }
        Ok(response)
    }

    fn canonical(&self, strict: &[StrictEntry]) -> Result<&ProbeCase> {
        super::cached(&self.canonical, || {
            Ok(self
                .materialize(
                    &ProbeRequest {
                        namespace: "probe-namespace".into(),
                        activity_id: "probe-activity".into(),
                        task_queue: "probe-queue".into(),
                        run_id: super::UUID.into(),
                        task_token: vec![8, 1],
                        ..Default::default()
                    },
                    strict,
                )?
                .0)
        })
    }

    fn materialize(
        &self,
        request: &ProbeRequest,
        strict: &[StrictEntry],
    ) -> Result<(ProbeCase, Vec<u8>)> {
        let mut metadata = self.metadata()?;
        if !matches!(self.case.applicability, MutationApplicability::Applicable) {
            return Ok((metadata, Vec::new()));
        }
        let wire = baseline(self.method.name(), request, "")?.decoded()?;
        if !wire.findings()?.is_empty() {
            return Err("invalid generic probe baseline".into());
        }
        let original = wire.bytes();
        let bytes = if request.baseline {
            original.clone()
        } else if let Ok(bytes) = mutate_workflow_request(
            self.method,
            &original,
            &self.case.field.path,
            &self.case.mutation,
        ) {
            bytes
        } else {
            metadata.applicability = ProbeApplicability::Refused as i32;
            metadata.refusal = "typed mutation construction refused".into();
            return Ok((metadata, Vec::new()));
        };
        if !request.baseline && bytes == original {
            metadata.applicability = ProbeApplicability::Unchanged as i32;
            metadata.refusal = "mutation leaves the baseline unchanged".into();
            return Ok((metadata, Vec::new()));
        }
        if !request.baseline {
            metadata.predictions =
                self.predictions(wire.decoded_from(&bytes)?.findings()?, strict)?;
        }
        Ok((metadata, bytes))
    }

    fn predictions(
        &self,
        mut findings: Vec<Violation>,
        strict: &[StrictEntry],
    ) -> Result<Vec<ProbeExpectation>> {
        findings.sort_by_key(|finding| {
            (finding.path.clone(), finding.constraint.clone(), finding.fields.clone())
        });
        findings.dedup();
        let mut predictions = Vec::new();
        for finding in findings {
            let Some(expected) = &finding.expected else { continue };
            if matches!(expected.expectation, Expectation::Unobservable) {
                continue;
            }
            let declared = strict
                .iter()
                .find(|entry| {
                    entry.method == self.method.name()
                        && same_violation(&entry.target.violation, &finding)
                })
                .ok_or("generic mutation predicts an unregistered catalog claim")?;
            let metadata = declared.metadata()?;
            predictions.push(ProbeExpectation {
                id: self.prediction_id(declared),
                accepts: metadata.accepts,
                code: metadata.code,
                message_pattern: metadata.message_pattern,
                details_type: metadata.details_type,
            });
        }
        if predictions.len() > MAX_PREDICTIONS
            || predictions.iter().any(|prediction| prediction.id.len() > 4096)
        {
            return Err("generic predictions exceed metadata budget".into());
        }
        Ok(predictions)
    }

    fn prediction_id(&self, declared: &StrictEntry) -> String {
        let fingerprint = format!("{:?}", declared.target.violation)
            .bytes()
            .fold(0xcbf2_9ce4_8422_2325_u64, |hash, byte| {
                (hash ^ u64::from(byte)).wrapping_mul(0x100_0000_01b3)
            });
        format!(
            "{}::prediction:{}:{}:{fingerprint:016x}",
            self.id(),
            declared.target.violation.path,
            declared.target.violation.constraint
        )
    }

    pub(super) fn classify(
        &self,
        observation: &ProbeObservation,
        strict: &[StrictEntry],
    ) -> Result<Vec<GapEntry>> {
        let state = ProbeApplicability::try_from(observation.applicability)?;
        let expected = self.canonical(strict)?;
        if observation.predictions.len() > MAX_PREDICTIONS
            || observation.predictions.len() != expected.predictions.len()
            || observation.prediction_count as usize != observation.predictions.len()
            || observation.refusal.len() > 256
            || observation.message_matches
            || !observation.details_type.is_empty()
        {
            return Err("invalid generic observation metadata".into());
        }
        if !compatible_state(observation.applicability, &observation.refusal, expected) {
            return Err("generic observation diverges from canonical preparation".into());
        }
        let mut case = self.case.clone();
        if state != ProbeApplicability::Applicable {
            if observation.accepted || observation.code != 0 || !observation.predictions.is_empty()
            {
                return Err("invalid local mutation observation".into());
            }
            case.applicability = match state {
                ProbeApplicability::NotApplicable => {
                    MutationApplicability::NotApplicable(observation.refusal.clone())
                }
                ProbeApplicability::Refused => {
                    MutationApplicability::Refused(observation.refusal.clone())
                }
                ProbeApplicability::Unchanged => MutationApplicability::Unchanged,
                ProbeApplicability::Applicable => unreachable!(),
            };
            return Ok(mutation_entries(&case, &[], ProbeResult::Unverifiable)?);
        }
        if !matches!(case.applicability, MutationApplicability::Applicable)
            || !observation.refusal.is_empty()
            || (observation.accepted && observation.code != 0)
            || (!observation.accepted && !(1..=16).contains(&observation.code))
        {
            return Err("invalid dispatched mutation observation".into());
        }
        let result = |message_matches, details_type| {
            if observation.accepted {
                ProbeResult::Accepted
            } else {
                ProbeResult::Rejected { code: observation.code, message_matches, details_type }
            }
        };
        let mut indices = BTreeSet::new();
        let mut ids = BTreeSet::new();
        let mut predictions = Vec::new();
        for prediction in &observation.predictions {
            if prediction.index as usize >= observation.predictions.len()
                || expected
                    .predictions
                    .get(prediction.index as usize)
                    .is_none_or(|declared| declared.id != prediction.prediction_id)
                || !indices.insert(prediction.index)
                || prediction.prediction_id.len() > 4096
                || prediction.details_type.len() > 512
                || !ids.insert(&prediction.prediction_id)
            {
                return Err("invalid generic prediction correlation".into());
            }
            let declared = strict
                .iter()
                .find(|entry| {
                    entry.method == self.method.name()
                        && self.prediction_id(entry) == prediction.prediction_id
                })
                .ok_or("unknown, stale or foreign prediction identity")?;
            predictions.push((
                declared.target.violation.clone(),
                result(
                    prediction.message_matches,
                    (!prediction.details_type.is_empty()).then(|| prediction.details_type.clone()),
                ),
            ));
        }
        Ok(mutation_entries(&case, &predictions, result(false, None))?)
    }
}

fn same_violation(left: &Violation, right: &Violation) -> bool {
    let mut left_fields = left.fields.clone();
    let mut right_fields = right.fields.clone();
    left_fields.sort();
    right_fields.sort();
    left.path == right.path
        && left.constraint == right.constraint
        && left_fields == right_fields
        && left.expected == right.expected
}

fn compatible_state(actual: i32, refusal: &str, expected: &ProbeCase) -> bool {
    if actual == expected.applicability && refusal == expected.refusal {
        return true;
    }
    expected.predictions.is_empty()
        && matches!(expected.applicability, 0 | 3)
        && ((actual == 0 && refusal.is_empty())
            || (actual == 3 && refusal == "mutation leaves the baseline unchanged"))
}

fn applicability(value: &MutationApplicability) -> (ProbeApplicability, String) {
    match value {
        MutationApplicability::Applicable => (ProbeApplicability::Applicable, String::new()),
        MutationApplicability::NotApplicable(reason) => {
            (ProbeApplicability::NotApplicable, reason.clone())
        }
        MutationApplicability::Refused(reason) => (ProbeApplicability::Refused, reason.clone()),
        MutationApplicability::Unchanged => {
            (ProbeApplicability::Unchanged, "mutation leaves the baseline unchanged".into())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::Result;

    #[test]
    fn canonical_metadata_is_lazy_and_reused() -> Result<()> {
        let strict = super::super::inventory()?;
        let entries = super::build_inventory()?;
        assert!(entries.iter().all(|entry| entry.canonical.get().is_none()));
        let entry = entries
            .iter()
            .find(|entry| {
                entry.id().contains("DescribeActivityExecutionRequest.activity_id:mutation.empty")
            })
            .ok_or("missing canonical mutation")?;
        assert!(entry.canonical.get().is_none());
        assert_eq!(
            entry.canonical(&[]).err().ok_or("missing canonical validation refusal")?.to_string(),
            "generic mutation predicts an unregistered catalog claim"
        );
        assert!(entry.canonical.get().is_none());
        let canonical = entry.canonical(strict)?;
        assert_ne!(canonical.predictions.len(), 0);
        assert!(std::ptr::eq(canonical, entry.canonical(strict)?));
        Ok(())
    }
}
