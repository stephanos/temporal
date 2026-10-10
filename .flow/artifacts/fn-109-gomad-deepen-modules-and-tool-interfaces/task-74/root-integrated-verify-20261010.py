import collections
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
WORKER = PRIMARY / '.worktrees/fn-109-74-retention-candidate'
OUTPUT = WORKER / '.flow/tmp/fn10974-evidence'
ARTIFACT = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/worker'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


rows = subprocess.check_output(['git', 'ls-files', '-s', '-z'], cwd=WORKER).decode().split('\0')
identities = {}
for row in rows:
    if not row:
        continue
    metadata, name = row.split('\t', 1)
    mode, blob, stage = metadata.split()
    assert stage == '0'
    if name.startswith(('.flow/', '.turbo/')) or name == 'MILESTONES.md':
        continue
    left, right = PRIMARY / name, WORKER / name
    if mode == '160000':
        identities[name] = {'mode': mode, 'gitlink': blob}
        assert subprocess.check_output(['git', 'ls-files', '-s', '--', name], cwd=PRIMARY) == subprocess.check_output(['git', 'ls-files', '-s', '--', name], cwd=WORKER)
    elif mode == '120000':
        assert left.is_symlink() and right.is_symlink()
        assert os.readlink(left) == os.readlink(right)
        assert sha(left) == sha(right)
        identities[name] = {'mode': mode, 'literal_link': os.readlink(left), 'target_sha256': sha(left)}
    else:
        assert left.is_file() and right.is_file() and sha(left) == sha(right), name
        identities[name] = {'mode': mode, 'sha256': sha(left)}

packet = sorted(subprocess.check_output(['git', 'diff-tree', '--no-commit-id', '--name-only', '-r', '22ce10e287a9fe9a4c08df55c621b91d5cfe29a1'], cwd=WORKER).decode().splitlines())
assert len(packet) == 18 and all(name.startswith(ARTIFACT + '/') for name in packet)
assert all(sha(PRIMARY / name) == sha(WORKER / name) for name in packet)


def blocks(path):
    lines = path.read_text().splitlines()
    result = []
    for index, line in enumerate(lines):
        if re.match(r'^tools/gomad3/[^:]+:\d+:\d+: .+$', line):
            assert '^' in lines[index + 2]
            result.append('\n'.join(lines[index:index + 3]))
    return result


lint_paths = [OUTPUT / 'before-aggregate-lint.log', OUTPUT / 'after-aggregate-lint.log', PRIMARY / '.worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-aggregate-lint.log', PRIMARY / '.worktrees/fn-109-23-lint-complexity-candidate/.flow/tmp/fn10923-complexity/final-fast-lint.log', PRIMARY / '.worktrees/fn-109-23-lint-complexity-candidate/.flow/tmp/fn10923-complexity/final-gomad-lint.log']
lint = blocks(lint_paths[0])
assert len(lint) == 50 and all(blocks(path) == lint for path in lint_paths)
lint_digest = hashlib.sha256('\n'.join(lint).encode()).hexdigest()
assert lint_digest == '034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea'
assert collections.Counter(re.search(r'\(([^()]*)\)$', block.splitlines()[0])[1] for block in lint) == {'staticcheck': 42, 'forbidigo': 8}
assert len(blocks(OUTPUT / 'runner-lint.log')) == 6
assert blocks(OUTPUT / 'runner-lint.log') == blocks(PRIMARY / '.worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/runner-lint.log')

historical = WORKER / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
assert sha(historical) == '2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
domain = dict(line.split('  ', 1)[::-1] for line in historical.read_text().splitlines() if '  tests/' in line)
assert len(domain) == 132
assert all(sha(PRIMARY / name) == sha(WORKER / name) == digest for name, digest in domain.items())
top_tests = sorted(name for name in domain if Path(name).parent == Path('tests') and name.endswith('_test.go'))
assert len(top_tests) == 113 and top_tests == sorted(str(path.relative_to(PRIMARY)) for path in (PRIMARY / 'tests').glob('*_test.go'))
result = {'primary_head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=PRIMARY).decode().strip(), 'worker_head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=WORKER).decode().strip(), 'tracked_nonflow_identities': len(identities), 'identities_sha256': hashlib.sha256(json.dumps(identities, sort_keys=True).encode()).hexdigest(), 'special_modes': {name: value for name, value in identities.items() if value['mode'] in ('120000', '160000')}, 'packet_files_equal': len(packet), 'retention_sha256': sha(PRIMARY / 'tools/gomad3/runner/retention_characterization_test.go'), 'original_lint_blocks': len(lint), 'original_lint_blocks_sha256': lint_digest, 'runner_lint_blocks': 6, 'validation_domain_files': len(domain), 'discovered_top_level_tests': len(top_tests), 'product_acceptance_pass': False}
print(json.dumps(result, sort_keys=True))
