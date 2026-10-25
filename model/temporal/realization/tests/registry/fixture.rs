use super::*;
use umpire_runtime::Realize;
use umpire_temporal::registry::Registry;

#[umpire_dsl::machine]
pub(crate) mod specimen {
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(crate) struct World {
        pub(crate) tag: u8,
    }

    #[derive(Action)]
    pub(crate) enum Client {
        #[metadata(temporal(rpc = WorkflowService::RequestCancelActivityExecution,
            direction = client, request = RequestCancelActivityExecutionRequest,
            response = RequestCancelActivityExecutionResponse, fill = cancel, evidence = canceled, request_reads(namespace, activity_id, run_id, request_id)))]
        Ping,
    }

    #[derive(Action)]
    pub(crate) enum Worker {
        #[metadata(temporal(rpc = WorkflowService::RespondActivityTaskCompleted,
            direction = worker, request = RespondActivityTaskCompletedRequest,
            response = RespondActivityTaskCompletedResponse, fill = complete, evidence = completed, request_reads(namespace, task_token, identity)))]
        Notify { flag: bool },
        #[metadata(temporal(rpc = WorkflowService::RespondActivityTaskCompleted,
            direction = worker, request = RespondActivityTaskCompletedRequest,
            response = RespondActivityTaskCompletedResponse, fill = complete, evidence = retried,
            request_reads(namespace, task_token, identity)))]
        RetryNotify(bool),
    }

    #[derive(Action)]
    pub(crate) enum Environment {
        #[metadata(temporal(event = "explicit refresh observation", evidence = refreshed))]
        Refresh,
    }

    #[derive(State)]
    pub(crate) struct Model {
        pub(crate) ping: bool,
        pub(crate) done: bool,
    }
    impl Model {
        #[starts]
        fn initial() -> impl IntoIterator<Item = Self> {
            [Self { ping: false, done: false }]
        }
        #[on(Client)]
        fn client(&mut self, _: Client) {
            self.ping = true;
        }
        #[on(Worker)]
        fn worker(&mut self, _: Worker) {
            if self.ping {
                self.done = true;
            }
        }
        #[on(Environment)]
        fn environment(&mut self, _: Environment) {
            self.ping = false;
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct ActualId {
    pub(crate) namespace: String,
    pub(crate) entity: String,
    pub(crate) run: String,
    pub(crate) token: Vec<u8>,
}
pub(crate) struct Receipt {
    pub(crate) identity: ActualId,
    pub(crate) request_id: String,
}

pub(crate) struct Inputs {
    pub(crate) request_id: String,
}
#[derive(Default)]
pub(crate) struct Binding {
    pub(crate) calls: std::cell::Cell<usize>,
    pub(crate) root_cursor: std::rc::Rc<std::cell::Cell<usize>>,
    pub(crate) conflicting_root: std::cell::Cell<bool>,
}
impl Identity for Binding {
    type Id = ActualId;
    fn identify(&self, value: &Any) -> Result<ActualId, Refusal> {
        let request: RequestCancelActivityExecutionRequest = value.unpack()?;
        if request.namespace.is_empty()
            || request.activity_id.is_empty()
            || request.run_id.is_empty()
        {
            return Err(Refusal::malformed("identity missing"));
        }
        Ok(ActualId {
            namespace: request.namespace,
            entity: request.activity_id,
            run: request.run_id,
            token: vec![9],
        })
    }
}

specimen::action_metadata!(umpire_temporal::registry; specimen; {
    #[registry(inputs = Inputs)]
    #[entity(rpc = WorkflowService::StartActivityExecution,
        request = StartActivityExecutionRequest, response = StartActivityExecutionResponse,
        receipt = Receipt, admit = admitted, make_receipt = receipt, correlate = correlation)]
    #[read(rpc = WorkflowService::DescribeActivityExecution,
        request = DescribeActivityExecutionRequest, response = DescribeActivityExecutionResponse,
        target = bool, fill = read_request, project = read_target,
        finality = is_final, continue_polling = polling, correlate = read_correlation)]
    impl Binding {
        fn cancel(&self, _: &specimen::Client, world: &specimen::World, inputs: &Inputs, id: &ActualId)
            -> Result<RequestCancelActivityExecutionRequest, Refusal> {
            Ok(RequestCancelActivityExecutionRequest { namespace: id.namespace.clone(), activity_id: id.entity.clone(), run_id: id.run.clone(), request_id: format!("{}-{}", inputs.request_id, world.tag), ..Default::default() })
        }
        fn complete(&self, action: &specimen::Worker, world: &specimen::World, inputs: &Inputs, id: &ActualId)
            -> Result<RespondActivityTaskCompletedRequest, Refusal> {
            let flag = match action { specimen::Worker::Notify { flag } | specimen::Worker::RetryNotify(flag) => flag };
            Ok(RespondActivityTaskCompletedRequest { task_token: id.token.clone(), namespace: id.namespace.clone(), identity: format!("{}-{}-{flag}", inputs.request_id, world.tag), ..Default::default() })
        }
        fn canceled(&self, request: RequestCancelActivityExecutionRequest, response: Result<RequestCancelActivityExecutionResponse, i32>, path: &[&umpire_runtime::Evidence<specimen::World>]) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.calls.set(self.calls.get() + 1);
            if request.namespace.is_empty() || request.activity_id.is_empty() || request.request_id.is_empty() { return Err(Refusal::malformed("required request identity missing")); }
            if request.run_id != "run" { return Err(Refusal::malformed("crossed run")); }
            if path.is_empty() { return Ok(if response.is_ok() { umpire_runtime::EvidenceMatch::Continue { cursor: self.root_cursor.get() } } else { umpire_runtime::EvidenceMatch::Reject }); }
            Ok(if response.is_ok() && path.last().is_some_and(|edge| !edge.source.ping && edge.target.ping && edge.choice == 0 && edge.facts.is_empty()) { umpire_runtime::EvidenceMatch::Match { cursor: self.root_cursor.get() + path.len() } } else { umpire_runtime::EvidenceMatch::Reject })
        }
        fn completed(&self, request: RespondActivityTaskCompletedRequest, response: Result<RespondActivityTaskCompletedResponse, i32>, path: &[&umpire_runtime::Evidence<specimen::World>]) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.calls.set(self.calls.get() + 1);
            if request.task_token != [9] { return Err(Refusal::malformed("crossed worker identity")); }
            if path.is_empty() { return Ok(if response.is_ok() { umpire_runtime::EvidenceMatch::Continue { cursor: self.root_cursor.get() } } else { umpire_runtime::EvidenceMatch::Reject }); }
            Ok(if request.task_token == [9] && response.is_ok() && path.len() == 2 && !path[0].source.ping && path[0].target.ping && path[1].source.ping && path[1].target.done { umpire_runtime::EvidenceMatch::Match { cursor: self.root_cursor.get() + 2 } } else { umpire_runtime::EvidenceMatch::Reject })
        }
        fn retried(&self, request: RespondActivityTaskCompletedRequest, response: Result<RespondActivityTaskCompletedResponse, i32>, path: &[&umpire_runtime::Evidence<specimen::World>]) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            let admitted = self.completed(request, response, path)?;
            if path.is_empty() && self.conflicting_root.get() { return Ok(umpire_runtime::EvidenceMatch::Continue { cursor: self.root_cursor.get() + 1 }); }
            Ok(admitted)
        }
        fn refreshed(&self, observation: &umpire_temporal::support::Observed, path: &[&umpire_runtime::Evidence<specimen::World>]) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            use umpire_runtime::EvidenceMatch;
            if observation.method != "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution" { return Ok(EvidenceMatch::Reject); }
            let request: DescribeActivityExecutionRequest = observation.request()?.ok_or_else(|| Refusal::malformed("read request missing"))?;
            let response: DescribeActivityExecutionResponse = observation.response()?;
            if request.namespace != "ns" || request.run_id != "run" || response.run_id != "run" { return Err(Refusal::malformed("crossed refresh")); }
            if path.is_empty() { return Ok(EvidenceMatch::Continue { cursor: 7 }); }
            Ok(if path.len() == 1 && path[0].source.ping && !path[0].target.ping && matches!(path[0].action, specimen::Action::Environment(specimen::Environment::Refresh)) { EvidenceMatch::Match { cursor: 8 } } else { EvidenceMatch::Reject })
        }
        fn admitted(&self, request: &StartActivityExecutionRequest, response: &StartActivityExecutionResponse) -> Result<ActualId, Refusal> {
            if request.namespace.is_empty() || request.activity_id.is_empty() || request.request_id.is_empty() || response.run_id.is_empty() { return Err(Refusal::malformed("bootstrap identity missing")); }
            Ok(ActualId { namespace: request.namespace.clone(), entity: request.activity_id.clone(), run: response.run_id.clone(), token: vec![9] })
        }
        fn receipt(&self, id: &ActualId, request: &StartActivityExecutionRequest, _: &StartActivityExecutionResponse) -> Result<Receipt, Refusal> { Ok(Receipt { identity: id.clone(), request_id: request.request_id.clone() }) }
        fn correlation(&self, receipt: &Receipt, request: &StartActivityExecutionRequest, response: &StartActivityExecutionResponse) -> Result<(), Refusal> {
            if receipt.identity.namespace != request.namespace || receipt.identity.entity != request.activity_id || receipt.identity.run != response.run_id || receipt.request_id != request.request_id { return Err(Refusal::malformed("crossed bootstrap")); }
            Ok(())
        }
        fn read_request(&self, _: &specimen::World, receipt: &Receipt) -> Result<DescribeActivityExecutionRequest, Refusal> {
            Ok(DescribeActivityExecutionRequest { namespace: receipt.identity.namespace.clone(), activity_id: receipt.identity.entity.clone(), run_id: receipt.identity.run.clone(), ..Default::default() })
        }
        fn read_correlation(&self, receipt: &Receipt, request: &DescribeActivityExecutionRequest, response: &DescribeActivityExecutionResponse) -> Result<(), Refusal> {
            if receipt.identity.namespace != request.namespace || receipt.identity.entity != request.activity_id || receipt.identity.run != request.run_id || receipt.identity.run != response.run_id { return Err(Refusal::malformed("crossed read")); }
            Ok(())
        }
        fn read_target(&self, value: &DescribeActivityExecutionResponse) -> Result<bool, Refusal> { Ok(value.outcome.is_some()) }
        fn is_final(&self, value: &bool) -> bool { *value }
        fn polling(&self, value: &bool) -> bool { !value }
    }
});

