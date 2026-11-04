use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::temporal::api::{
    activity::v1::ActivityExecutionInfo,
    enums::v1::{ActivityExecutionStatus as Status, PendingActivityState as Run},
};

use super::activity::Phase;
use crate::activity_protocol::view::{RunState, Status as Visible};

/// The observable status and run state of actual info, refusing states outside the domain.
pub(super) fn lifecycle(
    info: &ActivityExecutionInfo,
) -> Result<(Visible, Option<RunState>), Refusal> {
    let status = Status::try_from(info.status)
        .map_err(|_| Refusal::unsupported("unknown activity execution status"))?;
    let run_state = Run::try_from(info.run_state)
        .map_err(|_| Refusal::unsupported("unknown activity run state"))?;
    let terminal = match status {
        Status::Unspecified => return Err(Refusal::malformed("activity status is missing")),
        Status::Running => {
            let running = match run_state {
                Run::Scheduled => RunState::Scheduled,
                Run::Started => RunState::Started,
                Run::CancelRequested => RunState::CancelRequested,
                Run::Unspecified => {
                    return Err(Refusal::malformed("running activity run state is missing"));
                }
                Run::Paused | Run::PauseRequested => {
                    return Err(Refusal::unsupported("unsupported running activity state"));
                }
            };
            return Ok((Visible::Running, Some(running)));
        }
        Status::Completed => Visible::Completed,
        Status::Failed => Visible::Failed,
        Status::Canceled => Visible::Canceled,
        Status::TimedOut => Visible::TimedOut,
        Status::Terminated | Status::Paused => {
            return Err(Refusal::unsupported("unsupported activity execution status"));
        }
    };
    match run_state {
        Run::Unspecified => Ok((terminal, None)),
        Run::Scheduled
        | Run::Started
        | Run::CancelRequested
        | Run::Paused
        | Run::PauseRequested => {
            Err(Refusal::malformed("terminal activity must have unspecified run state"))
        }
    }
}

impl TryFrom<ActivityExecutionInfo> for Phase {
    type Error = Refusal;

    fn try_from(info: ActivityExecutionInfo) -> Result<Self, Self::Error> {
        Ok(match lifecycle(&info)? {
            (_, Some(RunState::Scheduled)) => Self::Scheduled,
            (_, Some(RunState::Started)) => Self::Started,
            (_, Some(RunState::CancelRequested)) => {
                return Err(Refusal::unsupported("unsupported running activity state"));
            }
            (Visible::Completed, None) => Self::Completed,
            (Visible::Failed, None) => Self::Failed,
            (Visible::Canceled, None) => Self::Canceled,
            (Visible::TimedOut, None) => Self::TimedOut,
            (Visible::Running, None) => {
                return Err(Refusal::malformed("running activity run state is missing"));
            }
        })
    }
}
