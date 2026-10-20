#!/usr/bin/env python3
"""Build a typed consumer fixture from fresh helper and Go producer evidence.

This publishes no migrated artifact and runs no Rust minimizer. The fixture
retains the original specimen as an authenticated supplemental payload.
"""
import argparse
import copy
import json
from pathlib import Path

p = argparse.ArgumentParser()
p.add_argument('--cluster-input', type=Path, required=True)
p.add_argument('--go-minimization', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
args = p.parse_args()
value = json.loads(args.cluster_input.read_bytes())
reference_raw = args.go_minimization.read_bytes()
reference = json.loads(reference_raw)
minimization = copy.deepcopy(reference['minimization'])
assert minimization['schema'] == 'gomad.minimization/choice-prefix/v1'
minimization['schema'] = 'gomad.minimization-evidence/v1'


def digest(container, key):
    text = container[key]
    assert text.startswith('sha256:') and len(text) == 71, key
    assert all(character in '0123456789abcdef' for character in text[7:]), key
    container[key] = text[7:]


for field in ('implementation_sha256', 'parent_record_hash', 'parent_failure_signature',
              'original_candidate_sha256', 'final_candidate_sha256'):
    digest(minimization, field)
digest(minimization['predicate'], 'failure_signature')
for reduction in minimization['accepted']:
    digest(reduction, 'before_sha256')
    digest(reduction, 'after_sha256')
    for decision in reduction['removed']:
        digest(decision, 'identity')
value['details']['minimization'] = minimization
value['details']['outputs'] = None
value['payloads']['evidence/minimization-go-reference.json'] = list(reference_raw)
with args.output.open('xb') as output:
    output.write(json.dumps(value, sort_keys=True, separators=(',', ':')).encode())
print(json.dumps({'output': str(args.output), 'consumer_fixture': True,
                  'rust_minimization_producer': False}))
