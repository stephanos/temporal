#![forbid(unsafe_code)]
use umpire_catalog::{Catalog, kinds::{Map, Message, Text}};
type CatalogContext = ();
#[derive(Clone, PartialEq, prost::Message)]
struct Actual { #[prost(string, tag="1")] name: String }
#[derive(Clone, PartialEq, prost::Message)]
struct Wrong { #[prost(string, tag="1")] name: String }
#[derive(Clone, PartialEq, prost::Message)]
struct Container { #[prost(btree_map="string, message", tag="1")] fields: std::collections::BTreeMap<String, Actual> }
#[derive(Catalog)]
#[mirror(response="example.Actual", type=Actual)]
struct ActualClaim { name: Text }
#[derive(Catalog)]
#[mirror(response="example.Container", type=Container)]
struct ContainerClaim { fields: Map<Text, Message<Wrong>> }
fn main() {}
