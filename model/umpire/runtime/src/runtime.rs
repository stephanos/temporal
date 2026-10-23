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

impl<A> From<A> for Transition<A> {
    fn from(action: A) -> Self {
        Self { action, choice: 0 }
    }
}

/// A proposed path and the optional expected outcome of its final edge.
pub struct Candidate<A, O = umpire_dsl::Outcome> {
    /// Model transitions to replay in order.
    pub path: Vec<Transition<A>>,
    /// Required final outcome; absent means any outcome matches.
    pub outcome: Option<O>,
}

/// A validated observation's positive current match or proposed transition paths.
pub enum Projection<A, O = umpire_dsl::Outcome> {
    /// The observation positively matches the current state.
    Current,
    /// Candidate nonempty paths; none matching is an unexplained observation.
    Candidates(Vec<Candidate<A, O>>),
}

impl<A, O> Projection<A, O> {
    /// Proposes one action using its first enabled choice and any outcome.
    pub fn step(action: A) -> Self {
        Self::steps([vec![action.into()]])
    }

    /// Proposes paths in order, accepting any final outcome.
    pub fn steps(paths: impl IntoIterator<Item = Vec<Transition<A>>>) -> Self {
        Self::Candidates(paths.into_iter().map(|path| Candidate { path, outcome: None }).collect())
    }
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

impl<A> AnswerPlan<A> {
    /// Creates a delivery path and response with an already packed system reply.
    pub fn new(delivery: Vec<Transition<A>>, response_action: A, reply: Any) -> Self {
        Self { delivery, response_action, reply }
    }
}

/// A system-owned payload registry and projection realization for one Model.
pub trait Realize<M: Machine> {
    /// Whether this realization admits stateful operations.
    fn enabled(&self) -> bool {
        true
    }

    /// Validates and projects an observation without changing Model state.
    fn observe(
        &self,
        observation: &Any,
        state: &M::State,
    ) -> Result<Projection<M::Action, M::Outcome>, protocol::Refusal>;

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
    initial: Option<M::State>,
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

struct Replay<M: Machine> {
    state: M::State,
    steps: Vec<String>,
    outcomes: Vec<String>,
    outcome: Option<M::Outcome>,
}

impl<M: Machine, R: Realize<M>> Runtime<M, R> {
    /// Creates a private run. Stateful calls refuse zero or multiple declared starts.
    pub fn new(machine: M, realization: R) -> Self {
        let state = unique_start(&machine).ok();
        Self {
            machine,
            realization,
            initial: state.clone(),
            state,
            frozen: None,
            exploration: None,
        }
    }

    /// Creates a private run from an exact member of the Model's declared starts.
    ///
    /// Returns a State refusal when the complete initial state is not declared.
    pub fn with_initial(
        machine: M,
        realization: R,
        initial: M::State,
    ) -> Result<Self, protocol::Refusal> {
        if !machine.starts().contains(&initial) {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "initial state is not a declared start",
            ));
        }
        Ok(Self {
            machine,
            realization,
            state: Some(initial.clone()),
            initial: Some(initial),
            frozen: None,
            exploration: None,
        })
    }

