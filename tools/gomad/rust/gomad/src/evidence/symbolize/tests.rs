use super::*;

impl std::ops::Index<&Site> for Symbols {
    type Output = Symbol;
    fn index(&self, site: &Site) -> &Self::Output {
        self.get(*site)
    }
}

fn fixture(name: &str) -> Vec<u8> {
    std::fs::read(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../testdata/parity/symbolize")
            .join(name),
    )
    .expect("required pinned-Go fixture; absence must fail")
}

fn site(value: u64) -> Site {
    Site {
        value,
        missing: false,
    }
}

fn leb(mut value: u64, out: &mut Vec<u8>) {
    loop {
        let byte = u8::try_from(value & 127).unwrap();
        value >>= 7;
        out.push(byte | if value == 0 { 0 } else { 128 });
        if value == 0 {
            break;
        }
    }
}

fn signed_leb(mut value: i64, out: &mut Vec<u8>) {
    loop {
        let byte = u8::try_from(value & 127).unwrap();
        value >>= 7;
        let done = (value == 0 && byte & 64 == 0) || (value == -1 && byte & 64 != 0);
        out.push(byte | if done { 0 } else { 128 });
        if done {
            break;
        }
    }
}

fn sections(bytes: &[u8]) -> Vec<(u8, &[u8])> {
    let mut cursor = 8;
    let mut result = Vec::new();
    while cursor < bytes.len() {
        let id = bytes[cursor];
        cursor += 1;
        let mut length = 0;
        let mut shift = 0;
        loop {
            let byte = bytes[cursor];
            cursor += 1;
            length |= usize::from(byte & 127) << shift;
            shift += 7;
            if byte & 128 == 0 {
                break;
            }
        }
        result.push((id, &bytes[cursor..cursor + length]));
        cursor += length;
    }
    result
}

fn replace_section(bytes: &[u8], id: u8, value: &[u8]) -> Vec<u8> {
    let mut result = bytes[..8].to_vec();
    for (section_id, section) in sections(bytes) {
        let data = if section_id == id { value } else { section };
        result.push(section_id);
        leb(data.len() as u64, &mut result);
        result.extend_from_slice(data);
    }
    result
}

fn replace_memory(bytes: &[u8], address: u64, value: &[u8], chunk: usize) -> Vec<u8> {
    let module = wasm::Module::read(bytes, &mut bytes::Budget::default()).unwrap();
    let end = address + value.len() as u64;
    let mut segments = Vec::new();
    for segment in &module.memory.segments {
        for (index, part) in segment.bytes.chunks(chunk).enumerate() {
            let start = segment.start + (index * chunk) as u64;
            let part_end = start + part.len() as u64;
            if part_end <= address || start >= end {
                segments.push((start, part));
                continue;
            }
            if start < address {
                segments.push((start, &part[..usize::try_from(address - start).unwrap()]));
            }
            if part_end > end {
                segments.push((end, &part[usize::try_from(end - start).unwrap()..]));
            }
        }
    }
    if !value.is_empty() {
        segments.push((address, value));
    }
    let mut data = Vec::new();
    leb(segments.len() as u64, &mut data);
    for (address, bytes) in segments {
        data.extend_from_slice(&[0, 0x41]);
        signed_leb(i64::from(u32::try_from(address).unwrap()), &mut data);
        data.push(0x0b);
        leb(bytes.len() as u64, &mut data);
        data.extend_from_slice(bytes);
    }
    replace_section(bytes, 11, &data)
}

fn assert_reason(bytes: &[u8], reason: Reason) {
    let result = resolve_bytes(
        bytes,
        [
            site(5383),
            site(u64::MAX),
            Site {
                value: 0,
                missing: true,
            },
        ],
    );
    assert_eq!(result[&site(5383)], unavailable(reason));
    assert_eq!(result[&site(u64::MAX)], unavailable(reason));
    assert_eq!(
        result[&Site {
            value: 0,
            missing: true
        }],
        unavailable(Reason::SiteMissing)
    );
}

#[test]
fn symbolize_no_binary_metadata_is_unavailable() {
    assert_reason(b"(module)", Reason::NoSymbols);
    assert_reason(b"\0asm\x01\0\0\0\0", Reason::SymbolMismatch);
}

#[test]
fn symbolize_missing_flag_precedes_absent_metadata() {
    let site = Site {
        value: u64::MAX,
        missing: true,
    };
    assert_eq!(
        resolve_bytes(b"\0asm\x01\0\0\0", [site])[&site],
        Symbol::Unavailable {
            unavailable: Reason::SiteMissing
        }
    );
}

#[test]
fn symbolize_real_recorded_inline_select() {
    let bytes = std::fs::read(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../testdata/parity/symbolize/guest.wasm"
    ))
    .expect("required real Go fixture");
    let trace = std::fs::read(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../testdata/parity/symbolize/choices.bin"
    ))
    .expect("required real choice trace");
    let site = Site {
        value: u64::from_be_bytes(trace[24..32].try_into().unwrap()),
        missing: false,
    };
    let results = resolve_bytes(&bytes, [site]);
    let Symbol::Resolved {
        frame,
        inlined_from,
    } = &results[&site]
    else {
        panic!("real recorded select was not resolved: {results:?}")
    };
    assert_eq!((&*frame.package, &*frame.function), ("main", "choose"));
    assert_eq!(
        inlined_from
            .iter()
            .map(|frame| (&*frame.package, &*frame.function))
            .collect::<Vec<_>>(),
        [("main", "positive")]
    );
}

