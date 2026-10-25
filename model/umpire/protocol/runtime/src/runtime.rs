use crate::AnyMessage as _;
use prost::Message;
use std::rc::Rc;
use umpire_checker::search::{SearchLimit, Traversal};
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

/// A pure compatibility filter over a checked full Model state.
pub trait ReachTarget<S> {
    /// Whether the actual observation is compatible with this candidate.
    fn matches(&self, candidate: &S) -> bool;
}

/// Complete-state compatibility, without manufacturing hidden state fields.
pub struct Exact<S>(pub S);

impl<S: PartialEq> ReachTarget<S> for Exact<S> {
    fn matches(&self, candidate: &S) -> bool {
        self.0 == *candidate
    }
}

/// Adapter for realizations that only declare Current and Candidates.
pub enum NoTarget {}

impl<S> ReachTarget<S> for NoTarget {
    fn matches(&self, _: &S) -> bool {
        match *self {}
    }
}

mod singleton {
    pub trait Sealed {}
    impl<S> Sealed for super::Exact<S> {}
    impl Sealed for super::NoTarget {}
}

/// Targets admitted by the singleton runtime before compatible alternatives are supported.
///
/// Partial targets remain usable as pure filters, but cannot drive a greedy runtime.
///
/// ```compile_fail
/// use umpire_dsl::{Machine, Steps};
/// use umpire_runtime::{Projection, ReachTarget, Realize, Runtime, prost_types::Any, protocol};
/// struct World;
/// impl Machine for World {
///     type State = u8;
///     type Action = u8;
///     type Outcome = ();
///     type Fact = ();
///     fn starts(&self) -> Vec<u8> { vec![0] }
///     fn actions(&self) -> Vec<u8> { vec![1] }
///     fn internal(&self, _: &u8) -> bool { false }
///     fn step(&self, _: &u8, _: u8) -> Steps<u8, (), ()> { Steps::new() }
/// }
/// struct Partial;
/// impl ReachTarget<u8> for Partial {
///     fn matches(&self, _: &u8) -> bool { true }
/// }
/// struct Binding;
/// impl Realize<World> for Binding {
///     type Target = Partial;
///     fn observe(&self, _: &Any, _: &u8)
///         -> Result<Projection<u8, (), Partial>, protocol::Refusal> {
///         Ok(Projection::Reach(Partial))
///     }
/// }
/// let _ = Runtime::new(World, Binding);
/// ```
pub trait SingletonTarget<S>: ReachTarget<S> + singleton::Sealed {}

impl<S: PartialEq> SingletonTarget<S> for Exact<S> {}
impl<S> SingletonTarget<S> for NoTarget {}

/// Admitted finite search and retained-prefix limits.
#[derive(Clone, Copy)]
pub struct Scope {
    /// Maximum transitions in one Reach path.
    pub depth: usize,
    /// Maximum search labels, checked actions and examined emitted choices.
    pub work: usize,
    /// Maximum retained actual events plus typed witness edges.
    pub prefix: usize,
}

impl Default for Scope {
    fn default() -> Self {
        Self { depth: 64, work: 65_536, prefix: 1024 }
    }
}

/// One immutable checked edge, retaining ordered typed emissions.
pub struct Evidence<M: Machine> {
    /// Complete source state.
    pub source: M::State,
    /// Catalog action.
    pub action: M::Action,
    /// Enabled choice in written order.
    pub choice: usize,
    /// Actual checked-step outcome.
    pub outcome: M::Outcome,
    /// Actual checked-step facts in emitted order.
    pub facts: Vec<M::Fact>,
    /// Complete destination state.
    pub target: M::State,
}

