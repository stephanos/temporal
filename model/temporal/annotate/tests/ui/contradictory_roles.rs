#[derive(Clone, Default)]
struct Message {
    id: String,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { id: Uuid + Decided + Observed; }
}
fn main() {}
