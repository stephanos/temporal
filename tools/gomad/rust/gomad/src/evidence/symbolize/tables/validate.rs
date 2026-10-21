use super::super::{
    Reason,
    bytes::{Budget, Memory, Result, add, mul, size},
};
use super::{Function, Tables};

struct Run {
    end: u64,
    value: i32,
}

const _: () = {
    assert!(std::mem::size_of::<Run>() <= 16);
    assert!(std::mem::size_of::<Option<usize>>() <= 16);
    assert!(std::mem::size_of::<usize>() <= 8);
};

impl Tables {
    pub(super) fn validate_files(
        &self,
        memory: &Memory<'_>,
        count: u64,
        budget: &mut Budget,
    ) -> Result<()> {
        let mut cursor = 0;
        for _ in 0..count {
            let name = self.files.string(memory, cursor, budget)?;
            if name.is_empty() {
                return Err(Reason::SymbolMismatch);
            }
            cursor = add(cursor, add(name.len() as u64, 1)?)?;
        }
        if self.files.len - cursor > 7 {
            return Err(Reason::SymbolMismatch);
        }
        for offset in cursor..self.files.len {
            if memory.byte(self.files.at(offset, 1)?, budget)? != 0 {
                return Err(Reason::SymbolMismatch);
            }
        }
        for slot in 0..self.cu.len / 4 {
            budget.steps(1)?;
            let offset = memory.u32(self.cu.at(mul(slot, 4)?, 4)?, budget)?;
            if offset == u32::MAX {
                continue;
            }
            let offset = u64::from(offset);
            if memory.byte(self.files.at(offset, 1)?, budget)? == 0
                || offset > 0 && memory.byte(self.files.at(offset - 1, 1)?, budget)? != 0
            {
                return Err(Reason::SymbolMismatch);
            }
        }
        Ok(())
    }

    pub(super) fn validate_functions(
        &self,
        memory: &Memory<'_>,
        budget: &mut Budget,
    ) -> Result<()> {
        for function in &self.functions {
            let offsets = [
                memory.u32(add(function.address, 16)?, budget)?,
                function.pcfile,
                function.pcline,
            ];
            for (kind, offset) in offsets.into_iter().enumerate() {
                let runs = self.runs(memory, function, offset, budget)?;
                for run in runs {
                    budget.steps(1)?;
                    if kind == 1 && run.value >= 0 {
                        let file = u64::try_from(run.value).map_err(|_| Reason::SymbolMismatch)?;
                        self.cu.at(mul(add(u64::from(function.cu), file)?, 4)?, 4)?;
                    } else if kind > 0 && run.value < -1 {
                        return Err(Reason::SymbolMismatch);
                    }
                }
            }
            for slot in 0..function.npcdata {
                let offset =
                    memory.u32(add(function.address, 44 + u64::from(slot) * 4)?, budget)?;
                let runs = self.runs(memory, function, offset, budget)?;
                if slot == 2 {
                    self.validate_inline(memory, function, &runs, budget)?;
                }
            }
            for slot in 0..function.nfuncdata {
                let cursor = add(
                    function.address,
                    44 + u64::from(function.npcdata) * 4 + u64::from(slot) * 4,
                )?;
                let offset = memory.u32(cursor, budget)?;
                if offset != u32::MAX && add(self.gofunc, u64::from(offset))? >= self.gofunc_end {
                    return Err(Reason::SymbolMismatch);
                }
            }
        }
        Ok(())
    }

    fn runs(
        &self,
        memory: &Memory<'_>,
        function: &Function,
        offset: u32,
        budget: &mut Budget,
    ) -> Result<Vec<Run>> {
        if offset == 0 {
            return Ok(Vec::new());
        }
        let mut cursor = u64::from(offset);
        let mut value = -1_i32;
        let mut pc = 0;
        let mut runs = Vec::new();
        loop {
            budget.steps(1)?;
            let delta = self.varint(memory, &mut cursor, budget)?;
            if delta == 0 && !runs.is_empty() {
                return Ok(runs);
            }
            let signed =
                i32::from_ne_bytes(((delta >> 1) ^ (0_u32.wrapping_sub(delta & 1))).to_ne_bytes());
            value = value
                .checked_add(signed)
                .ok_or(Reason::SymbolTableTooLarge)?;
            let advance = self.varint(memory, &mut cursor, budget)?;
            if advance == 0 {
                return Err(Reason::SymbolMismatch);
            }
            pc = add(pc, u64::from(advance))?;
            if pc > mul(function.end - function.offset, 65536)? {
                return Err(Reason::SymbolMismatch);
            }
            budget.entries(1)?;
            budget.bytes(64)?;
            runs.push(Run { end: pc, value });
        }
    }

