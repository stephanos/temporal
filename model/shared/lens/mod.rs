//! One publication transaction owner for configured Temporal entities.

pub mod config;

use crate::support::Observed;
use prost::Message;
use umpire_runtime::{
    AnyMessage as _, Dispatch,
    prost_types::Any,
    protocol::{self, Refusal, RefusalCode, response::Outcome},
};

type ModelTransaction<B> = <<B as EntityBinding>::Model as Dispatch>::Transaction;
type Prepared<B> = Result<(Outcome, Option<Transaction<B>>), Refusal>;

/// Feature policies and typed projections used by the shared entity lifecycle.
pub trait EntityBinding {
    /// The independently owned Model runtime, retaining its existing transaction.
    type Model: Dispatch;
    /// Immutable evidence admitted by the bootstrap policy.
    type Receipt;
    /// Bootstrap RPC outside the Model action graph.
    const BOOTSTRAP: &'static str;
    /// Create an unconfigured Model with its original defaults.
    fn fresh(&self) -> Self::Model;
    /// Validate configuration and assemble its declared worker program.
    fn configure(&self, configuration: &Any) -> Result<Any, Refusal>;
    /// Fill a bootstrap using the admitted configuration and seed.
    fn bootstrap(&self, configuration: &Any, seed: u64) -> Result<Any, Refusal>;
    /// Decode the actual bootstrap request before pending-request correlation.
    fn bootstrap_request(&self, actual: &Any) -> Result<Any, Refusal>;
    /// Decode and admit a new bootstrap into a detached prospective runtime.
    fn admit(&self, actual: &Any) -> Result<Admission<Self>, Refusal>
    where
        Self: Sized;
    /// Validate an actual replay against immutable admitted evidence.
    fn replay(&self, receipt: &Self::Receipt, actual: &Any) -> Result<(), Refusal>;
    /// Validate complete actual evidence before delegating to the feature.
    fn validate(&self, receipt: Option<&Self::Receipt>, actual: &Observed) -> Result<(), Refusal>;
    /// Check admission after feature validation and the retained freeze refusal.
    fn ready(&self, receipt: &Self::Receipt) -> Result<(), Refusal>;
    /// Preserve replay-specific admission refusal after the freeze check.
    fn ready_replay(&self, receipt: &Self::Receipt) -> Result<(), Refusal> {
        self.ready(receipt)
    }
    /// Install proposal context derived only from the committed receipt.
    fn install(&self, model: &mut Self::Model, receipt: &Self::Receipt);
    /// Identify a declared compound instruction without interpreting its policy.
    fn instruction(&self, request: &Any) -> bool;
    /// Preserve the feature's current missing-admission diagnostic.
    fn missing_receipt(&self) -> Refusal;
}

/// Detached bootstrap effects, never installed until response publication succeeds.
pub enum Admission<B: EntityBinding> {
    /// A successful first judgment and its prospective Model.
    Admitted {
        /// Prospective independent runtime with its first judgment installed privately.
        model: B::Model,
        /// Actual immutable bootstrap provenance.
        receipt: B::Receipt,
        /// Response to publish before installing the runtime.
        outcome: Outcome,
    },
    /// Actual provenance retained with a named unsupported-domain refusal.
    Frozen {
        /// Actual immutable bootstrap provenance.
        receipt: B::Receipt,
        /// Domain refusal retained until Reset.
        refusal: Refusal,
    },
}

/// Entity admission, pending correlation, freeze and Reset share one commit owner.
pub struct EntityLens<B: EntityBinding> {
    binding: B,
    model: B::Model,
    receipt: Option<B::Receipt>,
    frozen: Option<Refusal>,
    configuration: Option<Any>,
    bootstrap: Option<Any>,
    pending: Option<Any>,
}

/// An opaque prospective entity change published by the existing guest transaction.
pub struct Transaction<B: EntityBinding> {
    change: Change<B>,
}

enum Change<B: EntityBinding> {
    Model(ModelTransaction<B>),
    Pending { model: ModelTransaction<B>, request: Option<Any> },
    Admit { model: Box<B::Model>, receipt: B::Receipt },
    Freeze { receipt: B::Receipt, refusal: Refusal },
    Reset(ModelTransaction<B>),
    Configure(Any),
    Propose(Any),
}

impl<B: EntityBinding> EntityLens<B> {
    /// Create an independent run without an admitted entity or receipt.
    pub fn new(binding: B) -> Self {
        let model = binding.fresh();
        Self {
            binding,
            model,
            receipt: None,
            frozen: None,
            configuration: None,
            bootstrap: None,
            pending: None,
        }
    }

    fn prepare_method(&self, method: u32, bytes: &[u8]) -> Prepared<B> {
        match i32::try_from(method).ok().and_then(|value| protocol::Method::try_from(value).ok()) {
            Some(protocol::Method::Reset) => {
                let request = protocol::ResetRequest::decode(bytes)
                    .map_err(|_| Refusal::malformed("malformed Reset request"))?;
                let configuration = request.configuration.or_else(|| self.configuration.clone());
                if let Some(configuration) = configuration {
                    let program = self.binding.configure(&configuration)?;
                    return Ok((
                        Outcome::Reset(protocol::ResetResponse::current(Some(program))),
                        Some(Transaction { change: Change::Configure(configuration) }),
                    ));
                }
                self.delegate_plain(method, bytes, true)
            }
            Some(protocol::Method::Judge) => self.judge(bytes),
            Some(protocol::Method::Answer) => {
                Err(Refusal::unsupported("Temporal workers follow the Reset program"))
            }
            Some(protocol::Method::Propose) => self.propose(bytes),
            _ => self.delegate_plain(method, bytes, false),
        }
    }

