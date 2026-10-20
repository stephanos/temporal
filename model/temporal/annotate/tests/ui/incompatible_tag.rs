#[derive(Clone, Default)]
struct Message {
    count: i32,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { count: Counter + IdempotencyKey; }
}
fn main() {}
