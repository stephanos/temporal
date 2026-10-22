#!/usr/bin/env python3
"""Reproducible Git-tree size inventory; code lines are a physical lexical metric."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

VERSION = 'gomad-source-size/v1'
LANGUAGES = {'.go': 'Go', '.rs': 'Rust', '.py': 'Python', '.sh': 'Shell',
             '.bash': 'Shell', '.c': 'C', '.h': 'C', '.cc': 'C++', '.cpp': 'C++',
             '.js': 'JavaScript', '.ts': 'TypeScript', '.tsx': 'TypeScript',
             '.sql': 'SQL', '.yaml': 'YAML', '.yml': 'YAML', '.toml': 'TOML',
             '.proto': 'Protobuf', '.css': 'CSS'}


def language(path, raw):
    name = path.removesuffix('.txt').removesuffix('.tmpl')
    if Path(name).name == 'Makefile':
        return 'Make'
    if Path(name).suffix in LANGUAGES:
        return LANGUAGES[Path(name).suffix]
    if raw.startswith(b'#!'):
        first = raw.splitlines()[0]
        if b'python' in first:
            return 'Python'
        if b'sh' in first:
            return 'Shell'
    return None


def code_lines(text, lang):
    """Count lines containing tokens outside comments; literal content is code.

    Nested Rust block comments and raw literals are recognized. Shell/Make/YAML/
    TOML use hash comments, Python supports triple strings, SQL uses -- comments.
    This is a physical-line counter, not a compiler parser or cloc result.
    """
    hash_comments = lang in {'Python', 'Shell', 'Make', 'YAML', 'TOML'}
    slash_comments = lang in {'Go', 'Rust', 'C', 'C++', 'JavaScript', 'TypeScript', 'Protobuf', 'CSS'}
    block, literal, raw_literal, count = 0, None, False, 0
    for line in text.splitlines():
        i, has_code = 0, False
        while i < len(line):
            if block:
                if lang == 'Rust' and line.startswith('/*', i):
                    block += 1
                    i += 2
                elif line.startswith('*/', i):
                    block -= 1
                    i += 2
                else:
                    i += 1
                continue
            if literal:
                has_code = has_code or bool(line.strip())
                if line.startswith(literal, i):
                    i += len(literal)
                    literal = None
                    raw_literal = False
                elif line[i] == '\\' and literal not in {'`'} and not raw_literal:
                    i += 2
                else:
                    i += 1
                continue
            if line[i].isspace():
                i += 1
            elif (hash_comments and line[i] == '#') or (slash_comments and line.startswith('//', i)) or (lang == 'SQL' and line.startswith('--', i)):
                break
            elif (slash_comments or lang == 'SQL') and line.startswith('/*', i):
                block, i = 1, i + 2
            else:
                has_code = True
                raw = re.match(r'(?:br|r)(#+)?"', line[i:]) if lang == 'Rust' else None
                if raw:
                    literal = '"' + (raw[1] or '')
                    raw_literal = True
                    i += len(raw[0])
                elif lang == 'Python' and line[i:i + 3] in {'"""', "'''"}:
                    literal, i = line[i:i + 3], i + 3
                elif line[i] in {'"', "'", '`'} and not (lang == 'Rust' and line[i] == "'" and not re.match(r"'(?:\\.|[^'])'", line[i:])):
                    literal, i = line[i], i + 1
                else:
                    i += 1
        count += has_code
        # Ordinary quoted literals cannot span physical lines in Go/Rust/C.
        if lang in {'Go', 'Rust', 'C', 'C++'} and literal in {'"', "'"}:
            literal = None
    return count


def category(path, raw):
    if re.search(br'Code generated[^\n]*DO NOT EDIT|@generated', raw[:2048]):
        return 'generated'
    parts = Path(path).parts
    if ('testdata' in parts or 'tests' in parts or Path(path).stem.startswith('test_')
            or Path(path).stem.endswith(('_test', '_tests')) or '/testfixture/' in path):
        return 'tests'
    return 'production'


