use super::super::*;
use super::common;
use crate::registry::Registry as _;

const OTHER: &str = "33333333-3333-4333-8333-333333333333";

fn receipt() -> Receipt {
    match ActivityEntity
        .admit(&Any::pack(&common::start(3)))
        .unwrap_or_else(|error| panic!("{error:?}"))
    {
        Admission::Admitted { receipt, .. } => receipt,
        Admission::Frozen { refusal, .. } => panic!("{refusal:?}"),
    }
}

fn mismatch(method: &str, target: &str, source: &str) -> String {
    let rpc = format!("temporal.api.workflowservice.v1.WorkflowService/{method}");
    let target_root = if target == "Request" { "request" } else { "response" };
    let source_root = if source == "Receipt" { "receipt" } else { "request" };
    format!(
        "{rpc} State/Compare EchoMismatch declaration={rpc}/{target_root}:temporal.activity.ProposalContext/run_id:Echo:{source_root}.run_id target={target}.run_id source={source}.run_id expected=\"\\\"{}\\\"\" actual=\"\\\"{OTHER}\\\"\"",
        common::RUN
    )
}

fn assert_generated(refusal: &Refusal, method: &str, target: &str, source: &str) {
    assert_eq!(refusal.code(), RefusalCode::State);
    assert_eq!(refusal.message, mismatch(method, target, source));
}

#[test]
fn activity_start_postadmission_and_replay_hook() {
    let mut checked = Vec::new();
    let admission = ActivityEntity::admit_with_observer(&Any::pack(&common::start(3)), |audit| {
        checked.push((audit.rpc().to_owned(), audit.declarations().to_vec()));
    })
    .unwrap_or_else(|error| panic!("{error:?}"));
    assert_eq!(checked, vec![(
        "temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution".into(),
        vec!["temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution/response:temporal.activity.ProposalContext/run_id:Echo:receipt.run_id".into()],
    )]);
    let Admission::Admitted { receipt, .. } = admission else { panic!("admitted Start") };
    let baseline = common::start(3);
    assert!(ActivityEntity.replay(&receipt, &Any::pack(&baseline)).is_ok());
    let mut changed = baseline.clone();
    let Some(ResultValue::Response(raw)) = &baseline.result else { panic!("response") };
    let mut response: StartActivityExecutionResponse =
        raw.unpack().unwrap_or_else(|error| panic!("{error:?}"));
    response.run_id = OTHER.into();
    response.link = None;
    changed.result = Some(ResultValue::Response(Any::pack(&response)));
    assert_generated(
        &refused(ActivityEntity.replay(&receipt, &Any::pack(&changed)), "generated replay check"),
        "StartActivityExecution",
        "Response",
        "Receipt",
    );
    assert_eq!(baseline, common::start(3));
    assert_eq!(receipt.entity.run_id, common::RUN);
    let mut empty = common::start(3);
    empty.result = Some(ResultValue::Response(Any::pack(&StartActivityExecutionResponse {
        started: true,
        ..Default::default()
    })));
    assert!(ActivityEntity.admit(&Any::pack(&empty)).is_err());
}

#[test]
fn activity_describe_actual_request_and_receipt_hooks() {
    let receipt = receipt();
    let baseline = common::describe(1, 1, 1);
    let observed = crate::support::Observed::new(&Any::pack(&baseline))
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert!(ActivityEntity.validate(Some(&receipt), &observed).is_ok());
    for request_target in [true, false] {
        let mut changed = baseline.clone();
        if request_target {
            let mut request: DescribeActivityExecutionRequest = changed
                .request
                .as_ref()
                .unwrap_or_else(|| panic!("request"))
                .unpack()
                .unwrap_or_else(|error| panic!("{error:?}"));
            request.run_id = OTHER.into();
            changed.request = Some(Any::pack(&request));
        } else {
            let Some(ResultValue::Response(raw)) = &changed.result else { panic!("response") };
            let mut response: DescribeActivityExecutionResponse =
                raw.unpack().unwrap_or_else(|error| panic!("{error:?}"));
            response.run_id = OTHER.into();
            changed.result = Some(ResultValue::Response(Any::pack(&response)));
        }
        let observed = crate::support::Observed::new(&Any::pack(&changed))
            .unwrap_or_else(|error| panic!("{error:?}"));
        assert_generated(
            &refused(ActivityEntity.validate(Some(&receipt), &observed), "generated read check"),
            "DescribeActivityExecution",
            if request_target { "Request" } else { "Response" },
            if request_target { "Receipt" } else { "Request" },
        );
    }
    assert_eq!(baseline, common::describe(1, 1, 1));
    assert_eq!(receipt.entity.run_id, common::RUN);
}

