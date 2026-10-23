"""Independent raw linked-process accounting for retained final Results."""
import hashlib
import json

from driver import require

U64 = (1 << 64)-1
GIB = 1 << 30
MINUTE = 60_000_000_000
HOUR = 3_600_000_000_000


def integer(value):
    require(type(value) is int and 0 <= value <= U64, 'invalid raw u64 accounting measurement')
    return value


def total(values):
    value = sum(integer(item) for item in values)
    require(value <= U64, 'raw accounting counter overflow')
    return value


def usage(value):
    require(isinstance(value,dict) and set(value) == {'user_nanos','system_nanos','peak_rss_bytes'},
            'actual process resource observation required')
    for item in value.values(): integer(item)
    return value


def monotone(after, before):
    usage(after); usage(before)
    require(all(after[key] >= before[key] for key in before), 'raw process resource counters regressed')


def cpu(after, before):
    monotone(after,before)
    return total((after['user_nanos']-before['user_nanos'],after['system_nanos']-before['system_nanos']))


def groups(observations):
    snapshots = [observations[key] for key in ('start','prepared','executed')]
    require(all(isinstance(snapshot,dict) and set(snapshot) == {'linked','children'} for snapshot in snapshots),
            'actual start/prepared/executed snapshots required')
    start,prepared,executed = snapshots
    require(integer(observations['operator_process_id']) > 0 and usage(executed['linked'])['peak_rss_bytes'] > 0,
            'actual operator PID and linked peak required')
    for earlier,later in zip(snapshots,snapshots[1:]):
        for kind in ('linked','children'): monotone(later[kind],earlier[kind])
    require(observations['samples'], 'actual guest store samples required')
    windows = []
    for sample in observations['samples']:
        require(sample['phase'] in ('execute','replay'), 'unknown store measurement phase')
        for field in ('iteration','wall_nanos','fuel_used','guest_peak_memory_bytes','guest_memory_limit_bytes','trace_bytes','virtual_time_nanos'):
            integer(sample[field])
        require(all(sample[field] > 0 for field in ('wall_nanos','guest_peak_memory_bytes','guest_memory_limit_bytes')),
                'actual store wall and guest memory required')
        if sample.get('store_peak_rss_bytes') is not None: integer(sample['store_peak_rss_bytes'])
        require(sample.get('cache_hit') in (None, True, False), 'invalid store cache-hit observation')
        key = sample['phase'],sample['iteration']
        if windows and (windows[-1][0]['phase'],windows[-1][0]['iteration']) == key:
            windows[-1].append(sample)
        else:
            windows.append([sample])
    counts = dict(execute=0,replay=0)
    previous = prepared['linked']
    residents = 0
    for window in windows:
        first = window[0]
        counts[first['phase']] += 1
        require(first['iteration'] == counts[first['phase']], 'sample phase iteration omission/duplicate/reordering')
        names,nodes = set(),set()
        for sample in window:
            require(all(sample.get(field) == first.get(field) for field in ('wall_nanos','virtual_time_nanos',
                    'linked_before','linked_after','store_peak_rss_bytes','cache_hit')), 'concurrent stores disagree on raw shared window')
            name = sample.get('incarnation')
            if name is None:
                require(len(window) == 1,'concurrent stores require original incarnation handles')
            else:
                require(isinstance(name,str) and '/' in name,'invalid store incarnation handle')
                node,number = name.split('/',1)
                require(node and number and all(char in '0123456789' for char in number) and name not in names,
                        'duplicate/invalid cluster store handle')
                names.add(name); nodes.add(node)
        monotone(first['linked_before'],previous)
        monotone(first['linked_after'],first['linked_before'])
        previous = first['linked_after']
        residents = max(residents,len(nodes) or 1)
    monotone(executed['linked'],previous)
    require(integer(observations['resident_stores']) == residents,'claimed store residency differs from raw windows')
    return snapshots,windows