#[test]
fn symbolize_real_corpus_including_conflicting_and_indirect_calls() {
    let expected: serde_json::Value = serde_json::from_slice(&fixture("expected.json")).unwrap();
    let requested: Vec<_> = expected
        .as_array()
        .unwrap()
        .iter()
        .map(|row| site(row["site"].as_u64().unwrap()))
        .collect();
    let result = resolve_bytes(&fixture("guest.wasm"), requested.clone());
    for row in expected.as_array().unwrap() {
        let key = site(row["site"].as_u64().unwrap());
        assert_eq!(
            serde_json::to_value(&result[&key]).unwrap(),
            row["symbol"],
            "{}",
            row["physical"]
        );
    }
    assert_eq!(result[&site(0)], unavailable(Reason::NoSymbols));
    assert_eq!(
        resolve_bytes(&fixture("guest.wasm"), [site(u64::MAX)])[&site(u64::MAX)],
        unavailable(Reason::OffsetOutOfRange)
    );
    let reversed = resolve_bytes(
        &fixture("guest.wasm"),
        requested.iter().rev().chain(requested.iter()).copied(),
    );
    for key in requested {
        assert_eq!(result[&key], reversed[&key]);
    }
}

#[test]
fn symbolize_real_stripped_and_genuinely_absent_tables() {
    for name in ["guest-stripped.wasm", "no-metadata.wasm"] {
        assert_reason(&fixture(name), Reason::NoSymbols);
    }
}

#[test]
fn symbolize_split_tables_and_zero_gaps() {
    let original = fixture("guest.wasm");
    let split = replace_memory(&original, 0, &[], 127);
    let expected = resolve_bytes(&original, [site(5383), site(5385)]);
    let actual = resolve_bytes(&split, [site(5383), site(5385)]);
    for key in [site(5383), site(5385)] {
        assert_eq!(actual[&key], expected[&key]);
    }
}

#[test]
fn symbolize_metadata_inconsistency_precedes_range() {
    let original = fixture("guest.wasm");
    for (address, replacement) in [
        (351_264 + 4, u32::MAX.to_le_bytes().to_vec()),
        (351_264 + 20, u32::MAX.to_le_bytes().to_vec()),
        (934_240 + 176, 1_u64.to_le_bytes().to_vec()),
        (934_240 + 8, 0_u64.to_le_bytes().to_vec()),
        (50784 + 40, 1_u64.to_le_bytes().to_vec()),
    ] {
        assert_reason(
            &replace_memory(&original, address, &replacement, usize::MAX),
            Reason::SymbolMismatch,
        );
    }
}

#[test]
fn symbolize_rejects_ambiguous_moduledata() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let copy: Vec<_> = (0..360)
        .map(|offset| module.memory.byte(934_240 + offset, &mut budget).unwrap())
        .collect();
    assert_reason(
        &replace_memory(&original, module.memory.end - 512, &copy, usize::MAX),
        Reason::SymbolMismatch,
    );
}

#[test]
fn symbolize_name_mismatch_and_malformed_name_sections() {
    let original = fixture("guest.wasm");
    let mut wrong = original.clone();
    let index = wrong
        .windows(b"main.positive".len())
        .rposition(|part| part == b"main.positive")
        .unwrap();
    wrong[index + 5] = b'x';
    assert_reason(&wrong, Reason::SymbolMismatch);
    let mut malformed = original;
    malformed.extend_from_slice(&[0, 7, 4, b'n', b'a', b'm', b'e', 1, 255]);
    assert_reason(&malformed, Reason::SymbolMismatch);
}

#[test]
fn symbolize_module_name_uses_the_fixed_string_limit() {
    let original = fixture("guest.wasm");
    for length in [100_001, 1 << 20, (1 << 20) + 1] {
        let mut name = Vec::new();
        leb(length, &mut name);
        name.resize(name.len() + usize::try_from(length).unwrap(), b'x');
        let mut subsection = vec![0];
        leb(name.len() as u64, &mut subsection);
        subsection.extend(name);
        let mut changed = original[..8].to_vec();
        for (id, data) in sections(&original) {
            let mut payload = data.to_vec();
            if id == 0 && data.starts_with(b"\x04name") {
                payload.splice(5..5, subsection.iter().copied());
            }
            changed.push(id);
            leb(payload.len() as u64, &mut changed);
            changed.extend(payload);
        }
        if length > 1 << 20 {
            assert_reason(&changed, Reason::SymbolTableTooLarge);
        } else {
            assert_eq!(
                resolve_bytes(&changed, [site(5383)])[&site(5383)],
                resolve_bytes(&original, [site(5383)])[&site(5383)],
            );
        }
    }
}

#[test]
fn symbolize_indirect_and_producer_names_use_the_fixed_string_limit() {
    let original = fixture("guest.wasm");
    for id in [2, 3, 10, 255] {
        for length in [100_001, (1 << 20) + 1] {
            let mut name = Vec::new();
            leb(length, &mut name);
            name.resize(name.len() + usize::try_from(length).unwrap(), b'x');
            let mut changed = original[..8].to_vec();
            for (section, data) in sections(&original) {
                let mut payload = data.to_vec();
                if id == 255 && section == 0 && data.starts_with(b"\x09producers") {
                    payload = b"\x09producers\x01\x08language\x01".to_vec();
                    payload.extend(&name);
                    payload.extend_from_slice(b"\x011");
                } else if id != 255 && section == 0 && data.starts_with(b"\x04name") {
                    let mut map = vec![1, 0, 1, 0];
                    map.extend(&name);
                    payload.push(id);
                    leb(map.len() as u64, &mut payload);
                    payload.extend(map);
                }
                changed.push(section);
                leb(payload.len() as u64, &mut changed);
                changed.extend(payload);
            }
            if length > 1 << 20 {
                assert_reason(&changed, Reason::SymbolTableTooLarge);
            } else {
                assert_eq!(
                    resolve_bytes(&changed, [site(5383)])[&site(5383)],
                    resolve_bytes(&original, [site(5383)])[&site(5383)],
                    "name kind {id} length {length}",
                );
            }
        }
    }
}

#[test]
fn symbolize_count_exhaustion_discards_every_result() {
    let original = fixture("guest.wasm");
    for count in [bytes::ENTRY_LIMIT + 1, u64::MAX] {
        assert_reason(
            &replace_memory(&original, 50784 + 8, &count.to_le_bytes(), usize::MAX),
            Reason::SymbolTableTooLarge,
        );
    }
    let result = resolve_bytes(&original, (0..=bytes::ENTRY_LIMIT).map(site));
    assert!(result.entries.is_empty());
    for key in [site(5383), site(5386), site(bytes::ENTRY_LIMIT)] {
        assert_eq!(result[&key], unavailable(Reason::SymbolTableTooLarge));
    }
}

