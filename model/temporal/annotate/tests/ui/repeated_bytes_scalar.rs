#[derive(Clone, Default)]
struct Message {
    value: Vec<u8>,
}
umpire_temporal_annotate::annotate! {
    catalog controls;
    message response message: Message = "test.Message" {
        value: Bytes[unique = true];
    }
}
fn main() {}
