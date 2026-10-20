#[derive(Clone, Default)]
pub struct Message {
    pub id: String,
    pub omitted: String,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { id: Uuid; }
}
fn main() {}