#[test]
fn symbolize_budget_boundaries_and_overflow() {
    for limit in [bytes::BYTE_LIMIT, bytes::ENTRY_LIMIT, bytes::STEP_LIMIT] {
        let mut used = 0;
        bytes::Budget::charge(&mut used, limit, limit).unwrap();
        assert_eq!(
            bytes::Budget::charge(&mut used, 1, limit),
            Err(Reason::SymbolTableTooLarge)
        );
        used = u64::MAX;
        assert_eq!(
            bytes::Budget::charge(&mut used, 1, limit),
            Err(Reason::SymbolTableTooLarge)
        );
    }
    assert_eq!(bytes::add(u64::MAX, 1), Err(Reason::SymbolTableTooLarge));
    assert_eq!(bytes::mul(u64::MAX, 2), Err(Reason::SymbolTableTooLarge));
    let memory = bytes::Memory {
        segments: vec![],
        end: u64::MAX,
    };
    assert_eq!(
        memory.range(0, bytes::BYTE_LIMIT + 1, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    );
    assert_eq!(
        memory.range(u64::MAX, 1, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    );
}

#[test]
fn symbolize_sparse_reads_and_string_limit() {
    let memory = bytes::Memory {
        segments: vec![
            bytes::Segment {
                start: 4,
                bytes: &[1, 2],
            },
            bytes::Segment {
                start: 8,
                bytes: &[3],
            },
        ],
        end: 12,
    };
    assert_eq!(
        memory.uint::<8>(2, &mut bytes::Budget::default()).unwrap(),
        [0, 0, 1, 2, 0, 0, 3, 0]
    );
    let mut name = vec![b'a'; usize::try_from(bytes::STRING_LIMIT).unwrap() + 1];
    name.push(0);
    let memory = bytes::Memory {
        segments: vec![bytes::Segment {
            start: 0,
            bytes: &name,
        }],
        end: name.len() as u64,
    };
    assert_eq!(
        memory.string(0, memory.end, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    );
    name[usize::try_from(bytes::STRING_LIMIT).unwrap()] = 0;
    let memory = bytes::Memory {
        segments: vec![bytes::Segment {
            start: 0,
            bytes: &name,
        }],
        end: name.len() as u64,
    };
    assert_eq!(
        memory
            .string(0, memory.end, &mut bytes::Budget::default())
            .unwrap()
            .len() as u64,
        bytes::STRING_LIMIT
    );
}

#[test]
fn symbolize_exact_go_package_and_receiver_spelling() {
    for (name, package, function) in [
        ("example.com/a.(*T).M", "example.com/a", "(*T).M"),
        (
            "example.com/a.F[other.net/b.Type]",
            "example.com/a",
            "F[other.net/b.Type]",
        ),
        ("go:buildid", "", "go:buildid"),
        ("type:eq.[4]int", "", "type:eq.[4]int"),
    ] {
        assert_eq!(
            tables::split(name),
            Frame {
                package: package.into(),
                function: function.into(),
                file: None,
                line: None
            }
        );
    }
}

#[test]
fn symbolize_rejects_contradictory_optional_producers() {
    let original = fixture("guest.wasm");
    let mut changed = original.clone();
    let index = changed
        .windows(b"go1.27.1".len())
        .rposition(|part| part == b"go1.27.1")
        .unwrap();
    changed[index + 5] = b'6';
    assert_reason(&changed, Reason::NoSymbols);
    let mut absent = original[..8].to_vec();
    for (id, data) in sections(&original) {
        if id == 0 && data.starts_with(b"\x09producers") {
            continue;
        }
        absent.push(id);
        leb(data.len() as u64, &mut absent);
        absent.extend_from_slice(data);
    }
    assert_eq!(
        resolve_bytes(&absent, [site(5383)])[&site(5383)],
        resolve_bytes(&original, [site(5383)])[&site(5383)]
    );
}

#[test]
fn symbolize_inline_cycles_and_outside_gofunc_rows_refuse() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    let function = tables
        .functions
        .iter()
        .find(|function| function.name == "main.positive")
        .unwrap();
    let offset = module
        .memory
        .u32(
            function.address + 44 + u64::from(function.npcdata) * 4 + 12,
            &mut budget,
        )
        .unwrap();
    let row = tables.gofunc + u64::from(offset);
    let cycle = replace_memory(&original, row + 8, &3_u32.to_le_bytes(), usize::MAX);
    assert_eq!(
        resolve_bytes(&cycle, [site(5383)])[&site(5383)],
        unavailable(Reason::SymbolMismatch)
    );
    let outside = replace_memory(
        &original,
        function.address + 44 + u64::from(function.npcdata) * 4 + 12,
        &0_u32.to_le_bytes(),
        usize::MAX,
    );
    assert_eq!(
        resolve_bytes(&outside, [site(5383)])[&site(5383)],
        unavailable(Reason::SymbolMismatch)
    );
}

fn inline_fixture(depth: u32) -> (Vec<u8>, tables::Tables, tables::Function) {
    let mut data = vec![0; 16384];
    data[512..523].copy_from_slice(b"main.inner\0");
    let mut stream = Vec::new();
    for _ in 0..depth {
        leb(2, &mut stream);
        leb(1, &mut stream);
    }
    leb(u64::from(depth) * 2 - 1, &mut stream);
    leb(1, &mut stream);
    stream.push(0);
    data[1025..1025 + stream.len()].copy_from_slice(&stream);
    data[256 + 52..256 + 56].copy_from_slice(&1_u32.to_le_bytes());
    data[256 + 68..256 + 72].copy_from_slice(&0_u32.to_le_bytes());
    for index in 0..depth {
        let row = 4096 + usize::try_from(index).unwrap() * 16;
        data[row + 8..row + 12].copy_from_slice(&(index + 1).to_le_bytes());
    }
    let tables = tables::Tables {
        text: 0,
        gofunc: 4096,
        gofunc_end: data.len() as u64,
        names: tables::Table {
            start: 512,
            len: 11,
        },
        cu: tables::Table { start: 0, len: 0 },
        files: tables::Table { start: 0, len: 0 },
        pc: tables::Table {
            start: 1024,
            len: stream.len() as u64 + 1,
        },
        functions: vec![],
        select: None,
    };
    let function = tables::Function {
        offset: 0x1000,
        end: 0x1001,
        address: 256,
        index: 0,
        name: "main.outer".into(),
        pcfile: 0,
        pcline: 0,
        npcdata: 3,
        cu: 0,
        nfuncdata: 4,
    };
    (data, tables, function)
}

#[test]
fn symbolize_inline_depth_boundary_and_cumulative_traversal_budget() {
    for depth in [255, 256] {
        let (data, tables, function) = inline_fixture(depth);
        let memory = bytes::Memory {
            segments: vec![bytes::Segment {
                start: 0,
                bytes: &data,
            }],
            end: data.len() as u64,
        };
        let result = tables.chain(&memory, &function, 0, &mut bytes::Budget::default());
        if depth == 255 {
            assert_eq!(result.unwrap().len(), 256);
        } else {
            assert_eq!(result.unwrap_err(), Reason::SymbolTableTooLarge);
        }
        let mut budget = bytes::Budget {
            steps: bytes::STEP_LIMIT - 1,
            ..bytes::Budget::default()
        };
        assert_eq!(
            tables
                .chain(&memory, &function, 0, &mut budget)
                .unwrap_err(),
            Reason::SymbolTableTooLarge
        );
    }
}

#[test]
fn symbolize_inline_rows_cannot_read_past_metadata_region() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    let function = tables
        .functions
        .iter()
        .find(|function| function.name == "main.positive")
        .unwrap();
    let offset = module
        .memory
        .u32(
            function.address + 44 + u64::from(function.npcdata) * 4 + 12,
            &mut budget,
        )
        .unwrap();
    let row = tables.gofunc + u64::from(offset);
    let shortened = replace_memory(&original, 934_240 + 352, &row.to_le_bytes(), usize::MAX);
    assert_eq!(
        resolve_bytes(&shortened, [site(5383)])[&site(5383)],
        unavailable(Reason::SymbolMismatch)
    );
}

#[test]
fn symbolize_corrupt_pc_stream_and_unsupported_select_grammar() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    let function = tables
        .functions
        .iter()
        .find(|function| function.name == "main.positive")
        .unwrap();
    let malformed = replace_memory(
        &original,
        tables.pc.start + u64::from(function.pcfile),
        &[0x80, 0x80, 0x80, 0x80, 0x10],
        usize::MAX,
    );
    assert_eq!(
        resolve_bytes(&malformed, [site(5383)])[&site(5383)],
        unavailable(Reason::SymbolTableTooLarge)
    );
    let mut absent = original.clone();
    let body = module.body(function.index).unwrap().range();
    let mut needle = vec![0x42];
    signed_leb(352_780_292, &mut needle);
    needle.extend_from_slice(&[0x37, 3, 0]);
    let index = absent[body.clone()]
        .windows(needle.len())
        .position(|part| part == needle)
        .unwrap()
        + body.start
        + needle.len()
        - 3;
    // Changing one store alignment keeps the byte grammar decodable but prevents the exact call proof.
    absent[index + 1] = 2;
    let actual = resolve_bytes(&absent, [site(5383)]);
    assert!(
        matches!(&actual[&site(5383)], Symbol::Resolved { frame: Frame { function, .. }, inlined_from } if function == "positive" && inlined_from.is_empty())
    );
}

