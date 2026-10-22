use super::{
    Reason,
    bytes::{BYTE_LIMIT, Budget, Memory, Result, Segment, add, mul, size},
};
use gomad_wasmhost::wasmparser::{
    self, BinaryReader, CustomSectionReader, FunctionBody, Name, Operator, Parser, Payload,
    Subsection, TypeRef,
};
use std::collections::BTreeMap;

const _: () = {
    assert!(std::mem::size_of::<FunctionBody<'_>>() <= 128);
    assert!(std::mem::size_of::<(u32, &str)>() <= 32);
    assert!(std::mem::size_of::<Segment<'_>>() <= 32);
    assert!(std::mem::size_of::<Operator<'_>>() <= 256);
    assert!(std::mem::size_of::<wasmparser::Catch>() <= 16);
    assert!(std::mem::size_of::<wasmparser::Handle>() <= 16);
    assert!(std::mem::size_of::<wasmparser::ValType>() <= 32);
    assert!(std::mem::size_of::<wasmparser::FrameKind>() <= 1);
};

pub(super) struct Module<'a> {
    pub memory: Memory<'a>,
    pub pinned_go_producer: bool,
    pub imported: u64,
    pub functions: Vec<FunctionBody<'a>>,
    pub names: BTreeMap<u32, &'a str>,
}

fn checked<T>(value: wasmparser::Result<T>) -> Result<T> {
    value.map_err(|_| Reason::SymbolMismatch)
}

impl<'a> Module<'a> {
    fn empty() -> Self {
        Self {
            memory: Memory {
                segments: Vec::new(),
                end: 0,
            },
            pinned_go_producer: false,
            imported: 0,
            functions: Vec::new(),
            names: BTreeMap::new(),
        }
    }

    pub fn read(bytes: &'a [u8], budget: &mut Budget) -> Result<Self> {
        Self::read_parts(bytes, budget, true)
    }

    pub fn read_metadata(bytes: &'a [u8], budget: &mut Budget) -> Result<Self> {
        Self::read_parts(bytes, budget, false)
    }

    fn read_parts(bytes: &'a [u8], budget: &mut Budget, symbols: bool) -> Result<Self> {
        let mut module = Self::empty();
        let mut count = 0;
        let mut bodies = 0;
        let mut memory_count = 0;
        let mut has_names = false;
        let mut has_producers = false;
        for payload in Parser::new(0).parse_all(bytes) {
            budget.entries(1)?;
            match checked(payload)? {
                Payload::Version {
                    num: 1,
                    encoding: wasmparser::Encoding::Module,
                    ..
                } => {}
                Payload::Version { .. } => return Err(Reason::NoSymbols),
                Payload::ImportSection(reader) => {
                    budget.steps(reader.range().len() as u64)?;
                    for import in reader.into_imports() {
                        budget.entries(1)?;
                        match checked(import)?.ty {
                            TypeRef::Func(_) | TypeRef::FuncExact(_) => {
                                module.imported = add(module.imported, 1)?;
                            }
                            TypeRef::Memory(_) => return Err(Reason::NoSymbols),
                            _ => {}
                        }
                    }
                }
                Payload::FunctionSection(reader) => {
                    count = u64::from(reader.count());
                    budget.entries(count)?;
                    if symbols {
                        budget.bytes(mul(count, 128)?)?;
                        module.functions = Vec::with_capacity(size(count)?);
                    }
                    for entry in reader {
                        checked(entry)?;
                    }
                }
                Payload::MemorySection(reader) => {
                    for memory in reader {
                        let memory = checked(memory)?;
                        memory_count += 1;
                        if memory_count != 1
                            || memory.memory64
                            || memory.shared
                            || memory.page_size_log2.is_some()
                        {
                            return Err(Reason::NoSymbols);
                        }
                        module.memory.end = mul(memory.initial, 65536)?;
                    }
                }
                Payload::DataSection(reader) => {
                    // Advancing the lazy Data reader would parse arbitrary expressions and
                    // allocate their control stacks before our budget/grammar checks.
                    if symbols {
                        budget.steps(reader.range().len() as u64)?;
                    }
                    budget.entries(u64::from(reader.count()))?;
                    let mut data = BinaryReader::new(
                        &bytes[reader.original_position()..reader.range().end],
                        reader.original_position(),
                    );
                    for _ in 0..reader.count() {
                        module.read_data(&mut data, budget)?;
                    }
                    if !data.eof() {
                        return Err(Reason::SymbolMismatch);
                    }
                }
                Payload::CodeSectionStart {
                    count: declared, ..
                } if u64::from(declared) != count => return Err(Reason::SymbolMismatch),
                Payload::CodeSectionEntry(body) => {
                    if bodies >= count {
                        return Err(Reason::SymbolMismatch);
                    }
                    bodies += 1;
                    if symbols {
                        Budget::charge(&mut budget.code, body.range().len() as u64, BYTE_LIMIT)?;
                        module.functions.push(body);
                    }
                }
                Payload::CustomSection(section) if section.name() == "name" => {
                    if has_names {
                        return Err(Reason::SymbolMismatch);
                    }
                    has_names = true;
                    module.read_names(&section, budget)?;
                }
                Payload::CustomSection(section) if section.name() == "producers" => {
                    if has_producers {
                        return Err(Reason::SymbolMismatch);
                    }
                    has_producers = true;
                    module.pinned_go_producer = producers(&section, budget)?;
                }
                _ => {}
            }
        }
        module.finish(count, bodies, budget)
    }

    fn finish(mut self, count: u64, bodies: u64, budget: &mut Budget) -> Result<Self> {
        if bodies != count {
            return Err(Reason::SymbolMismatch);
        }
        let total = add(self.imported, count)?;
        if self.names.keys().any(|index| u64::from(*index) >= total) {
            return Err(Reason::SymbolMismatch);
        }
        let segments = self.memory.segments.len() as u64;
        if segments > 0 {
            budget.steps(mul(segments, u64::from(segments.ilog2()) + 1)?)?;
        }
        self.memory
            .segments
            .sort_unstable_by_key(|segment| segment.start);
        for pair in self.memory.segments.windows(2) {
            if add(pair[0].start, pair[0].bytes.len() as u64)? > pair[1].start {
                return Err(Reason::SymbolMismatch);
            }
        }
        Ok(self)
    }

    pub fn body(&self, index: u64) -> Result<&FunctionBody<'a>> {
        self.functions
            .get(size(
                index
                    .checked_sub(self.imported)
                    .ok_or(Reason::SymbolMismatch)?,
            )?)
            .ok_or(Reason::SymbolMismatch)
    }

    fn read_names(&mut self, section: &CustomSectionReader<'a>, budget: &mut Budget) -> Result<()> {
        budget.bytes(section.data().len() as u64)?;
        budget.steps(section.data().len() as u64)?;
        let mut reader = BinaryReader::new(section.data(), section.data_offset());
        while !reader.eof() {
            budget.entries(1)?;
            let id = checked(reader.read_u8())?;
            if id > 127 {
                return Err(Reason::SymbolMismatch);
            }
            let mut subsection = checked(reader.read_reader())?;
            if id == 0 {
                read_name(&mut subsection)?;
                if !subsection.eof() {
                    return Err(Reason::SymbolMismatch);
                }
                continue;
            }
            if matches!(id, 2 | 3 | 10) {
                for _ in 0..checked(subsection.read_var_u32())? {
                    budget.entries(1)?;
                    checked(subsection.read_var_u32())?;
                    for _ in 0..checked(subsection.read_var_u32())? {
                        budget.entries(1)?;
                        checked(subsection.read_var_u32())?;
                        read_name(&mut subsection)?;
                    }
                }
                if !subsection.eof() {
                    return Err(Reason::SymbolMismatch);
                }
                continue;
            }
            match checked(Name::from_reader(id, subsection))? {
                Name::Function(map) => {
                    for name in map {
                        budget.entries(1)?;
                        let name = checked(name)?;
                        bounded_name(name.name)?;
                        if self.names.is_empty() {
                            budget.bytes(512)?;
                        }
                        budget.bytes(128)?;
                        if self.names.insert(name.index, name.name).is_some() {
                            return Err(Reason::SymbolMismatch);
                        }
                    }
                }
                Name::Type(map)
                | Name::Table(map)
                | Name::Memory(map)
                | Name::Global(map)
                | Name::Element(map)
                | Name::Data(map)
                | Name::Tag(map) => {
                    for name in map {
                        budget.entries(1)?;
                        bounded_name(checked(name)?.name)?;
                    }
                }
                Name::Local(_) | Name::Label(_) | Name::Field(_) | Name::Module { .. } => {
                    return Err(Reason::SymbolMismatch);
                }
                Name::Unknown { .. } => {}
            }
        }
        Ok(())
    }

    fn read_data(&mut self, reader: &mut BinaryReader<'a>, budget: &mut Budget) -> Result<()> {
        budget.bytes(128)?;
        match checked(reader.read_var_u32())? {
            0 => {}
            2 if checked(reader.read_var_u32())? == 0 => {}
            1 | 2 => return Err(Reason::NoSymbols),
            _ => return Err(Reason::SymbolMismatch),
        }
        if checked(reader.read_u8())? != 0x41 {
            return Err(Reason::NoSymbols);
        }
        let value = checked(reader.read_var_i32())?;
        let start = u64::from(u32::from_ne_bytes(value.to_ne_bytes()));
        if checked(reader.read_u8())? != 0x0b {
            return Err(Reason::SymbolMismatch);
        }
        let length = checked(reader.read_var_u32())?;
        let data = checked(reader.read_bytes(length as usize))?;
        if add(start, u64::from(length))? > self.memory.end {
            return Err(Reason::SymbolMismatch);
        }
        self.memory.segments.push(Segment { start, bytes: data });
        Ok(())
    }
}

fn producers(section: &CustomSectionReader<'_>, budget: &mut Budget) -> Result<bool> {
    budget.bytes(section.data().len() as u64)?;
    budget.steps(section.data().len() as u64)?;
    let mut reader = BinaryReader::new(section.data(), section.data_offset());
    let mut language = false;
    let mut compiler = false;
    for _ in 0..checked(reader.read_var_u32())? {
        budget.entries(1)?;
        let field = read_name(&mut reader)?;
        if !matches!(field, "language" | "sdk" | "processed-by") {
            return Err(Reason::SymbolMismatch);
        }
        for _ in 0..checked(reader.read_var_u32())? {
            budget.entries(1)?;
            let name = read_name(&mut reader)?;
            let version = read_name(&mut reader)?;
            if matches!(name, "Go" | "Go cmd/compile") && version != "go1.27.1" {
                return Err(Reason::NoSymbols);
            }
            language |= field == "language" && name == "Go";
            compiler |= field == "processed-by" && name == "Go cmd/compile";
        }
    }
    if !reader.eof() {
        return Err(Reason::SymbolMismatch);
    }
    Ok(language && compiler)
}

fn read_name<'a>(reader: &mut BinaryReader<'a>) -> Result<&'a str> {
    let length = u64::from(checked(reader.read_var_u32())?);
    if length > super::bytes::STRING_LIMIT {
        return Err(Reason::SymbolTableTooLarge);
    }
    std::str::from_utf8(checked(reader.read_bytes(size(length)?))?)
        .map_err(|_| Reason::SymbolMismatch)
}

