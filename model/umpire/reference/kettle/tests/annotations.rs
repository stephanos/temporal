//! The reference catalog uses the same typed transport for filling and scoped validation.
use prost::Message;
use umpire_reference_kettle::{annotation_runtime, wire::umpire::reference::kettle::v1 as k};
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    Dispatch,
    annotations::{AnnotationRequest, AnnotationResponse},
    prost_types::Any,
    protocol::{self, response::Outcome},
};

type Result<T = ()> = std::result::Result<T, Box<dyn std::error::Error>>;

fn annotate(method: &str, response: bool, message: Vec<u8>) -> Result<AnnotationResponse> {
    let runtime = annotation_runtime();
    let request = protocol::AnnotateRequest {
        input: Some(Any::pack(&AnnotationRequest {
            method: format!("/umpire.reference.kettle.v1.KettleService/{method}"),
            response,
            message,
            seed: 233,
        })),
    };
    let (result, _) = runtime.prepare(7, &request.encode_to_vec());
    match result.outcome {
        Some(Outcome::Annotate(result)) => {
            result.output.ok_or("missing output")?.unpack().map_err(|error| error.message.into())
        }
        other => panic!("unexpected annotation response: {other:?}"),
    }
}

#[test]
fn generic_fill_preserves_supplied_keys_and_response_wire_bytes() -> Result {
    let request = k::FillRequest { kettle_id: "room".into(), ..Default::default() };
    let filled = annotate("Fill", false, request.encode_to_vec())?;
    assert_eq!(filled.violations, [] as [String; 0]);
    let request = k::FillRequest::decode(filled.message.as_slice())?;
    assert_ne!(request.idempotency_key, "");
    assert_eq!(annotate("Fill", false, filled.message.clone())?.message, filled.message);
    let mut response =
        k::FillResponse { state: Some(k::State { temperature: 20, ..Default::default() }) }
            .encode_to_vec();
    response.extend_from_slice(&[0x98, 0x06, 1]);
    assert_eq!(annotate("Fill", true, response.clone())?.message, response);
    Ok(())
}

#[test]
fn scoped_limits_and_nested_predicates_inspect_actual_wire_values() -> Result {
    use k::fill_request::{
        FillOptions,
        fill_options::{Source, WaterKind},
    };
    let options = FillOptions {
        water_kind: WaterKind::Tap as i32,
        source: Some(Source::Tap("sink".into())),
        tags: vec!["a".into(), "b".into(), "c".into()],
        ..Default::default()
    };
    let request = |id: &str, options| {
        k::FillRequest {
            kettle_id: id.into(),
            options: Some(options),
            idempotency_key: "supplied".into(),
        }
        .encode_to_vec()
    };
    assert_eq!(
        annotate("Fill", false, request("room", options.clone()))?.violations,
        [] as [String; 0]
    );
    let scoped = annotate("Fill", false, request("compact", options.clone()))?;
    assert_eq!(scoped.violations.len(), 1);
    assert!(scoped.violations[0].contains("max_items"));
    for source in [Some(Source::Tap("  ".into())), Some(Source::Cartridge(vec![1]))] {
        let bad = FillOptions { source, ..options.clone() };
        assert_ne!(annotate("Fill", false, request("room", bad))?.violations, [] as [String; 0]);
    }
    Ok(())
}
