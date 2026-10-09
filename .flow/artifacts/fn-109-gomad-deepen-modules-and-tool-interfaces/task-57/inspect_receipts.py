import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/tmp/gomad-fn109-parallel.pmgezCtg/task-57')
OUT = pathlib.Path(__file__).resolve().parent
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong worktree')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


records = []
events_by_name = {}
for name in ['focused-before', 'focused-before-corrected', 'format-owned', 'focused-after',
             'ordinary-distinct', 'affected-vet', 'standalone-errortype',
             'architecture-source-sets', 'fresh-validate', 'affected-configured-lint',
             'make-fast-admitted-base', 'make-gomad-original-base']:
    path = OUT / (name + '.json')
    receipt = json.loads(path.read_text())
    assert receipt['terminal'] and not receipt['timed_out'] and receipt['source_unchanged'], name
    for suffix in ['stdout', 'stderr']:
        assert sha(OUT / (name + '.' + suffix)) == receipt[suffix + '_sha256'], name
    record = {'name': name, 'receipt': path.name, 'receipt_sha256': sha(path),
              **{key: receipt[key] for key in ['command', 'cwd', 'exit_code', 'elapsed_seconds',
                                             'source_before_sha256', 'source_after_sha256',
                                             'stdout_sha256', 'stderr_sha256', 'terminal']}}
    events = []
    output = collections.defaultdict(str)
    for line in (OUT / (name + '.stdout')).read_text().splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(event, dict) or 'Action' not in event:
            continue
        key = (event.get('Package'), event.get('Test'))
        if event['Action'] == 'output' and event.get('Test'):
            output[key] += event.get('Output', '')
        if event['Action'] in ['pass', 'fail', 'skip'] and event.get('Test'):
            events.append((event['Package'], event['Test'], event['Action']))
    if events:
        counts = collections.Counter(event[2] for event in events)
        top = collections.Counter(event[2] for event in events if '/' not in event[1])
        record['observations'] = dict(counts)
        record['top_level'] = dict(top)
        record['failures'] = [test for _, test, action in events if action == 'fail']
        record['skips'] = [test for _, test, action in events if action == 'skip']
        record['failure_output'] = [{'package': package, 'test': test, 'output_prefix': output[(package, test)][:500],
                                    'complete_output_sha256': hashlib.sha256(output[(package, test)].encode()).hexdigest()}
                                    for package, test, action in events if action == 'fail']
        events_by_name[name] = set(events)
    if 'lint' in name or name.startswith('make-'):
        lines = (OUT / (name + '.stdout')).read_text().splitlines()
        record['diagnostics_reported'] = sum(bool(re.match(r'(?:tools/gomad3/)?[^:]+\.go:\d+:\d+: ', line)) for line in lines)
    records.append(record)
assert events_by_name['focused-before-corrected'] == events_by_name['focused-after']
final_bindings = {record['source_after_sha256'] for record in records
                  if record['name'] not in ['focused-before', 'focused-before-corrected']}
assert len(final_bindings) == 1, final_bindings
base = (ROOT / '.flow/tmp/base_commit').read_text().strip()
commits = subprocess.check_output(['git', 'rev-list', '--reverse', base + '..HEAD'], cwd=ROOT, text=True).splitlines()
print(json.dumps({'base_commit': base, 'commits': commits,
                  'final_source_sha256': next(iter(final_bindings)),
                  'focused_corrected_baseline_final_outcomes_identical': True,
                  'gate_observations': records}, indent=2))
