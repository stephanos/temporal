#[derive(Debug, Default)]
pub(crate) struct FailureReasons {
    reasons: Vec<String>,
}

impl FailureReasons {
    pub(crate) fn push(&mut self, reason: impl Into<String>) {
        let reason = reason.into();
        if !reason.is_empty() && !self.reasons.contains(&reason) {
            self.reasons.push(reason);
        }
    }

    #[must_use]
    pub(crate) fn message(&self) -> Option<String> {
        (!self.reasons.is_empty()).then(|| self.reasons.join("; "))
    }

    /// # Errors
    /// Returns the original failed observation after retaining its reason.
    pub(crate) fn observe<T>(&mut self, result: Result<T, String>) -> Result<T, String> {
        if let Err(error) = &result {
            self.push(error.clone());
        }
        result
    }

    /// # Errors
    /// Joins publication errors with the collection failures shown to the writer.
    pub(crate) fn publish<T>(
        &mut self,
        writer: impl FnOnce(Option<&str>) -> Result<T, String>,
    ) -> Result<T, String> {
        let failure = self.message();
        match writer(failure.as_deref()) {
            Ok(value) => Ok(value),
            Err(error) => {
                self.push(error);
                Err(self.reasons.join("; "))
            }
        }
    }
}

#[cfg(test)]
#[path = "collection_tests.rs"]
mod tests;
