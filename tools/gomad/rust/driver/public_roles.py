"""Derive public-verb role bindings from closed native build receipts."""
import argparse
import hashlib
from pathlib import Path

from driver import HERE, read, require, sha, write
import main_cli_build
import native_roles

SCHEMA = 'gomad.all-verb-builds/v2'
ROLES = {'release': 'release', 'fixture': 'host-fixture', 'stress': 'stress', 'stress-release': 'stress-release',
         'determinism': 'counterfactual-fixture', 'divergence': 'counterfactual-fixture',
         'reference-helper': 'conformance-reference'}
FEATURES = {name: (['conformance'] if name == 'reference-helper' else main_cli_build.ROLES[role])
            for name, role in ROLES.items()}


def source_inventory(receipt):
    import dry_run
    if receipt.get('schema') == dry_run.SCHEMA:
        # A dry-run build carries no source custody; inventory the whole checkout tree it was built from,
        # including staged untracked inputs (stress stages its broker source under .tmp).
        repo = HERE.parents[3]
        inventory = {str(path.relative_to(repo)): sha(path) for path in repo.rglob('*') if path.is_file()
                     and not {'.git', '.flow', '__pycache__'} & set(path.relative_to(repo).parts)}
        return ''.join(value + '  ' + name + '\n' for name, value in sorted(inventory.items()))
    plan = read(receipt['tools_plan'])
    repo = Path(plan['repo'])
    require(repo == HERE.parents[3], 'public role build belongs to another candidate checkout')
    files, digest, head, links = main_cli_build.source_state(repo, read(receipt['gnu_plan'])['git'])
    require((digest, head, links) == (receipt['source_sha256'], receipt['head'], receipt['symlinks']),
            'public role source inventory differs from its authenticated build')
    inventory = {str(Path(name).relative_to(repo)): value for name, value in files.items()}
    for name in links:
        path = Path(name)
        require(path.resolve().is_relative_to(repo) and path.is_file(), 'public source symlink escapes candidate')
        inventory[str(path.relative_to(repo))] = sha(path)
    return ''.join(value + '  ' + name + '\n' for name, value in sorted(inventory.items()))


def read_builds(path, inventory=None, frozen=None):
    path = native_roles.regular(path)
    value = read(path)
    require(set(value) == {'schema', 'source_inventory', 'source_inventory_sha256', 'roles'}
            and value['schema'] == SCHEMA and set(value['roles']) == set(ROLES),
            'public role inventory requires all closed typed build receipts')
    source = native_roles.regular(value['source_inventory'])
    require(source.resolve() == source and (inventory is None or source == Path(inventory))
            and sha(source) == value['source_inventory_sha256'], 'public candidate source inventory changed')
    built, checked = {}, {}
    paths = {str(path), str(source)}
    footprint = None
    for name, expected_role in ROLES.items():
        binding = value['roles'][name]
        require(isinstance(binding, dict) and set(binding) == {'build_receipt', 'sha256'}, 'public role has caller build claims')
        receipt_path = native_roles.regular(binding['build_receipt'])
        require(receipt_path.resolve() == receipt_path and sha(receipt_path) == binding['sha256'], 'public typed build receipt changed')
        key = (str(receipt_path), expected_role)
        if key not in checked:
            checked[key] = main_cli_build.read_build(receipt_path, expected_role=expected_role)
        receipt = checked[key]
        current = (receipt['source_sha256'], receipt['head'], receipt['symlinks'], receipt['tools_plan'], receipt['gnu_plan'])
        require(footprint is None or footprint == current, 'public roles mix candidate source/compiler lineages')
        footprint = current
        require(receipt['features'] == main_cli_build.ROLES[expected_role]
                and receipt['helper_features'] == main_cli_build.helper_features(expected_role), 'public compiled role features differ')
        paths.update(receipt['inputs']); paths.add(str(receipt_path))
        built[name] = receipt
    require(source.read_text() == source_inventory(built['release']), 'public source inventory is incomplete or foreign')
    import dry_run
    if frozen is not None:
        require(footprint[0] == dry_run.SCHEMA or (frozen['source_sha256'], frozen['head'], frozen['symlinks']) == footprint[:3],
                'public builds differ from frozen candidate')
        require(all(frozen['files'].get(name) == sha(name) for name in paths), 'public typed build closure is not fully frozen')
        require(built['release']['binary'] == frozen['binary'] and built['release']['binary_sha256'] == frozen['binary_sha256']
                and built['release']['host_sha256'] == frozen['identity']['host']
                and built['release']['helper_sha256'] == frozen['identity']['helper'], 'public ordinary role differs from candidate')
    return built, paths


def prepare(receipts_path, inventory, output):
    receipts = read(receipts_path)
    require(isinstance(receipts, dict) and set(receipts) == set(ROLES), 'missing/foreign public role receipt')
    release = main_cli_build.read_build(receipts['release'], expected_role='release')
    require(all(path.is_absolute() and path.resolve() == path for path in (inventory, output)), 'public role outputs must be canonical absolute')
    require(not inventory.exists() and not output.exists(), 'public role outputs must be fresh')
    write_value = source_inventory(release)
    value = dict(schema=SCHEMA, source_inventory=str(inventory),
                 source_inventory_sha256=hashlib.sha256(write_value.encode()).hexdigest(),
                 roles={name: dict(build_receipt=str(native_roles.regular(path)), sha256=sha(path)) for name, path in receipts.items()})
    # Authenticate every role before publishing either output; the inventory is derived, never caller text.
    for name, role in ROLES.items():
        receipt = main_cli_build.read_build(receipts[name], expected_role=role)
        require((receipt['source_sha256'], receipt['head'], receipt['symlinks'], receipt['tools_plan'], receipt['gnu_plan'])
                == (release['source_sha256'], release['head'], release['symlinks'], release['tools_plan'], release['gnu_plan']),
                'public role preparation mixes candidate lineages')
    with inventory.open('x') as handle: handle.write(write_value)
    write(output, value)
    return value


def static_reference(config, receipt):
    import static_stages
    entry = config['stages']['validate']
    argv = entry['argv']
    require('--plan' in argv, 'original validate stage lacks its fixed static plan')
    static_stages.authenticate_result(Path(argv[argv.index('--plan') + 1]), 'validate')
    report = read(Path(config['root']) / 'stages/validate/static/result.json')
    rows = [result for action in report['actions'] for result in action['results']
            if action['recipe']['kind'] == 'build-role' and result.get('role') == 'conformance-reference']
    consumed = read(Path(config['root']) / 'static-roles/conformance-reference/build.json')
    require(len(rows) == 1 and consumed == rows[0], 'retained Go commands consumed a different static conformance helper')
    require(len(rows) == 1 and rows[0]['features'] == ['conformance', 'reference-helper']
            and rows[0]['requested_features'] == ['conformance'] and rows[0]['binary_sha256'] == receipt['binary_sha256']
            and sha(rows[0]['binary']) == receipt['binary_sha256'], 'actual static conformance helper differs from admitted public helper')
    return rows[0]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipts', 'source-inventory', 'output'): parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    prepare(args.receipts, args.source_inventory, args.output)


if __name__ == '__main__': main()
