//! Stateless typed witnesses and value-free classification for activity RPCs.

use crate::activity_api::{catalog, worker_failure_presence};
use crate::foundation::dc::Limits;
use prost::Message;
use std::{collections::BTreeSet, sync::OnceLock};
use umpire_catalog::contextual_probe::{Entry, Registry, StrictSystem};
use umpire_catalog::{FieldDescriptor, Kind, Rng, Violation, validate};
use umpire_runtime::annotations::ProbeResponse;
use umpire_runtime::probe_recipe as recipe;
use umpire_runtime::prost_types::Duration;
use umpire_temporal_api::temporal::api::{
    common::v1::{Payload, Payloads},
    failure::v1::{ApplicationFailureInfo, Failure, failure::FailureInfo},
    workflowservice::v1::{DescribeActivityExecutionRequest, StartActivityExecutionRequest},
};
use umpire_temporal_api::umpire::temporal::v1::{ProbeRequest, probe_request::Operation};

#[path = "../../feature/activity/probe_bindings.rs"]
mod bindings;
mod dispositions;
mod generic;
mod report;

pub(crate) type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;
pub(crate) const UUID: &str = "12345678-1234-4234-8234-123456789abc";
fn cached<T>(cache: &OnceLock<T>, initialize: impl FnOnce() -> Result<T>) -> Result<&T> {
    if let Some(value) = cache.get() {
        return Ok(value);
    }
    let value = initialize()?;
    Ok(cache.get_or_init(|| value))
}

fn strict_registry() -> Result<&'static Registry> {
    static REGISTRY: OnceLock<Registry> = OnceLock::new();
    cached(&REGISTRY, build_inventory)
}
fn inventory() -> Result<&'static [Entry]> {
    Ok(strict_registry()?.entries())
}
fn build_inventory() -> Result<Registry> {
    let registry = bindings::registry()?;
    verify_methods(
        &registry
            .entries()
            .iter()
            .map(|entry| bare_method(&entry.metadata().method))
            .collect::<Result<BTreeSet<_>>>()?,
    )?;
    Ok(registry)
}
fn verify_methods(methods: &BTreeSet<&str>) -> Result<()> {
    if !methods.is_subset(&crate::activity_probe_fillers::methods()?)
        || !crate::activity_probe_fillers::baseline_methods()?.is_subset(methods)
    {
        return Err("probe inventory methods differ from registered baseline fillers".into());
    }
    Ok(())
}
fn bare_method(method: &str) -> Result<&str> {
    let name = method
        .strip_prefix("/temporal.api.workflowservice.v1.WorkflowService/")
        .ok_or("unknown request method")?;
    umpire_temporal_api::WorkflowServiceMethod::parse(name).ok_or("unknown request method")?;
    Ok(name)
}
struct ActivityStrict;
impl StrictSystem for ActivityStrict {
    type Slots = ProbeRequest;
    fn baseline(&self, method: &str, context: &str, slots: &ProbeRequest) -> Result<Vec<u8>> {
        baseline(bare_method(method)?, slots, context)
    }
    fn findings(&self, method: &str, bytes: &[u8]) -> Result<Vec<Violation>> {
        request_findings(bare_method(method)?, bytes)
    }
}
fn method(name: &str) -> String {
    format!("/temporal.api.workflowservice.v1.WorkflowService/{name}")
}
pub(crate) fn duration(seconds: i64) -> Duration {
    Duration { seconds, nanos: 0 }
}
pub(crate) fn application() -> Failure {
    Failure {
        failure_info: Some(FailureInfo::ApplicationFailureInfo(ApplicationFailureInfo {
            r#type: "ProbeFailure".into(),
            ..Default::default()
        })),
        ..Default::default()
    }
}

fn check_slots(name: &str, slots: &ProbeRequest) -> Result<()> {
    if slots.namespace.is_empty() || slots.activity_id.is_empty() || slots.task_queue.is_empty() {
        return Err("missing namespace/activity/queue prerequisite".into());
    }
    if matches!(
        name,
        "DescribeActivityExecution" | "PollActivityExecution" | "RequestCancelActivityExecution"
    ) && slots.run_id.is_empty()
    {
        return Err("missing run prerequisite".into());
    }
    if name.starts_with("RespondActivityTask") && slots.task_token.is_empty() {
        return Err("missing live task token prerequisite".into());
    }
    Ok(())
}

