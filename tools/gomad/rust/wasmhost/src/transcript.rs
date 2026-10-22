#[cfg(all(test, not(feature = "conformance")))]
use wasmtime::{Error, Result};

pub const HEADER: &[u8; 8] = b"GMWT\x01\0\0\0";
pub const RECORD_HEADER: u64 = 16;
#[cfg(not(feature = "conformance"))]
pub const ENTROPY_BYTES: u64 = RECORD_HEADER + 12;
pub const CPU_CLOCK: &str = "CPU clocks outside stock WASI profile";
pub const CPU_POLL: &str = "CPU poll clock outside stock WASI profile";
pub const CLOCK_RUN_BYTES: u64 = 44;
pub const MAX_CLOCK_COUNT: u64 = 1 << 28;

#[derive(Clone, Copy)]
struct ClockRun {
    id: u64,
    clock_id: u32,
    precision: u64,
    now: u64,
    count: u64,
}
#[derive(Default)]
pub struct Encoder {
    clock: Option<ClockRun>,
}
impl Encoder {
    pub fn folds(&self, id: u64, clock_id: u32, precision: u64, now: u64, step: u64) -> bool {
        self.clock.is_some_and(|run| {
            run.clock_id == clock_id
                && run.precision == precision
                && run.count < MAX_CLOCK_COUNT
                && run.id.checked_add(run.count) == Some(id)
                && run
                    .count
                    .checked_mul(step)
                    .and_then(|elapsed| run.now.checked_add(elapsed))
                    == Some(now)
        })
    }
    /// Records one successful read. Callers close a run that `folds` refuses
    /// before reading, so an open run only ever grows with its own key.
    pub fn read(&mut self, id: u64, clock_id: u32, precision: u64, now: u64) {
        if let Some(run) = self.clock.as_mut() {
            assert!(
                run.clock_id == clock_id
                    && run.precision == precision
                    && run.id.checked_add(run.count) == Some(id),
                "clock read does not extend the open run"
            );
            run.count += 1;
        } else {
            self.clock = Some(ClockRun {
                id,
                clock_id,
                precision,
                now,
                count: 1,
            });
        }
    }
    #[cfg(any(test, feature = "conformance"))]
    pub fn is_open(&self) -> bool {
        self.clock.is_some()
    }
    pub fn pending(&self) -> Option<[u8; 44]> {
        self.clock.map(|run| {
            let mut bytes = [0; 44];
            bytes[0] = 33;
            bytes[4..12].copy_from_slice(&run.id.to_le_bytes());
            bytes[12..16].copy_from_slice(&28_u32.to_le_bytes());
            bytes[16..20].copy_from_slice(&run.clock_id.to_le_bytes());
            bytes[20..28].copy_from_slice(&run.precision.to_le_bytes());
            bytes[28..36].copy_from_slice(&run.now.to_le_bytes());
            bytes[36..44].copy_from_slice(&run.count.to_le_bytes());
            bytes
        })
    }
    pub fn close(&mut self) -> Option<[u8; 44]> {
        let bytes = self.pending();
        self.clock = None;
        bytes
    }
}

