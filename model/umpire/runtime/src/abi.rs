use std::collections::BTreeMap;

use prost::Message;

use crate::{Dispatch, protocol, refusal::refused};

const MAX_BUFFER: usize = 1 << 20;
const MAX_LIVE: usize = 16 << 20;

/// An independently owned dispatcher and its bounded core-wasm buffers.
pub struct Guest<D: Dispatch> {
    dispatcher: D,
    buffers: BTreeMap<u32, Vec<u8>>,
    live: usize,
}

impl<D: Dispatch> Guest<D> {
    /// Creates a guest with no live allocations.
    pub fn new(dispatcher: D) -> Self {
        Self { dispatcher, buffers: BTreeMap::new(), live: 0 }
    }

    /// Allocates a zeroed request buffer, or returns zero on exhaustion.
    pub fn allocate(&mut self, len: usize) -> u32 {
        if len == 0 || len > MAX_BUFFER || len > MAX_LIVE - self.live {
            return 0;
        }
        let mut buffer = Vec::new();
        if buffer.try_reserve_exact(len).is_err() {
            return 0;
        }
        buffer.resize(len, 0);
        let Ok(ptr) = u32::try_from(buffer.as_ptr() as usize) else { return 0 };
        if ptr == 0 {
            return 0;
        }
        self.buffers.insert(ptr, buffer);
        self.live += len;
        ptr
    }

    /// Releases an exact live allocation; invalid ranges panic.
    pub fn release(&mut self, ptr: u32, len: u32) {
        if (ptr, len) == (0, 0) {
            return;
        }
        assert!(
            self.buffers.get(&ptr).is_some_and(|buffer| buffer.len() == len as usize),
            "umpire_buffer_release requires an exact live allocation"
        );
        self.buffers.remove(&ptr);
        self.live -= len as usize;
    }

    /// Dispatches an owned request and packs a separately owned response's length and pointer.
    pub fn call(&mut self, method: u32, ptr: u32, len: u32) -> u64 {
        let (response, transaction) = if (ptr, len) == (0, 0) {
            self.dispatcher.prepare(method, &[])
        } else if let Some(buffer) =
            self.buffers.get(&ptr).filter(|buffer| buffer.len() == len as usize)
        {
            self.dispatcher.prepare(method, buffer)
        } else {
            (
                protocol::Response {
                    outcome: Some(refused(
                        protocol::RefusalCode::Allocation,
                        "umpire_call requires an exact live allocation".into(),
                    )),
                },
                None,
            )
        };
        let size = response.encoded_len();
        let ptr = self.allocate(size);
        if ptr == 0 {
            return 0;
        }
        let Some(buffer) = self.buffers.get_mut(&ptr) else {
            unreachable!("response allocation is live")
        };
        if response.encode(&mut buffer.as_mut_slice()).is_err() {
            self.release(
                ptr,
                u32::try_from(size).unwrap_or_else(|_| unreachable!("bounded buffer")),
            );
            return 0;
        }
        if let Some(transaction) = transaction {
            self.dispatcher.commit(transaction);
        }
        let len = u32::try_from(size).unwrap_or_else(|_| unreachable!("bounded buffer"));
        (u64::from(len) << 32) | u64::from(ptr)
    }

    /// Publishes an encoded response before committing, for native transports.
    ///
    /// Allocation, encoding, publication failure, or a panic before commit leaves dispatcher
    /// state unchanged. The publisher must return true only after accepting the whole response.
    pub fn dispatch_to(
        &mut self,
        method: u32,
        request: &[u8],
        publish: impl FnOnce(&[u8]) -> bool,
    ) -> bool {
        let (response, transaction) = self.dispatcher.prepare(method, request);
        let Some(buffer) = encode_response(&response) else { return false };
        if !publish(&buffer) {
            return false;
        }
        if let Some(transaction) = transaction {
            self.dispatcher.commit(transaction);
        }
        true
    }
}

pub(crate) fn encode_response(response: &protocol::Response) -> Option<Vec<u8>> {
    let size = response.encoded_len();
    if size == 0 || size > MAX_BUFFER {
        return None;
    }
    let mut buffer = Vec::new();
    buffer.try_reserve_exact(size).ok()?;
    response.encode(&mut buffer).ok()?;
    Some(buffer)
}

/// Installs the three core-wasm application exports for one dispatcher type.
#[macro_export]
macro_rules! export_guest {
    ($dispatcher:ty, $constructor:expr) => {
        ::std::thread_local! {
            static GUEST: ::std::cell::RefCell<$crate::Guest<$dispatcher>> =
                ::std::cell::RefCell::new($crate::Guest::new($constructor));
        }

        /// Allocates a zeroed, bounded request buffer, or returns zero on exhaustion.
        #[allow(unsafe_code)]
        #[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
        pub extern "C" fn umpire_buffer_alloc(len: u32) -> u32 {
            GUEST.with_borrow_mut(|guest| guest.allocate(len as usize))
        }

        /// Releases an exact live allocation; invalid ranges trap.
        #[allow(unsafe_code)]
        #[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
        pub extern "C" fn umpire_buffer_release(ptr: u32, len: u32) {
            GUEST.with_borrow_mut(|guest| guest.release(ptr, len));
        }

        /// Dispatches a request and packs a separately owned response's length and pointer.
        #[allow(unsafe_code)]
        #[cfg_attr(target_arch = "wasm32", unsafe(no_mangle))]
        pub extern "C" fn umpire_call(method: u32, ptr: u32, len: u32) -> u64 {
            GUEST.with_borrow_mut(|guest| guest.call(method, ptr, len))
        }
    };
}
