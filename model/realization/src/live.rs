use prost::Message;
use umpire_dsl::Machine;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    Dispatch, Projection, Proposal, Realize, Runtime,
    prost_types::Any,
    protocol::{self, Refusal, RefusalCode, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal_api::temporal::api::{
    common::v1::{Header, Payload},
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        StartActivityExecutionRequest, StartActivityExecutionResponse,
    },
};

use crate::{ActivityTarget, Realization, activity, activity_realization::ProposalContext};

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
    entity: ProposalContext,
    manifest: Option<std::rc::Rc<crate::policy::Manifest>>,
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
    fn check_evidence(
        &self,
        view: &Self::View,
        path: &[&umpire_runtime::Evidence<activity::World>],
    ) -> umpire_runtime::EvidenceMatch {
        umpire_runtime::capability::Evidence::check_evidence(&self.inner, view, path)
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
    type Target = ActivityTarget;
    fn project(
        &self,
        world: &activity::World,
        actual: &Any,
        current: &activity::Model,
    ) -> Result<Projection<Action, umpire_dsl::Outcome, ActivityTarget>, Refusal> {
        let observed = crate::support::Observed::new(actual)?;
        if observed.method == CANCEL && current.phase == activity::Phase::Started {
            // The cooperative Model records a pending request here; live evidence of its
            // heartbeat report and acknowledgement is not bound yet.
            return Err(Refusal::unsupported(
                "cancel_requested: live admission cannot evidence a pending cancellation of a Started attempt",
            ));
        }
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
        activity::World::new(3, None, None, None, None),
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
        crate::policy::Manifest::production(&start)?.issued(&start)
    }
    fn bootstrap(&self, configuration: &Any, seed: u64) -> Result<Any, Refusal> {
        let start: StartActivityExecutionRequest = configuration.unpack()?;
        let filled = crate::activity_api::annotate(START, false, &start.encode_to_vec(), seed)?;
        if filled.out_of_scope.is_some() || !filled.violations.is_empty() {
            return Err(Refusal::malformed("configured Start violates its catalog"));
        }
        let mut start = StartActivityExecutionRequest::decode(filled.bytes.as_slice())
            .map_err(|_| Refusal::malformed("filled Start is malformed"))?;
        let manifest = crate::policy::Manifest::production(&start)?;
        let scenario = manifest.selected(seed)?.name();
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
        let (binding, entity) = (Realization::default(), Realization::entity());
        let identity = (entity.admit)(&binding, &request, &response)?;
        let mut receipt = Receipt {
            request: raw_request.clone(),
            response: raw_response.clone(),
            entity: (entity.receipt)(&binding, &identity, &request, &response)?,
            manifest: None,
        };
        let Some(_) = receipt.entity.maximum_attempts() else {
            let refusal =
                Refusal::unsupported("retry_policy.maximum_attempts must be explicitly in 1..=3");
            return Ok(Admission::Frozen { receipt, refusal });
        };
        let manifest = std::rc::Rc::new(crate::policy::Manifest::production(&request)?);
        let _ = manifest.issued(&request)?;
        let selected = request
            .header
            .as_ref()
            .and_then(|header| header.fields.get(crate::SCENARIO_HEADER))
            .and_then(|payload| std::str::from_utf8(&payload.data).ok())
            .ok_or_else(|| Refusal::malformed("Start canonical policy header is missing"))?;
        manifest.admit_name(selected)?;
        let world = manifest.world().clone();
        receipt.manifest = Some(manifest);
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
                let info = response
                    .info
                    .clone()
                    .ok_or_else(|| Refusal::malformed("activity info is missing"))?;
                let _ = activity::Phase::try_from(info)?;
                let request: DescribeActivityExecutionRequest = required_request(observation)?;
                let receipt = receipt.ok_or_else(|| self.missing_receipt())?;
                (Realization::read().correlate)(
                    &Realization::default(),
                    &receipt.entity,
                    &request,
                    &response,
                )?;
            }
            CANCEL => {
                if observed.code()?.is_none() {
                    let _: RequestCancelActivityExecutionResponse =
                        observed.response().map_err(|error| Refusal::malformed(error.message))?;
                }
                let request: RequestCancelActivityExecutionRequest = required_request(observation)?;
                receipt.ok_or_else(|| self.missing_receipt())?.entity.matches(
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
        if receipt.entity.maximum_attempts().is_none() || receipt.manifest.is_none() {
            return Err(state("live Start was not admitted"));
        }
        Ok(())
    }
    fn ready_replay(&self, receipt: &Receipt) -> Result<(), Refusal> {
        if receipt.entity.maximum_attempts().is_none() || receipt.manifest.is_none() {
            return Err(state("rejected Start requires a fresh guest"));
        }
        Ok(())
    }
    fn install(&self, model: &mut ModelRuntime, receipt: &Receipt) {
        model.replace_realization(LiveRealization {
            inner: Realization::configured(receipt.entity.clone(), receipt.manifest.clone()),
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
        (Realization::entity().correlate)(
            &Realization::default(),
            &self.entity,
            request,
            response,
        )?;
        if self.request != *raw_request {
            return Err(state("Start request contradicts the original receipt"));
        }
        if response.started && self.response != *raw_response {
            return Err(state("Start response contradicts the original receipt"));
        }
        if !response.started && self.entity.maximum_attempts().is_none() {
            return Err(state("Start replay has no original admitted provenance"));
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
#[cfg(test)]
#[path = "../tests/entity_lens/mod.rs"]
mod equivalence;
