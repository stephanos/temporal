#!/usr/bin/env python3
"""Run the controlled qualification accounting corpus through the Rust evaluators.

The fixture binary evaluates every raw observation case twice; both outputs
must agree byte for byte. An independent oracle here recomputes the linked
CPU and peak accounting and each budget status before the reviewed golden is
compared (or regenerated twice with GOMAD_GOLDEN_REGENERATE=1).
"""
import argparse
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
loader = SourceFileLoader('qualify_parity', str(CORPUS.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)
GIB = 1 << 30
MINUTE = 60_000_000_000
HOUR = 3_600_000_000_000


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label}: got {actual!r}, expected {expected!r}')


def cpu(after, before):
    return after['user_nanos'] - before['user_nanos'] + after['system_nanos'] - before['system_nanos']


def monotone(after, before):
    return all(after[key] >= before[key] for key in ('user_nanos', 'system_nanos', 'peak_rss_bytes'))


def interval_failure(observations):
    start, prepared, executed = (observations[key] for key in ('start', 'prepared', 'executed'))
    if start is None or prepared is None or executed is None:
        return 'missing'
    if executed['linked']['peak_rss_bytes'] == 0 or observations['operator_process_id'] == 0:
        return 'missing'
    for later, earlier in ((prepared, start), (executed, prepared)):
        if not monotone(later['linked'], earlier['linked']) or not monotone(later['children'], earlier['children']):
            return 'decreased'
    previous = prepared['linked']
    counts = {'execute': 0, 'replay': 0}
    residents = 0
    for window in windows(observations['samples']):
        first = window[0]
        counts[first['phase']] += 1
        if first['iteration'] != counts[first['phase']]:
            return 'order'
        for sample in window:
            if 0 in (sample['wall_nanos'], sample['guest_peak_memory_bytes'], sample['guest_memory_limit_bytes']):
                return 'missing'
            if sample['linked_before'] is None or sample['linked_after'] is None:
                return 'missing'
            if any(sample[key] != first[key] for key in ('wall_nanos', 'virtual_time_nanos', 'linked_before', 'linked_after')):
                return 'window'
        handles = [sample.get('incarnation') for sample in window]
        if len(window) > 1 and (None in handles or len(set(handles)) != len(handles)):
            return 'handles'
        residents = max(residents, len({handle.split('/')[0] for handle in handles if handle}) or 1)
        before, after = first['linked_before'], first['linked_after']
        if not monotone(before, previous) or not monotone(after, before):
            return 'overlap'
        previous = after
    if not monotone(executed['linked'], previous):
        return 'overlap'
    if residents != observations['resident_stores']:
        return 'residents'
    return None


def windows(samples):
    """Consecutive samples of one logical execution: its concurrent stores."""
    groups = []
    for sample in samples:
        if groups and (groups[-1][0]['phase'], groups[-1][0]['iteration']) == (sample['phase'], sample['iteration']):
            groups[-1].append(sample)
        else:
            groups.append([sample])
    return groups


