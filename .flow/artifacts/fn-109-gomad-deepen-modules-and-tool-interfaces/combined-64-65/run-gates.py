#!/usr/bin/env python3
import argparse
import collections
import datetime
import difflib
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shlex
import shutil
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/joined-current')
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-64-65'
ARTIFACTS = PACKET.parent
BASELINE = ARTIFACTS / 'task-63'
OLD_MANIFEST = ARTIFACTS / 'combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
SPEC = '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'
PRIMARY_SPEC = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal') / SPEC
SPEC_SHA = '851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c'
ADMITTED = {
    'TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures',
    'TestRunPublishesConnectedWorldBundle',
    'TestRunClassifiesConnectedWorldDeadlock',
    'TestRunCountsConnectedWorldReplayDivergence',
    'TestRunRejectsInvalidConnectedWorldBeforePublication',
    'TestRunRejectsPreparedTargetMutationBeforeFailurePublication',
}
RUNNER = 'go.temporal.io/server/tools/gomad3/runner'

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def save(name, value):
    with (PACKET / name).open('x') as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')

def inputs():
    paths = set(json.loads(OLD_MANIFEST.read_text()))
    paths.update(git('ls-files', '-z', '--cached', '--others', '--exclude-standard', '--',
                     'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration',
                     'cmd/tools/lintcode', 'go.mod', 'go.sum', 'Makefile',
                     '.github/.golangci.yml').decode().strip('\0').split('\0'))
    paths.add(SPEC)
    paths.discard('')
    relevant_count = sum((ROOT / path).is_file() for path in paths)
    retained_scope = set(json.loads(OLD_MANIFEST.read_text())) | {path for path in paths if path.startswith('tools/gomad3/')}
    retained_scope_count = sum((ROOT / path).is_file() for path in retained_scope)
    paths.update(str(path) for path in [PRIMARY_SPEC, Path(__file__).resolve(), OLD_MANIFEST,
                 BASELINE / 'final-ordinary.log', BASELINE / 'final-ordinary.json',
                 BASELINE / 'final-ordinary-source.sha256', BASELINE / 'final-ordinary-env.txt',
                 BASELINE / 'final-integrated-lint.log', BASELINE / 'final-integrated-lint.json',
                 BASELINE / 'final-integrated-lint-source.sha256', BASELINE / 'final-integrated-lint-env.txt'])
    present = {path: sha(ROOT / path) for path in sorted(paths) if (ROOT / path).is_file()}
    absent = sorted(path for path in paths if not (ROOT / path).is_file())
    tracked = set(git('ls-files', '-z', '--', 'tools/gomad3', 'tools/gomad3sim',
                      'tools/gomad3integration', 'cmd/tools/lintcode').decode().strip('\0').split('\0'))
    reference_sources = {path: hashlib.sha256(git('show', 'a1908ed759:' + path)).hexdigest()
                         for path in sorted({block['path'] for block in lint_blocks(BASELINE / 'final-integrated-lint.log')})}
    return {'files': present, 'absent_inventory_paths': absent, 'relevant_source_count': relevant_count,
            'original_inventory_plus_new_source_count': retained_scope_count,
            'lint_comparison_reference_commit': git('rev-parse', 'a1908ed759').decode().strip(),
            'lint_comparison_reference_source_sha256': reference_sources,
            'missing_tracked_relevant_paths': sorted(tracked.intersection(absent))}

def outcomes(path):
    named = {}
    unparsable = []
    for number, line in enumerate(path.read_text().splitlines(), 1):
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            unparsable.append({'line': number, 'text': line})
            continue
        if event.get('Package') == RUNNER and event.get('Test') and event.get('Action') in ('pass', 'fail', 'skip'):
            if event['Test'] in named:
                raise ValueError('Duplicate terminal outcome: ' + event['Test'])
            named[event['Test']] = event['Action']
    return named, unparsable

def compare_outcomes():
    old, old_unparsable = outcomes(BASELINE / 'final-ordinary.log')
    new, new_unparsable = outcomes(PACKET / 'ordinary-runner.log')
    changes = [{'test': name, 'before': old[name], 'after': new.get(name)}
               for name in sorted(old) if old[name] != new.get(name)]
    allowed = lambda name: name.split('/')[0] in ADMITTED
    original_tops = {name.split('/')[0] for name in old}
    additions = new.keys() - old.keys()
    controls = {name: new[name] for name in sorted(additions) if name.split('/')[0] not in original_tops}
    reached = {name: new[name] for name in sorted(additions) if name.split('/')[0] in original_tops}
    report = {'baseline_counts': dict(collections.Counter(old.values())),
              'current_counts': dict(collections.Counter(new.values())),
              'original_changes': changes, 'admitted_original_names': sorted(ADMITTED),
              'unadmitted_original_changes': [entry for entry in changes if not allowed(entry['test'])],
              'missing_original_outcomes': sorted(old.keys() - new.keys()),
              'new_control_outcomes': controls, 'newly_reached_original_subtests': reached,
              'unadmitted_newly_reached_original_subtests': {name: result for name, result in reached.items() if not allowed(name)},
              'baseline_non_json_lines': old_unparsable,
              'current_non_json_lines': new_unparsable,
              'baseline_log_sha256': sha(BASELINE / 'final-ordinary.log'),
              'current_log_sha256': sha(PACKET / 'ordinary-runner.log')}
    save('outcome-comparison.json', report)
    return report

