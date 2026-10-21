use umpire_temporal_api::temporal::api::{
    activity::v1::ActivityExecutionInfo,
    enums::v1::{ActivityExecutionStatus as Status, PendingActivityState as Run},
};
use umpire_wasm::protocol::Refusal;

use super::activity::Phase;

impl TryFrom<ActivityExecutionInfo> for Phase {
    type Error = Refusal;

    fn try_from(info: ActivityExecutionInfo) -> Result<Self, Self::Error> {
        let status = Status::try_from(info.status)
            .map_err(|_| Refusal::unsupported("unknown activity execution status"))?;
        let run_state = Run::try_from(info.run_state)
            .map_err(|_| Refusal::unsupported("unknown activity run state"))?;
        match (status, run_state) {
            (Status::Unspecified, _) => Err(Refusal::malformed("activity status is missing")),
            (Status::Running, Run::Scheduled) => Ok(Self::Scheduled),
            (Status::Running, Run::Started) => Ok(Self::Started),
            (Status::Running, Run::Unspecified) => {
                Err(Refusal::malformed("running activity run state is missing"))
            }
            (Status::Running, _) => Err(Refusal::unsupported("unsupported running activity state")),
            (Status::Completed, Run::Unspecified) => Ok(Self::Completed),
            (Status::Failed, Run::Unspecified) => Ok(Self::Failed),
            (Status::Canceled, Run::Unspecified) => Ok(Self::Canceled),
            (Status::TimedOut, Run::Unspecified) => Ok(Self::TimedOut),
            (Status::Completed | Status::Failed | Status::Canceled | Status::TimedOut, _) => {
                Err(Refusal::malformed("terminal activity must have unspecified run state"))
            }
            _ => Err(Refusal::unsupported("unsupported activity execution status")),
        }
    }
}
