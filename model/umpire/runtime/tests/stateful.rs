//! Stateful realization and transactional publication controls.

mod support;

use prost::Message;
use support::{any, runtime};
use umpire_runtime::protocol::{self, response::Outcome};

fn refusal(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(result)) = response.outcome else { panic!("expected refusal") };
    assert_eq!(result.code, code as i32);
}

fn unexplained(response: protocol::Response) -> Vec<String> {
    let Some(Outcome::Judge(result)) = response.outcome else { panic!("expected Judge response") };
    let Some(protocol::judge_response::Judgment::Unexplained(result)) = result.judgment else {
        panic!("expected unexplained")
    };
    result.expected_actions
}

fn answer(
    runtime: &mut umpire_runtime::Runtime<support::World, support::Realization>,
    value: u8,
) -> protocol::Response {
    runtime.dispatch(5, &protocol::AnswerRequest { callback: Some(any(value)) }.encode_to_vec())
}

fn judge(
    runtime: &mut umpire_runtime::Runtime<support::World, support::Realization>,
    value: u8,
) -> protocol::Response {
    runtime.dispatch(4, &protocol::JudgeRequest { observation: Some(any(value)) }.encode_to_vec())
}

fn explained(response: protocol::Response) -> Vec<String> {
    let Some(Outcome::Judge(result)) = response.outcome else { panic!("expected Judge response") };
    let Some(protocol::judge_response::Judgment::Explained(result)) = result.judgment else {
        panic!("expected explanation")
    };
    result.steps
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
        fn observe(
            &self,
            observation: &umpire_runtime::google::protobuf::Any,
            state: &support::State,
        ) -> Result<Projection<support::Action, ()>, protocol::Refusal> {
            match <Self as Realize<support::World>>::observe(self, observation, state)? {
                Projection::Current => Ok(Projection::Current),
                Projection::Candidates(candidates) => {
                    Ok(Projection::steps(candidates.into_iter().map(|candidate| candidate.path)))
                }
            }
        }
        fn answer(
            &self,
            callback: &umpire_runtime::google::protobuf::Any,
            state: &support::State,
        ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
            <Self as Realize<support::World>>::answer(self, callback, state)
        }
    }
    let mut runtime = Runtime::new(UnitWorld, support::Realization);
    let Some(Outcome::Judge(result)) = runtime
        .dispatch(4, &protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec())
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
    assert_eq!(explained(judge(&mut runtime, 0)), [] as [String; 0]);
    assert_eq!(explained(judge(&mut runtime, 0)), [] as [String; 0]);
    assert_eq!(explained(judge(&mut runtime, 1)), ["Deliver"]);
    assert_eq!(explained(judge(&mut runtime, 1)), [] as [String; 0]);
    let encoded = judge(&mut runtime, 1).encode_to_vec();
    assert_eq!(encoded, vec![0x2a, 0x02, 0x0a, 0x00]);
}

#[test]
fn malformed_and_unknown_requests_preserve_the_current_run() {
    let mut runtime = runtime();
    for method in [3, 4, 5] {
        refusal(runtime.dispatch(method, &[0xff]), protocol::RefusalCode::MalformedRequest);
    }
    for method in [4, 5] {
        refusal(runtime.dispatch(method, &[]), protocol::RefusalCode::MalformedRequest);
        for (value, code) in [
            (
                umpire_runtime::google::protobuf::Any {
                    type_url: "unknown".into(),
                    value: vec![1],
                },
                protocol::RefusalCode::Unsupported,
            ),
            (
                umpire_runtime::google::protobuf::Any {
                    type_url: support::TYPE.into(),
                    value: Vec::new(),
                },
                protocol::RefusalCode::MalformedRequest,
            ),
        ] {
            let request = if method == 4 {
                protocol::JudgeRequest { observation: Some(value) }.encode_to_vec()
            } else {
                protocol::AnswerRequest { callback: Some(value) }.encode_to_vec()
            };
            refusal(runtime.dispatch(method, &request), code);
            assert_eq!(explained(judge(&mut runtime, 0)), [] as [String; 0]);
        }
    }
    refusal(runtime.dispatch(999, &[]), protocol::RefusalCode::UnknownMethod);
    for observation in [14, 15] {
        assert!(
            std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| judge(
                &mut runtime,
                observation
            )))
            .is_err()
        );
        assert_eq!(explained(judge(&mut runtime, 0)), [] as [String; 0]);
    }
    assert_eq!(explained(judge(&mut runtime, 1)), ["Deliver"]);
}

