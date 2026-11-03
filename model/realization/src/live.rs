use prost::Message;
use umpire_catalog::{Catalog, validate};
use umpire_dsl::Machine;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    Dispatch, NoTarget, Projection, Proposal, Realize, Runtime,
    prost_types::Any,
    protocol::{self, Refusal, RefusalCode, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal_api::temporal::api::{
    common::v1::{Header, Payload, link},
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        StartActivityExecutionRequest, StartActivityExecutionResponse,
    },
};

use crate::{Realization, activity, activity_api::catalog, foundation::dc::Limits};

const START: &str = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution";
const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";
const CANCEL: &str =
    "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution";
type ModelRuntime = Runtime<activity::World, LiveRealization>;
type Action = <activity::World as Machine>::Action;

#[derive(Clone)]
/// Immutable activity bootstrap evidence retained by the migration adapter.
pub struct Receipt {
    request: Any,
    response: Any,
    namespace: String,
    activity_id: String,
    request_id: String,
    run_id: String,
    maximum_attempts: Option<u8>,
}

/// Activity projections around the existing independent Model runtime.
pub struct LiveRealization {
    inner: Realization,
    receipt: Option<Receipt>,
}

impl umpire_runtime::capability::Identity for LiveRealization {
    type Id = <Realization as umpire_runtime::capability::Identity>::Id;
    fn identify(&self, value: &Any) -> Result<Self::Id, Refusal> {
        umpire_runtime::capability::Identity::identify(&self.inner, value)
    }
}
impl umpire_runtime::capability::Fill<Action> for LiveRealization {
    type World = activity::World;
    type Inputs = <Realization as umpire_runtime::capability::Fill<Action>>::Inputs;
    type Request = <Realization as umpire_runtime::capability::Fill<Action>>::Request;
    fn fill(
        &self,
        action: &Action,
        world: &activity::World,
        inputs: &Self::Inputs,
        identity: &Self::Id,
    ) -> Result<Self::Request, Refusal> {
        umpire_runtime::capability::Fill::fill(&self.inner, action, world, inputs, identity)
    }
}
impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for LiveRealization {
    type World = activity::World;
    type View = crate::support::Observed;
    fn admit_evidence(&self, actual: &Any) -> Result<Self::View, Refusal> {
        crate::support::Observed::new(actual)
    }
}
impl umpire_runtime::capability::ReplyPolicy<activity::World> for LiveRealization {
    fn program(&self) -> Result<Any, Refusal> {
        Realize::program(&self.inner)
    }

    fn propose(
        &self,
        world: &activity::World,
        current: &activity::Model,
        seed: u64,
    ) -> Result<Proposal<Action>, Refusal> {
        Realize::propose(&self.inner, world, current, seed)
    }

    fn matches(&self, request: &Any, observation: &Any) -> Result<(), Refusal> {
        Realize::matches(&self.inner, request, observation)
    }

    fn continue_polling(
        &self,
        observation: &Any,
        current: &activity::Model,
        action: Option<&Action>,
    ) -> Result<bool, Refusal> {
        Realize::continue_polling(&self.inner, observation, current, action)
    }
}

impl umpire_runtime::capability::Snapshot<activity::World> for LiveRealization {
    type Target = NoTarget;
    fn project(
        &self,
        world: &activity::World,
        actual: &Any,
        current: &activity::Model,
    ) -> Result<Projection<Action, umpire_dsl::Outcome, NoTarget>, Refusal> {
        let observed = crate::support::Observed::new(actual)?;
        if observed.method != START {
            return Realize::observe(&self.inner, world, actual, current);
        }
        let observation: Observation = actual.unpack()?;
        let raw_request = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("Start actual request is missing"))?;
        let request: StartActivityExecutionRequest = raw_request.unpack()?;
        let Some(ResultValue::Response(raw_response)) = &observation.result else {
            return Err(Refusal::malformed("successful Start response is required"));
        };
        let response: StartActivityExecutionResponse = raw_response.unpack()?;
        validate_start(&request, &response)?;
        self.receipt
            .as_ref()
            .ok_or_else(|| state("live Start admission is required"))?
            .validate_replay(&request, &response, raw_request, raw_response)?;
        Ok(Projection::Current)
    }
}