def cost(observations, report):
    (start,prepared,executed),windows = groups(observations)
    require(integer(observations['preparation_wall_nanos']) > 0,'actual preparation wall required')
    compiler = cpu(executed['children'],start['children'])
    require(compiler > 0 and executed['children']['peak_rss_bytes'] > 0,'actual compiler CPU/RSS required')
    linked = cpu(executed['linked'],start['linked'])
    executed_windows = [window for window in windows if window[0]['phase'] == 'execute']
    replayed_windows = [window for window in windows if window[0]['phase'] == 'replay']
    executions = report['executions']
    require(executed_windows and len(executed_windows) == len(executions)
            and len(replayed_windows) == sum(execution['replay'] is not None for execution in executions),
            'raw store samples differ from actual report execution/replay count')
    for window,execution in zip(executed_windows,executions):
        require(window[0]['virtual_time_nanos'] == execution['record']['virtual_time_nanos'],
                'sample virtual time differs from canonical actual record')
        if len(window) == 1 and window[0].get('incarnation') is None:
            require(window[0]['guest_memory_limit_bytes'] == execution['record']['limits']['memory_bytes'],
                    'store memory limit differs from actual record')
    bugs = set()
    for execution in executions:
        if execution['record']['outcome']['class'] != 'success' and execution['replay'] and execution['replay']['matched']:
            record = dict(execution['record'],seed=0)
            bugs.add(hashlib.sha256(json.dumps(record,sort_keys=True,separators=(',',':'),ensure_ascii=False,allow_nan=False).encode()).hexdigest())
    actual = dict(compiler_cpu_nanos=compiler,linked_cpu_nanos=linked,cpu_nanos=total((compiler,linked)),
        compiler_peak_rss_bytes=executed['children']['peak_rss_bytes'],linked_peak_rss_bytes=executed['linked']['peak_rss_bytes'],
        preparation_wall_nanos=observations['preparation_wall_nanos'],executions=len(executed_windows),replays=len(replayed_windows),
        distinct_outcomes=len({execution['record_sha256'] for execution in executions}),replayable_bugs=len(bugs),
        execution_wall_nanos=total(window[0]['wall_nanos'] for window in executed_windows),
        replay_wall_nanos=total(window[0]['wall_nanos'] for window in replayed_windows),
        fuel_used=total(sample['fuel_used'] for window in executed_windows for sample in window),
        trace_bytes=total(sample['trace_bytes'] for window in executed_windows for sample in window),
        virtual_time_nanos=total(window[0]['virtual_time_nanos'] for window in executed_windows),
        guest_peak_memory_bytes=max(sample['guest_peak_memory_bytes'] for window in windows for sample in window))
    actual['virtual_to_wall'] = dict(numerator=actual['virtual_time_nanos'],denominator=actual['execution_wall_nanos'])
    actual['replayable_bugs_per_compute_hour'] = dict(numerator=total([len(bugs)*HOUR]),denominator=actual['cpu_nanos'])
    return actual


def check(name,quantity,observed,limit,exact):
    integer(observed)
    return dict(name=name,quantity=quantity,observed=observed,limit=limit,
                status='within' if observed <= limit else 'exceeded' if exact else 'unattributed')


