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
    if let Some(Value::Failure(failure)) =
        response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
    {
        response.info.as_mut().unwrap_or_else(|| panic!("info")).last_failure =
            Some(failure.clone());
    } else if attempt > 1 {
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

fn cancel_ack(request: Option<Any>) -> Observation {
    Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request,
        result: Some(ResultValue::Error(umpire_runtime::realization::Error {
            code: 5,
            message: "activity execution closed".into(),
        })),
    }
}
fn reconciled(acknowledged: &Observation, current: &Observation) -> Any {
    Any::pack(&umpire_temporal_api::umpire::temporal::v1::ReconciledObservation {
        acknowledged: Some(Any::pack(acknowledged)),
        current: Some(Any::pack(current)),
    })
}
fn cancellation(
    runtime: &mut impl Dispatch,
) -> umpire_temporal_api::umpire::temporal::v1::TemporalInstruction {
    let Outcome::Propose(proposal) = invoke(runtime, 6, &protocol::ProposeRequest { seed: 2 })
    else {
        panic!("cancel")
    };
    proposal
        .request
        .unwrap_or_else(|| panic!("instruction"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"))
}

fn read_terminal(runtime: &mut impl Dispatch, program: &WorkerProgram, scenario: &str) {
    let Outcome::Propose(read) = invoke(runtime, 6, &protocol::ProposeRequest { seed: 1 }) else {
        panic!("normal read")
    };
    assert_eq!(read.kind, protocol::ProposalKind::Read as i32);
    let Outcome::Judge(closed) = invoke(
        runtime,
        4,
        &protocol::JudgeRequest {
            observation: Some(Any::pack(&terminal(program, scenario, 1, false))),
        },
    ) else {
        panic!("terminal read")
    };
    assert!(!closed.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = closed.judgment else {
        panic!("terminal read must be explained")
    };
    assert_eq!(
        explanation.steps,
        [
            "Worker::Poll(1)",
            if scenario == "Fail" {
                "Worker::RespondFailed { retryable: false }"
            } else {
                "Worker::RespondCompleted"
            }
        ]
    );
}

#[test]
fn cancel_not_found_reconciles_only_programmed_terminal_server_evidence() {
    for (seed, scenario, wrong) in
        [(0, "Complete", false), (2, "Fail", false), (0, "Complete", true), (2, "Fail", true)]
    {
        let (mut runtime, program) = setup(seed);
        read_terminal(&mut runtime, &program, scenario);
        let Outcome::Propose(cancel) =
            invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 0 })
        else {
            panic!("terminal cancellation")
        };
        let instruction: umpire_temporal_api::umpire::temporal::v1::TemporalInstruction = cancel
            .request
            .unwrap_or_else(|| panic!("instruction"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        assert!(!instruction.hold_worker_polls);
        let mut observation = terminal(&program, scenario, 1, wrong && scenario != "Fail");
        if wrong && scenario == "Fail" {
            let Some(ResultValue::Response(value)) = observation.result.take() else {
                panic!("response")
            };
            let mut response: DescribeActivityExecutionResponse =
                value.unpack().unwrap_or_else(|error| panic!("{error:?}"));
            let Some(Value::Failure(failure)) =
                response.outcome.as_mut().and_then(|outcome| outcome.value.as_mut())
            else {
                panic!("failure")
            };
            failure.message = "wrong failure".into();
            observation.result = Some(ResultValue::Response(Any::pack(&response)));
        }
        let outcome = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest {
                observation: Some(reconciled(
                    &cancel_ack(instruction.request.clone()),
                    &observation,
                )),
            },
        );
        let Outcome::Judge(judgment) = outcome else {
            panic!("terminal must correlate with cancel: {outcome:?}")
        };
        assert_eq!(
            matches!(judgment.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))),
            wrong
        );
        if wrong {
            continue;
        }
        assert!(
            judgment.continue_polling,
            "the pending cancel still needs its actual acknowledgment"
        );
        assert!(matches!(
            invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 0 }),
            Outcome::Refusal(_)
        ));
        let Some(protocol::judge_response::Judgment::Explained(explanation)) = judgment.judgment
        else {
            panic!("fresh terminal evidence must be explained")
        };
        assert_eq!(explanation.steps, [] as [String; 0]);
        let ack = cancel_ack(instruction.request);
        let Outcome::Judge(judgment) =
            invoke(&mut runtime, 4, &protocol::JudgeRequest { observation: Some(Any::pack(&ack)) })
        else {
            panic!("cancel acknowledgment")
        };
        let Some(protocol::judge_response::Judgment::Explained(explanation)) = judgment.judgment
        else {
            panic!("explained")
        };
        assert_eq!(explanation.steps, ["Client::Cancel"]);
        assert_eq!(explanation.outcomes, ["Rejected(NotFound)"]);
        assert!(!judgment.continue_polling);
        assert!(matches!(
            invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 0 }),
            Outcome::Propose(_)
        ));
    }
}

