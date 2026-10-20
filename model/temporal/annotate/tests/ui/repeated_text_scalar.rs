#[derive(Clone, Default)]
struct Message {
    value: String,
}
umpire_temporal_annotate::annotate! {
    catalog controls;
    message response message: Message = "test.Message" {
        value: Text[min_items = 1];
    }
}
fn main() {}
