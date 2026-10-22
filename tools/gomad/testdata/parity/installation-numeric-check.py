#!/usr/bin/env python3
"""Public install ParseUint grammar with independent literal bounds; no guests."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

# Every successful spelling denotes exactly these unchanged configured bounds.
FUEL, MEMORY = 123456789, 134217728
CASES = [
    ('decimal', '123456789', '134217728', 0),
    ('hexadecimal', '0x75bcd15', '0x8000000', 0),
    ('binary', '0b111010110111100110100010101', '0b1000000000000000000000000000', 0),
    ('octal', '0o726746425', '0o1000000000', 0),
    ('legacy-octal', '0726746425', '01000000000', 0),
    ('decimal-underscore', '123_456_789', '134_217_728', 0),
    ('prefix-underscore', '0x_75bc_d15', '0x_800_0000', 0),
    ('positive-sign', '+123456789', '134217728', 2),
    ('negative-sign', '-123456789', '134217728', 2),
    ('invalid-octal', '08', '134217728', 2),
    ('bad-underscore', '123__456789', '134217728', 2),
    ('overflow', '18446744073709551616', '134217728', 2),
]


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def check(binary, compiler, root, helper=None):
    root.mkdir(parents=True, exist_ok=False)
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_') and key != 'GOROOT'}
    environment.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off')
    commands, views = [], []
    for name, fuel, memory, expected in CASES:
        output = root / (name + '.json')
        argv = [binary, 'install', '--compiler', compiler, '--cache', root / 'cache', '--output', output,
                '--fuel', fuel, '--memory-bytes', memory]
        if helper is not None:
            argv += ['--helper', helper]
        result = subprocess.run(list(map(str, argv)), env=environment, capture_output=True, timeout=120)
        (root / (name + '.stdout')).write_bytes(result.stdout)
        (root / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, argv=list(map(str, argv)), status=result.returncode, expected=expected,
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (root / 'commands.json').write_text(json.dumps(commands, sort_keys=True, indent=2) + '\n')
        if result.returncode != expected:
            raise ValueError(f'{name}.status: got {result.returncode}, expected {expected}: {result.stderr!r}')
        if expected == 0:
            manifest = json.loads(output.read_bytes())
            observed = ((manifest['options']['Fuel'], manifest['options']['MemoryBytes']) if helper else
                        (manifest['fuel'], manifest['memory_bytes']))
            if observed != (FUEL, MEMORY):
                raise ValueError(f'{name}.bounds: got {observed!r}, expected {(FUEL, MEMORY)!r}')
        elif output.exists():
            raise ValueError(name + ': invalid numeric flag published an installation')
        views.append(dict(case=name, fuel=fuel, memory_bytes=memory, status=result.returncode,
                          **(dict(parsed_fuel=FUEL, parsed_memory_bytes=MEMORY) if expected == 0 else {})))
    (root / 'view.json').write_text(json.dumps(views, sort_keys=True, indent=2) + '\n')
    return views


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'compiler', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--helper', type=Path)
    args = parser.parse_args()
    views = check(args.binary.resolve(), args.compiler.resolve(), args.work.resolve(), args.helper and args.helper.resolve())
    print(f'installation numeric: {len(views)} public cases match independent literal bounds and refusals')
