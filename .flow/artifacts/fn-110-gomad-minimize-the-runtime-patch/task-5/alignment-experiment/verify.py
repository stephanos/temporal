"""Measure one scratch-only blank-line grouping candidate with no Git index writes."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import time

REPO = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
TASK = OUT.parent
PRIOR = Path('/tmp/fn110-source-size-CNRyRbvw/final')
TOOL = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin')
PATCH = REPO / 'tools/gomad3/toolchain/runtime/go1.27.1.patch'
DESCRIPTOR = REPO / 'tools/gomad3/toolchain/version/version.json'
ARCHIVE = REPO / 'tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz'
OVERLAY = REPO / 'tools/gomad3/toolchain/runtime/overlay'


def utc():
    return datetime.now(timezone.utc).isoformat()


def identity(data):
    return dict(bytes=len(data), lines=data.count(b'\n'), sha256=hashlib.sha256(data).hexdigest())


def tree(root):
    result = {}
    for path in sorted(root.rglob('*')):
        assert not path.is_symlink(), path
        if path.is_file():
            result[path.relative_to(root).as_posix()] = identity(path.read_bytes())
    return result


def inputs():
    return dict(patch=identity(PATCH.read_bytes()), descriptor=identity(DESCRIPTOR.read_bytes()),
                archive=identity(ARCHIVE.read_bytes()), overlay=tree(OVERLAY),
                prior_pristine=tree(PRIOR / 'a'), prior_current=tree(PRIOR / 'b'))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--scratch', required=True)
    parser.add_argument('--prepare', action='store_true')
    args = parser.parse_args()
    scratch = Path(args.scratch).resolve()
    assert scratch.parent == Path('/tmp') and scratch.name.startswith('fn110-alignment-')
    retained = json.loads((TASK / 'source-size-evidence.json').read_text())
    commands = []
    started, start_clock = utc(), time.monotonic()

    def run(argv, cwd=scratch, expected=(0,), stdin=None, output=None, extra_env=None):
        begin, clock = utc(), time.monotonic()
        overrides = dict(GIT_OPTIONAL_LOCKS='0', LC_ALL='C', TZ='UTC', GOTOOLCHAIN='local',
                         GOWORK='off', GOENV='off', GO111MODULE='off', CGO_ENABLED='0',
                         GOCACHE=str(scratch / 'go-cache'), GOFLAGS='')
        overrides.update(extra_env or {})
        result = subprocess.run(list(map(str, argv)), cwd=cwd, input=stdin, capture_output=True,
                                env=dict(os.environ, **overrides), timeout=120)
        commands.append(dict(argv=list(map(str, argv)), cwd=str(cwd), started_utc=begin,
                             finished_utc=utc(), duration_seconds=time.monotonic()-clock,
                             exit_code=result.returncode, expected_exit_codes=list(expected),
                             environment_overrides=overrides,
                             stdin=identity(stdin) if stdin is not None else None,
                             stdout=identity(result.stdout), stderr=identity(result.stderr),
                             stdout_path=str(output) if output else None,
                             stdout_text=None if output else result.stdout.decode(errors='replace'),
                             stderr_text=result.stderr.decode(errors='replace')))
        (OUT / 'commands.json').write_text(json.dumps(commands, indent=2)+'\n')
        assert result.returncode in expected, (argv, result.returncode, result.stderr)
        if output:
            Path(output).write_bytes(result.stdout)
        return result.stdout

    if args.prepare:
        assert scratch.is_dir() and not list(scratch.iterdir())
        before = inputs()
        for key, expected_hash in (
                ('patch', '8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b'),
                ('descriptor', '94358dc0d221c0c0ba0e4f487303b0e9acd46afa212d24091ab44b124f276779'),
                ('archive', '4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1')):
            assert before[key]['sha256'] == expected_hash
        assert before['overlay'] == {row['path']: {key: row[key] for key in ('bytes', 'lines', 'sha256')}
                                     for row in retained['overlay_current']['inventory']}
        for label, key in (('pristine', 'prior_pristine'), ('final', 'prior_current')):
            assert before[key] == {row['path']: row[label] for row in retained['materialized_source_inventory']}
        for name, source in (('current/a', PRIOR / 'a'), ('current/b', PRIOR / 'b'),
                             ('candidate/a', PRIOR / 'a'), ('candidate/b', PRIOR / 'b')):
            shutil.copytree(source, scratch / name)
        assert inputs() == before
        preparation = dict(started_utc=started, finished_utc=utc(), duration_seconds=time.monotonic()-start_clock,
                           scratch=str(scratch), shared_before=before,
                           verified_all_22_prior_members_against_retained_evidence=True,
                           verified_all_79_overlay_files_against_retained_evidence=True)
        (OUT / 'prepared-inputs.json').write_text(json.dumps(preparation, indent=2)+'\n')
        print(json.dumps(dict(prepared=True, scratch=str(scratch), duration_seconds=preparation['duration_seconds'])))
        return

    preparation = json.loads((OUT / 'prepared-inputs.json').read_text())
    assert preparation['scratch'] == str(scratch)
    before = preparation['shared_before']
    assert inputs() == before
    assert tree(scratch / 'current/a') == before['prior_pristine']
    assert tree(scratch / 'current/b') == before['prior_current']
    assert tree(scratch / 'candidate/a') == before['prior_pristine']
    file = scratch / 'candidate/b/src/runtime/runtime2.go'
    run([TOOL / 'gofmt', '-w', file])
    canonical = run([TOOL / 'gofmt', '-l', file])
    assert canonical == b''
    shutil.copyfile(OUT / 'token-compare.go', scratch / 'token-compare.go')
    run([TOOL / 'gofmt', '-w', scratch / 'token-compare.go'])
    run([TOOL / 'go', 'build', '-tags', 'test_dep', '-o', scratch / 'token-compare', scratch / 'token-compare.go'])
    token_bytes = run([scratch / 'token-compare', scratch / 'current/b', scratch / 'candidate/b'],
                      output=OUT / 'tokens.json')
    tokens = json.loads(token_bytes)
    assert tokens['all_equal'] and tokens['go_files'] == 21
    candidate = tree(scratch / 'candidate/b')
    changed = [name for name in candidate if candidate[name] != before['prior_current'][name]]
    assert changed == ['src/runtime/runtime2.go'], changed
    name = changed[0]
    formatting = run(['git', 'diff', '--no-index', '--no-ext-diff', '--no-textconv', '--binary', '--no-prefix',
                      '--abbrev=7', '--diff-algorithm=myers', '--unified=3', '--',
                      'current/b/'+name, 'candidate/b/'+name], expected=(1,), output=OUT / 'formatting-only.patch')

    def diff(pair, context, output):
        return run(['git', 'diff', '--no-index', '--no-ext-diff', '--no-textconv', '--binary', '--no-prefix',
                    '--abbrev=7', '--diff-algorithm=myers', '--unified='+str(context), '--', 'a', 'b'],
                   cwd=scratch / pair, expected=(1,), output=output)

    current_u3 = diff('current', 3, OUT / 'current-U3.patch')
    current_u1 = diff('current', 1, OUT / 'current-U1.patch')
    assert current_u3 == (TASK / 'source-size-final-U3.patch').read_bytes()
    assert current_u1 == PATCH.read_bytes()
    original = (TASK / 'source-size-original-baseline-U3.patch').read_bytes()
    assert identity(original) == {key: retained['patches']['original_baseline_U3'][key]
                                  for key in ('bytes', 'lines', 'sha256')}
    shutil.copyfile(TASK / 'source-size-original-baseline-U3.patch', OUT / 'original-baseline-U3.patch')
    candidate_u3 = diff('candidate', 3, OUT / 'candidate-U3.patch')
    candidate_u1 = diff('candidate', 1, OUT / 'candidate-U1.patch')
    assert diff('candidate', 3, scratch / 'repeat-U3.patch') == candidate_u3
    assert diff('candidate', 1, scratch / 'repeat-U1.patch') == candidate_u1
    materializations = []
    for label, patch in (('U3', candidate_u3), ('U1', candidate_u1)):
        root = scratch / ('applied-'+label)
        shutil.copytree(scratch / 'candidate/a', root)
        for dry in (True, False):
            argv = ['patch']+(['--dry-run'] if dry else [])+['--batch', '-V', 'none', '-p1', '-F', '0']
            stdout = run(argv, cwd=root, stdin=patch)
            assert not re.search(rb'\b(fuzz|offset|FAILED|reversed|previously applied)\b', stdout, re.I)
        assert not list(root.rglob('*.orig')) and not list(root.rglob('*.rej'))
        assert tree(root) == candidate
        materializations.append(dict(context=label, exit_code=0, zero_fuzz_zero_offset=True,
                                     all_22_members_byte_identical_to_candidate=True))
    after = inputs()
    assert after == before
    tool_identity = {str(path): identity(path.read_bytes()) for path in (
        TOOL / 'go', TOOL / 'gofmt', OUT / 'verify.py', OUT / 'token-compare.go',
        scratch / 'token-compare.go', scratch / 'token-compare')}
    versions = dict(go=run([TOOL / 'go', 'version']).decode().strip(),
                    uname=run(['uname', '-s', '-m']).decode().strip(),
                    git=run(['git', '--version']).decode().strip(),
                    patch=run(['patch', '--version']).decode().strip())
    after = inputs()
    assert after == before
    stats = {label: identity(data) for label, data in (
        ('original_baseline_U3', original), ('current_U3', current_u3), ('current_U1', current_u1),
        ('candidate_U3', candidate_u3), ('candidate_U1', candidate_u1))}
    def sections(patch):
        return {part.splitlines()[0].decode().split()[2][2:]: identity(part)
                for part in re.split(rb'(?=^diff --git )', patch, flags=re.M) if part}
    result = dict(schema='fn110-alignment-experiment/v1', started_utc=started, finished_utc=utc(),
                  duration_seconds=time.monotonic()-start_clock, preparation=preparation,
                  scratch=str(scratch), requested_routing='thinking scout gpt-6.1-sol high',
                  actual_model_metadata='unknown; not host-observed', tools=tool_identity, versions=versions,
                  patches=stats, current_U3_sections=sections(current_u3), candidate_U3_sections=sections(candidate_u3),
                  formatting_diff=identity(formatting), token_equality=tokens,
                  candidate_source_inventory=candidate, changed_members=changed,
                  shared_input_stability={key: after[key] == before[key] for key in before},
                  materializations=materializations,
                  comparisons=dict(U3_bytes_saved=len(current_u3)-len(candidate_u3),
                                   U1_bytes_saved=len(current_u1)-len(candidate_u1),
                                   candidate_U3_minus_original_baseline_U3_bytes=len(candidate_u3)-len(original),
                                   additional_bytes_to_be_strictly_smaller=max(0, len(candidate_u3)-len(original)+1),
                                   candidate_U1_minus_candidate_U3_bytes=len(candidate_u1)-len(candidate_u3)),
                  limits=['Scratch-only formatting candidate; no adoption or acceptance claim.',
                          'Token and comment equality does not prove compiled ABI or native runtime behavior.',
                          'No Gomad package loading, generation, tests, builds, descriptor edits or Flow state mutation.',
                          'No Git index/history mutation; only index-free git diff commands.',
                          'Unsupported linux/arm64; native R4 and R7 gates remain open.'])
    (OUT / 'evidence.json').write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps(dict(patches=stats, comparisons=result['comparisons'],
                         token_equality=tokens['all_equal'], shared_input_stability=result['shared_input_stability'],
                         duration_seconds=result['duration_seconds']), indent=2))


if __name__ == '__main__':
    main()
