use prost::Message;
use umpire_catalog::{Catalog, validate};
use umpire_dsl::Machine;
use umpire_runtime::{
    Dispatch, Runtime, Transaction,
    google::protobuf::Any,
    protocol::{self, Refusal, RefusalCode, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal_api::temporal::api::{
    common::v1::link,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityTaskQueueResponse, RequestCancelActivityExecutionRequest,
        RequestCancelActivityExecutionResponse, StartActivityExecutionRequest,
        StartActivityExecutionResponse,
    },
};

use crate::{Realization, activity, activity_api::catalog, foundation::dc::Limits, runtime};

const START: &str = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution";
const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";
const CANCEL: &str =
    "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution";
type ModelRuntime = Runtime<activity::World, Realization>;
type Prepared = Result<(Outcome, Option<LiveTransaction>), Refusal>;

struct Receipt {
    request: Any,
    response: Any,
    namespace: String,
    activity_id: String,
    request_id: String,
    run_id: String,
    maximum_attempts: Option<u8>,
}

/// Temporal-owned live realization around an independently owned Model runtime.
pub struct LiveRuntime {
    model: ModelRuntime,
    receipt: Option<Receipt>,
    frozen: Option<Refusal>,
    configuration: Option<Any>,
    bootstrap: Option<Any>,
    callback: Option<(PollActivityTaskQueueResponse, Any)>,
}

/// Owned facade effects, installed only after the complete response is published.
pub struct LiveTransaction {
    change: Change,
}

enum Change {
    Model(Transaction<activity::World>),
    Admit {
        model: Box<ModelRuntime>,
        receipt: Receipt,
    },
    Freeze {
        receipt: Receipt,
        refusal: Refusal,
    },
    Reset(Transaction<activity::World>),
    Configure(Any),
    Propose(Any),
    Answer {
        transaction: Transaction<activity::World>,
        task: Box<PollActivityTaskQueueResponse>,
        response: Any,
    },
    Observe {
        transaction: Transaction<activity::World>,
        clear_callback: bool,
    },
}

impl LiveRuntime {
    pub(super) fn new() -> Self {
        Self {
            model: runtime(),
            receipt: None,
            frozen: None,
            configuration: None,
            bootstrap: None,
            callback: None,
        }
    }

    fn prepare_method(&self, method: u32, bytes: &[u8]) -> Prepared {
        match method {
            3 => {
                let request = protocol::ResetRequest::decode(bytes)
                    .map_err(|_| Refusal::malformed("malformed Reset request"))?;
                let configuration = request.configuration.or_else(|| self.configuration.clone());
                if let Some(configuration) = configuration {
                    let _: StartActivityExecutionRequest = configuration.unpack()?;
                    return Ok((
                        Outcome::Reset(protocol::ResetResponse {}),
                        Some(LiveTransaction { change: Change::Configure(configuration) }),
                    ));
                }
                let (response, transaction) = self.model.prepare(method, bytes);
                Ok((
                    response.outcome.ok_or_else(|| state("Reset response is missing"))?,
                    transaction
                        .map(|transaction| LiveTransaction { change: Change::Reset(transaction) }),
                ))
            }
            4 => self.judge(bytes),
            5 => self.answer(bytes),
            6 => self.propose(bytes),
            _ => {
                let (response, transaction) = self.model.prepare(method, bytes);
                Ok((
                    response.outcome.ok_or_else(|| state("Model response is missing"))?,
                    transaction
                        .map(|transaction| LiveTransaction { change: Change::Model(transaction) }),
                ))
            }
        }
    }

    fn judge(&self, bytes: &[u8]) -> Prepared {
        let request = protocol::JudgeRequest::decode(bytes)
            .map_err(|_| Refusal::malformed("malformed Judge request"))?;
        let value = Any::required(request.observation)?;
        let observed = umpire_runtime::Observed::new(&value)?;
        if observed.method == START {
            return self.start(&value);
        }
        let observation: Observation = value.unpack()?;
        let mut clear_callback = false;
        match observation.method.as_str() {
            DESCRIBE => {
                let response: DescribeActivityExecutionResponse = observed.response()?;
                let info =
                    response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?;
                let phase = activity::Phase::try_from(info.clone())?;
                clear_callback = phase != activity::Phase::Started;
                let request: DescribeActivityExecutionRequest = required_request(&observation)?;
                let receipt = self.receipt()?;
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
                    clear_callback = true;
                    let _: RequestCancelActivityExecutionResponse =
                        observed.response().map_err(|error| Refusal::malformed(error.message))?;
                }
                let request: RequestCancelActivityExecutionRequest =
                    required_request(&observation)?;
                self.receipt()?.matches(
                    &request.namespace,
                    &request.activity_id,
                    &request.run_id,
                )?;
            }
            _ => return Err(Refusal::unsupported("observation RPC is not supported")),
        }
        let receipt = self.receipt()?;
        let prepared = self.delegate(4, bytes)?;
        if let Outcome::Refusal(refusal) = &prepared.0 {
            return Err(refusal.clone());
        }
        if let Some(frozen) = &self.frozen {
            return Err(frozen.clone());
        }
        if receipt.maximum_attempts.is_none() {
            return Err(state("live Start was not admitted"));
        }
        if self.configuration.is_some() {
            let transaction = match prepared.1 {
                Some(LiveTransaction { change: Change::Model(transaction) }) => {
                    Some(LiveTransaction {
                        change: Change::Observe { transaction, clear_callback },
                    })
                }
                other => other,
            };
            return Ok((prepared.0, transaction));
        }
        Ok(prepared)
    }

    fn propose(&self, bytes: &[u8]) -> Prepared {
        let request = protocol::ProposeRequest::decode(bytes)
            .map_err(|_| Refusal::malformed("malformed Propose request"))?;
        let configuration = self
            .configuration
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        if self.frozen.is_some() || self.bootstrap.is_some() {
            return Err(state("admission frozen or bootstrap pending until Reset or final Judge"));
        }
        if self.receipt.is_some() {
            return self.delegate(6, bytes);
        }
        let start: StartActivityExecutionRequest = configuration.unpack()?;
        let filled =
            crate::activity_api::annotate(START, false, &start.encode_to_vec(), request.seed)?;
        if filled.out_of_scope.is_some() || !filled.violations.is_empty() {
            return Err(Refusal::malformed("configured Start violates its catalog"));
        }
        let start = StartActivityExecutionRequest::decode(filled.bytes.as_slice())
            .map_err(|_| Refusal::malformed("filled Start is malformed"))?;
        let request = Any::pack(&start);
        Ok((
            Outcome::Propose(protocol::ProposeResponse {
                request: Some(request.clone()),
                action: String::new(),
                kind: protocol::ProposalKind::Bootstrap as i32,
            }),
            Some(LiveTransaction { change: Change::Propose(request) }),
        ))
    }

    fn receipt(&self) -> Result<&Receipt, Refusal> {
        self.receipt.as_ref().ok_or_else(|| state("live Start admission is required"))
    }

    fn answer(&self, bytes: &[u8]) -> Prepared {
        let request = protocol::AnswerRequest::decode(bytes)
            .map_err(|_| Refusal::malformed("malformed Answer request"))?;
        let value = Any::required(request.callback)?;
        let task: PollActivityTaskQueueResponse = umpire_runtime::callback(&value)?;
        let receipt = self.receipt()?;
        receipt.matches(&task.workflow_namespace, &task.activity_id, &task.activity_run_id)?;
        if self.frozen.is_some() {
            return Err(state("admission is frozen until Reset"));
        }
        if receipt.maximum_attempts.is_none() {
            return Err(state("live Start was not admitted"));
        }
        let (outcome, transaction) = self.delegate(5, bytes)?;
        if self.configuration.is_some() {
            if let (
                Outcome::Answer(answer),
                Some(LiveTransaction { change: Change::Model(transaction) }),
            ) = (&outcome, transaction)
            {
                let reply: umpire_runtime::realization::Reply =
                    Any::required(answer.reply.clone())?.unpack()?;
                return Ok((
                    outcome,
                    Some(LiveTransaction {
                        change: Change::Answer {
                            transaction,
                            task: Box::new(task),
                            response: Any::required(reply.request)?,
                        },
                    }),
                ));
            }
            return Ok((outcome, None));
        }
        Ok((outcome, transaction))
    }

    fn delegate(&self, method: u32, bytes: &[u8]) -> Prepared {
        let (response, transaction) = self.model.prepare(method, bytes);
        Ok((
            response.outcome.ok_or_else(|| state("Model response is missing"))?,
            transaction.map(|transaction| LiveTransaction { change: Change::Model(transaction) }),
        ))
    }

    fn start(&self, value: &Any) -> Prepared {
        let observation: Observation = value.unpack()?;
        let raw_request = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("Start actual request is missing"))?;
        let request: StartActivityExecutionRequest = raw_request.unpack()?;
        if self.configuration.is_some()
            && self.receipt.is_none()
            && self.bootstrap.as_ref() != Some(raw_request)
        {
            return Err(state("Start actual request does not match pending bootstrap"));
        }
        let Some(ResultValue::Response(raw_response)) = &observation.result else {
            return Err(Refusal::malformed("successful Start response is required"));
        };
        let response: StartActivityExecutionResponse = raw_response.unpack()?;
        validate_start(&request, &response)?;
        if let Some(receipt) = &self.receipt {
            receipt.matches(&request.namespace, &request.activity_id, &response.run_id)?;
            if receipt.request_id != request.request_id || receipt.request != *raw_request {
                return Err(state("Start request contradicts the original receipt"));
            }
            if response.started && receipt.response != *raw_response {
                return Err(state("Start response contradicts the original receipt"));
            }
            if !response.started && receipt.maximum_attempts.is_none() {
                return Err(state("Start replay has no original admitted provenance"));
            }
            if let Some(frozen) = &self.frozen {
                return Err(frozen.clone());
            }
            if receipt.maximum_attempts.is_none() {
                return Err(state("rejected Start requires a fresh guest"));
            }
            return Ok((explained(), None));
        }
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
            return Ok((
                Outcome::Refusal(refusal.clone()),
                Some(LiveTransaction { change: Change::Freeze { receipt, refusal } }),
            ));
        };
        let world = activity::World::new(maximum_attempts);
        let starts = world.starts();
        let [initial] = starts.as_slice() else {
            return Err(state("admitted World requires exactly one start"));
        };
        let model = Runtime::with_initial(world, Realization::default(), initial.clone())?;
        Ok((
            explained(),
            Some(LiveTransaction { change: Change::Admit { model: Box::new(model), receipt } }),
        ))
    }
}