fn baseline(name: &str, slots: &ProbeRequest, context: &str) -> Result<Vec<u8>> {
    check_slots(name, slots)?;
    crate::activity_probe_fillers::fill(name, slots, context)
}

fn request_findings(name: &str, bytes: &[u8]) -> Result<Vec<Violation>> {
    macro_rules! supplemental {
        (RespondActivityTaskFailed, $message:ident, $limits:expr, $findings:ident) => {
            if let Some(failure) = &$message.failure {
                $findings.extend(validate(failure, &worker_failure_presence($limits)?));
            }
        };
        ($method:ident, $message:ident, $limits:expr, $findings:ident) => {};
    }
    let method =
        umpire_temporal_api::WorkflowServiceMethod::parse(name).ok_or("unknown request method")?;
    umpire_temporal_api::workflow_request_findings!(method, bytes, &Limits::default(), supplemental)
}

fn exceptional(entry: &Entry) -> bool {
    let target = entry.target();
    (target.violation.constraint == "required"
        && target
            .omitted_field
            .as_ref()
            .is_some_and(|p| p.starts_with("temporal.api.workflowservice.v1.")))
        || target.violation.path.ends_with(".namespace")
        || (entry.context() == "start"
            && target.violation.path
                == "temporal.api.workflowservice.v1.StartActivityExecutionRequest.input")
        || (entry.context() == "describe" && target.violation.constraint == "requires")
}
fn mutate_exception(bytes: &[u8], entry: &Entry) -> Result<Vec<u8>> {
    use umpire_catalog::WireMutation;
    use umpire_temporal_api::{WorkflowServiceMethod, mutate_workflow_request};
    let name = bare_method(&entry.metadata().method)?;
    let method = WorkflowServiceMethod::parse(name).ok_or("unknown request method")?;
    let target = entry.target();
    if let Some(path) = &target.omitted_field
        && target.violation.constraint == "required"
    {
        return Ok(mutate_workflow_request(method, bytes, path, &WireMutation::Absent)?);
    }
    if target.violation.path.ends_with(".namespace") {
        macro_rules! limit {
            ($($variant:ident => $factory:ident),* $(,)?) => { match method {
                $(WorkflowServiceMethod::$variant => catalog::$factory(&Limits::default())?.fields().find(|f| f.path == target.violation.path).cloned(),)*
                _ => None,
            }};
        }
        let field = limit! {
            StartActivityExecution => start_activity_execution_request,
            DescribeActivityExecution => describe_activity_execution_request,
            PollActivityExecution => poll_activity_execution_request,
            RequestCancelActivityExecution => request_cancel_activity_execution_request,
            PollActivityTaskQueue => poll_activity_task_queue_request,
            RespondActivityTaskCompleted => respond_activity_task_completed_request,
            RespondActivityTaskFailed => respond_activity_task_failed_request,
            RespondActivityTaskCanceled => respond_activity_task_canceled_request,
        };
        let Some(FieldDescriptor { kind: Kind::Text(rules), .. }) = field else {
            return Err("namespace kind missing".into());
        };
        let limit = rules.max_bytes.ok_or("namespace byte maximum missing")?;
        return Ok(mutate_workflow_request(
            method,
            bytes,
            &target.violation.path,
            &WireMutation::LimitPlusOne { limit: limit.try_into()? },
        )?);
    }
    if entry.context() == "describe" {
        let mut message = DescribeActivityExecutionRequest::decode(bytes)?;
        message.long_poll_token = vec![1];
        message.run_id.clear();
        return Ok(message.encode_to_vec());
    }
    let mut message = StartActivityExecutionRequest::decode(bytes)?;
    let catalog = catalog::start_activity_execution_request(&Limits::default())?;
    let field = catalog.fields().find(|f| f.path == target.violation.path);
    let Some(FieldDescriptor { kind: Kind::Bytes(rules), .. }) = field else {
        return Err("input kind missing".into());
    };
    let size = rules
        .max_len
        .ok_or("input maximum missing")?
        .checked_add(1)
        .filter(|s| *s <= 16 * 1024 * 1024)
        .ok_or("input witness exceeds probe budget")?;
    let data_size = size.checked_sub(10).ok_or("input bound too small for payload witness")?;
    let mut payloads =
        Payloads { payloads: vec![Payload { data: vec![b'x'; data_size], ..Default::default() }] };
    let padding = size.checked_sub(payloads.encoded_len()).ok_or("payload overhead")?;
    payloads.payloads[0].data.extend(std::iter::repeat_n(b'x', padding));
    if payloads.encoded_len() != size {
        return Err("payload exact size mismatch".into());
    }
    message.input = Some(payloads);
    Ok(message.encode_to_vec())
}
/// Enumerates, prepares and classifies activity probes independently of Model state.
///
/// # Errors
/// Refuses missing prerequisites, unsupported mutations, invalid baselines/witnesses and incomplete observations.
pub fn probe(request: &ProbeRequest) -> Result<ProbeResponse> {
    if request.encoded_len() > 1024 * 1024 - 1024
        || request.observations.len() > umpire_catalog::generic_probe::MAX_CASES
        || request.case_id.len() > 4096
        || request.observations.iter().any(|observation| {
            observation.case_id.len() > 4096
                || observation.predictions.len() > umpire_catalog::generic_probe::MAX_PREDICTIONS
                || observation.details_type.len() > 512
                || observation.refusal.len() > 256
        })
    {
        return Err("probe request exceeds metadata budget".into());
    }
    let operation = match Operation::try_from(request.operation)? {
        Operation::Inventory => umpire_catalog::generic_probe::Request::Inventory,
        Operation::Prepare => umpire_catalog::generic_probe::Request::Prepare {
            id: &request.case_id,
            baseline: request.baseline,
            context: request,
        },
        Operation::Classify => {
            umpire_catalog::generic_probe::Request::Classify(&request.observations)
        }
    };
    generic::engine()?.run(&operation)
}

