#[derive(Clone, Copy)]
enum Status {
    Ready,
    Added,
}
impl TryFrom<i32> for Status {
    type Error = i32;
    fn try_from(value: i32) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(Self::Ready),
            1 => Ok(Self::Added),
            other => Err(other),
        }
    }
}
umpire_catalog::annotate! {
    pub catalog controls;
    enum status: Status = "test.Status" { Status::Ready = "READY", }
}
fn main() {}
