"""Reconstruct the fixed native identity cases from frozen source and actual compiled context."""
from pathlib import Path
import hashlib

from driver import require
import native_roles

SCHEMA = 'gomad.core-predicate-observation/v1'
MARKER = 'GOMAD_CORE_OBSERVATION '
EXTRA = ('build.rs', 'build_identity.rs', 'Cargo.toml', '../Cargo.toml', '../Cargo.lock', '../rust-toolchain.toml')
REQUIRED = ('src/lib.rs', 'src/linked_cluster.rs', 'src/linked.rs', 'src/linked_storage.rs', 'src/storage_transport.rs',
            'src/volume/mod.rs', 'src/namespace/image.rs', 'src/namespace/call.rs', 'src/metrics.rs', 'src/volume/image.rs',
            'src/cluster/network/evidence.rs', 'src/linked_inspection.rs', 'src/linked_storage_inspection.rs',
            'src/model/inspection.rs', 'src/protocol.rs', 'src/model.rs')
OWNERS = {'identity.storage-owner': REQUIRED[1:10],
          'identity.inspection-owner': ('src/linked_inspection.rs', 'src/linked_storage_inspection.rs', 'src/model/inspection.rs', 'src/cluster/network/evidence.rs'),
          'identity.strict-json-owner': ('src/protocol.rs', 'src/model.rs', 'src/linked_storage.rs')}
TESTS = ('compiled_identity_matches_the_exact_captured_native_source_and_build_context',
         'native_storage_evidence_owners_rotate_identity_and_missing_owners_refuse',
         'native_inspection_owners_rotate_identity_and_missing_owners_refuse',
         'strict_typed_json_owners_rotate_identity_and_missing_owners_refuse',
         'empty_source_closure_refuses_before_producing_an_identity')


def source_inputs(config):
    root = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    paths = list((root / 'src').rglob('*'))
    require(paths and all(not path.is_symlink() for path in paths), 'identity source tree contains aliases or is empty')
    names = sorted({str(path.relative_to(root)) for path in paths if path.is_file() and path.suffix == '.rs'} | set(EXTRA))
    files = {}
    for name in names:
        path = root / name
        native_roles.pinned(config, path.resolve())
        files[name] = path.read_bytes()
    require(all(name in files for name in REQUIRED), 'identity source closure omits an original native owner')
    return files


def implementation(files, bindings, compiler_utf8):
    for name in REQUIRED:
        require(name in files, 'missing implementation owner ' + name)
    digest = hashlib.sha256(b'gomad.linked-implementation/v1\0')
    for name, value in sorted(bindings.items()):
        digest.update(name.encode() + b'\0' + value.encode() + b'\0')
    digest.update(compiler_utf8.encode())
    for name, raw in sorted(files.items()):
        encoded = name.encode()
        digest.update(len(encoded).to_bytes(8, 'big') + encoded + len(raw).to_bytes(8, 'big') + raw)
    return digest.hexdigest()


def observations(config, stdout, context):
    """Context must come from the fixed harness's authenticated actual build journals."""
    require(set(context) == {'compiler_utf8', 'bindings'} and context['bindings'].get('PROFILE') == 'release'
            and set(context['bindings']) == {'PROFILE', 'TARGET'} and context['bindings']['TARGET']
            and context['compiler_utf8'].startswith('rustc 1.99.1 '), 'foreign identity compiled context')
    files = source_inputs(config)
    root = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    expected_files = [dict(name=name, path=str(root / name), bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest()) for name, raw in sorted(files.items())]
    rows = [native_roles.decode(line[line.index(MARKER) + len(MARKER):]) for line in stdout.splitlines() if MARKER in line]
    required = {('identity.compiled-native', None), ('identity.empty-closure', None)}
    required.update((case, owner) for case, owners in OWNERS.items() for owner in owners)
    require(len(rows) == len(required) == 18, 'identity harness lacks its exact eighteen raw cases')
    before = implementation(files, context['bindings'], context['compiler_utf8'])
    seen = set()
    for row in rows:
        require(set(row) == {'schema', 'case', 'inputs', 'observed'} and row['schema'] == SCHEMA, 'foreign native identity observation')
        inputs, observed = row['inputs'], row['observed']
        case = row['case']
        mutated = case in OWNERS
        require(set(inputs) == {'files', 'bindings', 'compiler_utf8', 'required_owners'} | ({'owner', 'mutation_append_utf8'} if mutated else set()),
                'identity input fields changed')
        key = (case, inputs.get('owner'))
        require(key in required and key not in seen, 'unknown/duplicate identity case or owner')
        seen.add(key)
        require(inputs['bindings'] == context['bindings'] and inputs['compiler_utf8'] == context['compiler_utf8']
                and inputs['required_owners'] == list(REQUIRED), 'identity observation substituted a compiler, feature or required-owner context')
        require(all(type(entry['bytes']) is int for entry in inputs['files']), 'identity file sizes must be typed integer observations')
        require(inputs['files'] == ([] if case == 'identity.empty-closure' else expected_files), 'identity captured file closure differs from actual frozen source')
        if case == 'identity.compiled-native':
            require(observed == dict(compiled=before, recomputed=before), 'compiled identity differs from exact captured source/build context')
        elif case == 'identity.empty-closure':
            require(observed == dict(error='missing implementation owner src/lib.rs'), 'empty identity closure did not fail before hashing')
        else:
            owner = inputs['owner']
            require(inputs['mutation_append_utf8'] == 'changed implementation', 'identity mutation changed the original owner perturbation')
            changed = dict(files, **{owner: files[owner] + b'changed implementation'})
            after = implementation(changed, context['bindings'], context['compiler_utf8'])
            require(after != before and observed == dict(before=before, after=after, missing_error='missing implementation owner ' + owner),
                    'identity owner mutation/missing-owner refusal differs from raw frozen preimages')
    require(seen == required, 'identity harness omitted an original case')
    return dict(implementation_sha256=before, case_count=len(seen), source_inputs={name: hashlib.sha256(raw).hexdigest() for name, raw in files.items()})
