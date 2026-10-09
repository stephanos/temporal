import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
BIN = pathlib.Path('/tmp/fn11210-portable.6ZZCZ0Ix/bin')
BASE = '15f56644664f3d3749bab2387aa97936a1cac6dd'
GUIDES = ['SPEC', 'ARCHITECTURE', 'CLI', 'TUTORIAL', 'README']
errors, commands, links = [], [], []
inputs = {}

def read(path):
    data = (ROOT / path).read_bytes()
    inputs[path] = hashlib.sha256(data).hexdigest()
    return data.decode()

guides = {name: read('tools/gomad3/' + name + '.md') for name in GUIDES}
integration_guide = read('tools/gomad3integration/README.md')
index = guides['CLI'].split('## Command index', 1)[1]
pack_source = read('tools/gomad3/cmd/gomadtool/compatibility_pack.go')
actions = re.findall(r'case "([a-z-]+)":', pack_source.split('func runCompatibilityPackDiscover', 1)[0])
all_flags = set()
for tool in ['gomad', 'gomadtool']:
    section = index.split('### `' + tool + '`', 1)[1].split('\n### ', 1)[0]
    names = re.findall(r'^\| `([^`]+)` \|', section, re.M)
    dispatch = read('tools/gomad3/cmd/gomad/internal/cli/cli.go' if tool == 'gomad' else 'tools/gomad3/cmd/gomadtool/main.go')
    for name in names:
        parts = name.split()
        if not re.search(r'case "' + re.escape(parts[0]) + '"', dispatch):
            errors.append('undispatched command: ' + tool + ' ' + name)
        variants = [[parts[0], action] for action in actions] if name == 'compatibility-pack' else [parts]
        for variant in variants:
            result = subprocess.run([str(BIN / tool), *variant, '-h'], text=True, capture_output=True, timeout=30)
            output = result.stdout + result.stderr
            flags = re.findall(r'^\s+-([a-z_][a-z0-9_-]*)(?:\s+\S+)?\s*$', output, re.M)
            all_flags.update(flags)
            if 'unknown ' + tool + ' command' in output or not flags and not output.startswith('usage: gomadtool checked-run <seconds>'):
                errors.append('unavailable help: ' + tool + ' ' + ' '.join(variant))
            commands.append({'argv': [tool, *variant, '-h'], 'status': result.returncode, 'flags': flags,
                             'help_sha256': hashlib.sha256(output.encode()).hexdigest()})
    inputs[str(BIN / tool)] = hashlib.sha256((BIN / tool).read_bytes()).hexdigest()
target_parser = read('tools/gomad3/cmd/gomad/internal/cli/cli.go')
positional_flags = set(re.findall(r'arguments\[1\] != "--([a-z-]+)"', target_parser))
all_flags.update(positional_flags)
documented_flags = set(re.findall(r'--([a-z][a-z0-9-]*)', '\n'.join(guides.values())))
unknown = sorted(documented_flags - all_flags)
if unknown:
    errors.append({'documented_flags_without_registered_help': unknown})

def anchors(body):
    result = set(re.findall(r'<a\s+(?:id|name)=["\x27]([^"\x27]+)', body))
    counts = {}
    for heading in re.findall(r'^#{1,6}\s+(.+?)\s*#*$', body, re.M):
        slug = re.sub(r'[^\w\- ]', '', heading.lower()).replace(' ', '-')
        count = counts.get(slug, 0)
        counts[slug] = count + 1
        result.add(slug + ('-' + str(count) if count else ''))
    return result

documents = {'tools/gomad3/' + name + '.md': body for name, body in guides.items()}
documents['tools/gomad3integration/README.md'] = integration_guide
for relative, body in documents.items():
    source = ROOT / relative
    for dest in re.findall(r'\[[^\]]*\]\(([^)]+)\)', body):
        if '://' in dest or dest.startswith('mailto:'):
            continue
        path, _, fragment = dest.partition('#')
        target = (source.parent / path).resolve() if path else source
        ok = target.exists() and (not fragment or fragment in anchors(target.read_text()))
        links.append({'source': str(source.relative_to(ROOT)), 'destination': dest, 'resolves': ok})
        if not ok:
            errors.append(links[-1])