#[derive(Clone, Copy)]
pub struct Subscription {
    pub userdata: u64,
    pub kind: u8,
    pub clock_id: u32,
    pub timeout: u64,
    pub precision: u64,
    pub flags: u16,
    pub fd: u32,
}
#[derive(Clone, Copy)]
pub struct Event {
    pub userdata: u64,
    pub errno: u16,
    pub kind: u8,
    pub nbytes: u64,
    pub flags: u16,
}
#[expect(
    clippy::cast_possible_truncation,
    reason = "Transcript lengths retain the protocol's low 32 bits after caller admission"
)]
pub fn record(
    tag: u8,
    id: u64,
    mut input: Vec<u8>,
    output: &[u8],
    errno: u16,
    unsupported: &str,
) -> Vec<u8> {
    if !unsupported.is_empty() {
        input.extend_from_slice(&(unsupported.len() as u32).to_le_bytes());
        input.extend_from_slice(unsupported.as_bytes());
    } else if errno == 0 {
        input.extend_from_slice(output);
    }
    let mut bytes = Vec::with_capacity(16 + input.len());
    bytes.extend_from_slice(&[tag, u8::from(!unsupported.is_empty())]);
    bytes.extend_from_slice(&errno.to_le_bytes());
    bytes.extend_from_slice(&id.to_le_bytes());
    bytes.extend_from_slice(&(input.len() as u32).to_le_bytes());
    bytes.extend_from_slice(&input);
    bytes
}
#[cfg(not(feature = "conformance"))]
pub fn clock_reservation() -> u64 {
    RECORD_HEADER + 12 + 4 + CPU_CLOCK.len() as u64
}
#[cfg(not(feature = "conformance"))]
pub fn clock_unsupported(id: u64, clock_id: u32, precision: u64) -> Vec<u8> {
    let mut input = Vec::with_capacity(12);
    input.extend_from_slice(&clock_id.to_le_bytes());
    input.extend_from_slice(&precision.to_le_bytes());
    record(1, id, input, &[], 0, CPU_CLOCK)
}
pub fn yield_record(id: u64) -> Vec<u8> {
    record(2, id, Vec::new(), &[], 0, "")
}
#[cfg(not(feature = "conformance"))]
pub fn strings_record(id: u64, operation: crate::abi::Operation) -> Vec<u8> {
    use crate::abi::Operation;
    let tag = match operation {
        Operation::ArgsSizesGet => 3,
        Operation::ArgsGet => 4,
        Operation::EnvironSizesGet => 5,
        Operation::EnvironGet => 6,
        _ => unreachable!(),
    };
    record(tag, id, Vec::new(), &[], 0, "")
}
#[cfg(not(feature = "conformance"))]
pub fn entropy_record(id: u64, length: u32, offset: u64) -> Vec<u8> {
    let mut payload = length.to_le_bytes().to_vec();
    payload.extend(offset.to_le_bytes());
    record(7, id, payload, &[], 0, "")
}
#[expect(
    clippy::cast_possible_truncation,
    reason = "Subscription counts are admitted against the u32 WASI request count"
)]
pub fn poll_input(subscriptions: &[Subscription]) -> Vec<u8> {
    let mut input = Vec::with_capacity(4 + subscriptions.len() * 35);
    input.extend_from_slice(&(subscriptions.len() as u32).to_le_bytes());
    for sub in subscriptions {
        input.extend_from_slice(&sub.userdata.to_le_bytes());
        input.push(sub.kind);
        if sub.kind == 0 {
            input.extend_from_slice(&sub.clock_id.to_le_bytes());
            input.extend_from_slice(&sub.timeout.to_le_bytes());
            input.extend_from_slice(&sub.precision.to_le_bytes());
            input.extend_from_slice(&sub.flags.to_le_bytes());
        } else {
            input.extend_from_slice(&sub.fd.to_le_bytes());
        }
    }
    input
}
/// The length of `poll_input`, without building it.
fn poll_input_len(subscriptions: &[Subscription]) -> u64 {
    subscriptions.iter().fold(4, |total, sub| {
        total + 9 + if sub.kind == 0 { 22 } else { 4 }
    })
}
pub fn poll_reservation(subscriptions: &[Subscription]) -> u64 {
    RECORD_HEADER
        + poll_input_len(subscriptions)
        + (4 + subscriptions.len() as u64 * 23).max(4 + CPU_POLL.len() as u64)
}
#[expect(
    clippy::cast_possible_truncation,
    reason = "Event counts are bounded by the admitted u32 subscription count"
)]
pub fn poll_record(
    id: u64,
    subscriptions: &[Subscription],
    events: &[Event],
    errno: u16,
    unsupported: &str,
) -> Vec<u8> {
    let mut output = Vec::with_capacity(4 + events.len() * 23);
    output.extend_from_slice(&(events.len() as u32).to_le_bytes());
    for event in events {
        output.extend_from_slice(&event.userdata.to_le_bytes());
        output.extend_from_slice(&event.errno.to_le_bytes());
        output.push(event.kind);
        output.extend_from_slice(&event.nbytes.to_le_bytes());
        output.extend_from_slice(&event.flags.to_le_bytes());
    }
    record(
        32,
        id,
        poll_input(subscriptions),
        &output,
        errno,
        unsupported,
    )
}
#[cfg(all(test, not(feature = "conformance")))]
pub fn subscriptions(value: &serde_json::Value) -> Result<Vec<Subscription>> {
    value["subscriptions"]
        .as_array()
        .ok_or_else(|| Error::msg("WASI invalid: poll input"))?
        .iter()
        .map(|sub| {
            let number = |name: &str| {
                sub[name]
                    .as_u64()
                    .ok_or_else(|| Error::msg("WASI invalid: subscription"))
            };
            let kind = u8::try_from(number("type")?).map_err(Error::new)?;
            Ok(Subscription {
                userdata: number("userdata")?,
                kind,
                clock_id: if kind == 0 {
                    u32::try_from(number("clock_id")?).map_err(Error::new)?
                } else {
                    0
                },
                timeout: if kind == 0 { number("timeout")? } else { 0 },
                precision: if kind == 0 { number("precision")? } else { 0 },
                flags: if kind == 0 {
                    u16::try_from(number("flags")?).map_err(Error::new)?
                } else {
                    0
                },
                fd: if kind != 0 {
                    u32::try_from(number("fd")?).map_err(Error::new)?
                } else {
                    0
                },
            })
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn poll_input_length_matches_encoding() {
        let sub = |kind| Subscription {
            userdata: 1,
            kind,
            clock_id: 1,
            timeout: 2,
            precision: 3,
            flags: 1,
            fd: 4,
        };
        for subscriptions in [vec![], vec![sub(0)], vec![sub(1), sub(0), sub(2)]] {
            assert_eq!(
                poll_input_len(&subscriptions),
                poll_input(&subscriptions).len() as u64
            );
        }
    }
    #[test]
    fn clock_run_rejects_folding_count_and_id_overflow() {
        let mut encoder = Encoder::default();
        encoder.read(u64::MAX, 1, 0, 1);
        assert!(!encoder.folds(0, 1, 0, 1, 0));
        encoder.clock = Some(ClockRun {
            id: 1,
            clock_id: 1,
            precision: 0,
            now: 1,
            count: MAX_CLOCK_COUNT,
        });
        assert!(!encoder.folds(MAX_CLOCK_COUNT + 1, 1, 0, 1, 0));
    }
}