impl umpire_runtime::capability::Snapshot<specimen::World> for Binding {
    type Target = umpire_runtime::Exact<specimen::Model>;
    fn project(
        &self,
        _: &specimen::World,
        value: &Any,
        state: &specimen::Model,
    ) -> Result<umpire_runtime::Projection<specimen::Action, (), Self::Target>, Refusal> {
        let observation = umpire_temporal::support::Observed::new(value)?;
        let mut target = state.clone();
        match observation.method.as_str() {
            "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution" => {
                target.ping = true
            }
            "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution" => {
                target.ping = false
            }
            _ => return Err(Refusal::unsupported("synthetic projection method unsupported")),
        }
        Ok(umpire_runtime::Projection::Reach(umpire_runtime::Exact(target)))
    }
}
impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Binding {
    type World = specimen::World;
    type View = umpire_temporal::support::Observed;
    fn admit_evidence(&self, observation: &Any) -> Result<Self::View, Refusal> {
        umpire_temporal::support::Observed::new(observation)
    }
}
impl umpire_runtime::capability::ReplyPolicy<specimen::World> for Binding {}
impl Realize<specimen::World> for Binding {
    fn reach_evidence(
        &self,
        observation: &Any,
        path: &[&umpire_runtime::Evidence<specimen::World>],
    ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
        Registry::check_registered(self, observation, path)
    }
}