fn bounded_name(name: &str) -> Result<()> {
    if name.len() as u64 > super::bytes::STRING_LIMIT {
        Err(Reason::SymbolTableTooLarge)
    } else {
        Ok(())
    }
}

pub(super) fn returns(
    body: &FunctionBody<'_>,
    select: u64,
    entry: u64,
    budget: &mut Budget,
) -> Result<Option<Vec<u64>>> {
    let mut locals = checked(body.get_locals_reader())?;
    budget.entries(u64::from(locals.get_count()))?;
    budget.steps(add(mul(u64::from(locals.get_count()), 2)?, 1)?)?;
    for _ in 0..locals.get_count() {
        checked(locals.read())?;
    }
    let mut reader = checked(body.get_operators_reader())?;
    budget.bytes(9 * 256)?;
    let mut preceding = std::collections::VecDeque::with_capacity(9);
    let mut found = Vec::new();
    let mut complete = true;
    budget.bytes(16)?;
    let mut control = vec![false];
    while !reader.eof() {
        budget.steps(1)?;
        charge_operator(reader.get_binary_reader(), budget)?;
        let operator = checked(reader.read())?;
        if control.is_empty() {
            return Err(Reason::SymbolMismatch);
        }
        match &operator {
            Operator::Block { .. }
            | Operator::Loop { .. }
            | Operator::If { .. }
            | Operator::Try { .. }
            | Operator::TryTable { .. } => {
                control.push(matches!(operator, Operator::If { .. }));
                if matches!(operator, Operator::Try { .. } | Operator::TryTable { .. }) {
                    complete = false;
                }
            }
            Operator::Else => {
                if !control.last().copied().unwrap_or(false) {
                    return Err(Reason::SymbolMismatch);
                }
                *control.last_mut().ok_or(Reason::SymbolMismatch)? = false;
            }
            Operator::End => {
                control.pop();
            }
            Operator::Delegate { relative_depth } => {
                // The operator reader checks that delegate closes a legacy try.
                control.pop();
                if *relative_depth as usize >= control.len() {
                    return Err(Reason::SymbolMismatch);
                }
                complete = false;
            }
            Operator::Br { relative_depth } | Operator::BrIf { relative_depth } => {
                if *relative_depth as usize >= control.len() {
                    return Err(Reason::SymbolMismatch);
                }
            }
            Operator::BrTable { targets } => {
                if targets.default() as usize >= control.len() {
                    return Err(Reason::SymbolMismatch);
                }
                for target in targets.targets() {
                    budget.steps(1)?;
                    if checked(target)? as usize >= control.len() {
                        return Err(Reason::SymbolMismatch);
                    }
                }
            }
            Operator::Call { function_index } if u64::from(*function_index) == select => {
                if let Some(pc) = stored_return(preceding.make_contiguous(), entry) {
                    budget.entries(1)?;
                    budget.bytes(32)?;
                    found.push(pc);
                } else {
                    complete = false;
                }
            }
            Operator::CallIndirect { .. }
            | Operator::ReturnCall { .. }
            | Operator::ReturnCallIndirect { .. }
            | Operator::CallRef { .. }
            | Operator::ReturnCallRef { .. }
            | Operator::Catch { .. }
            | Operator::CatchAll
            | Operator::Rethrow { .. }
            | Operator::Throw { .. }
            | Operator::ThrowRef
            | Operator::Resume { .. }
            | Operator::ResumeThrow { .. }
            | Operator::ResumeThrowRef { .. }
            | Operator::Suspend { .. }
            | Operator::Switch { .. } => complete = false,
            _ => {}
        }
        if preceding.len() == 9 {
            preceding.pop_front();
        }
        preceding.push_back(operator);
    }
    if !control.is_empty() {
        return Err(Reason::SymbolMismatch);
    }
    Ok((complete && !found.is_empty()).then_some(found))
}

