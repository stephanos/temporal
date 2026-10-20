use super::canonical;
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Identity {
    pub target: String,
    pub toolchain: String,
    pub platform: String,
    pub implementation: String,
    pub configuration: String,
}

impl Identity {
    /// # Errors
    /// Refuses missing or malformed identity components.
    pub fn validate(&self) -> Result<(), String> {
        for digest in [
            &self.target,
            &self.toolchain,
            &self.platform,
            &self.implementation,
            &self.configuration,
        ] {
            if !canonical::valid_digest(digest) {
                return Err("identity requires lowercase SHA-256 digests".into());
            }
        }
        Ok(())
    }

    /// # Errors
    /// Refuses invalid identity components or a serialization failure.
    pub fn digest(&self) -> Result<String, String> {
        self.validate()?;
        Ok(canonical::digest(&canonical::encode(self)?))
    }
}