    /// Dispatches natively, committing after the response encodes successfully.
    pub fn dispatch(&mut self, method: u32, request: &[u8]) -> protocol::Response
    where
        M::Outcome: PartialEq,
    {
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

    fn prepare_method(&self, method: u32, request: &[u8]) -> Prepared<M>
    where
        M::Outcome: PartialEq,
    {
        match method {
            1 => self.explore(decode::<protocol::ExploreRequest>(request)?.budget),
            2 => self.rows(decode::<protocol::RowsRequest>(request)?.state_id),
            3 => {
                decode::<protocol::ResetRequest>(request)?;
                self.require_realization()?;
                let state = self.initial.clone().ok_or_else(|| {
                    protocol::Refusal::new(
                        protocol::RefusalCode::State,
                        "stateful run requires exactly one start",
                    )
                })?;
                Ok((
                    protocol::response::Outcome::Reset(protocol::ResetResponse {}),
                    Some(Transaction { change: Change::Run { state, frozen: None } }),
                ))
            }
            4 => {
                let request = decode::<protocol::JudgeRequest>(request)?;
                let observation = Any::required(request.observation)?;
                self.require_realization()?;
                self.judge(&observation)
            }
            5 => {
                let request = decode::<protocol::AnswerRequest>(request)?;
                let callback = Any::required(request.callback)?;
                self.require_realization()?;
                self.answer(&callback)
            }
            _ => Err(protocol::Refusal::new(
                protocol::RefusalCode::UnknownMethod,
                format!("unknown method: {method}"),
            )),
        }
    }

    fn require_realization(&self) -> Result<(), protocol::Refusal> {
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

    fn judge(&self, observation: &Any) -> Prepared<M>
    where
        M::Outcome: PartialEq,
    {
        use protocol::judge_response::Judgment;

        let current = self.current()?;
        let projection = self.realization.observe(observation, current)?;
        if let Some(frozen) = &self.frozen {
            return Ok((judgment(Judgment::Unexplained(frozen.clone())), None));
        }
        match projection {
            Projection::Current => {
                return Ok((judgment(Judgment::Explained(protocol::Explanation::default())), None));
            }
            Projection::Candidates(paths) => {
                for candidate in paths {
                    if candidate.path.is_empty() {
                        continue;
                    }
                    if let Some(replay) = self.replay(current, candidate.path)?
                        && candidate
                            .outcome
                            .as_ref()
                            .is_none_or(|expected| Some(expected) == replay.outcome.as_ref())
                    {
                        return Ok((
                            judgment(Judgment::Explained(protocol::Explanation {
                                steps: replay.steps,
                                outcomes: replay.outcomes,
                            })),
                            Some(Transaction {
                                change: Change::Run { state: replay.state, frozen: None },
                            }),
                        ));
                    }
                }
            }
        }
        let mut expected_actions = Vec::new();
        for action in self.catalog()? {
            if !self.checked_step(current, action.clone())?.is_empty() {
                expected_actions.push(action.to_string());
            }
        }
        let frozen = protocol::Unexplained { expected_actions };
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
        let Replay { state, .. } = self.replay(current, plan.delivery)?.ok_or_else(|| {
            protocol::Refusal::new(protocol::RefusalCode::State, "callback delivery is not enabled")
        })?;
        if plan.reply.type_url.is_empty() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::MalformedRequest,
                "reply type is missing",
            ));
        }
        if !self.catalog()?.contains(&plan.response_action)
            || self.checked_step(&state, plan.response_action)?.is_empty()
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
    ) -> Result<Option<Replay<M>>, protocol::Refusal> {
        let mut state = current.clone();
        let mut steps = Vec::new();
        let mut outcomes = Vec::new();
        let mut outcome = None;
        let catalog = self.catalog()?;
        for transition in path {
            if !catalog.contains(&transition.action) {
                return Ok(None);
            }
            let label = transition.action.to_string();
            let Some((emitted, _, next)) =
                self.checked_step(&state, transition.action)?.into_iter().nth(transition.choice)
            else {
                return Ok(None);
            };
            state = next;
            steps.push(label);
            outcomes.push(format!("{emitted:?}"));
            outcome = Some(emitted);
        }
        Ok(Some(Replay { state, steps, outcomes, outcome }))
    }

    fn catalog(&self) -> Result<Vec<M::Action>, protocol::Refusal> {
        self.machine
            .checked_catalog(Checker::CATALOG_LIMIT)
            .map(umpire_dsl::ActionCatalog::into_actions)
            .map_err(|error| checker_error(&error.into()))
    }

    fn checked_step(
        &self,
        state: &M::State,
        action: M::Action,
    ) -> Result<umpire_dsl::Steps<M::State, M::Outcome, M::Fact>, protocol::Refusal> {
        let label = action.to_string();
        self.machine.checked_step(state, action).map_err(|error| {
            checker_error(&umpire_checker::Refusal::Step {
                action: label,
                before: state.to_string(),
                error,
            })
        })
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

impl<M: Machine, R: Realize<M>> Dispatch for Runtime<M, R>
where
    M::Outcome: PartialEq,
{
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
