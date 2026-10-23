use crate::protocol;

pub(crate) fn refused(code: protocol::RefusalCode, message: String) -> protocol::response::Outcome {
    protocol::response::Outcome::Refusal(protocol::Refusal::new(code, message))
}

pub(crate) fn checker_refusal(error: &umpire_checker::Refusal) -> protocol::response::Outcome {
    let code = match error {
        umpire_checker::Refusal::Step { .. } => protocol::RefusalCode::MalformedRequest,
        umpire_checker::Refusal::ProgressBound { .. }
        | umpire_checker::Refusal::ProgressRegistrationMissing { .. }
        | umpire_checker::Refusal::ProgressRegistrationAmbiguous { .. }
        | umpire_checker::Refusal::ProgressCatalogChanged => {
            protocol::RefusalCode::MalformedRequest
        }
        umpire_checker::Refusal::Budget { .. }
        | umpire_checker::Refusal::CatalogLimit { .. }
        | umpire_checker::Refusal::CatalogOverflow { .. } => protocol::RefusalCode::Budget,
        umpire_checker::Refusal::State { .. } => protocol::RefusalCode::State,
        umpire_checker::Refusal::ScopeAction { .. } | umpire_checker::Refusal::EmptyAssumption => {
            protocol::RefusalCode::ScopeAction
        }
    };
    refused(code, error.to_string())
}
