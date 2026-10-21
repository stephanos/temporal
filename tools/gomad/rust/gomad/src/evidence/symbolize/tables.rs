use super::{
    Frame, Reason,
    bytes::{Budget, Memory, Result, add, mul},
    wasm::Module,
};

mod validate;

const _: () = assert!(std::mem::size_of::<Function>() <= 128);

#[derive(Clone, Copy)]
pub(super) struct Table {
    pub start: u64,
    pub len: u64,
}

impl Table {
    fn at(self, offset: u64, length: u64) -> Result<u64> {
        if add(offset, length)? > self.len {
            return Err(Reason::SymbolMismatch);
        }
        add(self.start, offset)
    }

    fn string(self, memory: &Memory<'_>, offset: u64, budget: &mut Budget) -> Result<String> {
        memory.string(self.at(offset, 1)?, add(self.start, self.len)?, budget)
    }
}

pub(super) struct Function {
    pub offset: u64,
    pub end: u64,
    pub address: u64,
    pub index: u64,
    pub name: String,
    pub pcfile: u32,
    pub pcline: u32,
    pub npcdata: u32,
    pub cu: u32,
    pub nfuncdata: u8,
}

pub(super) struct Tables {
    pub text: u64,
    pub gofunc: u64,
    pub gofunc_end: u64,
    pub names: Table,
    pub cu: Table,
    pub files: Table,
    pub pc: Table,
    pub functions: Vec<Function>,
    pub select: Option<u64>,
}

impl Tables {
    pub fn read(module: &Module<'_>, budget: &mut Budget) -> Result<Self> {
        let memory = &module.memory;
        let headers = memory.matches(&[0xf1, 0xff, 0xff, 0xff, 0, 0, 1, 8], true, budget)?;
        if headers.is_empty() {
            return Err(Reason::NoSymbols);
        }
        let mut result = None;
        for header in headers {
            let modules = memory.matches(&header.to_le_bytes(), true, budget)?;
            if modules.is_empty() {
                continue;
            }
            let (count, nfiles, starts) = read_header(module, header, budget)?;
            let mut matched = false;
            for address in modules {
                // A header pointer alone is not moduledata. The first slice pointer anchors the pinned moduledata prefix.
                if memory.u64(add(address, 8)?, budget)? != starts[0] {
                    continue;
                }
                matched = true;
                let slices = read_slices(memory, address, &starts, budget)?;
                if slices[5].len != add(count, 1)? || slices[1].len * 4 > starts[2] - starts[1] {
                    return Err(Reason::SymbolMismatch);
                }
                for index in 0..4 {
                    let length = if index == 1 {
                        mul(slices[index].len, 4)?
                    } else {
                        slices[index].len
                    };
                    if add(starts[index], length)? > starts[index + 1] {
                        return Err(Reason::SymbolMismatch);
                    }
                }
                let text = memory.u64(add(address, 176)?, budget)?;
                let gofunc = memory.u64(add(address, 344)?, budget)?;
                let epcln = memory.u64(add(address, 352)?, budget)?;
                if text & 65535 != 0
                    || gofunc < add(slices[4].start, slices[4].len)?
                    || epcln < gofunc
                    || epcln > memory.end
                {
                    return Err(Reason::SymbolMismatch);
                }
                if result.is_some() {
                    return Err(Reason::SymbolMismatch);
                }
                memory.range(gofunc, epcln - gofunc, budget)?;
                let mut tables = Self {
                    text,
                    gofunc,
                    gofunc_end: epcln,
                    names: slices[0],
                    cu: Table {
                        start: slices[1].start,
                        len: mul(slices[1].len, 4)?,
                    },
                    files: slices[2],
                    pc: slices[3],
                    functions: Vec::new(),
                    select: None,
                };
                tables.validate_files(memory, nfiles, budget)?;
                let absent_names = tables.read_functions(module, slices[4], count, budget)?;
                tables.validate_functions(memory, budget)?;
                let first = tables.functions.first().ok_or(Reason::SymbolMismatch)?;
                let last = tables.functions.last().ok_or(Reason::SymbolMismatch)?;
                if memory.u64(add(address, 160)?, budget)? != add(text, mul(first.offset, 65536)?)?
                    || memory.u64(add(address, 168)?, budget)?
                        <= add(text, mul(last.offset, 65536)?)?
                    || memory.u64(add(address, 168)?, budget)? >= add(text, mul(last.end, 65536)?)?
                    || add(text, mul(last.end, 65536)?)?
                        > mul(add(0x1000, module.functions.len() as u64)?, 65536)?
                {
                    return Err(Reason::SymbolMismatch);
                }
                result = Some((tables, absent_names));
            }
            if !matched {
                return Err(Reason::SymbolMismatch);
            }
        }
        let (tables, absent) = result.ok_or(Reason::SymbolMismatch)?;
        if absent {
            return Err(Reason::NoSymbols);
        }
        Ok(tables)
    }

