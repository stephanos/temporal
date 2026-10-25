//! Message-valued maps retain typed carriers and nested validation.
#![forbid(unsafe_code)]
use prost::Message as _;
use std::collections::BTreeMap;
use umpire_catalog::{
    Catalog, Inspect,
    kinds::{Map, Message, Text},
};
type CatalogContext = ();

#[derive(Clone, PartialEq, prost::Message)]
struct Child {
    #[prost(string, tag = "1")]
    name: String,
}
#[derive(Clone, PartialEq, prost::Message)]
struct Parent {
    #[prost(btree_map = "string, message", tag = "1")]
    children: BTreeMap<String, Child>,
}

#[derive(Catalog)]
#[mirror(response="example.Child", type=Child)]
#[allow(dead_code, reason = "Mirror fields declare the catalog")]
struct ChildClaim {
    #[rule(max_len = 3)]
    name: Text,
}
#[derive(Catalog)]
#[mirror(response="example.Parent", type=Parent)]
#[allow(dead_code, reason = "Mirror fields declare the catalog")]
struct ParentClaim {
    children: Map<Text, Message<Child>>,
}

#[test]
fn message_maps_round_trip_and_validate_actual_nested_values()
-> Result<(), Box<dyn std::error::Error>> {
    let parent =
        Parent { children: BTreeMap::from([("key".into(), Child { name: "long".into() })]) };
    let entries = umpire_catalog::views::message_entries(&parent.children);
    let decoded = umpire_catalog::views::decode_message_entries::<Child>(
        entries.iter().map(prost::Message::encode_to_vec).collect(),
    )?;
    assert_eq!(decoded, parent.children);
    let mut findings = Vec::new();
    parent.inspect(&(), true, &mut findings)?;
    assert_eq!(findings.len(), 1);
    assert_eq!(findings[0].path, "example.Child.name");
    assert_eq!(findings[0].constraint, "string.max_len");
    let invalid =
        umpire_catalog::MetadataEntry { key: "key".into(), value: vec![0xff] }.encode_to_vec();
    assert_eq!(
        umpire_catalog::views::decode_message_entries::<Child>(vec![invalid]),
        Err(umpire_catalog::GenerationError::InvalidEncoding)
    );
    Ok(())
}
