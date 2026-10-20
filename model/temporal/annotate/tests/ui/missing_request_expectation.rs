#[derive(Clone, Default)]
struct Message {
    id: String,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message request message: Message = "test.Message" { id: Uuid; }
}
fn main() {}
