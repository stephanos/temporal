//! Proposal publication and pending-run controls.
use prost::Message;
use umpire_runtime::{
    Guest,
    protocol::{self, response::Outcome},
};
#[path = "support/control.rs"]
mod control;
mod support;

fn refused(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(value)) = response.outcome else {
        panic!("expected refusal: {response:?}")
    };
    assert_eq!(value.code(), code);
}

#[test]
fn proposal_is_transactional_and_pending_until_final_observation() {
    let mut guest = Guest::new(support::runtime());
    let mut proposal = None;
    assert!(!guest.dispatch_to(6, &[8, 1], |bytes| {
        proposal =
            Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
        false
    }));
    let proposal = proposal.unwrap_or_else(|| panic!("proposal missing"));
    assert!(
        !matches!(proposal.outcome, Some(Outcome::Refusal(_))),
        "Propose missing: {proposal:?}"
    );
    assert!(guest.dispatch_to(6, &[8, 1], |bytes| {
        assert_eq!(
            protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")),
            proposal
        );
        true
    }));
    assert!(guest.dispatch_to(6, &[8, 1], |bytes| {
        refused(
            protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")),
            protocol::RefusalCode::State,
        );
        true
    }));
    let observed = protocol::JudgeRequest { observation: Some(support::any(1)) }.encode_to_vec();
    assert!(guest.dispatch_to(4, &observed, |_| true));
    assert!(guest.dispatch_to(6, &[8, 1], |bytes| {
        assert!(!matches!(
            protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")).outcome,
            Some(Outcome::Refusal(_))
        ));
        true
    }));
}

fn call(
    guest: &mut Guest<impl umpire_runtime::Dispatch>,
    method: u32,
    request: &impl Message,
) -> protocol::Response {
    let mut response = None;
    assert!(guest.dispatch_to(method, &request.encode_to_vec(), |bytes| {
        response =
            Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
        true
    }));
    response.unwrap_or_else(|| panic!("response missing"))
}

#[test]
fn bootstrap_configuration_admission_and_polling_publish_atomically() {
    let mut guest = Guest::new(control::control());
    let reset = protocol::ResetRequest { configuration: Some(support::any(0)) };
    let propose = protocol::ProposeRequest { seed: 7 };
    assert!(!guest.dispatch_to(3, &reset.encode_to_vec(), |_| false));
    let Some(Outcome::Propose(before)) = call(&mut guest, 6, &propose).outcome else {
        panic!("proposal")
    };
    assert_eq!(before.kind(), protocol::ProposalKind::Action);
    call(&mut guest, 3, &reset);
    refused(
        call(&mut guest, 5, &protocol::AnswerRequest { callback: Some(support::any(4)) }),
        protocol::RefusalCode::State,
    );
    let Some(Outcome::Propose(bootstrap)) = call(&mut guest, 6, &propose).outcome else {
        panic!("bootstrap")
    };
    assert_eq!(bootstrap.kind(), protocol::ProposalKind::Bootstrap);
    assert_eq!(bootstrap.action, "");
    let start = protocol::JudgeRequest { observation: Some(support::any(0)) };
    assert!(!guest.dispatch_to(4, &start.encode_to_vec(), |_| false));
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    refused(
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(1)) }),
        protocol::RefusalCode::State,
    );
    call(&mut guest, 4, &start);
    let Some(Outcome::Propose(action)) = call(&mut guest, 6, &propose).outcome else {
        panic!("action")
    };
    assert_eq!(action.action, "Deliver");
    let poll = protocol::JudgeRequest { observation: Some(support::any(20)) };
    let Some(Outcome::Judge(intermediate)) = call(&mut guest, 4, &poll).outcome else {
        panic!("poll")
    };
    assert!(intermediate.continue_polling);
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    call(&mut guest, 5, &protocol::AnswerRequest { callback: Some(support::any(4)) });
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let Some(Outcome::Judge(final_result)) =
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(2)) }).outcome
    else {
        panic!("final")
    };
    assert!(!final_result.continue_polling);
    call(&mut guest, 6, &propose);
    call(&mut guest, 3, &protocol::ResetRequest::default());
    let Some(Outcome::Propose(again)) = call(&mut guest, 6, &propose).outcome else {
        panic!("bootstrap after Reset")
    };
    assert_eq!(again, bootstrap);
}