/// Positive evidence admission for a particular typed witness prefix.
#[derive(Clone, Copy, PartialEq, Eq)]
pub enum EvidenceMatch {
    /// This prefix contradicts or lacks required evidence.
    Reject,
    /// Explicitly compatible progression, with evidence consumption position.
    Continue {
        /// Caller-owned consumption position.
        cursor: usize,
    },
    /// Positive completed evidence, with evidence consumption position.
    Match {
        /// Caller-owned consumption position.
        cursor: usize,
    },
    /// Explicit full-state-preserving edge evidence, distinct from a Model self-loop match.
    Stutter {
        /// Caller-owned consumption position.
        cursor: usize,
    },
}

impl EvidenceMatch {
    fn cursor(self) -> Option<usize> {
        match self {
            Self::Reject => None,
            Self::Continue { cursor } | Self::Match { cursor } | Self::Stutter { cursor } => {
                Some(cursor)
            }
        }
    }
}

/// A validated observation's positive current match or proposed transition paths.
pub enum Projection<A, O = umpire_dsl::Outcome, T = NoTarget> {
    /// Search checked Model paths under a typed observation and positive evidence predicate.
    Reach(T),
    /// Explicit no-edge protocol stutter, requiring the complete unchanged state.
    Stutter(T),
    /// The observation positively matches the current state.
    Current,
    /// Candidate nonempty paths; none matching is an unexplained observation.
    Candidates(Vec<Candidate<A, O>>),
}

impl<A, O, T> Projection<A, O, T> {
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
    /// Typed observation filter; old Current/Candidates consumers use `NoTarget`.
    type Target: ReachTarget<M::State>;

    /// Finite search and whole-prefix retention metadata.
    fn scope(&self) -> Scope {
        Scope::default()
    }

    /// Admits each complete candidate path using actual typed evidence.
    /// Empty-path Match is positive zero-step evidence. A missing predicate admits nothing.
    fn reach_evidence(&self, _observation: &Any, _path: &[&Evidence<M>]) -> EvidenceMatch {
        EvidenceMatch::Reject
    }

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
    ) -> Result<Projection<M::Action, M::Outcome, Self::Target>, protocol::Refusal>;

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
    prefix: Prefix<M>,
}

#[derive(Clone, PartialEq)]
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
        prefix: Prefix<M>,
    },
}

type Prepared<M> = Result<(protocol::response::Outcome, Option<Transaction<M>>), protocol::Refusal>;

struct Replay<M: Machine> {
    state: M::State,
    edges: Vec<Rc<Evidence<M>>>,
    cursors: Option<Vec<usize>>,
}

struct Prefix<M: Machine> {
    edges: Vec<Rc<Evidence<M>>>,
    events: Vec<Rc<Event<M>>>,
}

impl<M: Machine> Default for Prefix<M> {
    fn default() -> Self {
        Self { edges: Vec::new(), events: Vec::new() }
    }
}

struct Event<M: Machine> {
    before: M::State,
    after: M::State,
    start: usize,
    end: usize,
    kind: EventKind<M>,
}

enum EventKind<M: Machine> {
    Observation { actual: Any, continues: bool, cursors: Option<Vec<usize>> },
    Callback { actual: Any, reply: Any, response: M::Action },
    Proposal { request: Any, action: Option<M::Action> },
    Unexplained(Any),
}

