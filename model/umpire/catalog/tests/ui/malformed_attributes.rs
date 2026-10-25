use umpire_catalog::Catalog;

#[derive(Catalog)]
#[mirror(response = "test.Wire", supplemental, supplemental)]
struct DuplicateFlag {}

#[derive(Catalog)]
#[mirror(response = "test.Wire", context = First, context = Second, api = first, api = second)]
struct DuplicateOptions {}

#[derive(Catalog)]
#[mirror(response = "test.Wire", context = "crate::Context")]
struct QuotedPath {}

fn main() {}
