use super::Reason;

pub(super) type Result<T> = std::result::Result<T, Reason>;

pub(super) const BYTE_LIMIT: u64 = 64 << 20;
pub(super) const ENTRY_LIMIT: u64 = 1 << 20;
pub(super) const STEP_LIMIT: u64 = 1 << 24;
pub(super) const STRING_LIMIT: u64 = 1 << 20;
pub(super) const FRAME_CHARGE: u64 = 128;

pub(super) fn add(a: u64, b: u64) -> Result<u64> {
    a.checked_add(b).ok_or(Reason::SymbolTableTooLarge)
}

pub(super) fn mul(a: u64, b: u64) -> Result<u64> {
    a.checked_mul(b).ok_or(Reason::SymbolTableTooLarge)
}

pub(super) fn size(value: u64) -> Result<usize> {
    usize::try_from(value).map_err(|_| Reason::SymbolTableTooLarge)
}

#[derive(Default)]
pub(super) struct Budget {
    pub bytes: u64,
    pub code: u64,
    pub entries: u64,
    pub steps: u64,
}

impl Budget {
    pub fn charge(counter: &mut u64, amount: u64, limit: u64) -> Result<()> {
        *counter = add(*counter, amount)?;
        if *counter > limit {
            return Err(Reason::SymbolTableTooLarge);
        }
        Ok(())
    }

    pub fn bytes(&mut self, amount: u64) -> Result<()> {
        Self::charge(&mut self.bytes, amount, BYTE_LIMIT)
    }

    pub fn entries(&mut self, amount: u64) -> Result<()> {
        Self::charge(&mut self.entries, amount, ENTRY_LIMIT)
    }

    pub fn steps(&mut self, amount: u64) -> Result<()> {
        Self::charge(&mut self.steps, amount, STEP_LIMIT)
    }
}

#[derive(Clone, Copy)]
pub(super) struct Segment<'a> {
    pub start: u64,
    pub bytes: &'a [u8],
}

pub(super) struct Memory<'a> {
    pub segments: Vec<Segment<'a>>,
    pub end: u64,
}

impl Memory<'_> {
    pub fn range(&self, address: u64, length: u64, budget: &mut Budget) -> Result<()> {
        budget.bytes(length)?;
        if add(address, length)? > self.end {
            return Err(Reason::SymbolMismatch);
        }
        Ok(())
    }

    pub fn byte(&self, address: u64, budget: &mut Budget) -> Result<u8> {
        self.range(address, 1, budget)?;
        budget.steps(1)?;
        let mut comparisons = 0;
        let index = self.segments.partition_point(|segment| {
            comparisons += 1;
            segment.start <= address
        });
        budget.steps(comparisons)?;
        if let Some(segment) = index.checked_sub(1).map(|index| &self.segments[index])
            && let Some(value) = segment.bytes.get(size(address - segment.start)?)
        {
            return Ok(*value);
        }
        Ok(0)
    }

    pub fn uint<const N: usize>(&self, address: u64, budget: &mut Budget) -> Result<[u8; N]> {
        let mut bytes = [0; N];
        for (index, byte) in bytes.iter_mut().enumerate() {
            *byte = self.byte(add(address, index as u64)?, budget)?;
        }
        Ok(bytes)
    }

    pub fn u32(&self, address: u64, budget: &mut Budget) -> Result<u32> {
        Ok(u32::from_le_bytes(self.uint(address, budget)?))
    }

    pub fn u64(&self, address: u64, budget: &mut Budget) -> Result<u64> {
        Ok(u64::from_le_bytes(self.uint(address, budget)?))
    }

    pub fn string(&self, address: u64, end: u64, budget: &mut Budget) -> Result<String> {
        if address >= end || end > self.end {
            return Err(Reason::SymbolMismatch);
        }
        budget.bytes(8)?;
        let mut bytes = Vec::new();
        for offset in 0..=STRING_LIMIT {
            let cursor = add(address, offset)?;
            if cursor >= end {
                return Err(Reason::SymbolMismatch);
            }
            let byte = self.byte(cursor, budget)?;
            if byte == 0 {
                return String::from_utf8(bytes).map_err(|_| Reason::SymbolMismatch);
            }
            if offset == STRING_LIMIT {
                return Err(Reason::SymbolTableTooLarge);
            }
            budget.bytes(4)?;
            bytes.push(byte);
        }
        Err(Reason::SymbolTableTooLarge)
    }

    // A framing marker begins with a nonzero byte, so it cannot start in an
    // elided gap. Scan borrowed data in fixed eight-byte chunks; only segment
    // tails need sparse memory reads to cover adjacent segments and gaps.
    pub fn framing_matches(&self, needle: &[u8; 16], budget: &mut Budget) -> Result<Vec<u64>> {
        if needle[0] == 0 {
            return Err(Reason::SymbolMismatch);
        }
        let mut found = Vec::new();
        for segment in &self.segments {
            let interior = segment.bytes.len().saturating_sub(needle.len() - 1);
            for (chunk_index, chunk) in segment.bytes[..interior].chunks(8).enumerate() {
                budget.steps(1)?;
                if !chunk.contains(&needle[0]) {
                    continue;
                }
                for (offset, byte) in chunk.iter().enumerate() {
                    if *byte != needle[0] {
                        continue;
                    }
                    budget.steps(1)?;
                    let index = chunk_index * 8 + offset;
                    if segment.bytes[index..index + needle.len()] == *needle {
                        budget.entries(1)?;
                        budget.bytes(32)?;
                        found.push(add(segment.start, index as u64)?);
                    }
                }
            }
            for index in interior..segment.bytes.len() {
                budget.steps(1)?;
                if segment.bytes[index] != needle[0] {
                    continue;
                }
                let address = add(segment.start, index as u64)?;
                if add(address, needle.len() as u64)? <= self.end
                    && self.uint::<16>(address, budget)? == *needle
                {
                    budget.entries(1)?;
                    budget.bytes(32)?;
                    found.push(address);
                }
            }
        }
        Ok(found)
    }

    pub fn matches(&self, needle: &[u8], aligned: bool, budget: &mut Budget) -> Result<Vec<u64>> {
        let mut found = Vec::new();
        let mut previous_end = 0;
        for segment in &self.segments {
            let start = segment
                .start
                .saturating_sub(needle.len() as u64 - 1)
                .max(previous_end);
            let end = add(segment.start, segment.bytes.len() as u64)?;
            for address in start..end {
                budget.steps(1)?;
                if aligned && address % 8 != 0 {
                    continue;
                }
                let mut matches = true;
                for (offset, expected) in needle.iter().enumerate() {
                    if add(address, needle.len() as u64)? > self.end
                        || self.byte(add(address, offset as u64)?, budget)? != *expected
                    {
                        matches = false;
                        break;
                    }
                }
                if matches {
                    budget.entries(1)?;
                    budget.bytes(32)?;
                    found.push(address);
                }
            }
            previous_end = end;
        }
        Ok(found)
    }
}
