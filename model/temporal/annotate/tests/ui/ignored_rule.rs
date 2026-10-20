#[derive(Clone, Default)]
struct Message {
    opaque: bool,
}
umpire_temporal_annotate::annotate! {
    pub catalog controls;
    message response message: Message = "test.Message" {
        opaque: Bytes + Ignored("opaque boolean") opaque;
        rules { required(opaque); }
    }
}
fn main() {}