#[test]
fn cancel_reconciliation_describe_must_match_the_exact_instruction_request() {
    for field in ["include_outcome", "include_last_failure", "namespace", "activity_id", "run_id"] {
        let (mut runtime, program) = setup(0);
        let instruction = cancellation(&mut runtime);
        let mut observation = terminal(&program, "Complete", 1, false);
        let mut request: DescribeActivityExecutionRequest = observation
            .request
            .as_ref()
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        match field {
            "include_outcome" => request.include_outcome = false,
            "include_last_failure" => request.include_last_failure = false,
            "namespace" => request.namespace = "other".into(),
            "activity_id" => request.activity_id = "other".into(),
            "run_id" => request.run_id = "other".into(),
            _ => unreachable!(),
        }
        observation.request = Some(Any::pack(&request));
        let Outcome::Refusal(refusal) = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest {
                observation: Some(reconciled(
                    &cancel_ack(instruction.request.clone()),
                    &observation,
                )),
            },
        ) else {
            panic!("refusal for {field}")
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::State);
    }
}

#[test]
fn cancel_reconciliation_requires_the_actual_not_found_acknowledgment() {
    let (mut runtime, program) = setup(0);
    assert!(matches!(
        invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 2 }),
        Outcome::Propose(_)
    ));
    let observation = terminal(&program, "Complete", 1, false);
    let outcome = invoke(
        &mut runtime,
        4,
        &protocol::JudgeRequest { observation: Some(Any::pack(&observation)) },
    );
    assert!(
        matches!(outcome, Outcome::Refusal(_)),
        "missing acknowledgment must refuse: {outcome:?}"
    );
}

#[test]
fn compound_reconciliation_rejects_wrong_acknowledgments_and_non_cancel_contexts() {
    use umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionRequest;
    use umpire_temporal_api::umpire::temporal::v1::ReconciledObservation;
    for case in [
        "missing_ack",
        "missing_current",
        "wrong_ack_type",
        "wrong_current_type",
        "malformed_ack",
        "malformed_current",
        "missing_ack_request",
        "wrong_method",
        "wrong_code",
        "no_result",
        "success",
        "namespace",
        "activity_id",
        "run_id",
        "invalid_status",
        "invalid_run_state",
        "no_pending",
        "read_pending",
        "reset",
    ] {
        let (mut runtime, program) = setup(0);
        let instruction = cancellation(&mut runtime);
        let mut ack = cancel_ack(instruction.request);
        let mut current = terminal(&program, "Complete", 1, false);
        match case {
            "wrong_method" => ack.method = "other".into(),
            "wrong_code" => ack.result = Some(ResultValue::Error(umpire_runtime::realization::Error { code: 3, message: String::new() })),
            "no_result" => ack.result = None,
            "missing_ack_request" => ack.request = None,
            "success" => ack.result = Some(ResultValue::Response(Any::pack(&umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse::default()))),
            "namespace" | "activity_id" | "run_id" => { let mut request: RequestCancelActivityExecutionRequest = ack.request.as_ref().unwrap_or_else(|| panic!("request")).unpack().unwrap_or_else(|error| panic!("{error:?}")); match case { "namespace" => request.namespace = "other".into(), "activity_id" => request.activity_id = "other".into(), _ => request.run_id = "other".into() } ack.request = Some(Any::pack(&request)); },
            "invalid_status" | "invalid_run_state" => {
                let Some(ResultValue::Response(value)) = current.result.take() else { panic!("response") };
                let mut response: DescribeActivityExecutionResponse = value.unpack().unwrap_or_else(|error| panic!("{error:?}"));
                let info = response.info.as_mut().unwrap_or_else(|| panic!("info"));
                if case == "invalid_status" { info.status = 999; } else { info.run_state = 999; }
                current.result = Some(ResultValue::Response(Any::pack(&response)));
            }
            "no_pending" => { let (fresh, _) = setup(0); runtime = fresh; },
            "read_pending" => { let (fresh, _) = setup(0); runtime = fresh; assert!(matches!(invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 1 }), Outcome::Propose(_))); },
            "reset" => { let _ = invoke(&mut runtime, 3, &protocol::ResetRequest::default()); },
            _ => {}
        }
        let mut pair = ReconciledObservation {
            acknowledged: Some(Any::pack(&ack)),
            current: Some(Any::pack(&current)),
        };
        match case {
            "missing_ack" => pair.acknowledged = None,
            "missing_current" => pair.current = None,
            "wrong_ack_type" => {
                pair.acknowledged = Some(Any::pack(&protocol::ResetRequest::default()));
            }
            "wrong_current_type" => {
                pair.current = Some(Any::pack(&protocol::ResetRequest::default()));
            }
            "malformed_current" => {
                pair.current.as_mut().unwrap_or_else(|| panic!("current")).value = vec![255];
            }
            "malformed_ack" => {
                pair.acknowledged.as_mut().unwrap_or_else(|| panic!("ack")).value = vec![255];
            }
            _ => {}
        }
        let outcome = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&pair)) },
        );
        assert!(matches!(outcome, Outcome::Refusal(_)), "{case}: {outcome:?}");
    }
}

