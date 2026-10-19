use crate::protocol;

pub(crate) fn refused(code: protocol::RefusalCode, message: String) -> protocol::response::Outcome {
    protocol::response::Outcome::Refusal(protocol::Refusal { code: code as i32, message })
}

pub(crate) fn checker_refusal(error: &umpire_checker::Refusal) -> protocol::response::Outcome {
    let code = match error {
        umpire_checker::Refusal::Budget { .. } => protocol::RefusalCode::Budget,
        umpire_checker::Refusal::State { .. } => protocol::RefusalCode::State,
        umpire_checker::Refusal::ScopeAction { .. } | umpire_checker::Refusal::EmptyAssumption => {
            protocol::RefusalCode::ScopeAction
        }
    };
    refused(code, error.to_string())
}