    fn delegate_plain(&self, method: u32, bytes: &[u8], reset: bool) -> Prepared<B> {
        let (response, transaction) = self.model.prepare(method, bytes);
        let outcome = response.outcome.ok_or_else(|| {
            state(if reset { "Reset response is missing" } else { "Model response is missing" })
        })?;
        Ok((
            outcome,
            transaction.map(|transaction| Transaction {
                change: if reset { Change::Reset(transaction) } else { Change::Model(transaction) },
            }),
        ))
    }

    fn judge(&self, bytes: &[u8]) -> Prepared<B> {
        let request = protocol::JudgeRequest::decode(bytes)
            .map_err(|_| Refusal::malformed("malformed Judge request"))?;
        let value = Any::required(request.observation)?;
        let observed = Observed::new(&value)?;
        if observed.acknowledged().is_some() && self.pending.is_none() {
            return Err(state("compound evidence requires a pending cancellation"));
        }
        if observed.method == B::BOOTSTRAP {
            return self.admit(&value);
        }
        self.binding.validate(self.receipt.as_ref(), &observed)?;
        let receipt = self.receipt.as_ref().ok_or_else(|| self.binding.missing_receipt())?;
        let prepared = self.delegate(protocol::Method::Judge as u32, bytes)?;
        if let Outcome::Refusal(refusal) = &prepared.0 {
            return Err(refusal.clone());
        }
        if let Some(frozen) = &self.frozen {
            return Err(frozen.clone());
        }
        self.binding.ready(receipt)?;
        Ok(prepared)
    }

    fn propose(&self, bytes: &[u8]) -> Prepared<B> {
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
            return self.delegate(protocol::Method::Propose as u32, bytes);
        }
        let request = self.binding.bootstrap(configuration, request.seed)?;
        Ok((
            Outcome::Propose(protocol::ProposeResponse {
                request: Some(request.clone()),
                action: String::new(),
                kind: protocol::ProposalKind::Bootstrap as i32,
            }),
            Some(Transaction { change: Change::Propose(request) }),
        ))
    }

    fn delegate(&self, method: u32, bytes: &[u8]) -> Prepared<B> {
        let (response, transaction) = self.model.prepare(method, bytes);
        let outcome = response.outcome.ok_or_else(|| state("Model response is missing"))?;
        let pending = match &outcome {
            Outcome::Propose(proposal) => proposal
                .request
                .as_ref()
                .filter(|request| self.binding.instruction(request))
                .cloned(),
            Outcome::Judge(judgment) if !judgment.continue_polling => None,
            _ => self.pending.clone(),
        };
        Ok((
            outcome,
            transaction
                .map(|model| Transaction { change: Change::Pending { model, request: pending } }),
        ))
    }

    fn admit(&self, value: &Any) -> Prepared<B> {
        let request = self.binding.bootstrap_request(value)?;
        if self.configuration.is_some()
            && self.receipt.is_none()
            && self.bootstrap.as_ref() != Some(&request)
        {
            return Err(state("Start actual request does not match pending bootstrap"));
        }
        if let Some(receipt) = &self.receipt {
            self.binding.replay(receipt, value)?;
            if let Some(frozen) = &self.frozen {
                return Err(frozen.clone());
            }
            self.binding.ready_replay(receipt)?;
            return self.delegate(
                protocol::Method::Judge as u32,
                &protocol::JudgeRequest { observation: Some(value.clone()) }.encode_to_vec(),
            );
        }
        Ok(match self.binding.admit(value)? {
            Admission::Admitted { model, receipt, outcome } => (
                outcome,
                Some(Transaction { change: Change::Admit { model: Box::new(model), receipt } }),
            ),
            Admission::Frozen { receipt, refusal } => (
                Outcome::Refusal(refusal.clone()),
                Some(Transaction { change: Change::Freeze { receipt, refusal } }),
            ),
        })
    }
}

impl<B: EntityBinding> Dispatch for EntityLens<B> {
    type Transaction = Transaction<B>;
    fn prepare(&self, method: u32, request: &[u8]) -> (protocol::Response, Option<Transaction<B>>) {
        let (outcome, transaction) = self
            .prepare_method(method, request)
            .unwrap_or_else(|refusal| (Outcome::Refusal(refusal), None));
        (protocol::Response { outcome: Some(outcome) }, transaction)
    }
    fn commit(&mut self, transaction: Transaction<B>) {
        match transaction.change {
            Change::Model(transaction) => self.model.commit(transaction),
            Change::Pending { model, request } => {
                self.model.commit(model);
                self.pending = request;
            }
            Change::Admit { model, receipt } => {
                self.model = *model;
                self.receipt = Some(receipt);
                self.bootstrap = None;
                if self.configuration.is_some()
                    && let Some(receipt) = &self.receipt
                {
                    self.binding.install(&mut self.model, receipt);
                }
            }
            Change::Freeze { receipt, refusal } => {
                self.receipt = Some(receipt);
                self.frozen = Some(refusal);
            }
            Change::Configure(configuration) => {
                self.model = self.binding.fresh();
                self.configuration = Some(configuration);
                self.receipt = None;
                self.bootstrap = None;
                self.pending = None;
                self.frozen = None;
            }
            Change::Propose(request) => self.bootstrap = Some(request),
            Change::Reset(transaction) => {
                self.model.commit(transaction);
                self.pending = None;
                self.frozen = None;
            }
        }
    }
}

fn state(message: impl Into<String>) -> Refusal {
    Refusal::new(RefusalCode::State, message)
}
