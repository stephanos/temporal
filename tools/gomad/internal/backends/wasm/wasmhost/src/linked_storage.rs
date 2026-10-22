use crate::{engine, namespace::Namespace, protocol, storage_transport::StorageDescriptor};
use serde::{Deserialize, Serialize};
use wasmtime::{Error, Result};

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Input {
    descriptor: StorageDescriptor,
    #[serde(default)]
    image: Option<Vec<u8>>,
    #[serde(default)]
    context: Option<Vec<u8>>,
}

impl Input {
    pub(crate) fn retain_native_crash_reference(&mut self) -> Result<()> {
        if self.descriptor.crash_limits.is_none() {
            return Err(Error::msg(
                "native crash reference requires an enumerated durable owner",
            ));
        }
        self.descriptor.reference_precrash = true;
        Ok(())
    }

    pub(crate) fn set_simulation(&mut self, controller: crate::linked_simulation::Shared) {
        self.descriptor.set_simulation(controller);
    }

    /// # Errors
    /// Binds a lifecycle-owned recovery image using the existing storage declaration.
    pub(crate) fn with_recovery_image(&self, image: Vec<u8>) -> Result<Self> {
        use sha2::{Digest, Sha256};
        let mut input = self.clone();
        input.descriptor.image = Some(crate::storage_transport::ChunkedDescriptor {
            bytes: image.len() as u64,
            chunks: (image.len() as u64).div_ceil(64 * 1024),
            sha256: format!("{:x}", Sha256::digest(&image)),
        });
        input.image = Some(image);
        input.descriptor.validate()?;
        Ok(input)
    }

    pub(crate) fn descriptor(&self) -> &StorageDescriptor {
        &self.descriptor
    }
    pub(crate) fn attach(
        &mut self,
        namespace: &mut Namespace,
        transcript_limit: u64,
    ) -> Result<()> {
        self.descriptor.validate()?;
        let maximum = namespace
            .filesystem_bytes()
            .checked_add(crate::storage_transport::IMAGE_METADATA_BYTES)
            .ok_or_else(|| Error::msg("storage image bound overflow"))?;
        self.descriptor
            .authenticate_image_bytes(self.image.as_deref(), maximum)?;
        self.descriptor
            .attach_context_bytes(self.context.as_deref(), maximum.saturating_mul(18))?;
        self.descriptor
            .attach(namespace, self.image.as_deref(), transcript_limit)
            .map_err(Error::new)
    }

    pub(crate) fn finish(&self, state: &mut engine::State, outcome: &mut protocol::Outcome) {
        engine::finish_storage(state, Some(&self.descriptor), outcome);
    }
}
