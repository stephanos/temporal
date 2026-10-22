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
        #[derive(Clone, Default)]
        pub(crate) struct FillRequest {
            pub id: String,
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
