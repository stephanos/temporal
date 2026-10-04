#!/usr/bin/env python3
"""Attribute retained pprof raw samples without deducing payload copy counts."""
import hashlib
import json
from pathlib import Path
import re

HERE = Path(__file__).resolve().parent
RESULT = HERE / 'runs'
METRICS = ['alloc_objects', 'alloc_bytes', 'inuse_objects', 'inuse_bytes']
PROJECT = 'go.temporal.io/server/tools/gomad3/'

def add(target, values):
    for name, value in zip(METRICS, values):
        target[name] = target.get(name, 0) + value

def frame(text):
    matched = re.match(r'(.+?)\s+(\S+):(\d+):\d+', text)
    if not matched:
        raise ValueError(text)
    name, path, line = matched.groups()
    path = path.replace('/tmp/fn109-r19-bound-baseline.7hehYl3f/gomad3/', '')
    return {'function': name, 'file': path, 'line': int(line)}

def category(stack):
    names = [item['function'] for item in stack]
    joined = '\n'.join(names)
    if '.(*r19Executor).snapshot' in joined:
        return 'measurement_profile_overhead'
    if '.r19Stream' in joined:
        return 'fixture_stream_producer'
    if '.r19Transcript' in joined:
        if 'deterministicio.EncodeTranscript' in joined:
            return 'fixture_transcript_encoder'
        return 'fixture_transcript_other'
    if '.PublishArtifact' in joined:
        return 'artifact_publication'
    if '.assessWorld' in joined:
        return 'world_assessment'
    if '.assessCompletion' in joined:
        return 'coverage_outcome_assessment'
    named = ['.newShardedSeedController', '.NewSeedController', '.orderShardRunCompletions', '.ParseSeeds', '.SeedSelection.Iterator']
    if any(name in joined for name in named):
        return 'named_campaign_policy_sites' if '.runLocal' in joined else 'companion_policy_or_other_selection'
    if 'runner/internal/campaign.' in joined:
        return 'campaign_journal_other'
    project_frames = [item for item in stack if item['function'].startswith(PROJECT)]
    if project_frames and project_frames[0]['function'] == PROJECT+'runner.runLocal':
        return 'direct_runLocal_allocation_sites'
    return 'other_runtime_library_or_fixture'

def analyze(raw, completed):
    text = raw.read_text()
    sample_text, location_text = text.split('\nLocations\n', 1)
    location_text = location_text.split('\nMappings\n', 1)[0]
    locations = {}
    current = None
    for line in location_text.splitlines():
        matched = re.match(r'\s*(\d+):\s+0x[0-9a-f]+\s+M=\d+\s+(.*)', line)
        if matched:
            current = int(matched[1])
            locations[current] = [frame(matched[2])]
        elif line.strip():
            assert current is not None
            locations[current].append(frame(line.strip()))
    total, categories, sites, size_sites = {}, {}, {}, {}
    sample_count = 0
    prior_sample = None
    for line in sample_text.splitlines():
        label = re.match(r'\s*bytes:\[(\d+)\]', line)
        if label:
            assert prior_sample is not None
            key, values = prior_sample
            add(size_sites.setdefault(key+(int(label[1]),), {}), values)
            prior_sample = None
            continue
        matched = re.match(r'\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+):\s+(.*)', line)
        if not matched:
            continue
        values = list(map(int, matched.groups()[:4]))
        ids = list(map(int, matched[5].split()))
        stack = [entry for identity in ids for entry in locations[identity]]
        assert stack
        origin = category(stack)
        leaf = stack[0]
        key = (origin, leaf['function'], leaf['file'], leaf['line'])
        prior_sample = (key, values)
        add(total, values)
        add(categories.setdefault(origin, {}), values)
        add(sites.setdefault(key, {}), values)
        sample_count += 1
    assert sample_count > 0
    for data in categories.values():
        data['alloc_bytes_per_completed_execution'] = data['alloc_bytes']/completed if completed else None
    all_sites = [dict(category=key[0], function=key[1], file=key[2], line=key[3], **data) for key, data in sites.items()]
    all_sites.sort(key=lambda row: (-row['alloc_bytes'], row['function'], row['line']))
    all_sizes = [dict(category=key[0], function=key[1], file=key[2], line=key[3], allocation_size_bucket_bytes=key[4], **data) for key, data in size_sites.items()]
    all_sizes.sort(key=lambda row: (-row['alloc_bytes'], row['function'], row['line']))
    for name in METRICS:
        assert sum(data[name] for data in categories.values()) == total[name]
        assert sum(data[name] for data in all_sites) == total[name]
        assert sum(data[name] for data in all_sizes) == total[name]
    return {'raw_sha256': hashlib.sha256(raw.read_bytes()).hexdigest(), 'sample_count': sample_count, 'completed_execution_denominator': completed, 'totals': total, 'disjoint_stack_origin_categories': categories, 'all_allocation_leaf_sites': all_sites, 'allocation_size_bucket_sites': all_sizes}

report = {'method': 'Every raw sample is assigned once by the retained category function; leaf sites are exact pprof allocation sites. Categories are stack attribution, not object ownership or copy counts. All sites reconcile to profile sample totals.', 'cases': {}}
for mode in ['discard', 'novel']:
    for jobs in [10, 100]:
        case = f'{mode}-{jobs}'
        measured = json.loads((RESULT/case/'measurement.json').read_text())
        rows = {row['Name']: row for row in measured['snapshots']}
        stages = {}
        for stage in ['early-pair', 'late-pair', 'completed', 'process-end']:
            completed = jobs if stage == 'process-end' else rows[stage]['Committed']
            stages[stage] = analyze(RESULT/(case+'-'+stage+'-raw.log'), completed)
            for origin, values in stages[stage]['disjoint_stack_origin_categories'].items():
                if origin == 'artifact_publication':
                    values.pop('alloc_bytes_per_completed_execution')
                    publications = measured['artifact_count'] if stage in ['completed', 'process-end', 'late-pair'] else 0
                    values['publication_denominator'] = publications
                    values['alloc_bytes_per_publication'] = values['alloc_bytes']/publications if publications else None
        report['cases'][case] = {'measurement': measured, 'profiles': stages}
(RESULT/'profile-attribution.json').write_text(json.dumps(report, indent=2)+'\n')
for case, data in report['cases'].items():
    print(case, json.dumps({stage: value['totals'] for stage, value in data['profiles'].items()}))
