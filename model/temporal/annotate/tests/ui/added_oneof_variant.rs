#[derive(Clone, Default)]
struct Message {
    choice: Option<Choice>,
}
#[derive(Clone)]
enum Choice {
    Known(String),
    Added(bool),
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" {
        choice: Bytes + Ignored("opaque") opaque variants { Choice::Known = "known", };
    }
}
fn main() {}