impl Realize<activity::World> for LiveRealization {}

fn fresh_model() -> ModelRuntime {
    Runtime::new(
        activity::World::new(3, None, None, None, None, false),
        LiveRealization { inner: Realization::default(), receipt: None },
    )
}

use crate::lens::{Admission, EntityBinding, EntityLens};

/// Activity policy adapter retained while the feature declaration migrates.
pub struct ActivityEntity;

/// Activity policy composed with the shared entity lifecycle.
pub(crate) type LiveRuntime = EntityLens<ActivityEntity>;

pub(super) fn runtime() -> LiveRuntime {
    EntityLens::new(ActivityEntity)
}

impl EntityBinding for ActivityEntity {
    type Model = ModelRuntime;
    type Receipt = Receipt;
    const BOOTSTRAP: &'static str = START;
    fn fresh(&self) -> ModelRuntime {
        fresh_model()
    }
    fn configure(&self, configuration: &Any) -> Result<Any, Refusal> {
        let start: StartActivityExecutionRequest = configuration.unpack()?;
        let maximum = start.retry_policy.as_ref().map_or(0, |policy| policy.maximum_attempts);
        let maximum = u8::try_from(maximum).map_err(|_| {
            Refusal::unsupported("retry_policy.maximum_attempts must be explicitly in 1..=3")
        })?;
        let world = activity::World::new(maximum, None, None, None, None, false);
        Realization::worker_program(&start, &world)
    }
    fn bootstrap(&self, configuration: &Any, seed: u64) -> Result<Any, Refusal> {
        let start: StartActivityExecutionRequest = configuration.unpack()?;
        let filled = crate::activity_api::annotate(START, false, &start.encode_to_vec(), seed)?;
        if filled.out_of_scope.is_some() || !filled.violations.is_empty() {
            return Err(Refusal::malformed("configured Start violates its catalog"));
        }
        let mut start = StartActivityExecutionRequest::decode(filled.bytes.as_slice())
            .map_err(|_| Refusal::malformed("filled Start is malformed"))?;
        let scenario = crate::activity_realization::SCENARIOS[(seed % 3) as usize];
        start.header.get_or_insert_with(Header::default).fields.insert(
            crate::activity_realization::SCENARIO_HEADER.into(),
            Payload {
                metadata: std::collections::BTreeMap::from([(
                    "encoding".into(),
                    b"binary/plain".to_vec(),
                )]),
                data: scenario.as_bytes().to_vec(),
                ..Default::default()
            },
        );
        Ok(Any::pack(&start))
    }
    fn bootstrap_request(&self, actual: &Any) -> Result<Any, Refusal> {
        let observation: Observation = actual.unpack()?;
        let raw = observation
            .request
            .ok_or_else(|| Refusal::malformed("Start actual request is missing"))?;
        let _: StartActivityExecutionRequest = raw.unpack()?;
        Ok(raw)
    }
    fn admit(&self, actual: &Any) -> Result<Admission<Self>, Refusal> {
        let observation: Observation = actual.unpack()?;
        let raw_request = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("Start actual request is missing"))?;
        let request: StartActivityExecutionRequest = raw_request.unpack()?;
        let Some(ResultValue::Response(raw_response)) = &observation.result else {
            return Err(Refusal::malformed("successful Start response is required"));
        };
        let response: StartActivityExecutionResponse = raw_response.unpack()?;
        validate_start(&request, &response)?;
        if !response.started {
            return Err(state("initial Start must create a new run"));
        }
        let attempts = request.retry_policy.as_ref().map(|policy| policy.maximum_attempts);
        let maximum_attempts = attempts
            .filter(|attempts| (1..=3).contains(attempts))
            .and_then(|attempts| u8::try_from(attempts).ok());
        let receipt = Receipt {
            request: raw_request.clone(),
            response: raw_response.clone(),
            namespace: request.namespace,
            activity_id: request.activity_id,
            request_id: request.request_id,
            run_id: response.run_id,
            maximum_attempts,
        };
        let Some(maximum_attempts) = maximum_attempts else {
            let refusal =
                Refusal::unsupported("retry_policy.maximum_attempts must be explicitly in 1..=3");
            return Ok(Admission::Frozen { receipt, refusal });
        };
        let world = activity::World::new(maximum_attempts, None, None, None, None, false);
        let starts = world.starts();
        let [initial] = starts.as_slice() else {
            return Err(state("admitted World requires exactly one start"));
        };
        let mut model = Runtime::with_initial(
            world,
            LiveRealization { inner: Realization::default(), receipt: Some(receipt.clone()) },
            initial.clone(),
        )?;
        let (response, transaction) = model.prepare(
            protocol::Method::Judge as u32,
            &protocol::JudgeRequest { observation: Some(actual.clone()) }.encode_to_vec(),
        );
        let outcome = response.outcome.ok_or_else(|| state("Start judgment is missing"))?;
        if let Outcome::Refusal(refusal) = outcome {
            return Err(refusal);
        }
        if let Some(transaction) = transaction {
            model.commit(transaction);
        }
        Ok(Admission::Admitted { model, receipt, outcome })
    }
    fn replay(&self, receipt: &Receipt, actual: &Any) -> Result<(), Refusal> {
        let observation: Observation = actual.unpack()?;
        let raw_request = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("Start actual request is missing"))?;
        let request: StartActivityExecutionRequest = raw_request.unpack()?;
        let Some(ResultValue::Response(raw_response)) = &observation.result else {
            return Err(Refusal::malformed("successful Start response is required"));
        };
        let response: StartActivityExecutionResponse = raw_response.unpack()?;
        validate_start(&request, &response)?;
        receipt.validate_replay(&request, &response, raw_request, raw_response)
    }
    fn validate(
        &self,
        receipt: Option<&Receipt>,
        observed: &crate::support::Observed,
    ) -> Result<(), Refusal> {
        let observation = observed.current();
        match observation.method.as_str() {
            DESCRIBE => {
                let response: DescribeActivityExecutionResponse = observed.response()?;
                let info =
                    response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?;
                let _ = activity::Phase::try_from(info.clone())?;
                let request: DescribeActivityExecutionRequest = required_request(observation)?;
                let receipt = receipt.ok_or_else(|| self.missing_receipt())?;
                receipt.matches(&request.namespace, &request.activity_id, &request.run_id)?;
                receipt.matches(&request.namespace, &info.activity_id, &info.run_id)?;
                receipt.matches(&request.namespace, &request.activity_id, &response.run_id)?;
                if info.attempt < 1
                    || receipt
                        .maximum_attempts
                        .is_some_and(|maximum| info.attempt > i32::from(maximum))
                {
                    return Err(state("Describe attempt is outside the admitted attempts"));
                }
            }
            CANCEL => {
                if observed.code()?.is_none() {
                    let _: RequestCancelActivityExecutionResponse =
                        observed.response().map_err(|error| Refusal::malformed(error.message))?;
                }
                let request: RequestCancelActivityExecutionRequest = required_request(observation)?;
                receipt.ok_or_else(|| self.missing_receipt())?.matches(
                    &request.namespace,
                    &request.activity_id,
                    &request.run_id,
                )?;
            }
            _ => return Err(Refusal::unsupported("observation RPC is not supported")),
        }
        Ok(())
    }
    fn ready(&self, receipt: &Receipt) -> Result<(), Refusal> {
        if receipt.maximum_attempts.is_none() {
            return Err(state("live Start was not admitted"));
        }
        Ok(())
    }
    fn ready_replay(&self, receipt: &Receipt) -> Result<(), Refusal> {
        if receipt.maximum_attempts.is_none() {
            return Err(state("rejected Start requires a fresh guest"));
        }
        Ok(())
    }
    fn install(&self, model: &mut ModelRuntime, receipt: &Receipt) {
        let request: StartActivityExecutionRequest =
            receipt.request.unpack().unwrap_or_else(|_| unreachable!("validated Start receipt"));
        model.replace_realization(LiveRealization {
            inner: Realization::configured(crate::activity_realization::ProposalContext {
                start: request,
                run_id: receipt.run_id.clone(),
            }),
            receipt: Some(receipt.clone()),
        });
    }
    fn instruction(&self, request: &Any) -> bool {
        request.type_url == <umpire_temporal_api::umpire::temporal::v1::TemporalInstruction as prost::Name>::type_url()
    }
    fn missing_receipt(&self) -> Refusal {
        state("live Start admission is required")
    }
}

