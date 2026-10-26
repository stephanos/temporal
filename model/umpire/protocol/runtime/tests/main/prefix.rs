//! Singleton whole-prefix retention through the publication transaction.
#![forbid(unsafe_code)]

use crate::support;

use prost::Message as _;
use support::{Action, Outcome, State, World, any};
use umpire_runtime::{
    AnswerPlan, Dispatch as _, Guest, NoTarget, Projection, Proposal, Realize, Runtime, Scope,
    prost_types::Any,
    protocol,
    testing::{GuestExt as _, ResponseExt as _},
};

struct Bounded {
    prefix: usize,
    reject_old: bool,
}

impl umpire_runtime::capability::Identity for Bounded {
    type Id = ();

    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<World as umpire_dsl::Machine>::Action> for Bounded {
    type World = World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<World> for Bounded {
    type Target = NoTarget;

    fn project(
        &self,
        world: &World,
        actual: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome, NoTarget>, protocol::Refusal> {
        if self.reject_old && support::payload(actual)? == 1 && state.0 == 0 {
            return Ok(Projection::Candidates(Vec::new()));
        }
        Realize::observe(&support::Realization, world, actual, state)
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Bounded {
    type World = World;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<World> for Bounded {
    fn scope(&self) -> Scope {
        Scope { prefix: self.prefix, ..Scope::default() }
    }

    fn propose(
        &self,
        world: &World,
        state: &State,
        seed: u64,
    ) -> Result<Proposal<Action>, protocol::Refusal> {
        Realize::propose(&support::Realization, world, state, seed)
    }

    fn matches(&self, request: &Any, actual: &Any) -> Result<(), protocol::Refusal> {
        Realize::matches(&support::Realization, request, actual)
    }

    fn answer(
        &self,
        world: &World,
        actual: &Any,
        state: &State,
    ) -> Result<AnswerPlan<Action>, protocol::Refusal> {
        Realize::answer(&support::Realization, world, actual, state)
    }
}

impl Realize<World> for Bounded {}

fn run(prefix: usize) -> Runtime<World, Bounded> {
    Runtime::new(World { starts: 1 }, Bounded { prefix, reject_old: false })
}

#[test]
fn publication_and_prefix_bound_keep_witness_pending_and_prior_reports_for_retry() {
    let mut guest = Guest::new(run(5));
    let earlier = guest.judge(any(1)).explanation();
    let earlier_bytes = earlier.encode_to_vec();
    guest.propose(0);
    guest
        .judge(Any { type_url: support::TYPE.into(), value: vec![1; 1 << 20] })
        .refusal(protocol::RefusalCode::Allocation);
    let request = protocol::JudgeRequest { observation: Some(any(2)) }.encode_to_vec();
    assert!(!guest.dispatch_to(protocol::Method::Judge as u32, &request, |bytes| {
        let response = protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(response.explanation().steps, ["Deliver", "Respond"]);
        false
    }));
    guest.judge(any(3)).refusal(protocol::RefusalCode::State);
    assert_eq!(guest.judge(any(2)).explanation().steps, ["Deliver", "Respond"]);
    for _ in 0..2 {
        let refused = guest.judge(any(2));
        assert!(refused.refusal(protocol::RefusalCode::Budget).message.contains("prefix bound 5"));
    }
    assert_eq!(earlier.encode_to_vec(), earlier_bytes);
    assert!(!guest.dispatch_to(protocol::Method::Reset as u32, &[], |_| false));
    guest.judge(any(2)).refusal(protocol::RefusalCode::Budget);
    guest.reset(None);
    assert_eq!(guest.judge(any(0)).explanation().steps, [] as [String; 0]);
    assert_eq!(guest.judge(any(1)).explanation().steps, ["Deliver"]);
}

#[test]
fn positive_current_revalidates_every_earlier_actual_observation() {
    let mut runtime = run(64);
    let request = protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec();
    let (response, transaction) = runtime.prepare(protocol::Method::Judge as u32, &request);
    assert_eq!(response.explanation().steps, ["Deliver"]);
    runtime.commit(transaction.unwrap_or_else(|| panic!("missing transaction")));
    runtime.replace_realization(Bounded { prefix: 64, reject_old: true });
    let response = runtime.dispatch(protocol::Method::Judge as u32, &request);
    assert_eq!(
        response.refusal(protocol::RefusalCode::State).message,
        "retained prefix no longer validates"
    );
    runtime.replace_realization(Bounded { prefix: 64, reject_old: false });
    assert_eq!(
        runtime.dispatch(protocol::Method::Judge as u32, &request).explanation().steps,
        ["Deliver"]
    );
}

#[test]
fn latest_witness_retains_callback_and_proposal_order_without_changing_prior_reports() {
    let mut guest = Guest::new(run(64));
    guest.reset(None);
    let initial = guest.judge(any(0)).explanation();
    assert_eq!(initial.steps, [] as [String; 0]);
    assert_eq!(initial.outcomes, [] as [String; 0]);
    let mut reports = vec![(initial.clone(), initial.encode_to_vec())];

    let answer = umpire_runtime::testing::GuestExt::call(
        &mut guest,
        protocol::Method::Answer,
        &protocol::AnswerRequest { callback: Some(any(4)) },
    );
    let Some(protocol::response::Outcome::Answer(answer)) = answer.outcome else {
        panic!("expected callback answer")
    };
    assert_eq!(answer.reply, Some(any(42)));
    let delivered = guest.judge(any(1)).explanation();
    assert_eq!(delivered.steps, ["Deliver"]);
    assert_eq!(delivered.outcomes, ["Outcome(1)"]);
    reports.push((delivered.clone(), delivered.encode_to_vec()));

    for (observation, steps, outcomes) in [
        (2, vec!["Deliver", "Respond"], vec!["Outcome(1)", "Outcome(2)"]),
        (2, vec!["Deliver", "Respond"], vec!["Outcome(1)", "Outcome(2)"]),
        (3, vec!["Deliver", "Respond", "Finish"], vec!["Outcome(1)", "Outcome(2)", "Outcome(3)"]),
    ] {
        if observation == 3 {
            let proposed = guest.propose(0);
            assert_eq!(proposed.action, "Finish");
            assert_eq!(proposed.request, Some(any(3)));
        }
        let latest = guest.judge(any(observation)).explanation();
        assert_eq!(latest.steps, steps);
        assert_eq!(latest.outcomes, outcomes);
        for (report, bytes) in &reports {
            assert_eq!(report.encode_to_vec(), *bytes);
        }
        reports.push((latest.clone(), latest.encode_to_vec()));
    }
}
