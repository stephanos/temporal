//! Response-only raw checks supplement request normalization without duplicating inspection.

#![forbid(unsafe_code)]
#![allow(dead_code)]

use umpire_catalog::{Catalog, Inspect, kinds::*};

mod foundation {
    pub(crate) mod dc {
        pub(crate) struct Limits;
    }
}

#[derive(Clone, Default)]
struct Wire {
    duration: Option<umpire_runtime::prost_types::Duration>,
}

#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire)]
#[mirror(response_checks = Raw)]
struct Normalized {
    #[rule(accepts)]
    duration: Duration,
}

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire, supplemental)]
struct Raw {
    duration: RawDuration,
}

#[test]
fn raw_checks_run_for_responses_after_request_normalization()
-> Result<(), Box<dyn std::error::Error>> {
    let wire = Wire {
        duration: Some(umpire_runtime::prost_types::Duration { seconds: i64::MAX, nanos: 0 }),
    };
    let mut findings = Vec::new();
    wire.inspect(&foundation::dc::Limits, false, &mut findings)?;
    assert_eq!(findings, []);
    wire.inspect(&foundation::dc::Limits, true, &mut findings)?;
    assert_eq!(findings.len(), 1);
    assert_eq!(findings[0].constraint, "duration.valid");
    assert!(findings[0].expected.is_none());
    let valid =
        Wire { duration: Some(umpire_runtime::prost_types::Duration { seconds: 1, nanos: 0 }) };
    findings.clear();
    valid.inspect(&foundation::dc::Limits, true, &mut findings)?;
    assert_eq!(findings, []);
    Ok(())
}

type CatalogContext = foundation::dc::Limits;
