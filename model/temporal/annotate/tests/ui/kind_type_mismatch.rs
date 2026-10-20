#[derive(Clone, Default)]
struct Message {
    count: String,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { count: Counter; }
}
fn main() {}
