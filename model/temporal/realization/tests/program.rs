//! Configured worker programs are issued before any system interaction.

use prost::Message;
use umpire_runtime::{
    Dispatch,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;

mod common;

#[test]
fn configured_reset_issues_a_deterministic_program_and_unconfigured_reset_does_not() {
    let runtime = annotation_runtime();
    let (empty, _) = runtime.prepare(3, &[]);
    let Some(Outcome::Reset(empty)) = empty.outcome else { panic!("Reset") };
    assert_eq!(empty.encode_to_vec(), [] as [u8; 0]);
    let start: StartActivityExecutionRequest = common::start(3)
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let configuration = protocol::ResetRequest { configuration: Some(Any::pack(&start)) };
    let (first, _) = runtime.prepare(3, &configuration.encode_to_vec());
    let (second, _) = runtime.prepare(3, &configuration.encode_to_vec());
    assert_eq!(first, second);
    let Some(Outcome::Reset(reset)) = first.outcome else { panic!("Reset") };
    assert!(!reset.encode_to_vec().is_empty(), "configured Reset must issue the worker program");
}

#[test]
fn program_covers_every_scenario_and_admitted_attempt_with_the_reply_policy() {
    use umpire_temporal_api::{
        temporal::api::workflowservice::v1::{
            RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        },
        umpire::temporal::v1::{ActivityTaskKey, WorkerProgram},
    };
    for maximum in 1..=3 {
        let runtime = annotation_runtime();
        let request = protocol::ResetRequest { configuration: common::start(maximum).request };
        let (response, _) = runtime.prepare(3, &request.encode_to_vec());
        let Some(Outcome::Reset(reset)) = response.outcome else { panic!("Reset") };
        let program: WorkerProgram = reset
            .program
            .unwrap_or_else(|| panic!("program"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(program.namespace, "model-slice");
        assert_eq!(program.task_queue, "queue");
        assert_eq!(
            program.rules.len(),
            usize::try_from(3 * maximum).unwrap_or_else(|_| panic!("maximum"))
        );
        for (row, (scenario, attempt)) in program.rules.iter().zip(
            ["Complete", "RetryThenComplete", "Fail"]
                .into_iter()
                .flat_map(|scenario| (1..=maximum).map(move |attempt| (scenario, attempt))),
        ) {
            let key: ActivityTaskKey = row
                .key
                .as_ref()
                .unwrap_or_else(|| panic!("key"))
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
            assert_eq!(key.scenario, scenario);
            assert_eq!(key.attempt, attempt);
            assert_eq!(key.activity_type, "ModelSlice");
            let [reply] = row.respond.as_slice() else { panic!("one response") };
            if scenario == "Fail" || scenario == "RetryThenComplete" && attempt == 1 {
                let failed: RespondActivityTaskFailedRequest =
                    reply.unpack().unwrap_or_else(|e| panic!("{e:?}"));
                assert_eq!(failed.task_token, [] as [u8; 0]);
                assert_eq!(failed.identity, "");
                let failure = failed.failure.unwrap_or_else(|| panic!("failure"));
                assert_eq!(failure.message, if scenario == "Fail" { "Fail" } else { "Retry" });
                if scenario == "Fail" {
                    assert_eq!(
                        reply.value,
                        b"\x12\x1f\x0a\x04Fail\x2a\x17\x0a\x13MiniActivityFailure\x10\x01"
                    );
                }
                let Some(umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(info)) = failure.failure_info else { panic!("application failure") };
                assert_eq!(info.non_retryable, scenario == "Fail");
                assert_eq!(info.r#type, "MiniActivityFailure");
            } else {
                let complete: RespondActivityTaskCompletedRequest =
                    reply.unpack().unwrap_or_else(|e| panic!("{e:?}"));
                assert_eq!(complete.task_token, [] as [u8; 0]);
                assert_eq!(complete.identity, "");
                let result = complete.result.unwrap_or_else(|| panic!("result"));
                let [payload] = result.payloads.as_slice() else { panic!("one result") };
                assert_eq!(payload.data, b"Complete");
                assert_eq!(
                    reply.value,
                    b"\x12\x26\x0a\x24\x0a\x18\x0a\x08encoding\x12\x0cbinary/plain\x12\x08Complete"
                );
                assert_eq!(payload.metadata.get("encoding"), Some(&b"binary/plain".to_vec()));
                assert_eq!(payload.external_payloads.len(), 0);
            }
        }
    }
}

#[test]
fn bootstrap_scenario_is_in_header_and_reads_and_seeded_cancel_hold_are_action_distinct() {
    use umpire_temporal_api::umpire::temporal::v1::{ObserveActivity, TemporalInstruction};
    for seed in 0..3 {
        let mut runtime = annotation_runtime();
        let start = common::start(3);
        let mut configuration: StartActivityExecutionRequest = start
            .request
            .unwrap_or_else(|| panic!("configuration"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        configuration.input = Some(umpire_temporal_api::temporal::api::common::v1::Payloads {
            payloads: vec![umpire_temporal_api::temporal::api::common::v1::Payload {
                data: b"application input".to_vec(),
                ..Default::default()
            }],
        });
        let input = configuration.input.clone();
        let (response, reset) = runtime.prepare(
            3,
            &protocol::ResetRequest { configuration: Some(Any::pack(&configuration)) }
                .encode_to_vec(),
        );
        assert!(matches!(response.outcome, Some(Outcome::Reset(_))));
        runtime.commit(reset.unwrap_or_else(|| panic!("reset transaction")));
        let (response, bootstrap) =
            runtime.prepare(6, &protocol::ProposeRequest { seed }.encode_to_vec());
        let Some(Outcome::Propose(proposal)) = response.outcome else { panic!("bootstrap") };
        let filled: StartActivityExecutionRequest = proposal
            .request
            .as_ref()
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        let scenario = filled
            .header
            .unwrap_or_else(|| panic!("header"))
            .fields
            .get("umpire-scenario")
            .unwrap_or_else(|| panic!("scenario"))
            .data
            .clone();
        assert_eq!(
            scenario,
            [b"Complete".as_slice(), b"RetryThenComplete".as_slice(), b"Fail".as_slice()]
                [usize::try_from(seed).unwrap_or_else(|_| panic!("seed"))]
        );
        assert_eq!(filled.input, input);
        runtime.commit(bootstrap.unwrap_or_else(|| panic!("bootstrap transaction")));
        let mut evidence = common::start(3);
        evidence.request = proposal.request;
        let (judgment, admission) = runtime.prepare(
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&evidence)) }.encode_to_vec(),
        );
        assert!(matches!(judgment.outcome, Some(Outcome::Judge(_))));
        runtime.commit(admission.unwrap_or_else(|| panic!("admission transaction")));
        let (read, read_transaction) =
            runtime.prepare(6, &protocol::ProposeRequest { seed: 1 }.encode_to_vec());
        let Some(Outcome::Propose(read)) = read.outcome else { panic!("read") };
        assert_eq!(read.kind, protocol::ProposalKind::Read as i32);
        assert_eq!(read.action, "");
        let instruction: ObserveActivity = read
            .request
            .unwrap_or_else(|| panic!("read request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(instruction.read_timeout_ms, 30_000);
        for (seed, held) in [(0, false), (2, true)] {
            let (cancel, _) =
                runtime.prepare(6, &protocol::ProposeRequest { seed }.encode_to_vec());
            let Some(Outcome::Propose(cancel)) = cancel.outcome else { panic!("cancel") };
            assert_eq!(cancel.kind, protocol::ProposalKind::Action as i32);
            assert_eq!(cancel.action, "Client::Cancel");
            let instruction: TemporalInstruction = cancel
                .request
                .unwrap_or_else(|| panic!("cancel request"))
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
            assert_eq!(instruction.hold_worker_polls, held);
        }
        runtime.commit(read_transaction.unwrap_or_else(|| panic!("read transaction")));
        let (scheduled, _) = runtime.prepare(
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(&common::describe(1, 1, 1))) }
                .encode_to_vec(),
        );
        let Some(Outcome::Judge(scheduled)) = scheduled.outcome else { panic!("scheduled read") };
        assert!(scheduled.continue_polling);
    }
}