fn prepare_strict(request: &ProbeRequest, entries: &[Entry]) -> Result<ProbeResponse> {
    let entry = entries.iter().find(|entry| entry.id() == request.case_id);
    let Some(entry) = entry.filter(|entry| !request.baseline && exceptional(entry)) else {
        return strict_registry()?.prepare(
            &ActivityStrict,
            &request.case_id,
            request.baseline,
            request,
            &mut Rng::new(217),
        );
    };
    let name = bare_method(&entry.metadata().method)?;
    let bytes = baseline(name, request, entry.context())?;
    if !request_findings(name, &bytes)?.is_empty() {
        return Err("invalid baseline".into());
    }
    let bytes = mutate_exception(&bytes, entry)?;
    let mut findings = request_findings(name, &bytes)?;
    let mut expected = entry.companions().to_vec();
    expected.push(entry.target().violation.clone());
    findings.sort_by_key(|f| (f.path.clone(), f.constraint.clone()));
    expected.sort_by_key(|f| (f.path.clone(), f.constraint.clone()));
    if findings != expected {
        return Err(format!(
            "{}: witness does not isolate declared target: {findings:?}",
            entry.id()
        )
        .into());
    }
    let mut response = ProbeResponse { method: method(name), ..Default::default() };
    if bytes.len() > 512 * 1024 {
        response.recipe = Some(recipe(&bytes)?);
    } else {
        response.message = bytes;
    }
    Ok(response)
}

#[cfg(test)]
mod tests {
    use super::{Result, cached, generic, inventory};
    use std::sync::{Barrier, OnceLock};

    #[test]
    fn inventory_method_set_must_equal_registered_baseline_fillers() -> Result<()> {
        use super::{bare_method, verify_methods};
        let current = inventory()?;
        let full = current
            .iter()
            .map(|entry| bare_method(&entry.metadata().method))
            .collect::<Result<std::collections::BTreeSet<_>>>()?;
        verify_methods(&full)?;
        let guard = "probe inventory methods differ from registered baseline fillers";
        let truncated =
            full.iter().copied().filter(|method| *method != "PollActivityExecution").collect();
        assert_eq!(verify_methods(&truncated).map_err(|e| e.to_string()), Err(guard.into()));
        let mut extra = full;
        extra.insert("UnregisteredMethod");
        assert_eq!(verify_methods(&extra).map_err(|e| e.to_string()), Err(guard.into()));
        Ok(())
    }

