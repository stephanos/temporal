use super::{Entry, Result, UUID, baseline, cached, dispositions, inventory, method, report};
use crate::foundation::dc::{BLOB_SIZE_ERROR, Limits, MAX_ID_LENGTH};
use std::{fmt::Write, sync::OnceLock};
use umpire_catalog::{
    Expectation, GapEntry, Violation, WireCase, WireField,
    generic_probe::{Engine, ProbeSystem},
};
use umpire_runtime::annotations::{ProbeCase, ProbeObservation, ProbeResponse};
use umpire_temporal_api::{
    WorkflowServiceMethod as Method, mutate_workflow_request, umpire::temporal::v1::ProbeRequest,
    workflow_field_claim, workflow_request_fields,
};

pub(super) struct Activity {
    strict: &'static [Entry],
    dispositions: OnceLock<dispositions::Dispositions>,
}

pub(super) fn engine() -> Result<&'static Engine<Activity>> {
    static ENGINE: OnceLock<Engine<Activity>> = OnceLock::new();
    cached(&ENGINE, || {
        Engine::new(Activity { strict: inventory()?, dispositions: OnceLock::new() })
    })
}

impl ProbeSystem for Activity {
    type Method = Method;
    type Context = ProbeRequest;
    fn methods(&self) -> Vec<(Method, &'static str)> {
        vec![
            (Method::StartActivityExecution, "start"),
            (Method::DescribeActivityExecution, "scheduled"),
            (Method::PollActivityExecution, "completed"),
            (Method::RequestCancelActivityExecution, "scheduled"),
            (Method::PollActivityTaskQueue, "poll"),
            (Method::RespondActivityTaskCompleted, "token"),
            (Method::RespondActivityTaskFailed, "token"),
            (Method::RespondActivityTaskCanceled, "canceled_token"),
        ]
    }
    fn name(&self, method: Method) -> &'static str {
        method.name()
    }
    fn rpc_path(&self, value: Method) -> String {
        method(value.name())
    }
    fn fields(&self, method: Method) -> Vec<WireField> {
        workflow_request_fields(method)
    }
    fn admits_case(&self, method: Method, case: &WireCase) -> Result<bool> {
        let mut ignored = false;
        let mut secret_bound = false;
        for ancestor in workflow_request_fields(method) {
            if case.field.path != ancestor.path
                && !case
                    .field
                    .path
                    .strip_prefix(&ancestor.path)
                    .is_some_and(|rest| rest.starts_with('.'))
            {
                continue;
            }
            if let Some(claim) = workflow_field_claim(&ancestor.declaration) {
                ignored |= claim.ignored.is_some();
                secret_bound |= ancestor.path == case.field.path
                    && claim.kind == "Secret"
                    && matches!(case.mutation, umpire_catalog::WireMutation::LimitPlusOne { .. });
            }
        }
        if ignored || secret_bound {
            if !matches!(case.applicability, umpire_catalog::MutationApplicability::Applicable) {
                return Ok(false);
            }
            let bytes = self.baseline(method, &self.canonical_context())?;
            let mutated = match self.mutate(method, case, &bytes) {
                Ok(mutated) => mutated,
                Err(error) => match error.downcast_ref::<umpire_catalog::GenerationError>() {
                    Some(
                        umpire_catalog::GenerationError::SizeLimit
                        | umpire_catalog::GenerationError::UnsupportedType
                        | umpire_catalog::GenerationError::InapplicableMutation,
                    ) => return Ok(!ignored),
                    _ => return Err(error),
                },
            };
            return Ok(!self.findings(method, &mutated)?.is_empty());
        }
        Ok(true)
    }
    fn bounds(&self) -> Vec<u64> {
        let limits = Limits::default();
        vec![limits.get(&MAX_ID_LENGTH), limits.get(&BLOB_SIZE_ERROR)]
    }
    fn canonical_context(&self) -> ProbeRequest {
        ProbeRequest {
            namespace: "probe-namespace".into(),
            activity_id: "probe-activity".into(),
            task_queue: "probe-queue".into(),
            run_id: UUID.into(),
            task_token: vec![8, 1],
            ..Default::default()
        }
    }
    fn baseline(&self, method: Method, context: &ProbeRequest) -> Result<Vec<u8>> {
        baseline(method.name(), context, "")
    }
    fn mutate(&self, method: Method, case: &WireCase, bytes: &[u8]) -> Result<Vec<u8>> {
        Ok(mutate_workflow_request(method, bytes, &case.field.path, &case.mutation)?)
    }
    fn findings(&self, method: Method, bytes: &[u8]) -> Result<Vec<Violation>> {
        super::request_findings(method.name(), bytes)
    }
    fn prediction(&self, method: Method, finding: Violation) -> Result<Violation> {
        self.strict
            .iter()
            .find(|entry| {
                entry.method == method.name() && same_violation(&entry.target.violation, &finding)
            })
            .map(|entry| entry.target.violation.clone())
            .ok_or_else(|| "generic mutation predicts an unregistered catalog claim".into())
    }
    fn prediction_key(&self, finding: &Violation, _index: usize) -> String {
        let fingerprint =
            format!("{finding:?}").bytes().fold(0xcbf2_9ce4_8422_2325_u64, |hash, byte| {
                (hash ^ u64::from(byte)).wrapping_mul(0x100_0000_01b3)
            });
        format!("{}:{}:{fingerprint:016x}", finding.path, finding.constraint)
    }
    fn contextual_unchanged(&self) -> bool {
        true
    }
    fn additional_cases(&self) -> Result<Vec<ProbeCase>> {
        self.strict.iter().map(Entry::metadata).collect()
    }
    fn prepare_additional(
        &self,
        id: &str,
        baseline: bool,
        context: &ProbeRequest,
    ) -> Result<ProbeResponse> {
        super::prepare_strict(
            &ProbeRequest { case_id: id.into(), baseline, ..context.clone() },
            self.strict,
        )
    }
    fn validate_inventory(&self, ids: &[String]) -> Result<()> {
        let registry = dispositions::Dispositions::with_unannotated(
            ids.iter().map(String::as_str),
            crate::activity_api::accepted_probe_gaps(),
            dispositions::unannotated(),
        )?;
        self.dispositions.get_or_init(|| registry);
        Ok(())
    }
    fn classify_additional(&self, observations: &[ProbeObservation]) -> Result<ProbeResponse> {
        report::classify(
            observations,
            self.strict,
            self.dispositions.get().ok_or("missing admitted dispositions")?,
        )
    }
    fn apply_disposition(&self, id: &str, row: &mut GapEntry) {
        if let Some(dispositions) = self.dispositions.get() {
            dispositions.apply(id, row);
        }
    }
    fn finish_report(&self, response: &mut ProbeResponse) -> Result<()> {
        let loose = self
            .strict
            .iter()
            .filter(|entry| {
                matches!(
                    entry.target.violation.expected.as_ref().map(|expected| &expected.expectation),
                    Some(Expectation::RejectsCode(_) | Expectation::RejectsAnyStatus)
                )
            })
            .count();
        writeln!(response.report, "Loose expectations: {loose}")?;
        Ok(())
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
