use umpire_runtime::{AnyMessage as _, capability::Identity, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;
#[path = "../registry/fixture.rs"]
mod fixture;
use fixture::specimen;
struct Omitted;
impl Identity for Omitted {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), Refusal> {
        Ok(())
    }
}
umpire_temporal::registry! {
    model { specimen }
    actors { enum Client { #[metadata(temporal(event = "event", evidence = event))] Ping } }
    body {
        #[registry(inputs = ())]
        impl Omitted {
            fn event(&self, _: &umpire_temporal::support::Observed, _: &[&umpire_runtime::Evidence<specimen::World>]) -> Result<umpire_runtime::EvidenceMatch, Refusal> { Ok(umpire_runtime::EvidenceMatch::Reject) }
        }
    }
}
fn main() {}
