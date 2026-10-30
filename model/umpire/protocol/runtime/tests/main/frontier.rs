//! Whole-prefix witnesses retain latent countdown alternatives transactionally.
#![forbid(unsafe_code)]

use crate::support::{
    any,
    frontier::{self, Binding, Fault, GLOBAL, LOCAL, RENEW, World},
};
use prost::Message as _;
use umpire_dsl::Machine as _;
use umpire_runtime::{
    Guest, Runtime, Scope,
    prost_types::Any,
    protocol,
    testing::{GuestExt as _, ResponseExt as _},
};

#[test]
fn countdown_global_and_local_expiry_require_distinct_compatible_pasts() {
    for (actual, steps, outcomes) in [
        (
            GLOBAL,
            vec!["Tick", "Renew", "Tick", "Tick"],
            vec!["Advanced", "Renewed", "Advanced", "GlobalClosed"],
        ),
        (LOCAL, vec!["Renew", "Tick", "Tick"], vec!["Renewed", "Advanced", "PendingClosed"]),
    ] {
        let mut guest = Guest::new(frontier::runtime());
        let earlier = guest.judge(any(RENEW)).explanation();
        assert_eq!(earlier.steps, ["Renew"]);
        assert_eq!(earlier.outcomes, ["Renewed"]);
        let immutable = earlier.encode_to_vec();
        let latest = guest.judge(any(actual)).explanation();
        assert_eq!(latest.steps, steps);
        assert_eq!(latest.outcomes, outcomes);
        assert_eq!(earlier.encode_to_vec(), immutable);
    }
}

#[test]
fn countdown_fixture_global_wins_ties_and_emits_the_exact_expiry_fact() {
    use frontier::{Action, Emission, Outcome, Phase};
    let mut state = World.starts().remove(0);
    for action in [Action::Tick, Action::Renew, Action::Tick] {
        state = World.step(&state, action).remove(0).2;
    }
    assert_eq!((state.global, state.local), (1, 1));
    let (outcome, facts, state) = World.step(&state, Action::Tick).remove(0);
    assert_eq!(outcome, Outcome::GlobalClosed);
    assert_eq!(facts, [Emission::FactGlobal]);
    assert_eq!(state.phase, Phase::GlobalClosed);
    let mut state = World.starts().remove(0);
    for action in [Action::Renew, Action::Tick] {
        state = World.step(&state, action).remove(0).2;
    }
    let (outcome, facts, state) = World.step(&state, Action::Tick).remove(0);
    assert_eq!(outcome, Outcome::PendingClosed);
    assert_eq!(facts, [Emission::FactLocal]);
    assert_eq!(state.phase, Phase::PendingClosed);
}

#[test]
fn countdown_failed_publication_and_allocation_leave_both_continuations_for_retry() {
    for continuation in [GLOBAL, LOCAL] {
        let mut guest = Guest::new(frontier::runtime());
        let first = guest.judge(any(RENEW)).explanation();
        let immutable = first.encode_to_vec();
        guest.propose(0);
        guest
            .judge(Any { type_url: crate::support::TYPE.into(), value: vec![0; 1 << 20] })
            .refusal(protocol::RefusalCode::Allocation);
        let request =
            protocol::JudgeRequest { observation: Some(any(continuation)) }.encode_to_vec();
        let mut unpublished = None;
        assert!(!guest.dispatch_to(protocol::Method::Judge as u32, &request, |bytes| {
            unpublished = Some(
                protocol::Response::decode(bytes)
                    .unwrap_or_else(|error| panic!("{error}"))
                    .explanation(),
            );
            false
        }));
        assert_eq!(
            guest.judge(any(continuation)).explanation(),
            unpublished.unwrap_or_else(|| panic!("missing prospective report"))
        );
        assert_eq!(first.encode_to_vec(), immutable);
    }
}

#[test]
fn countdown_failed_reset_preserves_frontier_and_published_reset_clears_it() {
    let mut guest = Guest::new(frontier::runtime());
    guest.judge(any(RENEW)).explanation();
    assert!(!guest.dispatch_to(protocol::Method::Reset as u32, &[], |_| false));
    assert_eq!(guest.judge(any(GLOBAL)).explanation().steps, ["Tick", "Renew", "Tick", "Tick"]);
    guest.reset(None);
    assert_eq!(guest.judge(any(RENEW)).explanation().steps, ["Renew"]);
    assert_eq!(guest.judge(any(LOCAL)).explanation().steps, ["Renew", "Tick", "Tick"]);
}

#[test]
fn countdown_finite_bounds_refuse_deterministically_and_preserve_the_frontier() {
    for scope in [
        Scope { depth: 1, work: 4096, prefix: 128 },
        Scope { depth: 4, work: 1, prefix: 128 },
        Scope { depth: 4, work: 4096, prefix: 1 },
    ] {
        let mut runtime = Runtime::new(World, Binding { scope, ..Binding::default() });
        let request = protocol::JudgeRequest { observation: Some(any(RENEW)) }.encode_to_vec();
        let first = runtime.dispatch(protocol::Method::Judge as u32, &request);
        // A short witness may publish while compatible longer work remains retained.
        if scope.depth == 1 && matches!(first.outcome, Some(protocol::response::Outcome::Judge(_)))
        {
            let next = protocol::JudgeRequest { observation: Some(any(GLOBAL)) }.encode_to_vec();
            for _ in 0..2 {
                runtime
                    .dispatch(protocol::Method::Judge as u32, &next)
                    .refusal(protocol::RefusalCode::Budget);
            }
        } else {
            first.refusal(protocol::RefusalCode::Budget);
            runtime
                .dispatch(protocol::Method::Judge as u32, &request)
                .refusal(protocol::RefusalCode::Budget);
        }
        runtime.dispatch(protocol::Method::Reset as u32, &[]);
    }
}

