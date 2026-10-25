/// Stable `SplitMix64` stream; no operating-system randomness is consulted.
#[derive(Clone, Debug)]
pub struct SplitMix64(u64);

impl SplitMix64 {
    /// Starts a deterministic stream.
    pub const fn new(seed: u64) -> Self {
        Self(seed)
    }
    /// Draws the next integer.
    pub fn next_u64(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut value = self.0;
        value = (value ^ (value >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        value = (value ^ (value >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        value ^ (value >> 31)
    }
    /// Selects an index from a nonempty domain.
    pub fn index(&mut self, len: usize) -> usize {
        assert!(len > 0, "empty index domain");
        usize::try_from(self.next_u64() % u64::try_from(len).unwrap_or(u64::MAX)).unwrap_or(0)
    }
}
