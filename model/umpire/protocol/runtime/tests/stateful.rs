//! Stateful realization and transactional publication controls.
use umpire_runtime::{
    protocol::Method,
    testing::{GuestExt, ResponseExt},
};

mod support;

use prost::Message;
use support::{any, runtime};
use umpire_runtime::protocol::{self, response::Outcome};

fn answer(
    runtime: &mut umpire_runtime::Runtime<support::World, support::Realization>,
    value: u8,
) -> protocol::Response {
    runtime.dispatch(
        Method::Answer as u32,
        &protocol::AnswerRequest { callback: Some(any(value)) }.encode_to_vec(),
    )
}

fn judge(
    runtime: &mut umpire_runtime::Runtime<support::World, support::Realization>,
    value: u8,
) -> protocol::Response {
    runtime.dispatch(
        Method::Judge as u32,
        &protocol::JudgeRequest { observation: Some(any(value)) }.encode_to_vec(),
    )
}

#[test]
fn explanations_carry_all_replayed_non_clone_outcomes() {
    let mut runtime = runtime();
    let Some(Outcome::Judge(result)) = judge(&mut runtime, 10).outcome else {
        panic!("expected Judge response")
    };
    let Some(protocol::judge_response::Judgment::Explained(result)) = result.judgment else {
        panic!("expected explanation")
    };
    assert_eq!(result.steps, ["Deliver", "Respond"]);
    assert_eq!(result.outcomes, ["Outcome(1)", "Outcome(2)"]);
}

#[test]
fn runtime_formats_unit_outcomes_with_debug() {
    use umpire_dsl::{Machine, Steps};
    use umpire_runtime::{AnswerPlan, Projection, Realize, Runtime};

    struct UnitWorld;
    impl Machine for UnitWorld {
        type State = support::State;
        type Action = support::Action;
        type Outcome = ();
        type Fact = support::Fact;

        fn starts(&self) -> Vec<Self::State> {
            vec![support::State(0)]
        }
        fn actions(&self) -> Vec<Self::Action> {
            vec![support::Action::Deliver]
        }
        fn internal(&self, _: &Self::Action) -> bool {
            false
        }
        fn step(
            &self,
            state: &Self::State,
            action: Self::Action,
        ) -> Steps<Self::State, (), Self::Fact> {
            support::World { starts: 1 }
                .step(state, action)
                .into_iter()
                .map(|(_, facts, state)| ((), facts, state))
                .collect()
        }
    }
    impl Realize<UnitWorld> for support::Realization {
        type Target = umpire_runtime::NoTarget;

        fn observe(
            &self,
            observation: &prost_types::Any,
            state: &support::State,
        ) -> Result<Projection<support::Action, (), umpire_runtime::NoTarget>, protocol::Refusal>
        {
            match <Self as Realize<support::World>>::observe(self, observation, state)? {
                Projection::Reach(target) | Projection::Stutter(target) => match target {},
                Projection::Current => Ok(Projection::Current),
                Projection::Candidates(candidates) => {
                    Ok(Projection::steps(candidates.into_iter().map(|candidate| candidate.path)))
                }
            }
        }
        fn answer(
            &self,
            callback: &prost_types::Any,
            state: &support::State,
        ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
            <Self as Realize<support::World>>::answer(self, callback, state)
        }
    }
    let mut runtime = Runtime::new(UnitWorld, support::Realization);
    let Some(Outcome::Judge(result)) = runtime
        .dispatch(
            Method::Judge as u32,
            &protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec(),
        )
        .outcome
    else {
        panic!("expected Judge")
    };
    let Some(protocol::judge_response::Judgment::Explained(result)) = result.judgment else {
        panic!("expected explanation")
    };
    assert_eq!(result.steps, ["Deliver"]);
    assert_eq!(result.outcomes, ["()"]);
}