impl<M: Machine, R: Realize<M>> Runtime<M, R>
where
    R::Target: SingletonTarget<M::State>,
{
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
            prefix: Prefix::default(),
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
            prefix: Prefix::default(),
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
        match i32::try_from(method).ok().and_then(|value| protocol::Method::try_from(value).ok()) {
            Some(protocol::Method::Explore) => {
                self.explore(decode::<protocol::ExploreRequest>(request)?.budget)
            }
            Some(protocol::Method::Rows) => {
                self.rows(decode::<protocol::RowsRequest>(request)?.state_id)
            }
            Some(protocol::Method::Reset) => {
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
                            prefix: Prefix::default(),
                        },
                    }),
                ))
            }
            Some(protocol::Method::Judge) => {
                let request = decode::<protocol::JudgeRequest>(request)?;
                let observation = Any::required(request.observation)?;
                self.require_realization()?;
                self.judge(&observation)
            }
            Some(protocol::Method::Answer) => {
                let request = decode::<protocol::AnswerRequest>(request)?;
                let callback = Any::required(request.callback)?;
                self.require_realization()?;
                self.answer(&callback)
            }
            Some(protocol::Method::Propose) => {
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
        retainable(observation)?;
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
        self.validate_prefix()?;
        match projection {
            Projection::Reach(target) => {
                if let Some(replay) = self.reach(observation, current, &target)? {
                    return self.explained(observation, replay.state, replay.edges, replay.cursors);
                }
            }
            Projection::Stutter(target) => {
                if target.matches(current) {
                    return self.explained(observation, current.clone(), Vec::new(), None);
                }
            }
            Projection::Current => {
                return self.explained(observation, current.clone(), Vec::new(), None);
            }
            Projection::Candidates(paths) => {
                for candidate in paths {
                    if candidate.path.is_empty() {
                        continue;
                    }
                    if let Some(replay) = self.replay(current, candidate.path)?
                        && candidate.outcome.as_ref().is_none_or(|expected| {
                            replay.edges.last().is_some_and(|edge| expected == &edge.outcome)
                        })
                    {
                        return self.explained(
                            observation,
                            replay.state,
                            replay.edges,
                            replay.cursors,
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
                    prefix: self.extend_prefix(
                        current.clone(),
                        Vec::new(),
                        EventKind::Unexplained(observation.clone()),
                    )?,
                },
            }),
        ))
    }

    fn explained(
        &self,
        observation: &Any,
        state: M::State,
        edges: Vec<Rc<Evidence<M>>>,
        cursors: Option<Vec<usize>>,
    ) -> Prepared<M> {
        let continues = self.pending.is_some()
            && self.realization.continue_polling(
                observation,
                &state,
                self.pending.as_ref().and_then(|pending| pending.action.as_ref()),
            )?;
        let pending = if continues { self.pending.clone() } else { None };
        let prefix = self.extend_prefix(
            state.clone(),
            edges,
            EventKind::Observation { actual: observation.clone(), continues, cursors },
        )?;
        let explanation = Self::explanation(&prefix)?;
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
                    prefix,
                },
            }),
        ))
    }

    fn propose(&self, seed: u64) -> Prepared<M>
    where
        M::Outcome: PartialEq,
    {
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
        retainable(&request)?;
        self.validate_prefix()?;
        let prefix = self.extend_prefix(
            current.clone(),
            Vec::new(),
            EventKind::Proposal { request: request.clone(), action: pending_action.clone() },
        )?;
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
                    prefix,
                },
            }),
        ))
    }

    fn answer(&self, callback: &Any) -> Prepared<M>
    where
        M::Outcome: PartialEq,
    {
        if !self.admitted {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "bootstrap admission required",
            ));
        }
        let current = self.current()?;
        retainable(callback)?;
        if self.frozen.is_some() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "run is frozen until Reset",
            ));
        }
        self.validate_prefix()?;
        let plan = self.realization.answer(callback, current)?;
        let Replay { state, edges, .. } =
            self.replay(current, plan.delivery)?.ok_or_else(|| {
                protocol::Refusal::new(
                    protocol::RefusalCode::State,
                    "callback delivery is not enabled",
                )
            })?;
        if plan.reply.type_url.is_empty() {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::MalformedRequest,
                "reply type is missing",
            ));
        }
        if !self.catalog()?.contains(&plan.response_action)
            || self.checked_step(&state, plan.response_action.clone())?.is_empty()
        {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::State,
                "selected response is not enabled",
            ));
        }
        retainable(&plan.reply)?;
        let prefix = self.extend_prefix(
            state.clone(),
            edges,
            EventKind::Callback {
                actual: callback.clone(),
                reply: plan.reply.clone(),
                response: plan.response_action,
            },
        )?;
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
                    prefix,
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
        let mut edges = Vec::new();
        reserve(&mut edges, path.len())?;
        let catalog = self.catalog()?;
        for transition in path {
            if !catalog.contains(&transition.action) {
                return Ok(None);
            }
            let Some((outcome, facts, target)) = self
                .checked_step(&state, transition.action.clone())?
                .into_iter()
                .nth(transition.choice)
            else {
                return Ok(None);
            };
            let edge = Rc::new(Evidence {
                source: state,
                action: transition.action,
                choice: transition.choice,
                outcome,
                facts,
                target: target.clone(),
            });
            state = target;
            edges.push(edge);
        }
        Ok(Some(Replay { state, edges, cursors: None }))
    }

    fn reach(
        &self,
        observation: &Any,
        current: &M::State,
        target: &R::Target,
    ) -> Result<Option<Replay<M>>, protocol::Refusal> {
        let scope = self.realization.scope();
        let mut traversal = Traversal::new(scope.depth, scope.work);
        let admission = self.realization.reach_evidence(observation, &[]);
        let Some(root_cursor) = admission.cursor() else {
            return Ok(None);
        };
        traversal.root((current.clone(), root_cursor)).map_err(search_error)?;
        if matches!(admission, EvidenceMatch::Match { .. }) && target.matches(current) {
            return Ok(Some(Replay {
                state: current.clone(),
                edges: Vec::new(),
                cursors: Some(vec![root_cursor]),
            }));
        }
        let catalog = self.catalog()?;
        let mut cut = false;
        while let Some(id) = traversal.next() {
            let state = traversal.label(id).position.0.clone();
            for action in &catalog {
                traversal.charge().map_err(search_error)?;
                for (choice, (outcome, facts, next)) in
                    self.checked_step(&state, action.clone())?.into_iter().enumerate()
                {
                    traversal.charge().map_err(search_error)?;
                    let edge = Rc::new(Evidence {
                        source: state.clone(),
                        action: action.clone(),
                        choice,
                        outcome,
                        facts,
                        target: next.clone(),
                    });
                    let mut edges = Vec::new();
                    let mut ancestor = id;
                    let mut cursors = Vec::new();
                    loop {
                        reserve(&mut cursors, 1)?;
                        cursors.push(traversal.label(ancestor).position.1);
                        let Some((parent, _)) = &traversal.label(ancestor).parent else {
                            break;
                        };
                        ancestor = *parent;
                    }
                    cursors.reverse();
                    let mut ancestor = id;
                    while let Some((parent, emission)) = &traversal.label(ancestor).parent {
                        if let Some(emission) = emission {
                            reserve(&mut edges, 1)?;
                            edges.push(Rc::clone(emission));
                        }
                        ancestor = *parent;
                    }
                    edges.reverse();
                    reserve(&mut edges, 1)?;
                    edges.push(Rc::clone(&edge));
                    let mut refs = Vec::new();
                    reserve(&mut refs, edges.len())?;
                    refs.extend(edges.iter().map(Rc::as_ref));
                    let admission = self.realization.reach_evidence(observation, &refs);
                    let cursor = match admission {
                        EvidenceMatch::Reject => continue,
                        EvidenceMatch::Stutter { .. } if edge.source != edge.target => continue,
                        EvidenceMatch::Continue { cursor }
                        | EvidenceMatch::Match { cursor }
                        | EvidenceMatch::Stutter { cursor } => cursor,
                    };
                    reserve(&mut cursors, 1)?;
                    cursors.push(cursor);
                    if edges.len() > scope.depth {
                        cut = true;
                        continue;
                    }
                    if matches!(
                        admission,
                        EvidenceMatch::Match { .. } | EvidenceMatch::Stutter { .. }
                    ) && target.matches(&next)
                    {
                        return Ok(Some(Replay { state: next, edges, cursors: Some(cursors) }));
                    }
                    traversal.push(id, (next, cursor), Some(edge)).map_err(search_error)?;
                }
            }
        }
        if cut {
            return Err(search_error(SearchLimit::Depth { limit: scope.depth }));
        }
        Ok(None)
    }

    fn copy_prefix(&self, additional: usize) -> Result<Prefix<M>, protocol::Refusal> {
        let limit = self.realization.scope().prefix;
        if self
            .prefix
            .edges
            .len()
            .checked_add(self.prefix.events.len())
            .and_then(|count| count.checked_add(additional))
            .is_none_or(|count| count > limit)
        {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::Budget,
                format!("prefix bound {limit}"),
            ));
        }
        let mut prefix = Prefix::default();
        reserve(&mut prefix.edges, self.prefix.edges.len() + additional)?;
        reserve(&mut prefix.events, self.prefix.events.len() + usize::from(additional > 0))?;
        prefix.edges.extend(self.prefix.edges.iter().map(Rc::clone));
        prefix.events.extend(self.prefix.events.iter().map(Rc::clone));
        Ok(prefix)
    }

    fn extend_prefix(
        &self,
        state: M::State,
        edges: Vec<Rc<Evidence<M>>>,
        kind: EventKind<M>,
    ) -> Result<Prefix<M>, protocol::Refusal> {
        let mut prefix =
            self.copy_prefix(edges.len().checked_add(1).ok_or_else(allocation_error)?)?;
        let start = prefix.edges.len();
        prefix.edges.extend(edges);
        prefix.events.push(Rc::new(Event {
            before: self.current()?.clone(),
            after: state,
            start,
            end: prefix.edges.len(),
            kind,
        }));
        Ok(prefix)
    }

    fn explanation(prefix: &Prefix<M>) -> Result<protocol::Explanation, protocol::Refusal> {
        let mut explanation = protocol::Explanation::default();
        reserve(&mut explanation.steps, prefix.edges.len())?;
        reserve(&mut explanation.outcomes, prefix.edges.len())?;
        for edge in &prefix.edges {
            explanation.steps.push(edge.action.to_string());
            explanation.outcomes.push(format!("{:?}", edge.outcome));
        }
        Ok(explanation)
    }

    fn validate_prefix(&self) -> Result<(), protocol::Refusal>
    where
        M::Outcome: PartialEq,
    {
        let mut state = self.initial.as_ref().ok_or_else(|| {
            protocol::Refusal::new(protocol::RefusalCode::State, "initial state missing")
        })?;
        let mut pending: Option<Pending<M::Action>> = None;
        for event in &self.prefix.events {
            if state != &event.before {
                return Err(prefix_error());
            }
            let edges = &self.prefix.edges[event.start..event.end];
            for edge in edges {
                if state != &edge.source {
                    return Err(prefix_error());
                }
                state = &edge.target;
            }
            if state != &event.after {
                return Err(prefix_error());
            }
            match &event.kind {
                EventKind::Observation { actual, continues, cursors } => {
                    if let Some(request) = &pending {
                        self.realization.matches(&request.request, actual)?;
                    }
                    let valid = self.valid_observation(actual, event, edges, cursors.as_deref())?;
                    let expected = pending.is_some()
                        && self.realization.continue_polling(
                            actual,
                            &event.after,
                            pending.as_ref().and_then(|pending| pending.action.as_ref()),
                        )?;
                    if !valid || expected != *continues {
                        return Err(prefix_error());
                    }
                    if !continues {
                        pending = None;
                    }
                }
                EventKind::Callback { actual, reply, response } => {
                    let plan = self.realization.answer(actual, &event.before)?;
                    if plan.reply != *reply
                        || plan.response_action != *response
                        || plan.delivery.len() != edges.len()
                        || !plan.delivery.iter().zip(edges).all(|(transition, edge)| {
                            transition.action == edge.action && transition.choice == edge.choice
                        })
                    {
                        return Err(prefix_error());
                    }
                }
                EventKind::Unexplained(actual) => {
                    self.realization.observe(actual, &event.before)?;
                    return Err(prefix_error());
                }
                EventKind::Proposal { request, action } => {
                    if pending.is_some() || !edges.is_empty() {
                        return Err(prefix_error());
                    }
                    pending = Some(Pending { request: request.clone(), action: action.clone() });
                }
            }
        }
        if self.state.as_ref() != Some(state) || self.pending != pending {
            return Err(prefix_error());
        }
        Ok(())
    }

    fn valid_observation(
        &self,
        actual: &Any,
        event: &Event<M>,
        edges: &[Rc<Evidence<M>>],
        cursors: Option<&[usize]>,
    ) -> Result<bool, protocol::Refusal>
    where
        M::Outcome: PartialEq,
    {
        let mut refs = Vec::new();
        reserve(&mut refs, edges.len())?;
        refs.extend(edges.iter().map(Rc::as_ref));
        Ok(match self.realization.observe(actual, &event.before)? {
            Projection::Current => edges.is_empty(),
            Projection::Stutter(target) => edges.is_empty() && target.matches(&event.before),
            Projection::Candidates(candidates) => candidates.iter().any(|candidate| {
                !candidate.path.is_empty()
                    && candidate.path.len() == edges.len()
                    && candidate.path.iter().zip(edges).all(|(transition, edge)| {
                        transition.action == edge.action && transition.choice == edge.choice
                    })
                    && candidate.outcome.as_ref().is_none_or(|expected| {
                        edges.last().is_some_and(|edge| expected == &edge.outcome)
                    })
            }),
            Projection::Reach(target) => {
                let mut valid = cursors.is_some_and(|cursors| {
                    cursors.len() == refs.len() + 1
                        && self.realization.reach_evidence(actual, &[]).cursor()
                            == cursors.first().copied()
                });
                for position in 0..refs.len() {
                    let evidence = self.realization.reach_evidence(actual, &refs[..=position]);
                    valid &= cursors.and_then(|cursors| cursors.get(position + 1)).copied()
                        == evidence.cursor()
                        && evidence != EvidenceMatch::Reject
                        && (!matches!(evidence, EvidenceMatch::Stutter { .. })
                            || refs[position].source == refs[position].target);
                }
                valid
                    && target.matches(&event.after)
                    && matches!(
                        self.realization.reach_evidence(actual, &refs),
                        EvidenceMatch::Match { .. } | EvidenceMatch::Stutter { .. }
                    )
            }
        })
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
    R::Target: SingletonTarget<M::State>,
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
            Change::Run { state, frozen, pending, admitted, prefix } => {
                self.prefix = prefix;
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

fn allocation_error() -> protocol::Refusal {
    protocol::Refusal::new(protocol::RefusalCode::Allocation, "prefix allocation failed")
}

fn reserve<T>(values: &mut Vec<T>, count: usize) -> Result<(), protocol::Refusal> {
    values.try_reserve(count).map_err(|_| allocation_error())
}

fn prefix_error() -> protocol::Refusal {
    protocol::Refusal::new(protocol::RefusalCode::State, "retained prefix no longer validates")
}

fn search_error(error: SearchLimit) -> protocol::Refusal {
    let code = if matches!(error, SearchLimit::Allocation) {
        protocol::RefusalCode::Allocation
    } else {
        protocol::RefusalCode::Budget
    };
    protocol::Refusal::new(code, format!("Reach {error}"))
}

fn retainable(value: &Any) -> Result<(), protocol::Refusal> {
    if value.encoded_len() > 1 << 20 {
        return Err(protocol::Refusal::new(
            protocol::RefusalCode::Allocation,
            "prefix payload exceeds allocation bound",
        ));
    }
    Ok(())
}
