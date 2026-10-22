#!/usr/bin/env python3
"""Compile the unchanged transferred runtime guests with a declared cooperative overlay."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('source', 'compiler', 'overlay', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--runtime-identity', required=True)
    args = parser.parse_args()
    args.work = args.work.resolve()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    compiler = args.compiler.resolve()
    source = args.source.resolve()
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    environment.update(GOROOT=str(compiler.parent.parent), GOOS='wasip1', GOARCH='wasm',
        GOEXPERIMENT='nogreenteagc', GOWORK='off', GOENV='off', GOFLAGS='', GOTOOLCHAIN='local',
        CGO_ENABLED='0', GOMAXPROCS='1')
    version = subprocess.run([str(compiler), 'version'], capture_output=True, check=True, timeout=60)
    assert b'go1.27.1 ' in version.stdout, version.stdout
    records = []
    for name, package, relative in (
        ('choices', './internal/backends/wasm/toolchain/testdata/choices', 'tools/gomad'),
        ('runtime', './internal/backends/wasm/testdata/runtime_probes', 'tools/gomad'),
        ('inspection', './common/gomaddebug/testdata/fixture', '.')):
        output = args.work / (name + '.wasm')
        command = [str(compiler), 'build', '-p=1', '-trimpath', '-buildvcs=false',
            *(['-tags=gomad'] if name == 'inspection' else []), '-overlay', str(args.overlay.resolve()), '-o', str(output), package]
        result = subprocess.run(command, cwd=source / relative, env=environment, capture_output=True, timeout=600)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        records.append(dict(name=name, argv=command, status=result.returncode,
            package=package, working_directory=str(source / relative)))
        (args.work / 'commands.json').write_text(json.dumps(records, indent=2) + '\n')
        assert result.returncode == 0, (name, result.returncode, result.stderr)
        records[-1]['module_sha256'] = hashlib.sha256(output.read_bytes()).hexdigest()
        fixture_source = source / relative / package / 'main.go'
        records[-1]['source_sha256'] = hashlib.sha256(fixture_source.read_bytes()).hexdigest()
    (args.work / 'guests.json').write_text(json.dumps(dict(runtime_identity=args.runtime_identity,
        compiler_sha256=hashlib.sha256(compiler.read_bytes()).hexdigest(),
        overlay_sha256=hashlib.sha256(args.overlay.read_bytes()).hexdigest(), records=records), indent=2) + '\n')


if __name__ == '__main__':
    main()
