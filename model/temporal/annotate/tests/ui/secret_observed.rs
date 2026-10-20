#[derive(Clone, Default)]
struct Message {
    secret: String,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" { secret: Secret + Observed; }
}
fn main() {}
