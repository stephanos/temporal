use crate::linked::Request;
use crate::model::RemoteCapacity;
use crate::storage_transport::{IMAGE_METADATA_BYTES, STATE_MAGIC, StorageDescriptor};
use wasmtime::{Error, Result};

#[derive(Clone)]
pub(crate) struct Bounds {
    complete: u64,
    partial: u64,
    image: u64,
}

fn budget_add(limit: u64, count: u64, size: u64) -> u64 {
    limit.saturating_add(count.saturating_mul(size))
}

impl Bounds {
    pub(crate) fn for_request(request: &Request) -> Option<Self> {
        let namespace = request.namespace.as_ref()?;
        let model = request.model.as_ref()?;
        let storage = request.storage.as_ref()?;
        Some(Self::new(
            namespace.limits.filesystem_bytes,
            model.configuration.limits.calls,
            namespace.limits.files,
            storage.descriptor(),
        ))
    }

    fn new(filesystem_bytes: u64, calls: u64, files: u64, descriptor: &StorageDescriptor) -> Self {
        let image = filesystem_bytes.saturating_add(IMAGE_METADATA_BYTES);
        let mut complete = budget_add(256, image, 18);
        if let Some(enumeration) = descriptor.crash_limits {
            complete = budget_add(complete, 1, budget_add(12, image, 18));
            for volume in descriptor.model.volumes() {
                complete = budget_add(complete, enumeration.bytes, 6);
                complete = budget_add(complete, enumeration.states.min(enumeration.work), 80);
                complete = budget_add(complete, descriptor.model.limits.pending_operations, 2);
                complete = budget_add(complete, volume.id.len() as u64, 12);
                complete = budget_add(complete, 1, 1024);
            }
        }
        let call_files = calls.saturating_mul(files);
        let call_files_squared = call_files.saturating_mul(files);
        let path_bytes = crate::volume::MAXIMUM_PATH_BYTES as u64;
        let mut partial = budget_add(complete, calls, 2 * (361 + 3 * 21));
        partial = budget_add(
            partial,
            call_files,
            2 * 2 * 6 * path_bytes + 2 * 2 * 21 + 4 * 195,
        );
        partial = budget_add(partial, call_files_squared, 4 * 6 * path_bytes);
        for volume in descriptor.model.volumes() {
            partial = budget_add(partial, volume.id.len() as u64, 3 * 6);
            partial = budget_add(partial, volume.mount.len() as u64, 6);
            partial = budget_add(partial, 1, 1024);
        }
        Self {
            complete,
            partial,
            image,
        }
    }

    fn limit(&self, stream: &str, previous: &[u8], next: &[u8]) -> Option<u64> {
        match stream {
            "storage_state" => {
                let mut header = [0; 16];
                let first = previous.len().min(header.len());
                header[..first].copy_from_slice(&previous[..first]);
                let second = next.len().min(header.len() - first);
                header[first..first + second].copy_from_slice(&next[..second]);
                Some(
                    if first + second == 16
                        && &header[..8] == STATE_MAGIC
                        && header[8..12] == 2_u32.to_le_bytes()
                        && header[12..16] == 6_u32.to_le_bytes()
                    {
                        self.partial
                    } else {
                        self.complete
                    },
                )
            }
            "storage_image" | "storage_precrash_image" => Some(self.image),
            _ => None,
        }
    }

    pub(crate) fn admit(&self, stream: &str, previous: &[u8], next: &[u8]) -> Result<()> {
        if let Some(limit) = self.limit(stream, previous, next)
            && (previous.len() as u64)
                .checked_add(next.len() as u64)
                .is_none_or(|total| total > limit)
        {
            return Err(Error::new(RemoteCapacity(
                "WASI capacity: helper evidence bytes",
            )));
        }
        Ok(())
    }
}

#[cfg(test)]
#[path = "storage_evidence_bounds_tests.rs"]
mod tests;