struct Disabled;
impl umpire_runtime::Realize<support::World> for Disabled {
    fn propose(
        &self,
        _: &support::State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<support::Action>, protocol::Refusal> {
        Ok(umpire_runtime::Proposal::Action {
            action: support::Action::Respond,
            request: support::any(2),
        })
    }
    fn observe(
        &self,
        value: &prost_types::Any,
        state: &support::State,
    ) -> Result<umpire_runtime::Projection<support::Action, support::Outcome>, protocol::Refusal>
    {
        umpire_runtime::Realize::observe(&support::Realization, value, state)
    }
    fn answer(
        &self,
        value: &prost_types::Any,
        state: &support::State,
    ) -> Result<umpire_runtime::AnswerPlan<support::Action>, protocol::Refusal> {
        umpire_runtime::Realize::answer(&support::Realization, value, state)
    }
}
#[test]
fn disabled_proposal_refuses_without_committing_and_quiescent_is_distinct() {
    let mut disabled = umpire_runtime::Runtime::new(support::World { starts: 1 }, Disabled);
    refused(disabled.dispatch(6, &[8, 1]), protocol::RefusalCode::State);
    let observed = protocol::JudgeRequest { observation: Some(support::any(1)) }.encode_to_vec();
    assert!(matches!(disabled.dispatch(4, &observed).outcome, Some(Outcome::Judge(_))));
    assert!(matches!(disabled.dispatch(6, &[8, 1]).outcome, Some(Outcome::Propose(_))));
    let mut runtime = support::runtime();
    for value in [10, 3] {
        runtime.dispatch(
            4,
            &protocol::JudgeRequest { observation: Some(support::any(value)) }.encode_to_vec(),
        );
    }
    refused(runtime.dispatch(6, &[8, 1]), protocol::RefusalCode::Quiescent);
}

#[test]
fn selected_initial_state_still_requires_declared_bootstrap() {
    let mut runtime = umpire_runtime::Runtime::with_initial(
        support::World { starts: 1 },
        control::ControlRealization { bootstrap: true },
        support::State(0),
    )
    .unwrap_or_else(|error| panic!("{error:?}"));
    refused(
        runtime.dispatch(
            5,
            &protocol::AnswerRequest { callback: Some(support::any(4)) }.encode_to_vec(),
        ),
        protocol::RefusalCode::State,
    );
}

#[test]
fn unpublished_bootstrap_and_unexplained_pending_effects_roll_back() {
    for panic in [false, true] {
        let mut guest = Guest::new(control::control());
        call(&mut guest, 3, &protocol::ResetRequest { configuration: Some(support::any(0)) });
        let request = protocol::ProposeRequest { seed: 1 };
        let failed = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            guest.dispatch_to(6, &request.encode_to_vec(), |_| {
                assert!(!panic, "publisher failed");
                false
            })
        }));
        assert_eq!(failed.is_err(), panic);
        assert!(matches!(call(&mut guest, 6, &request).outcome, Some(Outcome::Propose(_))));
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(0)) });
        call(&mut guest, 6, &request);
        let impossible = protocol::JudgeRequest { observation: Some(support::any(2)) };
        assert!(!guest.dispatch_to(4, &impossible.encode_to_vec(), |_| false));
        let Some(Outcome::Judge(unexplained)) = call(&mut guest, 4, &impossible).outcome else {
            panic!("Unexplained")
        };
        assert!(matches!(
            unexplained.judgment,
            Some(protocol::judge_response::Judgment::Unexplained(_))
        ));
        assert!(!unexplained.continue_polling);
        refused(call(&mut guest, 6, &request), protocol::RefusalCode::State);
        call(&mut guest, 3, &protocol::ResetRequest::default());
        let Some(Outcome::Propose(again)) = call(&mut guest, 6, &request).outcome else {
            panic!("reset")
        };
        assert_eq!(again.kind(), protocol::ProposalKind::Bootstrap);
    }
}

struct InternalWorld {
    external: bool,
}
impl umpire_dsl::Machine for InternalWorld {
    type State = support::State;
    type Action = support::Action;
    type Outcome = support::Outcome;
    type Fact = support::Fact;
    fn starts(&self) -> Vec<Self::State> {
        vec![support::State(0)]
    }
    fn actions(&self) -> Vec<Self::Action> {
        if self.external {
            vec![support::Action::Deliver, support::Action::Finish]
        } else {
            vec![support::Action::Deliver]
        }
    }
    fn internal(&self, action: &Self::Action) -> bool {
        *action == support::Action::Deliver
    }
    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact> {
        if self.external && action == support::Action::Finish {
            let mut steps = umpire_dsl::Steps::new();
            steps.push((support::Outcome(3), vec![support::Fact], support::State(3)));
            return steps;
        }
        umpire_dsl::Machine::step(&support::World { starts: 1 }, state, action)
    }
}
struct InternalRealization;
impl umpire_runtime::Realize<InternalWorld> for InternalRealization {
    fn propose(
        &self,
        _: &support::State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<support::Action>, protocol::Refusal> {
        Ok(umpire_runtime::Proposal::Action {
            action: support::Action::Deliver,
            request: support::any(1),
        })
    }
    fn observe(
        &self,
        _: &prost_types::Any,
        _: &support::State,
    ) -> Result<umpire_runtime::Projection<support::Action, support::Outcome>, protocol::Refusal>
    {
        Ok(umpire_runtime::Projection::Current)
    }
    fn answer(
        &self,
        _: &prost_types::Any,
        _: &support::State,
    ) -> Result<umpire_runtime::AnswerPlan<support::Action>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("no callbacks"))
    }
}
#[test]
fn enabled_internal_action_is_never_a_proposal() {
    for (external, code) in
        [(false, protocol::RefusalCode::Quiescent), (true, protocol::RefusalCode::State)]
    {
        let mut runtime =
            umpire_runtime::Runtime::new(InternalWorld { external }, InternalRealization);
        refused(runtime.dispatch(6, &protocol::ProposeRequest { seed: 1 }.encode_to_vec()), code);
    }
}

