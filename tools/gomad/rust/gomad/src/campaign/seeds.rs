use serde::{Deserialize, Serialize};
use std::fmt;

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct SeedSelection {
    ranges: Vec<SeedRange>,
    count: u64,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct SeedRange {
    start: u64,
    end: u64,
}

impl SeedSelection {
    /// # Errors
    /// Refuses malformed seeds, reversed or overlapping ranges and count overflow.
    pub fn parse(input: &str) -> Result<Self, String> {
        if input.is_empty() {
            return Err("invalid seed selection: empty input".into());
        }
        let mut ranges = Vec::new();
        let mut count = 0_u64;
        for term in input.split(',') {
            let parts: Vec<_> = term.split('-').collect();
            if parts.len() > 2 {
                return Err(format!("invalid seed selection term {term:?}"));
            }
            let parse = |part| {
                parse_seed(part)
                    .map_err(|error| format!("invalid seed selection term {term:?}: {error}"))
            };
            let start = parse(parts[0])?;
            let end = if parts.len() == 2 {
                parse(parts[1])?
            } else {
                start
            };
            if end < start {
                return Err(format!(
                    "invalid seed selection term {term:?}: reversed range"
                ));
            }
            count = end
                .checked_sub(start)
                .and_then(|width| width.checked_add(1))
                .and_then(|width| count.checked_add(width))
                .ok_or("invalid seed selection: selected count overflows uint64")?;
            ranges.push(SeedRange { start, end });
        }
        let mut sorted = ranges.clone();
        sorted.sort_unstable_by_key(|range| (range.start, range.end));
        if sorted.windows(2).any(|pair| pair[1].start <= pair[0].end) {
            return Err("invalid seed selection: duplicate or overlapping ranges".into());
        }
        Ok(Self { ranges, count })
    }

    pub fn count(&self) -> u64 {
        self.count
    }

    pub fn seed_at(&self, mut ordinal: u64) -> Option<u64> {
        if ordinal >= self.count {
            return None;
        }
        for selected in &self.ranges {
            let width = selected.end - selected.start + 1;
            if ordinal < width {
                return Some(selected.start + ordinal);
            }
            ordinal -= width;
        }
        None
    }
}

impl fmt::Display for SeedSelection {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        for (index, selected) in self.ranges.iter().enumerate() {
            if index != 0 {
                formatter.write_str(",")?;
            }
            write!(formatter, "{}", selected.start)?;
            if selected.end != selected.start {
                write!(formatter, "-{}", selected.end)?;
            }
        }
        Ok(())
    }
}

fn parse_seed(input: &str) -> Result<u64, String> {
    if input.is_empty() {
        return Err("empty seed".into());
    }
    if input.len() > 1 && input.starts_with('0') {
        return Err("leading zero".into());
    }
    if !input.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err("non-decimal seed".into());
    }
    input.parse().map_err(|_| {
        format!("seed out of range: strconv.ParseUint: parsing {input:?}: value out of range")
    })
}
