use umpire_temporal::{activity, realization};
use umpire_temporal_api::temporal::api::enums::v1::ActivityExecutionStatus as Status;

struct Binding;

#[realization(activity)]
impl Binding {
    fn status(value: Status) {
        match value {
            Status::Running => (),
            _ => (),
        }
    }
}

fn main() {
    let _ = activity::World::new(3, None, None, None, None, false);
    let _ = Status::Running;
}
