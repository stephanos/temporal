use umpire_temporal::realization;

struct Binding;

#[realization(activity)]
impl Binding {
    fn status(value: ActivityExecutionStatus) {
        match value {
            ActivityExecutionStatus::Running => (),
            _ => (),
        }
    }
}

fn main() {}
