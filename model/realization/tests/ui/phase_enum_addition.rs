extern crate self as umpire_temporal_api;

pub mod temporal {
    pub mod api {
        pub mod activity {
            pub mod v1 {
                pub struct ActivityExecutionInfo {
                    pub status: i32,
                    pub run_state: i32,
                }
            }
        }
        pub mod enums {
            pub mod v1 {
                pub enum ActivityExecutionStatus {
                    Unspecified,
                    Running,
                    Completed,
                    Failed,
                    Canceled,
                    Terminated,
                    TimedOut,
                    Paused,
                    AddedUpstream,
                }
                impl TryFrom<i32> for ActivityExecutionStatus {
                    type Error = ();
                    fn try_from(_: i32) -> Result<Self, Self::Error> {
                        Ok(Self::Unspecified)
                    }
                }
                pub enum PendingActivityState {
                    Unspecified,
                    Scheduled,
                    Started,
                    CancelRequested,
                    Paused,
                    PauseRequested,
                    AddedUpstream,
                }
                impl TryFrom<i32> for PendingActivityState {
                    type Error = ();
                    fn try_from(_: i32) -> Result<Self, Self::Error> {
                        Ok(Self::Unspecified)
                    }
                }
            }
        }
    }
}

mod activity {
    pub use umpire_temporal::activity::Phase;
}

mod phase {
    include!("../../../feature/activity/phase.rs");
}

fn main() {}
