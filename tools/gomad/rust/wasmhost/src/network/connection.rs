use std::collections::VecDeque;

pub(super) struct Chunk {
    pub bytes: VecDeque<u8>,
    pub ready: u64,
}

#[derive(Default)]
#[expect(
    clippy::struct_excessive_bools,
    reason = "Read closure, write closure, close calls and stalls are independent connection state"
)]
pub(super) struct State {
    pub incoming: VecDeque<Chunk>,
    pub received: VecDeque<u8>,
    pub retained_bytes: u64,
    pub read_closed: bool,
    pub write_closed: bool,
    pub close_called: bool,
    pub read_deadline: u64,
    pub write_deadline: u64,
    pub stalled: bool,
    pub stall_start: u64,
    pub stall_due: u64,
    pub stall_epoch: u64,
    pub terminal: Option<String>,
}