struct DefaultProposal;
impl umpire_runtime::Realize<support::World> for DefaultProposal {
    fn observe(
        &self,
        value: &prost_types::Any,
        state: &support::State,
    ) -> Result<umpire_runtime::Projection<support::Action, support::Outcome>, protocol::Refusal>
    {
        umpire_runtime::Realize::observe(&support::Realization, value, state)
    }
    fn answer(
        &self,
        value: &prost_types::Any,
        state: &support::State,
    ) -> Result<umpire_runtime::AnswerPlan<support::Action>, protocol::Refusal> {
        umpire_runtime::Realize::answer(&support::Realization, value, state)
    }
}
#[test]
fn runtime_owns_quiescence_before_default_proposal_policy() {
    let mut runtime = umpire_runtime::Runtime::new(support::World { starts: 1 }, DefaultProposal);
    for value in [10, 3] {
        runtime.dispatch(
            4,
            &protocol::JudgeRequest { observation: Some(support::any(value)) }.encode_to_vec(),
        );
    }
    refused(
        runtime.dispatch(6, &protocol::ProposeRequest { seed: 1 }.encode_to_vec()),
        protocol::RefusalCode::Quiescent,
    );
}

struct ReadRealization {
    refuse_program: bool,
    continue_after_progress: bool,
}
impl<M> umpire_runtime::Realize<M> for ReadRealization
where
    M: umpire_dsl::Machine<
            State = support::State,
            Action = support::Action,
            Outcome = support::Outcome,
        >,
{
    fn program(&self) -> Result<prost_types::Any, protocol::Refusal> {
        if self.refuse_program {
            Err(protocol::Refusal::new(protocol::RefusalCode::State, "program refused"))
        } else {
            Ok(support::any(42))
        }
    }
    fn propose(
        &self,
        _: &support::State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<support::Action>, protocol::Refusal> {
        Ok(umpire_runtime::Proposal::Read { request: support::any(99) })
    }
    fn observe(
        &self,
        value: &prost_types::Any,
        state: &support::State,
    ) -> Result<umpire_runtime::Projection<support::Action, support::Outcome>, protocol::Refusal>
    {
        if support::payload(value)? == 20 {
            return Ok(umpire_runtime::Projection::Current);
        }
        umpire_runtime::Realize::observe(&support::Realization, value, state)
    }
    fn matches(
        &self,
        request: &prost_types::Any,
        value: &prost_types::Any,
    ) -> Result<(), protocol::Refusal> {
        assert_eq!(request, &support::any(99));
        if matches!(support::payload(value)?, 1 | 20)
            || self.continue_after_progress && support::payload(value)? == 2
        {
            Ok(())
        } else {
            Err(protocol::Refusal::new(protocol::RefusalCode::State, "crossed read"))
        }
    }
    fn continue_polling(
        &self,
        value: &prost_types::Any,
        _: &support::State,
        action: Option<&support::Action>,
    ) -> Result<bool, protocol::Refusal> {
        assert!(action.is_none());
        Ok(support::payload(value)? == 20
            || self.continue_after_progress && support::payload(value)? == 1)
    }
}

#[test]
fn read_remains_pending_until_correlated_final_explanation_publishes() {
    let mut guest = Guest::new(umpire_runtime::Runtime::new(
        support::World { starts: 1 },
        ReadRealization { refuse_program: false, continue_after_progress: false },
    ));
    let Some(Outcome::Judge(unproposed)) =
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(20)) })
            .outcome
    else {
        panic!("unproposed observation")
    };
    assert!(!unproposed.continue_polling);
    let propose = protocol::ProposeRequest { seed: 7 };
    let Some(Outcome::Propose(read)) = call(&mut guest, 6, &propose).outcome else {
        panic!("read")
    };
    assert_eq!(read.kind(), protocol::ProposalKind::Read);
    assert_eq!(read.action, "");
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    refused(
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(2)) }),
        protocol::RefusalCode::State,
    );
    let Some(Outcome::Judge(intermediate)) =
        call(&mut guest, 4, &protocol::JudgeRequest { observation: Some(support::any(20)) })
            .outcome
    else {
        panic!("intermediate")
    };
    assert!(intermediate.continue_polling);
    assert!(intermediate.judgment.is_some());
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let final_read = protocol::JudgeRequest { observation: Some(support::any(1)) };
    assert!(!guest.dispatch_to(4, &final_read.encode_to_vec(), |_| false));
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let Some(Outcome::Judge(final_result)) = call(&mut guest, 4, &final_read).outcome else {
        panic!("final")
    };
    assert!(!final_result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = final_result.judgment
    else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Deliver"]);
    assert!(matches!(call(&mut guest, 6, &propose).outcome, Some(Outcome::Propose(_))));
}

