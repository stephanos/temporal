"""Go StorageModel JSON bytes used by the existing native storage identity."""
import json


def encode(model):
    volumes = model.get('volumes')
    go_model = dict(volumes=None if volumes is None else [
        dict(ID=volume['ID'], Path=volume['Path'], CapacityBytes=volume['CapacityBytes'])
        for volume in volumes],
        limits=dict(PendingOperations=model['limits']['PendingOperations'],
                    Transitions=model['limits']['Transitions']))
    if model.get('faults'):
        go_model['faults'] = []
        for fault in model['faults']:
            row = {name: fault[name] for name in ('name', 'resource', 'operation', 'occurrence', 'action')}
            if fault.get('bytes'):
                row['bytes'] = fault['bytes']
            go_model['faults'].append(row)
    if model.get('sqlite_lock_databases'):
        go_model['sqlite_lock_databases'] = model['sqlite_lock_databases']
    encoded = json.dumps(go_model, ensure_ascii=False, separators=(',', ':'))
    for character, escaped in (('&', '\\u0026'), ('<', '\\u003c'), ('>', '\\u003e'),
                               ('\u2028', '\\u2028'), ('\u2029', '\\u2029')):
        encoded = encoded.replace(character, escaped)
    return encoded.encode()