    fn read_functions(
        &mut self,
        module: &Module<'_>,
        table: Table,
        count: u64,
        budget: &mut Budget,
    ) -> Result<bool> {
        let memory = &module.memory;
        budget.bytes(mul(count, 128)?)?;
        self.functions = Vec::with_capacity(super::bytes::size(count)?);
        let mut absent_names = false;
        let mut previous = None;
        for index in 0..count {
            budget.steps(1)?;
            let cursor = table.at(mul(index, 8)?, 12)?;
            let offset = u64::from(memory.u32(cursor, budget)?);
            let end = u64::from(memory.u32(add(cursor, 8)?, budget)?);
            if offset >= end || previous.is_some_and(|old| old >= offset) {
                return Err(Reason::SymbolMismatch);
            }
            previous = Some(offset);
            let synthetic = add(self.text, mul(offset, 65536)?)?;
            let ordinal = (synthetic >> 16)
                .checked_sub(0x1000)
                .ok_or(Reason::SymbolMismatch)?;
            if ordinal >= module.functions.len() as u64 {
                return Err(Reason::SymbolMismatch);
            }
            let wasm_index = add(module.imported, ordinal)?;
            let record_offset = u64::from(memory.u32(add(cursor, 4)?, budget)?);
            if record_offset < mul(add(count, 1)?, 8)? {
                return Err(Reason::SymbolMismatch);
            }
            let record = table.at(record_offset, 44)?;
            if u64::from(memory.u32(record, budget)?) != offset {
                return Err(Reason::SymbolMismatch);
            }
            let name_offset = memory.u32(add(record, 4)?, budget)?;
            if name_offset > i32::MAX as u32 {
                return Err(Reason::SymbolMismatch);
            }
            let name = self.names.string(memory, u64::from(name_offset), budget)?;
            if name.is_empty() {
                return Err(Reason::SymbolMismatch);
            }
            if name == "runtime.selectgo" && self.select.replace(wasm_index).is_some() {
                return Err(Reason::SymbolMismatch);
            }
            if let Some(expected) = module
                .names
                .get(&u32::try_from(wasm_index).map_err(|_| Reason::SymbolTableTooLarge)?)
            {
                budget.steps(name.len() as u64)?;
                if !normalized_eq(&name, expected) {
                    return Err(Reason::SymbolMismatch);
                }
            } else {
                absent_names = true;
            }
            let npcdata = memory.u32(add(record, 28)?, budget)?;
            let nfuncdata = memory.byte(add(record, 43)?, budget)?;
            budget.entries(add(u64::from(npcdata), u64::from(nfuncdata))?)?;
            let record_length = add(44, mul(add(u64::from(npcdata), u64::from(nfuncdata))?, 4)?)?;
            table.at(record_offset, record_length)?;
            let next = if index + 1 < count {
                u64::from(memory.u32(add(cursor, 12)?, budget)?)
            } else {
                table.len
            };
            if add(record_offset, record_length)? > next {
                return Err(Reason::SymbolMismatch);
            }
            let pcfile = memory.u32(add(record, 20)?, budget)?;
            let pcline = memory.u32(add(record, 24)?, budget)?;
            for offset in [memory.u32(add(record, 16)?, budget)?, pcfile, pcline] {
                self.pc.at(u64::from(offset), 1)?;
            }
            for slot in 0..npcdata {
                self.pc.at(
                    u64::from(memory.u32(add(record, 44 + u64::from(slot) * 4)?, budget)?),
                    1,
                )?;
            }
            self.functions.push(Function {
                offset,
                end,
                address: record,
                index: wasm_index,
                name,
                pcfile,
                pcline,
                npcdata,
                cu: memory.u32(add(record, 32)?, budget)?,
                nfuncdata,
            });
        }
        Ok(absent_names)
    }

