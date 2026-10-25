#[derive(Clone, Default)]
struct Wire { bytes: Vec<u8> }

umpire_catalog::annotate! {
    pub catalog controls;
    message response wire: Wire = "test.Wire" {
        bytes: Random + Decided;
    }
}
fn main() {}