impl Receipt {
    fn validate_replay(
        &self,
        request: &StartActivityExecutionRequest,
        response: &StartActivityExecutionResponse,
        raw_request: &Any,
        raw_response: &Any,
    ) -> Result<(), Refusal> {
        self.matches(&request.namespace, &request.activity_id, &response.run_id)?;
        if self.request_id != request.request_id || self.request != *raw_request {
            return Err(state("Start request contradicts the original receipt"));
        }
        if response.started && self.response != *raw_response {
            return Err(state("Start response contradicts the original receipt"));
        }
        if !response.started && self.maximum_attempts.is_none() {
            return Err(state("Start replay has no original admitted provenance"));
        }
        Ok(())
    }

    fn matches(&self, namespace: &str, activity: &str, run: &str) -> Result<(), Refusal> {
        for (matches, component) in [
            (!namespace.is_empty() && namespace == self.namespace, "namespace"),
            (!activity.is_empty() && activity == self.activity_id, "activity_id"),
            (!run.is_empty() && run == self.run_id, "run_id"),
        ] {
            if !matches {
                return Err(state(format!("evidence {component} does not match Start receipt")));
            }
        }
        Ok(())
    }
}

fn required_request<T: prost::Name + Default>(observation: &Observation) -> Result<T, Refusal> {
    observation
        .request
        .as_ref()
        .ok_or_else(|| Refusal::malformed("actual request is missing"))?
        .unpack()
}
fn state(message: impl Into<String>) -> Refusal {
    Refusal::new(RefusalCode::State, message)
}
fn checked<T: 'static>(
    value: &T,
    catalog: Result<Catalog<T>, umpire_catalog::AdmissionErrors>,
) -> Result<(), Refusal> {
    let catalog =
        catalog.map_err(|_| Refusal::unsupported("Start validation catalog admission failed"))?;
    if validate(value, &catalog).is_empty() {
        Ok(())
    } else {
        Err(Refusal::malformed("Start captured evidence violates its catalog"))
    }
}
fn validate_start(
    request: &StartActivityExecutionRequest,
    response: &StartActivityExecutionResponse,
) -> Result<(), Refusal> {
    for (present, field) in [
        (!request.namespace.is_empty(), "namespace"),
        (!request.activity_id.is_empty(), "activity_id"),
        (!request.request_id.is_empty(), "request_id"),
        (!response.run_id.is_empty(), "run_id"),
    ] {
        if !present {
            return Err(Refusal::malformed(format!("Start {field} is missing")));
        }
    }
    let limits = Limits::default();
    checked(request, catalog::start_activity_execution_request(&limits))?;
    checked(response, catalog::start_activity_execution_response(&limits))?;
    if let Some(value) = &request.activity_type {
        checked(value, catalog::activity_type(&limits))?;
    }
    if let Some(value) = &request.task_queue {
        checked(value, catalog::task_queue(&limits))?;
    }
    if let Some(value) = &request.retry_policy {
        let policy = catalog::retry_policy(&limits)
            .map_err(|_| Refusal::unsupported("Start validation catalog admission failed"))?;
        if validate(value, &policy)
            .iter()
            .any(|finding| !finding.path.ends_with(".maximum_attempts"))
        {
            return Err(Refusal::malformed("Start retry policy violates its catalog"));
        }
    }
    if let Some(link) = &response.link {
        let Some(link::Variant::Activity(link)) = &link.variant else {
            return Err(Refusal::malformed("Start response Link is not an activity"));
        };
        if link.namespace != request.namespace
            || link.activity_id != request.activity_id
            || link.run_id != response.run_id
        {
            return Err(state("Start response Link contradicts captured identity"));
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/entity_lens/mod.rs"]
mod equivalence;