#[test]
fn reconciliation_context_changes_only_after_publication_and_clears_on_final_ack() {
    let (mut runtime, program) = setup(0);
    let (proposal, transaction) =
        runtime.prepare(6, &protocol::ProposeRequest { seed: 2 }.encode_to_vec());
    let Some(Outcome::Propose(proposal)) = proposal.outcome else { panic!("cancel") };
    let instruction: umpire_temporal_api::umpire::temporal::v1::TemporalInstruction = proposal
        .request
        .unwrap_or_else(|| panic!("instruction"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let ack = cancel_ack(instruction.request);
    let request = protocol::JudgeRequest {
        observation: Some(reconciled(&ack, &terminal(&program, "Complete", 1, false))),
    };
    assert!(
        matches!(invoke(&mut runtime, 4, &request), Outcome::Refusal(_)),
        "unpublished proposal cannot authorize compound evidence"
    );
    runtime.commit(transaction.unwrap_or_else(|| panic!("proposal transaction")));
    let (unpublished, _) = runtime.prepare(4, &request.encode_to_vec());
    assert!(matches!(unpublished.outcome, Some(Outcome::Judge(_))));
    assert!(matches!(
        invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 0 }),
        Outcome::Refusal(_)
    ));
    let Outcome::Judge(judgment) = invoke(&mut runtime, 4, &request) else {
        panic!("reconciliation")
    };
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = judgment.judgment else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Worker::Poll(1)", "Worker::RespondCompleted"]);
    let _ = invoke(&mut runtime, 4, &protocol::JudgeRequest { observation: Some(Any::pack(&ack)) });
    assert!(
        matches!(invoke(&mut runtime, 4, &request), Outcome::Refusal(_)),
        "final acknowledgment consumes reconciliation context"
    );
}

#[test]
fn terminal_failure_validates_current_last_failure_against_the_program() {
    use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
    for variation in ["matching", "wrong_message", "wrong_type", "missing"] {
        let (mut runtime, program) = setup(2);
        let mut observation = terminal(&program, "Fail", 1, false);
        let Some(ResultValue::Response(value)) = observation.result.take() else {
            panic!("response")
        };
        let mut response: DescribeActivityExecutionResponse =
            value.unpack().unwrap_or_else(|error| panic!("{error:?}"));
        let Some(Value::Failure(failure)) =
            response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
        else {
            panic!("failure")
        };
        let mut last_failure = failure.clone();
        last_failure.source = "GoSDK".into();
        last_failure.stack_trace = "handler.go:42".into();
        match variation {
            "wrong_message" => last_failure.message = "different current failure".into(),
            "wrong_type" => {
                let Some(FailureInfo::ApplicationFailureInfo(info)) =
                    &mut last_failure.failure_info
                else {
                    panic!("application failure")
                };
                info.r#type = "OtherFailure".into();
            }
            _ => {}
        }
        response.info.as_mut().unwrap_or_else(|| panic!("info")).last_failure =
            (variation != "missing").then_some(last_failure);
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
            variation != "matching",
            "{variation}: {judgment:?}"
        );
    }
}

fn accept_cancellation(runtime: &mut impl Dispatch) {
    use umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse;
    let instruction = cancellation(runtime);
    let acknowledged = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: instruction.request,
        result: Some(ResultValue::Response(Any::pack(
            &RequestCancelActivityExecutionResponse::default(),
        ))),
    };
    let Outcome::Judge(judgment) =
        invoke(runtime, 4, &protocol::JudgeRequest { observation: Some(Any::pack(&acknowledged)) })
    else {
        panic!("cancellation")
    };
    assert!(matches!(judgment.judgment, Some(protocol::judge_response::Judgment::Explained(_))));
}

