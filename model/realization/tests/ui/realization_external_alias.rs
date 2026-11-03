use umpire_temporal::{activity, realization};

#[path = "support/enum_aliases.rs"]
mod aliases;
use aliases::*;

struct Binding;

fn sample() -> umpire_temporal_api::temporal::api::enums::v1::ActivityExecutionStatus {
    umpire_temporal_api::temporal::api::enums::v1::ActivityExecutionStatus::Running
}

#[realization(activity)]
impl Binding {
    fn status() {
        match sample() {
            Active => (),
            _ => (),
        }
    }
}

fn main() {
    let _ = activity::World::new(3, None, None, None, None, false);
    let _ = Active;
    let _ = sample();
}