HEADER = re.compile(r'^(tools/gomad3/[^:]+):(\d+):(\d+): (.+)$')

def lint_blocks(path):
    lines = path.read_text().splitlines()
    blocks = []
    for index, line in enumerate(lines):
        match = HEADER.match(line)
        if not match:
            continue
        if index + 2 >= len(lines) or '^' not in lines[index + 2]:
            raise ValueError('Incomplete lint diagnostic block: ' + line)
        blocks.append({'path': match[1], 'line': int(match[2]), 'column': int(match[3]),
                       'message': match[4], 'source': lines[index + 1], 'caret': lines[index + 2],
                       'block': '\n'.join(lines[index:index + 3])})
    return blocks

def compare_lint():
    old = lint_blocks(BASELINE / 'final-integrated-lint.log')
    new = lint_blocks(PACKET / 'integrated-lint.log')
    retained_hashes = {}
    for line in (BASELINE / 'final-integrated-lint-source.sha256').read_text().splitlines():
        digest, path = line.split(None, 1)
        retained_hashes[path.strip()] = digest
    mappings, gaps, mapped, source_bindings = [], [], [], {}
    for block in old:
        path = block['path']
        original = git('show', 'a1908ed759:' + path)
        source_bindings[path] = {'reference_sha256': hashlib.sha256(original).hexdigest(),
                                 'retained_task63_source_sha256': retained_hashes.get(path),
                                 'current_sha256': sha(ROOT / path)}
        if hashlib.sha256(original).hexdigest() != retained_hashes.get(path):
            gaps.append({'path': path, 'reason': 'reference does not match retained task63 actual source hash'})
            continue
        before = original.decode().splitlines()
        after = (ROOT / path).read_text().splitlines()
        line_map = {}
        for segment in difflib.SequenceMatcher(None, before, after, autojunk=False).get_matching_blocks():
            for offset in range(segment.size):
                line_map[segment.a + offset + 1] = segment.b + offset + 1
        projected = dict(block)
        if block['line'] in line_map:
            projected['line'] = line_map[block['line']]
            projected['block'] = f"{path}:{projected['line']}:{block['column']}: {block['message']}\n{block['source']}\n{block['caret']}"
            if projected['line'] != block['line']:
                mappings.append({'path': path, 'old_line': block['line'], 'new_line': projected['line'],
                                 'exact_source_line': before[block['line'] - 1]})
        mapped.append((block, projected))
    remaining = collections.Counter(block['block'] for block in new)
    removed, preserved = [], []
    for original, projection in mapped:
        if remaining[projection['block']]:
            remaining[projection['block']] -= 1
            preserved.append({'before': original['block'], 'after': projection['block']})
        else:
            removed.append(original['block'])
    introduced = list(remaining.elements())
    report = {'baseline_count': len(old), 'current_count': len(new),
              'baseline_full_blocks': [block['block'] for block in old],
              'current_full_blocks': [block['block'] for block in new],
              'preserved': preserved, 'exact_source_line_mappings': mappings,
              'removed': removed, 'introduced': introduced, 'mapping_gaps': gaps,
              'integrated_errortype_reached': False if re.search(r'Makefile:505: lint-code\] Error', (PACKET / 'integrated-lint.log').read_text()) else None,
              'integrated_errortype_disposition': 'unreached when make stops at golangci recipe; otherwise no separate invocation trace',
              'lint_source_mapping_reference': git('rev-parse', 'a1908ed759').decode().strip(),
              'source_mapping_input_sha256': source_bindings,
              'retained_source_manifest_sha256': sha(BASELINE / 'final-integrated-lint-source.sha256'),
              'baseline_log_sha256': sha(BASELINE / 'final-integrated-lint.log'),
              'current_log_sha256': sha(PACKET / 'integrated-lint.log')}
    save('lint-comparison.json', report)
    return report

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--lane-granted', required=True, choices=['root-explicit-combined64-65'])
    parser.add_argument('--frozen-head', required=True)
    args = parser.parse_args()
    os.chdir(ROOT)
    head = git('rev-parse', 'HEAD').decode().strip()
    if head != args.frozen_head:
        raise ValueError('Frozen joined65 HEAD mismatch')
    if sha(PRIMARY_SPEC) != SPEC_SHA:
        raise ValueError('Primary owner SPEC.md hash mismatch')
    if (PACKET / 'run-binding.json').exists():
        raise ValueError('Combined gates already started; raw receipts cannot be overwritten')
    env = dict(os.environ)
    removed = sorted(key for key in env if key.startswith('GO') or key.startswith('CGO') or key in ('BASH_ENV', 'ENV', 'MAKEFLAGS', 'MFLAGS'))
    for key in removed:
        env.pop(key, None)
    env.update({'PATH': str(GO_ROOT / 'bin') + ':/usr/bin:/bin',
                'GOCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache',
                'GOMODCACHE': '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
                'TMPDIR': '/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp',
                'GOPROXY': 'file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download',
                'GOSUMDB': 'off', 'GOTOOLCHAIN': 'local', 'GOENV': 'off', 'GOWORK': 'off',
                'GOFLAGS': '', 'TZ': 'UTC', 'SANDBOX_START_DIR': str(ROOT),
                'GOMAD3_STOCK_GO': str(GO_ROOT / 'bin/go')})
    for key in ('GOCACHE', 'GOMODCACHE', 'TMPDIR'):
        if not Path(env[key]).is_dir():
            raise ValueError('Required existing directory absent: ' + key)
    before = inputs()
    if before['missing_tracked_relevant_paths']:
        raise ValueError('Tracked relevant source absent: ' + repr(before['missing_tracked_relevant_paths']))
    save('source-before.json', before)
    settings_command = ['go', 'env', '-json', 'GOOS', 'GOARCH', 'GOROOT', 'CGO_ENABLED', 'CC', 'CXX', 'GOVERSION', 'GOCACHE', 'GOMODCACHE']
    settings_start = time.monotonic()
    settings = subprocess.run(settings_command, cwd=ROOT, env=env, capture_output=True, timeout=30)
    save('actual-go-settings.json', {'argv': settings_command, 'exit': settings.returncode,
         'elapsed_seconds': time.monotonic() - settings_start, 'stdout': settings.stdout.decode(),
         'stderr': settings.stderr.decode(), 'head': head, 'source_before_after_equal': inputs() == before,
         'handle_terminal': True})
    settings.check_returncode()
    actual_go = json.loads(settings.stdout)
    tool_paths = [GO_ROOT / 'bin/go', GO_ROOT / 'bin/gofmt', GO_ROOT / 'VERSION',
                  *sorted((GO_ROOT / 'pkg/tool/linux_arm64').glob('*')),
                  TOOLS / 'golangci-lint-v2.13.0', TOOLS / 'errortype',
                  Path(shutil.which('make', path=env['PATH'])), Path(shutil.which('git', path=env['PATH'])),
                  Path(shutil.which('timeout', path=env['PATH'])), Path(sys.executable)]
    for variable in ('CC', 'CXX'):
        command = shlex.split(actual_go[variable])
        compiler = shutil.which(command[0], path=env['PATH']) if command else None
        if compiler:
            tool_paths.append(Path(compiler))
    tools = {str(path): sha(path) for path in tool_paths if path.is_file()}
    save('tools.json', tools)
    selected_env = {key: value for key, value in env.items() if key.startswith(('GO', 'CGO')) or key in ('PATH', 'TMPDIR', 'TZ', 'SANDBOX_START_DIR')}
    save('environment.json', {'used': selected_env, 'cleared_ambient_names': removed})
    baseline_env = dict(line.split('=', 1) for line in (BASELINE / 'final-ordinary-env.txt').read_text().splitlines() if '=' in line)
    save('ordinary-environment-comparison.json', {
         'baseline_environment_sha256': sha(BASELINE / 'final-ordinary-env.txt'),
         'current_environment_sha256': sha(PACKET / 'environment.json'),
         'recorded_setting_differences': {key: {'baseline': baseline_env.get(key), 'current': selected_env.get(key)}
          for key in sorted(baseline_env.keys() | selected_env.keys()) if baseline_env.get(key) != selected_env.get(key)},
         'current_actual_go_settings_sha256': sha(PACKET / 'actual-go-settings.json'),
         'cgo_limit': 'Historical ordinary receipt left CGO_ENABLED unset and did not retain its effective default; current ordinary and lint gates clear ambient CGO settings and use the captured Go default. Historical effective settings remain unknown.',
         'command_difference': 'Historical ordinary command also selected CLI and campaign packages and used -timeout=8m; current frozen command selects only Runner with the requested argv.'})
    save('run-binding.json', {'head': head, 'cwd': str(ROOT), 'wrapper_sha256': sha(__file__),
         'source_manifest_sha256': sha(PACKET / 'source-before.json'), 'source_count': len(before['files']),
         'relevant_source_count': before['relevant_source_count'],
         'original_inventory_plus_new_source_count': before['original_inventory_plus_new_source_count'],
         'tools_manifest_sha256': sha(PACKET / 'tools.json'), 'environment_sha256': sha(PACKET / 'environment.json'),
         'primary_owner_spec_path': str(PRIMARY_SPEC), 'primary_owner_spec_sha256': sha(PRIMARY_SPEC),
         'joined_historical_owner_spec_sha256': sha(ROOT / SPEC), 'prior_manifest_sha256': sha(OLD_MANIFEST),
         'host': {'system': platform.system(), 'machine': platform.machine()},
         'free_bytes_start': shutil.disk_usage(ROOT).free,
         'limits': ['Existing stock Go installation; tool binaries hashed, entire Go installation not recursively hashed.',
                    'Existing build and module caches reused; complete cache contents not independently requalified.',
                    'Ordinary and lint gates use captured default CGO settings; cross-source vet explicitly uses CGO_ENABLED=0. C compiler executables are hashed when resolvable, but C headers, libc/tool installation internals and complete C inputs are not inventoried.',
                    'No patched native toolchain installed or qualified; linux/arm64 host supplies no supported-native test-host pass.',
                    'Required ordinary Runner and original-base lint failures remain source-owned; transferred native owners stay deferred.']})
    commands = [
        ('ordinary-runner', ['go', '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-json', './runner'], {}),
        ('integrated-lint', ['make', 'lint-code-gomad3', 'GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c',
         'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + str(TOOLS / 'golangci-lint-v2.13.0'),
         'ERRORTYPE=' + str(TOOLS / 'errortype'), 'ALL_TEST_TAGS=test_dep'], {}),
        ('vet-darwin-arm64', ['go', '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', './runner', './runner/internal/execution'], {'GOOS': 'darwin', 'GOARCH': 'arm64', 'CGO_ENABLED': '0'}),
        ('vet-linux-amd64', ['go', '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', './runner', './runner/internal/execution'], {'GOOS': 'linux', 'GOARCH': 'amd64', 'CGO_ENABLED': '0'}),
    ]
    receipts = []
    for name, command, overrides in commands:
        if inputs() != before or git('rev-parse', 'HEAD').decode().strip() != head:
            raise ValueError('Source changed before gate ' + name)
        started = time.monotonic()
        utc_start = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with (PACKET / (name + '.log')).open('xb') as log:
            process = subprocess.Popen(['timeout', '--signal=TERM', '--kill-after=15s', '900s', *command],
                                       cwd=ROOT, env=env | overrides, stdout=log, stderr=subprocess.STDOUT)
            code = process.wait()
        stable = inputs() == before and git('rev-parse', 'HEAD').decode().strip() == head
        receipt = {'name': name, 'argv': command, 'outer_timeout_seconds': 900, 'exit': code,
                   'utc_start': utc_start, 'utc_end': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   'elapsed_seconds': time.monotonic() - started, 'source_before_after_equal': stable,
                   'environment_overrides': overrides, 'log_sha256': sha(PACKET / (name + '.log')),
                   'run_binding_sha256': sha(PACKET / 'run-binding.json'), 'handle_terminal': True}
        save(name + '.json', receipt)
        receipts.append(receipt)
        print(f'{name}: exit={code} source_stable={stable}', flush=True)
        if not stable:
            raise ValueError('Source changed during gate ' + name)
    save('source-after.json', inputs())
    ordinary = compare_outcomes()
    lint = compare_lint()
    save('summary.json', {'head': head, 'source_count': len(before['files']), 'receipts': receipts,
         'relevant_source_count': before['relevant_source_count'],
         'original_inventory_plus_new_source_count': before['original_inventory_plus_new_source_count'],
         'ordinary_counts': ordinary['current_counts'], 'unadmitted_original_changes': ordinary['unadmitted_original_changes'],
         'new_controls': ordinary['new_control_outcomes'], 'newly_reached_original_subtests': ordinary['newly_reached_original_subtests'],
         'unadmitted_newly_reached_original_subtests': ordinary['unadmitted_newly_reached_original_subtests'], 'lint_count': lint['current_count'],
         'lint_introduced_count': len(lint['introduced']), 'lint_removed_count': len(lint['removed']),
         'lint_mapping_gaps': lint['mapping_gaps'], 'all_handles_terminal': True,
         'execution_lane_released_to_root': True, 'native_qualification': 'unverified'})

if __name__ == '__main__':
    main()
