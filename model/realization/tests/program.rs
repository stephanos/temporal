//! Configured worker programs are issued before any system interaction.

#![forbid(unsafe_code)]

use prost::Message;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    Dispatch,
    prost_types::Any,
    protocol::{self, response::Outcome},
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;

mod common;

#[test]
fn configured_reset_issues_a_deterministic_program_and_unconfigured_reset_does_not() {
    use umpire_temporal_api::umpire::temporal::v1::WorkerProgram;

    let runtime = annotation_runtime();
    let (empty, _) = runtime.prepare(3, &[]);
    let Some(Outcome::Reset(empty)) = empty.outcome else { panic!("Reset") };
    assert!(empty.program.is_none());
    assert_eq!(empty.proof_contract, protocol::ProofContract::WholePrefixV1 as i32);
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
    assert_eq!(reset.proof_contract, protocol::ProofContract::WholePrefixV1 as i32);
    let _: WorkerProgram = reset
        .program
        .unwrap_or_else(|| panic!("configured Reset must issue the worker program"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
}

#[test]
#[expect(
    clippy::print_stderr,
    reason = "retain actual complete wire sizes in qualification evidence"
)]
#[expect(
    clippy::too_many_lines,
    reason = "one complete wire artifact round-trip and independent coverage control"
)]
fn program_covers_the_complete_model_manifest_with_exact_typed_fill() {
    use umpire_temporal::{
        activity,
        policy::{Choice, Domain, Manifest, TaskKey},
    };
    use umpire_temporal_api::{
        temporal::api::workflowservice::v1::{
            RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        },
        umpire::temporal::v1::{ActivityTaskKey, WorkerProgram},
    };
    for (maximum, policies, rows, domains) in
        [(1, 15, 39, 12), (2, 85, 416, 128), (3, 435, 3211, 988)]
    {
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
        let start: StartActivityExecutionRequest = common::start(maximum)
            .request
            .unwrap_or_else(|| panic!("start"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        let world = activity::World::new(
            u8::try_from(maximum).unwrap_or_else(|_| panic!("maximum")),
            None,
            None,
            None,
            None,
        );
        let manifest = Manifest::derive(
            &world,
            Domain { heartbeats_per_task: 1 },
            umpire_runtime::policy::Limits {
                depth: 32,
                work: 300_000_000,
                assignments: 150_000,
                catalog: umpire_dsl::CATALOG_LIMIT,
            },
        )
        .unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(manifest.policies().len(), policies);
        assert_eq!(program.rules.len(), rows);
        assert_eq!(program.continuation_domains.len(), domains);
        let envelope = protocol::Response {
            outcome: Some(Outcome::Reset(protocol::ResetResponse::current(Some(Any::pack(
                &program,
            ))))),
        };
        assert!(envelope.encoded_len() <= 1 << 20);
        eprintln!(
            "production-wire maximum={maximum} policies={policies} rows={rows} domains={domains} program_bytes={} reset_bytes={}",
            program.encoded_len(),
            envelope.encoded_len()
        );
        if maximum == 3 {
            let mut oversized = start.clone();
            oversized.activity_type.as_mut().unwrap_or_else(|| panic!("activity type")).name =
                "X".repeat(64);
            let (refused, transaction) = runtime.prepare(
                3,
                &protocol::ResetRequest { configuration: Some(Any::pack(&oversized)) }
                    .encode_to_vec(),
            );
            let Some(Outcome::Refusal(refusal)) = refused.outcome else {
                panic!("oversized complete Reset must refuse");
            };
            assert!(refusal.message.contains("complete policy Reset exceeds ABI"));
            assert!(!refusal.message.contains("before wire assembly"));
            assert!(transaction.is_none());
            for length in [256, 64 * 1024] {
                oversized.activity_type.as_mut().unwrap_or_else(|| panic!("activity type")).name =
                    "X".repeat(length);
                let reset = protocol::ResetRequest { configuration: Some(Any::pack(&oversized)) };
                assert!(reset.encoded_len() < 1 << 20);
                let (refused, transaction) = runtime.prepare(3, &reset.encode_to_vec());
                let Some(Outcome::Refusal(refusal)) = refused.outcome else {
                    panic!("pre-expansion size refusal");
                };
                assert!(refusal.message.contains("before wire assembly"));
                assert!(refusal.message.contains("repeated_activity_type_bytes="));
                assert!(transaction.is_none());
            }
        }
        let expected = manifest.policies().iter().flat_map(|policy| {
            policy.rows().iter().map(move |(key, choice)| (policy.name(), key, *choice))
        });
        for (row, (name, task, choice)) in program.rules.iter().zip(expected) {
            let key: ActivityTaskKey = row
                .key
                .as_ref()
                .unwrap_or_else(|| panic!("key"))
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
            assert_eq!(key.scenario, name);
            assert_eq!(key.attempt, i32::from(task.attempt));
            assert_eq!(key.activity_type, "ModelSlice");
            assert_eq!(key.heartbeat_history, task.heartbeat_history);
            let exact = choice
                .fill(
                    &world,
                    &start,
                    &TaskKey {
                        attempt: task.attempt,
                        heartbeat_history: task.heartbeat_history.clone(),
                    },
                )
                .unwrap_or_else(|e| panic!("{e:?}"));
            assert_eq!(row.respond, [exact]);
            let [reply] = row.respond.as_slice() else { panic!("one response") };
            if matches!(choice, Choice::Fail | Choice::Retry) {
                let failed: RespondActivityTaskFailedRequest =
                    reply.unpack().unwrap_or_else(|e| panic!("{e:?}"));
                assert_eq!(failed.task_token, [] as [u8; 0]);
                assert_eq!(failed.identity, "");
                let failure = failed.failure.unwrap_or_else(|| panic!("failure"));
                assert_eq!(failure.message, if choice == Choice::Fail { "Fail" } else { "Retry" });
                if choice == Choice::Fail {
                    assert_eq!(
                        reply.value,
                        b"\x12\x1f\x0a\x04Fail\x2a\x17\x0a\x13MiniActivityFailure\x10\x01"
                    );
                }
                let Some(umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(info)) = failure.failure_info else { panic!("application failure") };
                assert_eq!(info.non_retryable, choice == Choice::Fail);
                assert_eq!(info.r#type, "MiniActivityFailure");
            } else if choice == Choice::Complete {
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
        let expected_domains: Vec<_> = manifest
            .policies()
            .iter()
            .flat_map(|policy| {
                policy.successors().iter().map(move |(key, bits)| {
                    Any::pack(&umpire_temporal_api::umpire::temporal::v1::ActivityTaskSuccessors {
                        key: Some(ActivityTaskKey {
                            scenario: policy.name().into(),
                            attempt: i32::from(key.attempt),
                            activity_type: "ModelSlice".into(),
                            heartbeat_history: key.heartbeat_history.clone(),
                        }),
                        cancel_requested: bits.clone(),
                    })
                })
            })
            .collect();
        assert_eq!(program.continuation_domains, expected_domains);
    }
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "one complete bootstrap, admission, read and held-action transaction"
)]
fn bootstrap_scenario_is_in_header_and_reads_and_seeded_cancel_hold_are_action_distinct() {
    use umpire_temporal_api::umpire::temporal::v1::{ObserveActivity, TemporalInstruction};
    for witness in ["Complete", "RetryThenComplete", "Fail"] {
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
        let Some(Outcome::Reset(issued)) = response.outcome else { panic!("reset") };
        let program: umpire_temporal_api::umpire::temporal::v1::WorkerProgram = issued
            .program
            .unwrap_or_else(|| panic!("program"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        let mut names = Vec::new();
        for rule in &program.rules {
            let key: umpire_temporal_api::umpire::temporal::v1::ActivityTaskKey = rule
                .key
                .as_ref()
                .unwrap_or_else(|| panic!("key"))
                .unpack()
                .unwrap_or_else(|e| panic!("{e:?}"));
            if names.last() != Some(&key.scenario) {
                names.push(key.scenario);
            }
        }
        assert_eq!(names.len(), 435);
        let (_, seed) = common::terminal_witness(&program, witness);
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
            names[usize::try_from(seed).unwrap_or_else(|_| panic!("seed"))].as_bytes()
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
            if !held {
                let Some(Outcome::Refusal(refusal)) = cancel.outcome else {
                    panic!("unheld cancellation refusal")
                };
                assert_eq!(refusal.code(), protocol::RefusalCode::Unsupported);
                continue;
            }
            let Some(Outcome::Propose(cancel)) = cancel.outcome else {
                panic!("held cancellation")
            };
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

#[test]
fn bootstrap_selects_every_canonical_policy_including_heartbeat_policies() {
    use umpire_temporal_api::umpire::temporal::v1::{ActivityTaskKey, WorkerProgram};
    let mut runtime = annotation_runtime();
    let (response, transaction) = runtime.prepare(
        3,
        &protocol::ResetRequest { configuration: common::start(3).request }.encode_to_vec(),
    );
    let Some(Outcome::Reset(issued)) = response.outcome else { panic!("reset") };
    let program: WorkerProgram = issued
        .program
        .unwrap_or_else(|| panic!("program"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    runtime.commit(transaction.unwrap_or_else(|| panic!("reset transaction")));
    let mut names = Vec::new();
    for rule in &program.rules {
        let key: ActivityTaskKey = rule
            .key
            .as_ref()
            .unwrap_or_else(|| panic!("key"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        if names.last() != Some(&key.scenario) {
            names.push(key.scenario);
        }
    }
    assert_eq!(names.len(), 435);
    for (seed, name) in names.iter().chain(names.first()).enumerate() {
        let (response, _) = runtime.prepare(
            6,
            &protocol::ProposeRequest {
                seed: u64::try_from(seed).unwrap_or_else(|_| panic!("seed")),
            }
            .encode_to_vec(),
        );
        let Some(Outcome::Propose(proposal)) = response.outcome else { panic!("bootstrap") };
        let request: StartActivityExecutionRequest = proposal
            .request
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        let header = request.header.unwrap_or_else(|| panic!("header"));
        assert_eq!(header.fields["umpire-scenario"].data, name.as_bytes());
    }
}

#[test]
fn finite_history_codec_keeps_unknown_false_true_and_order() {
    use umpire_temporal_api::umpire::temporal::v1::{
        ActivityTaskKey, ActivityTaskSuccessors, WorkerProgram,
    };
    let mut encodings = std::collections::HashSet::new();
    for history in
        [vec![], vec![false], vec![true], vec![false, false], vec![false, true], vec![true, true]]
    {
        let key = ActivityTaskKey {
            scenario: "history".into(),
            attempt: 2,
            activity_type: "activity".into(),
            heartbeat_history: history,
        };
        let bytes = key.encode_to_vec();
        assert!(encodings.insert(bytes.clone()));
        assert_eq!(
            ActivityTaskKey::decode(bytes.as_slice()).unwrap_or_else(|e| panic!("{e}")),
            key
        );
        let program = WorkerProgram {
            namespace: "namespace".into(),
            task_queue: "queue".into(),
            continuation_domains: vec![Any::pack(&ActivityTaskSuccessors {
                key: Some(key),
                cancel_requested: vec![true],
            })],
            ..Default::default()
        };
        assert_eq!(
            Any::pack(&program).unpack::<WorkerProgram>().unwrap_or_else(|e| panic!("{e:?}")),
            program
        );
    }
}

#[test]
fn typed_worker_filler_packs_heartbeat_and_canceled_for_the_selected_attempt()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_dsl::Machine;
    use umpire_runtime::capability::{Fill, Identity, Request};
    use umpire_temporal::{Realization, activity};
    use umpire_temporal_api::temporal::api::{
        common::v1::{Payload, Payloads},
        workflowservice::v1::{
            RecordActivityTaskHeartbeatRequest, RespondActivityTaskCanceledRequest,
        },
    };
    type Action = <activity::World as Machine>::Action;
    type Inputs = <Realization as Fill<Action>>::Inputs;
    let binding = Realization::default();
    let world = activity::World::new(3, None, None, None, None);
    let identity = binding
        .identify(&common::start(3).request.ok_or("start request")?)
        .map_err(|e| e.message)?;
    let details = Some(Payloads {
        payloads: vec![Payload { data: b"progress".to_vec(), ..Default::default() }],
    });
    let inputs = Inputs::Worker { attempt: 2, details: details.clone() };
    let heartbeat = binding
        .fill(&Action::from(activity::Worker::Heartbeat), &world, &inputs, &identity)
        .map_err(|e| e.message)?
        .pack_request();
    let heartbeat: RecordActivityTaskHeartbeatRequest =
        heartbeat.unpack().map_err(|e| e.message)?;
    assert_eq!(heartbeat.details, details);
    assert_eq!(heartbeat.namespace, "");
    assert_eq!(heartbeat.task_token, [] as [u8; 0]);
    let canceled = binding
        .fill(&Action::from(activity::Worker::RespondCanceled(2)), &world, &inputs, &identity)
        .map_err(|e| e.message)?
        .pack_request();
    let canceled: RespondActivityTaskCanceledRequest = canceled.unpack().map_err(|e| e.message)?;
    assert_eq!(canceled.details, details);
    assert_eq!(canceled.namespace, "");
    assert_eq!(canceled.task_token, [] as [u8; 0]);
    assert!(
        binding
            .fill(&Action::from(activity::Worker::RespondCanceled(3)), &world, &inputs, &identity)
            .is_err()
    );
    assert!(
        binding
            .fill(
                &Action::from(activity::Worker::Heartbeat),
                &world,
                &Inputs::Worker { attempt: 4, details: None },
                &identity
            )
            .is_err()
    );
    assert!(
        binding
            .fill(&Action::from(activity::Worker::RespondCompleted), &world, &inputs, &identity)
            .is_err()
    );
    Ok(())
}