def budget(observations):
    (start,prepared,executed),windows = groups(observations)
    require(integer(observations['preparation_wall_nanos']) > 0 and executed['children']['peak_rss_bytes'] > 0,
            'actual preparation wall and compiler peak required')
    checks = [check('preparation wall','exact',observations['preparation_wall_nanos'],10*MINUTE,True),
              check('execution/replay store wall','exact',max(window[0]['wall_nanos'] for window in windows),15*MINUTE,True),
              check('individual compiler peak RSS','largest waited compiler child',executed['children']['peak_rss_bytes'],6*GIB,True),
              check('resident guests','exact',observations['resident_stores'],2,True)]
    for sample in observations['samples']:
        if sample['guest_peak_memory_bytes'] > sample['guest_memory_limit_bytes']:
            suffix = ' '+sample['incarnation'] if sample.get('incarnation') is not None else ''
            checks.append(check(f"guest linear memory {sample['phase']}:{sample['iteration']}{suffix}",'exact',
                          sample['guest_peak_memory_bytes'],sample['guest_memory_limit_bytes'],True))
    linked = executed['linked']['peak_rss_bytes']
    operator = observations.get('operator_peak_rss_bytes') or None
    if operator is not None: integer(operator)
    for window in windows:
        peak = window[0].get('store_peak_rss_bytes')
        require(peak is None or peak <= min(linked,window[0]['linked_after']['peak_rss_bytes']),
                'raw window peak exceeds linked lifetime/boundary')
    attributed = operator is not None and all(window[0].get('store_peak_rss_bytes') for window in windows)
    if attributed:
        initial,previous = start['linked']['peak_rss_bytes'],prepared['linked']['peak_rss_bytes']
        require(initial <= operator and previous <= max(initial,operator),'raw operator peak fails initial/preparation attribution')
        for window in windows:
            first = window[0]
            before,after,peak = first['linked_before']['peak_rss_bytes'],first['linked_after']['peak_rss_bytes'],first['store_peak_rss_bytes']
            require(before <= max(previous,operator) and peak <= after and after <= max(before,peak,operator),
                    'raw store boundary peak attribution inconsistent')
            previous = after
        largest = max(window[0]['store_peak_rss_bytes'] for window in windows)
        require(linked <= max(previous,operator) and largest <= linked <= max(largest,operator),
                'raw phase peaks do not explain actual linked lifetime peak')
        single = max([window[0]['store_peak_rss_bytes'] for window in windows if window[0].get('incarnation') is None] or [0])
        warm = [window[0]['store_peak_rss_bytes'] for window in windows
                if window[0].get('incarnation') is None and window[0].get('cache_hit') is True]
        clusters = [window[0]['store_peak_rss_bytes'] for window in windows if window[0].get('incarnation') is not None]
        checks.extend([check('operator lifetime peak RSS','operator phases',operator,2*GIB,True),
                       check('individual warm helper peak RSS','single cache-hit store window',max(warm),4*GIB,True) if warm else
                       check('individual warm helper peak RSS','single store window upper bound',single,4*GIB,False),
                       check('individual cold helper peak RSS','single store window',single,6*GIB,True)])
        if clusters:
            checks.append(check('conservative resident helper peak upper bound','cluster store window',max(clusters),8*GIB,True))
    else:
        def known(cluster):
            return max([window[0]['store_peak_rss_bytes'] for window in windows if window[0].get('store_peak_rss_bytes')
                        and (window[0].get('incarnation') is not None) == cluster] or [0]) or None
        for name,quantity,peak,limit in [('operator lifetime peak RSS','operator phases',operator,2*GIB),
            ('individual warm helper peak RSS','',None,4*GIB),('individual cold helper peak RSS','single store window',known(False),6*GIB),
            ('conservative resident helper peak upper bound','cluster store window',known(True),8*GIB)]:
            checks.append(check(name,quantity,peak,limit,True) if peak is not None and peak > limit else
                          check(name,'linked process upper bound',linked,limit,False))
    violations = [row['name'] for row in checks if row['status'] == 'exceeded']
    pending = [row['name'] for row in checks if row['status'] == 'unattributed']
    return dict(resident_stores=observations['resident_stores'],within_budget=not violations and not pending,
                checks=checks,violations=violations,pending=pending)


def validate(result):
    actual_cost = cost(result['observations'],result['report'])
    actual_budget = budget(result['observations'])
    require(result['cost'] == actual_cost and result['cost_failure'] is None,'Result cost differs from complete raw accounting')
    require(result['budget'] == actual_budget and result['budget_failure'] is None and actual_budget['within_budget'],
            'Result budget differs from complete raw accounting or is pending/exceeded')
    return actual_cost,actual_budget
