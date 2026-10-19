use prost::Message;
use umpire_checker::{Checker, StateId};
use umpire_dsl::Machine;

use crate::{Dispatch, google::protobuf::Any, protocol, refusal::checker_refusal};

/// One declared action and its written-order enabled choice.
pub struct Transition<A> {
    /// The Model action to replay.
    pub action: A,
    /// Index into the Model's enabled choices.
    pub choice: usize,
}

/// A validated observation's positive current match or proposed transition paths.
pub enum Projection<A> {
    /// The observation positively matches the current state.
    Current,
    /// Candidate nonempty paths; none matching is an unexplained observation.
    Candidates(Vec<Vec<Transition<A>>>),
}

/// A validated callback's delivery evidence and selected response.
pub struct AnswerPlan<A> {
    /// Model steps evidenced by delivery, committed only after response publication.
    pub delivery: Vec<Transition<A>>,
    /// The action the reply requests, checked for enablement but not committed.
    pub response_action: A,
    /// The system-owned reply envelope.
    pub reply: Any,
}

/// A system-owned payload registry and projection binding for one Model.
pub trait Realize<M: Machine> {
    /// Whether this binding admits stateful operations.
    fn enabled(&self) -> bool {
        true
    }

    /// Validates and projects an observation without changing Model state.
    fn observe(
        &self,
        observation: &Any,
        state: &M::State,
    ) -> Result<Projection<M::Action>, protocol::Refusal>;

    /// Positively matches an observation against an actual replay endpoint.
    fn matches(&self, observation: &Any, state: &M::State) -> Result<bool, protocol::Refusal>;

    /// Validates the complete callback and selects its delivery path and response.
    fn answer(
        &self,
        callback: &Any,
        state: &M::State,
    ) -> Result<AnswerPlan<M::Action>, protocol::Refusal>;
}

/// One independently owned Model, realization, exploration, and current run.
pub struct Runtime<M: Machine, R: Realize<M>> {
    machine: M,
    realization: R,
    state: Option<M::State>,
    frozen: Option<protocol::Unexplained>,
    exploration: Option<umpire_checker::Result<M>>,
}

/// A prepared dispatch's opaque prospective changes.
pub struct Transaction<M: Machine> {
    change: Change<M>,
}

enum Change<M: Machine> {
    Explore(Box<umpire_checker::Result<M>>),
    Run { state: M::State, frozen: Option<protocol::Unexplained> },
}

type Prepared<M> = Result<(protocol::response::Outcome, Option<Transaction<M>>), protocol::Refusal>;

impl<M: Machine, R: Realize<M>> Runtime<M, R> {
    /// Creates a private run. Stateful calls refuse zero or multiple declared starts.
    pub fn new(machine: M, realization: R) -> Self {
        let state = unique_start(&machine).ok();
        Self { machine, realization, state, frozen: None, exploration: None }
    }

    /// Dispatches natively, committing after the response encodes successfully.
    pub fn dispatch(&mut self, method: u32, request: &[u8]) -> protocol::Response {
        let (response, transaction) = self.prepare(method, request);
        if crate::abi::encode_response(&response).is_none() {
            return protocol::Response {
                outcome: Some(protocol::response::Outcome::Refusal(protocol::Refusal::new(
                    protocol::RefusalCode::Allocation,
                    "response encoding or allocation failed",
                ))),
            };
        }
        if let Some(transaction) = transaction {
            self.commit(transaction);
        }
        response
    }

    fn prepare_method(&self, method: u32, request: &[u8]) -> Prepared<M> {
        match method {
            1 => self.explore(decode::<protocol::ExploreRequest>(request)?.budget),
            2 => self.rows(decode::<protocol::RowsRequest>(request)?.state_id),
            3 => {
                decode::<protocol::ResetRequest>(request)?;
                self.require_binding()?;
                let state = unique_start(&self.machine)?;
                Ok((
                    protocol::response::Outcome::Reset(protocol::ResetResponse {}),
                    Some(Transaction { change: Change::Run { state, frozen: None } }),
                ))
            }
            4 => {
                let request = decode::<protocol::JudgeRequest>(request)?;
                let observation = Any::required(request.observation)?;
                self.require_binding()?;
                self.judge(&observation)
            }
            5 => {
                let request = decode::<protocol::AnswerRequest>(request)?;
                let callback = Any::required(request.callback)?;
                self.require_binding()?;
                self.answer(&callback)
            }
            _ => Err(protocol::Refusal::new(
                protocol::RefusalCode::UnknownMethod,
                format!("unknown method: {method}"),
            )),
        }
    }

    fn require_binding(&self) -> Result<(), protocol::Refusal> {
        if self.realization.enabled() {
            Ok(())
        } else {
            Err(protocol::Refusal::new(
                protocol::RefusalCode::Unsupported,
                "stateful operation unsupported",
            ))
        }
    }

    fn current(&self) -> Result<&M::State, protocol::Refusal> {
        self.state.as_ref().ok_or_else(|| {
            protocol::Refusal::new(
                protocol::RefusalCode::State,
                "stateful run requires exactly one start",
            )
        })
    }