    fn validate_inline(
        &self,
        memory: &Memory<'_>,
        function: &Function,
        runs: &[Run],
        budget: &mut Budget,
    ) -> Result<()> {
        budget.steps(mul(runs.len() as u64, 2)?)?;
        if runs.iter().any(|run| run.value < -1) {
            return Err(Reason::SymbolMismatch);
        }
        let Some(maximum) = runs
            .iter()
            .map(|run| run.value)
            .filter(|value| *value >= 0)
            .max()
        else {
            return Ok(());
        };
        let count = add(
            u64::try_from(maximum).map_err(|_| Reason::SymbolMismatch)?,
            1,
        )?;
        budget.entries(count)?;
        budget.bytes(mul(count, 32)?)?;
        if function.nfuncdata <= 3 {
            return Err(Reason::SymbolMismatch);
        }
        let offset = memory.u32(
            add(function.address, 44 + u64::from(function.npcdata) * 4 + 12)?,
            budget,
        )?;
        if offset == u32::MAX {
            return Err(Reason::SymbolMismatch);
        }
        let base = add(self.gofunc, u64::from(offset))?;
        let length = mul(count, 16)?;
        if add(base, length)? > self.gofunc_end {
            return Err(Reason::SymbolMismatch);
        }
        memory.range(base, length, budget)?;
        let mut edges = Vec::with_capacity(size(count)?);
        for index in 0..count {
            budget.steps(1)?;
            let row = add(base, mul(index, 16)?)?;
            let name = memory.u32(add(row, 4)?, budget)?;
            if name > i32::MAX as u32
                || self
                    .names
                    .string(memory, u64::from(name), budget)?
                    .is_empty()
            {
                return Err(Reason::SymbolMismatch);
            }
            let parent = u64::from(memory.u32(add(row, 8)?, budget)?);
            if parent > i32::MAX as u64 {
                return Err(Reason::SymbolMismatch);
            }
            let mut comparisons = 0;
            let position = runs.partition_point(|run| {
                comparisons += 1;
                run.end <= parent
            });
            budget.steps(comparisons)?;
            let value = runs.get(position).ok_or(Reason::SymbolMismatch)?.value;
            edges.push(if value == -1 {
                None
            } else {
                Some(usize::try_from(value).map_err(|_| Reason::SymbolMismatch)?)
            });
        }
        check_chains(&edges, budget)
    }
}

fn check_chains(edges: &[Option<usize>], budget: &mut Budget) -> Result<()> {
    let mut finished = vec![0_u16; edges.len()];
    let mut visiting = vec![false; edges.len()];
    for origin in 0..edges.len() {
        let mut path = Vec::new();
        let mut current = Some(origin);
        while let Some(index) = current {
            budget.steps(1)?;
            if index >= edges.len() || visiting[index] {
                return Err(Reason::SymbolMismatch);
            }
            if finished[index] != 0 {
                if path.len() + usize::from(finished[index]) > 255 {
                    return Err(Reason::SymbolTableTooLarge);
                }
                break;
            }
            if path.len() == 255 {
                return Err(Reason::SymbolTableTooLarge);
            }
            budget.bytes(32)?;
            visiting[index] = true;
            path.push(index);
            current = edges[index];
        }
        let mut depth = current.map_or(0, |index| finished[index]);
        budget.steps(path.len() as u64)?;
        for index in path.into_iter().rev() {
            depth += 1;
            finished[index] = depth;
            visiting[index] = false;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn symbolize_cached_parent_depth_preserves_limit_and_cycle_detection() {
        let edges: Vec<_> = (0..256_usize).map(|index| index.checked_sub(1)).collect();
        check_chains(&edges[..255], &mut Budget::default()).unwrap();
        assert_eq!(
            check_chains(&edges, &mut Budget::default()),
            Err(Reason::SymbolTableTooLarge)
        );
        assert_eq!(
            check_chains(&[Some(1), Some(0)], &mut Budget::default()),
            Err(Reason::SymbolMismatch)
        );
    }
}