#[test]
fn symbolize_complete_candidates_include_dead_branches_and_reject_unknown_stores() {
    use gomad_wasmhost::wasmparser::{BinaryReader, FunctionBody};
    let entry = 0x1000_0000_u64;
    let mut body = vec![0, 0x23, 0, 0x21, 1, 0x41, 0, 0x04, 0x40];
    let mut call = vec![0x20, 1, 0x41, 8, 0x6b, 0x22, 1, 0x24, 0, 0x20, 1, 0x42];
    signed_leb(i64::try_from(entry + 4).unwrap(), &mut call);
    call.extend_from_slice(&[0x37, 3, 0, 0x41, 0, 0x10, 7]);
    body.extend_from_slice(&call);
    body.extend_from_slice(&[0x0b, 0x0b]);
    let function = FunctionBody::new(BinaryReader::new(&body, 0));
    assert_eq!(
        wasm::returns(&function, 7, entry, &mut bytes::Budget::default()).unwrap(),
        Some(vec![entry + 4])
    );
    let store = body
        .windows(3)
        .position(|part| part == [0x37, 3, 0])
        .unwrap();
    body[store + 2] = 8;
    let function = FunctionBody::new(BinaryReader::new(&body, 0));
    assert_eq!(
        wasm::returns(&function, 7, entry, &mut bytes::Budget::default()).unwrap(),
        None
    );
}

#[test]
fn symbolize_malformed_unused_name_map_is_not_ignored() {
    let mut original = fixture("guest.wasm");
    let (offset, length) = {
        let reader = gomad_wasmhost::wasmparser::Parser::new(0)
            .parse_all(&original)
            .find_map(|payload| match payload.unwrap() {
                gomad_wasmhost::wasmparser::Payload::CustomSection(section)
                    if section.name() == "name" =>
                {
                    Some(section)
                }
                _ => None,
            })
            .unwrap();
        (reader.data_offset(), reader.data().len())
    };
    // Turn the function map into a malformed type-name map without altering the outer section bounds.
    original[offset] = 4;
    original[offset + length - 1] = 255;
    assert_reason(&original, Reason::SymbolMismatch);
}

struct Cleanup(std::path::PathBuf);