#[test]
fn describe_rejects_failure_history_and_outcomes_that_contradict_the_observed_phase() {
    use umpire_temporal::activity::Phase;
    use umpire_temporal_api::temporal::api::failure::v1::{
        CanceledFailureInfo, Failure, failure::FailureInfo,
    };
    for (phase, variation, wrong) in [
        (Phase::Completed, "last_failure", true),
        (Phase::Canceled, "last_failure", true),
        (Phase::Started, "last_failure", true),
        (Phase::Scheduled, "last_failure", true),
        (Phase::Started, "outcome", true),
        (Phase::Scheduled, "outcome", true),
        (Phase::Canceled, "result", true),
        (Phase::Canceled, "application_failure", true),
        (Phase::Canceled, "canceled_failure", false),
        (Phase::Completed, "retry_history", false),
    ] {
        let (mut runtime, program) = setup(u64::from(variation == "retry_history"));
        if phase == Phase::Canceled {
            accept_cancellation(&mut runtime);
        }
        let mut observation = if phase == Phase::Completed {
            terminal(
                &program,
                if variation == "retry_history" { "RetryThenComplete" } else { "Complete" },
                if variation == "retry_history" { 2 } else { 1 },
                false,
            )
        } else {
            common::describe(
                1,
                if phase == Phase::Canceled { 4 } else { 1 },
                if phase == Phase::Canceled {
                    0
                } else if phase == Phase::Started {
                    2
                } else {
                    1
                },
            )
        };
        let Some(ResultValue::Response(value)) = observation.result.take() else {
            panic!("response")
        };
        let mut response: DescribeActivityExecutionResponse =
            value.unpack().unwrap_or_else(|error| panic!("{error:?}"));
        let failed: RespondActivityTaskFailedRequest =
            row(&program, "Fail", 1).unpack().unwrap_or_else(|error| panic!("{error:?}"));
        match variation {
            "last_failure" => {
                response.info.as_mut().unwrap_or_else(|| panic!("info")).last_failure =
                    failed.failure;
            }
            "outcome" | "application_failure" => {
                response.outcome = Some(ActivityExecutionOutcome {
                    value: Some(Value::Failure(
                        failed.failure.unwrap_or_else(|| panic!("failure")),
                    )),
                    ..Default::default()
                });
            }
            "result" => {
                let completed: RespondActivityTaskCompletedRequest = row(&program, "Complete", 1)
                    .unpack()
                    .unwrap_or_else(|error| panic!("{error:?}"));
                response.outcome = Some(ActivityExecutionOutcome {
                    value: Some(Value::Result(
                        completed.result.unwrap_or_else(|| panic!("result")),
                    )),
                    ..Default::default()
                });
            }
            "canceled_failure" => {
                response.outcome = Some(ActivityExecutionOutcome {
                    value: Some(Value::Failure(Failure {
                        failure_info: Some(FailureInfo::CanceledFailureInfo(
                            CanceledFailureInfo::default(),
                        )),
                        ..Default::default()
                    })),
                    ..Default::default()
                });
            }
            _ => {}
        }
        observation.result = Some(ResultValue::Response(Any::pack(&response)));
        let Outcome::Judge(judgment) = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&observation)) },
        ) else {
            panic!("judgment")
        };
        assert_eq!(
            matches!(judgment.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))),
            wrong,
            "{phase:?}/{variation}: {judgment:?}"
        );
    }
}

#[test]
fn unheld_open_cancellation_is_refused_before_a_proposal_becomes_pending() {
    for (started, seed) in [(true, 18), (false, 0)] {
        let (mut runtime, program) = setup(0);
        if started {
            let Outcome::Judge(judgment) = invoke(
                &mut runtime,
                4,
                &protocol::JudgeRequest {
                    observation: Some(Any::pack(&common::describe(1, 1, 2))),
                },
            ) else {
                panic!("delivery")
            };
            assert!(matches!(
                judgment.judgment,
                Some(protocol::judge_response::Judgment::Explained(_))
            ));
        }
        let Outcome::Refusal(refusal) = invoke(&mut runtime, 6, &protocol::ProposeRequest { seed })
        else {
            panic!("unheld open cancellation must refuse: started={started}, seed={seed}")
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::Unsupported);
        assert_eq!(
            refusal.message,
            "unheld cancellation of an open activity is outside the atomic cancellation Model"
        );
        let Outcome::Propose(read) = invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 1 })
        else {
            panic!("refusal must leave no pending proposal")
        };
        assert_eq!(read.kind, protocol::ProposalKind::Read as i32);
        let Outcome::Judge(judgment) = invoke(
            &mut runtime,
            4,
            &protocol::JudgeRequest {
                observation: Some(Any::pack(&terminal(&program, "Complete", 1, false))),
            },
        ) else {
            panic!("terminal evidence")
        };
        assert!(matches!(
            judgment.judgment,
            Some(protocol::judge_response::Judgment::Explained(_))
        ));
        let Outcome::Propose(cancel) =
            invoke(&mut runtime, 6, &protocol::ProposeRequest { seed: 0 })
        else {
            panic!("terminal cancellation remains realizable")
        };
        assert_eq!(cancel.action, "Client::Cancel");
    }
}