#[test]
fn judge_replays_model_steps_and_repeated_current_observations_are_positive() {
    let mut runtime = runtime();
    assert_eq!(judge(&mut runtime, 0).explanation().steps, [] as [String; 0]);
    assert_eq!(judge(&mut runtime, 0).explanation().steps, [] as [String; 0]);
    assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
    assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
    let encoded = judge(&mut runtime, 1).encode_to_vec();
    assert_eq!(encoded, b"\x2a\x17\x0a\x15\x0a\x07Deliver\x12\x0aOutcome(1)");
}

#[test]
fn malformed_and_unknown_requests_preserve_the_current_run() {
    let mut runtime = runtime();
    for method in [3, 4, 5] {
        runtime.dispatch(method, &[0xff]).refusal(protocol::RefusalCode::MalformedRequest);
    }
    for method in [4, 5] {
        runtime.dispatch(method, &[]).refusal(protocol::RefusalCode::MalformedRequest);
        for (value, code) in [
            (
                prost_types::Any { type_url: "unknown".into(), value: vec![1] },
                protocol::RefusalCode::Unsupported,
            ),
            (
                prost_types::Any { type_url: support::TYPE.into(), value: Vec::new() },
                protocol::RefusalCode::MalformedRequest,
            ),
        ] {
            let request = if method == 4 {
                protocol::JudgeRequest { observation: Some(value) }.encode_to_vec()
            } else {
                protocol::AnswerRequest { callback: Some(value) }.encode_to_vec()
            };
            runtime.dispatch(method, &request).refusal(code);
            assert_eq!(judge(&mut runtime, 0).explanation().steps, [] as [String; 0]);
        }
    }
    runtime.dispatch(999, &[]).refusal(protocol::RefusalCode::UnknownMethod);
    for observation in [14, 15] {
        assert!(
            std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| judge(
                &mut runtime,
                observation
            )))
            .is_err()
        );
        assert_eq!(judge(&mut runtime, 0).explanation().steps, [] as [String; 0]);
    }
    assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
}

#[test]
fn invalid_candidates_freeze_the_first_enabled_actions_until_reset() {
    for observation in [2, 7, 8, 9, 11, 12, 13] {
        let mut runtime = runtime();
        assert_eq!(judge(&mut runtime, observation).unexplained(), ["Deliver"]);
        runtime
            .dispatch(Method::Judge as u32, &[0xff])
            .refusal(protocol::RefusalCode::MalformedRequest);
        runtime.dispatch(999, &[]).refusal(protocol::RefusalCode::UnknownMethod);
        let unknown = protocol::JudgeRequest {
            observation: Some(prost_types::Any { type_url: "unknown".into(), value: vec![1] }),
        }
        .encode_to_vec();
        runtime
            .dispatch(Method::Judge as u32, &unknown)
            .refusal(protocol::RefusalCode::Unsupported);
        assert_eq!(judge(&mut runtime, 1).unexplained(), ["Deliver"]);
        answer(&mut runtime, 4).refusal(protocol::RefusalCode::State);
        assert!(matches!(
            runtime.dispatch(Method::Reset as u32, &[]).outcome,
            Some(Outcome::Reset(_))
        ));
        assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
    }
}

#[test]
fn explanations_replay_whole_candidate_sequences_and_isolate_instances() {
    let mut advanced = runtime();
    let mut untouched = runtime();
    assert_eq!(judge(&mut advanced, 10).explanation().steps, ["Deliver", "Respond"]);
    assert_eq!(judge(&mut advanced, 2).explanation().steps, ["Deliver", "Respond"]);
    assert_eq!(judge(&mut untouched, 0).explanation().steps, [] as [String; 0]);
    assert_eq!(judge(&mut advanced, 1).unexplained(), ["Finish"]);
    assert_eq!(judge(&mut untouched, 1).explanation().steps, ["Deliver"]);
    assert!(matches!(
        advanced.dispatch(Method::Reset as u32, &[]).outcome,
        Some(Outcome::Reset(_))
    ));
    assert_eq!(judge(&mut advanced, 0).explanation().steps, [] as [String; 0]);
    assert_eq!(judge(&mut untouched, 1).explanation().steps, ["Deliver"]);
}

