//! Exact Kettle observation and callback controls.
#![forbid(unsafe_code)]

use prost::Name;
use umpire_reference_kettle::{
    runtime as kettle_runtime,
    wire::umpire::reference::kettle::v1::{
        DescribeResponse, HeatRequest, HeatResponse, HeatingAction, PollHeatingResponse,
        RespondHeatingRequest, RespondHeatingResponse, State,
    },
};
use umpire_runtime::AnyMessage as _;
use umpire_runtime::protocol::Method;
use umpire_runtime::{
    Guest,
    testing::{GuestExt, ResponseExt},
};
use umpire_runtime::{
    prost_types::Any,
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
fn steps(response: protocol::Response, expected: &[&str]) {
    let value = response.explanation();
    assert_eq!(value.steps, expected);
}
fn heat(runtime: &mut Guest<umpire_reference_kettle::KettleRuntime>) -> protocol::Explanation {
    let explanation = runtime
        .judge(observe(
            "Heat",
            &HeatResponse { state: Some(state(20, true, false)) },
            Some(Any::pack(&HeatRequest { kettle_id: "k".into(), enabled: true })),
        ))
        .explanation();
    assert_eq!(explanation.steps, ["User::PressButton"]);
    explanation
}
#[test]
fn healthy_trace_answer_does_not_commit_and_receipt_commits_once() {
    let mut runtime = Guest::new(kettle_runtime());
    let first = heat(&mut runtime);
    let original = first.clone();
    let mut witness = vec!["User::PressButton"];
    for temperature in 20..=100 {
        let action = if temperature < 100 { HeatingAction::Heat } else { HeatingAction::Failover };
        let task = PollHeatingResponse {
            state: Some(state(temperature, true, false)),
            action: action as i32,
            kettle_id: "k".into(),
        };
        let response = GuestExt::call(
            &mut runtime,
            Method::Answer,
            &protocol::AnswerRequest {
                callback: Some(Any::pack(&Callback { task: Some(Any::pack(&task)) })),
            },
        );
        assert!(matches!(response.outcome, Some(Outcome::Answer(_))));
        steps(
            runtime.judge(observe("Describe", &DescribeResponse { state: task.state }, None)),
            &witness,
        );
        let next = if temperature < 100 {
            state(temperature + 1, true, false)
        } else {
            state(100, false, true)
        };
        witness.push(if temperature < 100 { "Heater::Heat" } else { "Heater::Failover" });
        steps(
            runtime.judge(observe(
                "RespondHeating",
                &RespondHeatingResponse { state: Some(next), action: action as i32 },
                Some(Any::pack(&RespondHeatingRequest {
                    kettle_id: "k".into(),
                    action: action as i32,
                })),
            )),
            &witness,
        );
    }
    assert_eq!(first, original);
    assert_eq!(first.steps, ["User::PressButton"]);
}
#[test]
fn describe_never_infers_a_transition() {
    let mut runtime = Guest::new(kettle_runtime());
    let response = runtime.judge(observe(
        "Describe",
        &DescribeResponse { state: Some(state(20, true, false)) },
        None,
    ));
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
        let mut runtime = Guest::new(kettle_runtime());
        heat(&mut runtime);
        let response = runtime.judge(observe(
            "RespondHeating",
            &RespondHeatingResponse { state: Some(next), action: action as i32 },
            None,
        ));
        assert!(!matches!(
            response.outcome,
            Some(Outcome::Judge(protocol::JudgeResponse {
                judgment: Some(protocol::judge_response::Judgment::Explained(_)),
                ..
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
    let response = GuestExt::call(
        &mut Guest::new(kettle_runtime()),
        Method::Explore,
        &protocol::ExploreRequest { budget: 10000 },
    );
    let Some(Outcome::Explore(value)) = response.outcome else { panic!("not Explore") };
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path("../expected");
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    settings.bind(|| insta::assert_snapshot!("report.txt", value.report));
}

#[test]
fn malformed_evidence_refuses_without_advancing() {
    let mut runtime = Guest::new(kettle_runtime());
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
        assert!(matches!(runtime.judge(value).outcome, Some(Outcome::Refusal(_))));
        steps(
            runtime.judge(observe(
                "Describe",
                &DescribeResponse { state: Some(state(20, true, false)) },
                None,
            )),
            &["User::PressButton"],
        );
    }
}

#[test]
fn callback_requires_exact_state_identity_and_enabled_action() {
    let mut runtime = Guest::new(kettle_runtime());
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
        let response = GuestExt::call(
            &mut runtime,
            Method::Answer,
            &protocol::AnswerRequest {
                callback: Some(Any::pack(&Callback { task: Some(Any::pack(&task)) })),
            },
        );
        assert!(matches!(response.outcome, Some(Outcome::Refusal(_))));
        steps(
            runtime.judge(observe(
                "Describe",
                &DescribeResponse { state: Some(state(20, true, false)) },
                None,
            )),
            &["User::PressButton"],
        );
    }
}

#[test]
fn heat_projects_reach_with_an_exact_target_and_method_owned_evidence() {
    use umpire_dsl::Machine as _;
    use umpire_reference_kettle::{Realization, kettle};
    use umpire_runtime::{Evidence, EvidenceMatch, Projection, ReachTarget as _, Realize as _};

    let world = kettle::world();
    let initial = kettle::Model { temperature: 20, heating_on: false, beeping: false };
    let observation = observe(
        "Heat",
        &HeatResponse { state: Some(state(20, true, false)) },
        Some(Any::pack(&HeatRequest { kettle_id: "k".into(), enabled: true })),
    );
    let Projection::Reach(target) = Realization
        .observe(&world, &observation, &initial)
        .unwrap_or_else(|error| panic!("{error:?}"))
    else {
        panic!("Heat must use Reach");
    };
    let action = kettle::User::PressButton.into();
    let (outcome, facts, next) =
        world.checked_step(&initial, action).unwrap_or_else(|error| panic!("{error:?}")).remove(0);
    assert!(target.matches(&next));
    assert!(!target.matches(&initial));
    assert!(!target.matches(&kettle::Model { beeping: true, ..next.clone() }));
    let edge = Evidence {
        source: initial,
        action: kettle::User::PressButton.into(),
        choice: 0,
        outcome,
        facts,
        target: next,
    };
    assert!(matches!(
        Realization.reach_evidence(&observation, &[&edge]),
        EvidenceMatch::Match { cursor: 1 },
    ));
    let describe =
        observe("Describe", &DescribeResponse { state: Some(state(20, true, false)) }, None);
    assert!(matches!(Realization.reach_evidence(&describe, &[&edge]), EvidenceMatch::Reject));
}
