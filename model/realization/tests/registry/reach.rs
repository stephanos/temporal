use super::*;
use umpire_runtime::{
    EvidenceMatch, Guest, Runtime,
    protocol::RefusalCode,
    realization::{Observation, observation::Result as ResultValue},
    testing::{GuestExt, ResponseExt},
};

fn cancel() -> Observation {
    Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: Some(Any::pack(&RequestCancelActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "entity".into(),
            run_id: "run".into(),
            request_id: "request".into(),
            ..Default::default()
        })),
        result: Some(ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse {}))),
    }
}

fn refresh() -> Observation {
    Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution".into(),
        request: Some(Any::pack(&DescribeActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "entity".into(),
            run_id: "run".into(),
            ..Default::default()
        })),
        result: Some(ResultValue::Response(Any::pack(&DescribeActivityExecutionResponse {
            run_id: "run".into(),
            ..Default::default()
        }))),
    }
}

#[test]
fn runtime_reach_admits_typed_roots_and_preserves_prefixes_and_refusals() -> Result<(), Refusal> {
    let binding = Binding::default();
    let cursor = std::rc::Rc::clone(&binding.root_cursor);
    let rpc = Any::pack(&cancel());
    let event = Any::pack(&refresh());
    let mut guest = Guest::new(Runtime::new(specimen::World::new(4), binding));
    for request in [Any::default(), Any::pack(&RespondActivityTaskCompletedRequest::default())] {
        let malformed = request.type_url.is_empty();
        let mut wrong = cancel();
        wrong.request = Some(request);
        guest.judge(Any::pack(&wrong)).refusal(if malformed {
            RefusalCode::MalformedRequest
        } else {
            RefusalCode::Unsupported
        });
    }
    let mut unsupported = cancel();
    unsupported.method = "/unknown.Service/Other".into();
    guest.judge(Any::pack(&unsupported)).refusal(RefusalCode::Unsupported);
    let mut crossed = cancel();
    let mut request: RequestCancelActivityExecutionRequest =
        crossed.request.as_ref().unwrap_or_else(|| panic!("request")).unpack()?;
    request.run_id = "crossed".into();
    crossed.request = Some(Any::pack(&request));
    guest.judge(Any::pack(&crossed)).refusal(RefusalCode::MalformedRequest);
    let explained = guest.judge(rpc.clone()).explanation();
    assert_eq!(explained.steps, ["Client::Ping"]);
    assert_eq!(explained.outcomes, ["()"]);
    let binding = Binding::default();
    assert!(matches!(binding.reach_evidence(&rpc, &[])?, EvidenceMatch::Continue { cursor: 0 }));
    assert!(matches!(binding.reach_evidence(&event, &[])?, EvidenceMatch::Continue { cursor: 7 }));
    cursor.set(5);
    guest.judge(event.clone()).refusal(RefusalCode::State);
    cursor.set(0);
    let mut wrong = refresh();
    wrong.result =
        Some(ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse {})));
    guest.judge(Any::pack(&wrong)).refusal(RefusalCode::Unsupported);
    let explained = guest.judge(event).explanation();
    assert_eq!(explained.steps, ["Client::Ping", "Environment::Refresh"]);
    assert_eq!(explained.outcomes, ["()", "()"]);
    let mut rejected = cancel();
    rejected.result = Some(ResultValue::Error(umpire_runtime::realization::Error {
        code: 5,
        message: String::new(),
    }));
    assert!(matches!(binding.reach_evidence(&Any::pack(&rejected), &[])?, EvidenceMatch::Reject));
    let mut rejected_guest = Guest::new(Runtime::new(specimen::World::new(4), Binding::default()));
    rejected_guest.judge(Any::pack(&rejected)).unexplained();
    let completion = Any::pack(&Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskCompleted"
            .into(),
        request: Some(Any::pack(&RespondActivityTaskCompletedRequest {
            namespace: "ns".into(),
            task_token: vec![9],
            ..Default::default()
        })),
        result: Some(ResultValue::Response(Any::pack(&RespondActivityTaskCompletedResponse {}))),
    });
    assert!(matches!(
        binding.reach_evidence(&completion, &[])?,
        EvidenceMatch::Continue { cursor: 0 }
    ));
    binding.conflicting_root.set(true);
    let refusal = binding
        .reach_evidence(&completion, &[])
        .err()
        .unwrap_or_else(|| panic!("conflicting root admitted"));
    assert_eq!(refusal.code(), RefusalCode::State);
    let mut zero = Guest::new(Runtime::new(specimen::World::new(4), Binding::default()));
    zero.judge(Any::pack(&refresh())).unexplained();
    Ok(())
}