#[test]
fn invalid_candidates_freeze_the_first_enabled_actions_until_reset() {
    for observation in [2, 7, 8, 9, 11, 12, 13] {
        let mut runtime = runtime();
        assert_eq!(unexplained(judge(&mut runtime, observation)), ["Deliver"]);
        refusal(runtime.dispatch(4, &[0xff]), protocol::RefusalCode::MalformedRequest);
        refusal(runtime.dispatch(999, &[]), protocol::RefusalCode::UnknownMethod);
        let unknown = protocol::JudgeRequest {
            observation: Some(umpire_runtime::google::protobuf::Any {
                type_url: "unknown".into(),
                value: vec![1],
            }),
        }
        .encode_to_vec();
        refusal(runtime.dispatch(4, &unknown), protocol::RefusalCode::Unsupported);
        assert_eq!(unexplained(judge(&mut runtime, 1)), ["Deliver"]);
        refusal(answer(&mut runtime, 4), protocol::RefusalCode::State);
        assert!(matches!(runtime.dispatch(3, &[]).outcome, Some(Outcome::Reset(_))));
        assert_eq!(explained(judge(&mut runtime, 1)), ["Deliver"]);
    }
}

#[test]
fn explanations_replay_whole_candidate_sequences_and_isolate_instances() {
    let mut advanced = runtime();
    let mut untouched = runtime();
    assert_eq!(explained(judge(&mut advanced, 10)), ["Deliver", "Respond"]);
    assert_eq!(explained(judge(&mut advanced, 2)), [] as [String; 0]);
    assert_eq!(explained(judge(&mut untouched, 0)), [] as [String; 0]);
    assert_eq!(unexplained(judge(&mut advanced, 1)), ["Finish"]);
    assert_eq!(explained(judge(&mut untouched, 1)), ["Deliver"]);
    assert!(matches!(advanced.dispatch(3, &[]).outcome, Some(Outcome::Reset(_))));
    assert_eq!(explained(judge(&mut advanced, 0)), [] as [String; 0]);
    assert_eq!(explained(judge(&mut untouched, 1)), [] as [String; 0]);
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
        refusal(answer(&mut runtime, callback), code);
        assert_eq!(explained(judge(&mut runtime, 0)), [] as [String; 0]);
    }
    let Some(Outcome::Answer(response)) = answer(&mut runtime, 4).outcome else {
        panic!("expected answer")
    };
    assert_eq!(response.reply, Some(any(42)));
    assert_eq!(explained(judge(&mut runtime, 1)), [] as [String; 0]);
    refusal(answer(&mut runtime, 4), protocol::RefusalCode::State);
    assert_eq!(explained(judge(&mut runtime, 1)), [] as [String; 0]);
    assert_eq!(explained(judge(&mut runtime, 2)), ["Respond"]);
    assert_eq!(explained(judge(&mut runtime, 3)), ["Finish"]);
    assert_eq!(unexplained(judge(&mut runtime, 0)), [] as [String; 0]);
    assert_eq!(unexplained(judge(&mut runtime, 1)), [] as [String; 0]);
}

#[test]
fn stateful_operations_require_a_unique_initial_state() {
    for starts in [0, 2] {
        let mut runtime =
            umpire_runtime::Runtime::new(support::World { starts }, support::Realization);
        refusal(judge(&mut runtime, 0), protocol::RefusalCode::State);
        refusal(answer(&mut runtime, 4), protocol::RefusalCode::State);
        refusal(runtime.dispatch(3, &[]), protocol::RefusalCode::State);
    }
}

fn publish(
    guest: &mut umpire_runtime::Guest<
        umpire_runtime::Runtime<support::World, support::Realization>,
    >,
    method: u32,
    request: &[u8],
) -> protocol::Response {
    let mut result = None;
    assert!(guest.dispatch_to(method, request, |bytes| {
        result = Some(
            protocol::Response::decode(bytes)
                .unwrap_or_else(|error| panic!("invalid response: {error}")),
        );
        true
    }));
    result.unwrap_or_else(|| panic!("response not published"))
}

#[test]
fn failed_response_publication_preserves_judge_answer_reset_and_exploration() {
    for method in [1, 3, 4, 5] {
        for panics in [false, true] {
            let mut guest = umpire_runtime::Guest::new(runtime());
            if method == 3 {
                explained(publish(
                    &mut guest,
                    4,
                    &protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec(),
                ));
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
                refusal(publish(&mut guest, 2, &[]), protocol::RefusalCode::NotExplored);
            }
            let current = u8::from(method == 3);
            assert_eq!(
                explained(publish(
                    &mut guest,
                    4,
                    &protocol::JudgeRequest { observation: Some(any(current)) }.encode_to_vec()
                )),
                [] as [String; 0]
            );
        }
    }
}

#[test]
fn failed_unexplained_publication_does_not_freeze_the_run() {
    let mut guest = umpire_runtime::Guest::new(runtime());
    let impossible = protocol::JudgeRequest { observation: Some(any(2)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &impossible, |_| false));
    let delivery = protocol::JudgeRequest { observation: Some(any(1)) }.encode_to_vec();
    assert_eq!(explained(publish(&mut guest, 4, &delivery)), ["Deliver"]);
    assert!(matches!(publish(&mut guest, 3, &[]).outcome, Some(Outcome::Reset(_))));
    assert_eq!(unexplained(publish(&mut guest, 4, &impossible)), ["Deliver"]);
    assert!(!guest.dispatch_to(3, &[], |_| false));
    assert_eq!(unexplained(publish(&mut guest, 4, &delivery)), ["Deliver"]);
}