impl Drop for Cleanup {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}

fn retained_artifact(module: &[u8]) -> (Cleanup, crate::evidence::store::Manifest) {
    use crate::evidence::{
        canonical,
        identity::Identity,
        record::{Limits, Outcome, Record},
        store::{FileEntry, Manifest},
    };
    use std::io::Write;
    use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt};
    let root = std::env::temp_dir().join(format!("gomad-symbolize-{}", std::process::id()));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(root.join("target"))
        .unwrap();
    let digest = canonical::digest(&[]);
    let identity = Identity {
        target: canonical::digest(module),
        toolchain: digest.clone(),
        platform: digest.clone(),
        implementation: digest.clone(),
        configuration: digest,
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest().unwrap(),
        identity,
        profile: "gomad.wasi-cooperative/v1".into(),
        seed: 7,
        arguments: vec![],
        environment: BTreeMap::new(),
        limits: Limits {
            fuel: 1,
            memory_bytes: 1,
            wall_millis: 1,
            output_bytes: 1,
            evidence_bytes: 1,
        },
        virtual_time_nanos: 0,
        outcome: Outcome {
            class: "success".into(),
            termination: "exit".into(),
            exit_code: Some(0),
            reason: String::new(),
        },
        streams: BTreeMap::new(),
    };
    let manifest = Manifest {
        schema: "gomad.artifact-manifest/v2".into(),
        record_sha256: record.hash().unwrap(),
        record,
        files: BTreeMap::from([(
            "target/module.wasm".into(),
            FileEntry {
                bytes: module.len() as u64,
                sha256: canonical::digest(module),
                module: true,
            },
        )]),
    };
    for (name, bytes) in [
        (
            "storage.json",
            b"{\"format\":\"gomad.artifact-storage/v2\"}".to_vec(),
        ),
        ("manifest.json", canonical::encode(&manifest).unwrap()),
        ("target/module.wasm", module.to_vec()),
    ] {
        std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(root.join(name))
            .unwrap()
            .write_all(&bytes)
            .unwrap();
    }
    (Cleanup(root), manifest)
}

#[test]
fn symbolize_authenticates_retained_handle_before_missing_site_and_preserves_identity() {
    let module = fixture("guest.wasm");
    let (cleanup, manifest) = retained_artifact(&module);
    let root = &cleanup.0;
    let record = manifest.record.clone();
    let opened = crate::evidence::store::Opened::open(root).unwrap();
    assert!(matches!(
        resolve(&opened, [site(5383)]).unwrap()[&site(5383)],
        Symbol::Resolved { .. }
    ));
    assert_eq!(opened.manifest(), &manifest);
    assert_eq!(record.hash().unwrap(), manifest.record_sha256);
    assert_eq!(
        record.failure_signature().unwrap(),
        manifest.record.failure_signature().unwrap()
    );
    assert_eq!(
        std::fs::read(root.join("target/module.wasm")).unwrap(),
        module
    );
    let mut changed = module;
    changed[0] = 1;
    std::fs::write(root.join("target/module.wasm"), changed).unwrap();
    let missing = Site {
        value: 0,
        missing: true,
    };
    assert_eq!(
        resolve(&opened, [missing]).unwrap_err(),
        "artifact payload identity mismatch"
    );
}

#[test]
fn symbolize_validates_pc_and_inline_tables_before_classifying_range() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    let function = tables
        .functions
        .iter()
        .find(|function| function.name == "main.positive")
        .unwrap();
    let offset = module
        .memory
        .u32(
            function.address + 44 + u64::from(function.npcdata) * 4 + 12,
            &mut budget,
        )
        .unwrap();
    for changed in [
        replace_memory(
            &original,
            tables.pc.start + u64::from(function.pcfile),
            &[2, 0],
            usize::MAX,
        ),
        replace_memory(
            &original,
            tables.gofunc + u64::from(offset) + 8,
            &3_u32.to_le_bytes(),
            usize::MAX,
        ),
        replace_memory(&original, 50784 + 16, &0_u64.to_le_bytes(), usize::MAX),
    ] {
        assert_eq!(
            resolve_bytes(&changed, [site(u64::MAX)])[&site(u64::MAX)],
            unavailable(Reason::SymbolMismatch)
        );
    }
}

#[test]
fn symbolize_function_scale_gap_never_attributes_the_previous_function() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let count = module.memory.u64(50784 + 8, &mut budget).unwrap();
    let table = module.memory.u64(934_240 + 128, &mut budget).unwrap();
    let mut entries: Vec<_> = (0..(count + 1) * 8)
        .map(|offset| module.memory.byte(table + offset, &mut budget).unwrap())
        .collect();
    let omitted = (5384 - 4096) * 8;
    entries.drain(omitted..omitted + 8);
    let mut changed = replace_memory(&original, table, &entries, usize::MAX);
    for (address, count) in [
        (50784 + 8, count - 1),
        (934_240 + 136, count),
        (934_240 + 144, count),
    ] {
        changed = replace_memory(&changed, address, &count.to_le_bytes(), usize::MAX);
    }
    assert_eq!(
        resolve_bytes(&changed, [site(5384)])[&site(5384)],
        unavailable(Reason::OffsetOutOfRange)
    );
}

#[test]
fn symbolize_budget_failure_after_a_resolved_site_replaces_the_entire_map() {
    let original = fixture("guest.wasm");
    let module = wasm::Module::read(&original, &mut bytes::Budget::default()).unwrap();
    let mut code = Vec::new();
    leb(module.functions.len() as u64, &mut code);
    for (ordinal, body) in module.functions.iter().enumerate() {
        if ordinal == 5383 - 4096 {
            let count = usize::try_from(bytes::STEP_LIMIT + 1).unwrap();
            leb((count + 4) as u64, &mut code);
            code.push(0);
            code.resize(code.len() + count, 1);
            code.extend_from_slice(&[0x41, 0, 0x0b]);
        } else {
            let body = &original[body.range()];
            leb(body.len() as u64, &mut code);
            code.extend_from_slice(body);
        }
    }
    let changed = replace_section(&original, 10, &code);
    let first = site(4096);
    let last = site(5383);
    assert!(matches!(
        resolve_bytes(&changed, [first])[&first],
        Symbol::Resolved { .. }
    ));
    let results = resolve_bytes(&changed, [last, first]);
    assert!(results.entries.is_empty());
    assert_eq!(results[&first], unavailable(Reason::SymbolTableTooLarge));
    assert_eq!(results[&last], unavailable(Reason::SymbolTableTooLarge));
    assert_eq!(
        results[&Site {
            value: 0,
            missing: true
        }],
        unavailable(Reason::SiteMissing)
    );
}

