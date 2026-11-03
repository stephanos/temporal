use umpire_temporal_api::temporal::api::enums::v1::ActivityExecutionStatus as Status;
umpire_temporal::registry! {
    model { specimen }
    actors {}
    body {
        #[registry(inputs = ())]
        impl Binding {
            fn status(value: Result<Status, i32>) {
                match value {
                    Ok(status) => match status { Status::Running => (), _ => () },
                    Err(_) => (),
                }
            }
        }
    }
}
fn main() {}
