#[derive(Clone, Default)]
struct Request {
    value: i32,
}
umpire_temporal_annotate::annotate! {
    catalog controls;
    message response request: Request = "test.Request" {
        value: Uuid + Ignored("outside");
    }
}
fn main() {}