    #[test]
    fn contextual_witnesses_preserve_exact_duration_ascii_and_queue_policy() -> Result<()> {
        use super::{prepare_strict, request_findings};
        use prost::Message;
        use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;
        let entries = inventory()?;
        let slots = super::ProbeRequest {
            namespace: "namespace".into(),
            activity_id: "activity".into(),
            task_queue: "queue".into(),
            run_id: super::UUID.into(),
            task_token: vec![1],
            ..Default::default()
        };
        let leaf = |context: &str,
                    path: &str,
                    constraint: &str|
         -> Result<&umpire_catalog::contextual_probe::Entry> {
            entries
                .iter()
                .find(|entry| {
                    entry.context() == context
                        && entry.target().violation.path.ends_with(path)
                        && entry.target().violation.constraint == constraint
                })
                .ok_or_else(|| "missing exact contextual target".into())
        };
        for (constraint, seconds, nanos) in [("duration.valid", 1, -1), ("duration.gte", -1, 0)] {
            let entry = leaf("start", ".heartbeat_timeout", constraint)?;
            let response = prepare_strict(
                &super::ProbeRequest { case_id: entry.id().into(), ..slots.clone() },
                entries,
            )?;
            let actual = StartActivityExecutionRequest::decode(response.message.as_slice())?;
            assert_eq!(actual.heartbeat_timeout, Some(super::Duration { seconds, nanos }));
            assert_eq!(
                request_findings("StartActivityExecution", &response.message)?,
                vec![entry.target().violation.clone()]
            );
        }
        let entry = leaf("start.type", ".name", "string.max_bytes")?;
        let response = prepare_strict(
            &super::ProbeRequest { case_id: entry.id().into(), ..slots.clone() },
            entries,
        )?;
        let actual = StartActivityExecutionRequest::decode(response.message.as_slice())?;
        assert_eq!(actual.activity_type.ok_or("missing type")?.name, "x".repeat(1001));
        let entry = leaf("start.queue", ".name", "string.max_bytes")?;
        let response = prepare_strict(
            &super::ProbeRequest { case_id: entry.id().into(), ..slots.clone() },
            entries,
        )?;
        let actual = StartActivityExecutionRequest::decode(response.message.as_slice())?;
        let baseline = super::baseline("StartActivityExecution", &slots, entry.context())?;
        let baseline = StartActivityExecutionRequest::decode(baseline.as_slice())?;
        let expected = super::catalog::task_queue(&super::Limits::default())?.probe(
            baseline.task_queue.as_ref().ok_or("missing queue")?,
            entry.target(),
            &mut umpire_catalog::Rng::new(217),
        )?;
        assert_eq!(actual.task_queue, Some(expected.message));
        Ok(())
    }

    #[test]
    fn generated_findings_preserve_contextual_intents_and_recursive_checks() -> Result<()> {
        use super::{
            Failure, FailureInfo, ProbeRequest, StartActivityExecutionRequest, UUID, application,
            baseline, duration, request_findings,
        };
        use prost::Message;
        use umpire_temporal_api::temporal::api::{
            failure::v1::TimeoutFailureInfo,
            workflowservice::v1::{
                RequestCancelActivityExecutionRequest, RespondActivityTaskFailedRequest,
            },
        };
        let slots = ProbeRequest {
            namespace: "namespace".into(),
            activity_id: "activity".into(),
            task_queue: "queue".into(),
            run_id: UUID.into(),
            task_token: vec![1],
            ..Default::default()
        };
        for method in ["StartActivityExecution", "RequestCancelActivityExecution"] {
            let bytes = baseline(method, &slots, "")?;
            let bytes = if method == "StartActivityExecution" {
                let mut request = StartActivityExecutionRequest::decode(bytes.as_slice())?;
                request.request_id = "invalid-uuid".into();
                request.encode_to_vec()
            } else {
                let mut request = RequestCancelActivityExecutionRequest::decode(bytes.as_slice())?;
                request.request_id = "invalid-uuid".into();
                request.encode_to_vec()
            };
            let findings = request_findings(method, &bytes)?;
            assert_eq!(findings.len(), 1, "{method}: {findings:?}");
            assert_eq!(findings[0].constraint, "string.uuid");
        }
        let mut request = RespondActivityTaskFailedRequest::decode(
            baseline("RespondActivityTaskFailed", &slots, "")?.as_slice(),
        )?;
        let mut failure = application();
        failure.cause = Some(Box::new(Failure {
            failure_info: Some(FailureInfo::TimeoutFailureInfo(TimeoutFailureInfo {
                timeout_type: i32::MAX,
                ..Default::default()
            })),
            ..Default::default()
        }));
        request.failure.as_mut().ok_or("missing failure")?.cause = Some(Box::new(failure));
        let findings = request_findings("RespondActivityTaskFailed", &request.encode_to_vec())?;
        assert_eq!(findings.len(), 1, "{findings:?}");
        assert!(findings[0].path.ends_with("TimeoutFailureInfo.timeout_type"));
        request.failure.as_mut().ok_or("missing failure")?.failure_info = None;
        let findings = request_findings("RespondActivityTaskFailed", &request.encode_to_vec())?;
        assert_eq!(findings.len(), 2, "{findings:?}");
        assert!(findings.iter().any(|finding| finding.constraint == "required"));
        let mut request = StartActivityExecutionRequest::decode(
            baseline("StartActivityExecution", &slots, "")?.as_slice(),
        )?;
        let retry = request.retry_policy.as_mut().ok_or("missing retry")?;
        retry.maximum_attempts = 1;
        retry.initial_interval = Some(duration(-1));
        retry.maximum_interval = Some(duration(-1));
        retry.backoff_coefficient = 0.5;
        assert_eq!(request_findings("StartActivityExecution", &request.encode_to_vec())?, []);
        request.retry_policy.as_mut().ok_or("missing retry")?.maximum_attempts = 2;
        assert_eq!(request_findings("StartActivityExecution", &request.encode_to_vec())?.len(), 3);
        for (method, bytes) in [
            ("unknown", &[][..]),
            ("GetSystemInfo", &[][..]),
            ("StartActivityExecution", &[0xff][..]),
        ] {
            assert!(request_findings(method, bytes).is_err());
        }
        Ok(())
    }