#[test]
fn symbolize_counts_function_imports_instead_of_all_imports() {
    let bytes = [
        0, 97, 115, 109, 1, 0, 0, 0, 1, 4, 1, 0x60, 0, 0, 2, 14, 2, 1, b'h', 1, b'g', 3, 0x7f, 0,
        1, b'h', 1, b'f', 0, 0, 3, 2, 1, 0, 10, 4, 1, 2, 0, 0x0b,
    ];
    let module = wasm::Module::read(&bytes, &mut bytes::Budget::default()).unwrap();
    assert_eq!(module.imported, 1);
    assert_eq!(module.functions.len(), 1);
    assert!(module.body(1).is_ok());
    assert!(matches!(module.body(2), Err(Reason::SymbolMismatch)));
}

#[test]
fn symbolize_records_small_fixture_analysis_budget() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    for site in 5383..=5387 {
        resolve_site(&module, &tables, site, &mut budget).unwrap();
    }
    println!(
        "small real fixture parse + five site lookups: bytes={} code={} entries={} steps={}; request map reservation excluded",
        budget.bytes, budget.code, budget.entries, budget.steps
    );
}

#[test]
fn symbolize_fixture_identity_and_recorded_sites_match_retained_proof() {
    let proof: serde_json::Value = serde_json::from_slice(&fixture("proof.json")).unwrap();
    for name in [
        "guest.wasm",
        "guest-stripped.wasm",
        "choices.bin",
        "guest.go.txt",
    ] {
        assert_eq!(
            crate::evidence::canonical::digest(&fixture(name)),
            proof["digests"][name].as_str().unwrap()
        );
    }
    let trace = fixture("choices.bin");
    let sites: Vec<_> = trace
        .as_chunks::<96>()
        .0
        .iter()
        .filter(|row| row[8] == 2)
        .map(|row| u64::from_be_bytes(row[24..32].try_into().unwrap()))
        .collect();
    assert_eq!(sites, [5383, 5384, 5384, 5385, 5385, 5386, 5387, 5386]);
}

#[test]
fn symbolize_unreferenced_magic_is_not_a_second_metadata_interpretation() {
    let original = fixture("guest.wasm");
    let module = wasm::Module::read(&original, &mut bytes::Budget::default()).unwrap();
    let mut unrelated = [0; 72];
    unrelated[..8].copy_from_slice(&[0xf1, 0xff, 0xff, 0xff, 0, 0, 1, 8]);
    let changed = replace_memory(&original, module.memory.end - 416, &unrelated, usize::MAX);
    assert_eq!(
        resolve_bytes(&changed, [site(5383)])[&site(5383)],
        resolve_bytes(&original, [site(5383)])[&site(5383)]
    );
}

#[test]
fn symbolize_pointed_decoy_is_not_authoritative_metadata() {
    let original = fixture("guest.wasm");
    let module = wasm::Module::read(&original, &mut bytes::Budget::default()).unwrap();
    let header = module.memory.end - 416;
    for count in [0, u64::MAX] {
        let mut decoy = [0; 72];
        decoy[..8].copy_from_slice(&[0xf1, 0xff, 0xff, 0xff, 0, 0, 1, 8]);
        decoy[8..16].copy_from_slice(&count.to_le_bytes());
        decoy[32..40].copy_from_slice(&72_u64.to_le_bytes());
        let changed = replace_memory(&original, header, &decoy, usize::MAX);
        let changed = replace_memory(
            &changed,
            module.memory.end - 512,
            &header.to_le_bytes(),
            usize::MAX,
        );
        let actual = resolve_bytes(&changed, [site(5383)]);
        assert!(
            matches!(&actual[&site(5383)], Symbol::Resolved { frame: Frame { function, line: Some(4), .. }, .. } if function == "choose")
        );
    }
}

#[test]
fn symbolize_balanced_unsupported_control_falls_back_physically() {
    use gomad_wasmhost::wasmparser::{BinaryReader, FunctionBody};
    // Hand-encoded balanced exception-control bodies; none has a supported inline proof.
    for body in [
        vec![0, 0x1f, 0x40, 0, 0x0b, 0x0b], // try_table [] ... end; function end
        vec![0, 0x06, 0x40, 0x0b, 0x0b],    // legacy try ... end
        vec![0, 0x06, 0x40, 0x07, 0, 0x0b, 0x0b], // try ... catch tag0
        vec![0, 0x06, 0x40, 0x19, 0x0b, 0x0b], // try ... catch_all
        vec![0, 0x06, 0x40, 0x18, 0, 0x0b], // try ... delegate0; function end
    ] {
        let function = FunctionBody::new(BinaryReader::new(&body, 0));
        assert_eq!(
            wasm::returns(&function, 7, 0x1000_0000, &mut bytes::Budget::default()),
            Ok(None),
            "{body:02x?}"
        );
    }
}

