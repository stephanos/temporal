use crate::AnyMessage as _;
use prost::Message;
use umpire_checker::{Checker, StateId};
use umpire_dsl::Machine;

use crate::{Dispatch, prost_types::Any, protocol, refusal::checker_refusal};

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

/// A realization-selected external request, checked against the Model before publication.
pub enum Proposal<A> {
    /// One enabled external Model action, without committing its step.
    Action {
        /// Selected external action.
        action: A,
        /// Fully filled system request.
        request: Any,
    },
    /// A system read, with no proposed Model action.
    Read {
        /// Fully filled observation instruction.
        request: Any,
    },
    /// Admission to a fresh run, outside the Model transition graph.
    Bootstrap {
        /// Fully filled admission request.
        request: Any,
    },
}

/// A system-owned payload registry and projection realization for one Model.
pub trait Realize<M: Machine> {
    /// Whether this realization admits stateful operations.
    fn enabled(&self) -> bool {
        true
    }

    /// Whether a fresh run requires admission before callbacks and action proposals.
    fn needs_bootstrap(&self) -> bool {
        false
    }

    /// Issues the worker program for this realization and its bound World.
    fn program(&self) -> Result<Any, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("program unsupported"))
    }

    /// Selects a deterministic external request for the current state and seed.
    fn propose(
        &self,
        _state: &M::State,
        _seed: u64,
    ) -> Result<Proposal<M::Action>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("proposal unsupported"))
    }

    /// Supplies admission independently of the action proposal policy.
    fn bootstrap(
        &self,
        state: &M::State,
        seed: u64,
    ) -> Result<Proposal<M::Action>, protocol::Refusal> {
        self.propose(state, seed)
    }

    /// Verifies that an observation belongs to the pending request.
    fn matches(&self, _request: &Any, _observation: &Any) -> Result<(), protocol::Refusal> {
        Err(protocol::Refusal::unsupported("proposal correlation unsupported"))
    }

    /// Requests another observation after a positively explained intermediate read.
    fn continue_polling(
        &self,
        _observation: &Any,
        _state: &M::State,
        _action: Option<&M::Action>,
    ) -> Result<bool, protocol::Refusal> {
        Ok(false)
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
        _callback: &Any,
        _state: &M::State,
    ) -> Result<AnswerPlan<M::Action>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("answer unsupported"))
    }
}

/// One independently owned Model, realization, exploration, and current run.
pub struct Runtime<M: Machine, R: Realize<M>> {
    machine: M,
    realization: R,
    state: Option<M::State>,
    initial: Option<M::State>,
    frozen: Option<protocol::Unexplained>,
    exploration: Option<umpire_checker::Result<M>>,
    pending: Option<Pending<M::Action>>,
    admitted: bool,
}

#[derive(Clone)]
struct Pending<A> {
    request: Any,
    action: Option<A>,
}

/// A prepared dispatch's opaque prospective changes.
pub struct Transaction<M: Machine> {
    change: Change<M>,
}

enum Change<M: Machine> {
    Explore(Box<umpire_checker::Result<M>>),
    Run {
        state: M::State,
        frozen: Option<protocol::Unexplained>,
        pending: Option<Pending<M::Action>>,
        admitted: bool,
    },
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
        let admitted = !realization.needs_bootstrap();
        Self {
            machine,
            realization,
            initial: state.clone(),
            state,
            frozen: None,
            exploration: None,
            pending: None,
            admitted,
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
        let admitted = !realization.needs_bootstrap();
        Ok(Self {
            machine,
            realization,
            state: Some(initial.clone()),
            initial: Some(initial),
            frozen: None,
            exploration: None,
            pending: None,
            admitted,
        })
    }