    #[test]
    fn validated_inventories_are_reused() -> Result<()> {
        let contextual = inventory()?;
        let contextual_again = inventory()?;
        let generic = generic::engine()?;
        let generic_again = generic::engine()?;
        assert!(!contextual.is_empty());
        assert!(
            generic
                .run(&umpire_catalog::generic_probe::Request::Inventory)?
                .cases
                .iter()
                .any(|case| case.generic)
        );
        assert_eq!(
            (contextual.as_ptr(), std::ptr::from_ref(generic)),
            (contextual_again.as_ptr(), std::ptr::from_ref(generic_again)),
            "successfully validated metadata must not be rebuilt for each probe"
        );
        Ok(())
    }

    #[test]
    fn failed_or_panicked_initialization_publishes_nothing() -> Result<()> {
        let cache = OnceLock::new();
        let failed = cached(&cache, || -> Result<Vec<u8>> {
            let mut partial = vec![1];
            partial.push(2);
            Err("metadata validation refused".into())
        });
        assert_eq!(
            failed.err().ok_or("missing initialization error")?.to_string(),
            "metadata validation refused"
        );
        assert!(cache.get().is_none());
        let panicked = std::panic::catch_unwind(|| {
            cached(&cache, || -> Result<Vec<u8>> {
                std::panic::resume_unwind(Box::new("initializer panic"))
            })
        });
        assert!(panicked.is_err());
        assert!(cache.get().is_none());
        assert_eq!(cached(&cache, || Ok(vec![3]))?, &[3]);
        assert_eq!(cached(&cache, || Err("must not rerun".into()))?, &[3]);
        Ok(())
    }

    #[test]
    fn concurrent_initialization_publishes_one_complete_inventory() -> Result<()> {
        let cache = OnceLock::new();
        let barrier = Barrier::new(4);
        std::thread::scope(|scope| -> Result<()> {
            let handles: Vec<_> = (0..4)
                .map(|_| {
                    scope.spawn(|| {
                        cached(&cache, || {
                            let entries = super::build_inventory()?;
                            barrier.wait();
                            Ok(entries)
                        })
                        .map(|entries| entries.entries().as_ptr() as usize)
                        .map_err(|error| error.to_string())
                    })
                })
                .collect();
            let addresses = handles
                .into_iter()
                .map(|handle| {
                    handle
                        .join()
                        .map_err(|_| "inventory initialization thread panicked")?
                        .map_err(Into::into)
                })
                .collect::<Result<Vec<_>>>()?;
            assert!(addresses.iter().all(|address| *address == addresses[0]));
            assert!(!cache.get().ok_or("missing complete inventory")?.entries().is_empty());
            Ok(())
        })
    }
}

#[cfg(test)]
#[path = "../tests/entity_lens/probe_baseline.rs"]
mod baseline_equivalence;
