#!/usr/bin/env python3
"""Verify retained standalone proof objects without the original build caches."""
import hashlib
import json
import pathlib
import tarfile

HERE = pathlib.Path(__file__).resolve().parent
manifest = json.loads((HERE/'recovery-manifest.json').read_text())
archives = json.loads((HERE/'archive-manifests.json').read_text())
errors, checked = [], set()
for record in manifest['records']:
    path = pathlib.Path(record['recovery_object'])
    if str(path) in checked:
        continue
    checked.add(str(path))
    if not path.is_file() or path.stat().st_size != record['size'] or hashlib.sha256(path.read_bytes()).hexdigest() != record['sha256']:
        errors.append('recovery object differs: ' + record['destination'])
archive_members = 0
for name, archive in archives.items():
    with tarfile.open(archive['object']['recovery_object'], 'r:gz') as retained:
        actual = retained.getmembers()
        expected = archive['verified_members']
        if [r.name for r in actual] != [r['path'] for r in expected]:
            errors.append('archive inventory differs: ' + name)
        for member, record in zip(actual, expected):
            if (not member.isfile() or member.size != record['size'] or oct(member.mode) != record['mode']
                    or hashlib.sha256(retained.extractfile(member).read()).hexdigest() != record['sha256']):
                errors.append('archive member differs: ' + name + ':' + member.name)
            archive_members += 1
result = {'source_commit': manifest['source_commit'], 'source_tree': manifest['source_tree'],
          'unique_objects_verified': len(checked), 'archive_members_verified': archive_members,
          'original_cache_accessed': False, 'errors': errors, 'result': 'pass' if not errors else 'fail'}
(HERE/'recovery-verification.json').write_text(json.dumps(result, indent=2)+'\n')
print(json.dumps(result, indent=2))
if errors:
    raise SystemExit(1)