    /// Replaces transport context inside a composing dispatcher's publication commit.
    pub fn replace_realization(&mut self, realization: R) {
        self.realization = realization;
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
                let request = decode::<protocol::ResetRequest>(request)?;
                if request.configuration.is_some() {
                    return Err(protocol::Refusal::unsupported("configuration unsupported"));
                }
                self.require_realization()?;
                let state = self.initial.clone().ok_or_else(|| {
                    protocol::Refusal::new(
                        protocol::RefusalCode::State,
                        "stateful run requires exactly one start",
                    )
                })?;
                let program = match self.realization.program() {
                    Ok(program) => Some(program),
                    Err(refusal) if refusal.code() == protocol::RefusalCode::Unsupported => None,
                    Err(refusal) => return Err(refusal),
                };
                Ok((
                    protocol::response::Outcome::Reset(protocol::ResetResponse { program }),
                    Some(Transaction {
                        change: Change::Run {
                            state,
                            frozen: None,
                            pending: None,
                            admitted: !self.realization.needs_bootstrap(),
                        },
                    }),
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
            6 => {
                let request = decode::<protocol::ProposeRequest>(request)?;
                self.require_realization()?;
                self.propose(request.seed)
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
        if !self.admitted && self.pending.is_none() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "bootstrap proposal required",
            ));
        }
        let projection = self.realization.observe(observation, current)?;
        if !self.admitted && !matches!(projection, Projection::Current) {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "bootstrap cannot step the Model",
            ));
        }
        if let Some(request) = &self.pending {
            self.realization.matches(&request.request, observation)?;
        }
        if let Some(frozen) = &self.frozen {
            return Ok((judgment(Judgment::Unexplained(frozen.clone())), None));
        }
        match projection {
            Projection::Current => {
                return self.explained(
                    observation,
                    current.clone(),
                    protocol::Explanation::default(),
                );
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
                        return self.explained(
                            observation,
                            replay.state,
                            protocol::Explanation {
                                steps: replay.steps,
                                outcomes: replay.outcomes,
                            },
                        );
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
                change: Change::Run {
                    state: current.clone(),
                    frozen: Some(frozen),
                    pending: self.pending.clone(),
                    admitted: self.admitted,
                },
            }),
        ))
    }

    fn explained(
        &self,
        observation: &Any,
        state: M::State,
        explanation: protocol::Explanation,
    ) -> Prepared<M> {
        let continues = self.pending.is_some()
            && self.realization.continue_polling(
                observation,
                &state,
                self.pending.as_ref().and_then(|pending| pending.action.as_ref()),
            )?;
        let pending = if continues { self.pending.clone() } else { None };
        Ok((
            protocol::response::Outcome::Judge(protocol::JudgeResponse {
                judgment: Some(protocol::judge_response::Judgment::Explained(explanation)),
                continue_polling: continues,
            }),
            Some(Transaction {
                change: Change::Run {
                    state,
                    frozen: None,
                    pending,
                    admitted: self.admitted || (self.pending.is_some() && !continues),
                },
            }),
        ))
    }

    fn propose(&self, seed: u64) -> Prepared<M> {
        if self.frozen.is_some() || self.pending.is_some() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "run frozen or proposal pending until final Judge or Reset",
            ));
        }
        let current = self.current()?;
        let mut external = false;
        if self.admitted {
            for action in self.catalog()? {
                if !self.machine.internal(&action)
                    && !self.checked_step(current, action)?.is_empty()
                {
                    external = true;
                    break;
                }
            }
        }
        let proposal = if self.admitted {
            self.realization.propose(current, seed)
        } else {
            self.realization.bootstrap(current, seed)
        };
        if self.admitted && !external && !matches!(proposal, Ok(Proposal::Read { .. })) {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::Quiescent,
                "no enabled external action",
            ));
        }
        let (request, action, kind, pending_action) = match proposal? {
            Proposal::Action { action, request } => {
                if !self.admitted
                    || !self.catalog()?.contains(&action)
                    || self.machine.internal(&action)
                    || self.checked_step(current, action.clone())?.is_empty()
                {
                    return Err(protocol::Refusal::new(
                        protocol::RefusalCode::State,
                        "proposed action is not enabled externally",
                    ));
                }
                (request, action.to_string(), protocol::ProposalKind::Action, Some(action))
            }
            Proposal::Read { request } => {
                if !self.admitted {
                    return Err(protocol::Refusal::new(
                        protocol::RefusalCode::State,
                        "read requires bootstrap admission",
                    ));
                }
                (request, String::new(), protocol::ProposalKind::Read, None)
            }
            Proposal::Bootstrap { request } => {
                if self.admitted || !self.realization.needs_bootstrap() {
                    return Err(protocol::Refusal::new(
                        protocol::RefusalCode::State,
                        "bootstrap unavailable",
                    ));
                }
                (request, String::new(), protocol::ProposalKind::Bootstrap, None)
            }
        };
        if request.type_url.is_empty() {
            return Err(protocol::Refusal::malformed("proposal request type is missing"));
        }
        Ok((
            protocol::response::Outcome::Propose(protocol::ProposeResponse {
                request: Some(request.clone()),
                action,
                kind: kind as i32,
            }),
            Some(Transaction {
                change: Change::Run {
                    state: current.clone(),
                    frozen: None,
                    pending: Some(Pending { request, action: pending_action }),
                    admitted: self.admitted,
                },
            }),
        ))
    }

    fn answer(&self, callback: &Any) -> Prepared<M> {
        if !self.admitted {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "bootstrap admission required",
            ));
        }
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
            Some(Transaction {
                change: Change::Run {
                    state,
                    frozen: None,
                    pending: self.pending.clone(),
                    admitted: self.admitted,
                },
            }),
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
            Change::Run { state, frozen, pending, admitted } => {
                self.pending = pending;
                self.admitted = admitted;
                self.state = Some(state);
                self.frozen = frozen;
            }
        }
    }
}

fn judgment(value: protocol::judge_response::Judgment) -> protocol::response::Outcome {
    protocol::response::Outcome::Judge(protocol::JudgeResponse {
        judgment: Some(value),
        continue_polling: false,
    })
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
