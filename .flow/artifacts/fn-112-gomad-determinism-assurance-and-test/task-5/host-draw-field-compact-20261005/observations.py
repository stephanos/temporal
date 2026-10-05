"""Summarize immutable Go JSON receipts without re-running their suites."""
import json
import pathlib
import sys

OUT = pathlib.Path(__file__).resolve().parent
label = sys.argv[1]
destination = OUT / (label + '-observations.json')
assert not destination.exists(), 'observation already exists'
receipt = json.loads((OUT / (label + '.json')).read_text())
packages = {}
tests = {}
events = 0
for line in (OUT / (label + '.stdout')).read_text().splitlines():
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    if not isinstance(event, dict) or 'Action' not in event:
        continue
    events += 1
    package = event.get('Package', '')
    test = event.get('Test')
    action = event['Action']
    if test:
        key = package + ':' + test
        entry = tests.setdefault(key, {'package': package, 'test': test, 'output': []})
        if action == 'run':
            entry['started'] = event.get('Time')
        elif action in ['pass', 'fail', 'skip']:
            entry['result'] = action
            entry['elapsed'] = event.get('Elapsed')
        elif action == 'output':
            entry['output'].append(event.get('Output', ''))
    elif action in ['pass', 'fail', 'skip']:
        packages[package] = action

summary = {'receipt': label + '.json', 'exit': receipt['exit'],
           'elapsed_seconds': receipt['elapsed_seconds'], 'json_events': events,
           'source_changes': receipt['source_changes'], 'packages': packages,
           'test_event_counts': {result: sum(entry.get('result') == result for entry in tests.values())
                                 for result in ['pass', 'fail', 'skip']},
           'top_level_counts': {result: sum(entry.get('result') == result and '/' not in entry['test']
                                            for entry in tests.values())
                                for result in ['pass', 'fail', 'skip']},
           'unfinished_tests': [key for key, entry in tests.items() if 'result' not in entry],
           'failures_and_skips': {key: entry for key, entry in tests.items()
                                  if entry.get('result') in ['fail', 'skip']}}
destination.write_text(json.dumps(summary, indent=2) + '\n')
print(label, 'exit', receipt['exit'], 'packages', len(packages),
      'test event counts', summary['test_event_counts'],
      'unfinished', len(summary['unfinished_tests']))
for key in summary['unfinished_tests']:
    print('unfinished:', key)