#[test]
fn symbolize_try_table_preserves_the_physical_site() {
    let original = fixture("guest.wasm");
    let module = wasm::Module::read(&original, &mut bytes::Budget::default()).unwrap();
    let mut code = Vec::new();
    leb(module.functions.len() as u64, &mut code);
    for (ordinal, body) in module.functions.iter().enumerate() {
        let mut bytes = original[body.range()].to_vec();
        if ordinal == 1287 {
            let operators = body.get_operators_reader().unwrap();
            let start = operators.original_position() - body.range().start;
            bytes.splice(start..start, [0x1f, 0x40, 0, 0x0b]);
        }
        leb(bytes.len() as u64, &mut code);
        code.extend_from_slice(&bytes);
    }
    let changed = replace_section(&original, 10, &code);
    let actual = resolve_bytes(&changed, [site(5383)]);
    assert!(
        matches!(&actual[&site(5383)], Symbol::Resolved { frame: Frame { function, file: None, line: None, .. }, inlined_from } if function == "positive" && inlined_from.is_empty())
    );
}

#[test]
fn symbolize_bounds_cumulative_parser_catch_vectors_across_sites() {
    let original = fixture("guest.wasm");
    let mut budget = bytes::Budget::default();
    let module = wasm::Module::read(&original, &mut budget).unwrap();
    let tables = tables::Tables::read(&module, &mut budget).unwrap();
    let indices: Vec<_> = tables
        .functions
        .iter()
        .filter(|function| (5383..=5387).contains(&function.offset))
        .map(|function| function.index - module.imported)
        .collect();
    assert_eq!(indices.len(), 5);
    let mut code = Vec::new();
    leb(module.functions.len() as u64, &mut code);
    let mut catch_entries = 0;
    for (ordinal, body) in module.functions.iter().enumerate() {
        let mut encoded = original[body.range()].to_vec();
        if indices.contains(&(ordinal as u64)) {
            let start =
                body.get_operators_reader().unwrap().original_position() - body.range().start;
            encoded.truncate(start);
            encoded.extend_from_slice(&[0x02, 0x40]);
            for _ in 0..360 {
                encoded.extend_from_slice(&[0x1f, 0x40]);
                leb(10_000, &mut encoded);
                for _ in 0..10_000 {
                    encoded.extend_from_slice(&[0x02, 0]);
                    catch_entries += 1;
                }
                encoded.push(0x0b);
            }
            encoded.extend_from_slice(&[0x0b, 0x0b]);
            assert!(encoded.len() < 7_654_321);
        }
        leb(encoded.len() as u64, &mut code);
        code.extend_from_slice(&encoded);
    }
    assert_eq!(catch_entries, 18_000_000);
    assert!(code.len() as u64 <= bytes::BYTE_LIMIT);
    let changed = replace_section(&original, 10, &code);
    let requested: Vec<_> = (5383..=5387).map(site).collect();
    let actual = resolve_bytes(&changed, requested.iter().copied());
    for key in requested {
        assert_eq!(actual[&key], unavailable(Reason::SymbolTableTooLarge));
    }
    assert!(actual.entries.is_empty());
    assert_eq!(
        actual[&Site {
            value: 0,
            missing: true
        }],
        unavailable(Reason::SiteMissing)
    );
}

#[test]
fn symbolize_precharges_variable_operators_before_decoder_work() {
    use gomad_wasmhost::wasmparser::{BinaryReader, FunctionBody};
    for (name, header) in [
        ("try_table", vec![0x1f, 0x40, 2]),
        ("typed_select", vec![0x1c, 2]),
        ("br_table", vec![0x0e, 2]),
        ("resume", vec![0xe3, 0, 2]),
        ("resume_throw", vec![0xe4, 0, 0, 2]),
        ("resume_throw_ref", vec![0xe5, 0, 2]),
    ] {
        let mut encoded = vec![0];
        encoded.extend_from_slice(&header);
        let function = FunctionBody::new(BinaryReader::new(&encoded, 0));
        // Missing vector contents distinguish pre-work refusal from a decoder error.
        for mut budget in [
            bytes::Budget {
                entries: bytes::ENTRY_LIMIT - 1,
                ..Default::default()
            },
            bytes::Budget {
                steps: bytes::STEP_LIMIT - 4,
                ..Default::default()
            },
            bytes::Budget {
                bytes: bytes::BYTE_LIMIT - (9 * 256 + 16 + 8),
                ..Default::default()
            },
        ] {
            // br_table borrows its contents and never allocates a vector.
            if name == "br_table" && budget.bytes != 0 {
                continue;
            }
            assert_eq!(
                wasm::returns(&function, 7, 0x1000_0000, &mut budget),
                Err(Reason::SymbolTableTooLarge),
                "{name}"
            );
        }
    }
}

#[test]
fn symbolize_parser_vector_budget_boundaries_and_block_types() {
    use gomad_wasmhost::wasmparser::{BinaryReader, FunctionBody};
    for encoded in [
        vec![0, 0x1f, 0x40, 2, 2, 0, 2, 0, 0x0b, 0x0b],
        vec![0, 0x1f, 0x7f, 2, 2, 0, 2, 0, 0x0b, 0x0b],
        vec![0, 0x1f, 0x63, 0x70, 2, 2, 0, 2, 0, 0x0b, 0x0b],
        vec![0, 0x1f, 0x80, 0, 2, 2, 0, 2, 0, 0x0b, 0x0b],
        vec![0, 0x1c, 2, 0x7f, 0x7e, 0x0b],
        vec![0, 0x0e, 2, 0, 0, 0, 0x0b],
        vec![0, 0xe3, 0, 2, 1, 0, 1, 0, 0x0b],
        vec![0, 0xe4, 0, 0, 2, 1, 0, 1, 0, 0x0b],
        vec![0, 0xe5, 0, 2, 1, 0, 1, 0, 0x0b],
    ] {
        let function = FunctionBody::new(BinaryReader::new(&encoded, 0));
        let mut used = bytes::Budget::default();
        assert_eq!(wasm::returns(&function, 7, 0, &mut used), Ok(None));
        assert_eq!(used.entries, 2);
        let mut exact = bytes::Budget {
            bytes: bytes::BYTE_LIMIT - used.bytes,
            entries: bytes::ENTRY_LIMIT - used.entries,
            steps: bytes::STEP_LIMIT - used.steps,
            ..Default::default()
        };
        assert_eq!(wasm::returns(&function, 7, 0, &mut exact), Ok(None));
        assert_eq!(exact.bytes, bytes::BYTE_LIMIT);
        assert_eq!(exact.entries, bytes::ENTRY_LIMIT);
        assert_eq!(exact.steps, bytes::STEP_LIMIT);
        for (bytes, entries, steps) in [(1, 0, 0), (0, 1, 0), (0, 0, 1)] {
            let mut over = bytes::Budget {
                bytes: bytes::BYTE_LIMIT - used.bytes + bytes,
                entries: bytes::ENTRY_LIMIT - used.entries + entries,
                steps: bytes::STEP_LIMIT - used.steps + steps,
                ..Default::default()
            };
            assert_eq!(
                wasm::returns(&function, 7, 0, &mut over),
                Err(Reason::SymbolTableTooLarge),
                "{encoded:02x?}"
            );
        }
    }
}