    pub fn pcvalue(
        &self,
        memory: &Memory<'_>,
        offset: u32,
        target: u64,
        budget: &mut Budget,
    ) -> Result<i32> {
        if offset == 0 {
            return Ok(-1);
        }
        let mut cursor = u64::from(offset);
        let mut pc = 0;
        let mut value: i32 = -1;
        let mut first = true;
        loop {
            budget.steps(1)?;
            let delta = self.varint(memory, &mut cursor, budget)?;
            if delta == 0 && !first {
                return Err(Reason::SymbolMismatch);
            }
            first = false;
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
            if target < pc {
                return Ok(value);
            }
        }
    }

    fn varint(&self, memory: &Memory<'_>, cursor: &mut u64, budget: &mut Budget) -> Result<u32> {
        let mut result = 0_u32;
        for shift in (0..35).step_by(7) {
            let byte = memory.byte(self.pc.at(*cursor, 1)?, budget)?;
            *cursor = add(*cursor, 1)?;
            if shift == 28 && byte > 15 {
                return Err(Reason::SymbolTableTooLarge);
            }
            result |= u32::from(byte & 0x7f) << shift;
            if byte & 0x80 == 0 {
                return Ok(result);
            }
        }
        Err(Reason::SymbolTableTooLarge)
    }

    fn location(
        &self,
        memory: &Memory<'_>,
        function: &Function,
        target: u64,
        budget: &mut Budget,
    ) -> Result<(Option<String>, Option<i32>)> {
        let file = self.pcvalue(memory, function.pcfile, target, budget)?;
        let line = self.pcvalue(memory, function.pcline, target, budget)?;
        if file == -1 || line <= 0 {
            return Ok((None, None));
        }
        let file = u64::try_from(file).map_err(|_| Reason::SymbolMismatch)?;
        let offset = memory.u32(
            self.cu.at(mul(add(u64::from(function.cu), file)?, 4)?, 4)?,
            budget,
        )?;
        if offset == u32::MAX {
            return Ok((None, None));
        }
        Ok((
            Some(self.files.string(memory, u64::from(offset), budget)?),
            Some(line),
        ))
    }

    pub fn chain(
        &self,
        memory: &Memory<'_>,
        function: &Function,
        mut target: u64,
        budget: &mut Budget,
    ) -> Result<Vec<Frame>> {
        let inline_pc = if function.npcdata > 2 {
            memory.u32(add(function.address, 52)?, budget)?
        } else {
            0
        };
        let inline_data = if function.nfuncdata > 3 {
            Some(memory.u32(
                add(function.address, 44 + u64::from(function.npcdata) * 4 + 12)?,
                budget,
            )?)
        } else {
            None
        };
        let mut chain = Vec::new();
        let mut seen = std::collections::BTreeSet::new();
        loop {
            if chain.len() == 256 {
                return Err(Reason::SymbolTableTooLarge);
            }
            budget.entries(1)?;
            budget.bytes(super::bytes::FRAME_CHARGE * 4 + 64)?;
            let index = self.pcvalue(memory, inline_pc, target, budget)?;
            if index < -1 || index >= 0 && (inline_data.is_none() || inline_data == Some(u32::MAX))
            {
                return Err(Reason::SymbolMismatch);
            }
            if index != -1 && inline_data.is_some() && inline_data != Some(u32::MAX) {
                let index = u64::try_from(index).map_err(|_| Reason::SymbolMismatch)?;
                if !seen.insert(index) {
                    return Err(Reason::SymbolMismatch);
                }
                let row = add(
                    add(
                        self.gofunc,
                        u64::from(inline_data.ok_or(Reason::SymbolMismatch)?),
                    )?,
                    mul(index, 16)?,
                )?;
                if row < self.gofunc || add(row, 16)? > self.gofunc_end {
                    return Err(Reason::SymbolMismatch);
                }
                memory.range(row, 16, budget)?;
                let name_offset = memory.u32(add(row, 4)?, budget)?;
                let parent = memory.u32(add(row, 8)?, budget)?;
                if parent > i32::MAX as u32
                    || u64::from(parent) >= mul(function.end - function.offset, 65536)?
                {
                    return Err(Reason::SymbolMismatch);
                }
                let name = self.names.string(memory, u64::from(name_offset), budget)?;
                budget.bytes(mul(name.len() as u64, 2)?)?;
                budget.steps(mul(name.len() as u64, 3)?)?;
                let mut frame = split(&name);
                (frame.file, frame.line) = self.location(memory, function, target, budget)?;
                chain.push(frame);
                target = u64::from(parent);
                continue;
            }
            budget.bytes(mul(function.name.len() as u64, 2)?)?;
            budget.steps(mul(function.name.len() as u64, 3)?)?;
            let mut frame = split(&function.name);
            (frame.file, frame.line) = self.location(memory, function, target, budget)?;
            chain.push(frame);
            return Ok(chain);
        }
    }
}

