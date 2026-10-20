#[derive(Clone, Default)]
struct Message {
    bytes: Vec<u8>,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { bytes: Random + Decided; }
}
fn main() {}