def oracle(case):
    """Independent expectations for the facts each case exists to prove."""
    observations = case['observations']
    failure = interval_failure(observations)
    expected = {'interval_failure': failure}
    if failure is None and (observations['executed']['children']['peak_rss_bytes'] == 0
                            or observations['preparation_wall_nanos'] == 0):
        failure = 'budget-measurement'
    expected['interval_failure'] = failure
    if failure is None:
        start, executed = observations['start'], observations['executed']
        compiler = cpu(executed['children'], start['children'])
        linked = cpu(executed['linked'], start['linked'])
        expected['cpu'] = (compiler, linked, compiler + linked)
        samples = observations['samples']
        statuses = {
            'preparation wall': 'within' if observations['preparation_wall_nanos'] <= 10 * MINUTE else 'exceeded',
            'execution/replay store wall': 'within' if max(s['wall_nanos'] for s in samples) <= 15 * MINUTE else 'exceeded',
            'individual compiler peak RSS': 'within' if executed['children']['peak_rss_bytes'] <= 6 * GIB else 'exceeded',
            'resident guests': 'within' if observations['resident_stores'] <= 2 else 'exceeded',
        }
        peak = executed['linked']['peak_rss_bytes']
        groups = windows(samples)
        # A zero phase peak is a missing measurement, never an attributed one.
        operator = observations.get('operator_peak_rss_bytes') or None
        attributed = operator is not None and all(window[0].get('store_peak_rss_bytes') for window in groups)
        largest = attributed and max(window[0]['store_peak_rss_bytes'] for window in groups)
        # Each window lies within the lifetime peak after it; any rise across
        # its boundaries is explained by the window or by operator work.
        # Rises outside windows (preparation, between windows, after the last)
        # are explained by operator work alone.
        # Even without full attribution, every measured window fits the
        # lifetime peaks around it.
        unexplained = any(window[0].get('store_peak_rss_bytes') and window[0]['store_peak_rss_bytes']
                          > min(window[0]['linked_after']['peak_rss_bytes'], peak) for window in groups)
        if unexplained:
            expected['budget_failure'] = 'attribution'
        elif attributed:
            # Operator phases begin before the baseline: the operator figure
            # covers the initial peak too.
            initial = observations['start']['linked']['peak_rss_bytes']
            previous = observations['prepared']['linked']['peak_rss_bytes']
            unexplained = initial > operator or previous > max(initial, operator)
            for window in groups:
                before, after = (window[0][key]['peak_rss_bytes'] for key in ('linked_before', 'linked_after'))
                store = window[0]['store_peak_rss_bytes']
                unexplained |= before > max(previous, operator) or store > after or after > max(before, store, operator)
                previous = after
            unexplained |= peak > max(previous, operator)
        if expected.get('budget_failure'):
            pass
        elif attributed and (unexplained or largest > peak or max(largest, operator) < peak):
            # Attributed phases must account for the lifetime peak.
            expected['budget_failure'] = 'attribution'
        elif attributed:
            # Phase attribution: operator phases, standalone store windows (always
            # compiling) and cluster windows (stores with node/incarnation
            # handles, even a single node) meet their own historical ceilings.
            single = max([window[0]['store_peak_rss_bytes'] for window in groups if window[0].get('incarnation') is None] or [0])
            clusters = [window[0]['store_peak_rss_bytes'] for window in groups if window[0].get('incarnation') is not None]
            statuses['operator lifetime peak RSS'] = 'within' if operator <= 2 * GIB else 'exceeded'
            # A window that loaded a cached compiled module is a warm window.
            warm = [window[0]['store_peak_rss_bytes'] for window in groups
                    if window[0].get('incarnation') is None and window[0].get('cache_hit') is True]
            statuses['individual warm helper peak RSS'] = (('within' if max(warm) <= 4 * GIB else 'exceeded') if warm
                                                          else 'within' if single <= 4 * GIB else 'unattributed')
            statuses['individual cold helper peak RSS'] = 'within' if single <= 6 * GIB else 'exceeded'
            if clusters:
                statuses['conservative resident helper peak upper bound'] = 'within' if max(clusters) <= 8 * GIB else 'exceeded'
        else:
            # A measured overrun survives the fallback; otherwise the lifetime
            # peak upper-bounds the phase.
            def known(cluster):
                peaks = [window[0]['store_peak_rss_bytes'] for window in groups if window[0].get('store_peak_rss_bytes')
                         and (window[0].get('incarnation') is not None) == cluster]
                return max(peaks) if peaks else None
            for name, measured, limit in (('operator lifetime peak RSS', operator, 2 * GIB),
                                          ('individual warm helper peak RSS', None, 4 * GIB),
                                          ('individual cold helper peak RSS', known(False), 6 * GIB),
                                          ('conservative resident helper peak upper bound', known(True), 8 * GIB)):
                statuses[name] = ('exceeded' if measured is not None and measured > limit
                                  else 'within' if peak <= limit else 'unattributed')
        for sample in samples:
            if sample['guest_peak_memory_bytes'] > sample['guest_memory_limit_bytes']:
                store = ' ' + sample['incarnation'] if sample.get('incarnation') else ''
                statuses[f"guest linear memory {sample['phase']}:{sample['iteration']}{store}"] = 'exceeded'
        expected['statuses'] = statuses
    return expected


def gib(value):
    return f'{value} B ({value / GIB:.3f} GiB)'


