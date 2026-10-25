#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};

#[derive(Clone, Default)]
struct Wire { value: String }
#[derive(Clone, Default)]
struct OtherWire { value: String }
#[derive(Clone, Default)]
struct ThirdWire { value: String }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire, response_checks = Wrong)]
struct Mirror { value: Text }

#[derive(Catalog)]
#[mirror(response = "test.OtherWire", type = OtherWire, supplemental)]
struct Wrong { value: Text }

#[derive(Catalog)]
#[mirror(response = "test.ThirdWire", type = ThirdWire, response_checks = Missing)]
struct Third { value: Text }

fn main() {}

type CatalogContext = foundation::dc::Limits;
