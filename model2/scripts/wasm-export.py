"""Normalize the oracle's export name without changing executable sections."""

import os
import pathlib
import shlex
import sys
import tempfile


def encode_u32(value):
    result = bytearray()
    while value >= 128:
        result.append((value & 127) | 128)
        value >>= 7
    result.append(value)
    return bytes(result)


def normalize(data):
    if data[:8] != b"\0asm\x01\0\0\0":
        raise ValueError("expected a version-one core wasm module")
    position = 8
    sections = []
    changed = False
    while position < len(data):
        start = position
        kind = data[position]
        size, payload_start = decode_u32(data, position + 1)
        end = payload_start + size
        if end > len(data):
            raise ValueError("truncated wasm section")
        if kind == 7:
            if changed:
                raise ValueError("duplicate export section")
            payload = normalize_exports(data[payload_start:end])
            sections.append(bytes([kind]) + encode_u32(len(payload)) + payload)
            changed = True
        else:
            sections.append(data[start:end])
        position = end
    if not changed:
        raise ValueError("missing guest export section")
    return data[:8] + b"".join(sections)


def decode_u32(data, position):
    value = 0
    for shift in range(0, 35, 7):
        if position >= len(data):
            raise ValueError("truncated wasm u32")
        byte = data[position]
        position += 1
        if shift == 28 and byte > 15:
            raise ValueError("overflowing wasm u32")
        value |= (byte & 127) << shift
        if not byte & 128:
            return value, position
    raise ValueError("unterminated wasm u32")


def normalize_exports(payload):
    count, position = decode_u32(payload, 0)
    result = [payload[:position]]
    names = set()
    renamed = 0
    for _ in range(count):
        start = position
        length, name_start = decode_u32(payload, position)
        name_end = name_start + length
        if name_end >= len(payload):
            raise ValueError("truncated wasm export")
        name = payload[name_start:name_end].decode("utf-8")
        if name in names or name == "free":
            raise ValueError("duplicate or conflicting guest export")
        names.add(name)
        kind = payload[name_end]
        if kind > 4:
            raise ValueError("invalid wasm export kind")
        _, position = decode_u32(payload, name_end + 1)
        if name == "umpire_free":
            if kind != 0:
                raise ValueError("umpire_free must export a function")
            result.append(encode_u32(4) + b"free" + payload[name_end:position])
            renamed += 1
        else:
            result.append(payload[start:position])
    if position != len(payload) or renamed != 1:
        raise ValueError("expected exactly one umpire_free function export")
    return b"".join(result)


def linker_output(arguments, depth=0):
    if depth > 8:
        raise ValueError("linker response-file nesting is too deep")
    expanded = []
    for argument in arguments:
        if argument.startswith("@"):
            expanded.extend(shlex.split(pathlib.Path(argument[1:]).read_text()))
        else:
            expanded.append(argument)
    if any(argument.startswith("@") for argument in expanded):
        return linker_output(expanded, depth + 1)
    for index, argument in enumerate(expanded):
        if argument == "-o" and index + 1 < len(expanded):
            return pathlib.Path(expanded[index + 1])
    raise ValueError("missing linker output path")


def main():
    output = linker_output(sys.argv[1:])
    # Other packages using this WASI linker retain their original export tables.
    if output.name != "umpire_wasm.wasm":
        return
    original = output.read_bytes()
    normalized = normalize(original)
    with tempfile.NamedTemporaryFile(dir=output.parent, delete=False) as staged:
        temporary = pathlib.Path(staged.name)
        try:
            staged.write(normalized)
            staged.flush()
            os.replace(temporary, output)
        finally:
            temporary.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
