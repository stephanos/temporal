import datetime
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess
import sys
import urllib.parse

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
HERE = Path(__file__).resolve().parent
DOCS = [f'tools/gomad3/{name}.md' for name in ('README', 'CLI', 'SPEC', 'ARCHITECTURE', 'TUTORIAL')]
DOCS += ['.plans/GOMAD_CMP.md', '.plans/GOMAD_NEXT.md']
inputs = {}


def read(path):
    data = (ROOT / path).read_bytes()
    inputs[path] = hashlib.sha256(data).hexdigest()
    return data.decode()


def sha(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


registrations = {}
for directory in ('tools/gomad3/cmd/gomad/internal/cli', 'tools/gomad3/cmd/gomadtool'):
    for path in sorted((ROOT / directory).glob('*.go')):
        if path.name.endswith('_test.go'):
            continue
        source = read(str(path.relative_to(ROOT)))
        starts = list(re.finditer(r'flags := flag.NewFlagSet\("([^"]+)"', source))
        for index, start in enumerate(starts):
            body = source[start.end():starts[index + 1].start() if index + 1 < len(starts) else len(source)]
            flags = set()
            for match in re.finditer(r'flags\.(String|Bool|Uint64|Int|Duration|Var|StringVar|BoolVar|Uint64Var|IntVar|DurationVar)\(([^\n]*)', body):
                literal = re.search(r'"([a-z][a-z0-9_-]*)"', match.group(2))
                if literal:
                    flags.add(literal.group(1))
            registrations.setdefault(start.group(1), set()).update(flags)
registrations['gomad plan'] = registrations['gomad explore']
for name in ('patch-validate', 'patch-materialize'):
    registrations.setdefault('gomadtool ' + name, registrations.get('gomadtool ', set()))
all_flags = set().union(*registrations.values()) | {'help', 'provenance'}
errors, occurrences, examples, links = [], [], [], []
for path in DOCS:
    body = read(path)
    for line, text in enumerate(body.splitlines(), 1):
        for name in re.findall(r'(?<![\w-])--([a-z][a-z0-9_-]*)', text):
            occurrences.append({'path': path, 'line': line, 'flag': name})
            if name not in all_flags:
                errors.append({'path': path, 'line': line, 'unknown_flag': name})
    fences = re.findall(r'^```(?:sh|bash)\n(.*?)^```', body, re.M | re.S)
    for block in fences:
        for text in block.replace('\\\n', ' ').splitlines():
            if not re.search(r'(?:\.bin/gomad|\./cmd/gomadtool)\s', text):
                continue
            tokens = shlex.split(text, comments=True)
            start = next(i for i, token in enumerate(tokens) if token.endswith('.bin/gomad') or token == './cmd/gomadtool')
            tool = 'gomadtool' if tokens[start] == './cmd/gomadtool' else 'gomad'
            command = tool + ' ' + tokens[start + 1]
            offset = start + 2
            if command == 'gomadtool compatibility-pack':
                command += ' ' + tokens[offset]
                offset += 1
            flags = registrations.get(command)
            seen = []
            for token in tokens[offset:]:
                if token in ('--', 'go-run', 'go-test', 'exec'):
                    break
                if token.startswith('--'):
                    name = token[2:].split('=', 1)[0]
                    seen.append(name)
                    if flags is None or name not in flags:
                        errors.append({'path': path, 'command': command, 'unregistered_example_flag': name})
            examples.append({'path': path, 'command': command, 'flags': seen})
    for destination in re.findall(r'\[[^]]+\]\(([^)\s]+)\)', body):
        if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', destination):
            continue
        target = urllib.parse.unquote(destination.split('#', 1)[0])
        resolved = (ROOT / path).parent / target if target else ROOT / path
        links.append({'path': path, 'destination': destination, 'exists': resolved.exists()})
        if not resolved.exists():
            errors.append(links[-1])

qualification = {}
for directory in ('tools/gomad3/qualification', 'tools/gomad3integration/qualification'):
    for path in sorted((ROOT / directory).rglob('*.json')):
        relative = str(path.relative_to(ROOT))
        json.loads(read(relative))
        qualification[relative] = sha(relative)
prior_path = '.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-13/source-acceptance-20261007/source-proof.json'
prior = json.loads(read(prior_path))
bindings = json.loads(read(prior['bindings']['path']))
if sha(prior['bindings']['path']) != prior['bindings']['sha256']:
    errors.append({'changed_reference': prior['bindings']['path']})
for path, expected in bindings['scoped_bindings'].items():
    if sha(path) != expected:
        errors.append({'changed_preservation_input': path})
for reference in [prior['preservation'], *(item['receipt'] for item in prior['reused']), *prior['unchanged_primary_fixtures']]:
    if sha(reference['path']) != reference['sha256']:
        errors.append({'changed_reference': reference['path']})
    inputs[reference['path']] = sha(reference['path'])
    if reference['path'].endswith('.json'):
        json.loads((ROOT / reference['path']).read_text())
raw_reference = prior['retained_raw']['manifest']
raw_manifest = json.loads(read(raw_reference['path']))
if sha(raw_reference['path']) != raw_reference['sha256']:
    errors.append({'changed_reference': raw_reference['path']})
raw_root = (ROOT / raw_reference['path']).parent / 'raw'
for entry in raw_manifest:
    retained = (raw_root / entry.get('retained_name', entry['name'])).read_bytes()
    if entry.get('encoding') == 'utf8-json-string':
        container = json.loads(retained)
        if hashlib.sha256(retained).hexdigest() != entry['retained_sha256'] or container['encoding'] != entry['encoding']:
            errors.append({'changed_retained_container': entry['name']})
        payload = container['value'].encode()
    else:
        payload = retained
    if len(payload) != entry['bytes'] or hashlib.sha256(payload).hexdigest() != entry['sha256']:
        errors.append({'changed_retained_raw': entry['name']})
    if entry['name'].endswith('.json'):
        json.loads(payload)
for path in ('tools/gomad3/choice/no_op_select.go',
             'tools/gomad3/runner/internal/exploration/choice/select_readiness.go',
             'tools/gomad3/artifact/retained_bytes.go',
             'tools/gomad3/runner/internal/minimizer/workspace.go',
             'tools/gomad3/target/target.go',
             'tools/gomad3/toolchain/version/descriptor.go',
             'tools/gomad3/runner/internal/corpus/model.go',
             'tools/gomad3/runner/internal/corpus/corpus.go',
             'tools/gomad3/runner/internal/corpus/corpus_test.go',
             'tools/gomad3/runner/internal/corpus/guide_test.go',
             'tools/gomad3/cmd/gomad/internal/cli/characterization_test.go',
             'tools/gomad3/cmd/gomad/internal/cli/cli_test.go',
             'tools/gomad3/cmd/gomad/internal/cli/guidance_test.go',
             'tools/gomad3/target/target_test.go',
             'tools/gomad3/target/coverage_test.go',
             'tools/gomad3/runner/coverage_replay_test.go',
             'tools/gomad3/runner/internal/minimizer/workspace_unix_test.go',
             'tools/gomad3/runner/internal/exploration/choice/engine.go',
             'tools/gomad3/runner/choice_exploration_campaign.go',
             'tools/gomad3/runner/seeds.go',
             'tools/gomad3/runner/minimize_operation.go',
             'tools/gomad3/artifact/target_pool.go',
             'tools/gomad3/artifact/retained_bytes_test.go',
             'tools/gomad3/artifact/target_pool_test.go',
             'tools/gomad3/runner/internal/exploration/choice/engine_test.go',
             'tools/gomad3/runner/internal/exploration/choice/engine_divergence_test.go',
             'tools/gomad3/runner/choice_exploration_divergence_test.go',
             'tools/gomad3/runner/guided_selection_test.go',
             'tools/gomad3/runner/guidance.go',
             'tools/gomad3/runner/internal/campaign/retained_evidence.go',
             'tools/gomad3/runner/internal/campaign/retained_evidence_test.go',
             'tools/gomad3/runner/internal/campaign/merge_capacity_test.go',
             'tools/gomad3/runner/runner_test.go',
             'tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go',
             'tools/gomad3/toolchain/goroutine_inventory_test.go',
             'tools/gomad3/Makefile', 'Makefile',
             '.flow/artifacts/native-scope-transfer-2026-10-07.md'):
    read(path)
report = {'checked_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'inputs': inputs, 'qualification_manifests': qualification,
          'registrations': {key: sorted(value) for key, value in registrations.items()},
          'documented_flags': occurrences, 'examples': examples, 'links': links,
          'unchanged_preservation_inputs': len(bindings['scoped_bindings']),
          'verified_retained_raw': {'files': len(raw_manifest), 'bytes': sum(item['bytes'] for item in raw_manifest)},
          'prior_receipt': {'path': prior_path, 'sha256': sha(prior_path)},
          'reuse_limit': 'Only unchanged named code/runtime/identity inputs. The historical full-module snapshot is not current documentation evidence.',
          'errors': errors}
if len(sys.argv) > 1 and sys.argv[1] == 'baseline':
    output = HERE / 'baseline-docs.json'
else:
    output = HERE / 'final-docs.json'
    baseline = json.loads((HERE / 'baseline-docs.json').read_text())
    if qualification != baseline['qualification_manifests']:
        errors.append({'error': 'qualification manifest bytes changed'})
    changed = subprocess.run(['git', 'diff', '--name-only', (HERE / 'base_commit').read_text().strip(), '--', 'tools/gomad3'], cwd=ROOT, capture_output=True, text=True, check=True).stdout.splitlines()
    report['changed_module_paths'] = changed
    if set(changed) - set(DOCS):
        errors.append({'error': 'non-documentation module paths changed', 'paths': changed})
output.write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({'registered_commands': len(registrations), 'documented_flags': len(occurrences),
                  'examples': len(examples), 'manifests': len(qualification),
                  'preservation_inputs': len(bindings['scoped_bindings']), 'errors': errors}))
raise SystemExit(bool(errors))
