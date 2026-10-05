"""Compare actual final Go events to the pre-edit host observation."""
import json
import pathlib
import sys

OUT = pathlib.Path(__file__).resolve().parent


def events(label):
    results, errors = {}, {}
    for line in (OUT / (label + '.stdout')).read_text().splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(event, dict) or not event.get('Test'):
            continue
        key = event['Package'] + ':' + event['Test']
        if event['Action'] in ['pass', 'fail', 'skip']:
            results[key] = event['Action']
        elif event.get('OutputType') == 'error':
            errors.setdefault(key, []).append(event['Output'])
    return results, errors


baseline, baseline_errors = events('baseline-host-developmental')
for label in sys.argv[1:]:
    destination = OUT / (label + '-baseline-comparison.json')
    assert not destination.exists(), 'comparison already exists'
    final, final_errors = events(label)
    unmatched = sorted(set(final) - set(baseline))
    changed = {key: {'baseline': baseline[key], 'final': result}
               for key, result in final.items() if key in baseline and baseline[key] != result}
    errors_changed = {key: {'baseline': baseline_errors.get(key), 'final': final_errors.get(key)}
                      for key in final if final[key] == 'fail'
                      and baseline_errors.get(key) != final_errors.get(key)}
    info = {'baseline': 'baseline-host-developmental.json', 'final': label + '.json',
            'compared_test_events': len(final), 'unmatched_final_test_events': unmatched,
            'changed_verdicts': changed, 'changed_raw_error_outputs': errors_changed,
            'final_failures': sorted(key for key, result in final.items() if result == 'fail'),
            'final_skips': sorted(key for key, result in final.items() if result == 'skip'),
            'exact_fail_skip_pairs_match_baseline': not unmatched and not changed,
            'all_final_failure_error_outputs_match_baseline': not errors_changed}
    destination.write_text(json.dumps(info, indent=2) + '\n')
    print(label, 'compared', len(final), 'unmatched', len(unmatched),
          'changed verdicts', len(changed), 'changed raw error outputs', len(errors_changed))
