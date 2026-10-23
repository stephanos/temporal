//! Shared annotation dispatch applied to the independent reference service.
use crate::{
    KettleServiceMethod,
    api::{RpcRole, classify},
    config::Limits,
};
use umpire_catalog::annotate_message;
use umpire_runtime::{
    annotations::{AnnotationRequest, AnnotationResponse},
    google::protobuf::Any,
    protocol::Refusal,
};

pub(crate) fn dispatch(input: &Any) -> Result<Any, Refusal> {
    if input.type_url
        == Any::pack(&crate::wire::umpire::reference::kettle::v1::ProbeRequest::default()).type_url
    {
        let request = input.unpack()?;
        let response =
            crate::probes::run(&request).map_err(|error| Refusal::malformed(error.to_string()))?;
        return Ok(Any::pack(&response));
    }
    let request: AnnotationRequest = input.unpack()?;
    let method = request
        .method
        .strip_prefix("/umpire.reference.kettle.v1.KettleService/")
        .or_else(|| {
            (!request.method.contains('/') && !request.method.contains('.'))
                .then_some(request.method.as_str())
        })
        .and_then(KettleServiceMethod::parse)
        .ok_or_else(|| Refusal::unsupported("annotation RPC is not classified"))?;
    if let RpcRole::OutOfScope(reason) = classify(method) {
        return Ok(Any::pack(&AnnotationResponse {
            message: request.message,
            violations: vec![],
            out_of_scope: reason.into(),
        }));
    }
    let limits = if !request.response && method == KettleServiceMethod::Fill {
        use prost::Message;
        let value = crate::wire::umpire::reference::kettle::v1::FillRequest::decode(
            request.message.as_slice(),
        )
        .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
        Limits::for_kettle(&value.kettle_id)
    } else {
        Limits::default()
    };
    let result = dispatch_kettle_annotations!(
        method,
        request.response,
        M,
        { annotate_message::<M, _>(&request.message, request.response, request.seed, &limits) },
        Err(Refusal::unsupported("classified RPC has no annotation message"))
    )?;
    Ok(Any::pack(&AnnotationResponse {
        message: result.bytes,
        violations: result.violations,
        out_of_scope: result.out_of_scope.unwrap_or_default(),
    }))
}
