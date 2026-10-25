#[derive(Clone, Default)]
struct Message {
    id: String,
}
umpire_catalog::annotate! {
    pub catalog controls;
    tag Visibility { kinds: [Timestamp]; consequence: EventuallyConsistent(3); }
    message response message: Message = "test.Message" { id: Uuid + Visibility; }
}
fn main() {}
