//! Programmed worker judgments use server result and failure evidence.
use prost::Message;
use umpire_runtime::{
    Dispatch,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::{
    temporal::api::{
        activity::v1::{ActivityExecutionOutcome, activity_execution_outcome::Value},
        workflowservice::v1::{
            DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
            RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        },
    },
    umpire::temporal::v1::{ActivityTaskKey, ObserveActivity, WorkerProgram},
};
mod common;
fn invoke(runtime: &mut impl Dispatch, method: u32, request: &impl Message) -> Outcome {
    let (response, transaction) = runtime.prepare(method, &request.encode_to_vec());
    if let Some(transaction) = transaction {
        runtime.commit(transaction);
    }
    response.outcome.unwrap_or_else(|| panic!("outcome"))
}
fn setup(seed: u64) -> (impl Dispatch, WorkerProgram) {
    let mut runtime = annotation_runtime();
    let Outcome::Reset(reset) = invoke(
        &mut runtime,
        3,
        &protocol::ResetRequest { configuration: common::start(3).request },
    ) else {
        panic!("reset")
    };
    let program = reset
        .program
        .unwrap_or_else(|| panic!("program"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let Outcome::Propose(bootstrap) = invoke(&mut runtime, 6, &protocol::ProposeRequest { seed })
    else {
        panic!("bootstrap")
    };
    let mut start = common::start(3);
    start.request = bootstrap.request;
    assert!(matches!(
        invoke(&mut runtime, 4, &protocol::JudgeRequest { observation: Some(Any::pack(&start)) }),
        Outcome::Judge(_)
    ));
    (runtime, program)
}
fn row(program: &WorkerProgram, scenario: &str, attempt: i32) -> Any {
    program
        .rules
        .iter()
        .find(|rule| {
            let key: ActivityTaskKey = rule
                .key
                .as_ref()
                .unwrap_or_else(|| panic!("key"))
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
            key.scenario == scenario && key.attempt == attempt
        })
        .unwrap_or_else(|| panic!("row"))
        .respond[0]
        .clone()
}
fn terminal(program: &WorkerProgram, scenario: &str, attempt: i32, wrong: bool) -> Observation {
    let mut observation = common::describe(attempt, if scenario == "Fail" { 3 } else { 2 }, 0);
    let mut response: DescribeActivityExecutionResponse = match observation.result.take() {
        Some(ResultValue::Response(value)) => value.unpack().unwrap_or_else(|e| panic!("{e:?}")),
        _ => panic!("response"),
    };
    let value = if scenario == "Fail" && !wrong {
        let failed: RespondActivityTaskFailedRequest =
            row(program, scenario, attempt).unpack().unwrap_or_else(|e| panic!("{e:?}"));
        Value::Failure(failed.failure.unwrap_or_else(|| panic!("failure")))
    } else {
        let completed: RespondActivityTaskCompletedRequest =
            row(program, "Complete", attempt).unpack().unwrap_or_else(|e| panic!("{e:?}"));
        let mut result = completed.result.unwrap_or_else(|| panic!("result"));
        if wrong && scenario != "Fail" {
            result.payloads[0].data = b"wrong".to_vec();
        }
        if scenario == "Fail" {
            response.info.as_mut().unwrap_or_else(|| panic!("info")).status = 2;
        }
        Value::Result(result)
    };
    response.outcome = Some(ActivityExecutionOutcome { value: Some(value), ..Default::default() });
    if attempt > 1 {
        let failed: RespondActivityTaskFailedRequest =
            row(program, "RetryThenComplete", attempt - 1)
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
        response.info.as_mut().unwrap_or_else(|| panic!("info")).last_failure = failed.failure;
    }
    observation.result = Some(ResultValue::Response(Any::pack(&response)));
    observation
}
#[test]
fn describe_requests_result_and_last_failure() {
    let (mut runtime, _) = setup(0);
    let Outcome::Propose(proposal) = invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 1 })
    else {
        panic!("read")
    };
    let instruction: ObserveActivity = proposal
        .request
        .unwrap_or_else(|| panic!("instruction"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let request: DescribeActivityExecutionRequest = instruction
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    assert!(request.include_outcome);
    assert!(request.include_last_failure);
}
#[test]
fn program_content_gates_fast_terminal_paths_and_both_wrong_completion_controls() {
    for (seed, scenario, attempt, wrong) in [
        (0, "Complete", 1, false),
        (1, "RetryThenComplete", 2, false),
        (2, "Fail", 1, false),
        (0, "Complete", 1, true),
        (2, "Fail", 1, true),
    ] {
        let (mut runtime, program) = setup(seed);
        let observation = terminal(&program, scenario, attempt, wrong);
        let Outcome::Judge(judgment) = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&observation)) },
        ) else {
            panic!("judge")
        };
        assert_eq!(
            matches!(judgment.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))),
            wrong,
            "{scenario} attempt {attempt}: {judgment:?}"
        );
        if !wrong {
            let Some(protocol::judge_response::Judgment::Explained(explanation)) =
                judgment.judgment
            else {
                panic!("explained")
            };
            assert_eq!(explanation.steps.len(), if attempt == 2 { 4 } else { 2 });
        }
    }
}
#[test]
fn retry_history_requires_the_programmed_failure_and_unprogrammed_is_refused() {
    use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
    for unprogrammed in [false, true] {
        let (mut runtime, program) = setup(1);
        let mut observation = terminal(&program, "RetryThenComplete", 2, false);
        let Some(ResultValue::Response(value)) = observation.result.take() else {
            panic!("response")
        };
        let mut response: DescribeActivityExecutionResponse =
            value.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        let failure = response
            .info
            .as_mut()
            .unwrap_or_else(|| panic!("info"))
            .last_failure
            .as_mut()
            .unwrap_or_else(|| panic!("failure"));
        if unprogrammed {
            let Some(FailureInfo::ApplicationFailureInfo(info)) = &mut failure.failure_info else {
                panic!("application failure")
            };
            info.r#type = "umpire.unprogrammed".into();
        } else {
            failure.message = "wrong retry".into();
        }
        observation.result = Some(ResultValue::Response(Any::pack(&response)));
        let outcome = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&observation)) },
        );
        if unprogrammed {
            let Outcome::Refusal(refusal) = outcome else { panic!("refusal") };
            assert!(refusal.message.contains("unprogrammed"));
        } else {
            let Outcome::Judge(judgment) = outcome else { panic!("judge") };
            assert!(matches!(
                judgment.judgment,
                Some(protocol::judge_response::Judgment::Unexplained(_))
            ));
        }
    }
}

#[test]
fn failure_contents_are_checked_while_sdk_diagnostics_are_not_program_fields() {
    for wrong in [false, true] {
        let (mut runtime, program) = setup(2);
        let mut observation = terminal(&program, "Fail", 1, false);
        let Some(ResultValue::Response(value)) = observation.result.take() else {
            panic!("response")
        };
        let mut response: DescribeActivityExecutionResponse =
            value.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        let Some(Value::Failure(failure)) =
            response.outcome.as_mut().and_then(|outcome| outcome.value.as_mut())
        else {
            panic!("failure")
        };
        failure.source = "GoSDK".into();
        failure.stack_trace = "handler.go:42".into();
        if wrong {
            failure.message = "wrong failure".into();
        }
        observation.result = Some(ResultValue::Response(Any::pack(&response)));
        let Outcome::Judge(judgment) = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&observation)) },
        ) else {
            panic!("judge")
        };
        assert_eq!(
            matches!(judgment.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))),
            wrong
        );
    }
}
