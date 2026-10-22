use umpire_catalog::{Catalog, kinds::{Text, Ignored}};
struct Limits;
#[derive(Clone)]
struct Wire {
    kettle_id: String,
    options: Option<Vec<u8>>,
    idempotency_key: String,
}
#[derive(Catalog)]
#[mirror(type = Wire, context = Limits, request = "umpire.reference.kettle.v1.FillRequest")]
struct FillRequest {
    kettle_id: Text,
    #[mirror(ignored = "Only field inventory is tested in this refusal fixture")]
    options: Ignored,
}
fn main() {}
