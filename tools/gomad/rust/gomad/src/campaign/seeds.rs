use serde::{Deserialize, Serialize};
use std::{collections::BTreeSet, fmt};

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

    pub fn exclude_answered(&self, answered: &[u64]) -> Self {
        let mut excluded = answered.to_vec();
        excluded.sort_unstable();
        let mut ranges = Vec::new();
        let mut count = 0;
        for selected in &self.ranges {
            let mut start = selected.start;
            let mut exhausted = false;
            let index = excluded.partition_point(|seed| *seed < start);
            for &seed in &excluded[index..] {
                if seed > selected.end {
                    break;
                }
                if seed < start {
                    continue;
                }
                if start < seed {
                    ranges.push(SeedRange {
                        start,
                        end: seed - 1,
                    });
                    count += seed - start;
                }
                if seed == selected.end {
                    exhausted = true;
                    break;
                }
                start = seed + 1;
            }
            if !exhausted {
                ranges.push(SeedRange {
                    start,
                    end: selected.end,
                });
                count += selected.end - start + 1;
            }
        }
        Self { ranges, count }
    }

    /// # Errors
    /// Refuses a selection that cannot reserve its unguided fraction.
    pub fn mix_guided(&self, prioritized: &[u64]) -> Result<Self, String> {
        let unguided = self.count / 4 + u64::from(self.count % 4 != 0);
        let maximum_guided = self.count - unguided;
        let mut guided = Vec::new();
        let mut guided_set = BTreeSet::new();
        for &seed in prioritized {
            if guided.len() as u64 == maximum_guided {
                break;
            }
            if guided_set.insert(seed) {
                guided.push(seed);
            }
        }
        if guided.is_empty() {
            return Ok(self.clone());
        }
        let mut ranges: Vec<_> = guided
            .iter()
            .map(|&seed| SeedRange {
                start: seed,
                end: seed,
            })
            .collect();
        let mut remaining = self.count - guided.len() as u64;
        for selected in &self.ranges {
            let mut start = selected.start;
            let excluded: Vec<_> = guided_set
                .range(selected.start..=selected.end)
                .copied()
                .collect();
            for &seed in &excluded {
                if start < seed {
                    append_guided_range(&mut ranges, start, seed - 1, &mut remaining);
                }
                if remaining == 0 {
                    break;
                }
                if seed != u64::MAX {
                    start = seed + 1;
                }
            }
            if remaining != 0 && excluded.last() != Some(&u64::MAX) && start <= selected.end {
                append_guided_range(&mut ranges, start, selected.end, &mut remaining);
            }
            if remaining == 0 {
                break;
            }
        }
        if remaining != 0 {
            return Err("guided seed selection could not reserve its unguided fraction".into());
        }
        Ok(Self {
            ranges,
            count: self.count,
        })
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

fn append_guided_range(ranges: &mut Vec<SeedRange>, start: u64, mut end: u64, remaining: &mut u64) {
    let mut width = end - start + 1;
    if width > *remaining {
        end = start + (*remaining - 1);
        width = *remaining;
    }
    ranges.push(SeedRange { start, end });
    *remaining -= width;
}