fn charge_operator(mut reader: BinaryReader<'_>, budget: &mut Budget) -> Result<()> {
    let opcode = checked(reader.read_u8())?;
    if matches!(opcode, 0x02 | 0x03 | 0x04 | 0x06 | 0x1f) {
        // Both control stacks grow during decoding; Vec growth costs less than
        // four times the inserted storage, including their initial capacity.
        budget.bytes(8)?;
    }
    let element_bytes = match opcode {
        0x0e => 0, // br_table scans its borrowed targets inside read().
        0x1c => 32,
        0x1f => {
            let mut peek = reader.clone();
            let first = checked(peek.read_u8())?;
            if first == 0x40 {
                checked(reader.read_u8())?;
            } else if first & 0xc0 == 0x40 {
                checked(reader.read::<wasmparser::ValType>())?;
            } else {
                checked(reader.read_var_s33())?;
            }
            16
        }
        0xe3 | 0xe5 => {
            checked(reader.read_var_u32())?;
            16
        }
        0xe4 => {
            checked(reader.read_var_u32())?;
            checked(reader.read_var_u32())?;
            16
        }
        _ => return Ok(()),
    };
    let count = u64::from(checked(reader.read_var_u32())?);
    budget.entries(count)?;
    budget.steps(mul(count, 3)?)?;
    if element_bytes != 0 && count != 0 && !(opcode == 0x1c && count == 1) {
        // Eager TryTable/ResumeTable/select vectors allocate before read()
        // returns. Cover cumulative geometric growth, not just live storage.
        budget.bytes(mul(mul(count.max(4), 4)?, element_bytes)?)?;
    }
    Ok(())
}

fn stored_return(window: &[Operator<'_>], entry: u64) -> Option<u64> {
    // SP's local cache and global stack pointer must both name the concrete return-address slot.
    let [
        Operator::LocalGet { local_index: 1 },
        Operator::I32Const { value: 8 },
        Operator::I32Sub,
        Operator::LocalTee { local_index: 1 },
        Operator::GlobalSet { global_index: 0 },
        Operator::LocalGet { local_index: 1 },
        Operator::I64Const { value },
        Operator::I64Store { memarg },
        Operator::I32Const { value: 0 },
    ] = window
    else {
        return None;
    };
    let pc = u64::try_from(*value).ok()?;
    (memarg.memory == 0
        && memarg.offset == 0
        && memarg.align == 3
        && pc >> 16 == entry >> 16
        && pc > entry)
        .then_some(pc)
}