def check(case, row):
    expected = oracle(case)
    name = case['name']
    if row.get('budget'):
        for item in row['budget']['checks']:
            show = gib if any(word in item['name'] for word in ('RSS', 'memory', 'peak')) else str
            print(f"{name}: {item['name']}: {show(item['observed'])} vs {show(item['limit'])}: {item['status']}")
    if expected['interval_failure'] is not None:
        equal(row['cost'], None, name + '.cost_refused')
        equal(row['budget'], None, name + '.budget_refused')
        equal(bool(row['cost_failure']) and bool(row['budget_failure']), True, name + '.failures_named')
        return
    budget = row['budget']
    if expected.get('budget_failure'):
        equal((budget, 'inconsistent' in (row['budget_failure'] or '')), (None, True), name + '.attribution_refused')
    else:
        equal({check['name']: check['status'] for check in budget['checks']}, expected['statuses'], name + '.statuses')
        equal(budget['within_budget'], all(status == 'within' for status in expected['statuses'].values()), name + '.within')
        equal(budget['pending'], [n for n, s in expected['statuses'].items() if s == 'unattributed'], name + '.pending')
    groups = windows(case['observations']['samples'])
    executed_windows = [window for window in groups if window[0]['phase'] == 'execute']
    executes = len(executed_windows)
    records_agree = all(window[0]['virtual_time_nanos'] == run['virtual_time_nanos'] for window, run in
                        zip(executed_windows, case['runs']))
    if observations_cost_valid(case, executes, len(groups) - executes) and records_agree:
        cost = row['cost']
        equal((cost['execution_wall_nanos'], cost['fuel_used'], cost['guest_peak_memory_bytes']),
              (sum(window[0]['wall_nanos'] for window in executed_windows),
               sum(sample['fuel_used'] for window in executed_windows for sample in window),
               max(sample['guest_peak_memory_bytes'] for sample in case['observations']['samples'])),
              name + '.per_store_attribution')
        compiler, linked, total = expected['cpu']
        equal((cost['compiler_cpu_nanos'], cost['linked_cpu_nanos'], cost['cpu_nanos']), (compiler, linked, total), name + '.cpu_once')
        sample_cpu = sum(cpu(window[0]['linked_after'], window[0]['linked_before']) for window in groups)
        if name == 'cpu-counted-once':
            equal(sample_cpu > 0, True, name + '.store_cpu_observed_but_not_added')
        bugs = len({run['class'] for run in case['runs'] if run['class'] != 'success' and run['replayed']})
        equal(cost['replayable_bugs'], bugs, name + '.bugs')
        equal(cost['replayable_bugs_per_compute_hour'], {'numerator': bugs * HOUR, 'denominator': total}, name + '.bug_rate')
    else:
        equal(row['cost'], None, name + '.cost_refused')
    if 'terminal' in case:
        terminal = case['terminal']
        executed = case['observations']['executed']['linked']
        operator = case.get('terminal_operator_peak_rss_bytes')
        retained = case['observations'].get('operator_peak_rss_bytes') or None
        attributed = operator is not None and retained is not None
        # After the sealed Result only operator work runs, so an attributed
        # terminal operator phase must explain any lifetime-peak rise.
        unexplained = attributed and terminal['peak_rss_bytes'] > max(operator, executed['peak_rss_bytes'])
        if not monotone(terminal, executed) or attributed and operator < retained or unexplained:
            equal('failure' in row['terminal'], True, name + '.terminal_refused')
        else:
            # A retained operator overrun stays exceeded without the receipt's own figure.
            overrun = (retained if retained is not None and retained > 2 * GIB
                       else operator if operator is not None and operator > 2 * GIB else None)
            exact = attributed or overrun is not None
            peak = operator if attributed else overrun if overrun is not None else terminal['peak_rss_bytes']
            status = 'within' if peak <= 2 * GIB else 'exceeded' if exact else 'unattributed'
            equal(row['terminal'], {'cpu_nanos': cpu(terminal, executed), 'peak_rss_bytes': peak, 'status': status},
                  name + '.terminal_tail')


def observations_cost_valid(case, executes, replays):
    observations = case['observations']
    compiler = cpu(observations['executed']['children'], observations['start']['children'])
    return (compiler > 0 and observations['executed']['children']['peak_rss_bytes'] > 0 and executes == len(case['runs'])
            and replays == len(case['runs']) and observations['preparation_wall_nanos'] > 0)