#[test]
fn countdown_complete_exhaustion_freezes_only_after_publication_and_reset_recovers() {
    let mut guest = Guest::new(frontier::runtime());
    guest.judge(any(RENEW)).explanation();
    guest.judge(any(GLOBAL)).explanation();
    let request = protocol::JudgeRequest { observation: Some(any(LOCAL)) }.encode_to_vec();
    assert!(!guest.dispatch_to(protocol::Method::Judge as u32, &request, |bytes| {
        protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")).unexplained();
        false
    }));
    guest.judge(any(LOCAL)).unexplained();
    guest.judge(any(RENEW)).unexplained();
    umpire_runtime::testing::GuestExt::call(
        &mut guest,
        protocol::Method::Propose,
        &protocol::ProposeRequest { seed: 0 },
    )
    .refusal(protocol::RefusalCode::State);
    guest.reset(None);
    assert_eq!(guest.judge(any(RENEW)).explanation().steps, ["Renew"]);
    assert_eq!(guest.judge(any(LOCAL)).explanation().steps, ["Renew", "Tick", "Tick"]);
}

#[test]
fn countdown_later_report_revalidates_the_exact_fact_at_every_earlier_observation() {
    let mut runtime = frontier::runtime();
    let first = protocol::JudgeRequest { observation: Some(any(RENEW)) }.encode_to_vec();
    let earlier = runtime.dispatch(protocol::Method::Judge as u32, &first).explanation();
    let immutable = earlier.encode_to_vec();
    runtime.replace_realization(Binding { fault: Fault::RejectRenewFact, ..Binding::default() });
    let later = protocol::JudgeRequest { observation: Some(any(GLOBAL)) }.encode_to_vec();
    for _ in 0..2 {
        runtime
            .dispatch(protocol::Method::Judge as u32, &later)
            .refusal(protocol::RefusalCode::State);
    }
    runtime.replace_realization(Binding::default());
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &later).explanation().steps,
        ["Tick", "Renew", "Tick", "Tick"]
    );
    assert_eq!(earlier.encode_to_vec(), immutable);
}

#[test]
fn countdown_read_policy_cannot_choose_an_arbitrary_hidden_budget() {
    let mut runtime = frontier::runtime();
    let first = protocol::JudgeRequest { observation: Some(any(RENEW)) }.encode_to_vec();
    runtime.dispatch(protocol::Method::Judge as u32, &first).explanation();
    runtime.replace_realization(Binding { fault: Fault::DivergentRead, ..Binding::default() });
    for _ in 0..2 {
        let refused = runtime
            .dispatch(protocol::Method::Propose as u32, &[])
            .refusal(protocol::RefusalCode::State);
        assert!(refused.message.contains("disagree"));
    }
    runtime.replace_realization(Binding::default());
    let later = protocol::JudgeRequest { observation: Some(any(GLOBAL)) }.encode_to_vec();
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &later).explanation().steps,
        ["Tick", "Renew", "Tick", "Tick"]
    );
}

#[test]
fn countdown_finality_cannot_depend_on_an_arbitrary_hidden_budget() {
    let mut runtime =
        Runtime::new(World, Binding { fault: Fault::DivergentPoll, ..Binding::default() });
    let proposal = runtime.dispatch(protocol::Method::Propose as u32, &[]);
    assert!(matches!(proposal.outcome, Some(protocol::response::Outcome::Propose(_))));
    let first = protocol::JudgeRequest { observation: Some(any(RENEW)) }.encode_to_vec();
    for _ in 0..2 {
        let refused = runtime
            .dispatch(protocol::Method::Judge as u32, &first)
            .refusal(protocol::RefusalCode::State);
        assert!(refused.message.contains("disagree"));
    }
    runtime.replace_realization(Binding::default());
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &first).explanation().steps,
        ["Renew"]
    );
    let later = protocol::JudgeRequest { observation: Some(any(GLOBAL)) }.encode_to_vec();
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &later).explanation().steps,
        ["Tick", "Renew", "Tick", "Tick"]
    );
}

#[test]
fn countdown_callback_reply_cannot_choose_an_arbitrary_hidden_budget() {
    let mut runtime = frontier::runtime();
    let first = protocol::JudgeRequest { observation: Some(any(RENEW)) }.encode_to_vec();
    runtime.dispatch(protocol::Method::Judge as u32, &first).explanation();
    runtime.replace_realization(Binding { fault: Fault::DivergentAnswer, ..Binding::default() });
    let callback = protocol::AnswerRequest { callback: Some(any(RENEW)) }.encode_to_vec();
    for _ in 0..2 {
        let refused = runtime
            .dispatch(protocol::Method::Answer as u32, &callback)
            .refusal(protocol::RefusalCode::State);
        assert!(refused.message.contains("disagree"));
    }
    runtime.replace_realization(Binding::default());
    let later = protocol::JudgeRequest { observation: Some(any(GLOBAL)) }.encode_to_vec();
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &later).explanation().steps,
        ["Tick", "Renew", "Tick", "Tick"]
    );
}
