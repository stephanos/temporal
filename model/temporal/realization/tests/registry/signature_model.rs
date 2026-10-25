use umpire_runtime::{capability::Identity, prost_types::Any, protocol::Refusal};
#[umpire_dsl::machine]
pub(crate) mod specimen {
    use umpire_dsl::{Action, State};
    #[derive(Action)]
    pub(crate) enum Client {
        #[metadata(temporal(rpc = WorkflowService::RequestCancelActivityExecution,
            direction = client, request = RequestCancelActivityExecutionRequest,
            response = RequestCancelActivityExecutionResponse, fill = fill, evidence = evidence))]
        Ping,
    }
    #[derive(State)]
    pub(crate) struct Model {
        active: bool,
    }
    impl Model {
        #[on(Client)]
        fn client(&mut self, _: Client) {
            self.active = true;
        }
    }
}
pub(crate) struct Binding;
impl Identity for Binding {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), Refusal> {
        Ok(())
    }
}
