use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::temporal::api::{
    activity::v1::ActivityExecutionInfo,
    enums::v1::{ActivityExecutionStatus as Status, PendingActivityState as Run},
};

use super::activity::Phase;

impl TryFrom<ActivityExecutionInfo> for Phase {
    type Error = Refusal;

    fn try_from(info: ActivityExecutionInfo) -> Result<Self, Self::Error> {
        let status = Status::try_from(info.status)
            .map_err(|_| Refusal::unsupported("unknown activity execution status"))?;
        let run_state = Run::try_from(info.run_state)
            .map_err(|_| Refusal::unsupported("unknown activity run state"))?;
        let terminal = match status {
            Status::Unspecified => return Err(Refusal::malformed("activity status is missing")),
            Status::Running => {
                return match run_state {
                    Run::Scheduled => Ok(Self::Scheduled),
                    Run::Started => Ok(Self::Started),
                    Run::Unspecified => {
                        Err(Refusal::malformed("running activity run state is missing"))
                    }
                    Run::CancelRequested | Run::Paused | Run::PauseRequested => {
                        Err(Refusal::unsupported("unsupported running activity state"))
                    }
                };
            }
            Status::Completed => Self::Completed,
            Status::Failed => Self::Failed,
            Status::Canceled => Self::Canceled,
            Status::TimedOut => Self::TimedOut,
            Status::Terminated | Status::Paused => {
                return Err(Refusal::unsupported("unsupported activity execution status"));
            }
        };
        match run_state {
            Run::Unspecified => Ok(terminal),
            Run::Scheduled
            | Run::Started
            | Run::CancelRequested
            | Run::Paused
            | Run::PauseRequested => {
                Err(Refusal::malformed("terminal activity must have unspecified run state"))
            }
        }
    }
}
