//! A second catalog consumer with no system-extension dependency.
#![allow(dead_code)]

use umpire_catalog::{Catalog, Inspect, kinds::*};

struct Limit;
const ID_LIMIT: Limit = Limit;
struct Limits(usize);
impl Limits {
    const fn get(&self, _: &Limit) -> usize {
        self.0
    }
}

mod api {
    pub(crate) mod kettle {
        #[derive(Clone, prost::Message)]
        pub(crate) struct FillRequest {
            #[prost(string, tag = "1")]
            pub id: String,
            #[prost(string, tag = "2")]
            pub request_id: String,
        }
    }
}

#[derive(Catalog)]
#[mirror(request = "kettle.FillRequest", api = crate::api, context = Limits)]
struct Fill {
    #[rule(max_bytes = ID_LIMIT, rejects)]
    id: Text,
    #[mirror(idempotency_key)]
    #[rule(rejects)]
    request_id: Uuid,
}

#[test]
fn independent_context_controls_inspection_and_identifier_fill()
-> Result<(), Box<dyn std::error::Error>> {
    let mut request = api::kettle::FillRequest { id: "kettle".into(), ..Default::default() };
    assert!(Inspect::<Limits>::fill_ids(&mut request, 7)?);
    let id = request.request_id.clone();
    assert!(!Inspect::<Limits>::fill_ids(&mut request, 8)?);
    assert_eq!(request.request_id, id);
    let mut findings = Vec::new();
    request.inspect(&Limits(4), false, &mut findings)?;
    assert_eq!(findings.len(), 1);
    assert_eq!(findings[0].path, "kettle.FillRequest.id");
    assert_eq!(findings[0].constraint, "string.max_bytes");
    findings.clear();
    request.inspect(&Limits(6), false, &mut findings)?;
    assert_eq!(findings, []);
    Ok(())
}

#[test]
fn shared_message_transport_keeps_response_bytes_and_fills_only_requests()
-> Result<(), Box<dyn std::error::Error>> {
    use prost::Message;
    let request = api::kettle::FillRequest { id: "kettle".into(), ..Default::default() };
    let mut bytes = request.encode_to_vec();
    bytes.extend_from_slice(&[0x18, 1]);
    let response = umpire_catalog::annotate_message::<api::kettle::FillRequest, _>(
        &bytes,
        true,
        7,
        &Limits(4),
    )
    .map_err(|error| error.message)?;
    assert_eq!(response.bytes, bytes);
    assert_eq!(response.violations.len(), 1);
    let filled = umpire_catalog::annotate_message::<api::kettle::FillRequest, _>(
        &bytes,
        false,
        7,
        &Limits(6),
    )
    .map_err(|error| error.message)?;
    assert_eq!(filled.violations, [] as [String; 0]);
    let message = api::kettle::FillRequest::decode(filled.bytes.as_slice())?;
    assert_ne!(message.request_id, "");
    assert_eq!(message.id, request.id);
    let refusal = umpire_catalog::annotate_message::<api::kettle::FillRequest, _>(
        &[0xff],
        false,
        7,
        &Limits(6),
    )
    .err()
    .ok_or("malformed bytes were accepted")?;
    assert_eq!(refusal.code, umpire_runtime::protocol::RefusalCode::MalformedRequest as i32);
    Ok(())
}

#[test]
fn probe_recipe_retains_literal_runs_integrity_and_bound() -> Result<(), Box<dyn std::error::Error>>
{
    let bytes = b"xyaaaaaaaaaaaaaaaaz";
    let recipe = umpire_runtime::probe_recipe(bytes)?;
    assert_eq!(recipe.decoded_length, 19);
    assert_eq!(recipe.fnv64, 14_352_489_339_731_911_568);
    assert_eq!(
        recipe
            .segments
            .iter()
            .map(|segment| (segment.literal.as_slice(), segment.repeat))
            .collect::<Vec<_>>(),
        vec![(b"xy".as_slice(), 1), (b"a".as_slice(), 16), (b"z".as_slice(), 1)]
    );
    assert!(umpire_runtime::probe_recipe(&vec![0; 16 * 1024 * 1024 + 1]).is_err());
    Ok(())
}