    fn judge(&self, observation: &Any) -> Prepared<M> {
        use protocol::judge_response::Judgment;

        let current = self.current()?;
        let projection = self.realization.observe(observation, current)?;
        if let Some(frozen) = &self.frozen {
            return Ok((judgment(Judgment::Unexplained(frozen.clone())), None));
        }
        match projection {
            Projection::Current if self.realization.matches(observation, current)? => {
                return Ok((
                    judgment(Judgment::Explained(protocol::Explanation { steps: Vec::new() })),
                    None,
                ));
            }
            Projection::Candidates(paths) => {
                for path in paths {
                    if path.is_empty() {
                        continue;
                    }
                    if let Some((state, steps)) = self.replay(current, path)
                        && self.realization.matches(observation, &state)?
                    {
                        return Ok((
                            judgment(Judgment::Explained(protocol::Explanation { steps })),
                            Some(Transaction { change: Change::Run { state, frozen: None } }),
                        ));
                    }
                }
            }
            Projection::Current => {}
        }
        let frozen = protocol::Unexplained {
            expected_actions: self
                .machine
                .actions()
                .into_iter()
                .filter(|action| !self.machine.step(current, action.clone()).is_empty())
                .map(|action| action.to_string())
                .collect(),
        };
        Ok((
            judgment(Judgment::Unexplained(frozen.clone())),
            Some(Transaction {
                change: Change::Run { state: current.clone(), frozen: Some(frozen) },
            }),
        ))
    }

    fn answer(&self, callback: &Any) -> Prepared<M> {
        let current = self.current()?;
        if self.frozen.is_some() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "run is frozen until Reset",
            ));
        }
        let plan = self.realization.answer(callback, current)?;
        let (state, _) = self.replay(current, plan.delivery).ok_or_else(|| {
            protocol::Refusal::new(protocol::RefusalCode::State, "callback delivery is not enabled")
        })?;
        if plan.reply.type_url.is_empty() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::MalformedRequest,
                "reply type is missing",
            ));
        }
        if !self.machine.actions().contains(&plan.response_action)
            || self.machine.step(&state, plan.response_action).is_empty()
        {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "selected response is not enabled",
            ));
        }
        Ok((
            protocol::response::Outcome::Answer(protocol::AnswerResponse {
                reply: Some(plan.reply),
            }),
            Some(Transaction { change: Change::Run { state, frozen: None } }),
        ))
    }

    fn replay(
        &self,
        current: &M::State,
        path: Vec<Transition<M::Action>>,
    ) -> Option<(M::State, Vec<String>)> {
        let mut state = current.clone();
        let mut steps = Vec::new();
        let catalog = self.machine.actions();
        for transition in path {
            if !catalog.contains(&transition.action) {
                return None;
            }
            let label = transition.action.to_string();
            let (_, _, next) =
                self.machine.step(&state, transition.action).into_iter().nth(transition.choice)?;
            state = next;
            steps.push(label);
        }
        Some((state, steps))
    }

    fn explore(&self, budget: u32) -> Prepared<M> {
        let result = Checker::explore(&self.machine, budget as usize)
            .map_err(|error| checker_error(&error))?;
        let report = result.verify_all().map_err(|error| checker_error(&error))?;
        let counts = &result.counts;
        let response = protocol::ExploreResponse {
            report: report.to_string(),
            starts: count(counts.starts),
            reached: count(counts.reached),
            emitted: count(counts.emitted),
            evaluated: count(counts.evaluated),
            refused: count(counts.refused),
            depth: count(counts.depth),
            cut: counts.cut,
        };
        Ok((
            protocol::response::Outcome::Explore(response),
            Some(Transaction { change: Change::Explore(Box::new(result)) }),
        ))
    }

    fn rows(&self, state_id: u32) -> Prepared<M> {
        let result = self.exploration.as_ref().ok_or_else(|| {
            protocol::Refusal::new(
                protocol::RefusalCode::NotExplored,
                "Rows requires a successful Explore",
            )
        })?;
        let mut rows = Vec::new();
        for edge in
            result.rows(StateId(state_id as usize)).map_err(|error| checker_error(&error))?
        {
            rows.push(format!(
                "{state_id}: {} choice={} outcome={:?} facts={:?} -> {}\n",
                edge.action, edge.choice, edge.emitted.0, edge.emitted.1, edge.emitted.2
            ));
        }
        Ok((protocol::response::Outcome::Rows(protocol::RowsResponse { rows }), None))
    }
}

impl<M: Machine, R: Realize<M>> Dispatch for Runtime<M, R> {
    type Transaction = Transaction<M>;

    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>) {
        let (outcome, transaction) = self
            .prepare_method(method, request)
            .unwrap_or_else(|refusal| (protocol::response::Outcome::Refusal(refusal), None));
        (protocol::Response { outcome: Some(outcome) }, transaction)
    }

    fn commit(&mut self, transaction: Self::Transaction) {
        match transaction.change {
            Change::Explore(result) => self.exploration = Some(*result),
            Change::Run { state, frozen } => {
                self.state = Some(state);
                self.frozen = frozen;
            }
        }
    }
}

fn judgment(value: protocol::judge_response::Judgment) -> protocol::response::Outcome {
    protocol::response::Outcome::Judge(protocol::JudgeResponse { judgment: Some(value) })
}

fn unique_start<M: Machine>(machine: &M) -> Result<M::State, protocol::Refusal> {
    let mut starts = machine.starts();
    if starts.len() != 1 {
        return Err(protocol::Refusal::new(
            protocol::RefusalCode::State,
            "stateful run requires exactly one start",
        ));
    }
    Ok(starts.remove(0))
}

fn decode<M: Message + Default>(request: &[u8]) -> Result<M, protocol::Refusal> {
    M::decode(request).map_err(|error| {
        protocol::Refusal::new(protocol::RefusalCode::MalformedRequest, error.to_string())
    })
}

fn checker_error(error: &umpire_checker::Refusal) -> protocol::Refusal {
    let protocol::response::Outcome::Refusal(refusal) = checker_refusal(error) else {
        unreachable!("checker errors are refusals")
    };
    refusal
}

fn count(value: usize) -> u32 {
    u32::try_from(value).unwrap_or_else(|_| unreachable!("bounded counts fit u32"))
}