impl LiveRuntime {
    fn install_context(&mut self) {
        if self.configuration.is_some()
            && let Some(receipt) = &self.receipt
        {
            let request: StartActivityExecutionRequest = receipt
                .request
                .unpack()
                .unwrap_or_else(|_| unreachable!("validated Start receipt"));
            self.model.replace_realization(Realization::configured(
                crate::temporal::ProposalContext {
                    start: request,
                    run_id: receipt.run_id.clone(),
                    response: self.callback.clone(),
                },
            ));
        }
    }
}

impl Receipt {
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

impl Dispatch for LiveRuntime {
    type Transaction = LiveTransaction;
    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<LiveTransaction>) {
        let (outcome, transaction) = self
            .prepare_method(method, request)
            .unwrap_or_else(|refusal| (Outcome::Refusal(refusal), None));
        (protocol::Response { outcome: Some(outcome) }, transaction)
    }
    fn commit(&mut self, transaction: LiveTransaction) {
        match transaction.change {
            Change::Model(transaction) => self.model.commit(transaction),
            Change::Admit { model, receipt } => {
                self.model = *model;
                self.receipt = Some(receipt);
                self.bootstrap = None;
                self.callback = None;
                self.install_context();
            }
            Change::Freeze { receipt, refusal } => {
                self.receipt = Some(receipt);
                self.frozen = Some(refusal);
            }
            Change::Configure(configuration) => {
                self.model = runtime();
                self.configuration = Some(configuration);
                self.receipt = None;
                self.bootstrap = None;
                self.callback = None;
                self.frozen = None;
            }
            Change::Propose(request) => self.bootstrap = Some(request),
            Change::Answer { transaction, task, response } => {
                self.model.commit(transaction);
                self.callback = Some((*task, response));
                self.install_context();
            }
            Change::Observe { transaction, clear_callback } => {
                self.model.commit(transaction);
                if clear_callback {
                    self.callback = None;
                }
                self.install_context();
            }
            Change::Reset(transaction) => {
                self.model.commit(transaction);
                self.frozen = None;
            }
        }
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
fn explained() -> Outcome {
    Outcome::Judge(protocol::JudgeResponse {
        continue_polling: false,
        judgment: Some(protocol::judge_response::Judgment::Explained(
            protocol::Explanation::default(),
        )),
    })
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
