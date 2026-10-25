//! Additions to the real Kettle descriptor must refuse naming their missing declaration.
#![forbid(unsafe_code)]

use umpire_catalog_syntax::{catalogs, descriptor::Coverage, services::classifications};

#[path = "../../../catalog/syntax/tests/support/mod.rs"]
mod support;

#[test]
fn descriptor_additions_name_the_unreviewed_field_enum_and_method()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("../api.rs");
    let mut coverage = Coverage::default();
    for catalog in catalogs(source)? {
        let c = catalog.coverage();
        coverage.messages.extend(c.messages);
        coverage.enums.extend(c.enums);
    }
    for classification in
        classifications(source, "umpire.reference.kettle.v1.KettleService", "KettleServiceMethod")?
    {
        coverage.services.push(classification.coverage);
    }
    support::assert_additions_refused(
        include_bytes!(concat!(env!("OUT_DIR"), "/kettle.binpb")),
        &coverage,
        "umpire.reference.kettle.v1.FillRequest",
        "umpire.reference.kettle.v1.HeatingAction",
        "umpire.reference.kettle.v1.KettleService",
    )
}
