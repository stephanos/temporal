#!/usr/bin/env python3
"""Supply committed AST inputs and finish the historical checkpoint validation."""
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
HEAD = 'add55cf6116fbed23e8c4051e120521b2f3f033c'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(['git', '-C', str(ROOT), *args])


def inventory(root):
    result = {}
    for path in sorted(root.rglob('*')):
        if '.toolchain' in path.relative_to(root).parts:
            continue
        assert not path.is_symlink(), str(path)
        if path.is_file():
            result[path.relative_to(root).as_posix()] = sha(path.read_bytes())
    return result


def main():
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    prior = OUT / 'checkpoint-verification.json'
    assert sha(prior.read_bytes()) == 'd5229d0d6e4c0ef31990ab651a03c80d1d84430eb50dc04d8597ab7a0d030ed0'
    original = json.loads(prior.read_text())
    scratch = Path(original['scratch'])
    before = inventory(scratch)
    assert before == original['candidate_sha256']
    immutable_paths = [prior, OUT / 'checkpoint-report.md',
                       OUT / 'checkpoint-baseline-validate.log', OUT / 'checkpoint-candidate-validate.log']
    immutable = {str(path): sha(path.read_bytes()) for path in immutable_paths}
    paths = [p for p in git('ls-tree', '-r', '--name-only', HEAD, 'tests').decode().splitlines()
             if re.fullmatch(r'tests/[^/]+_test\.go', p)]
    assert len(paths) == 113
    archive_data = git('archive', '--format=tar', HEAD, *paths)
    added = {}
    with tarfile.open(fileobj=io.BytesIO(archive_data), mode='r:') as archive:
        members = archive.getmembers()
        for member in members:
            name = PurePosixPath(member.name)
            assert not name.is_absolute() and '..' not in name.parts
            assert member.isdir() or member.isreg(), (member.name, member.type)
            if member.isreg():
                assert member.name in paths
                assert not (scratch / member.name).exists()
        for member in members:
            destination = scratch / member.name
            if member.isdir():
                destination.mkdir(parents=True, exist_ok=True)
                continue
            stream = archive.extractfile(member)
            assert stream is not None
            data = stream.read()
            assert len(data) == member.size
            assert data == git('show', HEAD + ':' + member.name)
            destination.write_bytes(data)
            destination.chmod(member.mode & 0o777)
            added[member.name] = sha(data)
    assert set(added) == set(paths)
    candidate = dict(before, **added)
    assert inventory(scratch) == candidate
    env = os.environ.copy()
    for name in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
        env.pop(name, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2',
               GOMAD3_STOCK_GO=str(GO),
               PATH=str(GO.parent) + os.pathsep + env.get('PATH', ''))
    command = ['make', 'validate']
    module = scratch / 'tools/gomad3'
    completed = subprocess.run(command, cwd=module, env=env, stdout=subprocess.PIPE,
                               stderr=subprocess.STDOUT, timeout=600)
    log = OUT / 'checkpoint-final-validate-with-root-inputs.log'
    log.write_bytes(completed.stdout)
    print('final validate with committed root AST inputs: exit ' + str(completed.returncode), flush=True)
    assert inventory(scratch) == candidate
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    for path, expected in immutable.items():
        assert sha(Path(path).read_bytes()) == expected
    for path, origin in original['historical_source_origins'].items():
        assert sha((scratch / path).read_bytes()) == origin['sha256']
    report = dict(base_head=HEAD, scratch=str(scratch),
                  previous_source_manifest=str(prior), previous_source_manifest_sha256=sha(prior.read_bytes()),
                  previous_red_logs_preserved=True, immutable_artifact_sha256=immutable,
                  added_root_input_archive_sha256=sha(archive_data), added_root_input_sha256=added,
                  input_selection='Every committed top-level tests/*_test.go; no recursion, dependencies, non-test files or root configuration needed.',
                  mechanism='manifestgen.Run reads the generator spec and existing output, then ListTests reads tests directory entries, build headers and AST declarations for each selected platform; no compilation or execution.',
                  generator_source_sha256=sha((module / 'qualification/set/manifestgen/manifestgen.go').read_bytes()),
                  command=command, cwd=str(module), exit_code=completed.returncode,
                  environment={k: env[k] for k in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'PATH', 'GOMAD3_STOCK_GO')},
                  unset=['GOMADSEED', 'GOMAD3_CHILD_SEED'], log=str(log), log_sha256=sha(completed.stdout),
                  full_source_unchanged_after_checks=True, source_file_count=len(candidate),
                  historical_source_identities_verified=17, changed_task_source_paths=original['changed_source_paths'],
                  native_acceptance=False, formal_ship_verdict=False, root_workloads_executed=False,
                  working_tree_source_copied=False, task19_source_copied=False,
                  shared_source_writes=False, git_index_writes=False, flow_mutations=False)
    destination = OUT / 'checkpoint-qualification-inputs-verification.json'
    destination.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(report=str(destination), report_sha256=sha(destination.read_bytes()),
                         added_root_files=len(added), exit_code=completed.returncode)), flush=True)
    assert completed.returncode == 0, completed.stdout.decode(errors='replace')


if __name__ == '__main__':
    main()