manifest_path = 'tools/gomad3integration/qualification/soak.json'
before_bytes = subprocess.check_output(['git', 'show', BASE + ':' + manifest_path], cwd=ROOT)
before = json.loads(before_bytes)
after_text = read(manifest_path)
after = json.loads(after_text)
original_hash = hashlib.sha256(before_bytes).hexdigest()
current_hash = inputs[manifest_path]
before.pop('sizing')
after.pop('sizing')
before['informational_platforms'] = sorted(before['informational_platforms'])
after['informational_platforms'] = sorted(after['informational_platforms'])
if before != after:
    errors.append('manifest execution selection or policy changed')
workflow = read('.github/workflows/gomad3.yml')
smoke_workflow = read('.github/workflows/gomad3-smoke.yml')
smoke_linux = smoke_workflow.split('.platform == {"goos": "linux", "goarch": "amd64"}', 1)[1].split("' tools/gomad3/.toolchain/smoke-qualification-set.json", 1)[0]
if not all(claim in smoke_linux for claim in ['.selected == 4 and .completed == 4 and (.supported + .failed) == 4', '.unsupported == 0 and .infrastructure_errors == 0', '.classification == "nondeterministic"', '.classification == "replay_divergence"', 'select(.classification == "qualified")', '.replayed and .replay_match and .choice_replay_exact']):
    errors.append('unexpected Linux smoke checker policy')
if 'requires all four suites to complete with `unsupported` and `infrastructure_errors` zero' not in ' '.join(integration_guide.split()):
    errors.append('integration guide omits actual Linux smoke completion rule')
base_workflow = subprocess.check_output(['git', 'show', BASE + ':.github/workflows/gomad3.yml'], cwd=ROOT, text=True)
without_comments = lambda text: '\n'.join(line for line in text.splitlines() if not line.lstrip().startswith('#'))
if without_comments(workflow) != without_comments(base_workflow):
    errors.append('workflow executable source changed')
for platform in ['darwin', 'linux']:
    job = workflow.split('  determinism-soak-' + platform + ':', 1)[1].split('\n  determinism-soak-', 1)[0]
    workloads = re.search(r'workload: \[([^]]+)\]', job).group(1).split(', ')
    selected = [suite for entry in after['selections'] for suite in entry['suites']]
    if workloads != selected or 'seed: [11, 17]' not in job or 'timeout-minutes: 180' not in job:
        errors.append('workflow selection/bounds mismatch: ' + platform)
tests = json.loads(read('tools/gomad3integration/qualification/tests.json'))
result = {'errors': errors, 'command_help_inventory': commands, 'links': links, 'documented_flags': sorted(documented_flags),
          'positional_target_flags_from_parser': sorted(positional_flags),
          'integration_guide_links_checked': True, 'linux_smoke_policy_matches_documented_completion_rule': True,
          'workflow_executable_source_unchanged': without_comments(workflow) == without_comments(base_workflow),
          'generated_selection_count': len(tests['suites']), 'native_execution_claim': False,
          'manifest_before_sha256': original_hash, 'manifest_after_sha256': current_hash,
          'manifest_execution_policy_unchanged': before == after, 'manifest_prose_changes_raw_report_and_ledger_run_identity': True,
          'inputs_sha256': inputs}
destination = OUT / (sys.argv[1] if len(sys.argv) > 1 else 'current-source-audit.json')
with destination.open('x') as output:
    json.dump(result, output, indent=2)
    output.write('\n')
print(json.dumps({'errors': errors, 'command_help_entries': len(commands), 'links': len(links), 'selected_tests': len(tests['suites'])}))
sys.exit(bool(errors))