fn read_header(
    module: &Module<'_>,
    header: u64,
    budget: &mut Budget,
) -> Result<(u64, u64, [u64; 5])> {
    let memory = &module.memory;
    let count = memory.u64(add(header, 8)?, budget)?;
    let nfiles = memory.u64(add(header, 16)?, budget)?;
    budget.entries(add(count, nfiles)?)?;
    if count == 0
        || count > module.functions.len() as u64
        || memory.u64(add(header, 24)?, budget)? != 0
    {
        return Err(Reason::SymbolMismatch);
    }
    let mut starts = [0; 5];
    for (index, start) in starts.iter_mut().enumerate() {
        *start = add(
            header,
            memory.u64(add(header, 32 + index as u64 * 8)?, budget)?,
        )?;
    }
    if starts[0] < add(header, 72)? || starts.windows(2).any(|pair| pair[0] > pair[1]) {
        return Err(Reason::SymbolMismatch);
    }
    Ok((count, nfiles, starts))
}

fn read_slices(
    memory: &Memory<'_>,
    address: u64,
    starts: &[u64; 5],
    budget: &mut Budget,
) -> Result<[Table; 6]> {
    let mut slices = [Table { start: 0, len: 0 }; 6];
    for (index, table) in slices.iter_mut().enumerate() {
        let cursor = add(address, 8 + index as u64 * 24)?;
        table.start = memory.u64(cursor, budget)?;
        table.len = memory.u64(add(cursor, 8)?, budget)?;
        let capacity = memory.u64(add(cursor, 16)?, budget)?;
        if capacity != table.len || table.start != starts[index.min(4)] {
            return Err(Reason::SymbolMismatch);
        }
        let length = match index {
            1 => mul(table.len, 4)?,
            5 => mul(table.len, 8)?,
            _ => table.len,
        };
        memory.range(table.start, length, budget)?;
    }
    Ok(slices)
}

fn normalized_eq(go: &str, wasm: &str) -> bool {
    go.chars()
        .map(|ch| {
            if ch.is_ascii_alphanumeric() || ch == '_' || ch == '.' {
                ch
            } else {
                '_'
            }
        })
        .eq(wasm.chars())
}

pub(super) fn split(name: &str) -> Frame {
    let mut masked = name.as_bytes().to_vec();
    let mut depth = 0_i32;
    for byte in &mut masked {
        match *byte {
            b'[' => depth += 1,
            b']' => {
                depth -= 1;
                if depth < 0 {
                    break;
                }
            }
            _ if depth > 0 => *byte = b'_',
            _ => {}
        }
    }
    if depth != 0 {
        masked = name.as_bytes().to_vec();
    }
    let package_end = if name.starts_with("go:") || name.starts_with("type:") {
        None
    } else {
        let slash = masked.iter().rposition(|byte| *byte == b'/').unwrap_or(0);
        masked[slash..]
            .iter()
            .position(|byte| *byte == b'.')
            .map(|index| slash + index)
    };
    let (package, function) =
        package_end.map_or(("", name), |index| (&name[..index], &name[index + 1..]));
    Frame {
        package: package.into(),
        function: function.into(),
        file: None,
        line: None,
    }
}
