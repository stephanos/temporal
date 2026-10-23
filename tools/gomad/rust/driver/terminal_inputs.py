"""Closed terminal preparation request and presealed shared compiler/overlay admission."""
import hashlib
import json
from pathlib import Path

from driver import require, sha
import live
import native_roles

SCHEMA = 'gomad.native-terminal-module-preparation-input/v1'
INSTALLATIONS = ('cooperative-runtime', 'onebox-cooperative')


def runtime_identity(config):
    digest = hashlib.sha256(b'gomad-wasm-runtime-implementation/v1\0')
    for name in native_roles.RUNTIME:
        path = Path(config['repo']) / 'tools/gomad/internal/backends/wasm/toolchain' / name
        live.sealed_input(config, str(path))
        raw = path.read_bytes()
        digest.update(len(raw).to_bytes(8, 'big') + raw)
    return digest.hexdigest()


def compiler_inputs(config):
    primary = config['primary']
    compiler = config['final_profiles']['stock_go']
    cache, runtime = Path(primary['cache']), Path(primary['runtime_root'])
    require(compiler == primary['compiler'] and Path(compiler).resolve() == Path(compiler),
            'terminal preparation requires the explicit admitted stock compiler')
    require(cache.is_absolute() and cache.resolve() == cache and cache.is_dir()
            and not cache.is_relative_to(Path(config['repo'])), 'terminal compiler cache must be its admitted external directory')
    require(runtime == Path(config['repo']) / 'tools/gomad/internal/model/runtimesource',
            'terminal runtime must be the frozen original runtime source root')
    live.sealed_input(config, compiler)
    stock_root = Path(compiler).parent.parent
    require(Path(compiler).parent.name == 'bin', 'terminal stock compiler must belong to its explicit distribution root')
    live.sealed_input(config, str(runtime))
    expected_runtime = runtime_identity(config)
    inventories = []
    for name in INSTALLATIONS:
        path = config['installations'][name]
        live.sealed_input(config, path)
        installed = native_roles.read(path)
        require(installed['schema'] == 'gomad.rust-wasm-installation/v1'
                and installed['profile'] == 'gomad.wasi-cooperative/v1' and installed['go_version'] == 'go1.27.1'
                and installed['compiler_path'] == compiler and installed['compiler_sha256'] == sha(compiler)
                and installed['cache_root'] == str(cache) and installed['runtime_root'] == str(runtime)
                and installed['overlay_module_root'] == str(Path(config['repo']) / 'tools/gomad')
                and installed['runtime_implementation_sha256'] == expected_runtime
                and installed['host_sha256'] == config['identity']['host'] and installed['helper_sha256'] == config['identity']['helper'],
                'terminal installation has a different stock compiler/cache/runtime/current owner')
        inventory = installed['admission']['compiler_inputs']
        require(inventory and hashlib.sha256(json.dumps(inventory, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()).hexdigest() == installed['admission']['inputs_sha256'],
                'terminal admitted compiler closure preimage differs')
        for member, expected in inventory.items():
            source = Path(member)
            require(source.is_absolute() and source.is_file() and not source.is_symlink()
                    and config['files'].get(str(source.resolve())) == expected == sha(source),
                    'terminal compiler/overlay member is not independently presealed: ' + member)
        require(inventory.get(compiler) == sha(compiler), 'terminal admission omits stock compiler')
        version = stock_root / 'VERSION'
        require(inventory.get(str(version)) == sha(version) and version.read_text().splitlines()[0] == 'go1.27.1',
                'terminal admission omits original stock Go distribution/version')
        for member in runtime.rglob('*'):
            if member.is_file():
                require(inventory.get(str(member)) == sha(member), 'terminal admission omits runtime source member')
        overlays = []
        for member in inventory:
            source = Path(member)
            if source.is_relative_to(cache / 'runtime-overlay') and source.suffix == '.json':
                value = native_roles.read(source)
                if isinstance(value, dict) and 'Replace' in value:
                    overlays.append(value['Replace'])
        require(len(overlays) == 1 and isinstance(overlays[0], dict) and overlays[0],
                'terminal admission requires its exact generated runtime overlay')
        for original, replacement in overlays[0].items():
            require(Path(original).is_absolute() and Path(original).resolve() == Path(original)
                    and Path(original).is_relative_to(stock_root / 'src'),
                    'terminal overlay rewrites a foreign stock compiler root')
            if Path(original).is_file():
                require(inventory.get(original) == sha(original), 'terminal overlay omits original stock source')
            path = Path(replacement)
            require(path.is_absolute() and path.resolve() == path and path.is_relative_to(cache / 'runtime-overlay')
                    and inventory.get(replacement) == sha(path),
                    'terminal generated overlay replacement is not admitted in the shared cache')
        inventories.append(inventory)
    require(inventories[0] == inventories[1], 'terminal cooperative/onebox compiler closures use different caches or overlays')
    return inventories[0]


def preparation_spec(config):
    compiler_inputs(config)
    result = dict(schema=SCHEMA, repo=config['repo'], stock_go=config['final_profiles']['stock_go'],
                  toolchain_cache=config['primary']['cache'])
    temporary = config.get('environment', {}).get('GOTMPDIR')
    if temporary:
        require(Path(temporary).is_absolute() and Path(temporary).resolve() == Path(temporary)
                and Path(temporary).is_dir(), 'terminal GOTMPDIR must be the actual assigned absolute directory')
        result['temporary_directory'] = temporary
    return result