def subsystem(path):
    if path.startswith('tools/gomad/internal/backends/wasm/wasmhost/'):
        return 'preexisting-rust-helper'
    if path.startswith('tools/gomad/rust/wasmhost/'):
        return 'linked-rust-engine'
    if path.startswith('tools/gomad/rust/gomad/'):
        return 'rust-host'
    if path.startswith(('tools/gomad/cmd/', 'tools/gomad/internal/runner/', 'tools/gomad/internal/host/', 'tools/gomad/internal/evidence/')):
        return 'go-host-or-reference-fixtures'
    if path.startswith('tools/gomad/internal/model/wasi/'):
        return 'retained-wasi-reference'
    if path.startswith('tools/gomad/internal/'):
        return 'go-model-oracle-toolchain-and-support'
    if path.startswith('tools/gomad/'):
        return 'gomad-support'
    return 'repository-other'


def inventory(revision):
    revision = subprocess.check_output(['git', 'rev-parse', revision + '^{commit}'], text=True).strip()
    tree = subprocess.check_output(['git', 'ls-tree', '-rz', '--full-tree', revision])
    entries = []
    for record in tree.split(b'\0'):
        if not record:
            continue
        meta, path = record.split(b'\t', 1)
        mode, kind, oid = meta.split()
        if kind == b'blob':
            entries.append((path.decode(), mode.decode(), oid.decode()))
    entries.sort()
    process = subprocess.Popen(['git', 'cat-file', '--batch'], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    files = []
    try:
        for path, mode, oid in entries:
            process.stdin.write((oid + '\n').encode())
            process.stdin.flush()
            header = process.stdout.readline().split()
            size = int(header[2])
            raw = process.stdout.read(size)
            assert process.stdout.read(1) == b'\n'
            lang = language(path, raw) if mode != '120000' else None
            row = dict(path=path, git_blob=oid, sha256=hashlib.sha256(raw).hexdigest(), bytes=size)
            if lang:
                text = raw.decode('utf-8', errors='replace')
                row.update(language=lang, category=category(path, raw), subsystem=subsystem(path),
                           total_lines=len(text.splitlines()), nonblank_lines=sum(bool(x.strip()) for x in text.splitlines()),
                           code_lines=code_lines(text, lang))
            files.append(row)
    finally:
        process.stdin.close()
        assert process.wait() == 0
    groups = {}
    for row in files:
        if 'language' not in row:
            continue
        key = '/'.join(row[k] for k in ('subsystem', 'language', 'category'))
        group = groups.setdefault(key, dict(files=0, bytes=0, total_lines=0, nonblank_lines=0, code_lines=0))
        group['files'] += 1
        for metric in ('bytes', 'total_lines', 'nonblank_lines', 'code_lines'):
            group[metric] += row[metric]
    paths = ''.join(row['path'] + '\n' for row in files)
    return dict(revision=revision, repository_files=len(files), repository_bytes=sum(row['bytes'] for row in files),
                paths_sha256=hashlib.sha256(paths.encode()).hexdigest(), groups=groups, files=files)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', required=True)
    parser.add_argument('--candidate', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    report = dict(schema=VERSION, counter_git_revision=subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(), counter_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                  command=['python3', 'tools/gomad/scripts/source-size.py', '--baseline', args.baseline,
                           '--candidate', args.candidate, '--output', str(args.output)],
                  definition='Physical nonblank lines with tokens outside comments; literal content counts as code. Not cloc. Categories classify whole files; inline tests remain in their production file. Generated markers take precedence over test paths. Full repository totals include all tracked blobs; per-file SHA256 and paths permit narrower comparisons.',
                  baseline=inventory(args.baseline), candidate=inventory(args.candidate))
    with args.output.open('x') as handle:
        json.dump(report, handle, sort_keys=True, indent=2)
        handle.write('\n')


if __name__ == '__main__':
    main()
