//! Resource-specific annotation context for the independent reference service.
use crate::{KettleServiceMethod, config::Limits};
use umpire_catalog::AnnotationResult;
use umpire_runtime::protocol::Refusal;

pub(crate) fn annotate(
    method: &str,
    response: bool,
    bytes: &[u8],
    seed: u64,
) -> Result<AnnotationResult, Refusal> {
    annotate_kettle!(method, response, bytes, seed, limits)
}

fn limits(method: KettleServiceMethod, response: bool, bytes: &[u8]) -> Result<Limits, Refusal> {
    if !response && method == KettleServiceMethod::Fill {
        use prost::Message;
        let value = crate::wire::umpire::reference::kettle::v1::FillRequest::decode(bytes)
            .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
        Ok(Limits::for_kettle(&value.kettle_id))
    } else {
        Ok(Limits::default())
    }
}