#[test]
fn answer_validates_the_response_before_committing_delivery_and_awaits_response_evidence() {
    let mut runtime = runtime();
    for (callback, code) in [
        (5, protocol::RefusalCode::State),
        (6, protocol::RefusalCode::Allocation),
        (7, protocol::RefusalCode::MalformedRequest),
        (8, protocol::RefusalCode::State),
    ] {
        answer(&mut runtime, callback).refusal(code);
        assert_eq!(judge(&mut runtime, 0).explanation().steps, [] as [String; 0]);
    }
    let Some(Outcome::Answer(response)) = answer(&mut runtime, 4).outcome else {
        panic!("expected answer")
    };
    assert_eq!(response.reply, Some(any(42)));
    assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
    answer(&mut runtime, 4).refusal(protocol::RefusalCode::State);
    assert_eq!(judge(&mut runtime, 1).explanation().steps, ["Deliver"]);
    assert_eq!(judge(&mut runtime, 2).explanation().steps, ["Deliver", "Respond"]);
    assert_eq!(judge(&mut runtime, 3).explanation().steps, ["Deliver", "Respond", "Finish"]);
    assert_eq!(judge(&mut runtime, 0).unexplained(), [] as [String; 0]);
    assert_eq!(judge(&mut runtime, 1).unexplained(), [] as [String; 0]);
}

#[test]
fn stateful_operations_require_a_unique_initial_state() {
    for starts in [0, 2] {
        let mut runtime =
            umpire_runtime::Runtime::new(support::World { starts }, support::Realization);
        judge(&mut runtime, 0).refusal(protocol::RefusalCode::State);
        answer(&mut runtime, 4).refusal(protocol::RefusalCode::State);
        runtime.dispatch(Method::Reset as u32, &[]).refusal(protocol::RefusalCode::State);
    }
}

#[test]
fn failed_response_publication_preserves_judge_answer_reset_and_exploration() {
    for method in [1, 3, 4, 5] {
        for panics in [false, true] {
            let mut guest = umpire_runtime::Guest::new(runtime());
            if method == 3 {
                guest.judge(any(1)).explanation();
            }
            let request = match method {
                1 => protocol::ExploreRequest { budget: 16 }.encode_to_vec(),
                3 => Vec::new(),
                4 => protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec(),
                5 => protocol::AnswerRequest { callback: Some(any(4)) }.encode_to_vec(),
                _ => unreachable!(),
            };
            let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                guest.dispatch_to(method, &request, |bytes| {
                    assert!(!panics, "response sink failed");
                    let response = protocol::Response::decode(bytes)
                        .unwrap_or_else(|error| panic!("invalid response: {error}"));
                    let mut destination: &mut [u8] = &mut [][..];
                    assert!(response.encode(&mut destination).is_err());
                    false
                })
            }));
            if panics {
                assert!(result.is_err());
            } else {
                assert!(!result.unwrap_or(true));
            }
            if method == 1 {
                GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest::default())
                    .refusal(protocol::RefusalCode::NotExplored);
            }
            let current = u8::from(method == 3);
            let expected = if method == 3 { vec!["Deliver"] } else { vec![] };
            assert_eq!(guest.judge(any(current)).explanation().steps, expected);
        }
    }
}

#[test]
fn failed_unexplained_publication_does_not_freeze_the_run() {
    let mut guest = umpire_runtime::Guest::new(runtime());
    let impossible = protocol::JudgeRequest { observation: Some(any(2)) }.encode_to_vec();
    assert!(!guest.dispatch_to(Method::Judge as u32, &impossible, |_| false));
    assert_eq!(guest.judge(any(1)).explanation().steps, ["Deliver"]);
    assert!(matches!(guest.reset(None).outcome, Some(Outcome::Reset(_))));
    assert_eq!(guest.judge(any(2)).unexplained(), ["Deliver"]);
    assert!(!guest.dispatch_to(Method::Reset as u32, &[], |_| false));
    assert_eq!(guest.judge(any(1)).unexplained(), ["Deliver"]);
}
