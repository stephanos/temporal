use umpire_runtime::{
    AnyMessage as _,
    capability::Identity,
    prost_types::Any,
    protocol::{Refusal, RefusalCode},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal::registry::Registry as _;
use umpire_temporal_api::temporal::api::workflowservice::v1::{
    RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
};

#[umpire_dsl::machine]
pub(crate) mod empty {
    use umpire_dsl::{Action, State};
    #[derive(Action)]
    pub(crate) enum Nobody {}
    #[derive(State)]
    pub(crate) struct Model {
        enabled: bool,
    }
    impl Model {
        #[on(Nobody)]
        #[expect(
            clippy::unused_self,
            clippy::needless_pass_by_value,
            reason = "an uninhabited action has no transition"
        )]
        fn nobody(&mut self, action: Nobody) {
            match action {}
        }
    }
}

struct Missing;
impl Identity for Missing {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), Refusal> {
        Ok(())
    }
}

empty::action_metadata!(umpire_temporal::registry; empty; {
    #[registry(inputs = ())]
    #[relations(rpc = WorkflowService::RequestCancelActivityExecution)]
    impl Missing {}
});

#[test]
fn selected_relation_hook_refuses_missing_registration_and_preserves_non_ok_route() {
    let mut observation = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: Some(Any::pack(&RequestCancelActivityExecutionRequest::default())),
        result: Some(ResultValue::Response(Any::pack(
            &RequestCancelActivityExecutionResponse::default(),
        ))),
    };
    let refusal = refused(
        Missing.check_registered(&Any::pack(&observation), &[]),
        "missing hook must refuse",
    );
    assert_eq!(refusal.code(), RefusalCode::Unsupported);
    assert_eq!(refusal.message, "missing required registered relation hook");
    observation.result = Some(ResultValue::Error(umpire_runtime::realization::Error {
        code: 5,
        message: String::new(),
    }));
    assert!(Missing.check_registered(&Any::pack(&observation), &[]).is_ok());
}

fn refused<T>(result: Result<T, Refusal>, context: &str) -> Refusal {
    match result {
        Ok(_) => panic!("{context}"),
        Err(refusal) => refusal,
    }
}
