use super::{Completed, Running};
use std::sync::atomic::AtomicBool;
pub(crate) fn terminate_all(mut children: Vec<Running>) -> Result<Vec<Completed>, String> {
    for child in &mut children {
        child.cancel();
    }
    let mut completed = Vec::new();
    let mut errors = Vec::new();
    for child in children {
        match child.wait(&AtomicBool::new(false)) {
            Ok(record) if record.cleanup_complete => completed.push(record),
            Ok(record) => errors.push(format!(
                "process {} cleanup inconclusive: {:?}",
                record.pid, record.cleanup_error
            )),
            Err(error) => errors.push(error),
        }
    }
    if errors.is_empty() {
        Ok(completed)
    } else {
        Err(format!(
            "owned process cleanup incomplete: {}",
            errors.join("; ")
        ))
    }
}