#[test]
fn read_commits_intermediate_progress_without_consuming_the_pending_request() {
    let mut guest = Guest::new(umpire_runtime::Runtime::new(
        support::World { starts: 1 },
        ReadRealization { refuse_program: false, continue_after_progress: true },
    ));
    let propose = protocol::ProposeRequest { seed: 7 };
    call(&mut guest, 6, &propose);
    let progress = protocol::JudgeRequest { observation: Some(support::any(1)) };
    let mut unpublished = None;
    assert!(!guest.dispatch_to(4, &progress.encode_to_vec(), |bytes| {
        unpublished =
            Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
        false
    }));
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let published = call(&mut guest, 4, &progress);
    assert_eq!(Some(&published), unpublished.as_ref());
    let Some(Outcome::Judge(intermediate)) = published.outcome else { panic!("progress") };
    assert!(intermediate.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = intermediate.judgment
    else {
        panic!("explained progress")
    };
    assert_eq!(explanation.steps, ["Deliver"]);
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let Some(Outcome::Judge(repeated)) = call(&mut guest, 4, &progress).outcome else {
        panic!("committed progress")
    };
    assert!(repeated.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = repeated.judgment else {
        panic!("explained current state")
    };
    assert_eq!(explanation.steps, [] as [String; 0]);
    let final_read = protocol::JudgeRequest { observation: Some(support::any(2)) };
    assert!(!guest.dispatch_to(4, &final_read.encode_to_vec(), |_| false));
    refused(call(&mut guest, 6, &propose), protocol::RefusalCode::State);
    let Some(Outcome::Judge(final_result)) = call(&mut guest, 4, &final_read).outcome else {
        panic!("final")
    };
    assert!(!final_result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = final_result.judgment
    else {
        panic!("explained final state")
    };
    assert_eq!(explanation.steps, ["Respond"]);
    assert!(matches!(call(&mut guest, 6, &propose).outcome, Some(Outcome::Propose(_))));
}

#[test]
fn read_does_not_require_enabled_external_model_action() {
    let mut runtime = umpire_runtime::Runtime::new(
        InternalWorld { external: false },
        ReadRealization { refuse_program: false, continue_after_progress: false },
    );
    let Some(Outcome::Propose(read)) = runtime.dispatch(6, &[]).outcome else { panic!("read") };
    assert_eq!(read.kind(), protocol::ProposalKind::Read);
}

#[test]
fn reset_program_and_refusing_defaults_preserve_run_transactions() {
    let mut legacy = support::runtime();
    let Some(Outcome::Reset(reset)) = legacy.dispatch(3, &[]).outcome else { panic!("reset") };
    assert!(reset.program.is_none());
    let mut runtime = umpire_runtime::Runtime::new(
        support::World { starts: 1 },
        ReadRealization { refuse_program: false, continue_after_progress: false },
    );
    for _ in 0..2 {
        let Some(Outcome::Reset(reset)) = runtime.dispatch(3, &[]).outcome else {
            panic!("reset program")
        };
        assert_eq!(reset.program, Some(support::any(42)));
    }
    refused(
        runtime.dispatch(
            5,
            &protocol::AnswerRequest { callback: Some(support::any(4)) }.encode_to_vec(),
        ),
        protocol::RefusalCode::Unsupported,
    );
    runtime.dispatch(6, &[]);
    runtime.replace_realization(ReadRealization {
        refuse_program: true,
        continue_after_progress: false,
    });
    refused(runtime.dispatch(3, &[]), protocol::RefusalCode::State);
    runtime.replace_realization(ReadRealization {
        refuse_program: false,
        continue_after_progress: false,
    });
    refused(runtime.dispatch(6, &[]), protocol::RefusalCode::State);
}