#[test]
fn activity_cancel_registered_action_hook() {
    let receipt = receipt();
    let binding = Realization::configured(receipt.entity.clone(), None);
    let mut request = RequestCancelActivityExecutionRequest {
        namespace: receipt.entity.start.namespace.clone(),
        activity_id: receipt.entity.start.activity_id.clone(),
        run_id: common::RUN.into(),
        ..Default::default()
    };
    let observed = |request: &RequestCancelActivityExecutionRequest| {
        Any::pack(&Observation {
            method: CANCEL.into(),
            request: Some(Any::pack(request)),
            result: Some(ResultValue::Response(Any::pack(
                &RequestCancelActivityExecutionResponse::default(),
            ))),
        })
    };
    assert!(binding.check_registered(&observed(&request), &[]).is_ok());
    request.run_id = OTHER.into();
    assert_generated(
        &refused(
            binding.check_registered(&observed(&request), &[]),
            "generated registered action check",
        ),
        "RequestCancelActivityExecution",
        "Request",
        "Receipt",
    );
    let non_ok = Any::pack(&Observation {
        method: CANCEL.into(),
        request: Some(Any::pack(&request)),
        result: Some(ResultValue::Error(umpire_runtime::realization::Error {
            code: 5,
            message: String::new(),
        })),
    });
    let refusal = refused(binding.check_registered(&non_ok, &[]), "legacy correlation is retained");
    assert!(!refusal.message.contains("EchoMismatch"));
    assert_eq!(receipt.entity.run_id, common::RUN);
}

fn refused<T>(result: Result<T, Refusal>, context: &str) -> Refusal {
    match result {
        Ok(_) => panic!("{context}"),
        Err(refusal) => refusal,
    }
}

#[test]
fn activity_cancel_live_validate_hook() {
    let receipt = receipt();
    let judge = |request: &RequestCancelActivityExecutionRequest, ok: bool| {
        let result = if ok {
            ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse::default()))
        } else {
            ResultValue::Error(umpire_runtime::realization::Error {
                code: 5,
                message: String::new(),
            })
        };
        let observation = Observation {
            method: CANCEL.into(),
            request: Some(Any::pack(request)),
            result: Some(result),
        };
        let observed = crate::support::Observed::new(&Any::pack(&observation))
            .unwrap_or_else(|error| panic!("{error:?}"));
        ActivityEntity.validate(Some(&receipt), &observed)
    };
    let mut request = RequestCancelActivityExecutionRequest {
        namespace: receipt.entity.start.namespace.clone(),
        activity_id: receipt.entity.start.activity_id.clone(),
        run_id: common::RUN.into(),
        ..Default::default()
    };
    assert!(judge(&request, true).is_ok());
    assert!(judge(&request, false).is_ok());
    request.run_id = OTHER.into();
    assert_generated(
        &refused(judge(&request, true), "generated live Cancel check"),
        "RequestCancelActivityExecution",
        "Request",
        "Receipt",
    );
    let refusal =
        refused(judge(&request, false), "non-OK Cancel keeps the established correlation");
    assert_eq!(refusal.code(), RefusalCode::State);
    assert!(!refusal.message.contains("EchoMismatch"));
    request.run_id = "not-a-uuid".into();
    let refusal = refused(judge(&request, true), "a differing generated class defers");
    assert_eq!(refusal.code(), RefusalCode::State);
    assert!(!refusal.message.contains("declaration="), "{}", refusal.message);
}
