//! Exact Kettle observation and callback controls.
use prost::{Message, Name};
use umpire_reference_kettle::{
    runtime,
    wire::umpire::reference::kettle::v1::{
        DescribeResponse, HeatRequest, HeatResponse, HeatingAction, PollHeatingResponse,
        RespondHeatingRequest, RespondHeatingResponse, State,
    },
};
use umpire_runtime::{
    google::protobuf::Any,
    protocol::{self, response::Outcome},
    realization::{Callback, Observation, observation::Result as ObservationResult},
};

const SERVICE: &str = "/umpire.reference.kettle.v1.KettleService/";
fn state(temperature: i32, heating_on: bool, beeping: bool) -> State {
    State { temperature, heating_on, beeping }
}
fn observe(method: &str, response: &impl Name, request: Option<Any>) -> Any {
    Any::pack(&Observation {
        method: format!("{SERVICE}{method}"),
        result: Some(ObservationResult::Response(Any::pack(response))),
        request,
    })
}
fn judge(runtime: &mut umpire_reference_kettle::KettleRuntime, value: Any) -> protocol::Response {
    runtime.dispatch(4, &protocol::JudgeRequest { observation: Some(value) }.encode_to_vec())
}
fn steps(response: protocol::Response, expected: &[&str]) {
    let Some(Outcome::Judge(value)) = response.outcome else { panic!("not Judge") };
    let Some(protocol::judge_response::Judgment::Explained(value)) = value.judgment else {
        panic!("not explained")
    };
    assert_eq!(value.steps, expected);
}
fn heat(runtime: &mut umpire_reference_kettle::KettleRuntime) {
    steps(
        judge(
            runtime,
            observe(
                "Heat",
                &HeatResponse { state: Some(state(20, true, false)) },
                Some(Any::pack(&HeatRequest { kettle_id: "k".into(), enabled: true })),
            ),
        ),
        &["User::PressButton"],
    );
}
#[test]
fn healthy_trace_answer_does_not_commit_and_receipt_commits_once() {
    let mut runtime = runtime();
    heat(&mut runtime);
    for temperature in 20..=100 {
        let action = if temperature < 100 { HeatingAction::Heat } else { HeatingAction::Failover };
        let task = PollHeatingResponse {
            state: Some(state(temperature, true, false)),
            action: action as i32,
            kettle_id: "k".into(),
        };
        let response = runtime.dispatch(
            5,
            &protocol::AnswerRequest {
                callback: Some(Any::pack(&Callback { task: Some(Any::pack(&task)) })),
            }
            .encode_to_vec(),
        );
        assert!(matches!(response.outcome, Some(Outcome::Answer(_))));
        steps(
            judge(&mut runtime, observe("Describe", &DescribeResponse { state: task.state }, None)),
            &[],
        );
        let next = if temperature < 100 {
            state(temperature + 1, true, false)
        } else {
            state(100, false, true)
        };
        steps(
            judge(
                &mut runtime,
                observe(
                    "RespondHeating",
                    &RespondHeatingResponse { state: Some(next), action: action as i32 },
                    Some(Any::pack(&RespondHeatingRequest {
                        kettle_id: "k".into(),
                        action: action as i32,
                    })),
                ),
            ),
            &[if temperature < 100 { "Heater::Heat" } else { "Heater::Failover" }],
        );
    }
}
#[test]
fn describe_never_infers_a_transition() {
    let mut runtime = runtime();
    let response = judge(
        &mut runtime,
        observe("Describe", &DescribeResponse { state: Some(state(20, true, false)) }, None),
    );
    let Some(Outcome::Judge(value)) = response.outcome else { panic!("not Judge") };
    assert!(matches!(value.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))));
}
#[test]
fn exact_receipts_reject_wrong_state_or_action() {
    for (action, next) in [
        (HeatingAction::Heat, state(22, true, false)),
        (HeatingAction::Failover, state(21, true, false)),
        (HeatingAction::Unspecified, state(21, true, false)),
    ] {
        let mut runtime = runtime();
        heat(&mut runtime);
        let response = judge(
            &mut runtime,
            observe(
                "RespondHeating",
                &RespondHeatingResponse { state: Some(next), action: action as i32 },
                None,
            ),
        );
        assert!(!matches!(
            response.outcome,
            Some(Outcome::Judge(protocol::JudgeResponse {
                judgment: Some(protocol::judge_response::Judgment::Explained(_))
            }))
        ));
    }
}
#[test]
fn explore_retains_canonical_report() {
    assert_eq!(
        DescribeResponse::type_url(),
        "type.googleapis.com/umpire.reference.kettle.v1.DescribeResponse"
    );
    let response =
        runtime().dispatch(1, &protocol::ExploreRequest { budget: 10000 }.encode_to_vec());
    let Some(Outcome::Explore(value)) = response.outcome else { panic!("not Explore") };
    assert_eq!(value.report, include_str!("../expected/report.snap.txt"));
}

#[test]
fn malformed_evidence_refuses_without_advancing() {
    let mut runtime = runtime();
    heat(&mut runtime);
    for request in [
        None,
        Some(Any::pack(&RespondHeatingRequest { kettle_id: String::new(), action: 1 })),
        Some(Any::pack(&RespondHeatingRequest { kettle_id: "k".into(), action: 2 })),
    ] {
        let value = if let Some(request) = request {
            observe(
                "RespondHeating",
                &RespondHeatingResponse { state: Some(state(21, true, false)), action: 1 },
                Some(request),
            )
        } else {
            observe("Heat", &HeatResponse { state: Some(state(20, false, false)) }, None)
        };
        assert!(matches!(judge(&mut runtime, value).outcome, Some(Outcome::Refusal(_))));
        steps(
            judge(
                &mut runtime,
                observe(
                    "Describe",
                    &DescribeResponse { state: Some(state(20, true, false)) },
                    None,
                ),
            ),
            &[],
        );
    }
}

#[test]
fn callback_requires_exact_state_identity_and_enabled_action() {
    let mut runtime = runtime();
    heat(&mut runtime);
    for task in [
        PollHeatingResponse {
            state: Some(state(21, true, false)),
            action: 1,
            kettle_id: "k".into(),
        },
        PollHeatingResponse {
            state: Some(state(20, true, false)),
            action: 1,
            kettle_id: String::new(),
        },
        PollHeatingResponse {
            state: Some(state(20, true, false)),
            action: 2,
            kettle_id: "k".into(),
        },
        PollHeatingResponse { state: None, action: 1, kettle_id: "k".into() },
    ] {
        let response = runtime.dispatch(
            5,
            &protocol::AnswerRequest {
                callback: Some(Any::pack(&Callback { task: Some(Any::pack(&task)) })),
            }
            .encode_to_vec(),
        );
        assert!(matches!(response.outcome, Some(Outcome::Refusal(_))));
        steps(
            judge(
                &mut runtime,
                observe(
                    "Describe",
                    &DescribeResponse { state: Some(state(20, true, false)) },
                    None,
                ),
            ),
            &[],
        );
    }
}