#[test]
fn symbolize_bounds_lazy_data_expression_work_before_parsing() {
    let mut data = vec![1, 0, 0x41, 0]; // one active segment, i32.const0
    data.resize(usize::try_from(bytes::STEP_LIMIT).unwrap() + 8, 0x01); // over-limit nop expression
    data.extend_from_slice(&[0x0b, 0]);
    let mut module = b"\0asm\x01\0\0\0".to_vec();
    module.push(11);
    leb(data.len() as u64, &mut module);
    module.extend_from_slice(&data);
    assert!(matches!(
        wasm::Module::read(&module, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    ));
}

#[test]
fn symbolize_bounds_nested_data_before_parser_control_allocation() {
    let mut data = vec![1, 0];
    for _ in 0..128 {
        data.extend_from_slice(&[0x02, 0x40]); // malformed constant expression with nested blocks
    }
    data.extend_from_slice(&[0x0b, 0]);
    let mut module = b"\0asm\x01\0\0\0".to_vec();
    module.push(11);
    leb(data.len() as u64, &mut module);
    module.extend_from_slice(&data);
    let mut budget = bytes::Budget {
        steps: bytes::STEP_LIMIT - 32,
        ..Default::default()
    };
    assert!(matches!(
        wasm::Module::read(&module, &mut budget),
        Err(Reason::SymbolTableTooLarge)
    ));
}

#[test]
fn symbolize_integer_reads_charge_one_lookup_for_contiguous_data() {
    let segments = (0..4096)
        .map(|index| bytes::Segment {
            start: index * 16,
            bytes: &[1; 8],
        })
        .collect();
    let memory = bytes::Memory {
        segments,
        end: 65536,
    };
    let mut comparisons = 0;
    let _ = memory.segments.partition_point(|segment| {
        comparisons += 1;
        segment.start <= 32768
    });
    let initial = bytes::STEP_LIMIT - comparisons - 8;
    let mut budget = bytes::Budget {
        steps: initial,
        entries: bytes::ENTRY_LIMIT,
        ..Default::default()
    };
    assert_eq!(memory.uint::<8>(32768, &mut budget).unwrap(), [1; 8]);
    assert_eq!(budget.bytes, 8);
    assert_eq!(budget.steps, bytes::STEP_LIMIT);
    assert_eq!(budget.entries, bytes::ENTRY_LIMIT);
    let mut exhausted = bytes::Budget {
        steps: initial + 1,
        ..Default::default()
    };
    assert_eq!(
        memory.uint::<8>(32768, &mut exhausted),
        Err(Reason::SymbolTableTooLarge)
    );
}

#[test]
fn symbolize_integer_reads_span_segments_gaps_and_preserve_boundaries() {
    let memory = bytes::Memory {
        segments: vec![
            bytes::Segment {
                start: 2,
                bytes: &[1, 2],
            },
            bytes::Segment {
                start: 4,
                bytes: &[3, 4],
            },
            bytes::Segment {
                start: 8,
                bytes: &[5],
            },
            bytes::Segment {
                start: 11,
                bytes: &[6],
            },
        ],
        end: 12,
    };
    let mut budget = bytes::Budget::default();
    assert_eq!(
        memory.uint::<12>(0, &mut budget).unwrap(),
        [0, 0, 1, 2, 3, 4, 0, 0, 5, 0, 0, 6]
    );
    assert_eq!(budget.bytes, 12);
    assert_eq!(
        memory.u32(2, &mut bytes::Budget::default()).unwrap(),
        0x0403_0201
    );
    assert_eq!(
        memory.uint::<1>(11, &mut bytes::Budget::default()).unwrap(),
        [6]
    );
    for address in [11, 12, 13] {
        assert_eq!(
            memory.uint::<2>(address, &mut bytes::Budget::default()),
            Err(Reason::SymbolMismatch)
        );
    }
    let framing = bytes::Memory {
        segments: memory.segments.clone(),
        end: 20,
    };
    assert_eq!(
        framing
            .uint::<16>(1, &mut bytes::Budget::default())
            .unwrap(),
        [0, 1, 2, 3, 4, 0, 0, 5, 0, 0, 6, 0, 0, 0, 0, 0]
    );
    let mut zero = bytes::Budget::default();
    assert_eq!(memory.uint::<0>(u64::MAX, &mut zero).unwrap(), [0_u8; 0]);
    assert_eq!((zero.bytes, zero.steps), (0, 0));
    assert_eq!(
        memory.uint::<1>(u64::MAX, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    );
    assert_eq!(
        memory.uint::<8>(u64::MAX - 2, &mut bytes::Budget::default()),
        Err(Reason::SymbolMismatch)
    );
    let enormous = bytes::Memory {
        segments: vec![],
        end: u64::MAX,
    };
    assert_eq!(
        enormous.uint::<4>(u64::MAX - 2, &mut bytes::Budget::default()),
        Err(Reason::SymbolTableTooLarge)
    );
    let mut bytes_exhausted = bytes::Budget {
        bytes: bytes::BYTE_LIMIT - 3,
        ..Default::default()
    };
    assert_eq!(
        memory.uint::<4>(2, &mut bytes_exhausted),
        Err(Reason::SymbolTableTooLarge)
    );
}
