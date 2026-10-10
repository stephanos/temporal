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

ROOT = Path('/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/unix-mode')
PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PACKET = PRIMARY / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-73'
ARTIFACTS = PACKET.parent
WORKER_PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-73'
BASELINE = PRIMARY / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-72'
REFERENCE = None
BASELINE_SEAL_SHA = None
SCOPES = ('tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'cmd/tools/lintcode')
OLD_MANIFEST = PRIMARY / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
SPEC = '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'
PRIMARY_SPEC = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal') / SPEC
SPEC_SHA = '851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c'
ADMITTED = {'TestRunEnforcesBatchModesIndependentOfUmask'}
ADMITTED_TOPS = set(ADMITTED)
RUNNER = 'go.temporal.io/server/tools/gomad3/runner'
ALLOWED_LINT_REMOVALS = set()
LOCAL_SPEC_SHA = '0866b495ef6150bd0341aa904358f5de49ab4977e88da25c67e9df1531106f8a'
RESEARCH_PATH = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/remaining-scripted-fixture-survey.md'
RESEARCH_SHA = '25d2f660a363458d4954d319556d280600dce10d17278fd180c3da42d5c8b1a7'
SEALED_OUTPUT_NAMES = (
    'run-gates.py', 'preparation-note.md', 'actual-go-settings.json',
    'source-before.json', 'source-after.json', 'tools.json', 'environment.json',
    'ordinary-environment-comparison.json', 'run-binding.json',
    'ordinary-runner.log', 'ordinary-runner.json', 'integrated-lint.log', 'integrated-lint.json',
    'vet-darwin-arm64.log', 'vet-darwin-arm64.json', 'vet-linux-amd64.log', 'vet-linux-amd64.json',
    'outcome-comparison.json', 'lint-comparison.json', 'summary.json',
)

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def make_tool_routes(path, make_search_path):
    make = shutil.which('make', path=make_search_path)
    make_sha = 'b12eeb672d64e798b84f297c116651ccbb3ca726a108c74ffe9a60d85547315d'
    if make != '/usr/bin/make' or sha(make) != make_sha:
        raise ValueError('Make shell-function PATH routing requires the bound /usr/bin/make 4.4.1 executable')
    shell = Path('/bin/sh')
    grep = shutil.which('grep', path=path)
    if not grep:
        raise ValueError('Make eager grep executable is absent from its actual search path')
    find = shutil.which('find', path=path)
    if not find:
        raise ValueError('Make eager find executable is absent from its actual search path')
    return {'make_search_path': make_search_path, 'make_selected_path': make,
            'make_resolved_target': str(Path(make).resolve(strict=True)), 'make_sha256': make_sha,
            'make_version_bound_by_sha256': '4.4.1',
            'shell_function_search_path': path,
            'exported_path_reaches_shell_function_for_bound_make': True,
            'shell_path': str(shell),
            'shell_link_target': os.readlink(shell) if shell.is_symlink() else None,
            'shell_resolved_target': str(shell.resolve(strict=True)),
            'grep_search_path': path, 'grep_selected_path': grep,
            'grep_resolved_target': str(Path(grep).resolve(strict=True)),
            'find_search_path': path, 'find_selected_path': find,
            'find_resolved_target': str(Path(find).resolve(strict=True))}

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def save(name, value):
    with (PACKET / name).open('x') as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')

def verify_baseline_seal():
    seal_path = BASELINE / 'postcapture-seal.json'
    if sha(seal_path) != BASELINE_SEAL_SHA:
        raise ValueError('Authoritative combined72 postcapture seal bytes changed')
    seal = json.loads(seal_path.read_text())
    if seal.get('head') != REFERENCE or seal.get('explicit_member_count') != 20:
        raise ValueError('Baseline postcapture seal execution identity or member count mismatch')
    if set(seal['files_sha256']) != set(SEALED_OUTPUT_NAMES):
        raise ValueError('Baseline postcapture seal must contain exactly the fixed 20 members')
    for name, digest in seal['files_sha256'].items():
        if sha(BASELINE / name) != digest:
            raise ValueError('Baseline postcapture member mismatch: ' + name)
    binding = json.loads((BASELINE / 'run-binding.json').read_text())
    if sha(BASELINE / 'actual-go-settings.json') != binding['actual_go_settings_pre_gate_sha256']:
        raise ValueError('Baseline actual Go settings differ from the pre-gate binding')
    if sha(BASELINE / 'ordinary-environment-comparison.json') != binding['environment_comparison_pre_gate_sha256']:
        raise ValueError('Baseline environment comparison differs from the pre-gate binding')
    if seal.get('comparison_valid') is not True or seal.get('source_before_after_equal') is not True or seal.get('all_handles_terminal') is not True:
        raise ValueError('Baseline comparisons, source stability and terminal handles must be verified')
    if seal.get('metadata_matches_pre_gate_binding') != {'actual_go_settings': True, 'environment_comparison': True}:
        raise ValueError('Baseline metadata prebinding is not verified')
    if seal.get('execution_input_binding_sha256') != sha(BASELINE / 'run-binding.json') or binding.get('head') != REFERENCE:
        raise ValueError('Baseline seal does not match its actual execution input binding')
    source_sha = sha(BASELINE / 'source-before.json')
    if source_sha != binding['source_manifest_sha256'] or sha(BASELINE / 'source-after.json') != source_sha:
        raise ValueError('Baseline full source manifests are not identical and execution-bound')
    tools_sha = sha(BASELINE / 'tools.json')
    if tools_sha != binding['tools_manifest_sha256'] or sha(BASELINE / 'environment.json') != binding['environment_sha256']:
        raise ValueError('Baseline tool or environment manifest is not execution-bound')
    for name in ('ordinary-runner', 'integrated-lint', 'vet-darwin-arm64', 'vet-linux-amd64'):
        receipt = json.loads((BASELINE / (name + '.json')).read_text())
        if (receipt.get('source_before_after_equal') is not True or receipt.get('handle_terminal') is not True
                or receipt.get('source_before_manifest_sha256') != source_sha
                or receipt.get('source_after_manifest_sha256') != source_sha
                or receipt.get('tools_before_after_equal') is not True
                or receipt.get('tools_before_manifest_sha256') != tools_sha
                or receipt.get('tools_after_manifest_sha256') != tools_sha
                or receipt.get('run_binding_sha256') != sha(BASELINE / 'run-binding.json')
                or receipt.get('log_sha256') != sha(BASELINE / (name + '.log'))):
            raise ValueError('Baseline gate input or raw receipt mismatch: ' + name)
    return seal

def manifest_digest(value):
    return hashlib.sha256((json.dumps(value, indent=2, sort_keys=True) + '\n').encode()).hexdigest()

def source_path(path):
    path = Path(path)
    if path.is_absolute():
        try:
            return str(path.relative_to(ROOT))
        except ValueError:
            return str(path)
    return str(path)

def inputs():
    baseline_seal = verify_baseline_seal()
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
    paths.update(str(path) for path in [
        PRIMARY_SPEC, Path(__file__).resolve(), PACKET / 'preparation-note.md', OLD_MANIFEST,
        ARTIFACTS / 'task-73/admission.md', ARTIFACTS / 'task-73/preparation-note.md', WORKER_PACKET / 'admission.md',
        ROOT / '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md',
        PRIMARY / '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md',
        ROOT / '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.json',
        PRIMARY / '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.json',
        ROOT / RESEARCH_PATH, PRIMARY / RESEARCH_PATH,
        BASELINE / 'run-gates.py', BASELINE / 'run-binding.json',
        BASELINE / 'ordinary-runner.log', BASELINE / 'ordinary-runner.json',
        BASELINE / 'integrated-lint.log', BASELINE / 'integrated-lint.json',
        BASELINE / 'source-before.json', BASELINE / 'source-after.json',
        BASELINE / 'environment.json', BASELINE / 'actual-go-settings.json',
        BASELINE / 'tools.json', BASELINE / 'outcome-comparison.json',
        BASELINE / 'lint-comparison.json', BASELINE / 'summary.json',
        BASELINE / 'postcapture-seal.json',
    ])
    if not WORKER_PACKET.is_dir():
        raise ValueError('Final task73 worker packet is absent')
    worker_files = sorted(path for path in WORKER_PACKET.rglob('*') if path.is_file())
    paths.update(str(path) for path in worker_files)
    paths.update(str(BASELINE / name) for name in baseline_seal['files_sha256'])
    paths = {source_path(path) for path in paths}
    present = {path: sha(ROOT / path) for path in sorted(paths) if (ROOT / path).is_file()}
    absent = sorted(path for path in paths if not (ROOT / path).is_file())
    tracked = set(git('ls-files', '-z', '--', 'tools/gomad3', 'tools/gomad3sim',
                      'tools/gomad3integration', 'cmd/tools/lintcode').decode().strip('\0').split('\0'))
    reference_sources = {path: hashlib.sha256(git('show', REFERENCE + ':' + path)).hexdigest()
                         for path in sorted({block['path'] for block in lint_blocks(BASELINE / 'integrated-lint.log')})}
    return {'files': present, 'absent_inventory_paths': absent, 'relevant_source_count': relevant_count,
            'original_inventory_plus_new_source_count': retained_scope_count,
            'lint_comparison_reference_commit': git('rev-parse', REFERENCE).decode().strip(),
            'lint_comparison_reference_source_sha256': reference_sources,
            'baseline_postcapture_seal_sha256': sha(BASELINE / 'postcapture-seal.json'),
            'baseline_postcapture_seal_verified_members': len(baseline_seal['files_sha256']),
            'task73_worker_packet_file_count': len(worker_files),
            'task73_worker_packet_paths': [source_path(path) for path in worker_files],
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
    old, old_unparsable = outcomes(BASELINE / 'ordinary-runner.log')
    new, new_unparsable = outcomes(PACKET / 'ordinary-runner.log')
    changes = [{'test': name, 'before': old[name], 'after': new.get(name)}
               for name in sorted(old) if old[name] != new.get(name)]
    allowed = lambda name: name in ADMITTED
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
              'baseline_log_sha256': sha(BASELINE / 'ordinary-runner.log'),
              'current_log_sha256': sha(PACKET / 'ordinary-runner.log')}
    expected_changes = [{'test': name, 'before': 'fail', 'after': 'pass'} for name in sorted(ADMITTED)]
    report['comparison_valid'] = (
        changes == expected_changes and not report['missing_original_outcomes']
        and not report['unadmitted_original_changes']
        and not report['unadmitted_newly_reached_original_subtests']
        and not old_unparsable and not new_unparsable
        and not additions and len(old) == len(new) == 673
    )
    report['comparison_requirement'] = 'Exactly the one emitted original parent TestRunEnforcesBatchModesIndependentOfUmask changes fail to pass; all 673 original names remain present; no new names or other outcome changes are admitted.'
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
    old = lint_blocks(BASELINE / 'integrated-lint.log')
    new = lint_blocks(PACKET / 'integrated-lint.log')
    retained_hashes = json.loads((BASELINE / 'source-before.json').read_text())['files']
    mappings, gaps, mapped, source_bindings = [], [], [], {}
    for block in old:
        path = block['path']
        original = git('show', REFERENCE + ':' + path)
        source_bindings[path] = {'reference_sha256': hashlib.sha256(original).hexdigest(),
                                 'retained_combined72_source_sha256': retained_hashes.get(path),
                                 'current_sha256': sha(ROOT / path)}
        if hashlib.sha256(original).hexdigest() != retained_hashes.get(path):
            gaps.append({'path': path, 'reason': 'reference does not match retained combined72 actual source hash'})
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
    unauthorized_removed = [block for block in removed
                            if (HEADER.match(block.splitlines()[0])[1], HEADER.match(block.splitlines()[0])[4].split(':', 1)[0])
                            not in ALLOWED_LINT_REMOVALS]
    admitted_baseline_blocks = [block['block'] for block in old
                               if (block['path'], block['message'].split(':', 1)[0]) in ALLOWED_LINT_REMOVALS]
    report = {'baseline_count': len(old), 'current_count': len(new),
              'baseline_full_blocks': [block['block'] for block in old],
              'current_full_blocks': [block['block'] for block in new],
              'preserved': preserved, 'exact_source_line_mappings': mappings,
              'removed': removed, 'introduced': introduced, 'mapping_gaps': gaps,
              'admitted_removal_locations_and_codes': sorted(ALLOWED_LINT_REMOVALS),
              'unadmitted_removed_full_blocks': unauthorized_removed,
              'admitted_baseline_full_blocks_for_removal': admitted_baseline_blocks,
              'integrated_errortype_reached': False if re.search(r'Makefile:505: lint-code\] Error', (PACKET / 'integrated-lint.log').read_text()) else None,
              'integrated_errortype_disposition': 'unreached when make stops at golangci recipe; otherwise no separate invocation trace',
              'lint_source_mapping_reference': git('rev-parse', REFERENCE).decode().strip(),
              'source_mapping_input_sha256': source_bindings,
              'retained_source_manifest_sha256': sha(BASELINE / 'source-before.json'),
              'baseline_log_sha256': sha(BASELINE / 'integrated-lint.log'),
              'current_log_sha256': sha(PACKET / 'integrated-lint.log')}
    report['comparison_valid'] = (
        not removed and not admitted_baseline_blocks
        and not introduced and not unauthorized_removed and not gaps
    )
    save('lint-comparison.json', report)
    return report

def main():
    global REFERENCE, BASELINE_SEAL_SHA
    parser = argparse.ArgumentParser()
    parser.add_argument('--lane-granted', required=True, choices=['root-explicit-combined73'])
    parser.add_argument('--frozen-head', required=True)
    parser.add_argument('--baseline-frozen-head', required=True)
    parser.add_argument('--baseline-seal-sha256', required=True)
    args = parser.parse_args()
    if not re.fullmatch(r'[0-9a-f]{40}', args.baseline_frozen_head) or not re.fullmatch(r'[0-9a-f]{64}', args.baseline_seal_sha256):
        raise ValueError('Root must provide actual full baseline execution HEAD and seal SHA-256')
    REFERENCE = args.baseline_frozen_head
    BASELINE_SEAL_SHA = args.baseline_seal_sha256
    if not ROOT.is_dir() or ROOT.resolve() != ROOT:
        raise ValueError('Root must materialize the exact nonsymlink Unix mode worktree before execution')
    os.chdir(ROOT)
    if Path(git('rev-parse', '--show-toplevel').decode().strip()) != ROOT:
        raise ValueError('Unix mode worktree top-level path mismatch')
    head = git('rev-parse', 'HEAD').decode().strip()
    if head != args.frozen_head:
        raise ValueError('Frozen combined73 HEAD mismatch')
    product_changes = git('diff', 'HEAD', '--name-only', '--', *SCOPES).decode().splitlines()
    untracked_products = git('ls-files', '--others', '--exclude-standard', '--', *SCOPES).decode().splitlines()
    if product_changes or untracked_products:
        raise ValueError('Root must checkpoint task73 before execution: ' + repr(product_changes + untracked_products))
    admission_path = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-73/admission.md'
    committed_admission = git('show', head + ':' + admission_path)
    admitted_names = set(re.findall(r'^Add exactly one syntactic assignment using the existing helper in `(Test[^`]+)`:', committed_admission.decode(), re.MULTILINE))
    if admitted_names != ADMITTED_TOPS or hashlib.sha256(committed_admission).hexdigest() != sha(ROOT / admission_path):
        raise ValueError('One-call Unix mode admission does not match the committed root checkpoint')
    if sha(PRIMARY_SPEC) != SPEC_SHA:
        raise ValueError('Primary owner SPEC.md hash mismatch')
    if sha(ROOT / SPEC) != LOCAL_SPEC_SHA:
        raise ValueError('Isolated historical owner SPEC.md hash mismatch')
    if sha(ROOT / RESEARCH_PATH) != RESEARCH_SHA or sha(PRIMARY / RESEARCH_PATH) != RESEARCH_SHA:
        raise ValueError('Unix mode fixture research identity mismatch')
    occupied = [name for name in (*SEALED_OUTPUT_NAMES[2:], 'postcapture-seal.json') if (PACKET / name).exists()]
    if occupied:
        raise ValueError('Capture outputs already exist; no overwrite or retry: ' + repr(occupied))
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
    baseline_names, baseline_non_json = outcomes(BASELINE / 'ordinary-runner.log')
    verify_baseline_seal()
    if len(baseline_names) != 673 or baseline_non_json or any(baseline_names.get(name) != 'fail' for name in ADMITTED):
        raise ValueError('Authoritative baseline must contain 673 actual outcomes and the one admitted parent failure')
    if len(lint_blocks(BASELINE / 'integrated-lint.log')) != 50:
        raise ValueError('Authoritative lint baseline differs from retained 50 blocks')
    baseline_binding = json.loads((BASELINE / 'run-binding.json').read_text())
    if baseline_binding['head'] != REFERENCE or sha(BASELINE / 'source-before.json') != baseline_binding['source_manifest_sha256']:
        raise ValueError('Baseline source execution binding mismatch')
    for name in ('ordinary-runner', 'integrated-lint'):
        receipt = json.loads((BASELINE / (name + '.json')).read_text())
        if sha(BASELINE / (name + '.log')) != receipt['log_sha256']:
            raise ValueError('Baseline raw log differs from its receipt: ' + name)
    baseline_test_source = 'tools/gomad3/runner/runner_mode_unix_test.go'
    admitted_source_sha = 'ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168'
    retained_sources = json.loads((BASELINE / 'source-before.json').read_text())['files']
    if hashlib.sha256(git('show', REFERENCE + ':' + baseline_test_source)).hexdigest() != admitted_source_sha or retained_sources.get(baseline_test_source) != admitted_source_sha:
        raise ValueError('Baseline Unix mode fixture differs from the admitted unchanged source')
    baseline_source = git('show', REFERENCE + ':' + baseline_test_source)
    attachment = b'\tconfigDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)\n'
    location = b'\tdefer syscall.Umask(oldUmask)\n\n'
    if baseline_source.count(location) != 1 or attachment in baseline_source:
        raise ValueError('Baseline Unix mode attachment location differs from admitted source')
    candidate_source = (ROOT / baseline_test_source).read_bytes()
    expected_source = baseline_source.replace(location, location + attachment, 1)
    if candidate_source != expected_source or candidate_source.count(attachment) != 1:
        raise ValueError('Unix mode candidate must contain exactly the admitted call-local insertion and no other source change')
    for path in sorted({block['path'] for block in lint_blocks(BASELINE / 'integrated-lint.log')}):
        if hashlib.sha256(git('show', REFERENCE + ':' + path)).hexdigest() != retained_sources.get(path):
            raise ValueError('Lint reference source does not match actual baseline manifest: ' + path)
    before = inputs()
    if before['absent_inventory_paths']:
        raise ValueError('Required manifested source or comparison input absent: ' + repr(before['absent_inventory_paths']))
    save('source-before.json', before)
    settings_command = ['go', 'env', '-json', 'GOOS', 'GOARCH', 'GOROOT', 'CGO_ENABLED', 'CC', 'CXX', 'GOVERSION', 'GOCACHE', 'GOMODCACHE']
    settings_start = time.monotonic()
    settings_utc_start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    settings = subprocess.run(settings_command, cwd=ROOT, env=env, capture_output=True, timeout=30)
    settings_after = inputs()
    save('actual-go-settings.json', {'argv': settings_command, 'exit': settings.returncode,
         'utc_start': settings_utc_start, 'utc_end': datetime.datetime.now(datetime.timezone.utc).isoformat(),
         'elapsed_seconds': time.monotonic() - settings_start, 'stdout': settings.stdout.decode(),
         'stderr': settings.stderr.decode(), 'head': head, 'source_before_after_equal': settings_after == before,
         'source_before_manifest_sha256': manifest_digest(before),
         'source_after_manifest_sha256': manifest_digest(settings_after),
         'handle_terminal': True})
    settings.check_returncode()
    if settings_after != before:
        raise ValueError('Source changed during effective Go settings capture')
    actual_go = json.loads(settings.stdout)
    make_path = str(ROOT / '.bin') + ':' + env['PATH']
    routing = make_tool_routes(make_path, env['PATH'])
    tool_paths = [GO_ROOT / 'bin/go', GO_ROOT / 'bin/gofmt', GO_ROOT / 'VERSION',
                  *sorted((GO_ROOT / 'pkg/tool/linux_arm64').glob('*')),
                  TOOLS / 'golangci-lint-v2.13.0', TOOLS / 'errortype',
                  Path(shutil.which('make', path=env['PATH'])), Path(shutil.which('git', path=env['PATH'])),
                  Path(shutil.which('timeout', path=env['PATH'])), Path(sys.executable),
                  Path('/bin/sh'), Path(routing['shell_resolved_target']),
                  Path(routing['grep_selected_path']), Path(routing['grep_resolved_target']),
                  Path(routing['find_selected_path']), Path(routing['find_resolved_target'])]
    for variable in ('CC', 'CXX'):
        command = shlex.split(actual_go[variable])
        compiler = shutil.which(command[0], path=env['PATH']) if command else None
        if compiler:
            tool_paths.append(Path(compiler))
    tools = {str(path): sha(path) for path in tool_paths if path.is_file()}
    save('tools.json', tools)
    selected_env = {key: value for key, value in env.items() if key.startswith(('GO', 'CGO')) or key in ('PATH', 'TMPDIR', 'TZ', 'SANDBOX_START_DIR')}
    save('environment.json', {'used': selected_env, 'cleared_ambient_names': removed})
    baseline_env = json.loads((BASELINE / 'environment.json').read_text())['used']
    baseline_tools = json.loads((BASELINE / 'tools.json').read_text())
    baseline_make_argv = json.loads((BASELINE / 'integrated-lint.json').read_text())['argv']
    baseline_actual_go = json.loads(json.loads((BASELINE / 'actual-go-settings.json').read_text())['stdout'])
    baseline_argv = json.loads((BASELINE / 'ordinary-runner.json').read_text())['argv']
    current_argv = ['go', '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-json', './runner']
    save('ordinary-environment-comparison.json', {
         'baseline_tools_manifest_sha256': sha(BASELINE / 'tools.json'),
         'current_tools_manifest_sha256': sha(PACKET / 'tools.json'),
         'recorded_tool_identity_differences': {path: {'baseline': baseline_tools.get(path), 'current': tools.get(path)}
           for path in sorted(baseline_tools.keys() | tools.keys()) if baseline_tools.get(path) != tools.get(path)},
         'current_make_tool_routes': routing,
         'baseline_make_tool_routes': baseline_binding.get('make_tool_routes'),
         'baseline_integrated_lint_argv': baseline_make_argv,
         'current_explicit_make_shell_argument': 'SHELL=/bin/sh',
         'baseline_explicit_make_shell_arguments': [entry for entry in baseline_make_argv if entry.startswith('SHELL=')],
         'historical_selected_tools_limit': 'Retained historical tool manifests bind only their listed paths and retained route metadata. Missing shell routes, grep or find entries cannot establish historical execution identities; current additions do not retroactively bind older runs. Combined70 and its historical baseline omit find route and executable identities; present hashes or later seals cannot repair that execution-time omission.',
         'baseline_environment_sha256': sha(BASELINE / 'environment.json'),
         'current_environment_sha256': sha(PACKET / 'environment.json'),
         'recorded_setting_differences': {key: {'baseline': baseline_env.get(key), 'current': selected_env.get(key)}
          for key in sorted(baseline_env.keys() | selected_env.keys()) if baseline_env.get(key) != selected_env.get(key)},
         'current_actual_go_settings_sha256': sha(PACKET / 'actual-go-settings.json'),
         'baseline_actual_go_settings_sha256': sha(BASELINE / 'actual-go-settings.json'),
         'baseline_effective_go_settings': baseline_actual_go, 'current_effective_go_settings': actual_go,
         'effective_go_setting_differences': {key: {'baseline': baseline_actual_go.get(key), 'current': actual_go.get(key)}
           for key in sorted(baseline_actual_go.keys() | actual_go.keys()) if baseline_actual_go.get(key) != actual_go.get(key)},
         'baseline_argv': baseline_argv, 'current_argv': current_argv, 'argv_equal': baseline_argv == current_argv,
         'baseline_working_directory': json.loads((BASELINE / 'run-binding.json').read_text())['cwd'],
         'current_working_directory': str(ROOT),
         'historical_metadata_binding_limit': 'The combined72 baseline prebound actual Go settings and its environment comparison and retained the verified explicit 20-member postcapture seal. Later outcome/lint comparisons remain postcapture outputs; the seal makes no retroactive pre-execution output-binding claim.',
         'limits': 'Selective environment capture preserves unset CGO configuration and records effective defaults for both batches. Inherited nonselected variables and C headers are not bound; no hermetic execution claim.'})
    save('run-binding.json', {'head': head, 'baseline_frozen_head': REFERENCE,
         'baseline_postcapture_seal_sha256': BASELINE_SEAL_SHA,
         'make_tool_routes': routing, 'make_tool_routes_pre_gate_sha256': manifest_digest(routing),
         'baseline_named_counts': dict(collections.Counter(baseline_names.values())), 'cwd': str(ROOT), 'wrapper_sha256': sha(__file__), 'wrapper_argv': sys.argv,
         'source_manifest_sha256': sha(PACKET / 'source-before.json'), 'source_count': len(before['files']),
         'relevant_source_count': before['relevant_source_count'],
         'original_inventory_plus_new_source_count': before['original_inventory_plus_new_source_count'],
         'tools_manifest_sha256': sha(PACKET / 'tools.json'), 'environment_sha256': sha(PACKET / 'environment.json'),
         'actual_go_settings_pre_gate_sha256': sha(PACKET / 'actual-go-settings.json'),
         'environment_comparison_pre_gate_sha256': sha(PACKET / 'ordinary-environment-comparison.json'),
         'primary_owner_spec_path': str(PRIMARY_SPEC), 'primary_owner_spec_sha256': sha(PRIMARY_SPEC),
         'joined_historical_owner_spec_sha256': sha(ROOT / SPEC), 'prior_manifest_sha256': sha(OLD_MANIFEST),
         'host': {'system': platform.system(), 'machine': platform.machine()},
         'free_bytes_start': shutil.disk_usage(ROOT).free,
         'limits': ['Existing stock Go installation; tool binaries hashed, entire Go installation not recursively hashed.',
                    'Existing build and module caches reused; complete cache contents not independently requalified.',
                    'Only selected environment variables are recorded; other inherited variables are not bound and execution is not claimed hermetic.',
                    'Ordinary and lint gates use captured default CGO settings; cross-source vet explicitly uses CGO_ENABLED=0. C compiler executables are hashed when resolvable, but C headers, libc/tool installation internals and complete C inputs are not inventoried.',
                    'No patched native toolchain installed or qualified; linux/arm64 host supplies no supported-native test-host pass.',
                    'Any observed ordinary Runner or original-base lint failures remain source-owned; transferred native owners stay deferred.']})
    commands = [
        ('ordinary-runner', ['go', '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-json', './runner'], {}),
        ('integrated-lint', ['make', 'lint-code-gomad3', 'SHELL=/bin/sh', 'GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c',
         'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + str(TOOLS / 'golangci-lint-v2.13.0'),
         'ERRORTYPE=' + str(TOOLS / 'errortype'), 'ALL_TEST_TAGS=test_dep'], {}),
        ('vet-darwin-arm64', ['go', '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', './runner', './internal/gomadtool/conformance', './runner/internal/execution'], {'GOOS': 'darwin', 'GOARCH': 'arm64', 'CGO_ENABLED': '0'}),
        ('vet-linux-amd64', ['go', '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', './runner', './internal/gomadtool/conformance', './runner/internal/execution'], {'GOOS': 'linux', 'GOARCH': 'amd64', 'CGO_ENABLED': '0'}),
    ]
    receipts = []
    for name, command, overrides in commands:
        command_before = inputs()
        tools_before = {path: sha(path) for path in tools}
        routing_before = make_tool_routes(make_path, env['PATH'])
        if command_before != before or tools_before != tools or routing_before != routing or git('rev-parse', 'HEAD').decode().strip() != head:
            raise ValueError('Source changed before gate ' + name)
        started = time.monotonic()
        utc_start = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with (PACKET / (name + '.log')).open('xb') as log:
            process = subprocess.Popen(['timeout', '--signal=TERM', '--kill-after=15s', '900s', *command],
                                       cwd=ROOT, env=env | overrides, stdout=log, stderr=subprocess.STDOUT)
            code = process.wait()
        command_after = inputs()
        tools_after = {path: sha(path) for path in tools}
        tools_stable = tools_after == tools_before == tools
        routing_after = make_tool_routes(make_path, env['PATH'])
        routing_stable = routing_after == routing_before == routing
        stable = command_after == command_before == before and tools_stable and routing_stable and git('rev-parse', 'HEAD').decode().strip() == head
        receipt = {'name': name, 'argv': command, 'outer_timeout_seconds': 900, 'exit': code,
                   'utc_start': utc_start, 'utc_end': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   'elapsed_seconds': time.monotonic() - started, 'source_before_after_equal': stable,
                   'environment_overrides': overrides, 'log_sha256': sha(PACKET / (name + '.log')),
                   'source_before_manifest_sha256': manifest_digest(command_before),
                   'source_after_manifest_sha256': manifest_digest(command_after),
                   'tools_before_manifest_sha256': manifest_digest(tools_before),
                   'tools_after_manifest_sha256': manifest_digest(tools_after),
                   'tools_before_after_equal': tools_stable,
                   'make_tool_routes_before': routing_before, 'make_tool_routes_after': routing_after,
                   'make_tool_routes_before_manifest_sha256': manifest_digest(routing_before),
                   'make_tool_routes_after_manifest_sha256': manifest_digest(routing_after),
                   'make_tool_routes_before_after_equal': routing_stable,
                   'run_binding_sha256': sha(PACKET / 'run-binding.json'), 'handle_terminal': True}
        save(name + '.json', receipt)
        receipts.append(receipt)
        print(f'{name}: exit={code} source_stable={stable}', flush=True)
        if not stable:
            raise ValueError('Source changed during gate ' + name)
    save('source-after.json', inputs())
    ordinary = compare_outcomes()
    lint = compare_lint()
    source_stable = inputs() == before and git('rev-parse', 'HEAD').decode().strip() == head
    comparison_valid = source_stable and ordinary['comparison_valid'] and lint['comparison_valid']
    save('summary.json', {'head': head, 'source_count': len(before['files']), 'receipts': receipts,
         'comparison_valid': comparison_valid, 'source_before_after_equal': source_stable,
         'ordinary_comparison_valid': ordinary['comparison_valid'], 'lint_comparison_valid': lint['comparison_valid'],
         'aggregate_gate_exits_all_zero': all(receipt['exit'] == 0 for receipt in receipts),
         'static_vet_exits_all_zero': all(receipt['exit'] == 0 for receipt in receipts if receipt['name'].startswith('vet-')),
         'relevant_source_count': before['relevant_source_count'],
         'original_inventory_plus_new_source_count': before['original_inventory_plus_new_source_count'],
         'ordinary_counts': ordinary['current_counts'], 'unadmitted_original_changes': ordinary['unadmitted_original_changes'],
         'new_controls': ordinary['new_control_outcomes'], 'newly_reached_original_subtests': ordinary['newly_reached_original_subtests'],
         'unadmitted_newly_reached_original_subtests': ordinary['unadmitted_newly_reached_original_subtests'], 'lint_count': lint['current_count'],
         'lint_introduced_count': len(lint['introduced']), 'lint_removed_count': len(lint['removed']),
         'lint_mapping_gaps': lint['mapping_gaps'], 'unadmitted_lint_removals': lint['unadmitted_removed_full_blocks'],
         'all_handles_terminal': True, 'root_owns_execution_lane_release': True,
         'native_qualification': 'unverified'})
    captured = {name: sha(PACKET / name) for name in SEALED_OUTPUT_NAMES}
    pre_gate_binding = json.loads((PACKET / 'run-binding.json').read_text())
    save('postcapture-seal.json', {'utc_created': datetime.datetime.now(datetime.timezone.utc).isoformat(),
         'head': head, 'files_sha256': captured,
         'explicit_member_count': len(SEALED_OUTPUT_NAMES), 'comparison_valid': comparison_valid,
         'scope': 'Postcapture hashes bind actual metadata, logs, per-command receipts and derived comparisons after capture. This is not a retroactive pre-execution binding of outputs.',
         'execution_input_binding_sha256': sha(PACKET / 'run-binding.json'),
         'metadata_matches_pre_gate_binding': {
             'actual_go_settings': sha(PACKET / 'actual-go-settings.json') == pre_gate_binding['actual_go_settings_pre_gate_sha256'],
             'environment_comparison': sha(PACKET / 'ordinary-environment-comparison.json') == pre_gate_binding['environment_comparison_pre_gate_sha256'],
         },
         'source_before_after_equal': inputs() == before, 'all_handles_terminal': True})
    if not comparison_valid:
        raise SystemExit(1)

if __name__ == '__main__':
    main()
