"""Reconstruct native retained admission and oracle facts from actual raw preimages."""
import base64
import copy

from driver import require
from native_roles import decode
from native_observations import canonical, digest

FILE = 'evidence/cluster-replay.json'
MARKER = 'GOMAD_RETAINED_OBSERVATION '
ORDINARY = {'canonical', 'indented', 'member-order', 'escaped-member', 'escaped-value', 'trailing-space', 'explicit-optional-null'}
def b64(text, maximum=128 << 20):
    require(isinstance(text, str) and len(text) <= (maximum + 2) // 3 * 4, 'retained preimage exceeds byte bound')
    value = base64.b64decode(text, validate=True)
    require(len(value) <= maximum and base64.b64encode(value).decode() == text, 'retained noncanonical base64')
    return value


def same(left, right):
    return canonical(left) == canonical(right)


def ordinary(rows):
    values = {row['case'].split('/', 1)[1]: row for row in rows if row['case'].startswith('ordinary/')}
    require(set(values) == ORDINARY, 'retained ordinary JSON variants missing or duplicated')
    facts = []
    for case, row in values.items():
        inputs, actual = row['inputs'], row['observed']
        raw, appended = b64(inputs['payload_base64']), b64(inputs['appended_base64'])
        decoded = decode(raw)
        if decoded['configuration'].get('oracle_plan') is None:
            decoded['configuration'].pop('oracle_plan', None)
        require(appended == raw + b'\n' and same(decoded, actual['canonical_inputs']), 'ordinary JSON spelling changed decoded values')
        require(same(actual['canonical_inputs'], inputs['canonical_inputs']) and actual['raw_sha256'] == digest(raw)
                and isinstance(actual['appended_error'], str) and actual['appended_error'], 'ordinary spelling bypassed exact raw authentication')
        descriptor = inputs['manifest']['files'][FILE]
        stream = inputs['manifest']['record']['streams'][FILE]
        require(descriptor['bytes'] == stream['bytes'] == len(raw) and descriptor['sha256'] == stream['sha256'] == digest(raw), 'ordinary JSON bytes are not both manifest and record bound')
        require(actual.get('public_status') == 0 and b'reproduced=true' in b64(actual.get('public_output_base64', '')), 'ordinary spelling did not pass real public replay')
        facts.append(dict(case=case, raw_sha256=digest(raw)))
    return facts


def messages(stdout, stderr, complete, report_only):
    output = []
    for raw in (stdout, stderr):
        try: text = raw.decode('utf8')
        except UnicodeError:
            complete = False
            continue
        for line in text.splitlines():
            line = line.lstrip(' \t')
            tagged = False
            record = line.find('{')
            json_start = record == 0 or record > 0 and line[:record].rstrip().endswith(':') and line[:record].rstrip()[:-1].rsplit(':', 1)[-1].isdigit()
            trailing = line.rfind('\t{')
            if json_start or trailing >= 0:
                at = record if json_start else trailing + 1
                try: value = decode(line[at:])
                except (ValueError, RuntimeError):
                    if 'failed assertion:' in line or 'failed-assertion' in line or 'FailedAssertion' in line: complete = False
                    continue
                if not isinstance(value, dict): continue
                tagged = any(value.get(key) is True for key in ('failed-assertion', 'FailedAssertion'))
                if any(value.get(key) is not None and type(value[key]) is not bool for key in ('failed-assertion', 'FailedAssertion')):
                    complete = False
                    continue
                if json_start:
                    line = value.get('msg', value.get('message', '')) or ''
                    if not isinstance(line, str):
                        complete &= not tagged
                        continue
                    if not tagged and not line.startswith('failed assertion: '): continue
                else:
                    line = line[:at].rstrip('\t')
                    if tagged and 'failed assertion: ' not in line: line = line.rsplit('\t', 1)[-1]
            at = line.find('failed assertion: ')
            if at < 0 and not tagged: continue
            message = line if at < 0 else line[at + len('failed assertion: '):]
            if not message or len(message.encode()) > 4096: complete = False
            else: output.append(message)
    output = sorted(set(output))
    if not output and complete and not report_only: return None
    return dict(messages=output, report_only=report_only, inconclusive=not complete)


def sealed_oracle(name, passed, evidence, *, report_only=False, inconclusive=False, soft=False):
    evidence = [dict(label=label, value=base64.b64encode(raw).decode(), full_sha256=digest(raw)) for label, raw in evidence]
    result = dict(name=name, passed=passed, inconclusive=inconclusive, soft_assertion=soft, report_only=report_only,
                  evidence=evidence, failure_identity='', identity='')
    if not passed: result['failure_identity'] = digest(canonical(dict(name=name, evidence=evidence)))
    result['identity'] = digest(canonical(result))
    return result


def report(oracles):
    failed = [oracle for oracle in oracles if not oracle['passed']]
    return dict(schema='gomad.cluster-oracles/v1', outcome='oracle_failed' if failed else 'completed',
                reason='soft_assertion' if any(item['soft_assertion'] for item in failed) else 'oracle_failed' if failed else '',
                failure_identity=digest(canonical(failed)) if failed else '', oracles=oracles)


def soft_report(inputs):
    stdout, stderr = b64(inputs['stdout_base64']), b64(inputs['stderr_base64'])
    handle = inputs['handle']
    plan = inputs['oracle_plan']
    policies = plan['soft_assertions'] if plan else []
    report_only = next((item['report_only'] for item in policies if item['handle'] == handle), False)
    complete = inputs['termination'] in ('exit', 'trap') and inputs['offered_bytes'] == len(stdout) + len(stderr)
    observed = messages(stdout, stderr, complete, report_only)
    oracles = []
    if observed:
        instance = (handle['node'] + '/' + str(handle['incarnation'])).encode()
        evidence = [('instance', instance), ('messages', canonical(observed['messages'])),
                    ('status', b'inconclusive' if observed['inconclusive'] else b'observed')]
        require(sum(len(raw) for _, raw in evidence) <= (plan['maximum_bytes'] if plan else 64 << 20), 'soft oracle exceeds original evidence bound')
        oracles.append(sealed_oracle('soft-assertion.instance-' + digest(instance),
                                    not observed['inconclusive'] and (not observed['messages'] or report_only), evidence,
                                    report_only=report_only, inconclusive=observed['inconclusive'], soft=True))
    return report(oracles)


def soft_cases(rows):
    expected = {'soft/' + name + '/report-only=' + str(policy).lower()
                for name in ('literal-json', 'literal-tagged', 'invalid-utf8', 'offered-mismatch', 'capacity-equal-counters') for policy in (False, True)}
    values = [row for row in rows if row['case'] in expected]
    require({row['case'] for row in values} == expected and len(values) == len(expected), 'soft oracle catalog changed')
    for row in values:
        require(same(row['observed'], soft_report(row['inputs'])), 'soft oracle report differs from actual unshortened output')
    extras = {row['case']: row for row in rows if row['case'].startswith('soft/') and row['case'] not in expected}
    require(set(extras) == {'soft/missing-execution/report-only=false', 'soft/missing-execution/report-only=true', 'soft/unstarted-unrequested'}, 'missing-execution soft catalog changed')
    for policy in (False, True):
        row = extras['soft/missing-execution/report-only=' + str(policy).lower()]
        inputs = row['inputs']
        require(inputs['incarnations'] == [] and inputs['oracle_plan']['soft_assertions'] == [dict(handle=inputs['configured_handle'], report_only=policy)], 'missing-execution control changed')
        instance = (inputs['configured_handle']['node'] + '/' + str(inputs['configured_handle']['incarnation'])).encode()
        oracle = sealed_oracle('soft-assertion.instance-' + digest(instance), False,
            [('instance', instance), ('messages', b'[]'), ('status', b'inconclusive')],
            report_only=policy, inconclusive=True, soft=True)
        require(same(row['observed'], report([oracle])), 'absent requested execution became a completed test')
    unstarted = extras['soft/unstarted-unrequested']
    require(unstarted['inputs']['incarnations'] == [] and unstarted['inputs']['oracle_plan'] is None and same(unstarted['observed'], report([])), 'unrequested execution gained a test oracle')
    return [dict(case=row['case'], report_sha256=digest(canonical(row['observed']))) for row in values + list(extras.values())]


def reseal_report(value):
    result = copy.deepcopy(value)
    result['oracles'] = [sealed_oracle(item['name'], item['passed'],
        [(entry['label'], b64(entry['value'])) for entry in item['evidence']],
        report_only=item['report_only'], inconclusive=item['inconclusive'], soft=item['soft_assertion'])
        for item in result['oracles']]
    failed = [item for item in result['oracles'] if not item['passed']]
    result['failure_identity'] = digest(canonical(failed)) if failed else ''
    return result


def oracle_cases(rows):
    values = {row['case']: row for row in rows if row['case'].startswith('retained-oracle/')}
    cases = ('valid', 'forged-message', 'forged-instance', 'missing-output', 'forged-digest',
             'generic', 'mixed', 'truncated-pass', 'noncanonical-summary', 'invalid-stream-summary')
    expected = {'retained-oracle/' + case for case in cases}
    expected.update('retained-oracle/' + case + '/shortened=' + str(shortened).lower()
                    for case in ('literal-json', 'literal-tagged') for shortened in (False, True))
    expected.update('retained-oracle/truncated-inconclusive/report-only=' + str(policy).lower() for policy in (False, True))
    require(set(values) == expected, 'retained oracle mutation catalog changed')
    baseline = values['retained-oracle/valid']['inputs']
    configuration = baseline['retained_inputs']['configuration']
    handle = configuration['incarnations'][0]['handle']
    summary_item = baseline['summary']['incarnations'][0]
    original_report = soft_report(dict(handle=handle, stdout_base64=baseline['stdout_base64'],
        stderr_base64=baseline['stderr_base64'], termination=summary_item['termination'],
        offered_bytes=summary_item['offered_output_bytes'], oracle_plan=configuration['oracle_plan']))
    require(same(original_report, baseline['report']), 'valid retained oracle differs from unshortened actual outputs')
    facts = []
    for case in cases:
        row = values['retained-oracle/' + case]
        inputs = row['inputs']
        expected_inputs = copy.deepcopy(baseline['retained_inputs'])
        expected_summary, expected_report = copy.deepcopy(baseline['summary']), copy.deepcopy(original_report)
        if case == 'forged-message':
            entry = expected_report['oracles'][0]['evidence'][1]
            raw = canonical(['forged invariant'])
            entry.update(value=base64.b64encode(raw).decode(), full_sha256=digest(raw))
        elif case == 'forged-instance':
            raw = b'foreign/1'
            expected_report['oracles'][0]['name'] = 'soft-assertion.instance-' + digest(raw)
            expected_report['oracles'][0]['evidence'][0].update(value=base64.b64encode(raw).decode(), full_sha256=digest(raw))
        elif case == 'missing-output': expected_inputs['retained'][0]['evidence'] = {}
        elif case == 'forged-digest':
            expected_inputs['retained'][0]['evidence']['stderr']['sha256'] = digest(b'forged output')
            expected_summary['incarnations'][0]['streams']['stderr']['sha256'] = digest(b'forged output')
        elif case in ('generic', 'mixed'):
            generic = sealed_oracle('generic', False, [])
            expected_report['oracles'] = [generic] if case == 'generic' else expected_report['oracles'] + [generic]
        elif case == 'truncated-pass': expected_summary['incarnations'][0]['offered_output_bytes'] = 100
        elif case == 'invalid-stream-summary': expected_summary['incarnations'][0]['streams'] = []
        expected_report = reseal_report(expected_report)
        require(same(inputs['retained_inputs'], expected_inputs) and same(inputs['summary'], expected_summary)
                and same(inputs['report'], expected_report), 'retained oracle mutation changed unrelated input: ' + case)
        summary_raw, report_raw = b64(inputs['summary_base64']), b64(inputs['report_base64'])
        require(same(decode(summary_raw), expected_summary) and report_raw == canonical(expected_report), 'retained oracle raw preimages changed')
        require((summary_raw != canonical(expected_summary)) == (case == 'noncanonical-summary'), 'retained oracle canonical-summary control changed')
        require((row['observed']['error'] is None) == (case == 'valid'), 'retained oracle mutation was not refused')
        facts.append(dict(case=case, summary_sha256=digest(summary_raw), report_sha256=digest(report_raw)))
    for case, message in (('literal-json', 'invalid {"left":1}'), ('literal-tagged', 'invalid {state}')):
        for shortened in (False, True):
            key = 'retained-oracle/' + case + '/shortened=' + str(shortened).lower()
            row = values[key]
            raw = b64(row['inputs']['stderr_base64'])
            require(messages(b'', raw, True, False)['messages'] == [message] == [row['inputs']['expected_message']], 'literal braces were shortened before oracle assessment')
            retained_report = decode(b64(row['inputs']['retained_report_base64']))
            expected_report = soft_report(dict(handle=handle, stdout_base64='', stderr_base64=base64.b64encode(raw).decode(),
                termination='exit', offered_bytes=len(raw), oracle_plan=configuration['oracle_plan']))
            if shortened:
                changed = canonical(['invalid'])
                expected_report['oracles'][0]['evidence'][1].update(value=base64.b64encode(changed).decode(), full_sha256=digest(changed))
                expected_report = reseal_report(expected_report)
            require(same(retained_report, expected_report) and (row['observed']['error'] is not None) == shortened,
                    'retained literal output forgery admitted or unrelated field changed')
            facts.append(dict(case=key, report_sha256=digest(canonical(retained_report))))
    for policy in (False, True):
        key = 'retained-oracle/truncated-inconclusive/report-only=' + str(policy).lower()
        row = values[key]
        plan = copy.deepcopy(configuration['oracle_plan'])
        plan['soft_assertions'][0]['report_only'] = policy
        require(row['inputs'] == dict(stdout_base64='', stderr_base64='', offered_bytes=100), 'truncated control lost actual offered bytes')
        expected_report = soft_report(dict(row['inputs'], handle=handle, termination='exit', oracle_plan=plan))
        require(same(row['observed'], expected_report), 'equal retained counters incorrectly established completeness')
        facts.append(dict(case=key, report_sha256=digest(canonical(expected_report))))
    return facts


def producer_cases(rows):
    values = {row['case']: row for row in rows if row['case'].startswith('producer/')}
    expected = {'producer/' + kind + '/report-only=' + str(policy).lower()
                for kind in ('cluster-oracles', 'scenario') for policy in (False, True)}
    require(set(values) == expected, 'actual producer matrix changed')
    facts = []
    for case, row in values.items():
        inputs, observed = row['inputs'], row['observed']
        retained = inputs['retained_inputs']
        configuration = retained['configuration']
        limits = configuration['limits']
        policy = case.endswith('=true')
        require(observed['prepare_error'] is None and limits['evidence_bytes'] > limits['output_bytes'], 'actual producer did not admit distinct evidence/output bounds')
        output = {name: b64(raw) for name, raw in observed['actual_outputs_base64'].items()}
        require(all(raw == b'PASS\nfailed assertion: invalid {state}\n' for raw in output.values()), 'producer output changed the unshortened actual message')
        handles = [item['handle'] for item in retained['retained']]
        require(handles == sorted(handles, key=lambda value: (value['node'], value['incarnation']))
                and handles == sorted([item['handle'] for item in configuration['incarnations']], key=lambda value: (value['node'], value['incarnation'])), 'producer omitted an actual incarnation')
        oracles, scenario = [], []
        seen = set()
        for item in retained['retained']:
            descriptor = item['evidence']['stdout']
            name = descriptor['file']
            seen.add(name)
            raw = output[name]
            require(descriptor['limit'] == limits['evidence_bytes'] and descriptor['bytes'] == len(raw)
                    and descriptor['sha256'] == digest(raw), 'producer descriptor bound differs from actual output')
            manifest = inputs['manifest']
            entry, stream = manifest['files'][name], manifest['record']['streams'][name]
            require(entry['bytes'] == stream['bytes'] == len(raw) and entry['sha256'] == stream['sha256'] == digest(raw), 'producer output is not retained with exact raw authentication')
            handle = item['handle']
            instance = (handle['node'] + '/' + str(handle['incarnation'])).encode()
            oracles.append(sealed_oracle('soft-assertion.instance-' + digest(instance), policy,
                [('instance', instance), ('messages', canonical(['invalid {state}'])), ('status', b'observed')],
                report_only=policy, inconclusive=False, soft=True))
            scenario.append(dict(handle=handle, report=dict(messages=['invalid {state}'], report_only=policy, inconclusive=False)))
        require(seen == set(output), 'producer attached an unattributed actual output')
        if '/cluster-oracles/' in case:
            by_handle = {(item['handle']['node'], item['handle']['incarnation']): oracle for item, oracle in zip(retained['retained'], oracles)}
            oracles = [by_handle[(item['handle']['node'], item['handle']['incarnation'])] for item in configuration['incarnations']]
            require(observed['output_descriptor_count'] == len(handles) and same(observed['actual_report'], report(oracles)), 'cluster producer report differs from actual outputs')
        else:
            require(observed['cluster_oracles_present'] is False and observed['scenario_report_present'] is True
                    and same(observed['actual_report'], dict(schema='gomad.cluster-soft-assertions/v1', incarnations=scenario)), 'scenario producer substituted a cluster oracle stream')
        facts.append(dict(case=case, actual_outputs_sha256={name: digest(raw) for name, raw in output.items()}, report_sha256=digest(canonical(observed['actual_report']))))
    return facts


SOURCES = ('tools/gomad/rust/gomad/src/replay/cluster/retained_tests.rs',
           'tools/gomad/rust/gomad/src/replay/cluster/oracles.rs',
           'tools/gomad/rust/gomad/src/replay/cluster.rs',
           'tools/gomad/rust/gomad/src/campaign/soft_assertions.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario.rs')


def observations(config, stdout):
    rows = [decode(line.split(MARKER, 1)[1]) for line in stdout.splitlines() if MARKER in line]
    ordinary_rows = [row for row in rows if 'original_root' not in row]
    require(all(set(row) == {'schema', 'case', 'inputs', 'observed'} and row['schema'] == 'gomad.retained-reader-observation/v1' for row in ordinary_rows), 'foreign retained observation envelope')
    require(len({row['case'] for row in ordinary_rows}) == len(ordinary_rows), 'duplicate retained receiving case')
    from native_roles import pinned
    from pathlib import Path
    for name in SOURCES:
        path = Path(config['repo']) / name
        pinned(config, path)
    spelling, soft = ordinary(ordinary_rows), soft_cases(ordinary_rows)
    oracle, producers = oracle_cases(ordinary_rows), producer_cases(ordinary_rows)
    roots = {
        'TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence': dict(cases=spelling),
        'TestSoftAssertionIncompleteClusterEvidenceIsInconclusive': dict(cases=soft),
        'TestSoftAssertionClusterCapacityCountersNeverProveCompleteness': dict(cases=[row for row in soft if 'capacity-equal-counters' in row['case']]),
        'TestSoftAssertionRetainedEvidenceRequiresActualOutput': dict(cases=oracle[:10]),
        'TestSoftAssertionRetainedLiteralBracesRederiveCompleteMessage': dict(cases=[row for row in oracle if '/shortened=' in row['case']]),
        'TestSoftAssertionRetainedTruncatedOutputIsInconclusive': dict(cases=oracle[-2:]),
    }
    return dict(roots=roots, spelling_cases=spelling, producer_cases=producers,
                oracle_cases=oracle, soft_cases=soft)