def unit_pins(rows):
    """Binary-unit regression pins: a measured 5.777 GiB window passes the
    6 GiB cold ceiling; a 6.1 GiB window (6.55 GB) must exceed it."""
    statuses = {row['name']: {item['name']: item['status'] for item in row['budget']['checks']} for row in rows if row['budget']}
    equal(6 * GIB, 6_442_450_944, 'units.cold_ceiling')
    equal(statuses['phases-onebox-measured-5.78GiB-warm-pending']['individual cold helper peak RSS'], 'within', 'units.measured_window')
    equal(statuses['phases-onebox-measured-5.78GiB-warm-pending']['individual warm helper peak RSS'], 'unattributed', 'units.measured_warm')
    equal(statuses['phases-window-6.1GiB-cold-exceeded']['individual cold helper peak RSS'], 'exceeded', 'units.cold_exceeded')
    equal(statuses['phases-zero-window-falls-back']['individual warm helper peak RSS'], 'unattributed', 'units.zero_window_pending')
    equal(statuses['phases-zero-operator-falls-back']['operator lifetime peak RSS'], 'unattributed', 'units.zero_operator_pending')
    equal(statuses['phases-single-node-cluster-window']['conservative resident helper peak upper bound'], 'within',
          'units.single_node_cluster_ceiling')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True, help='determinism-fixture gomad binary')
    parser.add_argument('--work', type=Path, required=True)
    args = parser.parse_args()
    args.work = args.work.resolve()
    args.work.mkdir(parents=True, exist_ok=False)
    corpus_path = CORPUS / 'qualify-accounting-corpus.json'
    corpus = parity.decode(corpus_path.read_bytes())
    equal(corpus['schema'], 'gomad.qualify-accounting-corpus/v1', 'corpus.schema')
    outputs = []
    for attempt in range(2):
        result = subprocess.run([str(args.binary.resolve()), 'fixture-qualify-accounting', '--input', str(corpus_path)],
                                capture_output=True, timeout=120, env={'PATH': os.environ.get('PATH', '')})
        (args.work / f'accounting-{attempt}.stdout').write_bytes(result.stdout)
        (args.work / f'accounting-{attempt}.stderr').write_bytes(result.stderr)
        equal(result.returncode, 0, f'accounting[{attempt}].status: {result.stderr!r}')
        outputs.append(result.stdout)
    equal(outputs[1], outputs[0], 'accounting.repeated_agreement')
    rows = parity.decode(outputs[0].rstrip(b'\n'))['cases']
    equal([row['name'] for row in rows], [case['name'] for case in corpus['cases']], 'accounting.cases')
    for case, row in zip(corpus['cases'], rows):
        check(case, row)
    unit_pins(rows)
    view = dict(schema='gomad.qualify-accounting-golden/v1', cases=rows)
    golden = CORPUS / 'goldens-qualify/accounting.json'
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    parity.golden(golden, view, regenerate)
    first = golden.read_bytes()
    parity.golden(golden, dict(schema=view['schema'], cases=parity.decode(outputs[1].rstrip(b'\n'))['cases']), regenerate)
    equal(golden.read_bytes(), first, 'accounting.second_regeneration_bytes')
    print(f'qualify accounting: {len(rows)} controlled cases agree with the oracle and golden')
    phase_boundaries(args)


PHASE_FACTS = ('freed_window_measured', 'freed_window_heap_not_charged_to_operator',
               'live_window_measured', 'live_window_heap_charged_to_operator')


def phase_boundaries(args):
    """Real Linux phase boundaries in fresh fixture processes: heap a store
    window freed is not charged to the next operator phase; heap it left live is."""
    if not sys.platform.startswith('linux'):
        print('qualify phase boundaries: attribution is Linux-only; budgets stay pending elsewhere')
        return
    views = []
    for attempt in range(2):
        result = subprocess.run([str(args.binary.resolve()), 'fixture-qualify-phases'], capture_output=True, timeout=120,
                                env={'PATH': os.environ.get('PATH', '')})
        (args.work / f'phases-{attempt}.stdout').write_bytes(result.stdout)
        (args.work / f'phases-{attempt}.stderr').write_bytes(result.stderr)
        equal(result.returncode, 0, f'phases[{attempt}].status: {result.stderr!r}')
        view = parity.decode(result.stdout.rstrip(b'\n'))
        equal(view, dict(schema='gomad.qualify-phase-fixture/v1', **{fact: True for fact in PHASE_FACTS}),
              f'phases[{attempt}].facts')
        views.append(view)
    golden = CORPUS / 'goldens-qualify/phases.json'
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    parity.golden(golden, views[0], regenerate)
    first = golden.read_bytes()
    parity.golden(golden, views[1], regenerate)
    equal(golden.read_bytes(), first, 'phases.second_regeneration_bytes')
    print('qualify phase boundaries: freed window heap is released, live window heap stays charged')


if __name__ == '__main__':
    main()
