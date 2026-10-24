//! Scalar oneof and text map projections retain their protobuf semantics.
#![allow(dead_code)]
use umpire_catalog::{Catalog, Inspect, kinds::*};

struct Limits;

#[derive(Clone, PartialEq, prost::Message)]
struct Wire {
    #[prost(oneof = "Source", tags = "1, 2")]
    source: Option<Source>,
    #[prost(btree_map = "string, string", tag = "3")]
    tags: std::collections::BTreeMap<String, String>,
}
#[derive(Clone, PartialEq, prost::Oneof)]
enum Source {
    #[prost(string, tag = "1")]
    Name(String),
    #[prost(bytes, tag = "2")]
    Token(Vec<u8>),
}
#[derive(Catalog)]
#[mirror(response = "fixture.Wire", type = Wire, context = Limits)]
#[rule(one_of(name, token))]
struct WireCatalog {
    #[mirror(oneof(Name = "name", Token = "token"))]
    #[rule(max_len = 3, rejects)]
    source: OneOf<Source>,
    tags: Map<Text, Text>,
}

#[test]
fn selected_empty_scalars_are_present_and_map_entries_are_text()
-> Result<(), Box<dyn std::error::Error>> {
    for source in [Source::Name(String::new()), Source::Token(Vec::new())] {
        let wire = Wire { source: Some(source), ..Default::default() };
        let mut findings = Vec::new();
        wire.inspect(&Limits, true, &mut findings)?;
        assert!(findings.is_empty(), "{findings:?}");
    }
    let mut findings = Vec::new();
    Wire::default().inspect(&Limits, true, &mut findings)?;
    assert_eq!(findings[0].constraint, "one_of");
    let catalog = WireCatalog::catalog(&Limits)?;
    for (source, constraint) in [
        (Source::Name("four".into()), "string.max_len"),
        (Source::Token(vec![0; 4]), "bytes.max_len"),
    ] {
        let wire = Wire { source: Some(source), ..Default::default() };
        let findings = umpire_catalog::validate(&wire, &catalog);
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].constraint, constraint);
    }
    let baseline = Wire { source: Some(Source::Name("ok".into())), ..Default::default() };
    for case in catalog.probe_cases() {
        let probe = catalog.probe(&baseline, &case, &mut umpire_catalog::Rng::new(1))?;
        assert!(matches!(
            (case.violation.path.as_str(), probe.message.source),
            ("fixture.Wire.name", Some(Source::Name(_)))
                | ("fixture.Wire.token", Some(Source::Token(_)))
        ));
    }
    let entry_catalog = WireCatalog::__tags_entry_catalog(&Limits)?;
    assert!(
        entry_catalog.fields().all(|field| matches!(field.kind, umpire_catalog::Kind::Text(_)))
    );
    let mut tags = std::collections::BTreeMap::from([("key".to_owned(), "value".to_owned())]);
    let projected = <_ as umpire_catalog::View<Map<Text, Text>>>::view(&tags);
    tags.clear();
    <_ as umpire_catalog::View<Map<Text, Text>>>::set(&mut tags, projected)?;
    assert_eq!(tags.get("key").map(String::as_str), Some("value"));
    let original = tags.clone();
    assert!(
        <_ as umpire_catalog::View<Map<Text, Text>>>::set(&mut tags, vec![vec![0xff]]).is_err()
    );
    assert_eq!(tags, original);
    Ok(())
}
